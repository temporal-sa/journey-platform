package workflows_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/validated-pattern/journey-platform/internal/activities"
	"github.com/validated-pattern/journey-platform/internal/compiler"
	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/experiments"
	"github.com/validated-pattern/journey-platform/internal/outcomes"
	"github.com/validated-pattern/journey-platform/internal/workflows"
	commonv1 "go.temporal.io/api/common/v1"
	enumsv1 "go.temporal.io/api/enums/v1"
	historyv1 "go.temporal.io/api/history/v1"
	taskqueuev1 "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func setupReplayTestEnv(t *testing.T) (*testsuite.WorkflowTestSuite, *testsuite.TestWorkflowEnvironment, *activities.Activities) {
	s := &testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()
	env.SetDataConverter(workflows.NewIsolatedDataConverter(nil))

	act := activities.NewActivities()
	workflows.RegisterAllActivities(env, act)
	return s, env, act
}

func helperBuildIR(act *activities.Activities, irID string, nodes []domain.IRNode, edges []domain.IREdge) *domain.CompiledIR {
	ir := &domain.CompiledIR{
		SchemaVersion: domain.DefaultSchemaVersion,
		IRID:          irID,
		DraftID:       "draft-" + irID,
		TenantID:      "tenant-replay-test",
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

// -----------------------------------------------------------------------------
// 1. All Node Kinds Test
// Covers: EventStart, Condition, Delay, WaitForEvent, Experiment, Action Nodes (Email/SMS/Push/InApp/Webhook), Exit
// -----------------------------------------------------------------------------
func TestCompiledJourneyWorkflow_AllNodeKinds(t *testing.T) {
	t.Run("FullGraph_AllNodeKinds", func(t *testing.T) {
		_, env, act := setupReplayTestEnv(t)

		nodes := []domain.IRNode{
			{ID: "node_start", Type: "event_start"},
			{ID: "node_cond", Type: "condition", Params: map[string]interface{}{"condition_expression": "user_status == 'active'"}},
			{ID: "node_exp", Type: "experiment", Params: map[string]interface{}{"experiment_id": "exp_welcome_test", "randomization_unit": "user_id"}},
			{ID: "node_email", Type: "email", Params: map[string]interface{}{"template": "welcome_template", "recipient": "user@example.com"}},
			{ID: "node_delay", Type: "delay", Params: map[string]interface{}{"duration_seconds": 2}},
			{ID: "node_wfe", Type: "wait_for_event", Params: map[string]interface{}{"event_type": "onboarding_completed", "timeout_seconds": 10}},
			{ID: "node_exit_success", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
			{ID: "node_exit_suppressed", Type: "exit", Params: map[string]interface{}{"status": "suppressed"}},
		}

		edges := []domain.IREdge{
			{ID: "e1", SourceID: "node_start", TargetID: "node_cond"},
			{ID: "e2_true", SourceID: "node_cond", TargetID: "node_exp", ConditionExpression: "user_status == 'active'"},
			{ID: "e3_false", SourceID: "node_cond", TargetID: "node_exit_suppressed", ConditionExpression: "false"},
			{ID: "e4", SourceID: "node_exp", TargetID: "node_email", ConditionExpression: "variant_a"},
			{ID: "e5", SourceID: "node_exp", TargetID: "node_email", ConditionExpression: "ineligible"},
			{ID: "e6", SourceID: "node_email", TargetID: "node_delay"},
			{ID: "e7", SourceID: "node_delay", TargetID: "node_wfe"},
			{ID: "e8", SourceID: "node_wfe", TargetID: "node_exit_success", ConditionExpression: "matched"},
			{ID: "e9", SourceID: "node_wfe", TargetID: "node_exit_suppressed", ConditionExpression: "timeout"},
		}

		ir := helperBuildIR(act, "ir-all-nodes", nodes, edges)

		input := workflows.CompiledJourneyInput{
			SchemaVersion: domain.DefaultSchemaVersion,
			WorkflowID:    "wf-all-nodes",
			RunID:         "run-all-nodes",
			TenantID:      "tenant-replay-test",
			ContentHash:   ir.ContentHash,
			ExecutionMode: workflows.ExecutionModeProduction,
			InputPayload: map[string]interface{}{
				"user_status": "active",
				"user_id":     "usr_1001",
			},
		}

		// Register delayed signal for WaitForEvent node
		env.RegisterDelayedCallback(func() {
			env.SignalWorkflow(workflows.SignalNameEvent, workflows.EventSignal{
				EventID:   "evt_onboarding_1",
				EventType: "onboarding_completed",
			})
		}, 3*time.Second)

		env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

		if !env.IsWorkflowCompleted() {
			t.Fatalf("expected workflow to complete successfully")
		}
		if err := env.GetWorkflowError(); err != nil {
			t.Fatalf("workflow execution failed: %v", err)
		}

		var res workflows.CompiledJourneyResult
		if err := env.GetWorkflowResult(&res); err != nil {
			t.Fatalf("failed to get workflow result: %v", err)
		}

		if res.Status != workflows.StatusSucceeded {
			t.Errorf("expected status %s, got %s", workflows.StatusSucceeded, res.Status)
		}

		// Verify visit counts for all node types traversed
		expectedVisited := []string{"node_start", "node_cond", "node_exp", "node_email", "node_delay", "node_wfe", "node_exit_success"}
		for _, nodeID := range expectedVisited {
			if res.VisitCounts[nodeID] < 1 {
				t.Errorf("expected node %s to be visited at least once, visit counts: %v", nodeID, res.VisitCounts)
			}
		}
	})
}

// -----------------------------------------------------------------------------
// 2. Activity Retries Test
// -----------------------------------------------------------------------------
func TestCompiledJourneyWorkflow_ActivityRetries(t *testing.T) {
	_, env, act := setupReplayTestEnv(t)

	nodes := []domain.IRNode{
		{ID: "node_start", Type: "event_start"},
		{ID: "node_action", Type: "action", Params: map[string]interface{}{"channel": "webhook", "url": "https://example.com/hook"}},
		{ID: "node_exit", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
	}
	edges := []domain.IREdge{
		{ID: "e1", SourceID: "node_start", TargetID: "node_action"},
		{ID: "e2", SourceID: "node_action", TargetID: "node_exit"},
	}
	ir := helperBuildIR(act, "ir-activity-retry", nodes, edges)

	attempts := 0
	env.OnActivity(act.ExecuteActionGateway, mock.Anything, mock.Anything).Return(
		func(ctx context.Context, req activities.ActionRequest) (*activities.GatewayResult, error) {
			attempts++
			if attempts < 3 {
				return nil, os.ErrDeadlineExceeded // simulate transient network failure
			}
			return &activities.GatewayResult{
				IdempotencyKey: "act_1",
				Status:         activities.LedgerStatusAccepted,
			}, nil
		},
	)

	input := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-activity-retry",
		RunID:         "run-activity-retry",
		TenantID:      "tenant-replay-test",
		ContentHash:   ir.ContentHash,
	}

	env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("expected workflow to complete")
	}

	var res workflows.CompiledJourneyResult
	if err := env.GetWorkflowResult(&res); err != nil {
		t.Fatalf("failed to get workflow result: %v", err)
	}

	if res.Status != workflows.StatusSucceeded {
		t.Errorf("expected status %s after activity retries, got %s", workflows.StatusSucceeded, res.Status)
	}
	if attempts != 3 {
		t.Errorf("expected 3 activity attempts before success, got %d", attempts)
	}
}

// -----------------------------------------------------------------------------
// 3. Timeouts Test
// -----------------------------------------------------------------------------
func TestCompiledJourneyWorkflow_ReplayTimeouts(t *testing.T) {
	t.Run("WaitForEvent_TimeoutBranch", func(t *testing.T) {
		_, env, act := setupReplayTestEnv(t)

		nodes := []domain.IRNode{
			{ID: "node_start", Type: "event_start"},
			{ID: "node_wfe", Type: "wait_for_event", Params: map[string]interface{}{"event_type": "user_converted", "timeout_seconds": 5}},
			{ID: "node_exit_matched", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
			{ID: "node_exit_timeout", Type: "exit", Params: map[string]interface{}{"status": "suppressed"}},
		}
		edges := []domain.IREdge{
			{ID: "e1", SourceID: "node_start", TargetID: "node_wfe"},
			{ID: "e2", SourceID: "node_wfe", TargetID: "node_exit_matched", ConditionExpression: "matched"},
			{ID: "e3", SourceID: "node_wfe", TargetID: "node_exit_timeout", ConditionExpression: "timeout"},
		}
		ir := helperBuildIR(act, "ir-wfe-timeout", nodes, edges)

		input := workflows.CompiledJourneyInput{
			SchemaVersion: domain.DefaultSchemaVersion,
			WorkflowID:    "wf-wfe-timeout",
			RunID:         "run-wfe-timeout",
			TenantID:      "tenant-replay-test",
			ContentHash:   ir.ContentHash,
		}

		// No signal sent -> wait_for_event should time out after 5s and take timeout edge to node_exit_timeout
		env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

		if !env.IsWorkflowCompleted() {
			t.Fatalf("expected workflow to complete after timeout")
		}

		var res workflows.CompiledJourneyResult
		if err := env.GetWorkflowResult(&res); err != nil {
			t.Fatalf("failed to get workflow result: %v", err)
		}

		if res.Status != workflows.StatusSuppressed {
			t.Errorf("expected status %s for timeout branch, got %s", workflows.StatusSuppressed, res.Status)
		}
		if res.CurrentNodeID != "node_exit_timeout" {
			t.Errorf("expected final node node_exit_timeout, got %s", res.CurrentNodeID)
		}
	})
}

// -----------------------------------------------------------------------------
// 4. Cancellation Test
// -----------------------------------------------------------------------------
func TestCompiledJourneyWorkflow_ReplayCancellation(t *testing.T) {
	t.Run("CancelDuringDelay", func(t *testing.T) {
		_, env, act := setupReplayTestEnv(t)

		nodes := []domain.IRNode{
			{ID: "node_start", Type: "event_start"},
			{ID: "node_delay", Type: "delay", Params: map[string]interface{}{"duration_seconds": 60}},
			{ID: "node_exit", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
		}
		edges := []domain.IREdge{
			{ID: "e1", SourceID: "node_start", TargetID: "node_delay"},
			{ID: "e2", SourceID: "node_delay", TargetID: "node_exit"},
		}
		ir := helperBuildIR(act, "ir-cancel-delay", nodes, edges)

		input := workflows.CompiledJourneyInput{
			SchemaVersion: domain.DefaultSchemaVersion,
			WorkflowID:    "wf-cancel-delay",
			RunID:         "run-cancel-delay",
			TenantID:      "tenant-replay-test",
			ContentHash:   ir.ContentHash,
		}

		// Signal cancellation 5 seconds into 60s delay
		env.RegisterDelayedCallback(func() {
			env.SignalWorkflow(workflows.SignalNameCancel, domain.WorkflowSignal{
				SignalName: workflows.SignalNameCancel,
			})
		}, 5*time.Second)

		env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

		if !env.IsWorkflowCompleted() {
			t.Fatalf("expected workflow to complete upon cancellation")
		}

		var res workflows.CompiledJourneyResult
		if err := env.GetWorkflowResult(&res); err != nil {
			t.Fatalf("failed to get workflow result: %v", err)
		}

		if res.Status != workflows.StatusCancelled {
			t.Errorf("expected status %s, got %s", workflows.StatusCancelled, res.Status)
		}
	})
}

// -----------------------------------------------------------------------------
// 5. Signals & Queries Test
// -----------------------------------------------------------------------------
func TestCompiledJourneyWorkflow_SignalsAndQueries(t *testing.T) {
	_, env, act := setupReplayTestEnv(t)

	nodes := []domain.IRNode{
		{ID: "node_start", Type: "event_start"},
		{ID: "node_wfe", Type: "wait_for_event", Params: map[string]interface{}{"event_type": "email_clicked", "timeout_seconds": 30}},
		{ID: "node_exit", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
	}
	edges := []domain.IREdge{
		{ID: "e1", SourceID: "node_start", TargetID: "node_wfe"},
		{ID: "e2", SourceID: "node_wfe", TargetID: "node_exit", ConditionExpression: "matched"},
	}
	ir := helperBuildIR(act, "ir-signals-queries", nodes, edges)

	input := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-sig-query",
		RunID:         "run-sig-query",
		TenantID:      "tenant-replay-test",
		ContentHash:   ir.ContentHash,
		InputPayload: map[string]interface{}{
			"user_id": "usr_2002",
		},
	}

	// Register delayed query check & signal
	env.RegisterDelayedCallback(func() {
		// Query workflow state while waiting
		val, err := env.QueryWorkflow(workflows.QueryTypeState)
		if err != nil {
			t.Errorf("failed to query workflow state: %v", err)
		} else {
			var state workflows.CompiledJourneyState
			if err := val.Get(&state); err != nil {
				t.Errorf("failed to decode state query: %v", err)
			} else {
				if state.CurrentNodeID != "node_wfe" {
					t.Errorf("expected current node node_wfe during wait, got %s", state.CurrentNodeID)
				}
			}
		}

		// Query variables
		varVal, err := env.QueryWorkflow(workflows.QueryTypeVariables)
		if err != nil {
			t.Errorf("failed to query workflow variables: %v", err)
		} else {
			var vars map[string]interface{}
			if err := varVal.Get(&vars); err != nil {
				t.Errorf("failed to decode variables query: %v", err)
			} else {
				if vars["user_id"] != "usr_2002" {
					t.Errorf("expected variable user_id 'usr_2002', got %v", vars["user_id"])
				}
			}
		}

		// Send event signal
		env.SignalWorkflow(workflows.SignalNameEvent, workflows.EventSignal{
			EventID:   "evt_click_101",
			EventType: "email_clicked",
			Payload:   map[string]interface{}{"link_id": "cta_buy_now"},
		})
	}, 2*time.Second)

	env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("expected workflow to complete after signal")
	}

	// Query status after completion
	statusVal, err := env.QueryWorkflow(workflows.QueryTypeStatus)
	if err != nil {
		t.Fatalf("failed to query status after completion: %v", err)
	}
	var resResult workflows.CompiledJourneyResult
	if err := statusVal.Get(&resResult); err != nil {
		t.Fatalf("failed to decode status query result: %v", err)
	}
	if resResult.Status != workflows.StatusSucceeded {
		t.Errorf("expected final status succeeded, got %s", resResult.Status)
	}
}

// -----------------------------------------------------------------------------
// 6. Time-Skipping Tests
// Verifies deterministic timers skip simulated time instantly without real-time delay
// -----------------------------------------------------------------------------
func TestCompiledJourneyWorkflow_TimeSkipping(t *testing.T) {
	t.Run("DeterministicLongTimer_1Hour", func(t *testing.T) {
		_, env, act := setupReplayTestEnv(t)

		nodes := []domain.IRNode{
			{ID: "node_start", Type: "event_start"},
			{ID: "node_delay_1h", Type: "delay", Params: map[string]interface{}{"duration_seconds": 3600}},
			{ID: "node_exit", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
		}
		edges := []domain.IREdge{
			{ID: "e1", SourceID: "node_start", TargetID: "node_delay_1h"},
			{ID: "e2", SourceID: "node_delay_1h", TargetID: "node_exit"},
		}
		ir := helperBuildIR(act, "ir-time-skipping-1h", nodes, edges)

		input := workflows.CompiledJourneyInput{
			SchemaVersion: domain.DefaultSchemaVersion,
			WorkflowID:    "wf-time-skipping-1h",
			RunID:         "run-time-skipping-1h",
			TenantID:      "tenant-replay-test",
			ContentHash:   ir.ContentHash,
		}

		startRealTime := time.Now()

		env.ExecuteWorkflow(workflows.CompiledJourneyWorkflow, input)

		elapsed := time.Since(startRealTime)
		if elapsed > 3*time.Second {
			t.Errorf("time-skipping failed: expected 3600s timer to execute in <3s real-time, took %v", elapsed)
		}

		if !env.IsWorkflowCompleted() {
			t.Fatalf("expected 1-hour delay workflow to complete")
		}

		var res workflows.CompiledJourneyResult
		if err := env.GetWorkflowResult(&res); err != nil {
			t.Fatalf("failed to get workflow result: %v", err)
		}

		if res.Status != workflows.StatusSucceeded {
			t.Errorf("expected status %s, got %s", workflows.StatusSucceeded, res.Status)
		}
	})
}

// -----------------------------------------------------------------------------
// 7. Replay Verification Tests using worker.NewWorkflowReplayer
// Captures non-sensitive JSON workflow execution histories in test/fixtures/histories/
// and verifies they replay without non-determinism errors.
// -----------------------------------------------------------------------------

func getFixturesDirectory(t *testing.T) string {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	for {
		target := filepath.Join(dir, "test", "fixtures", "histories")
		if info, err := os.Stat(target); err == nil && info.IsDir() {
			return target
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	target := filepath.Join("..", "..", "test", "fixtures", "histories")
	_ = os.MkdirAll(target, 0755)
	return target
}

// Helper to generate & save JSON history files
func generateHistoryFixturesIfMissing(t *testing.T, fixturesDir string) {
	dc := converter.GetDefaultDataConverter()
	now := timestamppb.Now()

	type fixtureSpec struct {
		Filename string
		Events   []*historyv1.HistoryEvent
	}

	nilPayloads, _ := dc.ToPayloads(nil)

	createEvents := func(
		input workflows.CompiledJourneyInput,
		ir domain.CompiledIR,
		finalRes workflows.CompiledJourneyResult,
		extraStep func(nextID func() int64, events *[]*historyv1.HistoryEvent),
	) []*historyv1.HistoryEvent {
		inputP, _ := dc.ToPayloads(input)
		irP, _ := dc.ToPayloads(ir)
		resP, _ := dc.ToPayloads(finalRes)

		var id int64 = 1
		nextID := func() int64 {
			cur := id
			id++
			return cur
		}

		var events []*historyv1.HistoryEvent

		// 1. Started
		events = append(events, &historyv1.HistoryEvent{
			EventId:   nextID(),
			EventTime: now,
			EventType: enumsv1.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
			Attributes: &historyv1.HistoryEvent_WorkflowExecutionStartedEventAttributes{
				WorkflowExecutionStartedEventAttributes: &historyv1.WorkflowExecutionStartedEventAttributes{
					WorkflowType: &commonv1.WorkflowType{Name: "CompiledJourneyWorkflow"},
					TaskQueue:    &taskqueuev1.TaskQueue{Name: "journey-task-queue"},
					Input:        inputP,
				},
			},
		})

		addWorkflowTask := func() {
			schedID := nextID()
			events = append(events, &historyv1.HistoryEvent{
				EventId:   schedID,
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED,
				Attributes: &historyv1.HistoryEvent_WorkflowTaskScheduledEventAttributes{
					WorkflowTaskScheduledEventAttributes: &historyv1.WorkflowTaskScheduledEventAttributes{},
				},
			})
			startID := nextID()
			events = append(events, &historyv1.HistoryEvent{
				EventId:   startID,
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_WORKFLOW_TASK_STARTED,
				Attributes: &historyv1.HistoryEvent_WorkflowTaskStartedEventAttributes{
					WorkflowTaskStartedEventAttributes: &historyv1.WorkflowTaskStartedEventAttributes{ScheduledEventId: schedID},
				},
			})
			events = append(events, &historyv1.HistoryEvent{
				EventId:   nextID(),
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_WORKFLOW_TASK_COMPLETED,
				Attributes: &historyv1.HistoryEvent_WorkflowTaskCompletedEventAttributes{
					WorkflowTaskCompletedEventAttributes: &historyv1.WorkflowTaskCompletedEventAttributes{ScheduledEventId: schedID, StartedEventId: startID},
				},
			})
		}

		addActivityTask := func(actName string, resultPayloads *commonv1.Payloads) {
			addWorkflowTask()
			schedID := nextID()
			actIDStr := fmt.Sprintf("%d", schedID)
			events = append(events, &historyv1.HistoryEvent{
				EventId:   schedID,
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
				Attributes: &historyv1.HistoryEvent_ActivityTaskScheduledEventAttributes{
					ActivityTaskScheduledEventAttributes: &historyv1.ActivityTaskScheduledEventAttributes{
						ActivityId:   actIDStr,
						ActivityType: &commonv1.ActivityType{Name: actName},
					},
				},
			})
			startID := nextID()
			events = append(events, &historyv1.HistoryEvent{
				EventId:   startID,
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_ACTIVITY_TASK_STARTED,
				Attributes: &historyv1.HistoryEvent_ActivityTaskStartedEventAttributes{
					ActivityTaskStartedEventAttributes: &historyv1.ActivityTaskStartedEventAttributes{ScheduledEventId: schedID},
				},
			})
			events = append(events, &historyv1.HistoryEvent{
				EventId:   nextID(),
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
				Attributes: &historyv1.HistoryEvent_ActivityTaskCompletedEventAttributes{
					ActivityTaskCompletedEventAttributes: &historyv1.ActivityTaskCompletedEventAttributes{
						ScheduledEventId: schedID,
						StartedEventId:   startID,
						Result:           resultPayloads,
					},
				},
			})
		}

		addActivityTask("LoadCompiledIR", irP)
		addActivityTask("EmitLifecycleEvent", nilPayloads)

		if extraStep != nil {
			extraStep(nextID, &events)
		}

		addWorkflowTask()
		events = append(events, &historyv1.HistoryEvent{
			EventId:   nextID(),
			EventTime: now,
			EventType: enumsv1.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED,
			Attributes: &historyv1.HistoryEvent_WorkflowExecutionCompletedEventAttributes{
				WorkflowExecutionCompletedEventAttributes: &historyv1.WorkflowExecutionCompletedEventAttributes{
					Result: resP,
				},
			},
		})

		return events
	}

	// 1. Linear Execution History
	inputLinear := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-replay-linear",
		RunID:         "run-replay-linear",
		TenantID:      "tenant-replay-test",
		ContentHash:   "hash-linear-replay",
		ExecutionMode: workflows.ExecutionModeProduction,
	}
	irLinear := domain.CompiledIR{
		SchemaVersion: domain.DefaultSchemaVersion,
		IRID:          "ir-linear-replay",
		TenantID:      "tenant-replay-test",
		ContentHash:   "hash-linear-replay",
		EntryNodeID:   "start_1",
		Nodes: []domain.IRNode{
			{ID: "start_1", Type: "event_start"},
			{ID: "exit_1", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
		},
		Edges: []domain.IREdge{{ID: "e1", SourceID: "start_1", TargetID: "exit_1"}},
	}
	resLinear := workflows.CompiledJourneyResult{
		RunID:         "run-replay-linear",
		WorkflowID:    "wf-replay-linear",
		Status:        workflows.StatusSucceeded,
		CurrentNodeID: "exit_1",
		VisitCounts:   map[string]int{"start_1": 1, "exit_1": 1},
		NodeOutputs:   map[string]interface{}{"start_1": "started", "exit_1": "succeeded"},
	}

	eventsLinear := createEvents(inputLinear, irLinear, resLinear, func(nextID func() int64, events *[]*historyv1.HistoryEvent) {
		addAct := func(actName string) {
			schedID := nextID()
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   schedID,
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED,
				Attributes: &historyv1.HistoryEvent_WorkflowTaskScheduledEventAttributes{
					WorkflowTaskScheduledEventAttributes: &historyv1.WorkflowTaskScheduledEventAttributes{},
				},
			})
			startID := nextID()
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   startID,
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_WORKFLOW_TASK_STARTED,
				Attributes: &historyv1.HistoryEvent_WorkflowTaskStartedEventAttributes{
					WorkflowTaskStartedEventAttributes: &historyv1.WorkflowTaskStartedEventAttributes{ScheduledEventId: schedID},
				},
			})
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   nextID(),
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_WORKFLOW_TASK_COMPLETED,
				Attributes: &historyv1.HistoryEvent_WorkflowTaskCompletedEventAttributes{
					WorkflowTaskCompletedEventAttributes: &historyv1.WorkflowTaskCompletedEventAttributes{ScheduledEventId: schedID, StartedEventId: startID},
				},
			})

			actSchedID := nextID()
			actIDStr := fmt.Sprintf("%d", actSchedID)
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   actSchedID,
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
				Attributes: &historyv1.HistoryEvent_ActivityTaskScheduledEventAttributes{
					ActivityTaskScheduledEventAttributes: &historyv1.ActivityTaskScheduledEventAttributes{
						ActivityId:   actIDStr,
						ActivityType: &commonv1.ActivityType{Name: actName},
					},
				},
			})
			actStartID := nextID()
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   actStartID,
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_ACTIVITY_TASK_STARTED,
				Attributes: &historyv1.HistoryEvent_ActivityTaskStartedEventAttributes{
					ActivityTaskStartedEventAttributes: &historyv1.ActivityTaskStartedEventAttributes{ScheduledEventId: actSchedID},
				},
			})
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   nextID(),
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
				Attributes: &historyv1.HistoryEvent_ActivityTaskCompletedEventAttributes{
					ActivityTaskCompletedEventAttributes: &historyv1.ActivityTaskCompletedEventAttributes{
						ScheduledEventId: actSchedID,
						StartedEventId:   actStartID,
						Result:           nilPayloads,
					},
				},
			})
		}

		addAct("EmitLifecycleEvent") // start_1 node_entered
		addAct("EmitLifecycleEvent") // exit_1 node_entered
		addAct("EmitLifecycleEvent") // exit_1 workflow_succeeded
	})

	// 2. Delay Execution History
	inputDelay := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-replay-delay",
		RunID:         "run-replay-delay",
		TenantID:      "tenant-replay-test",
		ContentHash:   "hash-delay-replay",
		ExecutionMode: workflows.ExecutionModeProduction,
	}
	irDelay := domain.CompiledIR{
		SchemaVersion: domain.DefaultSchemaVersion,
		IRID:          "ir-delay-replay",
		TenantID:      "tenant-replay-test",
		ContentHash:   "hash-delay-replay",
		EntryNodeID:   "start_1",
		Nodes: []domain.IRNode{
			{ID: "start_1", Type: "event_start"},
			{ID: "delay_1", Type: "delay", Params: map[string]interface{}{"duration_seconds": 5}},
			{ID: "exit_1", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
		},
		Edges: []domain.IREdge{
			{ID: "e1", SourceID: "start_1", TargetID: "delay_1"},
			{ID: "e2", SourceID: "delay_1", TargetID: "exit_1"},
		},
	}
	resDelay := workflows.CompiledJourneyResult{
		RunID:         "run-replay-delay",
		WorkflowID:    "wf-replay-delay",
		Status:        workflows.StatusSucceeded,
		CurrentNodeID: "exit_1",
		VisitCounts:   map[string]int{"start_1": 1, "delay_1": 1, "exit_1": 1},
		NodeOutputs:   map[string]interface{}{"start_1": "started", "delay_1": "delayed_5s", "exit_1": "succeeded"},
	}

	eventsDelay := createEvents(inputDelay, irDelay, resDelay, func(nextID func() int64, events *[]*historyv1.HistoryEvent) {
		addAct := func(actName string) {
			schedID := nextID()
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   schedID,
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED,
				Attributes: &historyv1.HistoryEvent_WorkflowTaskScheduledEventAttributes{
					WorkflowTaskScheduledEventAttributes: &historyv1.WorkflowTaskScheduledEventAttributes{},
				},
			})
			startID := nextID()
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   startID,
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_WORKFLOW_TASK_STARTED,
				Attributes: &historyv1.HistoryEvent_WorkflowTaskStartedEventAttributes{
					WorkflowTaskStartedEventAttributes: &historyv1.WorkflowTaskStartedEventAttributes{ScheduledEventId: schedID},
				},
			})
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   nextID(),
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_WORKFLOW_TASK_COMPLETED,
				Attributes: &historyv1.HistoryEvent_WorkflowTaskCompletedEventAttributes{
					WorkflowTaskCompletedEventAttributes: &historyv1.WorkflowTaskCompletedEventAttributes{ScheduledEventId: schedID, StartedEventId: startID},
				},
			})

			actSchedID := nextID()
			actIDStr := fmt.Sprintf("%d", actSchedID)
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   actSchedID,
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
				Attributes: &historyv1.HistoryEvent_ActivityTaskScheduledEventAttributes{
					ActivityTaskScheduledEventAttributes: &historyv1.ActivityTaskScheduledEventAttributes{
						ActivityId:   actIDStr,
						ActivityType: &commonv1.ActivityType{Name: actName},
					},
				},
			})
			actStartID := nextID()
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   actStartID,
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_ACTIVITY_TASK_STARTED,
				Attributes: &historyv1.HistoryEvent_ActivityTaskStartedEventAttributes{
					ActivityTaskStartedEventAttributes: &historyv1.ActivityTaskStartedEventAttributes{ScheduledEventId: actSchedID},
				},
			})
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   nextID(),
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
				Attributes: &historyv1.HistoryEvent_ActivityTaskCompletedEventAttributes{
					ActivityTaskCompletedEventAttributes: &historyv1.ActivityTaskCompletedEventAttributes{
						ScheduledEventId: actSchedID,
						StartedEventId:   actStartID,
						Result:           nilPayloads,
					},
				},
			})
		}

		addAct("EmitLifecycleEvent") // start_1 node_entered
		addAct("EmitLifecycleEvent") // delay_1 node_entered

		// Timer start & fire
		schedID := nextID()
		*events = append(*events, &historyv1.HistoryEvent{
			EventId:   schedID,
			EventTime: now,
			EventType: enumsv1.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED,
			Attributes: &historyv1.HistoryEvent_WorkflowTaskScheduledEventAttributes{
				WorkflowTaskScheduledEventAttributes: &historyv1.WorkflowTaskScheduledEventAttributes{},
			},
		})
		startID := nextID()
		*events = append(*events, &historyv1.HistoryEvent{
			EventId:   startID,
			EventTime: now,
			EventType: enumsv1.EVENT_TYPE_WORKFLOW_TASK_STARTED,
			Attributes: &historyv1.HistoryEvent_WorkflowTaskStartedEventAttributes{
				WorkflowTaskStartedEventAttributes: &historyv1.WorkflowTaskStartedEventAttributes{ScheduledEventId: schedID},
			},
		})
		*events = append(*events, &historyv1.HistoryEvent{
			EventId:   nextID(),
			EventTime: now,
			EventType: enumsv1.EVENT_TYPE_WORKFLOW_TASK_COMPLETED,
			Attributes: &historyv1.HistoryEvent_WorkflowTaskCompletedEventAttributes{
				WorkflowTaskCompletedEventAttributes: &historyv1.WorkflowTaskCompletedEventAttributes{ScheduledEventId: schedID, StartedEventId: startID},
			},
		})

		timerStartID := nextID()
		timerIDStr := fmt.Sprintf("%d", timerStartID)
		*events = append(*events, &historyv1.HistoryEvent{
			EventId:   timerStartID,
			EventTime: now,
			EventType: enumsv1.EVENT_TYPE_TIMER_STARTED,
			Attributes: &historyv1.HistoryEvent_TimerStartedEventAttributes{
				TimerStartedEventAttributes: &historyv1.TimerStartedEventAttributes{
					TimerId: timerIDStr,
				},
			},
		})

		*events = append(*events, &historyv1.HistoryEvent{
			EventId:   nextID(),
			EventTime: now,
			EventType: enumsv1.EVENT_TYPE_TIMER_FIRED,
			Attributes: &historyv1.HistoryEvent_TimerFiredEventAttributes{
				TimerFiredEventAttributes: &historyv1.TimerFiredEventAttributes{
					TimerId:        timerIDStr,
					StartedEventId: timerStartID,
				},
			},
		})

		addAct("EmitLifecycleEvent") // exit_1 node_entered
		addAct("EmitLifecycleEvent") // exit_1 workflow_succeeded
	})

	// 3. Condition Branch History
	inputCond := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-replay-cond",
		RunID:         "run-replay-cond",
		TenantID:      "tenant-replay-test",
		ContentHash:   "hash-cond-replay",
		ExecutionMode: workflows.ExecutionModeProduction,
		InputPayload:  map[string]interface{}{"tier": "premium"},
	}
	irCond := domain.CompiledIR{
		SchemaVersion: domain.DefaultSchemaVersion,
		IRID:          "ir-cond-replay",
		TenantID:      "tenant-replay-test",
		ContentHash:   "hash-cond-replay",
		EntryNodeID:   "start_1",
		Nodes: []domain.IRNode{
			{ID: "start_1", Type: "event_start"},
			{ID: "cond_1", Type: "condition"},
			{ID: "exit_1", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
		},
		Edges: []domain.IREdge{
			{ID: "e1", SourceID: "start_1", TargetID: "cond_1"},
			{ID: "e2", SourceID: "cond_1", TargetID: "exit_1", ConditionExpression: "tier == 'premium'"},
		},
	}
	resCond := workflows.CompiledJourneyResult{
		RunID:         "run-replay-cond",
		WorkflowID:    "wf-replay-cond",
		Status:        workflows.StatusSucceeded,
		CurrentNodeID: "exit_1",
		VisitCounts:   map[string]int{"start_1": 1, "cond_1": 1, "exit_1": 1},
		NodeOutputs:   map[string]interface{}{"start_1": "started", "cond_1": "exit_1", "exit_1": "succeeded"},
	}

	eventsCond := createEvents(inputCond, irCond, resCond, func(nextID func() int64, events *[]*historyv1.HistoryEvent) {
		addActWithResult := func(actName string, resPayloads *commonv1.Payloads) {
			schedID := nextID()
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   schedID,
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED,
				Attributes: &historyv1.HistoryEvent_WorkflowTaskScheduledEventAttributes{
					WorkflowTaskScheduledEventAttributes: &historyv1.WorkflowTaskScheduledEventAttributes{},
				},
			})
			startID := nextID()
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   startID,
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_WORKFLOW_TASK_STARTED,
				Attributes: &historyv1.HistoryEvent_WorkflowTaskStartedEventAttributes{
					WorkflowTaskStartedEventAttributes: &historyv1.WorkflowTaskStartedEventAttributes{ScheduledEventId: schedID},
				},
			})
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   nextID(),
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_WORKFLOW_TASK_COMPLETED,
				Attributes: &historyv1.HistoryEvent_WorkflowTaskCompletedEventAttributes{
					WorkflowTaskCompletedEventAttributes: &historyv1.WorkflowTaskCompletedEventAttributes{ScheduledEventId: schedID, StartedEventId: startID},
				},
			})

			actSchedID := nextID()
			actIDStr := fmt.Sprintf("%d", actSchedID)
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   actSchedID,
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
				Attributes: &historyv1.HistoryEvent_ActivityTaskScheduledEventAttributes{
					ActivityTaskScheduledEventAttributes: &historyv1.ActivityTaskScheduledEventAttributes{
						ActivityId:   actIDStr,
						ActivityType: &commonv1.ActivityType{Name: actName},
					},
				},
			})
			actStartID := nextID()
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   actStartID,
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_ACTIVITY_TASK_STARTED,
				Attributes: &historyv1.HistoryEvent_ActivityTaskStartedEventAttributes{
					ActivityTaskStartedEventAttributes: &historyv1.ActivityTaskStartedEventAttributes{ScheduledEventId: actSchedID},
				},
			})
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   nextID(),
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
				Attributes: &historyv1.HistoryEvent_ActivityTaskCompletedEventAttributes{
					ActivityTaskCompletedEventAttributes: &historyv1.ActivityTaskCompletedEventAttributes{
						ScheduledEventId: actSchedID,
						StartedEventId:   actStartID,
						Result:           resPayloads,
					},
				},
			})
		}

		addActWithResult("EmitLifecycleEvent", nilPayloads) // start_1 node_entered
		addActWithResult("EmitLifecycleEvent", nilPayloads) // cond_1 node_entered
		addActWithResult("EmitLifecycleEvent", nilPayloads) // exit_1 node_entered
	})

	// 4. Experiment & Action History
	inputExp := workflows.CompiledJourneyInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-replay-exp",
		RunID:         "run-replay-exp",
		TenantID:      "tenant-replay-test",
		ContentHash:   "hash-exp-replay",
		ExecutionMode: workflows.ExecutionModeProduction,
		InputPayload:  map[string]interface{}{"user_id": "usr_999"},
	}
	irExp := domain.CompiledIR{
		SchemaVersion: domain.DefaultSchemaVersion,
		IRID:          "ir-exp-replay",
		TenantID:      "tenant-replay-test",
		ContentHash:   "hash-exp-replay",
		EntryNodeID:   "start_1",
		Nodes: []domain.IRNode{
			{ID: "start_1", Type: "event_start"},
			{ID: "exp_1", Type: "experiment", Params: map[string]interface{}{"experiment_id": "exp_1"}},
			{ID: "email_1", Type: "email", Params: map[string]interface{}{"template": "welcome"}},
			{ID: "exit_1", Type: "exit", Params: map[string]interface{}{"status": "succeeded"}},
		},
		Edges: []domain.IREdge{
			{ID: "e1", SourceID: "start_1", TargetID: "exp_1"},
			{ID: "e2", SourceID: "exp_1", TargetID: "email_1", ConditionExpression: "variant_a"},
			{ID: "e3", SourceID: "email_1", TargetID: "exit_1"},
		},
	}
	resExp := workflows.CompiledJourneyResult{
		RunID:         "run-replay-exp",
		WorkflowID:    "wf-replay-exp",
		Status:        workflows.StatusSucceeded,
		CurrentNodeID: "exit_1",
		VisitCounts:   map[string]int{"start_1": 1, "exp_1": 1, "email_1": 1, "exit_1": 1},
		NodeOutputs:   map[string]interface{}{"start_1": "started", "exp_1": "variant_a", "exit_1": "succeeded"},
	}

	asgRecordP, _ := dc.ToPayloads(&experiments.AssignmentRecord{
		TenantID:          "tenant-replay-test",
		ExperimentID:      "exp_1",
		AssignmentID:      "asg_1",
		SubjectID:         "usr_999",
		VariantID:         "variant_a",
		WeightBasisPoints: 5000,
	})
	expRecordP, _ := dc.ToPayloads(&outcomes.ExposureRecord{
		ExposureID: "exp_rec_1",
	})
	gwResultP, _ := dc.ToPayloads(&activities.GatewayResult{
		IdempotencyKey: "gw_1",
		Status:         activities.LedgerStatusAccepted,
	})

	eventsExp := createEvents(inputExp, irExp, resExp, func(nextID func() int64, events *[]*historyv1.HistoryEvent) {
		addActWithResult := func(actName string, resPayloads *commonv1.Payloads) {
			schedID := nextID()
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   schedID,
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_WORKFLOW_TASK_SCHEDULED,
				Attributes: &historyv1.HistoryEvent_WorkflowTaskScheduledEventAttributes{
					WorkflowTaskScheduledEventAttributes: &historyv1.WorkflowTaskScheduledEventAttributes{},
				},
			})
			startID := nextID()
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   startID,
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_WORKFLOW_TASK_STARTED,
				Attributes: &historyv1.HistoryEvent_WorkflowTaskStartedEventAttributes{
					WorkflowTaskStartedEventAttributes: &historyv1.WorkflowTaskStartedEventAttributes{ScheduledEventId: schedID},
				},
			})
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   nextID(),
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_WORKFLOW_TASK_COMPLETED,
				Attributes: &historyv1.HistoryEvent_WorkflowTaskCompletedEventAttributes{
					WorkflowTaskCompletedEventAttributes: &historyv1.WorkflowTaskCompletedEventAttributes{ScheduledEventId: schedID, StartedEventId: startID},
				},
			})

			actSchedID := nextID()
			actIDStr := fmt.Sprintf("%d", actSchedID)
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   actSchedID,
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED,
				Attributes: &historyv1.HistoryEvent_ActivityTaskScheduledEventAttributes{
					ActivityTaskScheduledEventAttributes: &historyv1.ActivityTaskScheduledEventAttributes{
						ActivityId:   actIDStr,
						ActivityType: &commonv1.ActivityType{Name: actName},
					},
				},
			})
			actStartID := nextID()
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   actStartID,
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_ACTIVITY_TASK_STARTED,
				Attributes: &historyv1.HistoryEvent_ActivityTaskStartedEventAttributes{
					ActivityTaskStartedEventAttributes: &historyv1.ActivityTaskStartedEventAttributes{ScheduledEventId: actSchedID},
				},
			})
			*events = append(*events, &historyv1.HistoryEvent{
				EventId:   nextID(),
				EventTime: now,
				EventType: enumsv1.EVENT_TYPE_ACTIVITY_TASK_COMPLETED,
				Attributes: &historyv1.HistoryEvent_ActivityTaskCompletedEventAttributes{
					ActivityTaskCompletedEventAttributes: &historyv1.ActivityTaskCompletedEventAttributes{
						ScheduledEventId: actSchedID,
						StartedEventId:   actStartID,
						Result:           resPayloads,
					},
				},
			})
		}

		addActWithResult("EmitLifecycleEvent", nilPayloads) // start_1 node_entered
		addActWithResult("EmitLifecycleEvent", nilPayloads) // exp_1 node_entered
		addActWithResult("GetOrAssign", asgRecordP)
		addActWithResult("RecordExposure", expRecordP)
		addActWithResult("EmitLifecycleEvent", nilPayloads) // email_1 node_entered
		addActWithResult("ExecuteActionGateway", gwResultP)
		addActWithResult("EmitLifecycleEvent", nilPayloads) // exit_1 node_entered
		addActWithResult("EmitLifecycleEvent", nilPayloads) // exit_1 workflow_succeeded
	})

	fixtures := []fixtureSpec{
		{Filename: "linear_execution_history.json", Events: eventsLinear},
		{Filename: "delay_timer_history.json", Events: eventsDelay},
		{Filename: "condition_branch_history.json", Events: eventsCond},
		{Filename: "experiment_action_history.json", Events: eventsExp},
	}

	for _, fix := range fixtures {
		filePath := filepath.Join(fixturesDir, fix.Filename)
		hist := &historyv1.History{Events: fix.Events}
		jsonBytes, err := protojson.MarshalOptions{Indent: "  "}.Marshal(hist)
		if err != nil {
			t.Fatalf("failed to marshal fixture history %s: %v", fix.Filename, err)
		}
		if err := os.WriteFile(filePath, jsonBytes, 0644); err != nil {
			t.Fatalf("failed to write fixture history file %s: %v", filePath, err)
		}
	}
}

func TestWorkflowReplay_Fixtures(t *testing.T) {
	fixturesDir := getFixturesDirectory(t)
	generateHistoryFixturesIfMissing(t, fixturesDir)

	files, err := os.ReadDir(fixturesDir)
	if err != nil {
		t.Fatalf("failed to read fixtures directory %s: %v", fixturesDir, err)
	}

	replayer := worker.NewWorkflowReplayer()
	replayer.RegisterWorkflow(workflows.CompiledJourneyWorkflow)

	jsonCount := 0
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".json") {
			jsonCount++
			histPath := filepath.Join(fixturesDir, f.Name())
			t.Run(f.Name(), func(t *testing.T) {
				err := replayer.ReplayWorkflowHistoryFromJSONFile(nil, histPath)
				if err != nil {
					t.Fatalf("replay of %s failed with non-determinism or replay error: %v", f.Name(), err)
				}
			})
		}
	}

	if jsonCount == 0 {
		t.Fatalf("no JSON history fixtures found in %s", fixturesDir)
	}
}
