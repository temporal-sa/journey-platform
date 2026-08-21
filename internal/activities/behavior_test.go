package activities

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestBehaviorInjector_ErrorBehavior(t *testing.T) {
	injector := NewBehaviorInjector()

	// 1. Initially no error
	ctx := context.Background()
	info := ActivityInfo{ActivityName: "ExecuteActionGateway", NodeID: "node-1"}
	if err := injector.Intercept(ctx, info); err != nil {
		t.Fatalf("expected no error initially, got: %v", err)
	}

	// 2. Add global error behavior
	injector.AddBehavior(NewErrorBehavior("custom transient network failure", "", ""))
	err := injector.Intercept(ctx, info)
	if err == nil || err.Error() != "custom transient network failure" {
		t.Fatalf("expected custom error, got: %v", err)
	}

	// 3. Clear behaviors
	injector.Clear()
	if err := injector.Intercept(ctx, info); err != nil {
		t.Fatalf("expected no error after Clear(), got: %v", err)
	}
}

func TestBehaviorInjector_TargetedErrorBehavior(t *testing.T) {
	injector := NewBehaviorInjector()

	// Target only node-sms on ExecuteActionGateway
	injector.AddBehavior(NewErrorBehavior("sms gateway down", "ExecuteActionGateway", "node-sms"))

	ctx := context.Background()

	// Should not match node-email
	infoEmail := ActivityInfo{ActivityName: "ExecuteActionGateway", NodeID: "node-email"}
	if err := injector.Intercept(ctx, infoEmail); err != nil {
		t.Fatalf("expected node-email to pass, got: %v", err)
	}

	// Should not match ExecuteNode activity
	infoOtherAct := ActivityInfo{ActivityName: "ExecuteNode", NodeID: "node-sms"}
	if err := injector.Intercept(ctx, infoOtherAct); err != nil {
		t.Fatalf("expected ExecuteNode to pass, got: %v", err)
	}

	// Should match node-sms on ExecuteActionGateway
	infoSMS := ActivityInfo{ActivityName: "ExecuteActionGateway", NodeID: "node-sms"}
	err := injector.Intercept(ctx, infoSMS)
	if err == nil || err.Error() != "sms gateway down" {
		t.Fatalf("expected sms gateway down error, got: %v", err)
	}
}

func TestBehaviorInjector_DelayBehavior(t *testing.T) {
	injector := NewBehaviorInjector()

	delay := 50 * time.Millisecond
	injector.AddBehavior(NewDelayBehavior(delay, "", "node-delay-target"))

	infoTarget := ActivityInfo{ActivityName: "ExecuteNode", NodeID: "node-delay-target"}
	infoOther := ActivityInfo{ActivityName: "ExecuteNode", NodeID: "node-fast"}

	// Other node does not delay
	start := time.Now()
	if err := injector.Intercept(context.Background(), infoOther); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if time.Since(start) > 20*time.Millisecond {
		t.Fatalf("expected fast execution without delay, took: %v", time.Since(start))
	}

	// Target node executes with delay
	start = time.Now()
	if err := injector.Intercept(context.Background(), infoTarget); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if time.Since(start) < 40*time.Millisecond {
		t.Fatalf("expected delay of at least 40ms, took: %v", time.Since(start))
	}

	// Context cancellation cancels delay
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := injector.Intercept(ctx, infoTarget)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestBehaviorInjector_HookBehavior(t *testing.T) {
	injector := NewBehaviorInjector()

	var interceptedNodeIDs []string
	var mu sync.Mutex

	hook := NewHookBehavior(func(ctx context.Context, info ActivityInfo) error {
		mu.Lock()
		interceptedNodeIDs = append(interceptedNodeIDs, info.NodeID)
		mu.Unlock()
		if info.NodeID == "node-block" {
			return errors.New("blocked by hook policy")
		}
		return nil
	}, "", "")

	injector.AddBehavior(hook)

	ctx := context.Background()

	err1 := injector.Intercept(ctx, ActivityInfo{ActivityName: "ExecuteActionGateway", NodeID: "node-allow-1"})
	if err1 != nil {
		t.Fatalf("expected node-allow-1 to succeed, got: %v", err1)
	}

	err2 := injector.Intercept(ctx, ActivityInfo{ActivityName: "ExecuteActionGateway", NodeID: "node-block"})
	if err2 == nil || err2.Error() != "blocked by hook policy" {
		t.Fatalf("expected blocked by hook policy, got: %v", err2)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(interceptedNodeIDs) != 2 || interceptedNodeIDs[0] != "node-allow-1" || interceptedNodeIDs[1] != "node-block" {
		t.Fatalf("unexpected intercepted node IDs: %v", interceptedNodeIDs)
	}
}

func TestBehaviorInjector_SetFailureEnabledToggle(t *testing.T) {
	injector := NewBehaviorInjector()

	if injector.IsFailureEnabled() {
		t.Fatalf("expected failure to be disabled initially")
	}

	injector.SetFailureEnabled(true)
	if !injector.IsFailureEnabled() {
		t.Fatalf("expected failure to be enabled")
	}

	err := injector.Intercept(context.Background(), ActivityInfo{ActivityName: "ExecuteNode"})
	if err == nil {
		t.Fatalf("expected failure error when enabled")
	}

	injector.SetFailureEnabled(false)
	if injector.IsFailureEnabled() {
		t.Fatalf("expected failure to be disabled")
	}

	err = injector.Intercept(context.Background(), ActivityInfo{ActivityName: "ExecuteNode"})
	if err != nil {
		t.Fatalf("expected success when failure disabled, got: %v", err)
	}
}

func TestBehaviorInjector_ConcurrentAccess(t *testing.T) {
	injector := NewBehaviorInjector()

	var wg sync.WaitGroup
	workers := 20
	iterations := 100

	// Concurrent activity interceptors
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iterations {
				_ = injector.Intercept(context.Background(), ActivityInfo{
					ActivityName: "ExecuteActionGateway",
					NodeID:       "node-concurrent",
				})
			}
		}()
	}

	// Concurrent dynamic behavior modifications
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range iterations {
				if i%2 == 0 {
					injector.AddBehavior(NewErrorBehavior("transient", "", ""))
				} else {
					injector.Clear()
				}
				_ = injector.Behaviors()
			}
		}()
	}

	wg.Wait()
}
func TestBehaviorInjector_LatencyAffectsAllActivities(t *testing.T) {
	acts := NewActivities()

	// 1. Enable latency
	SetSimulatedActivityLatencyMS(30)
	defer SetSimulatedActivityLatencyMS(0)

	ctx := context.Background()

	// A. EvaluateCondition
	start := time.Now()
	_, err := acts.EvaluateCondition(ctx, EvaluateConditionInput{ConditionExpression: "true"})
	if err != nil {
		t.Fatalf("unexpected error in EvaluateCondition: %v", err)
	}
	if time.Since(start) < 25*time.Millisecond {
		t.Fatalf("expected EvaluateCondition to delay at least 25ms, took %v", time.Since(start))
	}

	// B. EmitLifecycleEvent
	start = time.Now()
	_, err = acts.EmitLifecycleEvent(ctx, LifecycleEvent{RunID: "run-lat-1", EventType: "test_event"})
	if err != nil {
		t.Fatalf("unexpected error in EmitLifecycleEvent: %v", err)
	}
	if time.Since(start) < 25*time.Millisecond {
		t.Fatalf("expected EmitLifecycleEvent to delay at least 25ms, took %v", time.Since(start))
	}

	// C. CreateSubscription
	start = time.Now()
	_, err = acts.CreateSubscription(ctx, CreateSubscriptionInput{SubscriptionID: "sub-lat-1", EventType: "test_evt"})
	if err != nil {
		t.Fatalf("unexpected error in CreateSubscription: %v", err)
	}
	if time.Since(start) < 25*time.Millisecond {
		t.Fatalf("expected CreateSubscription to delay at least 25ms, took %v", time.Since(start))
	}

	// D. Disable latency and verify immediate execution
	SetSimulatedActivityLatencyMS(0)
	start = time.Now()
	_, err = acts.EvaluateCondition(ctx, EvaluateConditionInput{ConditionExpression: "true"})
	if err != nil {
		t.Fatalf("unexpected error in EvaluateCondition without latency: %v", err)
	}
	if time.Since(start) > 20*time.Millisecond {
		t.Fatalf("expected EvaluateCondition without latency to be fast, took %v", time.Since(start))
	}
}
