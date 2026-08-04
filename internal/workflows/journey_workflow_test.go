package workflows_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/activities"
	"github.com/validated-pattern/journey-platform/internal/compiler"
	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/workflows"
	"go.temporal.io/sdk/testsuite"
)

func setupTestEnvironment(t *testing.T) (*testsuite.WorkflowTestSuite, *testsuite.TestWorkflowEnvironment, *activities.Activities) {
	s := &testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()
	env.SetDataConverter(workflows.NewIsolatedDataConverter(nil))

	act := activities.NewActivities()
	workflows.RegisterAllActivities(env, act)
	return s, env, act
}

func helperCreateAndRegisterIR(act *activities.Activities, nodes []domain.IRNode, edges []domain.IREdge) *domain.CompiledIR {
	ir := &domain.CompiledIR{
		SchemaVersion: domain.DefaultSchemaVersion,
		IRID:          "ir-test-gen",
		DraftID:       "draft-gen",
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

// Test 1: Invalid Artifact (Missing or corrupted IR content hash)
func TestCompiledJourneyWorkflow_InvalidArtifact(t *testing.T) {
	_, env, _ := setupTestEnvironment(t)

	input := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-invalid-1",
		RunID:         "run-invalid-1",
		TenantID:      "tenant-test",
		ContentHash:   "non-existent-hash-12345",
	}

	env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("expected workflow to complete")
	}
	if err := env.GetWorkflowError(); err == nil {
		t.Fatalf("expected workflow error for invalid artifact, got nil")
	}
}

// Test 2: Condition Branches (True branch vs False/Default branch)
func TestCompiledJourneyWorkflow_ConditionBranches(t *testing.T) {
	// Case A: Premium tier takes true branch to exit_premium
	t.Run("TrueBranch_Premium", func(t *testing.T) {
		_, env, act := setupTestEnvironment(t)

		nodes := []domain.IRNode{
			{ID: "node_start", Type: "event_start"},
			{ID: "node_cond", Type: "condition"},
			{ID: "exit_premium", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
			{ID: "exit_default", Type: "exit", Params: map[string]interface{}{"status": "suppressed"}},
		}
		edges := []domain.IREdge{
			{ID: "edge1", SourceID: "node_start", TargetID: "node_cond"},
			{ID: "edge2", SourceID: "node_cond", TargetID: "exit_premium", ConditionExpression: "tier == 'premium'"},
			{ID: "edge3", SourceID: "node_cond", TargetID: "exit_default", ConditionExpression: "true"},
		}
		ir := helperCreateAndRegisterIR(act, nodes, edges)

		input := workflows.CompiledJourneyInput{
			SchemaVersion: domain.DefaultSchemaVersion,
			WorkflowID:    "wf-cond-prem",
			RunID:         "run-cond-prem",
			TenantID:      "tenant-test",
			ContentHash:   ir.ContentHash,
			InputPayload: map[string]interface{}{
				"tier": "premium",
			},
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
		if res.CurrentNodeID != "exit_premium" {
			t.Errorf("expected exit_premium node, got %s", res.CurrentNodeID)
		}
		if res.VisitCounts["node_cond"] != 1 {
			t.Errorf("expected visit count 1 for node_cond, got %d", res.VisitCounts["node_cond"])
		}
	})

	// Case B: Free tier takes default branch to exit_default (suppressed)
	t.Run("DefaultBranch_Free", func(t *testing.T) {
		_, env, act := setupTestEnvironment(t)

		nodes := []domain.IRNode{
			{ID: "node_start", Type: "event_start"},
			{ID: "node_cond", Type: "condition"},
			{ID: "exit_premium", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
			{ID: "exit_default", Type: "exit", Params: map[string]interface{}{"status": "suppressed"}},
		}
		edges := []domain.IREdge{
			{ID: "edge1", SourceID: "node_start", TargetID: "node_cond"},
			{ID: "edge2", SourceID: "node_cond", TargetID: "exit_premium", ConditionExpression: "tier == 'premium'"},
			{ID: "edge3", SourceID: "node_cond", TargetID: "exit_default", ConditionExpression: "true"},
		}
		ir := helperCreateAndRegisterIR(act, nodes, edges)

		input := workflows.CompiledJourneyInput{
			SchemaVersion: domain.DefaultSchemaVersion,
			WorkflowID:    "wf-cond-free",
			RunID:         "run-cond-free",
			TenantID:      "tenant-test",
			ContentHash:   ir.ContentHash,
			InputPayload: map[string]interface{}{
				"tier": "free",
			},
		}

		env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

		if !env.IsWorkflowCompleted() {
			t.Fatalf("expected workflow to complete")
		}

		var res workflows.CompiledJourneyResult
		if err := env.GetWorkflowResult(&res); err != nil {
			t.Fatalf("failed to get result: %v", err)
		}

		if res.Status != workflows.StatusSuppressed {
			t.Errorf("expected status suppressed, got %s", res.Status)
		}
		if res.CurrentNodeID != "exit_default" {
			t.Errorf("expected exit_default node, got %s", res.CurrentNodeID)
		}
	})
}

// Test 3: Delay Nodes (Time skipping in Temporal testsuite)
func TestCompiledJourneyWorkflow_DelayNode(t *testing.T) {
	_, env, act := setupTestEnvironment(t)

	nodes := []domain.IRNode{
		{ID: "start_node", Type: "event_start"},
		{ID: "delay_node", Type: "delay", Params: map[string]interface{}{"duration_seconds": 3600}},
		{ID: "exit_node", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
	}
	edges := []domain.IREdge{
		{ID: "edge1", SourceID: "start_node", TargetID: "delay_node"},
		{ID: "edge2", SourceID: "delay_node", TargetID: "exit_node"},
	}
	ir := helperCreateAndRegisterIR(act, nodes, edges)

	input := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-delay-1",
		RunID:         "run-delay-1",
		TenantID:      "tenant-test",
		ContentHash:   ir.ContentHash,
	}

	env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("expected workflow to complete with time skipping")
	}

	var res workflows.CompiledJourneyResult
	if err := env.GetWorkflowResult(&res); err != nil {
		t.Fatalf("failed to get result: %v", err)
	}

	if res.Status != workflows.StatusSucceeded {
		t.Errorf("expected status succeeded, got %s", res.Status)
	}
	if res.NodeOutputs["delay_node"] != "delayed_3600s" {
		t.Errorf("expected delay_node output delayed_3600s, got %v", res.NodeOutputs["delay_node"])
	}
}

// Test 4: Cancellation Signal during execution / delay
func TestCompiledJourneyWorkflow_Cancellation(t *testing.T) {
	_, env, act := setupTestEnvironment(t)

	nodes := []domain.IRNode{
		{ID: "start_node", Type: "event_start"},
		{ID: "delay_node", Type: "delay", Params: map[string]interface{}{"duration_seconds": 100}},
		{ID: "exit_node", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
	}
	edges := []domain.IREdge{
		{ID: "edge1", SourceID: "start_node", TargetID: "delay_node"},
		{ID: "edge2", SourceID: "delay_node", TargetID: "exit_node"},
	}
	ir := helperCreateAndRegisterIR(act, nodes, edges)

	// Send cancel signal at 50ms into workflow execution
	env.RegisterDelayedCallback(func() {
		sig := domain.WorkflowSignal{
			SchemaVersion: domain.DefaultSchemaVersion,
			SignalName:    workflows.SignalNameCancel,
			RunID:         "run-cancel-1",
		}
		env.SignalWorkflow(workflows.SignalNameCancel, sig)
	}, time.Millisecond*50)

	input := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-cancel-1",
		RunID:         "run-cancel-1",
		TenantID:      "tenant-test",
		ContentHash:   ir.ContentHash,
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
}

// Test 5: Failure Node / Missing Node
func TestCompiledJourneyWorkflow_Failure(t *testing.T) {
	_, env, act := setupTestEnvironment(t)

	nodes := []domain.IRNode{
		{ID: "start_node", Type: "event_start"},
	}
	edges := []domain.IREdge{
		{ID: "edge1", SourceID: "start_node", TargetID: "non_existent_node"},
	}
	ir := helperCreateAndRegisterIR(act, nodes, edges)

	input := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-fail-1",
		RunID:         "run-fail-1",
		TenantID:      "tenant-test",
		ContentHash:   ir.ContentHash,
	}

	env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("expected workflow to complete with failed status")
	}

	if err := env.GetWorkflowError(); err == nil {
		t.Fatalf("expected workflow error for failed graph execution, got nil")
	}
}

// Test 6: Verification that workflow state contains no sensitive prohibited fields
func TestCompiledJourneyWorkflow_StateSafety_NoProhibitedFields(t *testing.T) {
	// 1. Struct Tag Inspection
	stateType := reflect.TypeOf(workflows.CompiledJourneyState{})
	for i := 0; i < stateType.NumField(); i++ {
		field := stateType.Field(i)
		histTag := field.Tag.Get("temporal_history")
		if histTag == "prohibited" {
			t.Fatalf("CompiledJourneyState contains prohibited field: %s", field.Name)
		}
		if histTag != "allowed" {
			t.Errorf("CompiledJourneyState field %s should have temporal_history:\"allowed\", got %q", field.Name, histTag)
		}
	}

	// 2. Query State Verification during Execution
	_, env, act := setupTestEnvironment(t)

	nodes := []domain.IRNode{
		{ID: "start_node", Type: "event_start"},
		{ID: "exit_node", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
	}
	edges := []domain.IREdge{
		{ID: "edge1", SourceID: "start_node", TargetID: "exit_node"},
	}
	ir := helperCreateAndRegisterIR(act, nodes, edges)

	input := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-safety-1",
		RunID:         "run-safety-1",
		TenantID:      "tenant-test",
		ContentHash:   ir.ContentHash,
		InputPayload: map[string]interface{}{
			"sensitive_ssn": "123-45-6789", // sensitive input payload
			"email":         "secret@example.com",
		},
	}

	env.RegisterDelayedCallback(func() {
		res, err := env.QueryWorkflow(workflows.QueryTypeState)
		if err != nil {
			t.Errorf("failed to query workflow state: %v", err)
			return
		}
		var state workflows.CompiledJourneyState
		if err := res.Get(&state); err != nil {
			t.Errorf("failed to decode state: %v", err)
			return
		}

		// Verify state struct contains only allowed node outputs and no prohibited fields
		stateValue := reflect.ValueOf(state)
		stateType := stateValue.Type()
		for i := 0; i < stateType.NumField(); i++ {
			field := stateType.Field(i)
			if field.Tag.Get("temporal_history") == "prohibited" {
				t.Errorf("prohibited field %s present in workflow state query output", field.Name)
			}
		}
	}, time.Millisecond*5)

	env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("expected workflow to complete")
	}
}
