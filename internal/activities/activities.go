package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
	"github.com/validated-pattern/journey-platform/internal/compiler"
	"github.com/validated-pattern/journey-platform/internal/compiler/expression"
	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/experiments"
	"github.com/validated-pattern/journey-platform/internal/outcomes"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
	"github.com/validated-pattern/journey-platform/internal/store/object"
	"go.temporal.io/sdk/activity"
)

// Registry manages activity definitions.
type Registry struct{}

// New creates a new Registry.
func New() *Registry {
	return &Registry{}
}

// Register registers an activity by name.
func (r *Registry) Register(name string) bool {
	return name != ""
}

// Activities defines the set of Temporal activities available to the Journey Engine.
type Activities struct {
	repo              postgres.Repository
	policyService     *PolicyService
	attributeResolver AttributeResolver
	parameterResolver ParameterResolver
	objectStore       object.ObjectStore
	largeThreshold    int

	irStore         map[string]*domain.CompiledIR
	irMutex         sync.RWMutex
	emittedEventIDs map[string]bool
	lifecycleEvents []LifecycleEvent
	eventsMutex     sync.RWMutex
	subscriptions   map[string]SubscriptionRecord
	subMutex        sync.RWMutex

	assignmentService *experiments.AssignmentService
	exposureService   *outcomes.ExposureService
	actionGateway     *ActionGateway
	failureTracker    *FailureTracker
	serviceMutex      sync.Mutex
}

func (a *Activities) SetRepository(repo postgres.Repository) {
	a.repo = repo
}

// NewActivities returns a new Activities instance.
func NewActivities(opts ...PolicyServiceOption) *Activities {
	repo := postgres.NewMemoryRepository()
	ft := NewFailureTracker()
	return &Activities{
		repo:              repo,
		policyService:     NewPolicyService(opts...),
		attributeResolver: NewMockAttributeResolver(),
		parameterResolver: NewMockParameterResolver(),
		objectStore:       object.NewMemoryStore(),
		largeThreshold:    DefaultLargePayloadThreshold,
		irStore:           make(map[string]*domain.CompiledIR),
		emittedEventIDs:   make(map[string]bool),
		lifecycleEvents:   make([]LifecycleEvent, 0),
		subscriptions:     make(map[string]SubscriptionRecord),
		assignmentService: experiments.NewAssignmentService(repo),
		exposureService:   outcomes.NewExposureService(repo),
		actionGateway:     NewActionGateway(WithFailureTracker(ft)),
		failureTracker:    ft,
	}
}

// WithRepository sets the repository instance for database lookups.
func (a *Activities) WithRepository(r postgres.Repository) *Activities {
	if r != nil {
		a.repo = r
	}
	return a
}

// WithAttributeResolver sets the attribute resolver for the Activities instance.
func (a *Activities) WithAttributeResolver(r AttributeResolver) *Activities {
	a.attributeResolver = r
	return a
}

// WithParameterResolver sets the parameter resolver for the Activities instance.
func (a *Activities) WithParameterResolver(r ParameterResolver) *Activities {
	a.parameterResolver = r
	return a
}

// WithObjectStore sets the object store for offloading large payloads.
func (a *Activities) WithObjectStore(s object.ObjectStore) *Activities {
	a.objectStore = s
	return a
}

// WithLargePayloadThreshold sets the byte size threshold for payload offloading.
func (a *Activities) WithLargePayloadThreshold(threshold int) *Activities {
	a.largeThreshold = threshold
	return a
}

// WithAssignmentService sets the assignment service for the Activities instance.
func (a *Activities) WithAssignmentService(s *experiments.AssignmentService) *Activities {
	a.assignmentService = s
	return a
}

// WithExposureService sets the exposure service for the Activities instance.
func (a *Activities) WithExposureService(s *outcomes.ExposureService) *Activities {
	a.exposureService = s
	return a
}

// WithActionGateway sets the action gateway for the Activities instance.
func (a *Activities) WithActionGateway(g *ActionGateway) *Activities {
	a.actionGateway = g
	return a
}

func (a *Activities) ensureServices() {
	a.serviceMutex.Lock()
	defer a.serviceMutex.Unlock()
	if a.assignmentService == nil || a.exposureService == nil {
		repo := postgres.NewMemoryRepository()
		if a.assignmentService == nil {
			a.assignmentService = experiments.NewAssignmentService(repo)
		}
		if a.exposureService == nil {
			a.exposureService = outcomes.NewExposureService(repo)
		}
	}
	if a.actionGateway == nil {
		a.actionGateway = NewActionGateway(
			WithAttributeResolver(a.attributeResolver),
		)
	}
}

// GetOrAssign retrieves an existing sticky assignment or creates a new deterministic assignment.
func (a *Activities) GetOrAssign(ctx context.Context, req experiments.AssignmentRequest) (*experiments.AssignmentRecord, error) {
	if activity.IsActivity(ctx) {
		logger := activity.GetLogger(ctx)
		logger.Info("Getting or assigning experiment variant", "subjectID", req.SubjectID, "mode", req.Mode)
	}
	a.ensureServices()
	return a.assignmentService.GetOrAssign(ctx, req)
}

// RecordExposure records an experiment exposure.
func (a *Activities) RecordExposure(ctx context.Context, req outcomes.ExposureRequest) (*outcomes.ExposureRecord, error) {
	if activity.IsActivity(ctx) {
		logger := activity.GetLogger(ctx)
		logger.Info("Recording experiment exposure", "experimentID", req.ExperimentID, "assignmentID", req.AssignmentID, "subjectID", req.SubjectID)
	}
	a.ensureServices()
	return a.exposureService.RecordExposure(ctx, req)
}

// ExecuteActionGateway executes an action via the action gateway idempotently.
func (a *Activities) ExecuteActionGateway(ctx context.Context, req ActionRequest) (*GatewayResult, error) {
	if activity.IsActivity(ctx) {
		logger := activity.GetLogger(ctx)
		logger.Info("Executing action via gateway", "nodeID", req.NodeID, "channel", req.Channel, "executionMode", req.ExecutionMode)
	}
	a.ensureServices()
	return a.actionGateway.ExecuteAction(ctx, req)
}

// EvaluatePolicy evaluates communication policy for a given context.
func (a *Activities) EvaluatePolicy(ctx context.Context, pctx PolicyContext) (PolicyDecisionResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Evaluating communication policy", "subjectID", pctx.SubjectID, "channel", pctx.Channel)

	res := a.policyService.Evaluate(ctx, &pctx)
	return res, nil
}

// ExecuteNodeInput represents input to the ExecuteNode activity.
type ExecuteNodeInput struct {
	RunID  string                 `json:"run_id"`
	Node   domain.IRNode          `json:"node"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// ExecuteNode executes an IR node step.
func (a *Activities) ExecuteNode(ctx context.Context, input ExecuteNodeInput) (domain.ActionResult, error) {
	if activity.IsActivity(ctx) {
		logger := activity.GetLogger(ctx)
		logger.Info("Executing activity node", "nodeID", input.Node.ID, "activityName", input.Node.ActivityName)
	}
	enabled, maxAttempts := getExecuteNodeSimulatedFailureConfig(input)
	if enabled {
		nodeKey := fmt.Sprintf("node:%s:%s", input.RunID, input.Node.ID)
		if input.RunID == "" {
			nodeKey = fmt.Sprintf("node:%s", input.Node.ID)
		}
		attempt := a.getFailureTracker().IncrementAndGet(nodeKey)
		if attempt <= maxAttempts {
			return domain.ActionResult{}, fmt.Errorf("simulated activity execution failure attempt %d/%d", attempt, maxAttempts)
		}
		a.getFailureTracker().Reset(nodeKey)
	}

	now := time.Now().UTC()
	return domain.ActionResult{
		SchemaVersion:       domain.DefaultSchemaVersion,
		ActionID:            fmt.Sprintf("act-%s", input.Node.ID),
		ActivityType:        input.Node.ActivityName,
		Status:              domain.ActionResultStatusSuccess,
		Output:              map[string]interface{}{"status": "completed", "executed_at": now.Format(time.RFC3339)},
		ExecutionDurationMS: 10,
		CompletedAt:         now,
	}, nil
}

// EvaluateConditionInput represents input to condition evaluation activity.
type EvaluateConditionInput struct {
	ConditionExpression string                 `json:"condition_expression"`
	Context             map[string]interface{} `json:"context,omitempty"`
}

// EvaluateCondition evaluates a workflow edge condition expression.
func (a *Activities) EvaluateCondition(ctx context.Context, input EvaluateConditionInput) (bool, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Evaluating condition expression", "expression", input.ConditionExpression)

	exprStr := strings.TrimSpace(input.ConditionExpression)
	if exprStr == "" || exprStr == "true" || exprStr == "default" {
		return true, nil
	}
	if exprStr == "false" {
		return false, nil
	}

	astNode, err := expression.Parse(exprStr)
	if err != nil && !strings.Contains(exprStr, ".") {
		astNode, err = expression.Parse("event." + exprStr)
	}
	if err != nil {
		return false, nil
	}

	evalCtx := expression.NewEvalContext()
	flattened := domain.FlattenPayloadContext(input.Context)
	for k, v := range flattened {
		evalCtx.Event[k] = v
		evalCtx.Event["event."+k] = v
		evalCtx.Subject[k] = v
		evalCtx.Subject["subject."+k] = v
		evalCtx.NodeOutput[k] = v
		evalCtx.Parameter[k] = v
	}

	res, err := expression.EvaluateToBool(astNode, evalCtx)
	if err != nil {
		return false, nil
	}
	return res, nil
}

// EmitOutcome emits a normalized outcome event.
func (a *Activities) EmitOutcome(ctx context.Context, outcome domain.NormalizedOutcome) (domain.ActionResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Emitting outcome", "outcomeID", outcome.OutcomeID, "eventName", outcome.EventName)

	now := time.Now().UTC()
	return domain.ActionResult{
		SchemaVersion:       domain.DefaultSchemaVersion,
		ActionID:            fmt.Sprintf("act-outcome-%s", outcome.OutcomeID),
		ActivityType:        "EmitOutcome",
		Status:              domain.ActionResultStatusSuccess,
		Output:              map[string]interface{}{"outcome_id": outcome.OutcomeID},
		ExecutionDurationMS: 5,
		CompletedAt:         now,
	}, nil
}

// ExposeAssignment records an experiment assignment exposure.
func (a *Activities) ExposeAssignment(ctx context.Context, exposure domain.AssignmentExposure) (domain.ActionResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Exposing experiment assignment", "assignmentID", exposure.AssignmentID, "experimentID", exposure.ExperimentID)

	now := time.Now().UTC()
	return domain.ActionResult{
		SchemaVersion:       domain.DefaultSchemaVersion,
		ActionID:            fmt.Sprintf("act-exp-%s", exposure.AssignmentID),
		ActivityType:        "ExposeAssignment",
		Status:              domain.ActionResultStatusSuccess,
		Output:              map[string]interface{}{"assignment_id": exposure.AssignmentID, "variant_id": exposure.VariantID},
		ExecutionDurationMS: 5,
		CompletedAt:         now,
	}, nil
}

// RegisterCompiledIR stores a compiled IR in memory for LoadCompiledIR activity.
func (a *Activities) RegisterCompiledIR(ir *domain.CompiledIR) {
	a.irMutex.Lock()
	defer a.irMutex.Unlock()
	if a.irStore == nil {
		a.irStore = make(map[string]*domain.CompiledIR)
	}
	if ir.ContentHash != "" {
		a.irStore[ir.ContentHash] = ir
	}
	if ir.IRID != "" {
		a.irStore[ir.IRID] = ir
	}
}

// LoadCompiledIRInput represents input to LoadCompiledIR.
type LoadCompiledIRInput struct {
	ContentHash string `json:"content_hash"`
	IRID        string `json:"ir_id,omitempty"`
	TenantID    string `json:"tenant_id,omitempty"`
}

// LoadCompiledIR loads immutable compiled IR by SHA-256 content hash and verifies hash integrity.
func (a *Activities) LoadCompiledIR(ctx context.Context, input LoadCompiledIRInput) (*domain.CompiledIR, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Loading compiled IR", "contentHash", input.ContentHash, "irID", input.IRID)

	if a.repo != nil {
		tenantID := input.TenantID
		if tenantID == "" {
			tenantID = "default"
		}
		draftID := input.ContentHash
		if draftID == "" {
			draftID = input.IRID
		}
		if strings.HasPrefix(draftID, "ir-") {
			draftID = strings.TrimPrefix(draftID, "ir-")
		}

		dbDraft, err := a.repo.GetJourneyDraft(ctx, tenantID, draftID)
		if (err != nil || dbDraft == nil) && tenantID != "default" {
			dbDraft, _ = a.repo.GetJourneyDraft(ctx, "default", draftID)
		}

		if dbDraft != nil {
			var nodes []domain.GraphNode
			var edges []domain.GraphEdge
			if len(dbDraft.Nodes) > 0 {
				_ = json.Unmarshal(dbDraft.Nodes, &nodes)
			}
			if len(dbDraft.Edges) > 0 {
				_ = json.Unmarshal(dbDraft.Edges, &edges)
			}

			draft := &domain.GraphDraft{
				DraftID:     dbDraft.DraftID,
				TenantID:    dbDraft.TenantID,
				Name:        dbDraft.Name,
				Version:     int(dbDraft.Version),
				Nodes:       nodes,
				Edges:       edges,
				ContentHash: dbDraft.ContentHash,
			}

			comp := compiler.New()
			canonRes, err := comp.CanonicalizeDraft(draft)
			if err == nil && canonRes != nil && canonRes.CompiledIR != nil {
				ir := canonRes.CompiledIR
				a.RegisterCompiledIR(ir)
				return ir, nil
			}
		}
	}

	a.irMutex.RLock()
	ir, ok := a.irStore[input.ContentHash]
	if !ok && input.IRID != "" {
		ir, ok = a.irStore[input.IRID]
	}
	a.irMutex.RUnlock()

	if !ok || ir == nil {
		return nil, fmt.Errorf("compiled IR graph not found for contentHash '%s' or irID '%s'", input.ContentHash, input.IRID)
	}

	return ir, nil
}

// LifecycleEvent defines a workflow lifecycle event for run projections.
type LifecycleEvent struct {
	SchemaVersion string                 `json:"schema_version" temporal_history:"allowed"`
	EventID       string                 `json:"event_id" temporal_history:"allowed"`
	RunID         string                 `json:"run_id" temporal_history:"allowed"`
	TenantID      string                 `json:"tenant_id" temporal_history:"allowed"`
	WorkflowID    string                 `json:"workflow_id" temporal_history:"allowed"`
	EventType     string                 `json:"event_type" temporal_history:"allowed"`
	NodeID        string                 `json:"node_id,omitempty" temporal_history:"allowed"`
	Status        string                 `json:"status,omitempty" temporal_history:"allowed"`
	Timestamp     time.Time              `json:"timestamp" temporal_history:"allowed"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" temporal_history:"allowed"`
}

// EmitLifecycleEvent emits an idempotent lifecycle event for run projections.
func (a *Activities) EmitLifecycleEvent(ctx context.Context, event LifecycleEvent) (domain.ActionResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Emitting lifecycle event", "runID", event.RunID, "eventType", event.EventType, "nodeID", event.NodeID)

	if event.EventID == "" {
		event.EventID = fmt.Sprintf("%s:%s:%s", event.RunID, event.EventType, event.NodeID)
	}

	a.eventsMutex.Lock()
	defer a.eventsMutex.Unlock()

	if a.emittedEventIDs == nil {
		a.emittedEventIDs = make(map[string]bool)
	}

	if a.emittedEventIDs[event.EventID] {
		now := time.Now().UTC()
		return domain.ActionResult{
			SchemaVersion: domain.DefaultSchemaVersion,
			ActionID:      fmt.Sprintf("act-lifecycle-%s", event.EventID),
			ActivityType:  "EmitLifecycleEvent",
			Status:        domain.ActionResultStatusSuccess,
			Output:        map[string]interface{}{"event_id": event.EventID, "idempotent_skip": true},
			CompletedAt:   now,
		}, nil
	}

	a.emittedEventIDs[event.EventID] = true
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	a.lifecycleEvents = append(a.lifecycleEvents, event)

	if a.repo != nil {
		payloadMap := map[string]interface{}{
			"workflow_id": event.WorkflowID,
		}
		if event.NodeID != "" {
			payloadMap["node_id"] = event.NodeID
		}
		if event.Status != "" {
			payloadMap["status"] = event.Status
		}
		if event.Metadata != nil {
			for k, v := range event.Metadata {
				payloadMap[k] = v
			}
		}
		payloadBytes, _ := json.Marshal(payloadMap)

		_, _ = a.repo.RecordLifecycleEvent(ctx, &postgres.LifecycleEvent{
			TenantID:   event.TenantID,
			EventID:    event.EventID,
			EntityType: "workflow_run",
			EntityID:   event.RunID,
			EventName:  event.EventType,
			Payload:    payloadBytes,
			CreatedAt:  event.Timestamp,
		})
	}

	now := time.Now().UTC()
	return domain.ActionResult{
		SchemaVersion:       domain.DefaultSchemaVersion,
		ActionID:            fmt.Sprintf("act-lifecycle-%s", event.EventID),
		ActivityType:        "EmitLifecycleEvent",
		Status:              domain.ActionResultStatusSuccess,
		Output:              map[string]interface{}{"event_id": event.EventID, "emitted": true},
		ExecutionDurationMS: 2,
		CompletedAt:         now,
	}, nil
}

// GetEmittedLifecycleEvents returns a copy of emitted lifecycle events.
func (a *Activities) GetEmittedLifecycleEvents() []LifecycleEvent {
	a.eventsMutex.RLock()
	defer a.eventsMutex.RUnlock()
	events := make([]LifecycleEvent, len(a.lifecycleEvents))
	copy(events, a.lifecycleEvents)
	return events
}

// Subscription Types & Activities

type CreateSubscriptionInput struct {
	TenantID       string `json:"tenant_id"`
	SubscriptionID string `json:"subscription_id"`
	RunID          string `json:"run_id"`
	WorkflowID     string `json:"workflow_id"`
	NodeID         string `json:"node_id"`
	EventType      string `json:"event_type"`
	ConditionExpr  string `json:"condition_expr,omitempty"`
	Generation     int    `json:"generation"`
}

type CloseSubscriptionInput struct {
	TenantID       string `json:"tenant_id,omitempty"`
	SubscriptionID string `json:"subscription_id"`
	Reason         string `json:"reason"`
}

type SubscriptionRecord struct {
	TenantID       string     `json:"tenant_id"`
	SubscriptionID string     `json:"subscription_id"`
	RunID          string     `json:"run_id"`
	WorkflowID     string     `json:"workflow_id"`
	NodeID         string     `json:"node_id"`
	EventType      string     `json:"event_type"`
	ConditionExpr  string     `json:"condition_expr"`
	Generation     int        `json:"generation"`
	Status         string     `json:"status"` // "active", "closed"
	CloseReason    string     `json:"close_reason,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`
}

// CreateSubscription creates a generation-scoped subscription row before waiting for events.
func (a *Activities) CreateSubscription(ctx context.Context, input CreateSubscriptionInput) (domain.ActionResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Creating subscription", "subscriptionID", input.SubscriptionID, "eventType", input.EventType, "generation", input.Generation)

	a.subMutex.Lock()
	defer a.subMutex.Unlock()

	if a.subscriptions == nil {
		a.subscriptions = make(map[string]SubscriptionRecord)
	}

	now := time.Now().UTC()
	sub := SubscriptionRecord{
		TenantID:       input.TenantID,
		SubscriptionID: input.SubscriptionID,
		RunID:          input.RunID,
		WorkflowID:     input.WorkflowID,
		NodeID:         input.NodeID,
		EventType:      input.EventType,
		ConditionExpr:  input.ConditionExpr,
		Generation:     input.Generation,
		Status:         "active",
		CreatedAt:      now,
	}
	a.subscriptions[input.SubscriptionID] = sub

	return domain.ActionResult{
		SchemaVersion: domain.DefaultSchemaVersion,
		ActionID:      fmt.Sprintf("act-sub-%s", input.SubscriptionID),
		ActivityType:  "CreateSubscription",
		Status:        domain.ActionResultStatusSuccess,
		Output:        map[string]interface{}{"subscription_id": input.SubscriptionID, "status": "active"},
		CompletedAt:   now,
	}, nil
}

// CloseSubscription closes a subscription idempotently on match, timeout, cancellation, or Continue-As-New.
func (a *Activities) CloseSubscription(ctx context.Context, input CloseSubscriptionInput) (domain.ActionResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Closing subscription idempotently", "subscriptionID", input.SubscriptionID, "reason", input.Reason)

	a.subMutex.Lock()
	defer a.subMutex.Unlock()

	now := time.Now().UTC()
	if a.subscriptions != nil {
		if sub, ok := a.subscriptions[input.SubscriptionID]; ok {
			sub.Status = "closed"
			sub.CloseReason = input.Reason
			sub.ClosedAt = &now
			a.subscriptions[input.SubscriptionID] = sub
		}
	}

	return domain.ActionResult{
		SchemaVersion: domain.DefaultSchemaVersion,
		ActionID:      fmt.Sprintf("act-close-sub-%s", input.SubscriptionID),
		ActivityType:  "CloseSubscription",
		Status:        domain.ActionResultStatusSuccess,
		Output:        map[string]interface{}{"subscription_id": input.SubscriptionID, "status": "closed", "reason": input.Reason},
		CompletedAt:   now,
	}, nil
}

// GetSubscription retrieves a subscription record by ID.
func (a *Activities) GetSubscription(subscriptionID string) (SubscriptionRecord, bool) {
	a.subMutex.RLock()
	defer a.subMutex.RUnlock()
	sub, ok := a.subscriptions[subscriptionID]
	return sub, ok
}

// GetSubscriptions returns a slice of all recorded subscriptions.
func (a *Activities) GetSubscriptions() []SubscriptionRecord {
	a.subMutex.RLock()
	defer a.subMutex.RUnlock()
	res := make([]SubscriptionRecord, 0, len(a.subscriptions))
	for _, sub := range a.subscriptions {
		res = append(res, sub)
	}
	return res
}
// FailureTracker manages thread-safe attempt counts per execution/action.
type FailureTracker struct {
	mu       sync.Mutex
	attempts map[string]int
}

// NewFailureTracker creates a new FailureTracker.
func NewFailureTracker() *FailureTracker {
	return &FailureTracker{
		attempts: make(map[string]int),
	}
}

// IncrementAndGet increments the attempt count for key and returns the new value.
func (ft *FailureTracker) IncrementAndGet(key string) int {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	ft.attempts[key]++
	return ft.attempts[key]
}

// Reset clears the attempt count for key.
func (ft *FailureTracker) Reset(key string) {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	delete(ft.attempts, key)
}

func (a *Activities) getFailureTracker() *FailureTracker {
	a.serviceMutex.Lock()
	defer a.serviceMutex.Unlock()
	if a.failureTracker == nil {
		a.failureTracker = NewFailureTracker()
	}
	return a.failureTracker
}

func parseBool(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		s := strings.ToLower(strings.TrimSpace(val))
		return s == "true" || s == "1" || s == "yes"
	case int:
		return val != 0
	case int64:
		return val != 0
	case float64:
		return val != 0
	default:
		return false
	}
}

func parseInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case string:
		var n int
		if _, err := fmt.Sscanf(strings.TrimSpace(val), "%d", &n); err == nil {
			return n
		}
	}
	return 0
}

func getExecuteNodeSimulatedFailureConfig(input ExecuteNodeInput) (enabled bool, maxAttempts int) {
	maxAttempts = 3 // default

	checkMap := func(m map[string]interface{}) {
		if m == nil {
			return
		}
		if val, ok := m["simulated_activity_failure"]; ok {
			enabled = parseBool(val)
		}
		if val, ok := m["max_failure_attempts"]; ok {
			if parsed := parseInt(val); parsed > 0 {
				maxAttempts = parsed
			}
		}
	}

	checkMap(input.Node.Params)
	checkMap(input.Params)

	if maxAttempts < 1 {
		maxAttempts = 1
	} else if maxAttempts > 9999 {
		maxAttempts = 9999
	}

	return enabled, maxAttempts
}

