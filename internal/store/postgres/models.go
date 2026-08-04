package postgres

import (
	"time"
)

type Catalog struct {
	TenantID         string    `json:"tenant_id"`
	RecordID         string    `json:"record_id"`
	Name             string    `json:"name"`
	ComponentType    string    `json:"component_type"`
	Version          string    `json:"version"`
	Description      string    `json:"description"`
	SchemaDefinition []byte    `json:"schema_definition"`
	ContentHash      string    `json:"content_hash"`
	Tags             []string  `json:"tags"`
	IsDeprecated     bool      `json:"is_deprecated"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type JourneyDraft struct {
	TenantID    string    `json:"tenant_id"`
	DraftID     string    `json:"draft_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     int32     `json:"version"`
	Nodes       []byte    `json:"nodes"`
	Edges       []byte    `json:"edges"`
	ContentHash string    `json:"content_hash"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type JourneyVersion struct {
	TenantID    string    `json:"tenant_id"`
	VersionID   string    `json:"version_id"`
	DraftID     string    `json:"draft_id"`
	Version     int32     `json:"version"`
	EntryNodeID string    `json:"entry_node_id"`
	Nodes       []byte    `json:"nodes"`
	Edges       []byte    `json:"edges"`
	ContentHash string    `json:"content_hash"`
	CompiledAt  time.Time `json:"compiled_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type IdempotencyKey struct {
	TenantID        string    `json:"tenant_id"`
	Key             string    `json:"key"`
	Scope           string    `json:"scope"`
	Status          string    `json:"status"`
	ResponsePayload []byte    `json:"response_payload"`
	ExpiresAt       time.Time `json:"expires_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type KafkaInbox struct {
	TenantID    string     `json:"tenant_id"`
	MessageID   string     `json:"message_id"`
	EventID     string     `json:"event_id"`
	Topic       string     `json:"topic"`
	Partition   int32      `json:"partition"`
	OffsetVal   int64      `json:"offset_val"`
	Payload     []byte     `json:"payload"`
	Status      string     `json:"status"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type TargetManifest struct {
	TenantID    string    `json:"tenant_id"`
	ManifestID  string    `json:"manifest_id"`
	Name        string    `json:"name"`
	QuerySpec   []byte    `json:"query_spec"`
	TotalCount  int64     `json:"total_count"`
	ContentHash string    `json:"content_hash"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type DispatchLedger struct {
	TenantID         string    `json:"tenant_id"`
	DispatchID       string    `json:"dispatch_id"`
	ManifestID       string    `json:"manifest_id"`
	JourneyVersionID string    `json:"journey_version_id"`
	SubjectID        string    `json:"subject_id"`
	Status           string    `json:"status"`
	DispatchedAt     time.Time `json:"dispatched_at"`
	CreatedAt        time.Time `json:"created_at"`
}

type Enrollment struct {
	TenantID         string     `json:"tenant_id"`
	EnrollmentID     string     `json:"enrollment_id"`
	JourneyVersionID string     `json:"journey_version_id"`
	SubjectID        string     `json:"subject_id"`
	Status           string     `json:"status"`
	CurrentNodeID    string     `json:"current_node_id"`
	StateData        []byte     `json:"state_data"`
	EnrolledAt       time.Time  `json:"enrolled_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

type Subscription struct {
	TenantID       string    `json:"tenant_id"`
	SubscriptionID string    `json:"subscription_id"`
	EnrollmentID   string    `json:"enrollment_id"`
	EventType      string    `json:"event_type"`
	ConditionExpr  string    `json:"condition_expr"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ActionLedger struct {
	TenantID            string     `json:"tenant_id"`
	ActionID            string     `json:"action_id"`
	EnrollmentID        string     `json:"enrollment_id"`
	NodeID              string     `json:"node_id"`
	ActivityType        string     `json:"activity_type"`
	Status              string     `json:"status"`
	Output              []byte     `json:"output"`
	ErrorMessage        string     `json:"error_message"`
	ExecutionDurationMS int64      `json:"execution_duration_ms"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

type ExperimentDefinition struct {
	TenantID       string    `json:"tenant_id"`
	ExperimentID   string    `json:"experiment_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Status         string    `json:"status"`
	Variants       []byte    `json:"variants"`
	TargetAudience string    `json:"target_audience"`
	ContentHash    string    `json:"content_hash"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Assignment struct {
	TenantID          string    `json:"tenant_id"`
	AssignmentID      string    `json:"assignment_id"`
	ExperimentID      string    `json:"experiment_id"`
	SubjectID         string    `json:"subject_id"`
	VariantID         string    `json:"variant_id"`
	WeightBasisPoints int32     `json:"weight_basis_points"`
	AssignedAt        time.Time `json:"assigned_at"`
	CreatedAt         time.Time `json:"created_at"`
}

type Exposure struct {
	TenantID          string    `json:"tenant_id"`
	ExposureID        string    `json:"exposure_id"`
	ExperimentID      string    `json:"experiment_id"`
	AssignmentID      string    `json:"assignment_id"`
	SubjectID         string    `json:"subject_id"`
	VariantID         string    `json:"variant_id"`
	WeightBasisPoints int32     `json:"weight_basis_points"`
	Context           []byte    `json:"context"`
	ExposedAt         time.Time `json:"exposed_at"`
	CreatedAt         time.Time `json:"created_at"`
}

type Outbox struct {
	TenantID      string     `json:"tenant_id"`
	ID            string     `json:"id"`
	AggregateType string     `json:"aggregate_type"`
	AggregateID   string     `json:"aggregate_id"`
	EventType     string     `json:"event_type"`
	Payload       []byte     `json:"payload"`
	Headers       []byte     `json:"headers"`
	Status        string     `json:"status"`
	RetryCount    int32      `json:"retry_count"`
	CreatedAt     time.Time  `json:"created_at"`
	ProcessedAt   *time.Time `json:"processed_at,omitempty"`
}

type StaticList struct {
	TenantID           string    `json:"tenant_id"`
	ListID             string    `json:"list_id"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	ItemCount          int32     `json:"item_count"`
	DataClassification string    `json:"data_classification"`
	Items              []byte    `json:"items"`
	ContentHash        string    `json:"content_hash"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type TestRun struct {
	TenantID         string    `json:"tenant_id"`
	TestRunID        string    `json:"test_run_id"`
	DraftID          string    `json:"draft_id"`
	IRID             string    `json:"ir_id"`
	Status           string    `json:"status"`
	MockInputs       []byte    `json:"mock_inputs"`
	ExpectedOutcomes []byte    `json:"expected_outcomes"`
	ActualOutcomes   []byte    `json:"actual_outcomes"`
	ExecutionTimeMS  int64     `json:"execution_time_ms"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type LifecycleEvent struct {
	TenantID   string    `json:"tenant_id"`
	EventID    string    `json:"event_id"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	EventName  string    `json:"event_name"`
	Payload    []byte    `json:"payload"`
	CreatedAt  time.Time `json:"created_at"`
}

type Tombstone struct {
	TenantID    string    `json:"tenant_id"`
	TombstoneID string    `json:"tombstone_id"`
	EntityType  string    `json:"entity_type"`
	EntityID    string    `json:"entity_id"`
	DeletedBy   string    `json:"deleted_by"`
	Reason      string    `json:"reason"`
	DeletedAt   time.Time `json:"deleted_at"`
}
