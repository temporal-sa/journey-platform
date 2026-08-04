package testaudience

import (
	"context"
	"strings"
	"testing"
	"time"
)

func helperCreateStaticList(t *testing.T, listID string, count int) *StaticList {
	t.Helper()
	parser := NewParser(DefaultParserOptions())
	var sb strings.Builder
	sb.WriteString("member_id,recipient\n")
	for i := 1; i <= count; i++ {
		sb.WriteString(strings.ReplaceAll("user-INDEX,userINDEX@example.com\n", "INDEX", string(rune('0'+i))))
	}
	list, err := parser.ParseReader(strings.NewReader(sb.String()), "tenant-test", listID)
	if err != nil {
		t.Fatalf("failed to create static list: %v", err)
	}
	return list
}

func TestCoordinator_IdempotentStart(t *testing.T) {
	evaluator := New()
	list := helperCreateStaticList(t, "list-idempotent", 5)
	evaluator.RegisterList(list)

	store := NewInMemoryStorage()
	coord := NewCoordinator(evaluator, store)

	ctx := context.Background()
	input := TestRunInput{
		TenantID:        "tenant-test",
		TestRunID:       "tr-idempotent-001",
		CandidateIRHash: "hash-ir-12345",
		ListVersionHash: list.ContentHash,
		StaticListID:    "list-idempotent",
		FixturePackHash: "hash-fix-67890",
		AssignmentMode:  AssignmentModeRealistic,
		FakeProvider:    "fake-email-provider",
		ExpiryDuration:  1 * time.Hour,
	}

	// First start
	run1, targets1, err := coord.StartTestRun(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error on first StartTestRun: %v", err)
	}
	if run1.TestRunID != "tr-idempotent-001" {
		t.Errorf("expected test run ID tr-idempotent-001, got %s", run1.TestRunID)
	}
	if run1.Status != TestRunStatusRunning {
		t.Errorf("expected status running, got %s", run1.Status)
	}
	if run1.TotalTargets != 5 {
		t.Errorf("expected 5 total targets, got %d", run1.TotalTargets)
	}
	if len(targets1) != 5 {
		t.Errorf("expected 5 targets returned, got %d", len(targets1))
	}
	if !run1.IsTest {
		t.Error("expected IsTest to be true")
	}

	// Verify each target has IsTest: true and correct inputs pinned
	for _, tgt := range targets1 {
		if !tgt.IsTest {
			t.Errorf("expected target %s to have IsTest true", tgt.TargetID)
		}
		if tgt.Status != TargetStatusPending {
			t.Errorf("expected target status pending, got %s", tgt.Status)
		}
	}

	// Second start (Idempotency test)
	run2, targets2, err := coord.StartTestRun(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error on second StartTestRun: %v", err)
	}
	if run2.TestRunID != run1.TestRunID {
		t.Errorf("expected same run ID %s, got %s", run1.TestRunID, run2.TestRunID)
	}
	if len(targets2) != len(targets1) {
		t.Errorf("expected %d targets on idempotent call, got %d", len(targets1), len(targets2))
	}

	// Ensure no duplicate targets were added to storage
	storedTargets, err := store.GetTestTargets(ctx, input.TestRunID)
	if err != nil {
		t.Fatalf("failed to query stored targets: %v", err)
	}
	if len(storedTargets) != 5 {
		t.Errorf("expected strictly 5 targets in store, got %d", len(storedTargets))
	}
}

func TestCoordinator_AssignmentModes(t *testing.T) {
	evaluator := New()
	list := helperCreateStaticList(t, "list-modes", 6)
	evaluator.RegisterList(list)

	variants := []string{"control", "variant_a", "variant_b"}

	t.Run("Realistic Mode", func(t *testing.T) {
		store := NewInMemoryStorage()
		coord := NewCoordinator(evaluator, store)
		ctx := context.Background()

		input := TestRunInput{
			TenantID:        "tenant-test",
			TestRunID:       "tr-mode-realistic",
			CandidateIRHash: "hash-ir-abc",
			StaticListID:    "list-modes",
			FixturePackHash: "hash-fix-xyz",
			AssignmentMode:  AssignmentModeRealistic,
			Variants:        variants,
		}

		run, targets, err := coord.StartTestRun(ctx, input)
		if err != nil {
			t.Fatalf("failed to start test run: %v", err)
		}
		if run.AssignmentMode != AssignmentModeRealistic {
			t.Errorf("expected assignment mode realistic, got %s", run.AssignmentMode)
		}

		// Verify deterministic hash assignment
		for _, tgt := range targets {
			expectedVar := selectDeterministicVariant(tgt.MemberKey, "hash-ir-abc", variants)
			if tgt.VariantID != expectedVar {
				t.Errorf("target %s expected variant %s, got %s", tgt.TargetID, expectedVar, tgt.VariantID)
			}
		}
	})

	t.Run("Forced Variant Coverage Mode", func(t *testing.T) {
		store := NewInMemoryStorage()
		coord := NewCoordinator(evaluator, store)
		ctx := context.Background()

		input := TestRunInput{
			TenantID:        "tenant-test",
			TestRunID:       "tr-mode-forced",
			CandidateIRHash: "hash-ir-abc",
			StaticListID:    "list-modes",
			FixturePackHash: "hash-fix-xyz",
			AssignmentMode:  AssignmentModeForcedVariantCoverage,
			Variants:        variants,
		}

		run, targets, err := coord.StartTestRun(ctx, input)
		if err != nil {
			t.Fatalf("failed to start test run: %v", err)
		}
		if run.AssignmentMode != AssignmentModeForcedVariantCoverage {
			t.Errorf("expected assignment mode forced_variant_coverage, got %s", run.AssignmentMode)
		}

		// Verify round-robin coverage: member 0 -> control, 1 -> variant_a, 2 -> variant_b, 3 -> control, etc.
		counts := make(map[string]int)
		for i, tgt := range targets {
			expectedVar := variants[i%len(variants)]
			if tgt.VariantID != expectedVar {
				t.Errorf("target index %d expected variant %s, got %s", i, expectedVar, tgt.VariantID)
			}
			counts[tgt.VariantID]++
		}

		// Every variant must be covered equally (2 of each for 6 members across 3 variants)
		for _, v := range variants {
			if counts[v] != 2 {
				t.Errorf("expected 2 assignments for variant %s, got %d", v, counts[v])
			}
		}
	})
}

func TestCoordinator_ExpiryAndCancellation(t *testing.T) {
	evaluator := New()
	list := helperCreateStaticList(t, "list-lifecycle", 5)
	evaluator.RegisterList(list)

	t.Run("Cancellation", func(t *testing.T) {
		store := NewInMemoryStorage()
		coord := NewCoordinator(evaluator, store)
		ctx := context.Background()

		input := TestRunInput{
			TenantID:        "tenant-test",
			TestRunID:       "tr-cancel",
			CandidateIRHash: "hash-ir-1",
			StaticListID:    "list-lifecycle",
			FixturePackHash: "hash-fix-1",
			ExpiryDuration:  1 * time.Hour,
		}

		_, targets, err := coord.StartTestRun(ctx, input)
		if err != nil {
			t.Fatalf("failed to start test run: %v", err)
		}

		// Dispatch 2 targets
		ledger1, err := coord.DispatchTarget(ctx, "tr-cancel", targets[0].TargetID, "node-1", "email_action")
		if err != nil {
			t.Fatalf("failed to dispatch target 0: %v", err)
		}
		if ledger1.Status != "success" || !ledger1.IsTest {
			t.Errorf("invalid action ledger: %+v", ledger1)
		}

		_, err = coord.DispatchTarget(ctx, "tr-cancel", targets[1].TargetID, "node-1", "email_action")
		if err != nil {
			t.Fatalf("failed to dispatch target 1: %v", err)
		}

		// Cancel test run
		cancelledRun, err := coord.CancelTestRun(ctx, "tr-cancel", "manual test cancellation")
		if err != nil {
			t.Fatalf("failed to cancel test run: %v", err)
		}

		if cancelledRun.Status != TestRunStatusCancelled {
			t.Errorf("expected status cancelled, got %s", cancelledRun.Status)
		}
		if cancelledRun.CancelledCount != 3 {
			t.Errorf("expected 3 cancelled targets, got %d", cancelledRun.CancelledCount)
		}

		// Verify target states: 2 dispatched, 3 cancelled
		storedTargets, err := store.GetTestTargets(ctx, "tr-cancel")
		if err != nil {
			t.Fatalf("failed to get targets: %v", err)
		}
		dispatchedCount := 0
		cancelledCount := 0
		for _, tgt := range storedTargets {
			if tgt.Status == TargetStatusDispatched {
				dispatchedCount++
			} else if tgt.Status == TargetStatusCancelled {
				cancelledCount++
			}
		}
		if dispatchedCount != 2 {
			t.Errorf("expected 2 dispatched targets preserved, got %d", dispatchedCount)
		}
		if cancelledCount != 3 {
			t.Errorf("expected 3 cancelled targets, got %d", cancelledCount)
		}

		// Verify diagnostics preserved dispatched action ledgers & facts
		diag, err := coord.GetDiagnostics(ctx, "tr-cancel")
		if err != nil {
			t.Fatalf("failed to get diagnostics: %v", err)
		}
		if len(diag.Actions) != 2 {
			t.Errorf("expected 2 action ledgers preserved in diagnostics, got %d", len(diag.Actions))
		}
	})

	t.Run("Expiry Handling", func(t *testing.T) {
		store := NewInMemoryStorage()
		coord := NewCoordinator(evaluator, store)
		ctx := context.Background()

		input := TestRunInput{
			TenantID:        "tenant-test",
			TestRunID:       "tr-expire",
			CandidateIRHash: "hash-ir-2",
			StaticListID:    "list-lifecycle",
			FixturePackHash: "hash-fix-2",
			ExpiryDuration:  20 * time.Millisecond,
		}

		_, targets, err := coord.StartTestRun(ctx, input)
		if err != nil {
			t.Fatalf("failed to start test run: %v", err)
		}

		// Dispatch 1 target immediately
		_, err = coord.DispatchTarget(ctx, "tr-expire", targets[0].TargetID, "node-1", "sms_action")
		if err != nil {
			t.Fatalf("failed to dispatch target 0: %v", err)
		}

		// Wait past expiry duration
		time.Sleep(30 * time.Millisecond)

		// Trigger HandleExpiry
		expiredRun, err := coord.HandleExpiry(ctx, "tr-expire", time.Now())
		if err != nil {
			t.Fatalf("failed to handle expiry: %v", err)
		}

		if expiredRun.Status != TestRunStatusExpired {
			t.Errorf("expected status expired, got %s", expiredRun.Status)
		}
		if expiredRun.ExpiredCount != 4 {
			t.Errorf("expected 4 expired targets, got %d", expiredRun.ExpiredCount)
		}

		// Attempting to dispatch after expiry must fail
		_, err = coord.DispatchTarget(ctx, "tr-expire", targets[1].TargetID, "node-1", "sms_action")
		if err == nil {
			t.Error("expected error when dispatching on expired test run, got nil")
		}

		// Diagnostic results preserved
		diag, err := coord.GetDiagnostics(ctx, "tr-expire")
		if err != nil {
			t.Fatalf("failed to get diagnostics: %v", err)
		}
		if len(diag.Actions) != 1 {
			t.Errorf("expected 1 action ledger preserved, got %d", len(diag.Actions))
		}
	})
}

func TestCoordinator_DataIsolation(t *testing.T) {
	evaluator := New()
	list := helperCreateStaticList(t, "list-iso", 3)
	evaluator.RegisterList(list)

	store := NewInMemoryStorage()
	coord := NewCoordinator(evaluator, store)
	ctx := context.Background()

	input := TestRunInput{
		TenantID:        "tenant-test",
		TestRunID:       "tr-isolation",
		CandidateIRHash: "hash-ir-iso",
		StaticListID:    "list-iso",
		FixturePackHash: "hash-fix-iso",
		AssignmentMode:  AssignmentModeRealistic,
	}

	run, targets, err := coord.StartTestRun(ctx, input)
	if err != nil {
		t.Fatalf("failed to start test run: %v", err)
	}

	_, err = coord.DispatchTarget(ctx, "tr-isolation", targets[0].TargetID, "node-iso", "push_action")
	if err != nil {
		t.Fatalf("failed to dispatch target: %v", err)
	}

	diag, err := coord.GetDiagnostics(ctx, "tr-isolation")
	if err != nil {
		t.Fatalf("failed to get diagnostics: %v", err)
	}

	// 1. Assert IsTest == true across all data structures
	if !run.IsTest {
		t.Error("TestRunRecord.IsTest is false")
	}
	for _, tgt := range diag.Targets {
		if !tgt.IsTest {
			t.Errorf("Target %s IsTest is false", tgt.TargetID)
		}
	}
	for _, asg := range diag.Assignments {
		if !asg.IsTest {
			t.Errorf("Assignment %s IsTest is false", asg.AssignmentID)
		}
	}
	for _, act := range diag.Actions {
		if !act.IsTest {
			t.Errorf("ActionLedger %s IsTest is false", act.LedgerID)
		}
	}
	for _, fact := range diag.Analytics {
		if !fact.IsTest {
			t.Errorf("AnalyticsFact %s IsTest is false", fact.FactID)
		}
	}

	// 2. Assert production tables were completely untouched
	if store.IsProductionTableTouched() {
		t.Error("expected IsProductionTableTouched() to be false, but production table was modified")
	}
}
