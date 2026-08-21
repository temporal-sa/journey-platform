package activities

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

// MockProvider for unit testing ActionGateway.
type MockProvider struct {
	mu             sync.Mutex
	dispatchCount  int32
	reconcileCount int32
	dispatchResp   *ProviderResponse
	dispatchErr    error
	reconcileResp  *ProviderResponse
	reconcileErr   error
}

func (m *MockProvider) Dispatch(ctx context.Context, req *ProviderRequest) (*ProviderResponse, error) {
	atomic.AddInt32(&m.dispatchCount, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.dispatchErr != nil {
		return nil, m.dispatchErr
	}
	if m.dispatchResp != nil {
		return m.dispatchResp, nil
	}
	return &ProviderResponse{
		Status:     LedgerStatusAccepted,
		ReasonCode: ReasonAllowed,
		ProviderID: "prov-mock-123",
	}, nil
}

func (m *MockProvider) Reconcile(ctx context.Context, idempotencyKey string) (*ProviderResponse, error) {
	atomic.AddInt32(&m.reconcileCount, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.reconcileErr != nil {
		return nil, m.reconcileErr
	}
	if m.reconcileResp != nil {
		return m.reconcileResp, nil
	}
	return &ProviderResponse{
		Status:     LedgerStatusUnknown,
		ReasonCode: "RECONCILE_UNKNOWN",
	}, nil
}

// MockPolicyEvaluator for testing policy behavior.
type MockPolicyEvaluator struct {
	decision PolicyDecisionResult
}

func (m *MockPolicyEvaluator) Evaluate(ctx context.Context, pctx *PolicyContext) PolicyDecisionResult {
	return m.decision
}

// 1. State Transition Tests (reservation -> dispatching -> accepted/failed)
func TestGateway_StateTransitions(t *testing.T) {
	t.Run("successful dispatch transition to accepted", func(t *testing.T) {
		ledger := NewMemoryActionLedgerStore()
		outbox := NewMemoryOutboxStore()
		provider := &MockProvider{
			dispatchResp: &ProviderResponse{
				Status:     LedgerStatusAccepted,
				ReasonCode: ReasonAllowed,
			},
		}

		gw := NewActionGateway(
			WithLedgerStore(ledger),
			WithOutboxStore(outbox),
			WithProvider(provider),
		)

		req := ActionRequest{
			TenantID:        "tenant-1",
			WorkflowID:      "wf-101",
			JourneyVersion:  "1.0",
			NodeID:          "node-send-email",
			NodeVisit:       1,
			ActionVersion:   "1.0",
			TemplateVersion: "v1.2",
			SubjectRef:      "user-42",
			Channel:         "email",
			ExecutionMode:   ExecutionModeProduction,
		}

		res, err := gw.ExecuteAction(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.Status != LedgerStatusAccepted {
			t.Errorf("expected status %s, got %s", LedgerStatusAccepted, res.Status)
		}

		// Verify row in ledger store
		row, err := ledger.Get(context.Background(), req.IdempotencyKey())
		if err != nil || row == nil {
			t.Fatalf("failed to fetch ledger row: %v", err)
		}
		if row.Status != LedgerStatusAccepted {
			t.Errorf("ledger row status expected %s, got %s", LedgerStatusAccepted, row.Status)
		}
		if row.CallStartedAt == nil || row.CompletedAt == nil {
			t.Errorf("expected call timestamps to be set")
		}

		// Verify outbox fact emitted
		facts, err := outbox.GetFacts(context.Background(), "tenant-1")
		if err != nil {
			t.Fatalf("failed to get outbox facts: %v", err)
		}
		if len(facts) != 1 {
			t.Fatalf("expected 1 outbox fact, got %d", len(facts))
		}
		if facts[0].EventType != "action.accepted" {
			t.Errorf("expected event_type 'action.accepted', got '%s'", facts[0].EventType)
		}
	})

	t.Run("permanent failure transition", func(t *testing.T) {
		ledger := NewMemoryActionLedgerStore()
		provider := &MockProvider{
			dispatchResp: &ProviderResponse{
				Status:       LedgerStatusPermanentFailure,
				ReasonCode:   "INVALID_RECIPIENT",
				ErrorMessage: "recipient address rejected",
			},
		}

		gw := NewActionGateway(
			WithLedgerStore(ledger),
			WithProvider(provider),
		)

		req := ActionRequest{
			TenantID:        "tenant-1",
			WorkflowID:      "wf-101",
			JourneyVersion:  "1.0",
			NodeID:          "node-send-email",
			NodeVisit:       2,
			ActionVersion:   "1.0",
			TemplateVersion: "v1.2",
			SubjectRef:      "user-42",
			ExecutionMode:   ExecutionModeProduction,
		}

		res, err := gw.ExecuteAction(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.Status != LedgerStatusPermanentFailure {
			t.Errorf("expected status %s, got %s", LedgerStatusPermanentFailure, res.Status)
		}

		row, _ := ledger.Get(context.Background(), req.IdempotencyKey())
		if row.Status != LedgerStatusPermanentFailure {
			t.Errorf("ledger status expected %s, got %s", LedgerStatusPermanentFailure, row.Status)
		}
	})
}

// 2. Concurrent Retries Yielding Single Provider Submission
func TestGateway_ConcurrentRetries(t *testing.T) {
	ledger := NewMemoryActionLedgerStore()
	outbox := NewMemoryOutboxStore()
	provider := &MockProvider{
		dispatchResp: &ProviderResponse{
			Status:     LedgerStatusAccepted,
			ReasonCode: ReasonAllowed,
		},
	}

	gw := NewActionGateway(
		WithLedgerStore(ledger),
		WithOutboxStore(outbox),
		WithProvider(provider),
	)

	req := ActionRequest{
		TenantID:        "tenant-concurrent",
		WorkflowID:      "wf-999",
		JourneyVersion:  "1.0",
		NodeID:          "node-action",
		NodeVisit:       1,
		ActionVersion:   "1.0",
		TemplateVersion: "v1.0",
		SubjectRef:      "user-concurrent",
		ExecutionMode:   ExecutionModeProduction,
	}

	const concurrency = 15
	var wg sync.WaitGroup
	wg.Add(concurrency)

	results := make([]*GatewayResult, concurrency)
	errs := make([]error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			res, err := gw.ExecuteAction(context.Background(), req)
			results[idx] = res
			errs[idx] = err
		}(i)
	}

	wg.Wait()

	for i := 0; i < concurrency; i++ {
		if errs[i] != nil {
			t.Fatalf("goroutine %d returned error: %v", i, errs[i])
		}
		if results[i].Status != LedgerStatusAccepted {
			t.Errorf("goroutine %d expected status %s, got %s", i, LedgerStatusAccepted, results[i].Status)
		}
	}

	// Verify exactly 1 dispatch call was made
	if provider.dispatchCount != 1 {
		t.Fatalf("expected exactly 1 provider dispatch call, got %d", provider.dispatchCount)
	}

	// Verify exactly 1 outbox record saved
	facts, _ := outbox.GetFacts(context.Background(), "tenant-concurrent")
	if len(facts) != 1 {
		t.Errorf("expected 1 outbox record, got %d", len(facts))
	}
}

// 3. Unknown State Reconciliation Before Resend
func TestGateway_UnknownStateReconciliation(t *testing.T) {
	ledger := NewMemoryActionLedgerStore()
	outbox := NewMemoryOutboxStore()
	provider := &MockProvider{
		dispatchErr: errors.New("network timeout during dispatch"),
	}

	gw := NewActionGateway(
		WithLedgerStore(ledger),
		WithOutboxStore(outbox),
		WithProvider(provider),
	)

	req := ActionRequest{
		TenantID:        "tenant-rec",
		WorkflowID:      "wf-rec",
		JourneyVersion:  "1.0",
		NodeID:          "node-rec",
		NodeVisit:       1,
		ActionVersion:   "1.0",
		TemplateVersion: "v1.0",
		SubjectRef:      "user-rec",
		ExecutionMode:   ExecutionModeProduction,
	}

	// First execution fails with network error -> recorded as unknown
	res1, err := gw.ExecuteAction(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected execution error: %v", err)
	}
	if res1.Status != LedgerStatusUnknown {
		t.Fatalf("expected status %s, got %s", LedgerStatusUnknown, res1.Status)
	}
	if provider.dispatchCount != 1 {
		t.Fatalf("expected 1 dispatch attempt, got %d", provider.dispatchCount)
	}

	// Now provider is aware that request was actually processed successfully
	provider.mu.Lock()
	provider.dispatchErr = nil
	provider.reconcileResp = &ProviderResponse{
		Status:     LedgerStatusAccepted,
		ReasonCode: "RECONCILED_SUCCESS",
		ProviderID: "prov-rec-456",
	}
	provider.mu.Unlock()

	// Second execution (retry) triggers reconciliation logic before resend
	res2, err := gw.ExecuteAction(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected retry error: %v", err)
	}

	if res2.Status != LedgerStatusAccepted {
		t.Errorf("expected status %s, got %s", LedgerStatusAccepted, res2.Status)
	}
	if !res2.Reconciled {
		t.Errorf("expected Reconciled to be true")
	}

	// Ensure no extra dispatch call was made!
	if provider.dispatchCount != 1 {
		t.Errorf("expected dispatch count to remain 1, got %d", provider.dispatchCount)
	}
	if provider.reconcileCount != 1 {
		t.Errorf("expected reconcile count to be 1, got %d", provider.reconcileCount)
	}

	// Verify outbox fact emitted upon reconciled acceptance
	facts, _ := outbox.GetFacts(context.Background(), "tenant-rec")
	if len(facts) != 1 {
		t.Errorf("expected 1 outbox fact after reconciliation, got %d", len(facts))
	}
}

// 4. Policy Suppression at Gateway
func TestGateway_PolicySuppression(t *testing.T) {
	ledger := NewMemoryActionLedgerStore()
	provider := &MockProvider{}
	policyEval := &MockPolicyEvaluator{
		decision: PolicyDecisionResult{
			Decision:      DecisionSuppress,
			ReasonCode:    ReasonConsentWithdrawn,
			PolicyVersion: DefaultPolicyVersion,
		},
	}

	gw := NewActionGateway(
		WithLedgerStore(ledger),
		WithProvider(provider),
		WithPolicyEvaluator(policyEval),
	)

	req := ActionRequest{
		TenantID:        "tenant-suppress",
		WorkflowID:      "wf-suppress",
		JourneyVersion:  "1.0",
		NodeID:          "node-suppress",
		NodeVisit:       1,
		ActionVersion:   "1.0",
		TemplateVersion: "v1.0",
		SubjectRef:      "user-optout",
		ExecutionMode:   ExecutionModeProduction,
	}

	res, err := gw.ExecuteAction(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Status != LedgerStatusSuppressed {
		t.Errorf("expected status %s, got %s", LedgerStatusSuppressed, res.Status)
	}
	if res.ReasonCode != ReasonConsentWithdrawn {
		t.Errorf("expected reason code %s, got %s", ReasonConsentWithdrawn, res.ReasonCode)
	}

	// Assert provider was NEVER dispatched
	if provider.dispatchCount != 0 {
		t.Errorf("expected provider dispatch count 0, got %d", provider.dispatchCount)
	}

	// Verify ledger state recorded as suppressed
	row, _ := ledger.Get(context.Background(), req.IdempotencyKey())
	if row.Status != LedgerStatusSuppressed {
		t.Errorf("expected ledger status %s, got %s", LedgerStatusSuppressed, row.Status)
	}
}

// Additional test: Idempotency Key format verification
func TestActionRequest_IdempotencyKey(t *testing.T) {
	req := ActionRequest{
		TenantID:   "tenantA",
		WorkflowID: "wfB",
		NodeID:     "nodeC",
		NodeVisit:  3,
	}
	expected := "tenantA:wfB:nodeC:3"
	if req.IdempotencyKey() != expected {
		t.Errorf("expected key %s, got %s", expected, req.IdempotencyKey())
	}
}
func TestGateway_SimulatedActivityFailure(t *testing.T) {
	ledger := NewMemoryActionLedgerStore()
	provider := &MockProvider{
		dispatchResp: &ProviderResponse{
			Status:     LedgerStatusAccepted,
			ReasonCode: ReasonAllowed,
		},
	}
	gw := NewActionGateway(
		WithLedgerStore(ledger),
		WithProvider(provider),
	)

	req := ActionRequest{
		TenantID:        "tenant-sim",
		WorkflowID:      "wf-sim-101",
		JourneyVersion:  "1.0",
		NodeID:          "node-email-sim",
		NodeVisit:       1,
		ActionVersion:   "1.0",
		TemplateVersion: "v1.0",
		SubjectRef:      "user-sim",
		ExecutionMode:   ExecutionModeProduction,
	}

	// 1. When enabled, ExecuteAction fails
	SetSimulatedActivityFailure(true)
	defer SetSimulatedActivityFailure(false)

	res1, err1 := gw.ExecuteAction(context.Background(), req)
	if err1 == nil {
		t.Fatalf("expected error when simulated failure enabled, got result: %+v", res1)
	}
	if err1.Error() != "simulated activity execution failure" {
		t.Errorf("unexpected error message: %v", err1)
	}

	// 2. When disabled, ExecuteAction succeeds immediately
	SetSimulatedActivityFailure(false)
	res2, err2 := gw.ExecuteAction(context.Background(), req)
	if err2 != nil {
		t.Fatalf("expected success after disabling simulated failure, got error: %v", err2)
	}
	if res2.Status != LedgerStatusAccepted {
		t.Errorf("expected status %s, got %s", LedgerStatusAccepted, res2.Status)
	}
}
