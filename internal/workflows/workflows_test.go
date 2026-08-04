package workflows_test

import (
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/activities"
	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/workflows"
	"go.temporal.io/sdk/testsuite"
)

type testUpdateCallbacks struct {
	onAccept   func()
	onReject   func(err error)
	onComplete func(result interface{}, err error)
}

func (c *testUpdateCallbacks) Accept() {
	if c.onAccept != nil {
		c.onAccept()
	}
}

func (c *testUpdateCallbacks) Reject(err error) {
	if c.onReject != nil {
		c.onReject(err)
	}
}

func (c *testUpdateCallbacks) Complete(result interface{}, err error) {
	if c.onComplete != nil {
		c.onComplete(result, err)
	}
}

func TestEngineExecute(t *testing.T) {
	e := workflows.New()
	if err := e.Execute("wf-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := e.Execute(""); err == nil {
		t.Fatalf("expected error for empty workflowID")
	}
}

func TestTaskQueueConstants(t *testing.T) {
	if workflows.JOURNEY_TASK_QUEUE != "journey-engine-task-queue" {
		t.Errorf("expected JOURNEY_TASK_QUEUE constant to be 'journey-engine-task-queue', got '%s'", workflows.JOURNEY_TASK_QUEUE)
	}
	if workflows.TaskQueue != workflows.JOURNEY_TASK_QUEUE {
		t.Errorf("expected TaskQueue alias to match JOURNEY_TASK_QUEUE")
	}
}

func TestJourneyWorkflow_TestSuiteHarness(t *testing.T) {
	s := &testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()

	// Set custom data converter
	env.SetDataConverter(workflows.NewIsolatedDataConverter(nil))

	// Register Activity implementations
	act := activities.NewActivities()
	workflows.RegisterAllActivities(env, act)

	input := domain.WorkflowInput{
		SchemaVersion:      domain.DefaultSchemaVersion,
		WorkflowID:         "wf-test-100",
		RunID:              "run-test-100",
		TenantID:           "tenant-alpha",
		TriggerEventID:     "evt-123",
		IRID:               "ir-456",
		DataClassification: domain.DataClassificationNonPII,
		InputPayload: map[string]interface{}{
			"user_id": "usr-1",
			"tier":    "premium",
		},
	}

	env.ExecuteWorkflow(workflows.JourneyWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("expected workflow to be completed")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow execution returned error: %v", err)
	}

	var result domain.RunProjection
	err := env.GetWorkflowResult(&result)
	if err != nil {
		t.Fatalf("failed to get workflow result: %v", err)
	}

	if result.WorkflowID != input.WorkflowID {
		t.Errorf("expected WorkflowID %s, got %s", input.WorkflowID, result.WorkflowID)
	}
	if result.RunID != input.RunID {
		t.Errorf("expected RunID %s, got %s", input.RunID, result.RunID)
	}
	if result.Status != domain.RunStatusCompleted {
		t.Errorf("expected Status %s, got %s", domain.RunStatusCompleted, result.Status)
	}
}

func TestJourneyWorkflow_SignalsAndQueries(t *testing.T) {
	s := &testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()
	env.SetDataConverter(workflows.NewIsolatedDataConverter(nil))

	act := activities.NewActivities()
	workflows.RegisterAllActivities(env, act)

	// Send signal before workflow finishes
	env.RegisterDelayedCallback(func() {
		sig := domain.WorkflowSignal{
			SchemaVersion:      domain.DefaultSchemaVersion,
			SignalName:         workflows.SignalNameEvent,
			RunID:              "run-test-200",
			Payload:            map[string]interface{}{"event_received": true},
			DataClassification: domain.DataClassificationNonPII,
		}
		env.SignalWorkflow(workflows.SignalNameEvent, sig)

		// Test status query
		res, err := env.QueryWorkflow(workflows.QueryTypeStatus)
		if err != nil {
			t.Errorf("failed to query workflow status: %v", err)
		} else {
			var proj domain.RunProjection
			if err := res.Get(&proj); err != nil {
				t.Errorf("failed to unmarshal status query result: %v", err)
			} else if proj.RunID != "run-test-200" {
				t.Errorf("expected RunID run-test-200, got %s", proj.RunID)
			}
		}

		// Test variables query
		varRes, err := env.QueryWorkflow(workflows.QueryTypeVariables)
		if err != nil {
			t.Errorf("failed to query workflow variables: %v", err)
		} else {
			var vars map[string]interface{}
			if err := varRes.Get(&vars); err != nil {
				t.Errorf("failed to unmarshal variables query result: %v", err)
			} else if vars["user_id"] != "usr-2" {
				t.Errorf("expected user_id usr-2, got %v", vars["user_id"])
			}
		}
	}, time.Millisecond*5)

	input := domain.WorkflowInput{
		SchemaVersion:      domain.DefaultSchemaVersion,
		WorkflowID:         "wf-test-200",
		RunID:              "run-test-200",
		TenantID:           "tenant-beta",
		TriggerEventID:     "evt-999",
		IRID:               "ir-888",
		DataClassification: domain.DataClassificationNonPII,
		InputPayload: map[string]interface{}{
			"user_id": "usr-2",
		},
	}

	env.ExecuteWorkflow(workflows.JourneyWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("expected workflow to complete")
	}
}

func TestJourneyWorkflow_Updates(t *testing.T) {
	s := &testsuite.WorkflowTestSuite{}
	env := s.NewTestWorkflowEnvironment()
	env.SetDataConverter(workflows.NewIsolatedDataConverter(nil))

	act := activities.NewActivities()
	workflows.RegisterAllActivities(env, act)

	env.RegisterDelayedCallback(func() {
		updatedProj := domain.RunProjection{
			Status:       domain.RunStatusRunning,
			CurrentNodes: []string{"node-state-update"},
		}

		callbacks := &testUpdateCallbacks{
			onAccept: func() {},
			onComplete: func(val interface{}, err error) {
				if err != nil {
					t.Errorf("update returned error: %v", err)
					return
				}
				res, ok := val.(domain.RunProjection)
				if !ok {
					t.Errorf("unexpected return type: %T", val)
					return
				}
				if len(res.CurrentNodes) == 0 || res.CurrentNodes[0] != "node-state-update" {
					t.Errorf("expected updated node 'node-state-update', got %v", res.CurrentNodes)
				}
			},
		}

		env.UpdateWorkflow(workflows.UpdateTypeState, "update-id-1", callbacks, updatedProj)
	}, time.Millisecond*5)

	input := domain.WorkflowInput{
		SchemaVersion: domain.DefaultSchemaVersion,
		WorkflowID:    "wf-test-300",
		RunID:         "run-test-300",
		TenantID:      "tenant-gamma",
	}

	env.ExecuteWorkflow(workflows.JourneyWorkflow, input)

	if !env.IsWorkflowCompleted() {
		t.Fatalf("expected workflow to complete")
	}
}

func TestIsolatedDataConverter_Boundaries(t *testing.T) {
	converter := workflows.NewIsolatedDataConverter(nil)

	// Test normal payload encoding/decoding
	input := domain.WorkflowInput{
		WorkflowID: "wf-1",
		RunID:      "run-1",
	}

	payloads, err := converter.ToPayloads(input)
	if err != nil {
		t.Fatalf("ToPayloads failed: %v", err)
	}
	if payloads == nil {
		t.Fatalf("expected non-nil payloads")
	}

	var decoded domain.WorkflowInput
	err = converter.FromPayloads(payloads, &decoded)
	if err != nil {
		t.Fatalf("FromPayloads failed: %v", err)
	}
	if decoded.WorkflowID != input.WorkflowID {
		t.Errorf("expected WorkflowID %s, got %s", input.WorkflowID, decoded.WorkflowID)
	}
}

func TestWorkerDefaultConfig(t *testing.T) {
	cfg := workflows.DefaultWorkerConfig()
	if cfg.TaskQueue != workflows.JOURNEY_TASK_QUEUE {
		t.Errorf("expected TaskQueue %s, got %s", workflows.JOURNEY_TASK_QUEUE, cfg.TaskQueue)
	}
	if cfg.Identity == "" {
		t.Errorf("expected non-empty worker identity")
	}
}
