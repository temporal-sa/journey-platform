package analytics

import (
	"context"
	"testing"
	"time"
)

func TestAggregateCalculations(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	tenantID := "tenant-alpha"
	expID := "exp-001"
	now := time.Now().UTC()

	// Insert exposures for control and treatment
	exposures := []ExposureFact{
		{ExposureID: "exp-c1", TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: "sub-1", ExposedAt: now, IsTest: false},
		{ExposureID: "exp-c2", TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: "sub-2", ExposedAt: now, IsTest: false},
		{ExposureID: "exp-c3", TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: "sub-3", ExposedAt: now, IsTest: false},
		{ExposureID: "exp-c4", TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: "sub-4", ExposedAt: now, IsTest: false},
		{ExposureID: "exp-c5", TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: "sub-5", ExposedAt: now, IsTest: false},
		{ExposureID: "exp-c6", TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: "sub-6", ExposedAt: now, IsTest: false},
		{ExposureID: "exp-c7", TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: "sub-7", ExposedAt: now, IsTest: false},
		{ExposureID: "exp-c8", TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: "sub-8", ExposedAt: now, IsTest: false},
		{ExposureID: "exp-c9", TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: "sub-9", ExposedAt: now, IsTest: false},
		{ExposureID: "exp-c10", TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: "sub-10", ExposedAt: now, IsTest: false},
		{ExposureID: "exp-c11", TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: "sub-11", ExposedAt: now, IsTest: false},
		{ExposureID: "exp-c12", TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: "sub-12", ExposedAt: now, IsTest: false},
	}
	if err := repo.InsertExposures(ctx, exposures); err != nil {
		t.Fatalf("failed to insert exposures: %v", err)
	}

	// Insert conversions (3 conversions for sub-1, sub-2, sub-3 with value 10.0 each)
	conversions := []ConversionFact{
		{ConversionID: "conv-1", TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: "sub-1", MetricName: "purchase", Value: 10.0, ConvertedAt: now, IsTest: false},
		{ConversionID: "conv-2", TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: "sub-2", MetricName: "purchase", Value: 15.0, ConvertedAt: now, IsTest: false},
		{ConversionID: "conv-3", TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: "sub-3", MetricName: "purchase", Value: 5.0, ConvertedAt: now, IsTest: false},
	}
	if err := repo.InsertConversions(ctx, conversions); err != nil {
		t.Fatalf("failed to insert conversions: %v", err)
	}

	query := MetricQuery{
		TenantID:         tenantID,
		ExperimentID:     expID,
		MetricName:       "purchase",
		IsTest:           false,
		StartTime:        now.Add(-time.Hour),
		EndTime:          now,
		MinCellThreshold: 5,
	}

	report, err := repo.QueryReport(ctx, query)
	if err != nil {
		t.Fatalf("QueryReport failed: %v", err)
	}

	if len(report.Variants) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(report.Variants))
	}

	variant := report.Variants[0]
	if variant.VariantID != "control" {
		t.Errorf("expected variant control, got %s", variant.VariantID)
	}

	if variant.Denominators.Exposures != 12 {
		t.Errorf("expected 12 exposures, got %d", variant.Denominators.Exposures)
	}

	if variant.Denominators.Conversions != 3 {
		t.Errorf("expected 3 conversions, got %d", variant.Denominators.Conversions)
	}

	if variant.Denominators.ConversionValue != 30.0 {
		t.Errorf("expected conversion value 30.0, got %f", variant.Denominators.ConversionValue)
	}

	expectedRate := 3.0 / 12.0
	if variant.ConversionRate != expectedRate {
		t.Errorf("expected conversion rate %f, got %f", expectedRate, variant.ConversionRate)
	}
}

func TestDeduplication(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	tenantID := "tenant-beta"
	expID := "exp-002"
	now := time.Now().UTC()

	// Insert duplicate assignments and exposures with the same IDs
	exposures := []ExposureFact{
		{ExposureID: "exp-dup-1", TenantID: tenantID, ExperimentID: expID, VariantID: "treatment", SubjectID: "sub-100", ExposedAt: now, IsTest: false},
		{ExposureID: "exp-dup-1", TenantID: tenantID, ExperimentID: expID, VariantID: "treatment", SubjectID: "sub-100", ExposedAt: now, IsTest: false}, // Duplicate
		{ExposureID: "exp-dup-2", TenantID: tenantID, ExperimentID: expID, VariantID: "treatment", SubjectID: "sub-101", ExposedAt: now, IsTest: false},
	}

	for i := 0; i < 10; i++ {
		exposures = append(exposures, ExposureFact{
			ExposureID:   "exp-unique-" + string(rune('a'+i)),
			TenantID:     tenantID,
			ExperimentID: expID,
			VariantID:    "treatment",
			SubjectID:    "sub-unique-" + string(rune('a'+i)),
			ExposedAt:    now,
			IsTest:       false,
		})
	}

	if err := repo.InsertExposures(ctx, exposures); err != nil {
		t.Fatalf("failed to insert exposures: %v", err)
	}

	query := MetricQuery{
		TenantID:         tenantID,
		ExperimentID:     expID,
		IsTest:           false,
		StartTime:        now.Add(-time.Hour),
		EndTime:          now,
		MinCellThreshold: 5,
	}

	report, err := repo.QueryReport(ctx, query)
	if err != nil {
		t.Fatalf("QueryReport failed: %v", err)
	}

	if len(report.Variants) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(report.Variants))
	}

	// 2 unique duplicate array items + 10 unique items = 12 total exposures (duplicate ignored)
	if report.Variants[0].Denominators.Exposures != 12 {
		t.Errorf("expected 12 deduplicated exposures, got %d", report.Variants[0].Denominators.Exposures)
	}
}

func TestTestModeIsolation(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	tenantID := "tenant-gamma"
	expID := "exp-003"
	now := time.Now().UTC()

	// Insert Prod Facts (is_test = false)
	prodExposures := make([]ExposureFact, 15)
	for i := 0; i < 15; i++ {
		prodExposures[i] = ExposureFact{
			ExposureID:   "prod-exp-" + string(rune('a'+i)),
			TenantID:     tenantID,
			ExperimentID: expID,
			VariantID:    "control",
			SubjectID:    "prod-sub-" + string(rune('a'+i)),
			ExposedAt:    now,
			IsTest:       false,
		}
	}
	if err := repo.InsertExposures(ctx, prodExposures); err != nil {
		t.Fatalf("failed to insert prod exposures: %v", err)
	}

	// Insert Test Facts (is_test = true)
	testExposures := make([]ExposureFact, 25)
	for i := 0; i < 25; i++ {
		testExposures[i] = ExposureFact{
			ExposureID:   "test-exp-" + string(rune('a'+i)),
			TenantID:     tenantID,
			ExperimentID: expID,
			VariantID:    "control",
			SubjectID:    "test-sub-" + string(rune('a'+i)),
			ExposedAt:    now,
			IsTest:       true,
		}
	}
	if err := repo.InsertExposures(ctx, testExposures); err != nil {
		t.Fatalf("failed to insert test exposures: %v", err)
	}

	// Query Prod Mode (IsTest = false)
	prodQuery := MetricQuery{
		TenantID:         tenantID,
		ExperimentID:     expID,
		IsTest:           false,
		StartTime:        now.Add(-time.Hour),
		EndTime:          now,
		MinCellThreshold: 5,
	}
	prodReport, err := repo.QueryReport(ctx, prodQuery)
	if err != nil {
		t.Fatalf("QueryReport for prod failed: %v", err)
	}

	if len(prodReport.Variants) != 1 || prodReport.Variants[0].Denominators.Exposures != 15 {
		t.Errorf("expected 15 prod exposures, got %v", prodReport.Variants)
	}

	// Query Test Mode (IsTest = true)
	testQuery := MetricQuery{
		TenantID:         tenantID,
		ExperimentID:     expID,
		IsTest:           true,
		StartTime:        now.Add(-time.Hour),
		EndTime:          now,
		MinCellThreshold: 5,
	}
	testReport, err := repo.QueryReport(ctx, testQuery)
	if err != nil {
		t.Fatalf("QueryReport for test failed: %v", err)
	}

	if len(testReport.Variants) != 1 || testReport.Variants[0].Denominators.Exposures != 25 {
		t.Errorf("expected 25 test exposures, got %v", testReport.Variants)
	}
}

func TestSmallCellThresholdSuppression(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	tenantID := "tenant-delta"
	expID := "exp-004"
	now := time.Now().UTC()

	// Insert 4 exposures for small-cell variant (threshold is 10)
	smallCellExposures := []ExposureFact{
		{ExposureID: "sc-1", TenantID: tenantID, ExperimentID: expID, VariantID: "variant-small", SubjectID: "s-1", ExposedAt: now, IsTest: false},
		{ExposureID: "sc-2", TenantID: tenantID, ExperimentID: expID, VariantID: "variant-small", SubjectID: "s-2", ExposedAt: now, IsTest: false},
		{ExposureID: "sc-3", TenantID: tenantID, ExperimentID: expID, VariantID: "variant-small", SubjectID: "s-3", ExposedAt: now, IsTest: false},
		{ExposureID: "sc-4", TenantID: tenantID, ExperimentID: expID, VariantID: "variant-small", SubjectID: "s-4", ExposedAt: now, IsTest: false},
	}
	if err := repo.InsertExposures(ctx, smallCellExposures); err != nil {
		t.Fatalf("failed to insert exposures: %v", err)
	}

	query := MetricQuery{
		TenantID:         tenantID,
		ExperimentID:     expID,
		IsTest:           false,
		StartTime:        now.Add(-time.Hour),
		EndTime:          now,
		MinCellThreshold: 10, // Threshold 10 > 4
	}

	report, err := repo.QueryReport(ctx, query)
	if err != nil {
		t.Fatalf("QueryReport failed: %v", err)
	}

	if len(report.Variants) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(report.Variants))
	}

	variant := report.Variants[0]
	if !variant.IsSuppressed {
		t.Errorf("expected variant to be suppressed due to small cell threshold")
	}

	if variant.Denominators.Exposures != 0 {
		t.Errorf("expected suppressed exposures to be masked to 0, got %d", variant.Denominators.Exposures)
	}

	if variant.ConversionRate != 0.0 {
		t.Errorf("expected suppressed conversion rate to be 0.0, got %f", variant.ConversionRate)
	}
}

func TestWatermarkAndProcessingLag(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	tenantID := "tenant-epsilon"
	expID := "exp-005"
	eventTime := time.Now().UTC().Add(-30 * time.Minute)

	exposures := []ExposureFact{
		{ExposureID: "wm-1", TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: "sub-wm1", ExposedAt: eventTime, IsTest: false},
	}
	if err := repo.InsertExposures(ctx, exposures); err != nil {
		t.Fatalf("failed to insert exposure: %v", err)
	}

	watermark, err := repo.GetWatermark(ctx, tenantID, expID, "", false)
	if err != nil {
		t.Fatalf("GetWatermark failed: %v", err)
	}

	if !watermark.Equal(eventTime) {
		t.Errorf("expected watermark %v, got %v", eventTime, watermark)
	}

	query := MetricQuery{
		TenantID:     tenantID,
		ExperimentID: expID,
		IsTest:       false,
		StartTime:    eventTime.Add(-time.Hour),
		EndTime:      eventTime.Add(10 * time.Minute), // EndTime after watermark => provisional
	}

	report, err := repo.QueryReport(ctx, query)
	if err != nil {
		t.Fatalf("QueryReport failed: %v", err)
	}

	if report.WindowStatus != WindowStatusProvisional {
		t.Errorf("expected WindowStatusProvisional, got %s", report.WindowStatus)
	}

	if report.ProcessingLag <= 0 {
		t.Errorf("expected positive processing lag, got %v", report.ProcessingLag)
	}
}

func TestProvisionalVsFinalWindowStatus(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	tenantID := "tenant-zeta"
	expID := "exp-006"
	eventTime := time.Now().UTC().Add(-2 * time.Hour)

	exposures := []ExposureFact{
		{ExposureID: "wm-final-1", TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: "sub-1", ExposedAt: eventTime, IsTest: false},
	}
	_ = repo.InsertExposures(ctx, exposures)

	// EndTime is before/equal eventTime => WindowStatusFinal
	queryFinal := MetricQuery{
		TenantID:     tenantID,
		ExperimentID: expID,
		IsTest:       false,
		StartTime:    eventTime.Add(-time.Hour),
		EndTime:      eventTime,
	}

	reportFinal, err := repo.QueryReport(ctx, queryFinal)
	if err != nil {
		t.Fatalf("QueryReport failed: %v", err)
	}

	if reportFinal.WindowStatus != WindowStatusFinal {
		t.Errorf("expected WindowStatusFinal, got %s", reportFinal.WindowStatus)
	}
}

func TestSampleRatioMismatchEvaluation(t *testing.T) {
	chRepo := &ClickHouseRepository{}

	// Case 1: Normal SRM (Equal distribution: 50 control, 50 treatment)
	variantsNormal := []VariantMetricReport{
		{VariantID: "control", Denominators: AggregateDenominators{Exposures: 50}},
		{VariantID: "treatment", Denominators: AggregateDenominators{Exposures: 50}},
	}
	resNormal := chRepo.evaluateSampleRatio(variantsNormal, nil, 10)
	if resNormal.Status != SampleRatioStatusNormal {
		t.Errorf("expected SampleRatioStatusNormal, got %s", resNormal.Status)
	}

	// Case 2: Mismatch Detected (Severe imbalance: 80 control, 20 treatment)
	variantsMismatch := []VariantMetricReport{
		{VariantID: "control", Denominators: AggregateDenominators{Exposures: 80}},
		{VariantID: "treatment", Denominators: AggregateDenominators{Exposures: 20}},
	}
	resMismatch := chRepo.evaluateSampleRatio(variantsMismatch, nil, 10)
	if resMismatch.Status != SampleRatioStatusMismatch {
		t.Errorf("expected SampleRatioStatusMismatch, got %s (chi2=%f)", resMismatch.Status, resMismatch.ChiSquare)
	}
}

func TestExclusionsReporting(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	tenantID := "tenant-eta"
	now := time.Now().UTC()

	qualityFacts := []DataQualityFact{
		{QualityEventID: "q1", TenantID: tenantID, IssueType: "filter_mismatch", OccurredAt: now, IsTest: false},
		{QualityEventID: "q2", TenantID: tenantID, IssueType: "quality_issue", OccurredAt: now, IsTest: false},
		{QualityEventID: "q3", TenantID: tenantID, IssueType: "schema_invalid", OccurredAt: now, IsTest: false},
		{QualityEventID: "q4", TenantID: tenantID, IssueType: "late_watermark", OccurredAt: now, IsTest: false},
	}
	if err := repo.InsertDataQuality(ctx, qualityFacts); err != nil {
		t.Fatalf("failed to insert data quality facts: %v", err)
	}

	query := MetricQuery{
		TenantID:  tenantID,
		IsTest:    false,
		StartTime: now.Add(-time.Hour),
		EndTime:   now,
	}

	report, err := repo.QueryReport(ctx, query)
	if err != nil {
		t.Fatalf("QueryReport failed: %v", err)
	}

	if report.Exclusions.FilterMismatch != 1 {
		t.Errorf("expected 1 filter mismatch exclusion, got %d", report.Exclusions.FilterMismatch)
	}
	if report.Exclusions.QualityIssues != 2 {
		t.Errorf("expected 2 quality issues exclusions, got %d", report.Exclusions.QualityIssues)
	}
	if report.Exclusions.LateWatermark != 1 {
		t.Errorf("expected 1 late watermark exclusion, got %d", report.Exclusions.LateWatermark)
	}
	if report.Exclusions.TotalExcluded != 4 {
		t.Errorf("expected 4 total exclusions, got %d", report.Exclusions.TotalExcluded)
	}
}
