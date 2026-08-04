package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// DataClassification defines privacy/security classification levels.
type DataClassification string

const (
	DataClassificationPII       DataClassification = "PII"
	DataClassificationNonPII    DataClassification = "NonPII"
	DataClassificationSensitive DataClassification = "Sensitive"
)

// ComponentType defines types of catalog components.
type ComponentType string

const (
	ComponentTypeActivity  ComponentType = "activity"
	ComponentTypeTrigger   ComponentType = "trigger"
	ComponentTypeAction    ComponentType = "action"
	ComponentTypeCondition ComponentType = "condition"
)

// ExperimentStatus defines experiment lifecycles.
type ExperimentStatus string

const (
	ExperimentStatusDraft     ExperimentStatus = "draft"
	ExperimentStatusActive    ExperimentStatus = "active"
	ExperimentStatusPaused    ExperimentStatus = "paused"
	ExperimentStatusCompleted ExperimentStatus = "completed"
)

// ActionResultStatus defines execution status of actions.
type ActionResultStatus string

const (
	ActionResultStatusSuccess  ActionResultStatus = "success"
	ActionResultStatusFailure  ActionResultStatus = "failure"
	ActionResultStatusRetrying ActionResultStatus = "retrying"
)

// TestRunStatus defines execution status of test runs.
type TestRunStatus string

const (
	TestRunStatusPending TestRunStatus = "pending"
	TestRunStatusPassed  TestRunStatus = "passed"
	TestRunStatusFailed  TestRunStatus = "failed"
)

// ValidationSeverity defines severity of workflow validation issues.
type ValidationSeverity string

const (
	ValidationSeverityError   ValidationSeverity = "error"
	ValidationSeverityWarning ValidationSeverity = "warning"
	ValidationSeverityInfo    ValidationSeverity = "info"
)

// RunStatus defines workflow run status.
type RunStatus string

const (
	RunStatusRunning    RunStatus = "running"
	RunStatusCompleted  RunStatus = "completed"
	RunStatusFailed     RunStatus = "failed"
	RunStatusTerminated RunStatus = "terminated"
)

// Standard Default Schema Version
const DefaultSchemaVersion = "1.0"

// -----------------------------------------------------------------------------
// 1. EventEnvelope
// -----------------------------------------------------------------------------

type EventEnvelope struct {
	SchemaVersion      string                 `json:"schema_version" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	EventID            string                 `json:"event_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	TraceID            string                 `json:"trace_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	EventType          string                 `json:"event_type" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Source             string                 `json:"source" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Subject            string                 `json:"subject,omitempty" storage:"allowed" logging:"redacted" temporal_history:"allowed" api_exposure:"internal"`
	Timestamp          time.Time              `json:"timestamp" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ContentHash        string                 `json:"content_hash,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	DataClassification DataClassification     `json:"data_classification" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Data               map[string]interface{} `json:"data" storage:"encrypted" logging:"prohibited" temporal_history:"prohibited" api_exposure:"prohibited"`
}

func (e *EventEnvelope) CalculateSHA256() (string, error) {
	clone := *e
	clone.ContentHash = ""
	return calculateDeterministicHash(clone)
}

// -----------------------------------------------------------------------------
// 2. GraphDraft
// -----------------------------------------------------------------------------

type NodePosition struct {
	X float64 `json:"x,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Y float64 `json:"y,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
}

type GraphNode struct {
	ID       string                 `json:"id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Type     string                 `json:"type" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Name     string                 `json:"name" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Config   map[string]interface{} `json:"config,omitempty" storage:"allowed" logging:"redacted" temporal_history:"allowed" api_exposure:"internal"`
	Position *NodePosition          `json:"position,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
}

type GraphEdge struct {
	ID        string `json:"id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Source    string `json:"source" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Target    string `json:"target" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Condition string `json:"condition,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
}

type GraphDraft struct {
	SchemaVersion string      `json:"schema_version" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	DraftID       string      `json:"draft_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	TenantID      string      `json:"tenant_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Name          string      `json:"name" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Description   string      `json:"description,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Version       int         `json:"version" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Nodes         []GraphNode `json:"nodes" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Edges         []GraphEdge `json:"edges" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ContentHash   string      `json:"content_hash,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	CreatedAt     time.Time   `json:"created_at,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	UpdatedAt     time.Time   `json:"updated_at,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
}

func (g *GraphDraft) CalculateSHA256() (string, error) {
	clone := *g
	clone.ContentHash = ""
	return calculateDeterministicHash(clone)
}

// -----------------------------------------------------------------------------
// 3. CompiledIR
// -----------------------------------------------------------------------------

type RetryPolicy struct {
	MaximumAttempts        int `json:"maximum_attempts,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	InitialIntervalSeconds int `json:"initial_interval_seconds,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
}

type IRNode struct {
	ID             string                 `json:"id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Type           string                 `json:"type" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ActivityName   string                 `json:"activity_name" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Params         map[string]interface{} `json:"params,omitempty" storage:"allowed" logging:"redacted" temporal_history:"allowed" api_exposure:"internal"`
	TimeoutSeconds int                    `json:"timeout_seconds,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	RetryPolicy    *RetryPolicy           `json:"retry_policy,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
}

type IREdge struct {
	ID                  string `json:"id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	SourceID            string `json:"source_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	TargetID            string `json:"target_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ConditionExpression string `json:"condition_expression,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
}

type CompiledIR struct {
	SchemaVersion string    `json:"schema_version" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	IRID          string    `json:"ir_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	DraftID       string    `json:"draft_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	TenantID      string    `json:"tenant_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Version       int       `json:"version" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	EntryNodeID   string    `json:"entry_node_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Nodes         []IRNode  `json:"nodes" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Edges         []IREdge  `json:"edges" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ContentHash   string    `json:"content_hash" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	CompiledAt    time.Time `json:"compiled_at,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
}

func (ir *CompiledIR) CalculateSHA256() (string, error) {
	clone := *ir
	clone.ContentHash = ""
	return calculateDeterministicHash(clone)
}

// -----------------------------------------------------------------------------
// 4. CatalogRecord
// -----------------------------------------------------------------------------

type CatalogRecord struct {
	SchemaVersion    string                 `json:"schema_version" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	RecordID         string                 `json:"record_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Name             string                 `json:"name" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ComponentType    ComponentType          `json:"component_type" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Version          string                 `json:"version" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Description      string                 `json:"description" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	SchemaDefinition map[string]interface{} `json:"schema_definition,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ContentHash      string                 `json:"content_hash,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Tags             []string               `json:"tags" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	IsDeprecated     bool                   `json:"is_deprecated" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
}

func (c *CatalogRecord) CalculateSHA256() (string, error) {
	clone := *c
	clone.ContentHash = ""
	return calculateDeterministicHash(clone)
}

// -----------------------------------------------------------------------------
// 5. ExperimentDefinition
// -----------------------------------------------------------------------------

type ExperimentVariant struct {
	VariantID         string                 `json:"variant_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Name              string                 `json:"name" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	WeightBasisPoints int                    `json:"weight_basis_points" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Config            map[string]interface{} `json:"config,omitempty" storage:"allowed" logging:"redacted" temporal_history:"allowed" api_exposure:"internal"`
}

type ExperimentDefinition struct {
	SchemaVersion  string              `json:"schema_version" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ExperimentID   string              `json:"experiment_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Name           string              `json:"name" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Description    string              `json:"description,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Status         ExperimentStatus    `json:"status" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Variants       []ExperimentVariant `json:"variants" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	TargetAudience string              `json:"target_audience,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ContentHash    string              `json:"content_hash,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	CreatedAt      time.Time           `json:"created_at,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
}

func (exp *ExperimentDefinition) CalculateSHA256() (string, error) {
	clone := *exp
	clone.ContentHash = ""
	return calculateDeterministicHash(clone)
}

// -----------------------------------------------------------------------------
// 6. AssignmentExposure
// -----------------------------------------------------------------------------

type AssignmentExposure struct {
	SchemaVersion      string                 `json:"schema_version" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	AssignmentID       string                 `json:"assignment_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ExperimentID       string                 `json:"experiment_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	VariantID          string                 `json:"variant_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	SubjectID          string                 `json:"subject_id" storage:"allowed" logging:"redacted" temporal_history:"allowed" api_exposure:"internal"`
	AssignedAt         time.Time              `json:"assigned_at" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ExposedAt          time.Time              `json:"exposed_at" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	WeightBasisPoints  int                    `json:"weight_basis_points" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	DataClassification DataClassification     `json:"data_classification" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Context            map[string]interface{} `json:"context,omitempty" storage:"encrypted" logging:"prohibited" temporal_history:"prohibited" api_exposure:"prohibited"`
}

// -----------------------------------------------------------------------------
// 7. NormalizedOutcome
// -----------------------------------------------------------------------------

type NormalizedOutcome struct {
	SchemaVersion      string                 `json:"schema_version" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	OutcomeID          string                 `json:"outcome_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	RunID              string                 `json:"run_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	EventName          string                 `json:"event_name" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Value              float64                `json:"value" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Unit               string                 `json:"unit" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	DataClassification DataClassification     `json:"data_classification" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Timestamp          time.Time              `json:"timestamp" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Metadata           map[string]interface{} `json:"metadata,omitempty" storage:"encrypted" logging:"prohibited" temporal_history:"prohibited" api_exposure:"prohibited"`
}

// -----------------------------------------------------------------------------
// 8. AggregateReport
// -----------------------------------------------------------------------------

type MetricSummary struct {
	MetricName              string  `json:"metric_name" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	TotalCount              int64   `json:"total_count" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Mean                    float64 `json:"mean" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	WeightBasisPoints       int     `json:"weight_basis_points" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ConfidenceIntervalLower float64 `json:"confidence_interval_lower,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ConfidenceIntervalUpper float64 `json:"confidence_interval_upper,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
}

type AggregateReport struct {
	SchemaVersion string          `json:"schema_version" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ReportID      string          `json:"report_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	TenantID      string          `json:"tenant_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ExperimentID  string          `json:"experiment_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	PeriodStart   time.Time       `json:"period_start" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	PeriodEnd     time.Time       `json:"period_end" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Metrics       []MetricSummary `json:"metrics" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	GeneratedAt   time.Time       `json:"generated_at" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
}

// -----------------------------------------------------------------------------
// 9. StaticList
// -----------------------------------------------------------------------------

type StaticList struct {
	SchemaVersion      string             `json:"schema_version" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ListID             string             `json:"list_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Name               string             `json:"name" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Description        string             `json:"description,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ItemCount          int                `json:"item_count" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	DataClassification DataClassification `json:"data_classification" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Items              []string           `json:"items" storage:"encrypted" logging:"prohibited" temporal_history:"prohibited" api_exposure:"prohibited"`
	ContentHash        string             `json:"content_hash,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	CreatedAt          time.Time          `json:"created_at,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	UpdatedAt          time.Time          `json:"updated_at,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
}

func (s *StaticList) CalculateSHA256() (string, error) {
	clone := *s
	clone.ContentHash = ""
	return calculateDeterministicHash(clone)
}

// -----------------------------------------------------------------------------
// 10. TestRun
// -----------------------------------------------------------------------------

type TestRun struct {
	SchemaVersion    string                 `json:"schema_version" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	TestRunID        string                 `json:"test_run_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	DraftID          string                 `json:"draft_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	IRID             string                 `json:"ir_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Status           TestRunStatus          `json:"status" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	MockInputs       map[string]interface{} `json:"mock_inputs,omitempty" storage:"encrypted" logging:"prohibited" temporal_history:"prohibited" api_exposure:"prohibited"`
	ExpectedOutcomes map[string]interface{} `json:"expected_outcomes,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ActualOutcomes   map[string]interface{} `json:"actual_outcomes,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ExecutionTimeMS  int64                  `json:"execution_time_ms" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	CreatedAt        time.Time              `json:"created_at" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
}

// -----------------------------------------------------------------------------
// 11. Workflow Inputs / Signals / Queries / Activities
// -----------------------------------------------------------------------------

type WorkflowInput struct {
	SchemaVersion      string                 `json:"schema_version" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	WorkflowID         string                 `json:"workflow_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	RunID              string                 `json:"run_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	TenantID           string                 `json:"tenant_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	TriggerEventID     string                 `json:"trigger_event_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	IRID               string                 `json:"ir_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	DataClassification DataClassification     `json:"data_classification" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	InputPayload       map[string]interface{} `json:"input_payload" storage:"encrypted" logging:"prohibited" temporal_history:"prohibited" api_exposure:"prohibited"`
}

type WorkflowSignal struct {
	SchemaVersion      string                 `json:"schema_version" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	SignalName         string                 `json:"signal_name" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	RunID              string                 `json:"run_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Payload            map[string]interface{} `json:"payload" storage:"encrypted" logging:"prohibited" temporal_history:"prohibited" api_exposure:"prohibited"`
	DataClassification DataClassification     `json:"data_classification" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
}

type WorkflowQuery struct {
	SchemaVersion string                 `json:"schema_version" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	QueryType     string                 `json:"query_type" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	RunID         string                 `json:"run_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Params        map[string]interface{} `json:"params,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
}

type WorkflowActivityResult struct {
	SchemaVersion string                 `json:"schema_version" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ActivityID    string                 `json:"activity_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	RunID         string                 `json:"run_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Status        ActionResultStatus     `json:"status" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Output        map[string]interface{} `json:"output,omitempty" storage:"encrypted" logging:"prohibited" temporal_history:"prohibited" api_exposure:"prohibited"`
	ErrorMessage  string                 `json:"error_message,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
}

// -----------------------------------------------------------------------------
// 12. ActionResult
// -----------------------------------------------------------------------------

type ActionResult struct {
	SchemaVersion       string                 `json:"schema_version" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ActionID            string                 `json:"action_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ActivityType        string                 `json:"activity_type" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Status              ActionResultStatus     `json:"status" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Output              map[string]interface{} `json:"output,omitempty" storage:"encrypted" logging:"prohibited" temporal_history:"prohibited" api_exposure:"prohibited"`
	ErrorMessage        string                 `json:"error_message,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ExecutionDurationMS int64                  `json:"execution_duration_ms" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	CompletedAt         time.Time              `json:"completed_at" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
}

// -----------------------------------------------------------------------------
// 13. ValidationIssue
// -----------------------------------------------------------------------------

type ValidationIssue struct {
	SchemaVersion string             `json:"schema_version" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	IssueID       string             `json:"issue_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	NodeID        string             `json:"node_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Severity      ValidationSeverity `json:"severity" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Code          string             `json:"code" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Message       string             `json:"message" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	FieldPath     string             `json:"field_path" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
}

// -----------------------------------------------------------------------------
// 14. RunProjection
// -----------------------------------------------------------------------------

type RunProjection struct {
	SchemaVersion string                 `json:"schema_version" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	RunID         string                 `json:"run_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	TenantID      string                 `json:"tenant_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	WorkflowID    string                 `json:"workflow_id" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	ExecutionMode string                 `json:"execution_mode,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Status        RunStatus              `json:"status" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	CurrentNodes  []string               `json:"current_nodes" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	Variables     map[string]interface{} `json:"variables,omitempty" storage:"encrypted" logging:"prohibited" temporal_history:"prohibited" api_exposure:"prohibited"`
	StartedAt     time.Time              `json:"started_at" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	UpdatedAt     time.Time              `json:"updated_at" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty" storage:"allowed" logging:"allowed" temporal_history:"allowed" api_exposure:"public"`
}

// -----------------------------------------------------------------------------
// Helper Hashing Function
// -----------------------------------------------------------------------------

func calculateDeterministicHash(v interface{}) (string, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal struct for SHA-256 calculation: %w", err)
	}
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:]), nil
}
