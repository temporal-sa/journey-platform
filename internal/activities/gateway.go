package activities

import (
	"context"
	"fmt"
	"sync"
	"time"
	"github.com/validated-pattern/journey-platform/internal/api/middleware"
)

// LedgerStatus defines the states in the action ledger state machine.
type LedgerStatus string

const (
	LedgerStatusReserved         LedgerStatus = "reserved"
	LedgerStatusDispatching      LedgerStatus = "dispatching"
	LedgerStatusAccepted         LedgerStatus = "accepted"
	LedgerStatusPermanentFailure LedgerStatus = "permanent_failure"
	LedgerStatusSuppressed       LedgerStatus = "suppressed"
	LedgerStatusRetryableFailure LedgerStatus = "retryable_failure"
	LedgerStatusUnknown          LedgerStatus = "unknown"
)

// ActionRequest contains all parameters required to execute an action via the gateway.
type ActionRequest struct {
	TenantID           string                 `json:"tenant_id"`
	WorkflowID         string                 `json:"workflow_id"`
	JourneyVersion     string                 `json:"journey_version"`
	NodeID             string                 `json:"node_id"`
	NodeVisit          int                    `json:"node_visit"`
	ActionVersion      string                 `json:"action_version"`
	TemplateVersion    string                 `json:"template_version"`
	SubjectRef         string                 `json:"subject_ref"`
	ParameterRefs      map[string]interface{} `json:"parameter_refs,omitempty"`
	ExecutionMode      ExecutionMode          `json:"execution_mode"`
	TestRunID          string                 `json:"test_run_id,omitempty"`
	ExperimentContexts map[string]interface{} `json:"experiment_contexts,omitempty"`
	SimulatedActivityFailure bool                   `json:"simulated_activity_failure,omitempty"`
	MaxFailureAttempts       int                    `json:"max_failure_attempts,omitempty"`

	// Additional routing & content context
	Channel          string `json:"channel,omitempty"`
	RecipientAddress string `json:"recipient_address,omitempty"`
	TemplateBody     string `json:"template_body,omitempty"`
}

// IdempotencyKey generates the stable idempotency key for this request.
// Format: tenant_id:workflow_id:node_id:visit_index
func (r *ActionRequest) IdempotencyKey() string {
	return fmt.Sprintf("%s:%s:%s:%d", r.TenantID, r.WorkflowID, r.NodeID, r.NodeVisit)
}

// ActionLedgerRow represents a record in the action ledger.
type ActionLedgerRow struct {
	IdempotencyKey   string                 `json:"idempotency_key"`
	TenantID         string                 `json:"tenant_id"`
	WorkflowID       string                 `json:"workflow_id"`
	NodeID           string                 `json:"node_id"`
	NodeVisit        int                    `json:"node_visit"`
	Status           LedgerStatus           `json:"status"`
	ReasonCode       string                 `json:"reason_code,omitempty"`
	RenderedContent  string                 `json:"rendered_content,omitempty"`
	ProviderResponse map[string]interface{} `json:"provider_response,omitempty"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
	CallStartedAt    *time.Time             `json:"call_started_at,omitempty"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty"`
	LeaseExpiresAt   *time.Time             `json:"lease_expires_at,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// GatewayResult represents the outcome returned by the ActionGateway.
type GatewayResult struct {
	IdempotencyKey   string                 `json:"idempotency_key"`
	Status           LedgerStatus           `json:"status"`
	ReasonCode       string                 `json:"reason_code,omitempty"`
	RenderedContent  string                 `json:"rendered_content,omitempty"`
	ProviderResponse map[string]interface{} `json:"provider_response,omitempty"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
	ExecutedAt       time.Time              `json:"executed_at"`
	Reconciled       bool                   `json:"reconciled,omitempty"`
}

// ProviderRequest represents the payload sent to an ActionProvider.
type ProviderRequest struct {
	IdempotencyKey   string                 `json:"idempotency_key"`
	TenantID         string                 `json:"tenant_id"`
	SubjectRef       string                 `json:"subject_ref"`
	RecipientAddress string                 `json:"recipient_address,omitempty"`
	Channel          string                 `json:"channel,omitempty"`
	RenderedContent  string                 `json:"rendered_content"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ProviderResponse represents the output returned by an ActionProvider.
type ProviderResponse struct {
	Status       LedgerStatus           `json:"status"`
	ProviderID   string                 `json:"provider_id,omitempty"`
	ReasonCode   string                 `json:"reason_code,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ActionProvider defines the protocol for dispatching actions and reconciling status.
type ActionProvider interface {
	Dispatch(ctx context.Context, req *ProviderRequest) (*ProviderResponse, error)
	Reconcile(ctx context.Context, idempotencyKey string) (*ProviderResponse, error)
}

// PolicyEvaluator evaluates communication policy.
type PolicyEvaluator interface {
	Evaluate(ctx context.Context, pctx *PolicyContext) PolicyDecisionResult
}

// TemplateRenderer defines template rendering capability.
type TemplateRenderer interface {
	Render(ctx context.Context, templateVersion string, templateBody string, params map[string]interface{}) (string, error)
}

// DefaultTemplateRenderer provides a standard implementation of TemplateRenderer.
type DefaultTemplateRenderer struct{}

func (r *DefaultTemplateRenderer) Render(ctx context.Context, templateVersion string, templateBody string, params map[string]interface{}) (string, error) {
	if templateBody != "" {
		content := templateBody
		for k, v := range params {
			placeholder := fmt.Sprintf("{{%s}}", k)
			content = fmt.Sprintf("%s [%s=%v]", content, placeholder, v)
		}
		return content, nil
	}
	return fmt.Sprintf("Template[%s]: params=%v", templateVersion, params), nil
}

// OutboxFact represents an accepted action fact for outbox emission.
type OutboxFact struct {
	ID            string                 `json:"id"`
	TenantID      string                 `json:"tenant_id"`
	AggregateType string                 `json:"aggregate_type"`
	AggregateID   string                 `json:"aggregate_id"`
	EventType     string                 `json:"event_type"`
	Payload       map[string]interface{} `json:"payload"`
	CreatedAt     time.Time              `json:"created_at"`
	Status        string                 `json:"status"`
}

// OutboxStore manages outbox records.
type OutboxStore interface {
	SaveFact(ctx context.Context, fact *OutboxFact) error
	GetFacts(ctx context.Context, tenantID string) ([]*OutboxFact, error)
}

// MemoryOutboxStore is an in-memory implementation of OutboxStore.
type MemoryOutboxStore struct {
	mu    sync.RWMutex
	facts []*OutboxFact
}

func NewMemoryOutboxStore() *MemoryOutboxStore {
	return &MemoryOutboxStore{
		facts: make([]*OutboxFact, 0),
	}
}

func (s *MemoryOutboxStore) SaveFact(ctx context.Context, fact *OutboxFact) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.facts = append(s.facts, fact)
	return nil
}

func (s *MemoryOutboxStore) GetFacts(ctx context.Context, tenantID string) ([]*OutboxFact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var res []*OutboxFact
	for _, f := range s.facts {
		if tenantID == "" || f.TenantID == tenantID {
			res = append(res, f)
		}
	}
	return res, nil
}

// ActionLedgerStore manages persistence of ActionLedgerRow records.
type ActionLedgerStore interface {
	Get(ctx context.Context, idempotencyKey string) (*ActionLedgerRow, error)
	ReserveOrLease(ctx context.Context, req *ActionRequest, leaseDuration time.Duration) (*ActionLedgerRow, error)
	Update(ctx context.Context, row *ActionLedgerRow) error
}

// MemoryActionLedgerStore implements ActionLedgerStore in memory.
type MemoryActionLedgerStore struct {
	mu   sync.RWMutex
	rows map[string]*ActionLedgerRow
}

func NewMemoryActionLedgerStore() *MemoryActionLedgerStore {
	return &MemoryActionLedgerStore{
		rows: make(map[string]*ActionLedgerRow),
	}
}

func (s *MemoryActionLedgerStore) Get(ctx context.Context, idempotencyKey string) (*ActionLedgerRow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	row, ok := s.rows[idempotencyKey]
	if !ok {
		return nil, nil
	}
	cp := *row
	return &cp, nil
}

func (s *MemoryActionLedgerStore) ReserveOrLease(ctx context.Context, req *ActionRequest, leaseDuration time.Duration) (*ActionLedgerRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := req.IdempotencyKey()
	now := time.Now().UTC()
	row, exists := s.rows[key]
	if !exists {
		expiresAt := now.Add(leaseDuration)
		newRow := &ActionLedgerRow{
			IdempotencyKey: key,
			TenantID:       req.TenantID,
			WorkflowID:     req.WorkflowID,
			NodeID:         req.NodeID,
			NodeVisit:      req.NodeVisit,
			Status:         LedgerStatusReserved,
			LeaseExpiresAt: &expiresAt,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		s.rows[key] = newRow
		cp := *newRow
		return &cp, nil
	}

	cp := *row
	return &cp, nil
}

func (s *MemoryActionLedgerStore) Update(ctx context.Context, row *ActionLedgerRow) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	row.UpdatedAt = now
	cp := *row
	s.rows[row.IdempotencyKey] = &cp
	return nil
}

// ActionGateway coordinates idempotent action dispatching, policy enforcement, template rendering, and reconciliation.
type ActionGateway struct {
	ledgerStore       ActionLedgerStore
	outboxStore       OutboxStore
	provider          ActionProvider
	policyEvaluator   PolicyEvaluator
	attributeResolver AttributeResolver
	renderer          TemplateRenderer
	leaseDuration     time.Duration
	failureTracker    *FailureTracker
	trackerMu         sync.Mutex
	keyLocks          sync.Map
}
// GatewayOption configures an ActionGateway.
type GatewayOption func(*ActionGateway)

func WithLedgerStore(store ActionLedgerStore) GatewayOption {
	return func(g *ActionGateway) {
		g.ledgerStore = store
	}
}

func WithOutboxStore(store OutboxStore) GatewayOption {
	return func(g *ActionGateway) {
		g.outboxStore = store
	}
}

func WithProvider(p ActionProvider) GatewayOption {
	return func(g *ActionGateway) {
		g.provider = p
	}
}

func WithPolicyEvaluator(p PolicyEvaluator) GatewayOption {
	return func(g *ActionGateway) {
		g.policyEvaluator = p
	}
}

func WithAttributeResolver(r AttributeResolver) GatewayOption {
	return func(g *ActionGateway) {
		g.attributeResolver = r
	}
}

func WithTemplateRenderer(r TemplateRenderer) GatewayOption {
	return func(g *ActionGateway) {
		g.renderer = r
	}
}

func WithLeaseDuration(d time.Duration) GatewayOption {
	return func(g *ActionGateway) {
		g.leaseDuration = d
	}
}
func WithFailureTracker(ft *FailureTracker) GatewayOption {
	return func(g *ActionGateway) {
		g.failureTracker = ft
	}
}

// NewActionGateway constructs a new ActionGateway instance.
func NewActionGateway(opts ...GatewayOption) *ActionGateway {
	g := &ActionGateway{
		ledgerStore:   NewMemoryActionLedgerStore(),
		outboxStore:   NewMemoryOutboxStore(),
		renderer:      &DefaultTemplateRenderer{},
		leaseDuration: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

func (g *ActionGateway) getKeyLock(key string) *sync.Mutex {
	val, _ := g.keyLocks.LoadOrStore(key, &sync.Mutex{})
	return val.(*sync.Mutex)
}

// ExecuteAction processes an action request idempotently.
func (g *ActionGateway) ExecuteAction(ctx context.Context, req ActionRequest) (*GatewayResult, error) {
	ctx = middleware.EnsureOTelSpanContext(ctx)

	enabled, maxAttempts := getActionRequestSimulatedFailureConfig(req)
	if enabled {
		actionKey := req.IdempotencyKey()
		if actionKey == ":::" || actionKey == ":::0" || actionKey == "" {
			actionKey = fmt.Sprintf("action:%s:%d", req.NodeID, req.NodeVisit)
		}
		attempt := g.getFailureTracker().IncrementAndGet(actionKey)
		if attempt <= maxAttempts {
			return nil, fmt.Errorf("simulated activity execution failure attempt %d/%d", attempt, maxAttempts)
		}
		g.getFailureTracker().Reset(actionKey)
	}

	key := req.IdempotencyKey()
	lock := g.getKeyLock(key)
	lock.Lock()
	defer lock.Unlock()

	row, err := g.ledgerStore.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ledger row: %w", err)
	}

	// 1. Check for Terminal / Complete states
	if row != nil {
		switch row.Status {
		case LedgerStatusAccepted, LedgerStatusPermanentFailure, LedgerStatusSuppressed:
			return &GatewayResult{
				IdempotencyKey:   key,
				Status:           row.Status,
				ReasonCode:       row.ReasonCode,
				RenderedContent:  row.RenderedContent,
				ProviderResponse: row.ProviderResponse,
				ErrorMessage:     row.ErrorMessage,
				ExecutedAt:       row.UpdatedAt,
			}, nil
		}

		// 2. Reconciliation logic for expired `dispatching` and `unknown` rows before any retry
		now := time.Now().UTC()
		isExpiredDispatch := row.Status == LedgerStatusDispatching && row.LeaseExpiresAt != nil && now.After(*row.LeaseExpiresAt)
		isUnknown := row.Status == LedgerStatusUnknown

		if (isExpiredDispatch || isUnknown) && g.provider != nil {
			recResp, recErr := g.provider.Reconcile(ctx, key)
			if recErr == nil && recResp != nil {
				if recResp.Status == LedgerStatusAccepted {
					row.Status = LedgerStatusAccepted
					row.ReasonCode = recResp.ReasonCode
					row.ProviderResponse = recResp.Metadata
					row.CompletedAt = &now
					if err := g.ledgerStore.Update(ctx, row); err != nil {
						return nil, fmt.Errorf("failed to update ledger during reconciliation: %w", err)
					}
					g.emitOutboxAccepted(ctx, req, row)
					return &GatewayResult{
						IdempotencyKey:   key,
						Status:           LedgerStatusAccepted,
						ReasonCode:       recResp.ReasonCode,
						ProviderResponse: recResp.Metadata,
						ExecutedAt:       now,
						Reconciled:       true,
					}, nil
				} else if recResp.Status == LedgerStatusPermanentFailure || recResp.Status == LedgerStatusSuppressed {
					row.Status = recResp.Status
					row.ReasonCode = recResp.ReasonCode
					row.ErrorMessage = recResp.ErrorMessage
					row.ProviderResponse = recResp.Metadata
					row.CompletedAt = &now
					if err := g.ledgerStore.Update(ctx, row); err != nil {
						return nil, fmt.Errorf("failed to update ledger during reconciliation: %w", err)
					}
					return &GatewayResult{
						IdempotencyKey:   key,
						Status:           recResp.Status,
						ReasonCode:       recResp.ReasonCode,
						ErrorMessage:     recResp.ErrorMessage,
						ProviderResponse: recResp.Metadata,
						ExecutedAt:       now,
						Reconciled:       true,
					}, nil
				}
			}
		}
	}

	// 3. Reserve/lease action-ledger row under stable idempotency key
	row, err = g.ledgerStore.ReserveOrLease(ctx, &req, g.leaseDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to reserve action ledger row: %w", err)
	}

	// 4. Authoritative policy re-check before address resolution and provider dispatch
	if g.policyEvaluator != nil {
		pctx := PolicyContext{
			SubjectID:        req.SubjectRef,
			Channel:          req.Channel,
			ExecutionMode:    req.ExecutionMode,
			RecipientAddress: req.RecipientAddress,
		}
		decision := g.policyEvaluator.Evaluate(ctx, &pctx)
		if decision.Decision == DecisionSuppress {
			now := time.Now().UTC()
			row.Status = LedgerStatusSuppressed
			row.ReasonCode = decision.ReasonCode
			row.CompletedAt = &now
			if err := g.ledgerStore.Update(ctx, row); err != nil {
				return nil, fmt.Errorf("failed to update ledger status to suppressed: %w", err)
			}
			return &GatewayResult{
				IdempotencyKey: key,
				Status:         LedgerStatusSuppressed,
				ReasonCode:     decision.ReasonCode,
				ExecutedAt:     now,
			}, nil
		}
	}

	// 5. Address resolution (if needed) & Pinned template rendering
	recipientAddress := req.RecipientAddress
	if recipientAddress == "" && g.attributeResolver != nil {
		rawVal, resErr := g.attributeResolver.ResolveAttribute(ctx, req.SubjectRef, "", "address")
		if resErr == nil && rawVal != nil && rawVal.Value != nil {
			recipientAddress = fmt.Sprintf("%v", rawVal.Value)
		}
	}

	rendered, renderErr := g.renderer.Render(ctx, req.TemplateVersion, req.TemplateBody, req.ParameterRefs)
	if renderErr != nil {
		now := time.Now().UTC()
		row.Status = LedgerStatusPermanentFailure
		row.ErrorMessage = fmt.Sprintf("template render error: %v", renderErr)
		row.CompletedAt = &now
		if err := g.ledgerStore.Update(ctx, row); err != nil {
			return nil, fmt.Errorf("failed to update ledger status on render failure: %w", err)
		}
		return &GatewayResult{
			IdempotencyKey: key,
			Status:         LedgerStatusPermanentFailure,
			ErrorMessage:   row.ErrorMessage,
			ExecutedAt:     now,
		}, nil
	}
	row.RenderedContent = rendered

	// 6. Fence row status to `dispatching` & record provider call start
	callStart := time.Now().UTC()
	leaseExpires := callStart.Add(g.leaseDuration)
	row.Status = LedgerStatusDispatching
	row.CallStartedAt = &callStart
	row.LeaseExpiresAt = &leaseExpires
	if err := g.ledgerStore.Update(ctx, row); err != nil {
		return nil, fmt.Errorf("failed to update ledger status to dispatching: %w", err)
	}

	// 7. Dispatch to provider with idempotency key
	pReq := &ProviderRequest{
		IdempotencyKey:   key,
		TenantID:         req.TenantID,
		SubjectRef:       req.SubjectRef,
		RecipientAddress: recipientAddress,
		Channel:          req.Channel,
		RenderedContent:  rendered,
		Metadata:         req.ExperimentContexts,
	}

	var pResp *ProviderResponse
	var dispatchErr error
	if g.provider != nil {
		pResp, dispatchErr = g.provider.Dispatch(ctx, pReq)
	} else {
		pResp = &ProviderResponse{
			Status:     LedgerStatusAccepted,
			ReasonCode: ReasonAllowed,
		}
	}

	// 8. Store result status (`accepted`, `permanent_failure`, `suppressed`, `retryable_failure`, `unknown`)
	now := time.Now().UTC()
	row.CompletedAt = &now

	if dispatchErr != nil {
		row.Status = LedgerStatusUnknown
		row.ErrorMessage = dispatchErr.Error()
	} else if pResp != nil {
		row.Status = pResp.Status
		row.ReasonCode = pResp.ReasonCode
		row.ErrorMessage = pResp.ErrorMessage
		row.ProviderResponse = pResp.Metadata
	} else {
		row.Status = LedgerStatusUnknown
		row.ErrorMessage = "nil provider response"
	}

	if err := g.ledgerStore.Update(ctx, row); err != nil {
		return nil, fmt.Errorf("failed to update ledger final status: %w", err)
	}

	// 9. Outbox transaction emitting accepted action facts
	if row.Status == LedgerStatusAccepted {
		g.emitOutboxAccepted(ctx, req, row)
	}
	g.getFailureTracker().Reset(fmt.Sprintf("%s:%s:%s:%d", req.TenantID, req.WorkflowID, req.NodeID, req.NodeVisit))
	return &GatewayResult{
		Status:           row.Status,
		ReasonCode:       row.ReasonCode,
		RenderedContent:  rendered,
		ProviderResponse: row.ProviderResponse,
		ErrorMessage:     row.ErrorMessage,
		ExecutedAt:       now,
	}, nil
}
func (g *ActionGateway) getFailureTracker() *FailureTracker {
	g.trackerMu.Lock()
	defer g.trackerMu.Unlock()
	if g.failureTracker == nil {
		g.failureTracker = NewFailureTracker()
	}
	return g.failureTracker
}

func getActionRequestSimulatedFailureConfig(req ActionRequest) (enabled bool, maxAttempts int) {
	maxAttempts = 3 // default

	if req.SimulatedActivityFailure {
		enabled = true
	}
	if req.MaxFailureAttempts > 0 {
		maxAttempts = req.MaxFailureAttempts
	}

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

	checkMap(req.ParameterRefs)
	checkMap(req.ExperimentContexts)

	if maxAttempts < 1 {
		maxAttempts = 1
	} else if maxAttempts > 10 {
		maxAttempts = 10
	}

	return enabled, maxAttempts
}

func (g *ActionGateway) emitOutboxAccepted(ctx context.Context, req ActionRequest, row *ActionLedgerRow) {
	if g.outboxStore == nil {
		return
	}
	fact := &OutboxFact{
		ID:            fmt.Sprintf("outbox-%s", row.IdempotencyKey),
		TenantID:      req.TenantID,
		AggregateType: "action",
		AggregateID:   row.IdempotencyKey,
		EventType:     "action.accepted",
		Payload: map[string]interface{}{
			"tenant_id":        req.TenantID,
			"workflow_id":      req.WorkflowID,
			"journey_version":  req.JourneyVersion,
			"node_id":          req.NodeID,
			"node_visit":       req.NodeVisit,
			"subject_ref":      req.SubjectRef,
			"action_version":   req.ActionVersion,
			"template_version": req.TemplateVersion,
			"idempotency_key":  row.IdempotencyKey,
			"accepted_at":      time.Now().UTC().Format(time.RFC3339),
			"reason_code":      row.ReasonCode,
		},
		CreatedAt: time.Now().UTC(),
		Status:    "pending",
	}
	_ = g.outboxStore.SaveFact(ctx, fact)
}
