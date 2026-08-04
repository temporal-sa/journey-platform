package experiments

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/store/analytics"
)

// 1. Verify statistical calculations against hand-calculated fixtures
func TestStatisticalCalculations_HandCalculatedFixtures(t *testing.T) {
	// Hand-calculated Fixture:
	// Control: N_c = 1000, M_c = 100 => Rate_c = 0.10
	// Treatment: N_v = 1000, M_v = 150 => Rate_v = 0.15

	t.Run("Control Rate and CI", func(t *testing.T) {
		rateC, ciC := CalculateRateAndCI(100, 1000, 0.95)
		if math.Abs(rateC-0.10) > 1e-6 {
			t.Errorf("expected rate 0.10, got %f", rateC)
		}
		// SE = sqrt(0.1*0.9/1000) = 0.009486833
		// Margin = 1.959964 * 0.009486833 = 0.01859385
		// Lower = 0.081406, Upper = 0.118594
		if math.Abs(ciC.Lower-0.081406) > 1e-4 {
			t.Errorf("expected CI lower ~0.081406, got %f", ciC.Lower)
		}
		if math.Abs(ciC.Upper-0.118594) > 1e-4 {
			t.Errorf("expected CI upper ~0.118594, got %f", ciC.Upper)
		}
	})

	t.Run("Treatment Rate and CI", func(t *testing.T) {
		rateV, ciV := CalculateRateAndCI(150, 1000, 0.95)
		if math.Abs(rateV-0.15) > 1e-6 {
			t.Errorf("expected rate 0.15, got %f", rateV)
		}
		// SE = sqrt(0.15*0.85/1000) = 0.011291590
		// Margin = 1.959964 * 0.011291590 = 0.022131
		// Lower = 0.127869, Upper = 0.172131
		if math.Abs(ciV.Lower-0.127869) > 1e-4 {
			t.Errorf("expected CI lower ~0.127869, got %f", ciV.Lower)
		}
		if math.Abs(ciV.Upper-0.172131) > 1e-4 {
			t.Errorf("expected CI upper ~0.172131, got %f", ciV.Upper)
		}
	})

	t.Run("Absolute Lift and CI", func(t *testing.T) {
		rateC := 0.10
		rateV := 0.15
		absLift, ciLift := CalculateAbsoluteLiftAndCI(rateV, rateC, 1000, 1000, 0.95)

		// Abs Lift = 0.15 - 0.10 = 0.05
		if math.Abs(absLift-0.05) > 1e-6 {
			t.Errorf("expected absolute lift 0.05, got %f", absLift)
		}
		// SE(diff) = sqrt(0.0001275 + 0.00009) = 0.01474788
		// Margin = 1.959964 * 0.01474788 = 0.028905
		// Lower = 0.021095, Upper = 0.078905
		if math.Abs(ciLift.Lower-0.021095) > 1e-4 {
			t.Errorf("expected lift CI lower ~0.021095, got %f", ciLift.Lower)
		}
		if math.Abs(ciLift.Upper-0.078905) > 1e-4 {
			t.Errorf("expected lift CI upper ~0.078905, got %f", ciLift.Upper)
		}
	})

	t.Run("Relative Lift and CI", func(t *testing.T) {
		rateC := 0.10
		rateV := 0.15
		relLift, ciRel := CalculateRelativeLiftAndCI(rateV, rateC, 1000, 1000, 0.95)

		// Rel Lift = (0.15 - 0.10)/0.10 = 0.50 (50%)
		if math.Abs(relLift-0.50) > 1e-6 {
			t.Errorf("expected relative lift 0.50, got %f", relLift)
		}
		// SE(rel) = 0.01474788 / 0.10 = 0.1474788
		// Margin = 1.959964 * 0.1474788 = 0.289053
		// Lower = 0.210947, Upper = 0.789053
		if math.Abs(ciRel.Lower-0.210947) > 1e-4 {
			t.Errorf("expected rel lift CI lower ~0.210947, got %f", ciRel.Lower)
		}
		if math.Abs(ciRel.Upper-0.789053) > 1e-4 {
			t.Errorf("expected rel lift CI upper ~0.789053, got %f", ciRel.Upper)
		}
	})

	t.Run("Two-Sample Z-Test P-Value", func(t *testing.T) {
		pValue, isSig := CalculateTwoSampleZTest(150, 1000, 100, 1000)
		if !isSig {
			t.Errorf("expected statistically significant result (p < 0.05)")
		}
		// Pooled rate = 250/2000 = 0.125
		// SE = sqrt(0.125*0.875*(0.002)) = 0.0147902
		// Z = 0.05 / 0.0147902 = 3.3806
		// p-value = erfc(3.3806/sqrt(2)) = 0.000723
		if math.Abs(pValue-0.000723) > 1e-4 {
			t.Errorf("expected p-value ~0.000723, got %f", pValue)
		}
	})

	t.Run("SRM Normal Distribution", func(t *testing.T) {
		sampleCounts := map[string]int64{
			"control":   1000,
			"treatment": 1000,
		}
		srm := CalculateSRM(sampleCounts, nil, 30)
		if srm.Status != "normal" {
			t.Errorf("expected SRM status 'normal', got %s", srm.Status)
		}
		if srm.ChiSquare != 0.0 {
			t.Errorf("expected chi-square 0.0, got %f", srm.ChiSquare)
		}
		if math.Abs(srm.PValue-1.0) > 1e-6 {
			t.Errorf("expected p-value 1.0, got %f", srm.PValue)
		}
	})

	t.Run("SRM Mismatch Distribution", func(t *testing.T) {
		sampleCounts := map[string]int64{
			"control":   600,
			"treatment": 400,
		}
		srm := CalculateSRM(sampleCounts, nil, 30)
		if srm.Status != "mismatch_detected" {
			t.Errorf("expected SRM status 'mismatch_detected', got %s", srm.Status)
		}
		// ChiSquare = (600-500)^2/500 + (400-500)^2/500 = 20 + 20 = 40.0
		if math.Abs(srm.ChiSquare-40.0) > 1e-6 {
			t.Errorf("expected chi-square 40.0, got %f", srm.ChiSquare)
		}
		if srm.PValue >= 0.001 {
			t.Errorf("expected p-value < 0.001, got %f", srm.PValue)
		}
	})
}

// 2. Test CSV formula injection escaping (=, @, +, -)
func TestCSVFormulaInjectionEscaping(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{input: "=SUM(A1:B10)", expected: "'=SUM(A1:B10)"},
		{input: "@ADMIN_EXECUTE()", expected: "'@ADMIN_EXECUTE()"},
		{input: "+100_BOOST", expected: "'+100_BOOST"},
		{input: "-CMD_RUN", expected: "'-CMD_RUN"},
		{input: "\tTAB_INJECT", expected: "'\tTAB_INJECT"},
		{input: "\rCR_INJECT", expected: "'\rCR_INJECT"},
		{input: "normal_variant_key", expected: "normal_variant_key"},
		{input: "tenant-100", expected: "tenant-100"},
	}

	for _, tc := range testCases {
		escaped := EscapeCSVCell(tc.input)
		if escaped != tc.expected {
			t.Errorf("EscapeCSVCell(%q): expected %q, got %q", tc.input, tc.expected, escaped)
		}
	}

	// Verify report CSV rendering escapes malicious strings
	rep := &ExperimentReport{
		TenantID:          "=DANGER_TENANT",
		ExperimentID:      "@EXPLOIT_EXP",
		ExperimentVersion: 1,
		JourneyID:         "+JOURNEY_BOOST",
		JourneyVersion:    1,
		MetricName:        "-METRIC_REDUCE",
		Mode:              ModeProduction,
		Variants: map[string]VariantStats{
			"=VAR_CONTROL": {
				VariantID:    "=VAR_CONTROL",
				IsControl:    true,
				Denominators: Denominators{Exposed: 100, Conversion: 10},
			},
		},
		SRM: SRMResult{Status: "normal"},
	}

	csvStr, err := rep.ToCSV()
	if err != nil {
		t.Fatalf("ToCSV failed: %v", err)
	}

	if !strings.Contains(csvStr, "'=DANGER_TENANT") {
		t.Errorf("expected escaped '=DANGER_TENANT' in CSV output, got:\n%s", csvStr)
	}
	if !strings.Contains(csvStr, "'@EXPLOIT_EXP") {
		t.Errorf("expected escaped '@EXPLOIT_EXP' in CSV output, got:\n%s", csvStr)
	}
	if !strings.Contains(csvStr, "'+JOURNEY_BOOST") {
		t.Errorf("expected escaped '+JOURNEY_BOOST' in CSV output, got:\n%s", csvStr)
	}
	if !strings.Contains(csvStr, "'-METRIC_REDUCE") {
		t.Errorf("expected escaped '-METRIC_REDUCE' in CSV output, got:\n%s", csvStr)
	}
	if !strings.Contains(csvStr, "'=VAR_CONTROL") {
		t.Errorf("expected escaped '=VAR_CONTROL' in CSV output, got:\n%s", csvStr)
	}
}

// 3. Test Mode & Experiment Version & Journey Version isolation checks
func TestModeAndVersionIsolation(t *testing.T) {
	ctx := context.Background()
	analyticsRepo := analytics.NewMemoryRepository()
	materializer := NewReportMaterializer(analyticsRepo)

	tenantID := "tenant-iso"
	expID := "exp-iso-100"
	now := time.Now().UTC()

	// Insert Facts for Version 1, Production Mode
	_ = analyticsRepo.InsertExposures(ctx, []analytics.ExposureFact{
		{ExposureID: "e1-v1-prod", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, JourneyID: "j1", JourneyVersion: 1, VariantID: "control", SubjectID: "s1", ExposedAt: now, IsTest: false},
		{ExposureID: "e2-v1-prod", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, JourneyID: "j1", JourneyVersion: 1, VariantID: "control", SubjectID: "s2", ExposedAt: now, IsTest: false},
	})

	// Insert Facts for Version 1, Test Mode
	_ = analyticsRepo.InsertExposures(ctx, []analytics.ExposureFact{
		{ExposureID: "e1-v1-test", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, JourneyID: "j1", JourneyVersion: 1, VariantID: "control", SubjectID: "s3", ExposedAt: now, IsTest: true},
		{ExposureID: "e2-v1-test", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, JourneyID: "j1", JourneyVersion: 1, VariantID: "control", SubjectID: "s4", ExposedAt: now, IsTest: true},
		{ExposureID: "e3-v1-test", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 1, JourneyID: "j1", JourneyVersion: 1, VariantID: "control", SubjectID: "s5", ExposedAt: now, IsTest: true},
	})

	// Insert Facts for Version 2, Production Mode
	_ = analyticsRepo.InsertExposures(ctx, []analytics.ExposureFact{
		{ExposureID: "e1-v2-prod", TenantID: tenantID, ExperimentID: expID, ExperimentVersion: 2, JourneyID: "j1", JourneyVersion: 1, VariantID: "control", SubjectID: "s6", ExposedAt: now, IsTest: false},
	})

	// Query Version 1 Production Report
	repV1Prod, err := materializer.MaterializeReport(ctx, ReportQuery{
		TenantID:          tenantID,
		ExperimentID:      expID,
		ExperimentVersion: 1,
		JourneyID:         "j1",
		JourneyVersion:    1,
		Mode:              ModeProduction,
		MinSampleSize:     1,
	})
	if err != nil {
		t.Fatalf("MaterializeReport v1 prod failed: %v", err)
	}
	if repV1Prod.Variants["control"].Denominators.Exposed != 2 {
		t.Errorf("expected 2 exposed facts for v1 production, got %d", repV1Prod.Variants["control"].Denominators.Exposed)
	}

	// Query Version 1 Test Mode Report
	repV1Test, err := materializer.MaterializeReport(ctx, ReportQuery{
		TenantID:          tenantID,
		ExperimentID:      expID,
		ExperimentVersion: 1,
		JourneyID:         "j1",
		JourneyVersion:    1,
		Mode:              ModeTest,
		MinSampleSize:     1,
	})
	if err != nil {
		t.Fatalf("MaterializeReport v1 test failed: %v", err)
	}
	if repV1Test.Variants["control"].Denominators.Exposed != 3 {
		t.Errorf("expected 3 exposed facts for v1 test mode, got %d", repV1Test.Variants["control"].Denominators.Exposed)
	}

	// Query Version 2 Production Report
	repV2Prod, err := materializer.MaterializeReport(ctx, ReportQuery{
		TenantID:          tenantID,
		ExperimentID:      expID,
		ExperimentVersion: 2,
		JourneyID:         "j1",
		JourneyVersion:    1,
		Mode:              ModeProduction,
		MinSampleSize:     1,
	})
	if err != nil {
		t.Fatalf("MaterializeReport v2 prod failed: %v", err)
	}
	if repV2Prod.Variants["control"].Denominators.Exposed != 1 {
		t.Errorf("expected 1 exposed fact for v2 production, got %d", repV2Prod.Variants["control"].Denominators.Exposed)
	}
}

// 4. Test all 8 Denominators support
func TestDenominatorsSupport(t *testing.T) {
	d := Denominators{
		Assigned:    10,
		Exposed:     9,
		Attempted:   8,
		Accepted:    7,
		Delivered:   6,
		UniqueOpen:  5,
		UniqueClick: 4,
		Conversion:  3,
	}

	if d.Get("assigned") != 10 {
		t.Errorf("expected assigned 10, got %d", d.Get("assigned"))
	}
	if d.Get("exposed") != 9 {
		t.Errorf("expected exposed 9, got %d", d.Get("exposed"))
	}
	if d.Get("attempted") != 8 {
		t.Errorf("expected attempted 8, got %d", d.Get("attempted"))
	}
	if d.Get("accepted") != 7 {
		t.Errorf("expected accepted 7, got %d", d.Get("accepted"))
	}
	if d.Get("delivered") != 6 {
		t.Errorf("expected delivered 6, got %d", d.Get("delivered"))
	}
	if d.Get("unique_open") != 5 {
		t.Errorf("expected unique_open 5, got %d", d.Get("unique_open"))
	}
	if d.Get("unique_click") != 4 {
		t.Errorf("expected unique_click 4, got %d", d.Get("unique_click"))
	}
	if d.Get("conversion") != 3 {
		t.Errorf("expected conversion 3, got %d", d.Get("conversion"))
	}
}

// 5. Test Metadata, Watermark, ProcessingLag, Exclusions, Insufficient Sample Warning, Provisional vs Final state
func TestReportMetadataAndWarnings(t *testing.T) {
	ctx := context.Background()
	analyticsRepo := analytics.NewMemoryRepository()
	materializer := NewReportMaterializer(analyticsRepo)

	tenantID := "tenant-meta"
	expID := "exp-meta-1"
	eventTime := time.Now().UTC().Add(-30 * time.Minute)

	// Insert facts
	_ = analyticsRepo.InsertExposures(ctx, []analytics.ExposureFact{
		{ExposureID: "exp-meta-1", TenantID: tenantID, ExperimentID: expID, VariantID: "control", SubjectID: "sub-1", ExposedAt: eventTime, IsTest: false},
	})
	_ = analyticsRepo.InsertDataQuality(ctx, []analytics.DataQualityFact{
		{QualityEventID: "q1", TenantID: tenantID, IssueType: "filter_mismatch", OccurredAt: eventTime, IsTest: false},
		{QualityEventID: "q2", TenantID: tenantID, IssueType: "quality_issue", OccurredAt: eventTime, IsTest: false},
	})

	// Query with EndTime > Watermark => Provisional Window State
	queryProvisional := ReportQuery{
		TenantID:      tenantID,
		ExperimentID:  expID,
		Mode:          ModeProduction,
		StartTime:     eventTime.Add(-time.Hour),
		EndTime:       eventTime.Add(15 * time.Minute),
		MinSampleSize: 30, // Sample N=1 < 30 => InsufficientSampleWarning
	}

	reportProv, err := materializer.MaterializeReport(ctx, queryProvisional)
	if err != nil {
		t.Fatalf("MaterializeReport provisional failed: %v", err)
	}

	if reportProv.Metadata.WindowState != "provisional" {
		t.Errorf("expected window state 'provisional', got %s", reportProv.Metadata.WindowState)
	}
	if !reportProv.Metadata.InsufficientSampleWarning {
		t.Errorf("expected InsufficientSampleWarning to be true")
	}
	if reportProv.Metadata.Exclusions.TotalExcluded != 2 {
		t.Errorf("expected 2 total exclusions, got %d", reportProv.Metadata.Exclusions.TotalExcluded)
	}
	if reportProv.ContentHash == "" {
		t.Errorf("expected non-empty content hash")
	}

	// Query with EndTime <= Watermark => Final Window State
	queryFinal := ReportQuery{
		TenantID:      tenantID,
		ExperimentID:  expID,
		Mode:          ModeProduction,
		StartTime:     eventTime.Add(-time.Hour),
		EndTime:       eventTime,
		MinSampleSize: 30,
	}

	reportFinal, err := materializer.MaterializeReport(ctx, queryFinal)
	if err != nil {
		t.Fatalf("MaterializeReport final failed: %v", err)
	}

	if reportFinal.Metadata.WindowState != "final" {
		t.Errorf("expected window state 'final', got %s", reportFinal.Metadata.WindowState)
	}
}
