package workflows_test

import (
	"context"
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/activities"
	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/experiments"
	"github.com/validated-pattern/journey-platform/internal/outcomes"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
	"github.com/validated-pattern/journey-platform/internal/workflows"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func setupNodesTestEnvironment(t *testing.T) (*testsuite.WorkflowTestSuite, *testsuite.TestWorkflowEnvironment, *activities.Activities, postgres.Repository) {
	s := &testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()
	env.SetDataConverter(workflows.NewIsolatedDataConverter(nil))

	repo := postgres.NewMemoryRepository()
	asgSvc := experiments.NewAssignmentService(repo)
	expSvc := outcomes.NewExposureService(repo)
	gw := activities.NewActionGateway()

	act := activities.NewActivities().
		WithAssignmentService(asgSvc).
		WithExposureService(expSvc).
		WithActionGateway(gw)

	workflows.RegisterAllActivities(env, act)
	return s, env, act, repo
}

// Test 1: Sticky production assignment retry & exposure recording
func TestExperimentNode_StickyProductionAssignment(t *testing.T) {
	_, env, act, repo := setupNodesTestEnvironment(t)

	nodes := []domain.IRNode{
		{ID: "start_node", Type: "event_start"},
		{ID: "exp_node", Type: "experiment", Params: map[string]interface{}{"experiment_id": "exp-prod-sticky", "randomization_unit": "user_id"}},
		{ID: "exit_treatment", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
		{ID: "exit_control", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
		{ID: "exit_ineligible", Type: "exit", Params: map[string]interface{}{"status": "suppressed"}},
	}
	edges := []domain.IREdge{
		{ID: "e1", SourceID: "start_node", TargetID: "exp_node"},
		{ID: "e2", SourceID: "exp_node", TargetID: "exit_treatment", ConditionExpression: "treatment"},
		{ID: "e3", SourceID: "exp_node", TargetID: "exit_control", ConditionExpression: "control"},
		{ID: "e4", SourceID: "exp_node", TargetID: "exit_ineligible", ConditionExpression: "ineligible"},
	}
	ir := helperCreateAndRegisterIR(act, nodes, edges)

	input := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-exp-sticky-1",
		RunID:         "run-exp-sticky-1",
		TenantID:      "tenant-prod",
		ContentHash:   ir.ContentHash,
		ExecutionMode: workflows.ExecutionModeProduction,
		InputPayload: map[string]interface{}{
			"user_id": "user-sticky-999",
		},
	}

	env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)
	if !env.IsWorkflowCompleted() {
		t.Fatalf("expected workflow to complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("unexpected workflow error: %v", err)
	}

	var res workflows.CompiledJourneyResult
	if err := env.GetWorkflowResult(&res); err != nil {
		t.Fatalf("failed to get result: %v", err)
	}

	if res.Status != workflows.StatusSucceeded {
		t.Errorf("expected succeeded status, got %s", res.Status)
	}
	assignedVariant := res.ExperimentContext["exp_node"]
	if assignedVariant == "" {
		t.Fatal("expected non-empty assigned variant")
	}

	// Verify sticky assignment recorded in repo
	asg, err := repo.GetAssignmentByExperimentSubject(context.Background(), "tenant-prod", "exp-prod-sticky", "user-sticky-999")
	if err != nil || asg == nil {
		t.Fatalf("expected sticky assignment record in repository, got err=%v", err)
	}
	if asg.VariantID != assignedVariant {
		t.Errorf("expected variant %s in repo, got %s", assignedVariant, asg.VariantID)
	}

	// Verify exposure record exists in repo
	expSvc := outcomes.NewExposureService(repo)
	expRec, err := expSvc.GetExposure(context.Background(), "tenant-prod", "exp-tenant-prod-exp-prod-sticky-user-sticky-999-1")
	if err != nil || expRec == nil {
		t.Fatalf("expected exposure record in repository, got err=%v", err)
	}
}

// Test 2: Cancellation after assignment but before exposure DOES NOT count exposure
func TestExperimentNode_CancellationBeforeExposure(t *testing.T) {
	s := &testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()
	env.SetDataConverter(workflows.NewIsolatedDataConverter(nil))

	repo := postgres.NewMemoryRepository()
	asgSvc := experiments.NewAssignmentService(repo)
	expSvc := outcomes.NewExposureService(repo)

	exposureRecorded := false

	act := activities.NewActivities().
		WithAssignmentService(asgSvc).
		WithExposureService(expSvc)

	env.RegisterActivity(act.ExecuteNode)
	env.RegisterActivity(act.EvaluateCondition)
	env.RegisterActivity(act.EmitOutcome)
	env.RegisterActivity(act.LoadCompiledIR)
	env.RegisterActivity(act.EmitLifecycleEvent)

	// Custom activity function for GetOrAssign that signals workflow cancellation before returning
	env.RegisterActivityWithOptions(func(ctx context.Context, req experiments.AssignmentRequest) (*experiments.AssignmentRecord, error) {
		res, err := act.GetOrAssign(ctx, req)
		env.SignalWorkflow("journey.signal.cancel", &domain.WorkflowSignal{SignalName: "journey.signal.cancel"})
		return res, err
	}, activity.RegisterOptions{Name: "GetOrAssign"})

	// Custom activity for RecordExposure to track if it's called
	env.RegisterActivityWithOptions(func(ctx context.Context, req outcomes.ExposureRequest) (*outcomes.ExposureRecord, error) {
		exposureRecorded = true
		return act.RecordExposure(ctx, req)
	}, activity.RegisterOptions{Name: "RecordExposure"})

	nodes := []domain.IRNode{
		{ID: "start_node", Type: "event_start"},
		{ID: "exp_node", Type: "experiment", Params: map[string]interface{}{"experiment_id": "exp-cancel-test", "randomization_unit": "user_id"}},
		{ID: "exit_treatment", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
	}
	edges := []domain.IREdge{
		{ID: "e1", SourceID: "start_node", TargetID: "exp_node"},
		{ID: "e2", SourceID: "exp_node", TargetID: "exit_treatment", ConditionExpression: "true"},
	}
	ir := helperCreateAndRegisterIR(act, nodes, edges)

	input := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-cancel-test",
		RunID:         "run-cancel-test",
		TenantID:      "tenant-prod",
		ContentHash:   ir.ContentHash,
		InputPayload: map[string]interface{}{
			"user_id": "user-cancel-123",
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

	if res.Status != workflows.StatusCancelled {
		t.Errorf("expected workflow status cancelled, got %s", res.Status)
	}

	if exposureRecorded {
		t.Errorf("expected exposure NOT to be recorded on cancellation before branch entry")
	}
}

// Test 3: Channel action node branch routing (success -> next, suppression -> suppressed branch, failure -> failure branch)
func TestChannelActionNode_BranchRouting(t *testing.T) {
	// Case 3a: Success -> next
	t.Run("SuccessRouting", func(t *testing.T) {
		_, env, act, _ := setupNodesTestEnvironment(t)

		nodes := []domain.IRNode{
			{ID: "start_node", Type: "event_start"},
			{ID: "email_node", Type: "email", Params: map[string]interface{}{"template_body": "Hello {{name}}"}},
			{ID: "exit_success", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
			{ID: "exit_suppressed", Type: "exit", Params: map[string]interface{}{"status": "suppressed"}},
			{ID: "exit_failure", Type: "exit", Params: map[string]interface{}{"status": "failed"}},
		}
		edges := []domain.IREdge{
			{ID: "e1", SourceID: "start_node", TargetID: "email_node"},
			{ID: "e2", SourceID: "email_node", TargetID: "exit_success", ConditionExpression: "accepted"},
			{ID: "e3", SourceID: "email_node", TargetID: "exit_suppressed", ConditionExpression: "suppressed"},
			{ID: "e4", SourceID: "email_node", TargetID: "exit_failure", ConditionExpression: "failure"},
		}
		ir := helperCreateAndRegisterIR(act, nodes, edges)

		input := workflows.CompiledJourneyInput{
			SchemaVersion: domain.DefaultSchemaVersion,
			WorkflowID:    "wf-action-succ",
			RunID:         "run-action-succ",
			TenantID:      "tenant-action",
			ContentHash:   ir.ContentHash,
			InputPayload: map[string]interface{}{
				"user_id": "user-succ",
				"email":   "user@example.com",
			},
		}

		env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

		var res workflows.CompiledJourneyResult
		if err := env.GetWorkflowResult(&res); err != nil {
			t.Fatalf("failed to get result: %v", err)
		}
		if res.Status != workflows.StatusSucceeded {
			t.Errorf("expected succeeded status, got %s", res.Status)
		}
		if res.CurrentNodeID != "exit_success" {
			t.Errorf("expected exit_success, got %s", res.CurrentNodeID)
		}
	})

	// Case 3b: Suppression -> suppressed branch
	t.Run("SuppressionRouting", func(t *testing.T) {
		s := &testsuite.WorkflowTestSuite{}
		env := s.NewTestWorkflowEnvironment()
		env.SetDataConverter(workflows.NewIsolatedDataConverter(nil))

		act := activities.NewActivities()
		env.RegisterActivity(act.ExecuteNode)
		env.RegisterActivity(act.EvaluateCondition)
		env.RegisterActivity(act.EmitOutcome)
		env.RegisterActivity(act.LoadCompiledIR)
		env.RegisterActivity(act.EmitLifecycleEvent)

		// Mock ExecuteActionGateway to return suppressed result
		env.RegisterActivityWithOptions(func(ctx context.Context, req activities.ActionRequest) (*activities.GatewayResult, error) {
			return &activities.GatewayResult{
				IdempotencyKey: req.IdempotencyKey(),
				Status:         activities.LedgerStatusSuppressed,
				ReasonCode:     activities.ReasonConsentWithdrawn,
				ExecutedAt:     time.Now().UTC(),
			}, nil
		}, activity.RegisterOptions{Name: "ExecuteActionGateway"})

		nodes := []domain.IRNode{
			{ID: "start_node", Type: "event_start"},
			{ID: "sms_node", Type: "sms"},
			{ID: "exit_success", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
			{ID: "exit_suppressed", Type: "exit", Params: map[string]interface{}{"status": "suppressed"}},
			{ID: "exit_failure", Type: "exit", Params: map[string]interface{}{"status": "failed"}},
		}
		edges := []domain.IREdge{
			{ID: "e1", SourceID: "start_node", TargetID: "sms_node"},
			{ID: "e2", SourceID: "sms_node", TargetID: "exit_success", ConditionExpression: "accepted"},
			{ID: "e3", SourceID: "sms_node", TargetID: "exit_suppressed", ConditionExpression: "suppressed"},
			{ID: "e4", SourceID: "sms_node", TargetID: "exit_failure", ConditionExpression: "failure"},
		}
		ir := helperCreateAndRegisterIR(act, nodes, edges)

		input := workflows.CompiledJourneyInput{
			SchemaVersion: domain.DefaultSchemaVersion,
			WorkflowID:    "wf-action-supp",
			RunID:         "run-action-supp",
			TenantID:      "tenant-action",
			ContentHash:   ir.ContentHash,
			InputPayload: map[string]interface{}{
				"user_id": "user-supp",
			},
		}

		env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

		var res workflows.CompiledJourneyResult
		if err := env.GetWorkflowResult(&res); err != nil {
			t.Fatalf("failed to get result: %v", err)
		}
		if res.Status != workflows.StatusSuppressed {
			t.Errorf("expected suppressed status, got %s", res.Status)
		}
		if res.CurrentNodeID != "exit_suppressed" {
			t.Errorf("expected exit_suppressed, got %s", res.CurrentNodeID)
		}
	})

	// Case 3c: Failure -> failure branch
	t.Run("FailureRouting", func(t *testing.T) {
		s := &testsuite.WorkflowTestSuite{}
		env := s.NewTestWorkflowEnvironment()
		env.SetDataConverter(workflows.NewIsolatedDataConverter(nil))

		act := activities.NewActivities()
		env.RegisterActivity(act.ExecuteNode)
		env.RegisterActivity(act.EvaluateCondition)
		env.RegisterActivity(act.EmitOutcome)
		env.RegisterActivity(act.LoadCompiledIR)
		env.RegisterActivity(act.EmitLifecycleEvent)

		// Mock ExecuteActionGateway to return permanent failure
		env.RegisterActivityWithOptions(func(ctx context.Context, req activities.ActionRequest) (*activities.GatewayResult, error) {
			return &activities.GatewayResult{
				IdempotencyKey: req.IdempotencyKey(),
				Status:         activities.LedgerStatusPermanentFailure,
				ErrorMessage:   "provider timeout",
				ExecutedAt:     time.Now().UTC(),
			}, nil
		}, activity.RegisterOptions{Name: "ExecuteActionGateway"})

		nodes := []domain.IRNode{
			{ID: "start_node", Type: "event_start"},
			{ID: "push_node", Type: "push"},
			{ID: "exit_success", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
			{ID: "exit_failure", Type: "exit", Params: map[string]interface{}{"status": "failed"}},
		}
		edges := []domain.IREdge{
			{ID: "e1", SourceID: "start_node", TargetID: "push_node"},
			{ID: "e2", SourceID: "push_node", TargetID: "exit_success", ConditionExpression: "accepted"},
			{ID: "e3", SourceID: "push_node", TargetID: "exit_failure", ConditionExpression: "failure"},
		}
		ir := helperCreateAndRegisterIR(act, nodes, edges)

		input := workflows.CompiledJourneyInput{
			SchemaVersion: domain.DefaultSchemaVersion,
			WorkflowID:    "wf-action-fail",
			RunID:         "run-action-fail",
			TenantID:      "tenant-action",
			ContentHash:   ir.ContentHash,
			InputPayload: map[string]interface{}{
				"user_id": "user-fail",
			},
		}

		env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

		var res workflows.CompiledJourneyResult
		if err := env.GetWorkflowResult(&res); err != nil {
			t.Fatalf("failed to get result: %v", err)
		}
		if res.Status != workflows.StatusFailed {
			t.Errorf("expected failed status, got %s", res.Status)
		}
		if res.CurrentNodeID != "exit_failure" {
			t.Errorf("expected exit_failure, got %s", res.CurrentNodeID)
		}
	})
}

// Test 4: Test mode history propagation referencing test ledgers and queues
func TestExecutionMode_TestModePropagation(t *testing.T) {
	s := &testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()
	env.SetDataConverter(workflows.NewIsolatedDataConverter(nil))

	act := activities.NewActivities()
	env.RegisterActivity(act.ExecuteNode)
	env.RegisterActivity(act.EvaluateCondition)
	env.RegisterActivity(act.EmitOutcome)
	env.RegisterActivity(act.LoadCompiledIR)
	env.RegisterActivity(act.EmitLifecycleEvent)

	var capturedReq activities.ActionRequest

	env.RegisterActivityWithOptions(func(ctx context.Context, req activities.ActionRequest) (*activities.GatewayResult, error) {
		capturedReq = req
		return &activities.GatewayResult{
			IdempotencyKey: req.IdempotencyKey(),
			Status:         activities.LedgerStatusAccepted,
			ExecutedAt:     time.Now().UTC(),
		}, nil
	}, activity.RegisterOptions{Name: "ExecuteActionGateway"})

	nodes := []domain.IRNode{
		{ID: "start_node", Type: "event_start"},
		{ID: "webhook_node", Type: "webhook", Params: map[string]interface{}{"url": "https://example.com/hook"}},
		{ID: "exit_node", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
	}
	edges := []domain.IREdge{
		{ID: "e1", SourceID: "start_node", TargetID: "webhook_node"},
		{ID: "e2", SourceID: "webhook_node", TargetID: "exit_node"},
	}
	ir := helperCreateAndRegisterIR(act, nodes, edges)

	input := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-test-mode-1",
		RunID:         "run-test-mode-1",
		TenantID:      "tenant-test",
		ContentHash:   ir.ContentHash,
		ExecutionMode: workflows.ExecutionModeTest,
		InputPayload: map[string]interface{}{
			"test_run_id": "test-run-ledger-999",
			"user_id":     "test-user-1",
		},
	}

	env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("expected workflow to complete")
	}

	if capturedReq.ExecutionMode != activities.ExecutionModeTest {
		t.Errorf("expected execution mode test, got %s", capturedReq.ExecutionMode)
	}
	if capturedReq.TestRunID != "test-run-ledger-999" {
		t.Errorf("expected test_run_id test-run-ledger-999, got %s", capturedReq.TestRunID)
	}
}

// Test 5: Missing randomization unit falling back to ineligible branch
func TestExperimentNode_MissingRandomizationUnitFallback(t *testing.T) {
	_, env, act, _ := setupNodesTestEnvironment(t)

	nodes := []domain.IRNode{
		{ID: "start_node", Type: "event_start"},
		{ID: "exp_node", Type: "experiment", Params: map[string]interface{}{"randomization_unit": "user_id"}},
		{ID: "exit_treatment", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
		{ID: "exit_ineligible", Type: "exit", Params: map[string]interface{}{"status": "suppressed"}},
	}
	edges := []domain.IREdge{
		{ID: "e1", SourceID: "start_node", TargetID: "exp_node"},
		{ID: "e2", SourceID: "exp_node", TargetID: "exit_treatment", ConditionExpression: "treatment"},
		{ID: "e3", SourceID: "exp_node", TargetID: "exit_ineligible", ConditionExpression: "ineligible"},
	}
	ir := helperCreateAndRegisterIR(act, nodes, edges)

	// InputPayload missing user_id!
	input := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-missing-unit-1",
		RunID:         "run-missing-unit-1",
		TenantID:      "tenant-prod",
		ContentHash:   ir.ContentHash,
		ExecutionMode: workflows.ExecutionModeProduction,
		InputPayload:  map[string]interface{}{},
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
		t.Errorf("expected suppressed status for ineligible fallback, got %s", res.Status)
	}
	if res.CurrentNodeID != "exit_ineligible" {
		t.Errorf("expected exit_ineligible, got %s", res.CurrentNodeID)
	}
	if res.NodeOutputs["exp_node"] != "ineligible" {
		t.Errorf("expected node output ineligible, got %v", res.NodeOutputs["exp_node"])
	}
}
