package workflows

import (
	"fmt"
	"strings"
	"time"

	"github.com/validated-pattern/journey-platform/internal/activities"
	"github.com/validated-pattern/journey-platform/internal/domain"
	"go.temporal.io/sdk/workflow"
)

// EventSignal represents an incoming event signal received via "journey.signal.event".
type EventSignal struct {
	SchemaVersion string                 `json:"schema_version,omitempty" temporal_history:"allowed"`
	EventID       string                 `json:"event_id" temporal_history:"allowed"`
	EventType     string                 `json:"event_type" temporal_history:"allowed"`
	Generation    int                    `json:"generation,omitempty" temporal_history:"allowed"`
	PayloadRef    string                 `json:"payload_ref,omitempty" temporal_history:"allowed"`
	Payload       map[string]interface{} `json:"payload,omitempty" temporal_history:"prohibited"`
	ReceivedAt    time.Time              `json:"received_at,omitempty" temporal_history:"allowed"`
}

// WaitForEventResult represents the outcome of executing a WaitForEvent node.
type WaitForEventResult struct {
	Status     string                 `json:"status" temporal_history:"allowed"` // "matched", "timed_out", "cancelled", "continue_as_new"
	EventID    string                 `json:"event_id,omitempty" temporal_history:"allowed"`
	EventType  string                 `json:"event_type,omitempty" temporal_history:"allowed"`
	PayloadRef string                 `json:"payload_ref,omitempty" temporal_history:"allowed"`
	Payload    map[string]interface{} `json:"payload,omitempty" temporal_history:"prohibited"`
	MatchedAt  time.Time              `json:"matched_at,omitempty" temporal_history:"allowed"`
	Error      error                  `json:"-"`
}

// WaitForEventConfig holds runtime settings for a WaitForEvent node execution step.
type WaitForEventConfig struct {
	TenantID        string
	WorkflowID      string
	RunID           string
	NodeID          string
	Generation      int
	EventType       string
	ConditionExpr   string
	TimeoutDuration time.Duration
}

// ExtractWaitForEventConfig parses node parameters into a WaitForEventConfig struct.
func ExtractWaitForEventConfig(node domain.IRNode, tenantID, workflowID, runID string, generation int) WaitForEventConfig {
	cfg := WaitForEventConfig{
		TenantID:   tenantID,
		WorkflowID: workflowID,
		RunID:      runID,
		NodeID:     node.ID,
		Generation: generation,
	}

	if generation <= 0 {
		cfg.Generation = 1
	}

	if node.Params != nil {
		for _, key := range []string{"event_type", "event_name", "type", "event"} {
			if val, ok := node.Params[key]; ok {
				cfg.EventType = fmt.Sprintf("%v", val)
				break
			}
		}
		for _, key := range []string{"condition_expression", "condition_expr", "condition"} {
			if val, ok := node.Params[key]; ok {
				cfg.ConditionExpr = fmt.Sprintf("%v", val)
				break
			}
		}
		sec := extractDelaySeconds(node.Params)
		if sec <= 0 && node.TimeoutSeconds > 0 {
			sec = int64(node.TimeoutSeconds)
		}
		if sec > 0 {
			cfg.TimeoutDuration = time.Duration(sec) * time.Second
		} else {
			cfg.TimeoutDuration = 86400 * time.Second
		}
	}

	if cfg.NodeID == "" {
		cfg.NodeID = "wait_for_event"
	}
	return cfg
}

// ExecuteWaitForEventNode executes the WaitForEvent node lifecycle:
// 1. Creates a generation-scoped subscription row via Activity before waiting.
// 2. Checks pre-buffered events and deduplicates seen event IDs.
// 3. Races signal channel and durable timer using a deterministic cutoff rule.
// 4. Closes subscription idempotently using workflow.NewDisconnectedContext on match, timeout, cancellation, or Continue-As-New.
func ExecuteWaitForEventNode(
	ctx workflow.Context,
	act *activities.Activities,
	cfg WaitForEventConfig,
	bufferedEvents map[string]EventSignal,
	seenEventIDs map[string]bool,
	variables map[string]interface{},
) (WaitForEventResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting ExecuteWaitForEventNode", "nodeID", cfg.NodeID, "eventType", cfg.EventType, "generation", cfg.Generation)

	subID := fmt.Sprintf("sub-%s-%d-%s", cfg.WorkflowID, cfg.Generation, cfg.NodeID)

	// Activity options for subscription management
	actCtx := WithActivitySummary(ctx, fmt.Sprintf("Create event subscription for event %s on node %s", cfg.EventType, cfg.NodeID))

	// 1. Create generation-scoped subscription row via Activity BEFORE waiting
	createSubInput := activities.CreateSubscriptionInput{
		TenantID:       cfg.TenantID,
		SubscriptionID: subID,
		RunID:          cfg.RunID,
		WorkflowID:     cfg.WorkflowID,
		NodeID:         cfg.NodeID,
		EventType:      cfg.EventType,
		ConditionExpr:  cfg.ConditionExpr,
		Generation:     cfg.Generation,
	}

	err := workflow.ExecuteActivity(actCtx, act.CreateSubscription, createSubInput).Get(actCtx, nil)
	if err != nil {
		logger.Error("Failed to create subscription", "subID", subID, "error", err)
	}

	// Track cleanup reason to close subscription idempotently
	closeReason := "timeout"
	subscriptionClosed := false

	closeSubFunc := func(reason string) {
		if subscriptionClosed {
			return
		}
		subscriptionClosed = true
		disconnectedCtx, _ := workflow.NewDisconnectedContext(ctx)
		discActCtx := WithActivitySummary(disconnectedCtx, fmt.Sprintf("Close event subscription %s (reason: %s)", subID, reason))
		closeInput := activities.CloseSubscriptionInput{
			TenantID:       cfg.TenantID,
			SubscriptionID: subID,
			Reason:         reason,
		}
		_ = workflow.ExecuteActivity(discActCtx, act.CloseSubscription, closeInput).Get(discActCtx, nil)
	}

	defer func() {
		closeSubFunc(closeReason)
	}()

	// Drain any signals sitting in eventSignalChan into bufferedEvents
	eventSignalChan := workflow.GetSignalChannel(ctx, SignalNameEvent)
	DrainEventSignals(ctx, eventSignalChan, bufferedEvents, seenEventIDs, cfg.Generation)

	// 2. Check pre-buffered events (event before wait)
	for eventID, evt := range bufferedEvents {
		if evt.Generation != 0 && evt.Generation < cfg.Generation {
			logger.Info("Discarding buffered event from old generation", "eventID", eventID, "eventGen", evt.Generation, "currentGen", cfg.Generation)
			delete(bufferedEvents, eventID)
			continue
		}

		if seenEventIDs[eventID] {
			delete(bufferedEvents, eventID)
			continue
		}

		if matchesEvent(ctx, act, evt, cfg, variables) {
			logger.Info("Matched pre-buffered event", "eventID", eventID, "nodeID", cfg.NodeID)
			seenEventIDs[eventID] = true
			delete(bufferedEvents, eventID)
			closeReason = "match"
			return WaitForEventResult{
				Status:     "matched",
				EventID:    evt.EventID,
				EventType:  evt.EventType,
				PayloadRef: evt.PayloadRef,
				Payload:    evt.Payload,
				MatchedAt:  workflow.Now(ctx),
			}, nil
		}
	}

	// 3. Setup signal channels and durable timer for racing
	cancelSignalChan := workflow.GetSignalChannel(ctx, SignalNameCancel)

	var timerFuture workflow.Future
	timerFired := false
	startTime := workflow.Now(ctx)
	var deadline time.Time

	if cfg.TimeoutDuration > 0 {
		timerFuture = workflow.NewTimer(ctx, cfg.TimeoutDuration)
		deadline = startTime.Add(cfg.TimeoutDuration)
	}

	selector := workflow.NewSelector(ctx)
	var matchedEvent *EventSignal
	isCancelled := false

	if timerFuture != nil {
		selector.AddFuture(timerFuture, func(f workflow.Future) {
			timerFired = true
		})
	}

	selector.AddReceive(cancelSignalChan, func(c workflow.ReceiveChannel, more bool) {
		var sig domain.WorkflowSignal
		c.Receive(ctx, &sig)
		isCancelled = true
	})

	var pendingSignal *EventSignal

	selector.AddReceive(eventSignalChan, func(c workflow.ReceiveChannel, more bool) {
		var raw interface{}
		c.Receive(ctx, &raw)
		sig := convertToEventSignal(raw)
		pendingSignal = &sig
	})

	// Wait loop with deterministic cutoff rule
	for matchedEvent == nil && !timerFired && !isCancelled {
		selector.Select(ctx)

		if isCancelled {
			closeReason = "cancellation"
			return WaitForEventResult{Status: "cancelled"}, nil
		}

		if timerFired {
			closeReason = "timeout"
			return WaitForEventResult{Status: "timed_out"}, nil
		}

		if pendingSignal != nil {
			sig := *pendingSignal
			pendingSignal = nil

			// Check old generation signal rejection
			if sig.Generation != 0 && sig.Generation < cfg.Generation {
				logger.Info("Rejected signal from old generation", "eventID", sig.EventID, "signalGen", sig.Generation, "currentGen", cfg.Generation)
				continue
			}

			// Check event deduplication
			if seenEventIDs[sig.EventID] {
				logger.Info("Deduplicated incoming signal event", "eventID", sig.EventID)
				continue
			}

			// Evaluate event matching
			if matchesEvent(ctx, act, sig, cfg, variables) {
				// Deterministic Cutoff Rule check:
				// If workflow logical time has passed deadline, timer cutoff wins.
				now := workflow.Now(ctx)
				if cfg.TimeoutDuration > 0 && !now.Before(deadline) && now != deadline {
					logger.Info("Signal received after deterministic cutoff deadline, marking as timed out", "now", now, "deadline", deadline)
					closeReason = "timeout"
					return WaitForEventResult{Status: "timed_out"}, nil
				}

				seenEventIDs[sig.EventID] = true
				closeReason = "match"
				matchedEvent = &sig
				break
			} else {
				// Buffer unmatched event for subsequent nodes
				bufferedEvents[sig.EventID] = sig
			}
		}
	}

	if matchedEvent != nil {
		return WaitForEventResult{
			Status:     "matched",
			EventID:    matchedEvent.EventID,
			EventType:  matchedEvent.EventType,
			PayloadRef: matchedEvent.PayloadRef,
			Payload:    matchedEvent.Payload,
			MatchedAt:  workflow.Now(ctx),
		}, nil
	}

	if timerFired {
		closeReason = "timeout"
		return WaitForEventResult{Status: "timed_out"}, nil
	}

	closeReason = "cancellation"
	return WaitForEventResult{Status: "cancelled"}, nil
}

// Helpers

func DrainEventSignals(ctx workflow.Context, eventSignalChan workflow.ReceiveChannel, bufferedEvents map[string]EventSignal, seenEventIDs map[string]bool, currentGen int) {
	if eventSignalChan == nil {
		return
	}
	for {
		var raw interface{}
		ok := eventSignalChan.ReceiveAsync(&raw)
		if !ok {
			break
		}
		sig := convertToEventSignal(raw)
		if sig.Generation != 0 && sig.Generation < currentGen {
			continue
		}
		if seenEventIDs != nil && seenEventIDs[sig.EventID] {
			continue
		}
		if bufferedEvents != nil && sig.EventID != "" {
			bufferedEvents[sig.EventID] = sig
		}
	}
}

func matchesEvent(ctx workflow.Context, act *activities.Activities, evt EventSignal, cfg WaitForEventConfig, variables map[string]interface{}) bool {
	if cfg.EventType != "" && evt.EventType != "" && !strings.EqualFold(cfg.EventType, evt.EventType) {
		return false
	}

	expr := strings.TrimSpace(cfg.ConditionExpr)
	if expr == "" || expr == "true" || expr == "default" {
		return true
	}

	evalCtx := make(map[string]interface{})
	for k, v := range variables {
		evalCtx[k] = v
	}
	if evt.Payload != nil {
		for k, v := range evt.Payload {
			evalCtx[k] = v
		}
	}
	evalCtx["event_id"] = evt.EventID
	evalCtx["event_type"] = evt.EventType

	condInput := activities.EvaluateConditionInput{
		ConditionExpression: expr,
		Context:             evalCtx,
	}
	var condResult bool
	evalActCtx := WithActivitySummary(ctx, fmt.Sprintf("Evaluate event condition '%s' for %s on node %s", expr, cfg.EventType, cfg.NodeID))
	err := workflow.ExecuteActivity(evalActCtx, act.EvaluateCondition, condInput).Get(evalActCtx, &condResult)
	if err != nil {
		return false
	}
	return condResult
}

func convertToEventSignal(raw interface{}) EventSignal {
	if raw == nil {
		return EventSignal{}
	}
	switch v := raw.(type) {
	case EventSignal:
		return v
	case *EventSignal:
		if v != nil {
			return *v
		}
	case domain.WorkflowSignal:
		es := EventSignal{
			SchemaVersion: v.SchemaVersion,
			Payload:       v.Payload,
		}
		if v.Payload != nil {
			if id, ok := v.Payload["event_id"].(string); ok {
				es.EventID = id
			}
			if et, ok := v.Payload["event_type"].(string); ok {
				es.EventType = et
			}
			if gen, ok := v.Payload["generation"].(float64); ok {
				es.Generation = int(gen)
			} else if genInt, ok := v.Payload["generation"].(int); ok {
				es.Generation = genInt
			}
			if pr, ok := v.Payload["payload_ref"].(string); ok {
				es.PayloadRef = pr
			}
		}
		if es.EventID == "" {
			es.EventID = fmt.Sprintf("evt-%s-%d", es.EventType, es.Generation)
		}
		return es
	case map[string]interface{}:
		es := EventSignal{}
		if p, ok := v["payload"].(map[string]interface{}); ok {
			es.Payload = p
		} else {
			es.Payload = v
		}
		if id, ok := v["event_id"].(string); ok {
			es.EventID = id
		}
		if et, ok := v["event_type"].(string); ok {
			es.EventType = et
		}
		if gen, ok := v["generation"].(float64); ok {
			es.Generation = int(gen)
		} else if genInt, ok := v["generation"].(int); ok {
			es.Generation = genInt
		}
		if pr, ok := v["payload_ref"].(string); ok {
			es.PayloadRef = pr
		}
		if es.EventID == "" {
			es.EventID = fmt.Sprintf("evt-%s-%d", es.EventType, es.Generation)
		}
		return es
	}
	return EventSignal{}
}
