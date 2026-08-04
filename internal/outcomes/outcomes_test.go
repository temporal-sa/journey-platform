package outcomes

import (
	"context"
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/experiments"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
)

func TestAssignmentWithoutExposureNotExposed(t *testing.T) {
	ctx := context.Background()
	repo := postgres.NewMemoryRepository()

	// 1. Create sticky assignment
	expSvc := experiments.NewAssignmentService(repo)
	expDef := &experiments.Experiment{
		ExperimentID:      "exp-001",
		Version:           1,
		Salt:              "exp-001-salt",
		Status:            experiments.ExperimentStatusActive,
		RandomizationUnit:        experiments.UnitSubjectID,
		AttributionWindowSeconds: 86400,
		Metrics: []experiments.MetricDefinition{
			{Key: "conv", EventType: "conversion"},
		},
		Variants: []experiments.Variant{
			{Key: "control", WeightBasisPoints: 5000},
			{Key: "treatment", WeightBasisPoints: 5000},
		},
	}

	asg, err := expSvc.GetOrAssign(ctx, experiments.AssignmentRequest{
		TenantID:   "tenant-a",
		Experiment: expDef,
		SubjectID:  "user-100",
		Mode:       experiments.ModeProduction,
	})
	if err != nil {
		t.Fatalf("unexpected assignment error: %v", err)
	}

	// Verify assignment exists
	if asg.AssignmentID == "" {
		t.Fatal("expected non-empty assignment ID")
	}

	// Verify NO exposure exists yet
	exposureSvc := NewExposureService(repo)
	_, err = exposureSvc.GetExposure(ctx, "tenant-a", "exp-tenant-a-exp-001-user-100-1")
	if err == nil {
		t.Error("expected assignment without exposure to NOT have exposure record")
	}

	// Now record exposure
	expRec, err := exposureSvc.RecordExposure(ctx, ExposureRequest{
		TenantID:          "tenant-a",
		ExposureID:        "exp-1001",
		ExperimentID:      "exp-001",
		ExperimentVersion: 1,
		AssignmentID:      asg.AssignmentID,
		SubjectID:         "user-100",
		VariantID:         asg.VariantID,
		ExecutionMode:     "production",
	})
	if err != nil {
		t.Fatalf("unexpected exposure error: %v", err)
	}

	if expRec.ExposureID != "exp-1001" {
		t.Errorf("expected exposure ID exp-1001, got %s", expRec.ExposureID)
	}

	// Verify exposure now exists in DB
	fetched, err := exposureSvc.GetExposure(ctx, "tenant-a", "exp-1001")
	if err != nil {
		t.Fatalf("failed to fetch recorded exposure: %v", err)
	}
	if fetched.SubjectID != "user-100" {
		t.Errorf("expected subject_id user-100, got %s", fetched.SubjectID)
	}
}

func TestExposureServiceOutboxIdempotency(t *testing.T) {
	ctx := context.Background()
	repo := postgres.NewMemoryRepository()
	expSvc := NewExposureService(repo)

	req := ExposureRequest{
		TenantID:          "tenant-test",
		ExposureID:        "exp-fixed-1",
		ExperimentID:      "exp-ab",
		ExperimentVersion: 1,
		JourneyID:         "journey-1",
		JourneyVersion:    1,
		AssignmentID:      "asg-999",
		SubjectID:         "subj-999",
		VariantID:         "var-a",
		ExecutionMode:     "production",
	}

	// First call -> creates exposure + outbox
	rec1, err := expSvc.RecordExposure(ctx, req)
	if err != nil {
		t.Fatalf("first exposure failed: %v", err)
	}
	if rec1.ExposureID != "exp-fixed-1" {
		t.Fatalf("expected exposure ID exp-fixed-1, got %s", rec1.ExposureID)
	}

	// Check outbox
	pending, err := repo.ListPendingOutboxEvents(ctx, "tenant-test", 10)
	if err != nil {
		t.Fatalf("failed to list outbox events: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(pending))
	}
	if pending[0].EventType != "experiment.exposure_recorded" {
		t.Errorf("expected event_type experiment.exposure_recorded, got %s", pending[0].EventType)
	}

	// Second call -> idempotent, no duplicate outbox or error
	rec2, err := expSvc.RecordExposure(ctx, req)
	if err != nil {
		t.Fatalf("second exposure failed: %v", err)
	}
	if rec2.ExposureID != "exp-fixed-1" {
		t.Fatalf("expected idempotent exposure ID exp-fixed-1, got %s", rec2.ExposureID)
	}

	pendingAfter, _ := repo.ListPendingOutboxEvents(ctx, "tenant-test", 10)
	if len(pendingAfter) != 1 {
		t.Errorf("expected still 1 outbox event after duplicate call, got %d", len(pendingAfter))
	}
}

func TestDuplicateCallbackIngestion(t *testing.T) {
	ctx := context.Background()
	h := New()

	event := &OutcomeEvent{
		EventID:           "evt-dup-101",
		Source:            "webhook",
		SourceIdentity:    "webhook:evt-dup-101",
		EventType:         "conversion",
		TenantID:          "tenant-x",
		ActionID:          "act-001",
		AssignmentID:      "asg-001",
		ExposureID:        "exp-001",
		ExperimentID:      "exp-1",
		ExperimentVersion: 1,
		JourneyID:         "j-1",
		JourneyVersion:    1,
		ExecutionMode:     "production",
		SubjectID:         "subj-1",
		VariantID:         "var-a",
		Value:             10.0,
		Timestamp:         time.Now().UTC(),
	}

	// First ingestion -> 1 raw, 1 unique
	res1, err := h.Ingest(ctx, event)
	if err != nil {
		t.Fatalf("first ingest failed: %v", err)
	}
	if res1.IsDuplicate {
		t.Error("first ingest should not be marked as duplicate")
	}
	if res1.RawCount != 1 || res1.UniqueCount != 1 {
		t.Errorf("expected raw=1 unique=1, got raw=%d unique=%d", res1.RawCount, res1.UniqueCount)
	}
	if res1.QualityTag.Classification != QualityClassNormal {
		t.Errorf("expected quality normal, got %s", res1.QualityTag.Classification)
	}

	// Second ingestion (duplicate callback) -> 2 raw, 1 unique
	res2, err := h.Ingest(ctx, event)
	if err != nil {
		t.Fatalf("second ingest failed: %v", err)
	}
	if !res2.IsDuplicate {
		t.Error("second ingest should be marked as duplicate")
	}
	if res2.RawCount != 2 || res2.UniqueCount != 1 {
		t.Errorf("expected raw=2 unique=1 after duplicate, got raw=%d unique=%d", res2.RawCount, res2.UniqueCount)
	}
	if res2.QualityTag.Classification != QualityClassDuplicateCallback {
		t.Errorf("expected quality duplicate_callback, got %s", res2.QualityTag.Classification)
	}
	if !res2.QualityTag.IsFiltered {
		t.Error("duplicate callback tag should have IsFiltered=true")
	}

	// Verify total raw facts recorded = 2
	outcomesList := h.GetOutcomes()
	if len(outcomesList) != 2 {
		t.Fatalf("expected 2 raw facts recorded in storage, got %d", len(outcomesList))
	}
	if outcomesList[0].QualityTag.Classification != QualityClassNormal {
		t.Errorf("first outcome fact should be normal")
	}
	if outcomesList[1].QualityTag.Classification != QualityClassDuplicateCallback {
		t.Errorf("second outcome fact should be duplicate_callback")
	}
}

func TestQualityClassifications(t *testing.T) {
	ctx := context.Background()
	h := New()

	// 1. Privacy Proxy Open (Apple Mail Privacy Protection)
	openProxy := &OutcomeEvent{
		EventID:      "evt-open-proxy",
		EventType:    "open",
		TenantID:     "tenant-1",
		SubjectID:    "user-1",
		UserAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 (Apple Mail Privacy Protection)",
		Timestamp:    time.Now().UTC(),
		AssignmentID: "asg-1",
		ExposureID:   "exp-1",
	}

	resOpen, err := h.Ingest(ctx, openProxy)
	if err != nil {
		t.Fatalf("open ingest failed: %v", err)
	}
	if resOpen.QualityTag.Classification != QualityClassPrivacyProxy {
		t.Errorf("expected classification privacy_proxy, got %s", resOpen.QualityTag.Classification)
	}
	if !resOpen.QualityTag.IsFiltered {
		t.Error("privacy proxy open should be tagged with IsFiltered=true")
	}

	// 2. Link Scanner Click (Barracuda crawler)
	clickScanner := &OutcomeEvent{
		EventID:      "evt-click-scanner",
		EventType:    "click",
		TenantID:     "tenant-1",
		SubjectID:    "user-2",
		UserAgent:    "Barracuda HTTP Crawler/1.0",
		TargetURL:    "https://example.com/promo",
		Timestamp:    time.Now().UTC(),
		AssignmentID: "asg-2",
		ExposureID:   "exp-2",
	}

	resClick, err := h.Ingest(ctx, clickScanner)
	if err != nil {
		t.Fatalf("click ingest failed: %v", err)
	}
	if resClick.QualityTag.Classification != QualityClassLinkScanner {
		t.Errorf("expected classification link_scanner, got %s", resClick.QualityTag.Classification)
	}
	if !resClick.QualityTag.IsFiltered {
		t.Error("link scanner click should be tagged with IsFiltered=true")
	}

	// 3. Normal Conversion Event
	normalConv := &OutcomeEvent{
		EventID:      "evt-norm-conv",
		EventType:    "conversion",
		TenantID:     "tenant-1",
		SubjectID:    "user-3",
		UserAgent:    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		Value:        49.99,
		Timestamp:    time.Now().UTC(),
		AssignmentID: "asg-3",
		ExposureID:   "exp-3",
	}

	resNorm, err := h.Ingest(ctx, normalConv)
	if err != nil {
		t.Fatalf("normal conv ingest failed: %v", err)
	}
	if resNorm.QualityTag.Classification != QualityClassNormal {
		t.Errorf("expected classification normal, got %s", resNorm.QualityTag.Classification)
	}
	if resNorm.QualityTag.IsFiltered {
		t.Error("normal conversion should NOT be filtered")
	}

	// Verify all 3 facts appear in raw outcomes list
	rawOutcomes := h.GetOutcomes()
	if len(rawOutcomes) != 3 {
		t.Fatalf("expected 3 raw facts recorded, got %d", len(rawOutcomes))
	}
}
