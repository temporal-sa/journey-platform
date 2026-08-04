package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// DBExecutor interface abstracts database operations for ClickHouse execution and unit testing.
type DBExecutor interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// ClickHouseRepository implements the Repository interface for ClickHouse analytics.
type ClickHouseRepository struct {
	db DBExecutor
}

// NewClickHouseRepository creates a new analytics repository instance.
func NewClickHouseRepository(db DBExecutor) *ClickHouseRepository {
	return &ClickHouseRepository{
		db: db,
	}
}

// QueryReport executes analytics queries and aggregates metrics with small-cell suppression.
func (r *ClickHouseRepository) QueryReport(ctx context.Context, query MetricQuery) (*AnalyticsReport, error) {
	threshold := query.MinCellThreshold
	if threshold <= 0 {
		threshold = DefaultMinCellThreshold
	}

	report := &AnalyticsReport{
		TenantID:          query.TenantID,
		ExperimentID:      query.ExperimentID,
		ExperimentVersion: query.ExperimentVersion,
		JourneyID:         query.JourneyID,
		JourneyVersion:    query.JourneyVersion,
		MetricName:        query.MetricName,
		FilterVersion:     query.FilterVersion,
		IsTest:            query.IsTest,
		StartTime:         query.StartTime,
		EndTime:           query.EndTime,
		Variants:          make([]VariantMetricReport, 0),
		GeneratedAt:       time.Now().UTC(),
	}

	// 1. Watermark & Processing Lag
	watermark, err := r.GetWatermark(ctx, query.TenantID, query.ExperimentID, query.JourneyID, query.IsTest)
	if err != nil {
		watermark = query.StartTime
	}
	report.Watermark = watermark

	if !watermark.IsZero() {
		report.ProcessingLag = time.Since(watermark)
		if report.ProcessingLag < 0 {
			report.ProcessingLag = 0
		}
	}

	if watermark.Before(query.EndTime) {
		report.WindowStatus = WindowStatusProvisional
	} else {
		report.WindowStatus = WindowStatusFinal
	}

	// 2. Query Exclusions
	exclusions, err := r.queryExclusions(ctx, query)
	if err == nil {
		report.Exclusions = exclusions
	}

	// 3. Query Variant Aggregates
	variantReports, err := r.queryVariantAggregates(ctx, query, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to query variant aggregates: %w", err)
	}
	report.Variants = variantReports

	// 4. Sample Ratio Mismatch (SRM) Evaluation
	report.SampleRatio = r.evaluateSampleRatio(variantReports, query.ExpectedWeights, threshold)

	return report, nil
}

func (r *ClickHouseRepository) queryExclusions(ctx context.Context, query MetricQuery) (Exclusions, error) {
	var excl Exclusions
	if r.db == nil {
		return excl, nil
	}

	sqlQuery := `
		SELECT
			countIf(issue_type = 'filter_mismatch') AS filter_mismatch,
			countIf(issue_type = 'quality_issue' OR issue_type = 'schema_invalid' OR issue_type = 'duplicate') AS quality_issues,
			countIf(issue_type = 'late_watermark') AS late_watermark,
			count() AS total_excluded
		FROM fact_data_quality
		WHERE tenant_id = ? AND is_test = ?
	`
	args := []interface{}{query.TenantID, query.IsTest}

	if !query.StartTime.IsZero() && !query.EndTime.IsZero() {
		sqlQuery += " AND occurred_at BETWEEN ? AND ?"
		args = append(args, query.StartTime, query.EndTime)
	}

	row := r.db.QueryRowContext(ctx, sqlQuery, args...)
	err := row.Scan(&excl.FilterMismatch, &excl.QualityIssues, &excl.LateWatermark, &excl.TotalExcluded)
	if err != nil && err != sql.ErrNoRows {
		return excl, err
	}
	return excl, nil
}

func (r *ClickHouseRepository) queryVariantAggregates(ctx context.Context, query MetricQuery, threshold int64) ([]VariantMetricReport, error) {
	if r.db == nil {
		return []VariantMetricReport{}, nil
	}

	// Query ClickHouse fact tables joined/grouped by variant_id
	sqlQuery := `
		SELECT
			variant_id,
			uniqExact(assignment_id) AS assignments,
			uniqExact(exposure_id) AS exposures,
			uniqExact(send_id) AS sends,
			uniqExact(delivery_id) AS deliveries,
			uniqExact(open_id) AS opens,
			uniqExact(click_id) AS clicks,
			uniqExact(conversion_id) AS conversions,
			sum(conversion_value) AS conversion_value,
			uniqExact(subject_id) AS unique_subjects
		FROM (
			SELECT variant_id, assignment_id, '' AS exposure_id, '' AS send_id, '' AS delivery_id, '' AS open_id, '' AS click_id, '' AS conversion_id, 0.0 AS conversion_value, subject_id
			FROM fact_assignments
			WHERE tenant_id = ? AND is_test = ?
			UNION ALL
			SELECT variant_id, '' AS assignment_id, exposure_id, '' AS send_id, '' AS delivery_id, '' AS open_id, '' AS click_id, '' AS conversion_id, 0.0 AS conversion_value, subject_id
			FROM fact_exposures
			WHERE tenant_id = ? AND is_test = ?
			UNION ALL
			SELECT variant_id, '' AS assignment_id, '' AS exposure_id, '' AS send_id, '' AS delivery_id, '' AS open_id, '' AS click_id, '' AS conversion_id, 0.0 AS conversion_value, subject_id
			FROM fact_accepted_sends
			WHERE tenant_id = ? AND is_test = ?
			UNION ALL
			SELECT variant_id, '' AS assignment_id, '' AS exposure_id, '' AS send_id, delivery_id, '' AS open_id, '' AS click_id, '' AS conversion_id, 0.0 AS conversion_value, subject_id
			FROM fact_deliveries
			WHERE tenant_id = ? AND is_test = ?
			UNION ALL
			SELECT variant_id, '' AS assignment_id, '' AS exposure_id, '' AS send_id, '' AS delivery_id, open_id, '' AS click_id, '' AS conversion_id, 0.0 AS conversion_value, subject_id
			FROM fact_opens
			WHERE tenant_id = ? AND is_test = ?
			UNION ALL
			SELECT variant_id, '' AS assignment_id, '' AS exposure_id, '' AS send_id, '' AS delivery_id, '' AS open_id, click_id, '' AS conversion_id, 0.0 AS conversion_value, subject_id
			FROM fact_clicks
			WHERE tenant_id = ? AND is_test = ?
			UNION ALL
			SELECT variant_id, '' AS assignment_id, '' AS exposure_id, '' AS send_id, '' AS delivery_id, '' AS open_id, '' AS click_id, conversion_id, value AS conversion_value, subject_id
			FROM fact_conversions
			WHERE tenant_id = ? AND is_test = ?
		)
		GROUP BY variant_id
		ORDER BY variant_id
	`
	args := []interface{}{
		query.TenantID, query.IsTest,
		query.TenantID, query.IsTest,
		query.TenantID, query.IsTest,
		query.TenantID, query.IsTest,
		query.TenantID, query.IsTest,
		query.TenantID, query.IsTest,
		query.TenantID, query.IsTest,
	}

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []VariantMetricReport
	for rows.Next() {
		var vr VariantMetricReport
		var den AggregateDenominators
		err := rows.Scan(
			&vr.VariantID,
			&den.Assignments,
			&den.Exposures,
			&den.Sends,
			&den.Deliveries,
			&den.Opens,
			&den.Clicks,
			&den.Conversions,
			&den.ConversionValue,
			&den.UniqueSubjects,
		)
		if err != nil {
			return nil, err
		}

		// Calculate rate and mean before suppression check
		if den.Exposures > 0 {
			vr.ConversionRate = float64(den.Conversions) / float64(den.Exposures)
		} else if den.Assignments > 0 {
			vr.ConversionRate = float64(den.Conversions) / float64(den.Assignments)
		}
		if den.UniqueSubjects > 0 {
			vr.MeanValuePerSubject = den.ConversionValue / float64(den.UniqueSubjects)
		}

		// Apply small-cell suppression
		r.applySuppression(&vr, den, threshold)
		reports = append(reports, vr)
	}

	return reports, nil
}

func (r *ClickHouseRepository) applySuppression(vr *VariantMetricReport, den AggregateDenominators, threshold int64) {
	primaryCount := den.Exposures
	if primaryCount == 0 {
		primaryCount = den.Assignments
	}
	if primaryCount == 0 {
		primaryCount = den.UniqueSubjects
	}

	// Small cell threshold enforcement: if count is > 0 and < threshold, suppress cell
	if primaryCount > 0 && primaryCount < threshold {
		vr.IsSuppressed = true
		vr.SuppressionReason = fmt.Sprintf("small-cell threshold suppression (count %d < %d)", primaryCount, threshold)
		vr.Denominators = AggregateDenominators{}
		vr.ConversionRate = 0.0
		vr.MeanValuePerSubject = 0.0
	} else {
		vr.IsSuppressed = false
		vr.Denominators = den
	}
}

func (r *ClickHouseRepository) evaluateSampleRatio(variants []VariantMetricReport, expectedWeights map[string]float64, threshold int64) SampleRatioResult {
	result := SampleRatioResult{
		Status:        SampleRatioStatusNormal,
		ExpectedRatio: make(map[string]float64),
		ActualRatio:   make(map[string]float64),
		SampleCounts:  make(map[string]int64),
	}

	if len(variants) == 0 {
		return result
	}

	var totalSamples int64
	for _, v := range variants {
		if v.IsSuppressed {
			result.IsSuppressed = true
			return result
		}
		count := v.Denominators.Exposures
		if count == 0 {
			count = v.Denominators.Assignments
		}
		result.SampleCounts[v.VariantID] = count
		totalSamples += count
	}

	if totalSamples < threshold {
		result.IsSuppressed = true
		return result
	}

	numVariants := float64(len(variants))
	var chiSquare float64
	for _, v := range variants {
		count := result.SampleCounts[v.VariantID]
		actualRatio := float64(count) / float64(totalSamples)
		result.ActualRatio[v.VariantID] = actualRatio

		expectedWeight := 1.0 / numVariants
		if w, ok := expectedWeights[v.VariantID]; ok && w > 0 {
			expectedWeight = w
		}
		result.ExpectedRatio[v.VariantID] = expectedWeight

		expectedCount := float64(totalSamples) * expectedWeight
		if expectedCount > 0 {
			diff := float64(count) - expectedCount
			chiSquare += (diff * diff) / expectedCount
		}
	}

	result.ChiSquare = chiSquare
	// Chi-square threshold evaluation (df = k-1)
	// For 1 degree of freedom (2 variants):
	// chi2 > 10.83 => p < 0.001 (Mismatch)
	// chi2 > 3.84  => p < 0.05  (Warning)
	if chiSquare > 10.83 {
		result.Status = SampleRatioStatusMismatch
		result.PValue = 0.0009
	} else if chiSquare > 3.84 {
		result.Status = SampleRatioStatusWarning
		result.PValue = 0.04
	} else {
		result.Status = SampleRatioStatusNormal
		result.PValue = 0.50
	}

	return result
}

// GetWatermark queries the maximum ingested/event timestamp for the specified entity scope.
func (r *ClickHouseRepository) GetWatermark(ctx context.Context, tenantID, experimentID, journeyID string, isTest bool) (time.Time, error) {
	if r.db == nil {
		return time.Time{}, nil
	}

	sqlQuery := `
		SELECT max(wm) FROM (
			SELECT max(exposed_at) AS wm FROM fact_exposures WHERE tenant_id = ? AND is_test = ?
			UNION ALL
			SELECT max(converted_at) AS wm FROM fact_conversions WHERE tenant_id = ? AND is_test = ?
			UNION ALL
			SELECT max(assigned_at) AS wm FROM fact_assignments WHERE tenant_id = ? AND is_test = ?
		)
	`
	var watermark sql.NullTime
	row := r.db.QueryRowContext(ctx, sqlQuery, tenantID, isTest, tenantID, isTest, tenantID, isTest)
	err := row.Scan(&watermark)
	if err != nil || !watermark.Valid {
		return time.Time{}, err
	}
	return watermark.Time, nil
}

// Batch Insert Operations

func (r *ClickHouseRepository) InsertAssignments(ctx context.Context, facts []AssignmentFact) error {
	if r.db == nil || len(facts) == 0 {
		return nil
	}
	query := `INSERT INTO fact_assignments (assignment_id, tenant_id, experiment_id, experiment_version, journey_id, journey_version, variant_id, subject_id, filter_version, is_test, assigned_at, ingested_at, context) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	for _, f := range facts {
		ingestedAt := f.IngestedAt
		if ingestedAt.IsZero() {
			ingestedAt = time.Now().UTC()
		}
		_, err := r.db.ExecContext(ctx, query, f.AssignmentID, f.TenantID, f.ExperimentID, f.ExperimentVersion, f.JourneyID, f.JourneyVersion, f.VariantID, f.SubjectID, f.FilterVersion, f.IsTest, f.AssignedAt, ingestedAt, f.Context)
		if err != nil {
			return fmt.Errorf("failed to insert assignment fact: %w", err)
		}
	}
	return nil
}

func (r *ClickHouseRepository) InsertExposures(ctx context.Context, facts []ExposureFact) error {
	if r.db == nil || len(facts) == 0 {
		return nil
	}
	query := `INSERT INTO fact_exposures (exposure_id, assignment_id, tenant_id, experiment_id, experiment_version, journey_id, journey_version, variant_id, subject_id, filter_version, is_test, exposed_at, ingested_at, context) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	for _, f := range facts {
		ingestedAt := f.IngestedAt
		if ingestedAt.IsZero() {
			ingestedAt = time.Now().UTC()
		}
		_, err := r.db.ExecContext(ctx, query, f.ExposureID, f.AssignmentID, f.TenantID, f.ExperimentID, f.ExperimentVersion, f.JourneyID, f.JourneyVersion, f.VariantID, f.SubjectID, f.FilterVersion, f.IsTest, f.ExposedAt, ingestedAt, f.Context)
		if err != nil {
			return fmt.Errorf("failed to insert exposure fact: %w", err)
		}
	}
	return nil
}

func (r *ClickHouseRepository) InsertConversions(ctx context.Context, facts []ConversionFact) error {
	if r.db == nil || len(facts) == 0 {
		return nil
	}
	query := `INSERT INTO fact_conversions (conversion_id, outcome_id, run_id, tenant_id, experiment_id, experiment_version, journey_id, journey_version, variant_id, subject_id, metric_name, value, unit, filter_version, is_test, converted_at, ingested_at, metadata) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	for _, f := range facts {
		ingestedAt := f.IngestedAt
		if ingestedAt.IsZero() {
			ingestedAt = time.Now().UTC()
		}
		val := f.Value
		if val == 0 {
			val = 1.0
		}
		unit := f.Unit
		if unit == "" {
			unit = "count"
		}
		_, err := r.db.ExecContext(ctx, query, f.ConversionID, f.OutcomeID, f.RunID, f.TenantID, f.ExperimentID, f.ExperimentVersion, f.JourneyID, f.JourneyVersion, f.VariantID, f.SubjectID, f.MetricName, val, unit, f.FilterVersion, f.IsTest, f.ConvertedAt, ingestedAt, f.Metadata)
		if err != nil {
			return fmt.Errorf("failed to insert conversion fact: %w", err)
		}
	}
	return nil
}

func (r *ClickHouseRepository) InsertDataQuality(ctx context.Context, facts []DataQualityFact) error {
	if r.db == nil || len(facts) == 0 {
		return nil
	}
	query := `INSERT INTO fact_data_quality (quality_event_id, tenant_id, experiment_id, experiment_version, journey_id, journey_version, entity_type, entity_id, rule_id, issue_type, severity, filter_version, is_test, occurred_at, ingested_at, details) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	for _, f := range facts {
		ingestedAt := f.IngestedAt
		if ingestedAt.IsZero() {
			ingestedAt = time.Now().UTC()
		}
		severity := f.Severity
		if severity == "" {
			severity = "error"
		}
		_, err := r.db.ExecContext(ctx, query, f.QualityEventID, f.TenantID, f.ExperimentID, f.ExperimentVersion, f.JourneyID, f.JourneyVersion, f.EntityType, f.EntityID, f.RuleID, f.IssueType, severity, f.FilterVersion, f.IsTest, f.OccurredAt, ingestedAt, f.Details)
		if err != nil {
			return fmt.Errorf("failed to insert data quality fact: %w", err)
		}
	}
	return nil
}

// In-Memory Repository for fast unit testing & isolated verification without external ClickHouse
type MemoryRepository struct {
	assignments []AssignmentFact
	exposures   []ExposureFact
	attempts    []ActionAttemptFact
	conversions []ConversionFact
	quality     []DataQualityFact
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		assignments: make([]AssignmentFact, 0),
		exposures:   make([]ExposureFact, 0),
		attempts:    make([]ActionAttemptFact, 0),
		conversions: make([]ConversionFact, 0),
		quality:     make([]DataQualityFact, 0),
	}
}

func (m *MemoryRepository) InsertAssignments(ctx context.Context, facts []AssignmentFact) error {
	m.assignments = append(m.assignments, facts...)
	return nil
}

func (m *MemoryRepository) InsertExposures(ctx context.Context, facts []ExposureFact) error {
	m.exposures = append(m.exposures, facts...)
	return nil
}

func (m *MemoryRepository) InsertActionAttempts(ctx context.Context, facts []ActionAttemptFact) error {
	m.attempts = append(m.attempts, facts...)
	return nil
}

func (m *MemoryRepository) InsertConversions(ctx context.Context, facts []ConversionFact) error {
	m.conversions = append(m.conversions, facts...)
	return nil
}

func (m *MemoryRepository) InsertDataQuality(ctx context.Context, facts []DataQualityFact) error {
	m.quality = append(m.quality, facts...)
	return nil
}

func (m *MemoryRepository) GetWatermark(ctx context.Context, tenantID, experimentID, journeyID string, isTest bool) (time.Time, error) {
	var maxTime time.Time
	for _, a := range m.assignments {
		if a.TenantID == tenantID && a.IsTest == isTest && a.AssignedAt.After(maxTime) {
			maxTime = a.AssignedAt
		}
	}
	for _, e := range m.exposures {
		if e.TenantID == tenantID && e.IsTest == isTest && e.ExposedAt.After(maxTime) {
			maxTime = e.ExposedAt
		}
	}
	for _, c := range m.conversions {
		if c.TenantID == tenantID && c.IsTest == isTest && c.ConvertedAt.After(maxTime) {
			maxTime = c.ConvertedAt
		}
	}
	return maxTime, nil
}

func (m *MemoryRepository) QueryReport(ctx context.Context, query MetricQuery) (*AnalyticsReport, error) {
	threshold := query.MinCellThreshold
	if threshold <= 0 {
		threshold = DefaultMinCellThreshold
	}

	report := &AnalyticsReport{
		TenantID:          query.TenantID,
		ExperimentID:      query.ExperimentID,
		ExperimentVersion: query.ExperimentVersion,
		JourneyID:         query.JourneyID,
		JourneyVersion:    query.JourneyVersion,
		MetricName:        query.MetricName,
		FilterVersion:     query.FilterVersion,
		IsTest:            query.IsTest,
		StartTime:         query.StartTime,
		EndTime:           query.EndTime,
		GeneratedAt:       time.Now().UTC(),
	}

	watermark, _ := m.GetWatermark(ctx, query.TenantID, query.ExperimentID, query.JourneyID, query.IsTest)
	report.Watermark = watermark

	if !watermark.IsZero() {
		report.ProcessingLag = time.Since(watermark)
		if report.ProcessingLag < 0 {
			report.ProcessingLag = 0
		}
	}

	if watermark.Before(query.EndTime) {
		report.WindowStatus = WindowStatusProvisional
	} else {
		report.WindowStatus = WindowStatusFinal
	}

	// Exclusions
	var excl Exclusions
	for _, q := range m.quality {
		if q.TenantID == query.TenantID && q.IsTest == query.IsTest {
			switch q.IssueType {
			case "filter_mismatch":
				excl.FilterMismatch++
			case "quality_issue", "schema_invalid", "duplicate":
				excl.QualityIssues++
			case "late_watermark":
				excl.LateWatermark++
			}
			excl.TotalExcluded++
		}
	}
	report.Exclusions = excl

	// Aggregate metrics by variant with deduplication & test mode isolation
	variantMap := make(map[string]*VariantMetricReport)
	uniqueSubjectsMap := make(map[string]map[string]bool)
	seenAssignments := make(map[string]bool)
	seenExposures := make(map[string]bool)
	seenAttempts := make(map[string]bool)
	seenConversions := make(map[string]bool)

	for _, a := range m.assignments {
		if a.TenantID != query.TenantID || a.IsTest != query.IsTest {
			continue
		}
		if query.ExperimentID != "" && a.ExperimentID != query.ExperimentID {
			continue
		}
		if query.ExperimentVersion > 0 && a.ExperimentVersion != query.ExperimentVersion {
			continue
		}
		if query.JourneyID != "" && a.JourneyID != query.JourneyID {
			continue
		}
		if query.JourneyVersion > 0 && a.JourneyVersion != query.JourneyVersion {
			continue
		}
		if seenAssignments[a.AssignmentID] {
			continue
		}
		seenAssignments[a.AssignmentID] = true

		vID := a.VariantID
		if _, ok := variantMap[vID]; !ok {
			variantMap[vID] = &VariantMetricReport{VariantID: vID}
			uniqueSubjectsMap[vID] = make(map[string]bool)
		}
		variantMap[vID].Denominators.Assignments++
		uniqueSubjectsMap[vID][a.SubjectID] = true
	}

	for _, e := range m.exposures {
		if e.TenantID != query.TenantID || e.IsTest != query.IsTest {
			continue
		}
		if query.ExperimentID != "" && e.ExperimentID != query.ExperimentID {
			continue
		}
		if query.ExperimentVersion > 0 && e.ExperimentVersion != query.ExperimentVersion {
			continue
		}
		if query.JourneyID != "" && e.JourneyID != query.JourneyID {
			continue
		}
		if query.JourneyVersion > 0 && e.JourneyVersion != query.JourneyVersion {
			continue
		}
		if seenExposures[e.ExposureID] {
			continue
		}
		seenExposures[e.ExposureID] = true

		vID := e.VariantID
		if _, ok := variantMap[vID]; !ok {
			variantMap[vID] = &VariantMetricReport{VariantID: vID}
			uniqueSubjectsMap[vID] = make(map[string]bool)
		}
		variantMap[vID].Denominators.Exposures++
		uniqueSubjectsMap[vID][e.SubjectID] = true
	}

	for _, att := range m.attempts {
		if att.TenantID != query.TenantID || att.IsTest != query.IsTest {
			continue
		}
		if query.ExperimentID != "" && att.ExperimentID != query.ExperimentID {
			continue
		}
		if query.ExperimentVersion > 0 && att.ExperimentVersion != query.ExperimentVersion {
			continue
		}
		if query.JourneyID != "" && att.JourneyID != query.JourneyID {
			continue
		}
		if query.JourneyVersion > 0 && att.JourneyVersion != query.JourneyVersion {
			continue
		}
		if seenAttempts[att.AttemptID] {
			continue
		}
		seenAttempts[att.AttemptID] = true

		vID := att.VariantID
		if _, ok := variantMap[vID]; !ok {
			variantMap[vID] = &VariantMetricReport{VariantID: vID}
			uniqueSubjectsMap[vID] = make(map[string]bool)
		}
		variantMap[vID].Denominators.Attempts++
		uniqueSubjectsMap[vID][att.SubjectID] = true
	}

	for _, c := range m.conversions {
		if c.TenantID != query.TenantID || c.IsTest != query.IsTest {
			continue
		}
		if query.ExperimentID != "" && c.ExperimentID != query.ExperimentID {
			continue
		}
		if query.ExperimentVersion > 0 && c.ExperimentVersion != query.ExperimentVersion {
			continue
		}
		if query.JourneyID != "" && c.JourneyID != query.JourneyID {
			continue
		}
		if query.JourneyVersion > 0 && c.JourneyVersion != query.JourneyVersion {
			continue
		}
		if query.MetricName != "" && c.MetricName != query.MetricName {
			continue
		}
		if seenConversions[c.ConversionID] {
			continue
		}
		seenConversions[c.ConversionID] = true

		vID := c.VariantID
		if _, ok := variantMap[vID]; !ok {
			variantMap[vID] = &VariantMetricReport{VariantID: vID}
			uniqueSubjectsMap[vID] = make(map[string]bool)
		}
		variantMap[vID].Denominators.Conversions++
		val := c.Value
		if val == 0 {
			val = 1.0
		}
		variantMap[vID].Denominators.ConversionValue += val
		uniqueSubjectsMap[vID][c.SubjectID] = true
	}

	variants := make([]VariantMetricReport, 0, len(variantMap))
	for vID, vr := range variantMap {
		vr.Denominators.UniqueSubjects = int64(len(uniqueSubjectsMap[vID]))
		if vr.Denominators.Exposures > 0 {
			vr.ConversionRate = float64(vr.Denominators.Conversions) / float64(vr.Denominators.Exposures)
		} else if vr.Denominators.Assignments > 0 {
			vr.ConversionRate = float64(vr.Denominators.Conversions) / float64(vr.Denominators.Assignments)
		}
		if vr.Denominators.UniqueSubjects > 0 {
			vr.MeanValuePerSubject = vr.Denominators.ConversionValue / float64(vr.Denominators.UniqueSubjects)
		}

		chRepo := &ClickHouseRepository{}
		chRepo.applySuppression(vr, vr.Denominators, threshold)
		variants = append(variants, *vr)
	}

	report.Variants = variants
	chRepo := &ClickHouseRepository{}
	report.SampleRatio = chRepo.evaluateSampleRatio(variants, query.ExpectedWeights, threshold)

	return report, nil
}
