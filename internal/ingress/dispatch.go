package ingress

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

// Delivery Ledger Result Constants
const (
	StatusAccepted      = "accepted"
	StatusAlreadyExists = "already-exists"
	StatusRetryable     = "retryable"
	StatusPermanent     = "permanent"
	StatusQuarantined   = "quarantined"
)

// Fencing & Lane Error Definitions
var (
	ErrNoLease               = errors.New("partition lease not found")
	ErrFencedStaleOwner      = errors.New("partition owned by another worker (stale owner)")
	ErrFencedStaleGeneration = errors.New("partition generation is stale (fenced)")
	ErrLeaseExpired          = errors.New("partition lease expired")
	ErrInvalidTargetLane     = errors.New("invalid or cross-lane target dispatch prohibited")
	ErrManifestNotFound      = errors.New("target manifest not found")
)

// TargetLane represents strict queue & topic segregation.
type TargetLane string

const (
	LaneProduction    TargetLane = "production"
	LaneTransactional TargetLane = "transactional"
	LanePromotional   TargetLane = "promotional"
	LaneTest          TargetLane = "test"
)

// TaskQueueForLane returns the dedicated task queue name for a given target lane.
func TaskQueueForLane(lane TargetLane) string {
	switch lane {
	case LaneTransactional:
		return "JOURNEY_TASK_QUEUE_TRANSACTIONAL"
	case LanePromotional:
		return "JOURNEY_TASK_QUEUE_PROMOTIONAL"
	case LaneTest:
		return "JOURNEY_TASK_QUEUE_TEST"
	case LaneProduction:
		fallthrough
	default:
		return "JOURNEY_TASK_QUEUE_PRODUCTION"
	}
}

// TopicForLane returns the dedicated topic name for a given target lane.
func TopicForLane(lane TargetLane) string {
	switch lane {
	case LaneTransactional:
		return "targets-transactional"
	case LanePromotional:
		return "targets-promotional"
	case LaneTest:
		return "targets-test"
	case LaneProduction:
		fallthrough
	default:
		return "targets-production"
	}
}

// ParseTargetLane converts a string representation to TargetLane.
func ParseTargetLane(raw string) (TargetLane, error) {
	switch raw {
	case "production", "prod":
		return LaneProduction, nil
	case "transactional", "txn":
		return LaneTransactional, nil
	case "promotional", "promo":
		return LanePromotional, nil
	case "test", "sandbox":
		return LaneTest, nil
	case "":
		return LaneProduction, nil
	default:
		return "", fmt.Errorf("%w: unknown lane %q", ErrInvalidTargetLane, raw)
	}
}

// OffsetWatermarkTracker tracks processed message offsets contiguously per topic-partition.
// It guarantees that the watermark never advances past uncompleted lower offsets.
type OffsetWatermarkTracker struct {
	mu         sync.RWMutex
	watermarks map[string]map[int32]int64
	completed  map[string]map[int32]map[int64]bool
}

// NewOffsetWatermarkTracker creates a new OffsetWatermarkTracker.
func NewOffsetWatermarkTracker() *OffsetWatermarkTracker {
	return &OffsetWatermarkTracker{
		watermarks: make(map[string]map[int32]int64),
		completed:  make(map[string]map[int32]map[int64]bool),
	}
}

// MarkProcessed registers an offset as completed for (topic, partition) and recalculates contiguous watermark.
func (t *OffsetWatermarkTracker) MarkProcessed(topic string, partition int32, offset int64) int64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.watermarks[topic]; !ok {
		t.watermarks[topic] = make(map[int32]int64)
		t.completed[topic] = make(map[int32]map[int64]bool)
	}
	if _, ok := t.watermarks[topic][partition]; !ok {
		t.watermarks[topic][partition] = offset - 1
		t.completed[topic][partition] = make(map[int64]bool)
	}

	t.completed[topic][partition][offset] = true

	// Advance watermark contiguously from current watermark
	currWM := t.watermarks[topic][partition]
	next := currWM + 1
	for t.completed[topic][partition][next] {
		delete(t.completed[topic][partition], next)
		currWM = next
		next++
	}

	t.watermarks[topic][partition] = currWM
	return currWM
}

// Watermark returns the highest contiguous offset processed for (topic, partition).
func (t *OffsetWatermarkTracker) Watermark(topic string, partition int32) int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if pMap, ok := t.watermarks[topic]; ok {
		if wm, found := pMap[partition]; found {
			return wm
		}
	}
	return -1
}

// PartitionLease holds partition ownership details.
type PartitionLease struct {
	Topic          string
	Partition      int32
	OwnerID        string
	Generation     int64
	LeaseExpiresAt time.Time
}

// PartitionFencer manages partition leases and generation fencing.
type PartitionFencer struct {
	mu     sync.RWMutex
	leases map[string]*PartitionLease // key: "topic:partition"
}

// NewPartitionFencer creates a new PartitionFencer.
func NewPartitionFencer() *PartitionFencer {
	return &PartitionFencer{
		leases: make(map[string]*PartitionLease),
	}
}

// AcquireLease claims or renews ownership of a partition with generation & TTL.
func (f *PartitionFencer) AcquireLease(topic string, partition int32, ownerID string, generation int64, ttl time.Duration) (*PartitionLease, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := fmt.Sprintf("%s:%d", topic, partition)
	existing, ok := f.leases[key]
	if ok {
		if generation < existing.Generation {
			return nil, ErrFencedStaleGeneration
		}
		if generation == existing.Generation && existing.OwnerID != ownerID && time.Now().Before(existing.LeaseExpiresAt) {
			return nil, ErrFencedStaleOwner
		}
	}

	lease := &PartitionLease{
		Topic:          topic,
		Partition:      partition,
		OwnerID:        ownerID,
		Generation:     generation,
		LeaseExpiresAt: time.Now().Add(ttl),
	}
	f.leases[key] = lease
	return lease, nil
}

// ValidateOwnership verifies that the specified ownerID & generation hold a valid active lease.
func (f *PartitionFencer) ValidateOwnership(topic string, partition int32, ownerID string, generation int64) error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	key := fmt.Sprintf("%s:%d", topic, partition)
	lease, ok := f.leases[key]
	if !ok {
		return ErrNoLease
	}

	if generation < lease.Generation {
		return ErrFencedStaleGeneration
	}
	if lease.OwnerID != ownerID {
		return ErrFencedStaleOwner
	}
	if time.Now().After(lease.LeaseExpiresAt) {
		return ErrLeaseExpired
	}

	return nil
}

// WorkflowDispatchOptions configuration for launching Temporal workflows.
type WorkflowDispatchOptions struct {
	ID         string
	TaskQueue  string
	TargetLane TargetLane
}

// TemporalClient defines the interface for dispatching to Temporal.
type TemporalClient interface {
	StartWorkflow(ctx context.Context, opts WorkflowDispatchOptions, workflowType string, args ...interface{}) (string, error)
	SignalWorkflow(ctx context.Context, workflowID string, runID string, signalName string, arg interface{}) error
}

// SDKTemporalClient implements TemporalClient using the official Temporal Go SDK.
type SDKTemporalClient struct {
	client client.Client
}

// NewSDKTemporalClient creates a new SDKTemporalClient wrapping a Temporal SDK client.
func NewSDKTemporalClient(c client.Client) *SDKTemporalClient {
	return &SDKTemporalClient{client: c}
}

func (s *SDKTemporalClient) StartWorkflow(ctx context.Context, opts WorkflowDispatchOptions, workflowType string, args ...interface{}) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("temporal SDK client is nil")
	}
	taskQueue := opts.TaskQueue
	if taskQueue == "" {
		taskQueue = "journey-engine-task-queue"
	}
	wfOpts := client.StartWorkflowOptions{
		ID:        opts.ID,
		TaskQueue: taskQueue,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1,
		},
	}
	run, err := s.client.ExecuteWorkflow(ctx, wfOpts, workflowType, args...)
	if err != nil {
		return "", err
	}
	return run.GetRunID(), nil
}

func (s *SDKTemporalClient) SignalWorkflow(ctx context.Context, workflowID string, runID string, signalName string, arg interface{}) error {
	if s.client == nil {
		return fmt.Errorf("temporal SDK client is nil")
	}
	return s.client.SignalWorkflow(ctx, workflowID, runID, signalName, arg)
}

// MockTemporalClient implements TemporalClient for testing and offline fallback.
type MockTemporalClient struct {
	mu           sync.Mutex
	Workflows    map[string]WorkflowDispatchOptions
	Signals      map[string][]interface{}
	StartErr     error
	SignalErr    error
	AlreadyExist map[string]bool
}

// NewMockTemporalClient creates a new MockTemporalClient.
func NewMockTemporalClient() *MockTemporalClient {
	return &MockTemporalClient{
		Workflows:    make(map[string]WorkflowDispatchOptions),
		Signals:      make(map[string][]interface{}),
		AlreadyExist: make(map[string]bool),
	}
}

func (m *MockTemporalClient) StartWorkflow(ctx context.Context, opts WorkflowDispatchOptions, workflowType string, args ...interface{}) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.StartErr != nil {
		return "", m.StartErr
	}

	if m.AlreadyExist[opts.ID] {
		return opts.ID, fmt.Errorf("workflow execution already started: %s", opts.ID)
	}

	if _, exists := m.Workflows[opts.ID]; exists {
		m.AlreadyExist[opts.ID] = true
		return opts.ID, fmt.Errorf("workflow execution already started: %s", opts.ID)
	}

	m.Workflows[opts.ID] = opts
	return fmt.Sprintf("run-%s", opts.ID), nil
}

func (m *MockTemporalClient) SignalWorkflow(ctx context.Context, workflowID string, runID string, signalName string, arg interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.SignalErr != nil {
		return m.SignalErr
	}

	m.Signals[workflowID] = append(m.Signals[workflowID], arg)
	return nil
}

// DispatchRequest parameters for target manifest execution.
type DispatchRequest struct {
	TenantID   string
	ManifestID string
	OwnerID    string
	Generation int64
	Topic      string
	Partition  int32
	Offset     *int64
	TargetLane TargetLane
}

// DispatchResult summary of target manifest execution.
type DispatchResult struct {
	ManifestID    string
	TotalTargets  int
	Accepted      int
	AlreadyExists int
	Retryable     int
	Permanent     int
	Quarantined   int
	Watermark     int64
	Lane          TargetLane
	TaskQueue     string
	LedgerResults map[string]string // subjectID -> status
}

// ManifestQuerySpec parsed structure from target_manifests.query_spec.
type ManifestQuerySpec struct {
	EventID          string                 `json:"event_id"`
	EventType        string                 `json:"event_type"`
	JourneyVersionID string                 `json:"journey_version_id"`
	SubjectRef       string                 `json:"subject_ref"`
	Targets          []string               `json:"targets"`
	TargetLane       string                 `json:"target_lane"`
	Payload          map[string]interface{} `json:"payload"`
}

// TargetDispatcher coordinates manifest leasing, fencing, Temporal execution, and delivery ledger recording.
type TargetDispatcher struct {
	repo           postgres.Repository
	temporalClient TemporalClient
	fencer         *PartitionFencer
	tracker        *OffsetWatermarkTracker
}

// NewTargetDispatcher initializes a TargetDispatcher.
func NewTargetDispatcher(repo postgres.Repository, tc TemporalClient, fencer *PartitionFencer, tracker *OffsetWatermarkTracker) *TargetDispatcher {
	if fencer == nil {
		fencer = NewPartitionFencer()
	}
	if tracker == nil {
		tracker = NewOffsetWatermarkTracker()
	}
	if tc == nil {
		tc = NewMockTemporalClient()
	}
	return &TargetDispatcher{
		repo:           repo,
		temporalClient: tc,
		fencer:         fencer,
		tracker:        tracker,
	}
}

// Fencer returns the PartitionFencer.
func (d *TargetDispatcher) Fencer() *PartitionFencer {
	return d.fencer
}

// Tracker returns the OffsetWatermarkTracker.
func (d *TargetDispatcher) Tracker() *OffsetWatermarkTracker {
	return d.tracker
}

// LeaseFrozenTarget fetches target manifest after verifying partition ownership & fencing.
func (d *TargetDispatcher) LeaseFrozenTarget(ctx context.Context, req DispatchRequest) (*postgres.TargetManifest, error) {
	if req.Topic != "" && req.OwnerID != "" {
		if err := d.fencer.ValidateOwnership(req.Topic, req.Partition, req.OwnerID, req.Generation); err != nil {
			return nil, err
		}
	}

	tm, err := d.repo.GetTargetManifest(ctx, req.TenantID, req.ManifestID)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			return nil, ErrManifestNotFound
		}
		return nil, err
	}

	return tm, nil
}

// DispatchManifest executes idempotent Temporal workflows/signals for all frozen targets in a manifest.
func (d *TargetDispatcher) DispatchManifest(ctx context.Context, req DispatchRequest) (*DispatchResult, error) {
	tm, err := d.LeaseFrozenTarget(ctx, req)
	if err != nil {
		return nil, err
	}

	var spec ManifestQuerySpec
	if len(tm.QuerySpec) > 0 {
		if unmarshalErr := json.Unmarshal(tm.QuerySpec, &spec); unmarshalErr != nil {
			return nil, fmt.Errorf("invalid query spec in manifest %s: %w", req.ManifestID, unmarshalErr)
		}
	}

	// Validate & resolve target lane
	manifestLaneStr := spec.TargetLane
	if manifestLaneStr == "" {
		manifestLaneStr = "production"
	}
	manifestLane, laneErr := ParseTargetLane(manifestLaneStr)
	if laneErr != nil {
		return nil, laneErr
	}

	selectedLane := manifestLane
	if req.TargetLane != "" {
		if req.TargetLane != manifestLane {
			return nil, fmt.Errorf("%w: request lane %q does not match manifest lane %q", ErrInvalidTargetLane, req.TargetLane, manifestLane)
		}
		selectedLane = req.TargetLane
	}

	taskQueue := TaskQueueForLane(selectedLane)

	targets := spec.Targets
	if len(targets) == 0 && spec.SubjectRef != "" {
		targets = []string{spec.SubjectRef}
	}

	res := &DispatchResult{
		ManifestID:    req.ManifestID,
		TotalTargets:  len(targets),
		Lane:          selectedLane,
		TaskQueue:     taskQueue,
		LedgerResults: make(map[string]string),
	}

	journeyVersionID := spec.JourneyVersionID
	if journeyVersionID == "" {
		journeyVersionID = "jv-default"
	}

	for _, subjectID := range targets {
		status := d.dispatchSubject(ctx, req.TenantID, req.ManifestID, journeyVersionID, subjectID, selectedLane, taskQueue, spec)
		res.LedgerResults[subjectID] = status

		switch status {
		case StatusAccepted:
			res.Accepted++
		case StatusAlreadyExists:
			res.AlreadyExists++
		case StatusRetryable:
			res.Retryable++
		case StatusPermanent:
			res.Permanent++
		case StatusQuarantined:
			res.Quarantined++
		}
	}

	// Record offset watermark if offset is present
	if req.Topic != "" && req.Offset != nil {
		res.Watermark = d.tracker.MarkProcessed(req.Topic, req.Partition, *req.Offset)
	} else if req.Topic != "" {
		res.Watermark = d.tracker.Watermark(req.Topic, req.Partition)
	}

	return res, nil
}

func (d *TargetDispatcher) dispatchSubject(ctx context.Context, tenantID, manifestID, journeyVersionID, subjectID string, lane TargetLane, taskQueue string, spec ManifestQuerySpec) string {
	// 1. Check for tombstoned / quarantined subject
	tombstone, _ := d.repo.GetTombstoneByEntity(ctx, tenantID, "subject", subjectID)
	if tombstone == nil {
		tombstone, _ = d.repo.GetTombstoneByEntity(ctx, tenantID, "user", subjectID)
	}
	if tombstone != nil {
		d.recordLedger(ctx, tenantID, manifestID, journeyVersionID, subjectID, StatusQuarantined)
		return StatusQuarantined
	}

	// 2. Check for duplicate dispatch in DB (idempotency key based on dispatch boundary)
	dispatchID := fmt.Sprintf("disp-%s-%s", manifestID, subjectID)
	existingLedger, getErr := d.repo.GetDispatchLedger(ctx, tenantID, dispatchID)
	if getErr == nil && existingLedger != nil {
		return StatusAlreadyExists
	}

	// 3. Prepare Temporal workflow options & execution
	workflowID := fmt.Sprintf("wf-%s-%s-%s", tenantID, journeyVersionID, subjectID)
	opts := WorkflowDispatchOptions{
		ID:         workflowID,
		TaskQueue:  taskQueue,
		TargetLane: lane,
	}

	// Input parameters for Temporal workflow
	runID := fmt.Sprintf("run-%s-%s", journeyVersionID, subjectID)
	wfInput := map[string]interface{}{
		"schema_version":     domain.DefaultSchemaVersion,
		"workflow_id":        workflowID,
		"run_id":             runID,
		"tenant_id":          tenantID,
		"journey_version_id": journeyVersionID,
		"subject_id":         subjectID,
		"manifest_id":        manifestID,
		"trigger_event_id":   spec.EventID,
		"event_id":           spec.EventID,
		"event_type":         spec.EventType,
		"content_hash":       journeyVersionID,
		"execution_mode":     "production",
		"lane":               string(lane),
		"input_payload":      spec.Payload,
		"payload":            spec.Payload,
	}

	_, startErr := d.temporalClient.StartWorkflow(ctx, opts, "CompiledJourneyWorkflow", wfInput)
	var status string
	if startErr == nil {
		status = StatusAccepted
	} else {
		errStr := startErr.Error()
		if isAlreadyExistsError(errStr) {
			// Signal workflow idempotently if already running
			_ = d.temporalClient.SignalWorkflow(ctx, workflowID, "", "journey.signal.event", wfInput)
			status = StatusAlreadyExists
		} else if isRetryableError(errStr) {
			status = StatusRetryable
		} else {
			status = StatusPermanent
		}
	}

	// 4. Record result into dispatch_ledger
	d.recordLedger(ctx, tenantID, manifestID, journeyVersionID, subjectID, status)
	return status
}

func (d *TargetDispatcher) recordLedger(ctx context.Context, tenantID, manifestID, journeyVersionID, subjectID, status string) {
	dispatchID := fmt.Sprintf("disp-%s-%s", manifestID, subjectID)
	dl := &postgres.DispatchLedger{
		TenantID:         tenantID,
		DispatchID:       dispatchID,
		ManifestID:       manifestID,
		JourneyVersionID: journeyVersionID,
		SubjectID:        subjectID,
		Status:           status,
		DispatchedAt:     time.Now().UTC(),
		CreatedAt:        time.Now().UTC(),
	}
	_, _ = d.repo.CreateDispatchLedger(ctx, dl)
}

func isAlreadyExistsError(errStr string) bool {
	return errors.Is(errors.New(errStr), postgres.ErrAlreadyExists) ||
		errors.Is(errors.New(errStr), postgres.ErrConflict) ||
		containsIgnoreCase(errStr, "already started") ||
		containsIgnoreCase(errStr, "already exists") ||
		containsIgnoreCase(errStr, "duplicate")
}

func isRetryableError(errStr string) bool {
	return containsIgnoreCase(errStr, "timeout") ||
		containsIgnoreCase(errStr, "connection refused") ||
		containsIgnoreCase(errStr, "unavailable") ||
		containsIgnoreCase(errStr, "retryable") ||
		containsIgnoreCase(errStr, "temporary")
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) > 0 && hasSubstr(s, substr))
}

func hasSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			c1 := s[i+j]
			c2 := substr[j]
			if c1 >= 'A' && c1 <= 'Z' {
				c1 += 'a' - 'A'
			}
			if c2 >= 'A' && c2 <= 'Z' {
				c2 += 'a' - 'A'
			}
			if c1 != c2 {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
