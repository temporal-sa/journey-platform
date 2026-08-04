package activities

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockErrStore implements all store interfaces and returns errors for fail-closed testing.
type mockErrStore struct{}

func (m *mockErrStore) IsTestSubject(ctx context.Context, subjectID string) (bool, error) {
	return false, errors.New("db unavailable")
}

func (m *mockErrStore) HasConsent(ctx context.Context, subjectID, channel string) (bool, error) {
	return false, errors.New("db unavailable")
}

func (m *mockErrStore) IsSuppressed(ctx context.Context, subjectID, recipientAddress, channel string) (bool, error) {
	return false, errors.New("db unavailable")
}

func (m *mockErrStore) IsContactable(ctx context.Context, subjectID, recipientAddress string) (bool, error) {
	return false, errors.New("db unavailable")
}

func TestPolicyService_Table(t *testing.T) {
	// Fixed reference time for deterministic testing:
	// 2026-07-27 16:00:00 UTC -> In "America/New_York" (EDT), this is 12:00:00 (12 PM Noon).
	fixedNoonUTC := time.Date(2026, 7, 27, 16, 0, 0, 0, time.UTC)
	// 2026-07-28 03:00:00 UTC -> In "America/New_York" (EDT), this is 23:00:00 (11 PM).
	fixedNightUTC := time.Date(2026, 7, 28, 3, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		context        *PolicyContext
		setup          func(mem *MemoryPolicyStore, fl *FrequencyLimiter)
		clock          func() time.Time
		customStore    bool
		wantDecision   Decision
		wantReason     string
		checkDelayTime bool
	}{
		{
			name: "Allow - Standard Production Request",
			context: &PolicyContext{
				SubjectID:        "user-100",
				Channel:          "email",
				ExecutionMode:    ExecutionModeProduction,
				RecipientAddress: "user100@example.com",
				Timezone:         "America/New_York",
			},
			setup: func(mem *MemoryPolicyStore, fl *FrequencyLimiter) {
				mem.SetConsent("user-100", "email", true)
				mem.SetContactable("user-100", "user100@example.com", true)
			},
			clock:        func() time.Time { return fixedNoonUTC },
			wantDecision: DecisionAllow,
			wantReason:   ReasonAllowed,
		},
		{
			name: "Test Mode - Member of Test List",
			context: &PolicyContext{
				SubjectID:        "test-user-01",
				Channel:          "email",
				ExecutionMode:    ExecutionModeTest,
				RecipientAddress: "tester@example.com",
				Timezone:         "UTC",
			},
			setup: func(mem *MemoryPolicyStore, fl *FrequencyLimiter) {
				mem.SetTestSubject("test-user-01", true)
				mem.SetConsent("test-user-01", "email", true)
			},
			clock:        func() time.Time { return fixedNoonUTC },
			wantDecision: DecisionAllow,
			wantReason:   ReasonAllowed,
		},
		{
			name: "Test Mode - Not Member of Test List (Suppressed)",
			context: &PolicyContext{
				SubjectID:        "user-200",
				Channel:          "email",
				ExecutionMode:    ExecutionModeTest,
				RecipientAddress: "user200@example.com",
				Timezone:         "UTC",
			},
			setup: func(mem *MemoryPolicyStore, fl *FrequencyLimiter) {
				mem.SetTestSubject("user-200", false)
			},
			clock:        func() time.Time { return fixedNoonUTC },
			wantDecision: DecisionSuppress,
			wantReason:   ReasonNotInTestList,
		},
		{
			name: "Consent Withdrawn (Suppressed)",
			context: &PolicyContext{
				SubjectID:        "user-300",
				Channel:          "sms",
				ExecutionMode:    ExecutionModeProduction,
				RecipientAddress: "+15550001111",
				Timezone:         "UTC",
			},
			setup: func(mem *MemoryPolicyStore, fl *FrequencyLimiter) {
				mem.SetConsent("user-300", "sms", false)
			},
			clock:        func() time.Time { return fixedNoonUTC },
			wantDecision: DecisionSuppress,
			wantReason:   ReasonConsentWithdrawn,
		},
		{
			name: "Suppression List Match (Suppressed)",
			context: &PolicyContext{
				SubjectID:        "user-400",
				Channel:          "email",
				ExecutionMode:    ExecutionModeProduction,
				RecipientAddress: "suppressed@example.com",
				Timezone:         "UTC",
			},
			setup: func(mem *MemoryPolicyStore, fl *FrequencyLimiter) {
				mem.SetSuppressed("user-400", "suppressed@example.com", "email", true)
			},
			clock:        func() time.Time { return fixedNoonUTC },
			wantDecision: DecisionSuppress,
			wantReason:   ReasonSuppressed,
		},
		{
			name: "Not Contactable (Suppressed)",
			context: &PolicyContext{
				SubjectID:        "user-500",
				Channel:          "email",
				ExecutionMode:    ExecutionModeProduction,
				RecipientAddress: "bounced@example.com",
				Timezone:         "UTC",
			},
			setup: func(mem *MemoryPolicyStore, fl *FrequencyLimiter) {
				mem.SetContactable("user-500", "bounced@example.com", false)
			},
			clock:        func() time.Time { return fixedNoonUTC },
			wantDecision: DecisionSuppress,
			wantReason:   ReasonNotContactable,
		},
		{
			name: "Quiet Hours Delay (23:00 local time)",
			context: &PolicyContext{
				SubjectID:        "user-600",
				Channel:          "email",
				ExecutionMode:    ExecutionModeProduction,
				RecipientAddress: "user600@example.com",
				Timezone:         "America/New_York",
			},
			setup: func(mem *MemoryPolicyStore, fl *FrequencyLimiter) {
				mem.SetConsent("user-600", "email", true)
			},
			clock:          func() time.Time { return fixedNightUTC },
			wantDecision:   DecisionDelayUntil,
			wantReason:     ReasonQuietHours,
			checkDelayTime: true,
		},
		{
			name: "Frequency Cap Exceeded (Suppressed)",
			context: &PolicyContext{
				SubjectID:        "user-700",
				Channel:          "push",
				ExecutionMode:    ExecutionModeProduction,
				RecipientAddress: "device-token-123",
				Timezone:         "UTC",
			},
			setup: func(mem *MemoryPolicyStore, fl *FrequencyLimiter) {
				mem.SetConsent("user-700", "push", true)
				fl.SetCap("user-700", "push", 2)
				_, _ = fl.ReserveFrequencyCapacity(context.Background(), "user-700", "push", 2)
			},
			clock:        func() time.Time { return fixedNoonUTC },
			wantDecision: DecisionSuppress,
			wantReason:   ReasonFrequencyCapExceeded,
		},
		{
			name:         "Fail-Closed - Nil Context",
			context:      nil,
			wantDecision: DecisionSuppress,
			wantReason:   ReasonMissingContext,
		},
		{
			name: "Fail-Closed - Missing SubjectID",
			context: &PolicyContext{
				SubjectID:     "",
				Channel:       "email",
				ExecutionMode: ExecutionModeProduction,
				Timezone:      "UTC",
			},
			wantDecision: DecisionSuppress,
			wantReason:   ReasonMissingContext,
		},
		{
			name: "Fail-Closed - Missing Channel",
			context: &PolicyContext{
				SubjectID:     "user-800",
				Channel:       "",
				ExecutionMode: ExecutionModeProduction,
				Timezone:      "UTC",
			},
			wantDecision: DecisionSuppress,
			wantReason:   ReasonMissingContext,
		},
		{
			name: "Fail-Closed - Missing ExecutionMode",
			context: &PolicyContext{
				SubjectID: "user-800",
				Channel:   "email",
				Timezone:  "UTC",
			},
			wantDecision: DecisionSuppress,
			wantReason:   ReasonMissingContext,
		},
		{
			name: "Fail-Closed - Missing Timezone",
			context: &PolicyContext{
				SubjectID:     "user-800",
				Channel:       "email",
				ExecutionMode: ExecutionModeProduction,
				Timezone:      "",
			},
			wantDecision: DecisionSuppress,
			wantReason:   ReasonMissingContext,
		},
		{
			name: "Fail-Closed - Invalid Timezone",
			context: &PolicyContext{
				SubjectID:     "user-800",
				Channel:       "email",
				ExecutionMode: ExecutionModeProduction,
				Timezone:      "Invalid/NonExistent_TZ",
			},
			wantDecision: DecisionSuppress,
			wantReason:   ReasonMissingContext,
		},
		{
			name: "Fail-Closed - Store Error / Context Unavailable",
			context: &PolicyContext{
				SubjectID:     "user-900",
				Channel:       "email",
				ExecutionMode: ExecutionModeProduction,
				Timezone:      "UTC",
			},
			customStore:  true,
			wantDecision: DecisionSuppress,
			wantReason:   ReasonContextUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memStore := NewMemoryPolicyStore()
			fl := NewFrequencyLimiter(10)

			opts := []PolicyServiceOption{
				WithTestListStore(memStore),
				WithConsentStore(memStore),
				WithSuppressionStore(memStore),
				WithContactabilityStore(memStore),
				WithFrequencyLimiter(fl),
			}

			if tt.clock != nil {
				opts = append(opts, WithClock(tt.clock))
			}

			if tt.customStore {
				errStore := &mockErrStore{}
				opts = append(opts,
					WithTestListStore(errStore),
					WithConsentStore(errStore),
					WithSuppressionStore(errStore),
					WithContactabilityStore(errStore),
				)
			}

			if tt.setup != nil {
				tt.setup(memStore, fl)
			}

			service := NewPolicyService(opts...)
			result := service.Evaluate(context.Background(), tt.context)

			if result.Decision != tt.wantDecision {
				t.Errorf("Evaluate() Decision = %v, want %v", result.Decision, tt.wantDecision)
			}
			if result.ReasonCode != tt.wantReason {
				t.Errorf("Evaluate() ReasonCode = %v, want %v", result.ReasonCode, tt.wantReason)
			}
			if result.PolicyVersion != DefaultPolicyVersion {
				t.Errorf("Evaluate() PolicyVersion = %v, want %v", result.PolicyVersion, DefaultPolicyVersion)
			}

			if tt.checkDelayTime {
				if result.DelayUntil == nil {
					t.Fatalf("Evaluate() DelayUntil is nil, expected non-nil time")
				}
				// Verify DelayUntil corresponds to 09:00:00 EDT local time
				loc, _ := time.LoadLocation("America/New_York")
				delayLocal := result.DelayUntil.In(loc)
				if delayLocal.Hour() != 9 || delayLocal.Minute() != 0 {
					t.Errorf("DelayUntil local time = %v, expected 09:00", delayLocal)
				}
			}
		})
	}
}

func TestConcurrentReservationCapacityLimits(t *testing.T) {
	ctx := context.Background()
	fl := NewFrequencyLimiter(5) // Max capacity = 5
	service := NewPolicyService(WithFrequencyLimiter(fl))

	subjectID := "user-concurrent"
	channel := "email"
	fl.SetCap(subjectID, channel, 5)

	numGoroutines := 50
	var wg sync.WaitGroup
	var successCount int64
	var failCount int64

	reservations := make(chan string, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resID, err := service.ReserveFrequencyCapacity(ctx, subjectID, channel, 1)
			if err == nil {
				atomic.AddInt64(&successCount, 1)
				reservations <- resID
			} else {
				if errors.Is(err, ErrFrequencyCapExceeded) {
					atomic.AddInt64(&failCount, 1)
				} else {
					t.Errorf("Unexpected error during capacity reservation: %v", err)
				}
			}
		}()
	}

	wg.Wait()
	close(reservations)

	if successCount != 5 {
		t.Errorf("Expected exactly 5 successful reservations, got %d", successCount)
	}
	if failCount != int64(numGoroutines-5) {
		t.Errorf("Expected %d failed reservations, got %d", numGoroutines-5, failCount)
	}

	// Verify capacity is full now
	hasCap, err := fl.HasCapacity(ctx, subjectID, channel, 1)
	if err != nil || hasCap {
		t.Errorf("Expected capacity to be fully exhausted, got hasCap=%v, err=%v", hasCap, err)
	}

	// Release all successful reservations
	for resID := range reservations {
		if err := service.ReleaseFrequencyCapacity(ctx, resID); err != nil {
			t.Errorf("Failed to release reservation %s: %v", resID, err)
		}
	}

	// Verify capacity is completely freed
	hasCapAfterRelease, err := fl.HasCapacity(ctx, subjectID, channel, 5)
	if err != nil || !hasCapAfterRelease {
		t.Errorf("Expected full capacity available after release, got hasCap=%v, err=%v", hasCapAfterRelease, err)
	}

	// Try reserving again after release
	newResID, err := service.ReserveFrequencyCapacity(ctx, subjectID, channel, 5)
	if err != nil || newResID == "" {
		t.Errorf("Failed to reserve capacity after full release: %v", err)
	}
}
