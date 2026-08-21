package activities

import (
	"context"
	"testing"

	"github.com/validated-pattern/journey-platform/internal/domain"
)
func TestRegister(t *testing.T) {
	r := New()
	if !r.Register("send_email") {
		t.Error("expected successful registration")
	}
}
func TestExecuteNode_SimulatedFailure(t *testing.T) {
	acts := NewActivities()
	input := ExecuteNodeInput{
		RunID: "run-sim-101",
		Node: domain.IRNode{
			ID:           "node-email-1",
			ActivityName: "SendEmail",
		},
	}

	// 1. When enabled, ExecuteNode should fail
	SetSimulatedActivityFailure(true)
	defer SetSimulatedActivityFailure(false)

	res, err := acts.ExecuteNode(context.Background(), input)
	if err == nil {
		t.Fatalf("expected simulated failure error when enabled, got success: %+v", res)
	}
	expectedErr := "simulated activity execution failure"
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got %q", expectedErr, err.Error())
	}

	// 2. When disabled, ExecuteNode should succeed
	SetSimulatedActivityFailure(false)
	res, err = acts.ExecuteNode(context.Background(), input)
	if err != nil {
		t.Fatalf("expected success after disabling simulated failure, got error: %v", err)
	}
	if res.Status != domain.ActionResultStatusSuccess {
		t.Errorf("expected ActionResultStatusSuccess, got %s", res.Status)
	}
}

func TestExecuteNode_SimulatedFailureDisabled(t *testing.T) {
	SetSimulatedActivityFailure(false)
	acts := NewActivities()
	input := ExecuteNodeInput{
		RunID: "run-sim-102",
		Node: domain.IRNode{
			ID:           "node-email-2",
			ActivityName: "SendEmail",
		},
	}

	res, err := acts.ExecuteNode(context.Background(), input)
	if err != nil {
		t.Fatalf("expected success when simulated failure disabled, got error: %v", err)
	}
	if res.Status != domain.ActionResultStatusSuccess {
		t.Errorf("expected ActionResultStatusSuccess, got %s", res.Status)
	}
}
