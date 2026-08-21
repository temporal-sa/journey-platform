package activities

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrSimulatedActivityFailure is the default error returned by error injection behaviors.
var ErrSimulatedActivityFailure = errors.New("simulated activity execution failure")

// ActivityInfo captures execution metadata passed to behaviors during activity interception.
type ActivityInfo struct {
	ActivityName string                 `json:"activity_name"`
	NodeID       string                 `json:"node_id,omitempty"`
	TenantID     string                 `json:"tenant_id,omitempty"`
	WorkflowID   string                 `json:"workflow_id,omitempty"`
	Payload      map[string]interface{} `json:"payload,omitempty"`
}

// ActivityBehavior defines the contract for runtime activity interceptors.
type ActivityBehavior interface {
	// Match returns true if this behavior should execute for the given activity invocation.
	Match(info ActivityInfo) bool
	// Execute performs the behavior (e.g. latency, error, custom side effect).
	Execute(ctx context.Context, info ActivityInfo) error
}

// ErrorBehavior injects an error into matching activities.
type ErrorBehavior struct {
	TargetActivity string `json:"target_activity,omitempty"`
	TargetNodeID   string `json:"target_node_id,omitempty"`
	Err            error  `json:"-"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

// NewErrorBehavior creates an ErrorBehavior that matches all activities or specific targets.
func NewErrorBehavior(errMsg string, targetActivity, targetNodeID string) *ErrorBehavior {
	var err error
	if errMsg != "" {
		err = errors.New(errMsg)
	} else {
		err = ErrSimulatedActivityFailure
	}
	return &ErrorBehavior{
		TargetActivity: targetActivity,
		TargetNodeID:   targetNodeID,
		Err:            err,
		ErrorMessage:   err.Error(),
	}
}

func (b *ErrorBehavior) Match(info ActivityInfo) bool {
	if b.TargetActivity != "" && b.TargetActivity != info.ActivityName {
		return false
	}
	if b.TargetNodeID != "" && b.TargetNodeID != info.NodeID {
		return false
	}
	return true
}

func (b *ErrorBehavior) Execute(ctx context.Context, info ActivityInfo) error {
	if b.Err != nil {
		return b.Err
	}
	return ErrSimulatedActivityFailure
}

// DelayBehavior injects latency/sleep before activity execution with context awareness.
type DelayBehavior struct {
	Duration       time.Duration `json:"duration"`
	TargetActivity string        `json:"target_activity,omitempty"`
	TargetNodeID   string        `json:"target_node_id,omitempty"`
}

// NewDelayBehavior creates a DelayBehavior that sleeps for the specified duration.
func NewDelayBehavior(d time.Duration, targetActivity, targetNodeID string) *DelayBehavior {
	return &DelayBehavior{
		Duration:       d,
		TargetActivity: targetActivity,
		TargetNodeID:   targetNodeID,
	}
}

func (b *DelayBehavior) Match(info ActivityInfo) bool {
	if b.Duration <= 0 {
		return false
	}
	if b.TargetActivity != "" && b.TargetActivity != info.ActivityName {
		return false
	}
	if b.TargetNodeID != "" && b.TargetNodeID != info.NodeID {
		return false
	}
	return true
}

func (b *DelayBehavior) Execute(ctx context.Context, info ActivityInfo) error {
	if b.Duration <= 0 {
		return nil
	}
	select {
	case <-time.After(b.Duration):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// HookBehavior runs an arbitrary user-supplied callback during activity interception.
type HookBehavior struct {
	TargetActivity string                                                `json:"target_activity,omitempty"`
	TargetNodeID   string                                                `json:"target_node_id,omitempty"`
	Hook           func(ctx context.Context, info ActivityInfo) error   `json:"-"`
}

// NewHookBehavior creates a HookBehavior with a custom execution callback.
func NewHookBehavior(hook func(ctx context.Context, info ActivityInfo) error, targetActivity, targetNodeID string) *HookBehavior {
	return &HookBehavior{
		TargetActivity: targetActivity,
		TargetNodeID:   targetNodeID,
		Hook:           hook,
	}
}

func (b *HookBehavior) Match(info ActivityInfo) bool {
	if b.Hook == nil {
		return false
	}
	if b.TargetActivity != "" && b.TargetActivity != info.ActivityName {
		return false
	}
	if b.TargetNodeID != "" && b.TargetNodeID != info.NodeID {
		return false
	}
	return true
}

func (b *HookBehavior) Execute(ctx context.Context, info ActivityInfo) error {
	if b.Hook != nil {
		return b.Hook(ctx, info)
	}
	return nil
}

// BehaviorInjector manages thread-safe registration and execution of runtime activity behaviors.
type BehaviorInjector struct {
	mu        sync.RWMutex
	behaviors []ActivityBehavior
}

var (
	defaultInjectorOnce sync.Once
	defaultInjector     *BehaviorInjector
)

// DefaultBehaviorInjector returns the package-level shared BehaviorInjector singleton.
func DefaultBehaviorInjector() *BehaviorInjector {
	defaultInjectorOnce.Do(func() {
		defaultInjector = NewBehaviorInjector()
	})
	return defaultInjector
}

// NewBehaviorInjector creates an isolated BehaviorInjector instance.
func NewBehaviorInjector() *BehaviorInjector {
	return &BehaviorInjector{
		behaviors: make([]ActivityBehavior, 0),
	}
}

// AddBehavior registers an activity behavior at runtime.
func (bi *BehaviorInjector) AddBehavior(b ActivityBehavior) {
	if b == nil {
		return
	}
	bi.mu.Lock()
	defer bi.mu.Unlock()
	bi.behaviors = append(bi.behaviors, b)
}

// Clear removes all registered behaviors.
func (bi *BehaviorInjector) Clear() {
	bi.mu.Lock()
	defer bi.mu.Unlock()
	bi.behaviors = make([]ActivityBehavior, 0)
}

// Behaviors returns a copy of all registered behaviors.
func (bi *BehaviorInjector) Behaviors() []ActivityBehavior {
	bi.mu.RLock()
	defer bi.mu.RUnlock()
	res := make([]ActivityBehavior, len(bi.behaviors))
	copy(res, bi.behaviors)
	return res
}

// SetFailureEnabled toggles global error injection in the injector.
func (bi *BehaviorInjector) SetFailureEnabled(enabled bool) {
	bi.mu.Lock()
	defer bi.mu.Unlock()

	// Filter out any existing global error behaviors
	filtered := make([]ActivityBehavior, 0, len(bi.behaviors))
	for _, b := range bi.behaviors {
		if eb, ok := b.(*ErrorBehavior); ok && eb.TargetActivity == "" && eb.TargetNodeID == "" {
			continue
		}
		filtered = append(filtered, b)
	}

	if enabled {
		filtered = append(filtered, NewErrorBehavior("", "", ""))
	}
	bi.behaviors = filtered
}

// IsFailureEnabled checks if global error injection is active in the injector.
func (bi *BehaviorInjector) IsFailureEnabled() bool {
	bi.mu.RLock()
	defer bi.mu.RUnlock()
	for _, b := range bi.behaviors {
		if eb, ok := b.(*ErrorBehavior); ok && eb.TargetActivity == "" && eb.TargetNodeID == "" {
			return true
		}
	}
	return false
}
// SetDelay sets a global delay duration on all activities (0 disables delay).
func (bi *BehaviorInjector) SetDelay(d time.Duration) {
	bi.mu.Lock()
	defer bi.mu.Unlock()

	// Remove existing global delay behaviors
	filtered := make([]ActivityBehavior, 0, len(bi.behaviors))
	for _, b := range bi.behaviors {
		if db, ok := b.(*DelayBehavior); ok && db.TargetActivity == "" && db.TargetNodeID == "" {
			continue
		}
		filtered = append(filtered, b)
	}

	if d > 0 {
		filtered = append(filtered, NewDelayBehavior(d, "", ""))
	}
	bi.behaviors = filtered
}

// GetDelay returns the current global delay duration (0 if none).
func (bi *BehaviorInjector) GetDelay() time.Duration {
	bi.mu.RLock()
	defer bi.mu.RUnlock()
	for _, b := range bi.behaviors {
		if db, ok := b.(*DelayBehavior); ok && db.TargetActivity == "" && db.TargetNodeID == "" {
			return db.Duration
		}
	}
	return 0
}

// SetDelayMS sets the global delay duration in milliseconds (0 disables delay).
func (bi *BehaviorInjector) SetDelayMS(ms int) {
	if ms < 0 {
		ms = 0
	}
	bi.SetDelay(time.Duration(ms) * time.Millisecond)
}

// GetDelayMS returns the current global delay duration in milliseconds.
func (bi *BehaviorInjector) GetDelayMS() int {
	return int(bi.GetDelay() / time.Millisecond)
}

// SetSimulatedActivityLatencyMS sets the in-memory simulated activity latency in milliseconds on the default injector.
func SetSimulatedActivityLatencyMS(ms int) {
	DefaultBehaviorInjector().SetDelayMS(ms)
}

// GetSimulatedActivityLatencyMS returns the in-memory simulated activity latency in milliseconds from the default injector.
func GetSimulatedActivityLatencyMS() int {
	return DefaultBehaviorInjector().GetDelayMS()
}

// Intercept executes all matching behaviors against the activity info.
// Returns immediately if any behavior yields an error.
func (bi *BehaviorInjector) Intercept(ctx context.Context, info ActivityInfo) error {
	bi.mu.RLock()
	behaviors := make([]ActivityBehavior, len(bi.behaviors))
	copy(behaviors, bi.behaviors)
	bi.mu.RUnlock()

	for _, b := range behaviors {
		if b.Match(info) {
			if err := b.Execute(ctx, info); err != nil {
				return err
			}
		}
	}
	return nil
}
