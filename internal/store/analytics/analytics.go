package analytics

import (
	"context"
	"time"
)

// DefaultMinCellThreshold is the default threshold below which small-cell counts are suppressed.
const DefaultMinCellThreshold int64 = 10

// DataClassification defines privacy/security levels for analytics facts.
type DataClassification string

const (
	DataClassificationPII       DataClassification = "PII"
	DataClassificationNonPII    DataClassification = "NonPII"
	DataClassificationSensitive DataClassification = "Sensitive"
)

// WindowStatus indicates whether report metrics are provisional or final.
type WindowStatus string

const (
	WindowStatusProvisional WindowStatus = "provisional"
	WindowStatusFinal       WindowStatus = "final"
)

// SampleRatioStatus indicates Sample Ratio Mismatch (SRM) evaluation result.
type SampleRatioStatus string

const (
	SampleRatioStatusNormal   SampleRatioStatus = "normal"
	SampleRatioStatusWarning  SampleRatioStatus = "warning"
	SampleRatioStatusMismatch SampleRatioStatus = "mismatch_detected"
)

// MetricQuery defines parameters for querying analytics aggregate reports.
type MetricQuery struct {
	TenantID          string             `json:"tenant_id"`
	ExperimentID      string             `json:"experiment_id,omitempty"`
	ExperimentVersion uint32             `json:"experiment_version,omitempty"`
	JourneyID         string             `json:"journey_id,omitempty"`
	JourneyVersion    uint32             `json:"journey_version,omitempty"`
	VariantID         string             `json:"variant_id,omitempty"`
	MetricName        string             `json:"metric_name,omitempty"`
	FilterVersion     string             `json:"filter_version,omitempty"`
	IsTest            bool               `json:"is_test"`
	StartTime         time.Time          `json:"start_time"`
	EndTime           time.Time          `json:"end_time"`
	MinCellThreshold  int64              `json:"min_cell_threshold,omitempty"`
	ExpectedWeights   map[string]float64 `json:"expected_weights,omitempty"`
}

// AggregateDenominators holds total counts forming conversion denominators.
type AggregateDenominators struct {
	Assignments     int64   `json:"assignments"`
	Exposures       int64   `json:"exposures"`
	Attempts        int64   `json:"attempts"`
	Sends           int64   `json:"sends"`
	Deliveries      int64   `json:"deliveries"`
	Opens           int64   `json:"opens"`
	Clicks          int64   `json:"clicks"`
	Conversions     int64   `json:"conversions"`
	ConversionValue float64 `json:"conversion_value"`
	UniqueSubjects  int64   `json:"unique_subjects"`
}

// Exclusions contains counts of excluded records by reason.
type Exclusions struct {
	FilterMismatch int64 `json:"filter_mismatch"`
	QualityIssues  int64 `json:"quality_issues"`
	LateWatermark  int64 `json:"late_watermark"`
	TotalExcluded  int64 `json:"total_excluded"`
}

// SampleRatioResult contains SRM evaluation results across experiment variants.
type SampleRatioResult struct {
	Status        SampleRatioStatus  `json:"status"`
	ChiSquare     float64            `json:"chi_square,omitempty"`
	PValue        float64            `json:"p_value,omitempty"`
	ExpectedRatio map[string]float64 `json:"expected_ratio,omitempty"`
	ActualRatio   map[string]float64 `json:"actual_ratio,omitempty"`
	SampleCounts  map[string]int64   `json:"sample_counts,omitempty"`
	IsSuppressed  bool               `json:"is_suppressed"`
}

// VariantMetricReport contains aggregated metrics and denominators for a specific variant.
type VariantMetricReport struct {
	VariantID            string                `json:"variant_id"`
	Denominators         AggregateDenominators `json:"denominators"`
	ConversionRate       float64               `json:"conversion_rate"`
	MeanValuePerSubject float64               `json:"mean_value_per_subject"`
	IsSuppressed         bool                  `json:"is_suppressed"`
	SuppressionReason    string                `json:"suppression_reason,omitempty"`
}

// AnalyticsReport is the full structured response returned by the analytics repository.
type AnalyticsReport struct {
	TenantID          string                `json:"tenant_id"`
	ExperimentID      string                `json:"experiment_id,omitempty"`
	ExperimentVersion uint32                `json:"experiment_version,omitempty"`
	JourneyID         string                `json:"journey_id,omitempty"`
	JourneyVersion    uint32                `json:"journey_version,omitempty"`
	MetricName        string                `json:"metric_name,omitempty"`
	FilterVersion     string                `json:"filter_version"`
	IsTest            bool                  `json:"is_test"`
	StartTime         time.Time             `json:"start_time"`
	EndTime           time.Time             `json:"end_time"`
	Watermark         time.Time             `json:"watermark"`
	ProcessingLag     time.Duration         `json:"processing_lag"`
	WindowStatus      WindowStatus          `json:"window_status"`
	SampleRatio       SampleRatioResult     `json:"sample_ratio"`
	Exclusions        Exclusions            `json:"exclusions"`
	Variants          []VariantMetricReport `json:"variants"`
	GeneratedAt       time.Time             `json:"generated_at"`
}

// Fact types for appending data to ClickHouse tables

type AssignmentFact struct {
	AssignmentID      string    `json:"assignment_id"`
	TenantID          string    `json:"tenant_id"`
	ExperimentID      string    `json:"experiment_id"`
	ExperimentVersion uint32    `json:"experiment_version"`
	JourneyID         string    `json:"journey_id"`
	JourneyVersion    uint32    `json:"journey_version"`
	VariantID         string    `json:"variant_id"`
	SubjectID         string    `json:"subject_id"`
	FilterVersion     string    `json:"filter_version"`
	IsTest            bool      `json:"is_test"`
	AssignedAt        time.Time `json:"assigned_at"`
	IngestedAt        time.Time `json:"ingested_at"`
	Context           string    `json:"context"`
}

type ExposureFact struct {
	ExposureID        string    `json:"exposure_id"`
	AssignmentID      string    `json:"assignment_id"`
	TenantID          string    `json:"tenant_id"`
	ExperimentID      string    `json:"experiment_id"`
	ExperimentVersion uint32    `json:"experiment_version"`
	JourneyID         string    `json:"journey_id"`
	JourneyVersion    uint32    `json:"journey_version"`
	VariantID         string    `json:"variant_id"`
	SubjectID         string    `json:"subject_id"`
	FilterVersion     string    `json:"filter_version"`
	IsTest            bool      `json:"is_test"`
	ExposedAt         time.Time `json:"exposed_at"`
	IngestedAt        time.Time `json:"ingested_at"`
	Context           string    `json:"context"`
}

type ActionAttemptFact struct {
	AttemptID         string    `json:"attempt_id"`
	RunID             string    `json:"run_id"`
	NodeID            string    `json:"node_id"`
	ActivityType      string    `json:"activity_type"`
	TenantID          string    `json:"tenant_id"`
	ExperimentID      string    `json:"experiment_id"`
	ExperimentVersion uint32    `json:"experiment_version"`
	JourneyID         string    `json:"journey_id"`
	JourneyVersion    uint32    `json:"journey_version"`
	VariantID         string    `json:"variant_id"`
	SubjectID         string    `json:"subject_id"`
	AttemptNumber     uint32    `json:"attempt_number"`
	Status            string    `json:"status"`
	FilterVersion     string    `json:"filter_version"`
	IsTest            bool      `json:"is_test"`
	AttemptedAt       time.Time `json:"attempted_at"`
	IngestedAt        time.Time `json:"ingested_at"`
	Metadata          string    `json:"metadata"`
}

type AcceptedSendFact struct {
	SendID            string    `json:"send_id"`
	AttemptID         string    `json:"attempt_id"`
	RunID             string    `json:"run_id"`
	NodeID            string    `json:"node_id"`
	Channel           string    `json:"channel"`
	TenantID          string    `json:"tenant_id"`
	ExperimentID      string    `json:"experiment_id"`
	ExperimentVersion uint32    `json:"experiment_version"`
	JourneyID         string    `json:"journey_id"`
	JourneyVersion    uint32    `json:"journey_version"`
	VariantID         string    `json:"variant_id"`
	SubjectID         string    `json:"subject_id"`
	FilterVersion     string    `json:"filter_version"`
	IsTest            bool      `json:"is_test"`
	AcceptedAt        time.Time `json:"accepted_at"`
	IngestedAt        time.Time `json:"ingested_at"`
	Metadata          string    `json:"metadata"`
}

type DeliveryFact struct {
	DeliveryID        string    `json:"delivery_id"`
	SendID            string    `json:"send_id"`
	RunID             string    `json:"run_id"`
	NodeID            string    `json:"node_id"`
	Channel           string    `json:"channel"`
	TenantID          string    `json:"tenant_id"`
	ExperimentID      string    `json:"experiment_id"`
	ExperimentVersion uint32    `json:"experiment_version"`
	JourneyID         string    `json:"journey_id"`
	JourneyVersion    uint32    `json:"journey_version"`
	VariantID         string    `json:"variant_id"`
	SubjectID         string    `json:"subject_id"`
	DeliveryStatus    string    `json:"delivery_status"`
	FilterVersion     string    `json:"filter_version"`
	IsTest            bool      `json:"is_test"`
	DeliveredAt       time.Time `json:"delivered_at"`
	IngestedAt        time.Time `json:"ingested_at"`
	Metadata          string    `json:"metadata"`
}

type OpenFact struct {
	OpenID            string    `json:"open_id"`
	DeliveryID        string    `json:"delivery_id"`
	RunID             string    `json:"run_id"`
	TenantID          string    `json:"tenant_id"`
	ExperimentID      string    `json:"experiment_id"`
	ExperimentVersion uint32    `json:"experiment_version"`
	JourneyID         string    `json:"journey_id"`
	JourneyVersion    uint32    `json:"journey_version"`
	VariantID         string    `json:"variant_id"`
	SubjectID         string    `json:"subject_id"`
	UserAgent         string    `json:"user_agent"`
	FilterVersion     string    `json:"filter_version"`
	IsTest            bool      `json:"is_test"`
	OpenedAt          time.Time `json:"opened_at"`
	IngestedAt        time.Time `json:"ingested_at"`
	Metadata          string    `json:"metadata"`
}

type ClickFact struct {
	ClickID           string    `json:"click_id"`
	DeliveryID        string    `json:"delivery_id"`
	RunID             string    `json:"run_id"`
	TenantID          string    `json:"tenant_id"`
	ExperimentID      string    `json:"experiment_id"`
	ExperimentVersion uint32    `json:"experiment_version"`
	JourneyID         string    `json:"journey_id"`
	JourneyVersion    uint32    `json:"journey_version"`
	VariantID         string    `json:"variant_id"`
	SubjectID         string    `json:"subject_id"`
	TargetURL         string    `json:"target_url"`
	FilterVersion     string    `json:"filter_version"`
	IsTest            bool      `json:"is_test"`
	ClickedAt         time.Time `json:"clicked_at"`
	IngestedAt        time.Time `json:"ingested_at"`
	Metadata          string    `json:"metadata"`
}

type ConversionFact struct {
	ConversionID      string    `json:"conversion_id"`
	OutcomeID         string    `json:"outcome_id"`
	RunID             string    `json:"run_id"`
	TenantID          string    `json:"tenant_id"`
	ExperimentID      string    `json:"experiment_id"`
	ExperimentVersion uint32    `json:"experiment_version"`
	JourneyID         string    `json:"journey_id"`
	JourneyVersion    uint32    `json:"journey_version"`
	VariantID         string    `json:"variant_id"`
	SubjectID         string    `json:"subject_id"`
	MetricName        string    `json:"metric_name"`
	Value             float64   `json:"value"`
	Unit              string    `json:"unit"`
	FilterVersion     string    `json:"filter_version"`
	IsTest            bool      `json:"is_test"`
	ConvertedAt       time.Time `json:"converted_at"`
	IngestedAt        time.Time `json:"ingested_at"`
	Metadata          string    `json:"metadata"`
}

type DataQualityFact struct {
	QualityEventID    string    `json:"quality_event_id"`
	TenantID          string    `json:"tenant_id"`
	ExperimentID      string    `json:"experiment_id"`
	ExperimentVersion uint32    `json:"experiment_version"`
	JourneyID         string    `json:"journey_id"`
	JourneyVersion    uint32    `json:"journey_version"`
	EntityType        string    `json:"entity_type"`
	EntityID          string    `json:"entity_id"`
	RuleID            string    `json:"rule_id"`
	IssueType         string    `json:"issue_type"`
	Severity          string    `json:"severity"`
	FilterVersion     string    `json:"filter_version"`
	IsTest            bool      `json:"is_test"`
	OccurredAt        time.Time `json:"occurred_at"`
	IngestedAt        time.Time `json:"ingested_at"`
	Details           string    `json:"details"`
}

// Repository defines the contract for accessing analytical metrics and storing facts.
type Repository interface {
	QueryReport(ctx context.Context, query MetricQuery) (*AnalyticsReport, error)
	InsertAssignments(ctx context.Context, facts []AssignmentFact) error
	InsertExposures(ctx context.Context, facts []ExposureFact) error
	InsertActionAttempts(ctx context.Context, facts []ActionAttemptFact) error
	InsertConversions(ctx context.Context, facts []ConversionFact) error
	InsertDataQuality(ctx context.Context, facts []DataQualityFact) error
	GetWatermark(ctx context.Context, tenantID, experimentID, journeyID string, isTest bool) (time.Time, error)
}
