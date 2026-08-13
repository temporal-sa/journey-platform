package activities

import (
	"context"
	"fmt"
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
			Params: map[string]interface{}{
				"simulated_activity_failure": true,
				"max_failure_attempts":       3,
			},
		},
	}

	// Attempts 1, 2, 3 should fail
	for attempt := 1; attempt <= 3; attempt++ {
		res, err := acts.ExecuteNode(context.Background(), input)
		if err == nil {
			t.Fatalf("expected simulated failure error on attempt %d, got success: %+v", attempt, res)
		}
		expectedErr := fmt.Sprintf("simulated activity execution failure attempt %d/3", attempt)
		if err.Error() != expectedErr {
			t.Errorf("expected error %q, got %q", expectedErr, err.Error())
		}
	}

	// Attempt 4 should succeed
	res, err := acts.ExecuteNode(context.Background(), input)
	if err != nil {
		t.Fatalf("expected attempt 4 to succeed after max failure attempts reached, got error: %v", err)
	}
	if res.Status != domain.ActionResultStatusSuccess {
		t.Errorf("expected ActionResultStatusSuccess, got %s", res.Status)
	}
}

func TestExecuteNode_SimulatedFailureDisabled(t *testing.T) {
	acts := NewActivities()
	input := ExecuteNodeInput{
		RunID: "run-sim-102",
		Node: domain.IRNode{
			ID:           "node-email-2",
			ActivityName: "SendEmail",
			Params: map[string]interface{}{
				"simulated_activity_failure": false,
				"max_failure_attempts":       3,
			},
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
