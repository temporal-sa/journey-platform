package workflows_test

import (
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/activities"
	"github.com/validated-pattern/journey-platform/internal/compiler"
	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/workflows"
	"go.temporal.io/sdk/testsuite"
)

func setupWaitForEventEnv(t *testing.T) (*testsuite.WorkflowTestSuite, *testsuite.TestWorkflowEnvironment, *activities.Activities) {
	s := &testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()
	env.SetDataConverter(workflows.NewIsolatedDataConverter(nil))

	act := activities.NewActivities()
	workflows.RegisterAllActivities(env, act)
	return s, env, act
}

func helperCreateWaitForEventIR(act *activities.Activities, timeoutSec int64) *domain.CompiledIR {
	nodes := []domain.IRNode{
		{ID: "start_node", Type: "event_start"},
		{
			ID:   "wait_node",
			Type: "wait_for_event",
			Params: map[string]interface{}{
				"event_type":           "order.completed",
				"condition_expression": "amount > 100",
				"timeout_seconds":      timeoutSec,
			},
		},
		{ID: "exit_matched", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
		{ID: "exit_timeout", Type: "exit", Params: map[string]interface{}{"status": "suppressed"}},
	}
	edges := []domain.IREdge{
		{ID: "edge1", SourceID: "start_node", TargetID: "wait_node"},
		{ID: "edge2", SourceID: "wait_node", TargetID: "exit_matched", ConditionExpression: "matched"},
		{ID: "edge3", SourceID: "wait_node", TargetID: "exit_timeout", ConditionExpression: "timeout"},
	}

	ir := &domain.CompiledIR{
		SchemaVersion: domain.DefaultSchemaVersion,
		IRID:          "ir-wait-test",
		DraftID:       "draft-wait",
		TenantID:      "tenant-test",
		Version:       1,
		EntryNodeID:   nodes[0].ID,
		Nodes:         nodes,
		Edges:         edges,
		CompiledAt:    time.Now().UTC(),
	}

	hash, _ := compiler.ComputeIRHash(ir)
	ir.ContentHash = hash
	act.RegisterCompiledIR(ir)
	return ir
}

// 1. Event before wait (pre-buffered signal)
func TestWaitForEvent_EventBeforeWait(t *testing.T) {
	_, env, act := setupWaitForEventEnv(t)
	ir := helperCreateWaitForEventIR(act, 60)

	// Send signal at start before wait_node executes
	env.RegisterDelayedCallback(func() {
		sig := workflows.EventSignal{
			EventID:    "evt-before-1",
			EventType:  "order.completed",
			Generation: 1,
			Payload: map[string]interface{}{
				"amount": 150,
			},
		}
		env.SignalWorkflow(workflows.SignalNameEvent, sig)
	}, 0)

	input := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-wait-before",
		RunID:         "run-wait-before",
		TenantID:      "tenant-test",
		ContentHash:   ir.ContentHash,
		Generation:    1,
	}

	env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("expected workflow to complete")
	}

	var res workflows.CompiledJourneyResult
	if err := env.GetWorkflowResult(&res); err != nil {
		t.Fatalf("failed to get result: %v", err)
	}

	if res.Status != workflows.StatusSucceeded {
		t.Errorf("expected status succeeded, got %s", res.Status)
	}
	if res.CurrentNodeID != "exit_matched" {
		t.Errorf("expected exit_matched node, got %s", res.CurrentNodeID)
	}

	// Verify subscription created and closed with reason "match"
	subID := "sub-wf-wait-before-1-wait_node"
	sub, ok := act.GetSubscription(subID)
	if !ok {
		t.Fatalf("expected subscription %s to be created", subID)
	}
	if sub.Status != "closed" {
		t.Errorf("expected subscription status closed, got %s", sub.Status)
	}
	if sub.CloseReason != "match" {
		t.Errorf("expected subscription close reason match, got %s", sub.CloseReason)
	}
}

// 2. Event during wait
func TestWaitForEvent_EventDuringWait(t *testing.T) {
	_, env, act := setupWaitForEventEnv(t)
	ir := helperCreateWaitForEventIR(act, 60)

	// Send signal during wait at 100ms
	env.RegisterDelayedCallback(func() {
		sig := workflows.EventSignal{
			EventID:    "evt-during-1",
			EventType:  "order.completed",
			Generation: 1,
			Payload: map[string]interface{}{
				"amount": 200,
			},
		}
		env.SignalWorkflow(workflows.SignalNameEvent, sig)
	}, time.Millisecond*100)

	input := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-wait-during",
		RunID:         "run-wait-during",
		TenantID:      "tenant-test",
		ContentHash:   ir.ContentHash,
		Generation:    1,
	}

	env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("expected workflow to complete")
	}

	var res workflows.CompiledJourneyResult
	if err := env.GetWorkflowResult(&res); err != nil {
		t.Fatalf("failed to get result: %v", err)
	}

	if res.Status != workflows.StatusSucceeded {
		t.Errorf("expected status succeeded, got %s", res.Status)
	}
	if res.CurrentNodeID != "exit_matched" {
		t.Errorf("expected exit_matched node, got %s", res.CurrentNodeID)
	}

	subID := "sub-wf-wait-during-1-wait_node"
	sub, ok := act.GetSubscription(subID)
	if !ok {
		t.Fatalf("expected subscription %s to be created", subID)
	}
	if sub.CloseReason != "match" {
		t.Errorf("expected subscription close reason match, got %s", sub.CloseReason)
	}
}

// 3. Duplicate event deduplication
func TestWaitForEvent_DuplicateEvent(t *testing.T) {
	_, env, act := setupWaitForEventEnv(t)
	ir := helperCreateWaitForEventIR(act, 60)

	env.RegisterDelayedCallback(func() {
		sig1 := workflows.EventSignal{
			EventID:    "evt-dup-123",
			EventType:  "order.completed",
			Generation: 1,
			Payload: map[string]interface{}{
				"amount": 250,
			},
		}
		sig2 := workflows.EventSignal{
			EventID:    "evt-dup-123", // duplicate ID
			EventType:  "order.completed",
			Generation: 1,
			Payload: map[string]interface{}{
				"amount": 250,
			},
		}
		env.SignalWorkflow(workflows.SignalNameEvent, sig1)
		env.SignalWorkflow(workflows.SignalNameEvent, sig2)
	}, time.Millisecond*50)

	input := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-wait-dup",
		RunID:         "run-wait-dup",
		TenantID:      "tenant-test",
		ContentHash:   ir.ContentHash,
		Generation:    1,
	}

	env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("expected workflow to complete")
	}

	var res workflows.CompiledJourneyResult
	if err := env.GetWorkflowResult(&res); err != nil {
		t.Fatalf("failed to get result: %v", err)
	}

	if res.Status != workflows.StatusSucceeded {
		t.Errorf("expected status succeeded, got %s", res.Status)
	}
}

// 4. Timeout
func TestWaitForEvent_Timeout(t *testing.T) {
	_, env, act := setupWaitForEventEnv(t)
	ir := helperCreateWaitForEventIR(act, 10) // 10 second timeout

	input := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-wait-timeout",
		RunID:         "run-wait-timeout",
		TenantID:      "tenant-test",
		ContentHash:   ir.ContentHash,
		Generation:    1,
	}

	// No signal sent, workflow should time out via Temporal time skipping
	env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("expected workflow to complete after timeout")
	}

	var res workflows.CompiledJourneyResult
	if err := env.GetWorkflowResult(&res); err != nil {
		t.Fatalf("failed to get result: %v", err)
	}

	if res.Status != workflows.StatusSuppressed {
		t.Errorf("expected status suppressed (from timeout branch), got %s", res.Status)
	}
	if res.CurrentNodeID != "exit_timeout" {
		t.Errorf("expected exit_timeout node, got %s", res.CurrentNodeID)
	}

	subID := "sub-wf-wait-timeout-1-wait_node"
	sub, ok := act.GetSubscription(subID)
	if !ok {
		t.Fatalf("expected subscription %s to be created", subID)
	}
	if sub.Status != "closed" {
		t.Errorf("expected subscription status closed, got %s", sub.Status)
	}
	if sub.CloseReason != "timeout" {
		t.Errorf("expected subscription close reason timeout, got %s", sub.CloseReason)
	}
}

// 5. Cutoff rule (Signal before vs after deadline)
func TestWaitForEvent_CutoffRule(t *testing.T) {
	t.Run("SignalBeforeCutoff", func(t *testing.T) {
		_, env, act := setupWaitForEventEnv(t)
		ir := helperCreateWaitForEventIR(act, 10)

		// Signal sent at 5 seconds (well before 10s cutoff)
		env.RegisterDelayedCallback(func() {
			sig := workflows.EventSignal{
				EventID:    "evt-cutoff-1",
				EventType:  "order.completed",
				Generation: 1,
				Payload:    map[string]interface{}{"amount": 500},
			}
			env.SignalWorkflow(workflows.SignalNameEvent, sig)
		}, time.Second*5)

		input := workflows.CompiledJourneyInput{
			SchemaVersion: domain.DefaultSchemaVersion,
			WorkflowID:    "wf-cutoff-before",
			RunID:         "run-cutoff-before",
			TenantID:      "tenant-test",
			ContentHash:   ir.ContentHash,
			Generation:    1,
		}

		env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

		if !env.IsWorkflowCompleted() {
			t.Fatalf("expected workflow to complete")
		}

		var res workflows.CompiledJourneyResult
		_ = env.GetWorkflowResult(&res)
		if res.Status != workflows.StatusSucceeded {
			t.Errorf("expected status succeeded for signal before cutoff, got %s", res.Status)
		}
	})

	t.Run("TimeoutAfterCutoff", func(t *testing.T) {
		_, env, act := setupWaitForEventEnv(t)
		ir := helperCreateWaitForEventIR(act, 5)

		input := workflows.CompiledJourneyInput{
			SchemaVersion: domain.DefaultSchemaVersion,
			WorkflowID:    "wf-cutoff-after",
			RunID:         "run-cutoff-after",
			TenantID:      "tenant-test",
			ContentHash:   ir.ContentHash,
			Generation:    1,
		}

		env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

		if !env.IsWorkflowCompleted() {
			t.Fatalf("expected workflow to complete")
		}

		var res workflows.CompiledJourneyResult
		_ = env.GetWorkflowResult(&res)
		if res.Status != workflows.StatusSuppressed {
			t.Errorf("expected status suppressed (timeout) past cutoff, got %s", res.Status)
		}
	})
}

// 6. Old generation signal rejection
func TestWaitForEvent_OldGenerationSignalRejection(t *testing.T) {
	_, env, act := setupWaitForEventEnv(t)
	ir := helperCreateWaitForEventIR(act, 10)

	env.RegisterDelayedCallback(func() {
		// Signal from generation 1 sent to workflow running at generation 2
		oldSig := workflows.EventSignal{
			EventID:    "evt-old-gen-1",
			EventType:  "order.completed",
			Generation: 1, // OLD generation
			Payload:    map[string]interface{}{"amount": 300},
		}
		env.SignalWorkflow(workflows.SignalNameEvent, oldSig)

		// Send valid generation 2 signal after 100ms
		newSig := workflows.EventSignal{
			EventID:    "evt-new-gen-2",
			EventType:  "order.completed",
			Generation: 2, // CURRENT generation
			Payload:    map[string]interface{}{"amount": 300},
		}
		env.RegisterDelayedCallback(func() {
			env.SignalWorkflow(workflows.SignalNameEvent, newSig)
		}, time.Millisecond*100)
	}, time.Millisecond*50)

	input := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-gen-reject",
		RunID:         "run-gen-reject",
		TenantID:      "tenant-test",
		ContentHash:   ir.ContentHash,
		Generation:    2, // Workflow running at Generation 2
	}

	env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("expected workflow to complete")
	}

	var res workflows.CompiledJourneyResult
	if err := env.GetWorkflowResult(&res); err != nil {
		t.Fatalf("failed to get result: %v", err)
	}

	if res.Status != workflows.StatusSucceeded {
		t.Errorf("expected status succeeded from valid generation 2 signal, got %s", res.Status)
	}
}

// 7. Cancellation cleanup
func TestWaitForEvent_CancellationCleanup(t *testing.T) {
	_, env, act := setupWaitForEventEnv(t)
	ir := helperCreateWaitForEventIR(act, 300) // long timeout

	env.RegisterDelayedCallback(func() {
		sig := domain.WorkflowSignal{
			SchemaVersion: domain.DefaultSchemaVersion,
			SignalName:    workflows.SignalNameCancel,
			RunID:         "run-cancel-clean",
		}
		env.SignalWorkflow(workflows.SignalNameCancel, sig)
	}, time.Millisecond*100)

	input := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-cancel-clean",
		RunID:         "run-cancel-clean",
		TenantID:      "tenant-test",
		ContentHash:   ir.ContentHash,
		Generation:    1,
	}

	env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("expected workflow to complete after cancellation")
	}

	var res workflows.CompiledJourneyResult
	if err := env.GetWorkflowResult(&res); err != nil {
		t.Fatalf("failed to get result: %v", err)
	}

	if res.Status != workflows.StatusCancelled {
		t.Errorf("expected status cancelled, got %s", res.Status)
	}

	subID := "sub-wf-cancel-clean-1-wait_node"
	sub, ok := act.GetSubscription(subID)
	if !ok {
		t.Fatalf("expected subscription %s to be created", subID)
	}
	if sub.CloseReason != "cancellation" {
		t.Errorf("expected subscription close reason cancellation, got %s", sub.CloseReason)
	}
}

// 8. Continue-As-New state propagation
func TestWaitForEvent_ContinueAsNewStatePropagation(t *testing.T) {
	_, env, act := setupWaitForEventEnv(t)

	nodes := []domain.IRNode{
		{ID: "start_node", Type: "event_start"},
		{
			ID:   "wait_can",
			Type: "wait_for_event",
			Params: map[string]interface{}{
				"event_type":       "order.completed",
				"continue_as_new": true,
			},
		},
		{ID: "exit_node", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
	}
	edges := []domain.IREdge{
		{ID: "edge1", SourceID: "start_node", TargetID: "wait_can"},
		{ID: "edge2", SourceID: "wait_can", TargetID: "exit_node"},
	}

	ir := &domain.CompiledIR{
		SchemaVersion: domain.DefaultSchemaVersion,
		IRID:          "ir-can-test",
		DraftID:       "draft-can",
		TenantID:      "tenant-test",
		Version:       1,
		EntryNodeID:   nodes[0].ID,
		Nodes:         nodes,
		Edges:         edges,
		CompiledAt:    time.Now().UTC(),
	}
	hash, _ := compiler.ComputeIRHash(ir)
	ir.ContentHash = hash
	act.RegisterCompiledIR(ir)

	// Send signal before node execution
	env.RegisterDelayedCallback(func() {
		sig := workflows.EventSignal{
			EventID:    "evt-can-1",
			EventType:  "order.completed",
			Generation: 1,
			Payload:    map[string]interface{}{"order_id": "ord-777"},
		}
		env.SignalWorkflow(workflows.SignalNameEvent, sig)
	}, 0)

	input := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-can-prop",
		RunID:         "run-can-prop",
		TenantID:      "tenant-test",
		ContentHash:   ir.ContentHash,
		Generation:    1,
	}

	env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

	// In Temporal testsuite, NewContinueAsNewError causes IsWorkflowCompleted() to return true,
	// and env.GetWorkflowError() returns the ContinueAsNewError
	if !env.IsWorkflowCompleted() {
		t.Fatalf("expected workflow to complete with ContinueAsNew")
	}

	wfErr := env.GetWorkflowError()
	if wfErr == nil {
		t.Fatalf("expected ContinueAsNewError, got nil")
	}
}

// 9. Diagnostic Queries (journey.query.status, journey.query.variables)
func TestWaitForEvent_DiagnosticQueries(t *testing.T) {
	_, env, act := setupWaitForEventEnv(t)
	ir := helperCreateWaitForEventIR(act, 60)

	env.RegisterDelayedCallback(func() {
		// Query status
		statusRes, err := env.QueryWorkflow(workflows.QueryTypeStatus)
		if err != nil {
			t.Errorf("failed to query status: %v", err)
		} else {
			var result workflows.CompiledJourneyResult
			if err := statusRes.Get(&result); err != nil {
				t.Errorf("failed to unmarshal status query: %v", err)
			} else if result.WorkflowID != "wf-diag-query" {
				t.Errorf("expected WorkflowID wf-diag-query, got %s", result.WorkflowID)
			}
		}

		// Query variables
		varRes, err := env.QueryWorkflow(workflows.QueryTypeVariables)
		if err != nil {
			t.Errorf("failed to query variables: %v", err)
		} else {
			var vars map[string]interface{}
			if err := varRes.Get(&vars); err != nil {
				t.Errorf("failed to unmarshal variables query: %v", err)
			} else if vars["user_id"] != "usr-999" {
				t.Errorf("expected user_id usr-999, got %v", vars["user_id"])
			}
		}

		// Send matching signal to complete workflow
		sig := workflows.EventSignal{
			EventID:    "evt-diag-1",
			EventType:  "order.completed",
			Generation: 1,
			Payload:    map[string]interface{}{"amount": 250},
		}
		env.SignalWorkflow(workflows.SignalNameEvent, sig)
	}, time.Millisecond*50)

	input := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-diag-query",
		RunID:         "run-diag-query",
		TenantID:      "tenant-test",
		ContentHash:   ir.ContentHash,
		Generation:    1,
		InputPayload: map[string]interface{}{
			"user_id": "usr-999",
		},
	}

	env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("expected workflow to complete")
	}
}
