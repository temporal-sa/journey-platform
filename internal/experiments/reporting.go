package experiments

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/validated-pattern/journey-platform/internal/store/analytics"
)

// Denominators captures total counts for all 8 conversion and funnel denominator metrics.
type Denominators struct {
	Assigned    int64 `json:"assigned"`
	Exposed     int64 `json:"exposed"`
	Attempted   int64 `json:"attempted"`
	Accepted    int64 `json:"accepted"`
	Delivered   int64 `json:"delivered"`
	UniqueOpen  int64 `json:"unique_open"`
	UniqueClick int64 `json:"unique_click"`
	Conversion  int64 `json:"conversion"`
}

// Get returns the count for a specific denominator by name.
func (d Denominators) Get(name string) int64 {
	switch strings.ToLower(name) {
	case "assigned":
		return d.Assigned
	case "exposed":
		return d.Exposed
	case "attempted":
		return d.Attempted
	case "accepted":
		return d.Accepted
	case "delivered":
		return d.Delivered
	case "unique_open", "open", "opens":
		return d.UniqueOpen
	case "unique_click", "click", "clicks":
		return d.UniqueClick
	case "conversion", "conversions":
		return d.Conversion
	default:
		if d.Exposed > 0 {
			return d.Exposed
		}
		return d.Assigned
	}
}

// ConfidenceInterval holds lower and upper bounds for a 95% z-score confidence interval.
type ConfidenceInterval struct {
	Lower float64 `json:"lower"`
	Upper float64 `json:"upper"`
	CL    float64 `json:"confidence_level"`
}

// VariantStats contains statistical calculation results for an experiment variant.
type VariantStats struct {
	VariantID          string             `json:"variant_id"`
	IsControl          bool               `json:"is_control"`
	Denominators       Denominators       `json:"denominators"`
	SampleSize         int64              `json:"sample_size"`
	AbsoluteRate       float64            `json:"absolute_rate"`
	RateCI             ConfidenceInterval `json:"rate_ci"`
	AbsoluteLift       float64            `json:"absolute_lift"`
	AbsoluteLiftCI     ConfidenceInterval `json:"absolute_lift_ci"`
	RelativeLift       float64            `json:"relative_lift"`
	RelativeLiftCI     ConfidenceInterval `json:"relative_lift_ci"`
	PValue             float64            `json:"p_value"`
	IsStatisticallySig bool               `json:"is_statistically_significant"`
	InsufficientSample bool               `json:"insufficient_sample"`
	WarningMessage     string             `json:"warning_message,omitempty"`
}

// SRMResult contains Sample Ratio Mismatch chi-square test results across variants.
type SRMResult struct {
	Status        string             `json:"status"` // "normal", "warning", "mismatch_detected", "suppressed"
	ChiSquare     float64            `json:"chi_square"`
	PValue        float64            `json:"p_value"`
	ExpectedRatio map[string]float64 `json:"expected_ratio"`
	ActualRatio   map[string]float64 `json:"actual_ratio"`
	SampleCounts  map[string]int64   `json:"sample_counts"`
	IsSuppressed  bool               `json:"is_suppressed"`
}

// Exclusions captures counts of excluded event records.
type Exclusions struct {
	FilterMismatch int64 `json:"filter_mismatch"`
	QualityIssues  int64 `json:"quality_issues"`
	LateWatermark  int64 `json:"late_watermark"`
	TotalExcluded  int64 `json:"total_excluded"`
}

// ReportMetadata contains event watermark, processing lag, exclusions, and window state.
type ReportMetadata struct {
	Watermark                 time.Time     `json:"watermark"`
	ProcessingLag             time.Duration `json:"processing_lag"`
	Exclusions                Exclusions    `json:"exclusions"`
	InsufficientSampleWarning bool          `json:"insufficient_sample_warning"`
	WindowState               string        `json:"window_state"` // "provisional" or "final"
	GeneratedAt               time.Time     `json:"generated_at"`
}

// ExperimentReport represents an immutable materialized aggregate statistical report.
type ExperimentReport struct {
	ReportID          string                  `json:"report_id"`
	TenantID          string                  `json:"tenant_id"`
	ExperimentID      string                  `json:"experiment_id"`
	ExperimentVersion uint32                  `json:"experiment_version"`
	JourneyID         string                  `json:"journey_id"`
	JourneyVersion    uint32                  `json:"journey_version"`
	MetricName        string                  `json:"metric_name"`
	DenominatorName   string                  `json:"denominator_name"`
	Mode              AssignmentMode          `json:"mode"`
	StartTime         time.Time               `json:"start_time"`
	EndTime           time.Time               `json:"end_time"`
	Metadata          ReportMetadata          `json:"metadata"`
	SRM               SRMResult               `json:"srm"`
	Variants          map[string]VariantStats `json:"variants"`
	ContentHash       string                  `json:"content_hash"`
}

// ReportQuery defines parameters to query and materialize an experiment report.
type ReportQuery struct {
	TenantID          string             `json:"tenant_id"`
	ExperimentID      string             `json:"experiment_id"`
	ExperimentVersion uint32             `json:"experiment_version"`
	JourneyID         string             `json:"journey_id"`
	JourneyVersion    uint32             `json:"journey_version"`
	MetricName        string             `json:"metric_name"`
	DenominatorName   string             `json:"denominator_name"`
	Mode              AssignmentMode     `json:"mode"`
	StartTime         time.Time          `json:"start_time"`
	EndTime           time.Time          `json:"end_time"`
	MinSampleSize     int64              `json:"min_sample_size,omitempty"`
	ExpectedWeights   map[string]float64 `json:"expected_weights,omitempty"`
}

// EscapeCSVCell escapes formula injection characters (=, @, +, -, \t, \r) by prepending a single quote (').
func EscapeCSVCell(val string) string {
	if len(val) == 0 {
		return val
	}
	r := val[0]
	if r == '=' || r == '@' || r == '+' || r == '-' || r == '\t' || r == '\r' {
		return "'" + val
	}
	return val
}

// CalculateContentHash computes a deterministic SHA-256 hash over immutable report fields.
func (r *ExperimentReport) CalculateContentHash() (string, error) {
	type HashableReport struct {
		TenantID          string                  `json:"tenant_id"`
		ExperimentID      string                  `json:"experiment_id"`
		ExperimentVersion uint32                  `json:"experiment_version"`
		JourneyID         string                  `json:"journey_id"`
		JourneyVersion    uint32                  `json:"journey_version"`
		MetricName        string                  `json:"metric_name"`
		DenominatorName   string                  `json:"denominator_name"`
		Mode              AssignmentMode          `json:"mode"`
		StartTime         time.Time               `json:"start_time"`
		EndTime           time.Time               `json:"end_time"`
		SRM               SRMResult               `json:"srm"`
		Variants          map[string]VariantStats `json:"variants"`
	}
	h := HashableReport{
		TenantID:          r.TenantID,
		ExperimentID:      r.ExperimentID,
		ExperimentVersion: r.ExperimentVersion,
		JourneyID:         r.JourneyID,
		JourneyVersion:    r.JourneyVersion,
		MetricName:        r.MetricName,
		DenominatorName:   r.DenominatorName,
		Mode:              r.Mode,
		StartTime:         r.StartTime,
		EndTime:           r.EndTime,
		SRM:               r.SRM,
		Variants:          r.Variants,
	}
	data, err := json.Marshal(h)
	if err != nil {
		return "", fmt.Errorf("failed to marshal report for content hash: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// ToCSV exports the report in CSV format with formula-safe cell escaping.
func (r *ExperimentReport) ToCSV() (string, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	header := []string{
		"tenant_id",
		"experiment_id",
		"experiment_version",
		"journey_id",
		"journey_version",
		"mode",
		"metric_name",
		"denominator_name",
		"variant_id",
		"is_control",
		"assigned",
		"exposed",
		"attempted",
		"accepted",
		"delivered",
		"unique_open",
		"unique_click",
		"conversion",
		"sample_size",
		"absolute_rate",
		"rate_ci_lower",
		"rate_ci_upper",
		"absolute_lift",
		"lift_ci_lower",
		"lift_ci_upper",
		"relative_lift",
		"rel_lift_ci_lower",
		"rel_lift_ci_upper",
		"p_value",
		"is_stat_sig",
		"srm_status",
		"srm_p_value",
		"watermark",
		"window_state",
	}
	if err := w.Write(header); err != nil {
		return "", err
	}

	keys := make([]string, 0, len(r.Variants))
	for k := range r.Variants {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := r.Variants[k]
		row := []string{
			EscapeCSVCell(r.TenantID),
			EscapeCSVCell(r.ExperimentID),
			fmt.Sprintf("%d", r.ExperimentVersion),
			EscapeCSVCell(r.JourneyID),
			fmt.Sprintf("%d", r.JourneyVersion),
			EscapeCSVCell(string(r.Mode)),
			EscapeCSVCell(r.MetricName),
			EscapeCSVCell(r.DenominatorName),
			EscapeCSVCell(v.VariantID),
			fmt.Sprintf("%t", v.IsControl),
			fmt.Sprintf("%d", v.Denominators.Assigned),
			fmt.Sprintf("%d", v.Denominators.Exposed),
			fmt.Sprintf("%d", v.Denominators.Attempted),
			fmt.Sprintf("%d", v.Denominators.Accepted),
			fmt.Sprintf("%d", v.Denominators.Delivered),
			fmt.Sprintf("%d", v.Denominators.UniqueOpen),
			fmt.Sprintf("%d", v.Denominators.UniqueClick),
			fmt.Sprintf("%d", v.Denominators.Conversion),
			fmt.Sprintf("%d", v.SampleSize),
			fmt.Sprintf("%.6f", v.AbsoluteRate),
			fmt.Sprintf("%.6f", v.RateCI.Lower),
			fmt.Sprintf("%.6f", v.RateCI.Upper),
			fmt.Sprintf("%.6f", v.AbsoluteLift),
			fmt.Sprintf("%.6f", v.AbsoluteLiftCI.Lower),
			fmt.Sprintf("%.6f", v.AbsoluteLiftCI.Upper),
			fmt.Sprintf("%.6f", v.RelativeLift),
			fmt.Sprintf("%.6f", v.RelativeLiftCI.Lower),
			fmt.Sprintf("%.6f", v.RelativeLiftCI.Upper),
			fmt.Sprintf("%.6f", v.PValue),
			fmt.Sprintf("%t", v.IsStatisticallySig),
			EscapeCSVCell(r.SRM.Status),
			fmt.Sprintf("%.6f", r.SRM.PValue),
			r.Metadata.Watermark.Format(time.RFC3339),
			EscapeCSVCell(r.Metadata.WindowState),
		}
		if err := w.Write(row); err != nil {
			return "", err
		}
	}

	w.Flush()
	return buf.String(), w.Error()
}

// Statistical calculation engine constants and methods.

const StandardZ95 = 1.959963984540054

// CalculateRateAndCI computes absolute conversion rate and 95% confidence interval bounds.
func CalculateRateAndCI(conversions, sampleSize int64, confidenceLevel float64) (float64, ConfidenceInterval) {
	if confidenceLevel <= 0 {
		confidenceLevel = 0.95
	}
	ci := ConfidenceInterval{CL: confidenceLevel}
	if sampleSize <= 0 {
		return 0.0, ci
	}
	rate := float64(conversions) / float64(sampleSize)
	se := math.Sqrt(rate * (1.0 - rate) / float64(sampleSize))
	margin := StandardZ95 * se

	ci.Lower = math.Max(0.0, rate-margin)
	ci.Upper = math.Min(1.0, rate+margin)
	return rate, ci
}

// CalculateAbsoluteLiftAndCI computes absolute lift (rateV - rateC) and 95% confidence interval bounds.
func CalculateAbsoluteLiftAndCI(rateV, rateC float64, sampleV, sampleC int64, confidenceLevel float64) (float64, ConfidenceInterval) {
	if confidenceLevel <= 0 {
		confidenceLevel = 0.95
	}
	ci := ConfidenceInterval{CL: confidenceLevel}
	lift := rateV - rateC
	if sampleV <= 0 || sampleC <= 0 {
		return lift, ci
	}

	varV := rateV * (1.0 - rateV) / float64(sampleV)
	varC := rateC * (1.0 - rateC) / float64(sampleC)
	se := math.Sqrt(varV + varC)
	margin := StandardZ95 * se

	ci.Lower = lift - margin
	ci.Upper = lift + margin
	return lift, ci
}

// CalculateRelativeLiftAndCI computes relative lift (rateV - rateC)/rateC and 95% confidence interval bounds.
func CalculateRelativeLiftAndCI(rateV, rateC float64, sampleV, sampleC int64, confidenceLevel float64) (float64, ConfidenceInterval) {
	if confidenceLevel <= 0 {
		confidenceLevel = 0.95
	}
	ci := ConfidenceInterval{CL: confidenceLevel}
	if rateC == 0.0 || sampleV <= 0 || sampleC <= 0 {
		return 0.0, ci
	}

	relLift := (rateV - rateC) / rateC

	varV := rateV * (1.0 - rateV) / float64(sampleV)
	varC := rateC * (1.0 - rateC) / float64(sampleC)
	seLift := math.Sqrt(varV + varC)
	seRel := seLift / rateC

	margin := StandardZ95 * seRel
	ci.Lower = relLift - margin
	ci.Upper = relLift + margin
	return relLift, ci
}

// CalculateTwoSampleZTest computes two-tailed z-test p-value and statistical significance (alpha = 0.05).
func CalculateTwoSampleZTest(conversionsV, sampleV, conversionsC, sampleC int64) (float64, bool) {
	if sampleV <= 0 || sampleC <= 0 {
		return 1.0, false
	}
	rateV := float64(conversionsV) / float64(sampleV)
	rateC := float64(conversionsC) / float64(sampleC)
	diff := rateV - rateC

	pooledRate := float64(conversionsV+conversionsC) / float64(sampleV+sampleC)
	se := math.Sqrt(pooledRate * (1.0 - pooledRate) * (1.0/float64(sampleV) + 1.0/float64(sampleC)))
	if se == 0.0 {
		return 1.0, false
	}

	z := math.Abs(diff) / se
	pValue := math.Erfc(z / math.Sqrt(2.0))
	return pValue, pValue < 0.05
}

// CalculateSRM computes Sample Ratio Mismatch chi-square test across variants.
func CalculateSRM(sampleCounts map[string]int64, expectedWeights map[string]float64, minSampleSize int64) SRMResult {
	if minSampleSize <= 0 {
		minSampleSize = 30
	}
	result := SRMResult{
		Status:        "normal",
		ExpectedRatio: make(map[string]float64),
		ActualRatio:   make(map[string]float64),
		SampleCounts:  sampleCounts,
	}

	if len(sampleCounts) == 0 {
		return result
	}

	var totalSamples int64
	for _, count := range sampleCounts {
		totalSamples += count
	}

	if totalSamples < minSampleSize {
		result.IsSuppressed = true
		result.Status = "suppressed"
		return result
	}

	numVariants := float64(len(sampleCounts))
	var chiSquare float64

	for vID, count := range sampleCounts {
		actualRatio := float64(count) / float64(totalSamples)
		result.ActualRatio[vID] = actualRatio

		expWeight := 1.0 / numVariants
		if w, ok := expectedWeights[vID]; ok && w > 0 {
			expWeight = w
		}
		result.ExpectedRatio[vID] = expWeight

		expCount := float64(totalSamples) * expWeight
		if expCount > 0 {
			diff := float64(count) - expCount
			chiSquare += (diff * diff) / expCount
		}
	}

	df := len(sampleCounts) - 1
	if df < 1 {
		df = 1
	}

	pValue := ChiSquarePValue(chiSquare, df)
	result.ChiSquare = chiSquare
	result.PValue = pValue

	if pValue < 0.001 {
		result.Status = "mismatch_detected"
	} else if pValue < 0.05 {
		result.Status = "warning"
	} else {
		result.Status = "normal"
	}

	return result
}

// ChiSquarePValue calculates the upper tail p-value for a chi-square statistic with df degrees of freedom.
func ChiSquarePValue(chi2 float64, df int) float64 {
	if chi2 <= 0 || df <= 0 {
		return 1.0
	}
	if df == 1 {
		return math.Erfc(math.Sqrt(chi2 / 2.0))
	}
	a := float64(df) / 2.0
	x := chi2 / 2.0
	return gammaQ(a, x)
}

// gammaQ computes upper regularized incomplete gamma function Q(a, x).
func gammaQ(a, x float64) float64 {
	if x <= 0 || a <= 0 {
		return 1.0
	}
	if x < a+1.0 {
		return 1.0 - gammaP(a, x)
	}

	const maxIter = 200
	const eps = 1e-15

	gln, _ := math.Lgamma(a)
	b := x + 1.0 - a
	c := 1.0 / 1e-30
	d := 1.0 / b
	h := d

	for i := 1; i <= maxIter; i++ {
		an := -float64(i) * (float64(i) - a)
		b += 2.0
		d = an*d + b
		if math.Abs(d) < 1e-30 {
			d = 1e-30
		}
		c = b + an/c
		if math.Abs(c) < 1e-30 {
			c = 1e-30
		}
		d = 1.0 / d
		del := d * c
		h *= del
		if math.Abs(del-1.0) < eps {
			break
		}
	}

	return math.Exp(a*math.Log(x)-x-gln) * h
}

// gammaP computes lower regularized incomplete gamma function P(a, x).
func gammaP(a, x float64) float64 {
	const maxIter = 200
	const eps = 1e-15

	gln, _ := math.Lgamma(a)
	ap := a
	sum := 1.0 / a
	del := sum

	for n := 1; n <= maxIter; n++ {
		ap++
		del *= x / ap
		sum += del
		if math.Abs(del) < math.Abs(sum)*eps {
			break
		}
	}

	return sum * math.Exp(a*math.Log(x)-x-gln)
}

// ReportMaterializer service handles experiment report materialization.
type ReportMaterializer struct {
	analyticsRepo analytics.Repository
}

// NewReportMaterializer creates a new ReportMaterializer instance.
func NewReportMaterializer(analyticsRepo analytics.Repository) *ReportMaterializer {
	return &ReportMaterializer{
		analyticsRepo: analyticsRepo,
	}
}

// MaterializeReport materializes versioned aggregate reports from ClickHouse append-only facts.
func (m *ReportMaterializer) MaterializeReport(ctx context.Context, query ReportQuery) (*ExperimentReport, error) {
	if query.MinSampleSize <= 0 {
		query.MinSampleSize = 30
	}
	if query.MetricName == "" {
		query.MetricName = "conversion"
	}
	if query.DenominatorName == "" {
		query.DenominatorName = "exposed"
	}
	if query.Mode == "" {
		query.Mode = ModeProduction
	}

	isTest := (query.Mode == ModeTest)

	mq := analytics.MetricQuery{
		TenantID:          query.TenantID,
		ExperimentID:      query.ExperimentID,
		ExperimentVersion: query.ExperimentVersion,
		JourneyID:         query.JourneyID,
		JourneyVersion:    query.JourneyVersion,
		MetricName:        query.MetricName,
		IsTest:            isTest,
		StartTime:         query.StartTime,
		EndTime:           query.EndTime,
		MinCellThreshold: 1,
		ExpectedWeights:  query.ExpectedWeights,
	}

	now := time.Now().UTC()
	var rawReport *analytics.AnalyticsReport
	var err error

	if m.analyticsRepo != nil {
		rawReport, err = m.analyticsRepo.QueryReport(ctx, mq)
		if err != nil {
			return nil, fmt.Errorf("failed to query analytics repository: %w", err)
		}
	}

	if rawReport == nil {
		rawReport = &analytics.AnalyticsReport{
			TenantID:          query.TenantID,
			ExperimentID:      query.ExperimentID,
			ExperimentVersion: query.ExperimentVersion,
			JourneyID:         query.JourneyID,
			JourneyVersion:    query.JourneyVersion,
			MetricName:        query.MetricName,
			IsTest:            isTest,
			StartTime:         query.StartTime,
			EndTime:           query.EndTime,
			GeneratedAt:       now,
		}
	}

	metadata := ReportMetadata{
		Watermark:     rawReport.Watermark,
		ProcessingLag: rawReport.ProcessingLag,
		Exclusions: Exclusions{
			FilterMismatch: rawReport.Exclusions.FilterMismatch,
			QualityIssues:  rawReport.Exclusions.QualityIssues,
			LateWatermark:  rawReport.Exclusions.LateWatermark,
			TotalExcluded:  rawReport.Exclusions.TotalExcluded,
		},
		GeneratedAt: now,
	}

	if rawReport.Watermark.IsZero() || (!query.EndTime.IsZero() && rawReport.Watermark.Before(query.EndTime)) {
		metadata.WindowState = "provisional"
	} else {
		metadata.WindowState = "final"
	}

	variantDenoms := make(map[string]Denominators)
	sampleCounts := make(map[string]int64)

	for _, vr := range rawReport.Variants {
		d := Denominators{
			Assigned:    vr.Denominators.Assignments,
			Exposed:     vr.Denominators.Exposures,
			Attempted:   vr.Denominators.Attempts,
			Accepted:    vr.Denominators.Sends,
			Delivered:   vr.Denominators.Deliveries,
			UniqueOpen:  vr.Denominators.Opens,
			UniqueClick: vr.Denominators.Clicks,
			Conversion:  vr.Denominators.Conversions,
		}
		variantDenoms[vr.VariantID] = d
		sampleCounts[vr.VariantID] = d.Get(query.DenominatorName)
	}

	controlID := ""
	for vID := range variantDenoms {
		if strings.Contains(strings.ToLower(vID), "control") {
			controlID = vID
			break
		}
	}
	if controlID == "" {
		keys := make([]string, 0, len(variantDenoms))
		for k := range variantDenoms {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if len(keys) > 0 {
			controlID = keys[0]
		}
	}

	controlDenom := variantDenoms[controlID]
	controlSample := controlDenom.Get(query.DenominatorName)
	controlConv := controlDenom.Conversion
	controlRate, _ := CalculateRateAndCI(controlConv, controlSample, 0.95)

	variantStatsMap := make(map[string]VariantStats)
	insufficientSampleFlag := false

	for vID, d := range variantDenoms {
		isControl := (vID == controlID)
		sampleSize := d.Get(query.DenominatorName)
		convCount := d.Conversion

		rate, rateCI := CalculateRateAndCI(convCount, sampleSize, 0.95)

		vs := VariantStats{
			VariantID:    vID,
			IsControl:    isControl,
			Denominators: d,
			SampleSize:   sampleSize,
			AbsoluteRate: rate,
			RateCI:       rateCI,
		}

		if sampleSize < query.MinSampleSize {
			vs.InsufficientSample = true
			vs.WarningMessage = fmt.Sprintf("insufficient sample size (N=%d < %d)", sampleSize, query.MinSampleSize)
			insufficientSampleFlag = true
		}

		if isControl {
			vs.AbsoluteLift = 0.0
			vs.AbsoluteLiftCI = ConfidenceInterval{Lower: 0, Upper: 0, CL: 0.95}
			vs.RelativeLift = 0.0
			vs.RelativeLiftCI = ConfidenceInterval{Lower: 0, Upper: 0, CL: 0.95}
			vs.PValue = 1.0
			vs.IsStatisticallySig = false
		} else {
			absLift, absLiftCI := CalculateAbsoluteLiftAndCI(rate, controlRate, sampleSize, controlSample, 0.95)
			relLift, relLiftCI := CalculateRelativeLiftAndCI(rate, controlRate, sampleSize, controlSample, 0.95)
			pValue, isSig := CalculateTwoSampleZTest(convCount, sampleSize, controlConv, controlSample)

			vs.AbsoluteLift = absLift
			vs.AbsoluteLiftCI = absLiftCI
			vs.RelativeLift = relLift
			vs.RelativeLiftCI = relLiftCI
			vs.PValue = pValue
			vs.IsStatisticallySig = isSig
		}

		variantStatsMap[vID] = vs
	}

	metadata.InsufficientSampleWarning = insufficientSampleFlag
	srm := CalculateSRM(sampleCounts, query.ExpectedWeights, query.MinSampleSize)

	reportID := fmt.Sprintf("rep-%s-%s-v%d-%s", query.TenantID, query.ExperimentID, query.ExperimentVersion, string(query.Mode))

	rep := &ExperimentReport{
		ReportID:          reportID,
		TenantID:          query.TenantID,
		ExperimentID:      query.ExperimentID,
		ExperimentVersion: query.ExperimentVersion,
		JourneyID:         query.JourneyID,
		JourneyVersion:    query.JourneyVersion,
		MetricName:        query.MetricName,
		DenominatorName:   query.DenominatorName,
		Mode:              query.Mode,
		StartTime:         query.StartTime,
		EndTime:           query.EndTime,
		Metadata:          metadata,
		SRM:               srm,
		Variants:          variantStatsMap,
	}

	hash, err := rep.CalculateContentHash()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate content hash: %w", err)
	}
	rep.ContentHash = hash

	return rep, nil
}
