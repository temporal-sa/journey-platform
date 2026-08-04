package experiments

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/store/analytics"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
	"github.com/validated-pattern/journey-platform/internal/testaudience"
)

// Helper to construct a valid multi-variant test experiment.
func helperIsolationExperiment(id string, salt string, variants []Variant) *Experiment {
	return &Experiment{
		SchemaVersion:     "1.0",
		TenantID:          "tenant-isolation",
		ExperimentID:      id,
		Version:           1,
		Name:              "Isolation Correctness Test Experiment",
		Status:            ExperimentStatusActive,
		Salt:              salt,
		RandomizationUnit: UnitUserID,
		Variants:          variants,
		Metrics: []MetricDefinition{
			{
				Key:       "met-conversion",
				Name:      "Conversion Metric",
				EventType: "user_signup",
				Type:      "conversion",
			},
		},
		AttributionWindowSeconds: 86400,
	}
}

// 1. Allocation & Version Isolation Test Suite
func TestAllocationVectorsAndVersionIsolation(t *testing.T) {
	ctx := context.Background()

	t.Run("50/50 Allocation Vector and Distribution", func(t *testing.T) {
		exp5050 := helperIsolationExperiment("exp-5050", "salt-5050", []Variant{
			{Key: "control", WeightBasisPoints: 5000, IsControl: true},
			{Key: "treatment", WeightBasisPoints: 5000, IsControl: false},
		})

		if err := ValidateExperiment(exp5050); err != nil {
			t.Fatalf("50/50 experiment invalid: %v", err)
		}

		counts := make(map[string]int)
		totalUnits := 10000
		for i := 0; i < totalUnits; i++ {
			unitID := fmt.Sprintf("usr-5050-%d", i)
			variant, bucket, err := SelectVariant(exp5050, unitID)
			if err != nil {
				t.Fatalf("SelectVariant failed: %v", err)
			}
			if bucket >= 10000 {
				t.Fatalf("bucket out of range [0, 9999]: %d", bucket)
			}
			counts[variant.Key]++
		}

		// Expect ~5000 each (within ±300 tolerance)
		for _, v := range exp5050.Variants {
			cnt := counts[v.Key]
			if cnt < 4700 || cnt > 5300 {
				t.Errorf("Variant %s count %d outside expected tolerance range [4700, 5300]", v.Key, cnt)
			}
		}
	})

	t.Run("70/30 Allocation Vector and Distribution", func(t *testing.T) {
		exp7030 := helperIsolationExperiment("exp-7030", "salt-7030", []Variant{
			{Key: "control", WeightBasisPoints: 7000, IsControl: true},
			{Key: "treatment", WeightBasisPoints: 3000, IsControl: false},
		})

		if err := ValidateExperiment(exp7030); err != nil {
			t.Fatalf("70/30 experiment invalid: %v", err)
		}

		counts := make(map[string]int)
		totalUnits := 10000
		for i := 0; i < totalUnits; i++ {
			unitID := fmt.Sprintf("usr-7030-%d", i)
			variant, _, err := SelectVariant(exp7030, unitID)
			if err != nil {
				t.Fatalf("SelectVariant failed: %v", err)
			}
			counts[variant.Key]++
		}

		if counts["control"] < 6700 || counts["control"] > 7300 {
			t.Errorf("Control count %d outside expected range [6700, 7300]", counts["control"])
		}
		if counts["treatment"] < 2700 || counts["treatment"] > 3300 {
			t.Errorf("Treatment count %d outside expected range [2700, 3300]", counts["treatment"])
		}
	})

	t.Run("Multi-Variant Allocation Vector (50/30/20)", func(t *testing.T) {
		expMulti := helperIsolationExperiment("exp-multi", "salt-multi", []Variant{
			{Key: "control", WeightBasisPoints: 5000, IsControl: true},
			{Key: "variant_a", WeightBasisPoints: 3000, IsControl: false},
			{Key: "variant_b", WeightBasisPoints: 2000, IsControl: false},
		})

		if err := ValidateExperiment(expMulti); err != nil {
			t.Fatalf("Multi-variant experiment invalid: %v", err)
		}

		counts := make(map[string]int)
		totalUnits := 10000
		for i := 0; i < totalUnits; i++ {
			unitID := fmt.Sprintf("usr-multi-%d", i)
			variant, _, err := SelectVariant(expMulti, unitID)
			if err != nil {
				t.Fatalf("SelectVariant failed: %v", err)
			}
			counts[variant.Key]++
		}

		if counts["control"] < 4700 || counts["control"] > 5300 {
			t.Errorf("Control count %d outside expected range [4700, 5300]", counts["control"])
		}
		if counts["variant_a"] < 2700 || counts["variant_a"] > 3300 {
			t.Errorf("Variant A count %d outside expected range [2700, 3300]", counts["variant_a"])
		}
		if counts["variant_b"] < 1700 || counts["variant_b"] > 2300 {
			t.Errorf("Variant B count %d outside expected range [1700, 2300]", counts["variant_b"])
		}
	})

	t.Run("Sticky Retries Idempotency", func(t *testing.T) {
		repo := postgres.NewMemoryRepository()
		svc := NewAssignmentService(repo)
		exp := helperIsolationExperiment("exp-sticky", "salt-sticky", []Variant{
			{Key: "control", WeightBasisPoints: 5000, IsControl: true},
			{Key: "treatment", WeightBasisPoints: 5000, IsControl: false},
		})

		req := AssignmentRequest{
			TenantID:   "tenant-sticky",
			Experiment: exp,
			SubjectID:  "subject-sticky-1",
			Mode:       ModeProduction,
		}

		// Perform initial assignment
		firstAsg, err := svc.GetOrAssign(ctx, req)
		if err != nil {
			t.Fatalf("Initial GetOrAssign failed: %v", err)
		}

		// Perform 10 consecutive retry calls
		for i := 0; i < 10; i++ {
			retryAsg, err := svc.GetOrAssign(ctx, req)
			if err != nil {
				t.Fatalf("Retry GetOrAssign #%d failed: %v", i, err)
			}
			if retryAsg.AssignmentID != firstAsg.AssignmentID {
				t.Errorf("Retry #%d: AssignmentID changed from %s to %s", i, firstAsg.AssignmentID, retryAsg.AssignmentID)
			}
			if retryAsg.VariantID != firstAsg.VariantID {
				t.Errorf("Retry #%d: VariantID changed from %s to %s", i, firstAsg.VariantID, retryAsg.VariantID)
			}
		}
	})

	t.Run("Subject Re-entry Sticky Persistence", func(t *testing.T) {
		repo := postgres.NewMemoryRepository()
		svc := NewAssignmentService(repo)
		exp := helperIsolationExperiment("exp-reentry", "salt-reentry", []Variant{
			{Key: "control", WeightBasisPoints: 5000, IsControl: true},
			{Key: "treatment", WeightBasisPoints: 5000, IsControl: false},
		})

		req := AssignmentRequest{
			TenantID:   "tenant-reentry",
			Experiment: exp,
			SubjectID:  "subject-reentry-42",
			Mode:       ModeProduction,
		}

		// First entry
		asg1, err := svc.GetOrAssign(ctx, req)
		if err != nil {
			t.Fatalf("First entry assignment failed: %v", err)
		}

		// Re-entry with additional context metadata
		reqReentry := req
		reqReentry.Context = map[string]interface{}{"session": "session-2", "reentry": true}
		asg2, err := svc.GetOrAssign(ctx, reqReentry)
		if err != nil {
			t.Fatalf("Re-entry assignment failed: %v", err)
		}

		if asg2.AssignmentID != asg1.AssignmentID {
			t.Errorf("Subject re-entry produced new AssignmentID: expected %s, got %s", asg1.AssignmentID, asg2.AssignmentID)
		}
		if asg2.VariantID != asg1.VariantID {
			t.Errorf("Subject re-entry changed variant: expected %s, got %s", asg1.VariantID, asg2.VariantID)
		}
		if asg2.Context["session"] != "session-2" {
			t.Errorf("Expected context update on re-entry, got %v", asg2.Context)
		}
	})

	t.Run("New-Version Isolation and Version Increment", func(t *testing.T) {
		repo := postgres.NewMemoryRepository()
		svc := NewAssignmentService(repo)

		v1 := helperIsolationExperiment("exp-versioned", "salt-versioned", []Variant{
			{Key: "control", WeightBasisPoints: 5000, IsControl: true},
			{Key: "treatment", WeightBasisPoints: 5000, IsControl: false},
		})
		v1.Version = 1
		h1, _ := v1.CalculateContentHash()
		v1.ContentHash = h1

		// Assign subject to V1
		asgV1, err := svc.GetOrAssign(ctx, AssignmentRequest{
			TenantID:   "tenant-ver",
			Experiment: v1,
			SubjectID:  "subject-version-test",
			Mode:       ModeProduction,
		})
		if err != nil {
			t.Fatalf("V1 assignment failed: %v", err)
		}

		// Create V2 with mutated allocation weights (7000/3000)
		v2Mutated := v1.Clone()
		v2Mutated.Variants[0].WeightBasisPoints = 7000
		v2Mutated.Variants[1].WeightBasisPoints = 3000

		v2, err := CreateNextVersion(v1, v2Mutated)
		if err != nil {
			t.Fatalf("CreateNextVersion failed: %v", err)
		}

		if v2.Version != 2 {
			t.Fatalf("expected version 2, got %d", v2.Version)
		}
		if v2.ContentHash == v1.ContentHash {
			t.Fatalf("expected content hash to change between versions")
		}

		// Verify versioned keys use distinct version suffixes
		vk1 := VersionedKey(v1.Salt, v1.Version)
		vk2 := VersionedKey(v2.Salt, v2.Version)
		if vk1 != "salt-versioned:v1" || vk2 != "salt-versioned:v2" {
			t.Errorf("Versioned keys mismatch: vk1=%s, vk2=%s", vk1, vk2)
		}

		// Immutability validation: modifying V1 payload without version increment must fail
		if err := ValidateImmutability(v1, v2Mutated); err == nil {
			t.Errorf("expected ValidateImmutability error when version is not incremented")
		}

		// V1 sticky assignment remains untouched
		asgV1Requery, err := svc.GetOrAssign(ctx, AssignmentRequest{
			TenantID:   "tenant-ver",
			Experiment: v1,
			SubjectID:  "subject-version-test",
			Mode:       ModeProduction,
		})
		if err != nil {
			t.Fatalf("V1 re-query failed: %v", err)
		}
		if asgV1Requery.AssignmentID != asgV1.AssignmentID {
			t.Errorf("V1 sticky assignment altered by V2 creation: expected %s, got %s", asgV1.AssignmentID, asgV1Requery.AssignmentID)
		}
	})
}

// 2. Metric Attribution & Event Isolation Test Suite
func TestMetricAttributionAndEventIsolation(t *testing.T) {
	ctx := context.Background()

	t.Run("Assignment vs Exposure and Action-Conditioned Metrics", func(t *testing.T) {
		analyticsRepo := analytics.NewMemoryRepository()
		materializer := NewReportMaterializer(analyticsRepo)

		tenantID := "tenant-metrics"
		expID := "exp-metrics"
		now := time.Now().UTC()

		// Scenario: 100 assigned subjects, only 50 exposed subjects, 25 conversions
		for i := 1; i <= 100; i++ {
			subjID := fmt.Sprintf("subj-%d", i)
			_ = analyticsRepo.InsertAssignments(ctx, []analytics.AssignmentFact{
				{
					AssignmentID:      fmt.Sprintf("asg-%d", i),
					TenantID:          tenantID,
					ExperimentID:      expID,
					ExperimentVersion: 1,
					VariantID:         "treatment",
					SubjectID:         subjID,
					IsTest:            false,
					AssignedAt:        now,
				},
			})
			// Only subjects 1-50 are exposed
			if i <= 50 {
				_ = analyticsRepo.InsertExposures(ctx, []analytics.ExposureFact{
					{
						ExposureID:        fmt.Sprintf("exp-%d", i),
						AssignmentID:      fmt.Sprintf("asg-%d", i),
						TenantID:          tenantID,
						ExperimentID:      expID,
						ExperimentVersion: 1,
						VariantID:         "treatment",
						SubjectID:         subjID,
						IsTest:            false,
						ExposedAt:         now,
					},
				})
			}
			// Only subjects 1-25 convert
			if i <= 25 {
				_ = analyticsRepo.InsertConversions(ctx, []analytics.ConversionFact{
					{
						ConversionID:      fmt.Sprintf("conv-%d", i),
						TenantID:          tenantID,
						ExperimentID:      expID,
						ExperimentVersion: 1,
						VariantID:         "treatment",
						SubjectID:         subjID,
						MetricName:        "conversion",
						Value:             1.0,
						IsTest:            false,
						ConvertedAt:       now,
					},
				})
			}
		}

		// Query Intent-To-Treat (ITT) Report: Denominator = "assigned"
		reportITT, err := materializer.MaterializeReport(ctx, ReportQuery{
			TenantID:          tenantID,
			ExperimentID:      expID,
			ExperimentVersion: 1,
			MetricName:        "conversion",
			DenominatorName:   "assigned",
			Mode:              ModeProduction,
			MinSampleSize:     1,
		})
		if err != nil {
			t.Fatalf("MaterializeReport ITT failed: %v", err)
		}

		statsITT := reportITT.Variants["treatment"]
		if statsITT.SampleSize != 100 {
			t.Errorf("ITT sample size expected 100, got %d", statsITT.SampleSize)
		}
		// ITT conversion rate = 25 / 100 = 0.25
		if math.Abs(statsITT.AbsoluteRate-0.25) > 1e-6 {
			t.Errorf("ITT rate expected 0.25, got %f", statsITT.AbsoluteRate)
		}

		// Query Action-Conditioned Report: Denominator = "exposed"
		reportAction, err := materializer.MaterializeReport(ctx, ReportQuery{
			TenantID:          tenantID,
			ExperimentID:      expID,
			ExperimentVersion: 1,
			MetricName:        "conversion",
			DenominatorName:   "exposed",
			Mode:              ModeProduction,
			MinSampleSize:     1,
		})
		if err != nil {
			t.Fatalf("MaterializeReport Action-conditioned failed: %v", err)
		}

		statsAction := reportAction.Variants["treatment"]
		if statsAction.SampleSize != 50 {
			t.Errorf("Action-conditioned sample size expected 50, got %d", statsAction.SampleSize)
		}
		// Action-conditioned rate = 25 / 50 = 0.50
		if math.Abs(statsAction.AbsoluteRate-0.50) > 1e-6 {
			t.Errorf("Action-conditioned rate expected 0.50, got %f", statsAction.AbsoluteRate)
		}
	})

	t.Run("Duplicate and Late Outcomes Handling", func(t *testing.T) {
		analyticsRepo := analytics.NewMemoryRepository()
		materializer := NewReportMaterializer(analyticsRepo)

		tenantID := "tenant-dups"
		expID := "exp-dups"
		now := time.Now().UTC()

		// Insert duplicate assignment facts for same assignment ID
		_ = analyticsRepo.InsertAssignments(ctx, []analytics.AssignmentFact{
			{AssignmentID: "asg-dup-1", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, VariantID: "control", SubjectID: "subj-dup", AssignedAt: now},
			{AssignmentID: "asg-dup-1", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, VariantID: "control", SubjectID: "subj-dup", AssignedAt: now},
		})

		// Insert duplicate conversion facts for same conversion ID
		_ = analyticsRepo.InsertConversions(ctx, []analytics.ConversionFact{
			{ConversionID: "conv-dup-1", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, VariantID: "control", SubjectID: "subj-dup", MetricName: "conversion", Value: 1.0, ConvertedAt: now},
			{ConversionID: "conv-dup-1", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, VariantID: "control", SubjectID: "subj-dup", MetricName: "conversion", Value: 1.0, ConvertedAt: now},
		})

		// Insert DataQualityFact for late watermark and duplicate quality issues
		_ = analyticsRepo.InsertDataQuality(ctx, []analytics.DataQualityFact{
			{QualityEventID: "dq-late", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, IssueType: "late_watermark", OccurredAt: now},
			{QualityEventID: "dq-dup", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, IssueType: "duplicate", OccurredAt: now},
		})

		report, err := materializer.MaterializeReport(ctx, ReportQuery{
			TenantID:          tenantID,
			ExperimentID:      expID,
			ExperimentVersion: 1,
			MetricName:        "conversion",
			DenominatorName:   "assigned",
			Mode:              ModeProduction,
			MinSampleSize:     1,
		})
		if err != nil {
			t.Fatalf("MaterializeReport failed: %v", err)
		}

		// Deduplication check: 1 unique assignment & 1 unique conversion
		vStats := report.Variants["control"]
		if vStats.Denominators.Assigned != 1 {
			t.Errorf("Expected 1 assigned denominator after deduplication, got %d", vStats.Denominators.Assigned)
		}
		if vStats.Denominators.Conversion != 1 {
			t.Errorf("Expected 1 conversion denominator after deduplication, got %d", vStats.Denominators.Conversion)
		}

		// Exclusion tracking check
		if report.Metadata.Exclusions.LateWatermark != 1 {
			t.Errorf("Expected 1 LateWatermark exclusion, got %d", report.Metadata.Exclusions.LateWatermark)
		}
		if report.Metadata.Exclusions.QualityIssues != 1 {
			t.Errorf("Expected 1 QualityIssues exclusion, got %d", report.Metadata.Exclusions.QualityIssues)
		}
	})

	t.Run("Attribution Boundaries Filtering", func(t *testing.T) {
		analyticsRepo := analytics.NewMemoryRepository()
		materializer := NewReportMaterializer(analyticsRepo)

		tenantID := "tenant-attr"
		expTarget := "exp-target"
		expOther := "exp-other"
		now := time.Now().UTC()

		// Facts for target experiment
		_ = analyticsRepo.InsertExposures(ctx, []analytics.ExposureFact{
			{ExposureID: "exp-target-1", TenantID: tenantID, ExperimentID: expTarget, ExperimentVersion: 1, VariantID: "control", SubjectID: "s1", ExposedAt: now},
		})

		// Facts for different experiment (exp-other)
		_ = analyticsRepo.InsertExposures(ctx, []analytics.ExposureFact{
			{ExposureID: "exp-other-1", TenantID: tenantID, ExperimentID: expOther, ExperimentVersion: 1, VariantID: "control", SubjectID: "s2", ExposedAt: now},
		})

		// Facts for different tenant
		_ = analyticsRepo.InsertExposures(ctx, []analytics.ExposureFact{
			{ExposureID: "exp-foreign-tenant", TenantID: "other-tenant", ExperimentID: expTarget, ExperimentVersion: 1, VariantID: "control", SubjectID: "s3", ExposedAt: now},
		})

		report, err := materializer.MaterializeReport(ctx, ReportQuery{
			TenantID:          tenantID,
			ExperimentID:      expTarget,
			ExperimentVersion: 1,
			MetricName:        "conversion",
			DenominatorName:   "exposed",
			Mode:              ModeProduction,
			MinSampleSize:     1,
		})
		if err != nil {
			t.Fatalf("MaterializeReport failed: %v", err)
		}

		if len(report.Variants) != 1 {
			t.Fatalf("Expected 1 variant in report, got %d", len(report.Variants))
		}
		if report.Variants["control"].Denominators.Exposed != 1 {
			t.Errorf("Expected 1 exposed subject within attribution boundaries, got %d", report.Variants["control"].Denominators.Exposed)
		}
	})

	t.Run("Privacy Proxies and Link Scanners Data Quality Isolation", func(t *testing.T) {
		analyticsRepo := analytics.NewMemoryRepository()
		materializer := NewReportMaterializer(analyticsRepo)

		tenantID := "tenant-proxy"
		expID := "exp-proxy"
		now := time.Now().UTC()

		// Genuine subject exposures
		_ = analyticsRepo.InsertExposures(ctx, []analytics.ExposureFact{
			{ExposureID: "exp-gen-1", TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: "user-real-1", ExposedAt: now},
			{ExposureID: "exp-gen-2", TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: "user-real-2", ExposedAt: now},
		})

		// Privacy proxy opens & link scanner clicks logged to data quality partition
		_ = analyticsRepo.InsertDataQuality(ctx, []analytics.DataQualityFact{
			{
				QualityEventID: "dq-proxy-1",
				TenantID:       tenantID,
				ExperimentID:   expID,
				EntityType:     "open",
				EntityID:       "open-bot-1",
				IssueType:      "filter_mismatch",
				Details:        "Apple Mail Privacy Protection pre-fetch proxy open detected",
				OccurredAt:     now,
			},
			{
				QualityEventID: "dq-scanner-1",
				TenantID:       tenantID,
				ExperimentID:   expID,
				EntityType:     "click",
				EntityID:       "click-bot-1",
				IssueType:      "quality_issue",
				Details:        "Automated link scanner bot click filtered out",
				OccurredAt:     now,
			},
		})

		report, err := materializer.MaterializeReport(ctx, ReportQuery{
			TenantID:          tenantID,
			ExperimentID:      expID,
			MetricName:        "conversion",
			DenominatorName:   "exposed",
			Mode:              ModeProduction,
			MinSampleSize:     1,
		})
		if err != nil {
			t.Fatalf("MaterializeReport failed: %v", err)
		}

		// Verify genuine metrics are unpolluted
		if report.Variants["control"].Denominators.Exposed != 2 {
			t.Errorf("Expected 2 genuine exposures, got %d", report.Variants["control"].Denominators.Exposed)
		}

		// Verify exclusions record privacy proxy & link scanner events
		if report.Metadata.Exclusions.FilterMismatch != 1 {
			t.Errorf("Expected 1 FilterMismatch exclusion for privacy proxy, got %d", report.Metadata.Exclusions.FilterMismatch)
		}
		if report.Metadata.Exclusions.QualityIssues != 1 {
			t.Errorf("Expected 1 QualityIssues exclusion for link scanner, got %d", report.Metadata.Exclusions.QualityIssues)
		}
		if report.Metadata.Exclusions.TotalExcluded != 2 {
			t.Errorf("Expected 2 TotalExcluded, got %d", report.Metadata.Exclusions.TotalExcluded)
		}
	})

	t.Run("SRM (Sample Ratio Mismatch) Detection", func(t *testing.T) {
		// Test Normal SRM
		normalCounts := map[string]int64{"control": 1000, "treatment": 1000}
		srmNormal := CalculateSRM(normalCounts, nil, 30)
		if srmNormal.Status != "normal" || srmNormal.ChiSquare != 0.0 || srmNormal.PValue < 0.05 {
			t.Errorf("Normal SRM failed: status=%s, chi2=%f, p=%f", srmNormal.Status, srmNormal.ChiSquare, srmNormal.PValue)
		}

		// Test Mismatch SRM (800 control vs 200 treatment on expected 50/50)
		mismatchCounts := map[string]int64{"control": 800, "treatment": 200}
		srmMismatch := CalculateSRM(mismatchCounts, nil, 30)
		if srmMismatch.Status != "mismatch_detected" {
			t.Errorf("Expected SRM status 'mismatch_detected', got %s", srmMismatch.Status)
		}
		if srmMismatch.ChiSquare <= 10.83 {
			t.Errorf("Expected chi-square > 10.83, got %f", srmMismatch.ChiSquare)
		}
		if srmMismatch.PValue >= 0.001 {
			t.Errorf("Expected p-value < 0.001, got %f", srmMismatch.PValue)
		}

		// Test Skewed SRM with custom expected weights (70/30)
		weights7030 := map[string]float64{"control": 0.70, "treatment": 0.30}
		counts7030 := map[string]int64{"control": 700, "treatment": 300}
		srmCustom := CalculateSRM(counts7030, weights7030, 30)
		if srmCustom.Status != "normal" {
			t.Errorf("Expected normal SRM for 70/30 actual vs 70/30 expected, got %s", srmCustom.Status)
		}
	})
}

// 3. Static Test Runs & Execution Controls Test Suite
func TestStaticTestRunsAndExecutionControls(t *testing.T) {
	ctx := context.Background()

	setupCoordinator := func(t *testing.T, memberCount int) (*testaudience.Coordinator, *testaudience.Evaluator, *testaudience.StaticList) {
		eval := testaudience.New()
		members := make([]testaudience.Member, memberCount)
		for i := 0; i < memberCount; i++ {
			mID := fmt.Sprintf("member-%d", i+1)
			mKey := testaudience.ResolveMemberKey("tenant-static", mID)
			recip := fmt.Sprintf("user%d@example.com", i+1)
			members[i] = testaudience.Member{
				MemberKey:          mKey,
				MemberID:           mID,
				Recipient:          recip,
				MaskedDisplayValue: testaudience.MaskDisplayValue(recip),
			}
		}

		list := &testaudience.StaticList{
			ListID:   "list-static-1",
			TenantID: "tenant-static",
			Members:  members,
		}
		list.Finalize()
		eval.RegisterList(list)

		store := testaudience.NewInMemoryStorage()
		coord := testaudience.NewCoordinator(eval, store)
		return coord, eval, list
	}

	t.Run("Realistic vs Forced Variant-Coverage Static Runs", func(t *testing.T) {
		coord, _, list := setupCoordinator(t, 10)

		// 1. Realistic Mode Run
		runReal, tgtsReal, err := coord.StartTestRun(ctx, testaudience.TestRunInput{
			TenantID:        "tenant-static",
			CandidateIRHash: "ir-hash-100",
			ListVersionHash: list.ContentHash,
			StaticListID:    list.ListID,
			FixturePackHash: "fix-hash-100",
			AssignmentMode:  testaudience.AssignmentModeRealistic,
			Variants:        []string{"control", "variant_a", "variant_b"},
		})
		if err != nil {
			t.Fatalf("StartTestRun realistic failed: %v", err)
		}
		if runReal.AssignmentMode != testaudience.AssignmentModeRealistic {
			t.Errorf("Expected realistic mode, got %s", runReal.AssignmentMode)
		}
		if len(tgtsReal) != 10 {
			t.Fatalf("Expected 10 targets, got %d", len(tgtsReal))
		}

		// 2. Forced Variant-Coverage Mode Run
		runForced, tgtsForced, err := coord.StartTestRun(ctx, testaudience.TestRunInput{
			TenantID:        "tenant-static",
			CandidateIRHash: "ir-hash-100",
			ListVersionHash: list.ContentHash,
			StaticListID:    list.ListID,
			FixturePackHash: "fix-hash-100",
			AssignmentMode:  testaudience.AssignmentModeForcedVariantCoverage,
			Variants:        []string{"control", "variant_a", "variant_b"},
		})
		if err != nil {
			t.Fatalf("StartTestRun forced coverage failed: %v", err)
		}
		if runForced.AssignmentMode != testaudience.AssignmentModeForcedVariantCoverage {
			t.Errorf("Expected forced variant coverage mode, got %s", runForced.AssignmentMode)
		}

		// Assert forced coverage guarantees exact round-robin variant distribution
		forcedCounts := make(map[string]int)
		for i, tgt := range tgtsForced {
			forcedCounts[tgt.VariantID]++
			expectedVar := []string{"control", "variant_a", "variant_b"}[i%3]
			if tgt.VariantID != expectedVar {
				t.Errorf("Target #%d forced variant mismatch: expected %s, got %s", i, expectedVar, tgt.VariantID)
			}
		}
		if forcedCounts["control"] != 4 || forcedCounts["variant_a"] != 3 || forcedCounts["variant_b"] != 3 {
			t.Errorf("Forced variant counts mismatch: %v", forcedCounts)
		}
	})

	t.Run("Partial Small-Cell Threshold Suppression", func(t *testing.T) {
		analyticsRepo := analytics.NewMemoryRepository()

		tenantID := "tenant-suppress"
		expID := "exp-suppress"
		now := time.Now().UTC()

		// Insert 5 exposures for control (below threshold of 10)
		for i := 1; i <= 5; i++ {
			_ = analyticsRepo.InsertExposures(ctx, []analytics.ExposureFact{
				{ExposureID: fmt.Sprintf("e-sup-%d", i), TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: fmt.Sprintf("s-%d", i), ExposedAt: now},
			})
		}

		// Query analytics memory repository directly with MinCellThreshold = 10
		anReport, err := analyticsRepo.QueryReport(ctx, analytics.MetricQuery{
			TenantID:         tenantID,
			ExperimentID:     expID,
			MinCellThreshold: 10,
		})
		if err != nil {
			t.Fatalf("analyticsRepo.QueryReport failed: %v", err)
		}

		if len(anReport.Variants) != 1 {
			t.Fatalf("Expected 1 variant report, got %d", len(anReport.Variants))
		}

		vr := anReport.Variants[0]

		// Assert small-cell suppression rules applied
		if !vr.IsSuppressed {
			t.Errorf("Expected variant report to be suppressed for count 5 < threshold 10")
		}
		if vr.ConversionRate != 0.0 {
			t.Errorf("Expected suppressed conversion rate to be 0.0, got %f", vr.ConversionRate)
		}
		if !anReport.SampleRatio.IsSuppressed {
			t.Errorf("Expected SRM result to be suppressed when variant is suppressed")
		}

		// Also verify ReportMaterializer warning flag when sample size < MinSampleSize
		materializer := NewReportMaterializer(analyticsRepo)
		report, err := materializer.MaterializeReport(ctx, ReportQuery{
			TenantID:      tenantID,
			ExperimentID:  expID,
			Mode:          ModeProduction,
			MinSampleSize: 10,
		})
		if err != nil {
			t.Fatalf("MaterializeReport failed: %v", err)
		}

		vStats, ok := report.Variants["control"]
		if !ok {
			t.Fatalf("Variant 'control' missing from report")
		}

		if !vStats.InsufficientSample {
			t.Errorf("Expected InsufficientSample flag to be true when sample size 5 < MinSampleSize 10")
		}
		if !report.Metadata.InsufficientSampleWarning {
			t.Errorf("Expected InsufficientSampleWarning to be true in metadata")
		}
	})

	t.Run("Test Run Cancellation and Expiry Controls", func(t *testing.T) {
		coord, _, list := setupCoordinator(t, 5)

		// 1. Cancellation Test
		runCancel, tgtsCancel, err := coord.StartTestRun(ctx, testaudience.TestRunInput{
			TenantID:        "tenant-static",
			CandidateIRHash: "ir-cancel",
			ListVersionHash: list.ContentHash,
			StaticListID:    list.ListID,
			FixturePackHash: "fix-cancel",
		})
		if err != nil {
			t.Fatalf("StartTestRun failed: %v", err)
		}

		// Dispatch target 0 before cancellation
		_, err = coord.DispatchTarget(ctx, runCancel.TestRunID, tgtsCancel[0].TargetID, "node-1", "email_dispatch")
		if err != nil {
			t.Fatalf("DispatchTarget failed: %v", err)
		}

		// Cancel test run
		cancelledRun, err := coord.CancelTestRun(ctx, runCancel.TestRunID, "operator manual stop")
		if err != nil {
			t.Fatalf("CancelTestRun failed: %v", err)
		}

		if cancelledRun.Status != testaudience.TestRunStatusCancelled {
			t.Errorf("Expected run status 'cancelled', got %s", cancelledRun.Status)
		}
		if cancelledRun.CancelledCount != 4 {
			t.Errorf("Expected 4 cancelled pending targets, got %d", cancelledRun.CancelledCount)
		}

		// Attempting to dispatch remaining cancelled targets must fail
		_, err = coord.DispatchTarget(ctx, runCancel.TestRunID, tgtsCancel[1].TargetID, "node-1", "email_dispatch")
		if err == nil || !errors.Is(err, testaudience.ErrTestRunTerminated) {
			t.Errorf("Expected ErrTestRunTerminated when dispatching cancelled target, got %v", err)
		}

		// 2. Expiry Handling Test
		runExpiry, tgtsExpiry, err := coord.StartTestRun(ctx, testaudience.TestRunInput{
			TenantID:        "tenant-static",
			CandidateIRHash: "ir-expiry",
			ListVersionHash: list.ContentHash,
			StaticListID:    list.ListID,
			FixturePackHash: "fix-expiry",
			ExpiryDuration:  5 * time.Millisecond,
		})
		if err != nil {
			t.Fatalf("StartTestRun failed: %v", err)
		}

		time.Sleep(10 * time.Millisecond)

		// Dispatching after expiry triggers HandleExpiry and returns ErrTestRunTerminated
		_, err = coord.DispatchTarget(ctx, runExpiry.TestRunID, tgtsExpiry[0].TargetID, "node-1", "email_dispatch")
		if err == nil || !errors.Is(err, testaudience.ErrTestRunTerminated) {
			t.Errorf("Expected ErrTestRunTerminated on expired run, got %v", err)
		}

		diag, err := coord.GetDiagnostics(ctx, runExpiry.TestRunID)
		if err != nil {
			t.Fatalf("GetDiagnostics failed: %v", err)
		}
		if diag.Run.Status != testaudience.TestRunStatusExpired {
			t.Errorf("Expected diagnostics run status 'expired', got %s", diag.Run.Status)
		}
	})

	t.Run("Fake Provider Failure Diagnostics", func(t *testing.T) {
		coord, _, list := setupCoordinator(t, 2)

		run, tgts, err := coord.StartTestRun(ctx, testaudience.TestRunInput{
			TenantID:        "tenant-static",
			CandidateIRHash: "ir-fail",
			ListVersionHash: list.ContentHash,
			StaticListID:    list.ListID,
			FixturePackHash: "fix-fail",
			FakeProvider:    "fake_failing_provider_v1",
		})
		if err != nil {
			t.Fatalf("StartTestRun failed: %v", err)
		}

		ledger, err := coord.DispatchTarget(ctx, run.TestRunID, tgts[0].TargetID, "node-sms", "sms_send")
		if err != nil {
			t.Fatalf("DispatchTarget failed: %v", err)
		}

		if ledger.FakeProvider != "fake_failing_provider_v1" {
			t.Errorf("Expected fake provider 'fake_failing_provider_v1', got %s", ledger.FakeProvider)
		}
		if !ledger.IsTest {
			t.Errorf("Expected action ledger IsTest = true")
		}

		diag, err := coord.GetDiagnostics(ctx, run.TestRunID)
		if err != nil {
			t.Fatalf("GetDiagnostics failed: %v", err)
		}

		if len(diag.Actions) != 1 {
			t.Fatalf("Expected 1 action ledger in diagnostics, got %d", len(diag.Actions))
		}
		if len(diag.Analytics) < 2 {
			t.Fatalf("Expected at least 2 analytical facts (assignment + action), got %d", len(diag.Analytics))
		}
	})
}

// 4. Direct Database & ClickHouse Assertions (Test-Mode Isolation)
func TestDirectDatabaseAndClickHouseAssertions(t *testing.T) {
	ctx := context.Background()

	t.Run("PostgreSQL Table Segregation Assertions", func(t *testing.T) {
		repo := postgres.NewMemoryRepository()
		svc := NewAssignmentService(repo)

		exp := helperIsolationExperiment("exp-db-isolation", "salt-db-iso", []Variant{
			{Key: "control", WeightBasisPoints: 5000, IsControl: true},
			{Key: "treatment", WeightBasisPoints: 5000, IsControl: false},
		})

		tenantID := "tenant-db-iso"
		prodSubject := "subject-production-user-1"
		testSubject := "subject-test-user-1"

		// 1. Production Mode Assignment
		prodAsg, err := svc.GetOrAssign(ctx, AssignmentRequest{
			TenantID:   tenantID,
			Experiment: exp,
			SubjectID:  prodSubject,
			Mode:       ModeProduction,
		})
		if err != nil {
			t.Fatalf("Production GetOrAssign failed: %v", err)
		}
		if prodAsg.IsTestMode {
			t.Errorf("Expected production assignment IsTestMode = false")
		}
		if prodAsg.ExperimentID != "exp-db-isolation" {
			t.Errorf("Expected raw experiment ID 'exp-db-isolation', got %s", prodAsg.ExperimentID)
		}

		// 2. Test Mode Assignment
		testAsg, err := svc.GetOrAssign(ctx, AssignmentRequest{
			TenantID:        tenantID,
			Experiment:      exp,
			SubjectID:       testSubject,
			Mode:            ModeTest,
			ForcedVariantID: "treatment",
		})
		if err != nil {
			t.Fatalf("Test Mode GetOrAssign failed: %v", err)
		}
		if !testAsg.IsTestMode {
			t.Errorf("Expected test assignment IsTestMode = true")
		}

		// 3. Direct DB Query Assertions:
		// Production query for raw experiment ID returns ONLY production subject
		storedProd, err := repo.GetAssignmentByExperimentSubject(ctx, tenantID, "exp-db-isolation", prodSubject)
		if err != nil {
			t.Fatalf("GetAssignmentByExperimentSubject production failed: %v", err)
		}
		if storedProd.SubjectID != prodSubject {
			t.Errorf("Stored production subject mismatch: expected %s, got %s", prodSubject, storedProd.SubjectID)
		}

		// Direct assertion: Production table query for raw experiment ID and test subject MUST return ErrNotFound
		_, errTestInProd := repo.GetAssignmentByExperimentSubject(ctx, tenantID, "exp-db-isolation", testSubject)
		if errTestInProd == nil || !errors.Is(errTestInProd, postgres.ErrNotFound) {
			t.Fatalf("DIRECT DB ASSERTION FAILED: Test subject was found in production table for raw experiment ID! Error: %v", errTestInProd)
		}

		// Test mode assignment MUST be stored under segregated key 'exp-db-isolation:test'
		storedTest, errTestKey := repo.GetAssignmentByExperimentSubject(ctx, tenantID, "exp-db-isolation:test", testSubject)
		if errTestKey != nil {
			t.Fatalf("GetAssignmentByExperimentSubject test key failed: %v", errTestKey)
		}
		if storedTest.SubjectID != testSubject {
			t.Errorf("Stored test subject mismatch: expected %s, got %s", testSubject, storedTest.SubjectID)
		}
	})

	t.Run("ClickHouse Partition and Analytics Report Isolation Assertions", func(t *testing.T) {
		analyticsRepo := analytics.NewMemoryRepository()
		materializer := NewReportMaterializer(analyticsRepo)

		tenantID := "tenant-ch-iso"
		expID := "exp-ch-isolation"
		now := time.Now().UTC()

		// 1. Insert Production Facts (IsTest = false)
		_ = analyticsRepo.InsertAssignments(ctx, []analytics.AssignmentFact{
			{AssignmentID: "asg-prod-1", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, VariantID: "control", SubjectID: "prod-sub-1", IsTest: false, AssignedAt: now},
			{AssignmentID: "asg-prod-2", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, VariantID: "treatment", SubjectID: "prod-sub-2", IsTest: false, AssignedAt: now},
		})
		_ = analyticsRepo.InsertExposures(ctx, []analytics.ExposureFact{
			{ExposureID: "exp-prod-1", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, VariantID: "control", SubjectID: "prod-sub-1", IsTest: false, ExposedAt: now},
			{ExposureID: "exp-prod-2", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, VariantID: "treatment", SubjectID: "prod-sub-2", IsTest: false, ExposedAt: now},
		})
		_ = analyticsRepo.InsertConversions(ctx, []analytics.ConversionFact{
			{ConversionID: "conv-prod-1", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, VariantID: "treatment", SubjectID: "prod-sub-2", MetricName: "conversion", Value: 100.0, IsTest: false, ConvertedAt: now},
		})

		// 2. Insert Test-Mode Facts (IsTest = true)
		_ = analyticsRepo.InsertAssignments(ctx, []analytics.AssignmentFact{
			{AssignmentID: "asg-test-1", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, VariantID: "control", SubjectID: "test-sub-1", IsTest: true, AssignedAt: now},
			{AssignmentID: "asg-test-2", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, VariantID: "treatment", SubjectID: "test-sub-2", IsTest: true, AssignedAt: now},
			{AssignmentID: "asg-test-3", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, VariantID: "treatment", SubjectID: "test-sub-3", IsTest: true, AssignedAt: now},
		})
		_ = analyticsRepo.InsertExposures(ctx, []analytics.ExposureFact{
			{ExposureID: "exp-test-1", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, VariantID: "control", SubjectID: "test-sub-1", IsTest: true, ExposedAt: now},
			{ExposureID: "exp-test-2", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, VariantID: "treatment", SubjectID: "test-sub-2", IsTest: true, ExposedAt: now},
			{ExposureID: "exp-test-3", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, VariantID: "treatment", SubjectID: "test-sub-3", IsTest: true, ExposedAt: now},
		})

		// 3. Query Production Report (Mode = ModeProduction / IsTest = false)
		prodReport, err := materializer.MaterializeReport(ctx, ReportQuery{
			TenantID:          tenantID,
			ExperimentID:      expID,
			ExperimentVersion: 1,
			MetricName:        "conversion",
			DenominatorName:   "exposed",
			Mode:              ModeProduction,
			MinSampleSize:     1,
		})
		if err != nil {
			t.Fatalf("MaterializeReport Production failed: %v", err)
		}

		// DIRECT ASSERTION: Production report MUST contain ONLY production exposures (1 control, 1 treatment)
		if prodReport.Mode != ModeProduction {
			t.Errorf("DIRECT ASSERTION FAILED: Production report Mode mismatch: got %s", prodReport.Mode)
		}
		if prodReport.Variants["control"].Denominators.Exposed != 1 {
			t.Errorf("DIRECT ASSERTION FAILED: Production report control exposed count = %d, expected 1 (test facts leaked!)", prodReport.Variants["control"].Denominators.Exposed)
		}
		if prodReport.Variants["treatment"].Denominators.Exposed != 1 {
			t.Errorf("DIRECT ASSERTION FAILED: Production report treatment exposed count = %d, expected 1 (test facts leaked!)", prodReport.Variants["treatment"].Denominators.Exposed)
		}

		// 4. Query Test Mode Report (Mode = ModeTest / IsTest = true)
		testReport, err := materializer.MaterializeReport(ctx, ReportQuery{
			TenantID:          tenantID,
			ExperimentID:      expID,
			ExperimentVersion: 1,
			MetricName:        "conversion",
			DenominatorName:   "exposed",
			Mode:              ModeTest,
			MinSampleSize:     1,
		})
		if err != nil {
			t.Fatalf("MaterializeReport Test Mode failed: %v", err)
		}

		// DIRECT ASSERTION: Test report MUST contain ONLY test mode exposures (1 control, 2 treatment)
		if testReport.Mode != ModeTest {
			t.Errorf("DIRECT ASSERTION FAILED: Test mode report Mode mismatch: got %s", testReport.Mode)
		}
		if testReport.Variants["control"].Denominators.Exposed != 1 {
			t.Errorf("DIRECT ASSERTION FAILED: Test report control exposed count = %d, expected 1", testReport.Variants["control"].Denominators.Exposed)
		}
		if testReport.Variants["treatment"].Denominators.Exposed != 2 {
			t.Errorf("DIRECT ASSERTION FAILED: Test report treatment exposed count = %d, expected 2", testReport.Variants["treatment"].Denominators.Exposed)
		}
		if testReport.Variants["treatment"].Denominators.Conversion != 0 {
			t.Errorf("DIRECT ASSERTION FAILED: Test report treatment conversion count = %d, expected 0 (production conversion leaked into test report!)", testReport.Variants["treatment"].Denominators.Conversion)
		}
	})
}
