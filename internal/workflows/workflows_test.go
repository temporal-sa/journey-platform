package workflows_test

import (
	"testing"

	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/workflows"
)

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
