package workflows

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/validated-pattern/journey-platform/internal/activities"
	"github.com/validated-pattern/journey-platform/internal/domain"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type ExecutionMode string

const (
	ExecutionModeProduction ExecutionMode = "production"
	ExecutionModeTest       ExecutionMode = "test"
)

type TerminalStatus string

const (
	StatusSucceeded       TerminalStatus = "succeeded"
	StatusSuppressed      TerminalStatus = "suppressed"
	StatusInvalidArtifact TerminalStatus = "invalid_artifact"
	StatusCancelled       TerminalStatus = "cancelled"
	StatusFailed          TerminalStatus = "failed"
)

const (
	QueryTypeState = "journey.query.state"
)

// CompiledJourneyInput holds input parameters for executing a compiled journey workflow.
type CompiledJourneyInput struct {
	SchemaVersion      string                    `json:"schema_version" temporal_history:"allowed"`
	WorkflowID         string                    `json:"workflow_id" temporal_history:"allowed"`
	RunID              string                    `json:"run_id" temporal_history:"allowed"`
	TenantID           string                    `json:"tenant_id" temporal_history:"allowed"`
	TriggerEventID     string                    `json:"trigger_event_id,omitempty" temporal_history:"allowed"`
	IRID               string                    `json:"ir_id,omitempty" temporal_history:"allowed"`
	ContentHash        string                    `json:"content_hash" temporal_history:"allowed"`
	ExecutionMode      ExecutionMode             `json:"execution_mode" temporal_history:"allowed"`
	DataClassification domain.DataClassification `json:"data_classification" temporal_history:"allowed"`
	InputPayload       map[string]interface{}    `json:"input_payload,omitempty" temporal_history:"prohibited"`
	ExperimentContext  map[string]string         `json:"experiment_context,omitempty" temporal_history:"allowed"`
	Generation         int                       `json:"generation,omitempty" temporal_history:"allowed"`
	InitialState       *CompiledJourneyState     `json:"initial_state,omitempty" temporal_history:"allowed"`
}

// CompiledJourneyState holds compact, isolated history workflow state.
type CompiledJourneyState struct {
	CurrentNodeID     string                 `json:"current_node_id" temporal_history:"allowed"`
	VisitCounts       map[string]int         `json:"visit_counts" temporal_history:"allowed"`
	NodeOutputs       map[string]interface{} `json:"node_outputs,omitempty" temporal_history:"allowed"`
	ExperimentContext map[string]string      `json:"experiment_context,omitempty" temporal_history:"allowed"`
	ExecutionMode     ExecutionMode          `json:"execution_mode" temporal_history:"allowed"`
	Status            TerminalStatus         `json:"status" temporal_history:"allowed"`
	Generation        int                    `json:"generation,omitempty" temporal_history:"allowed"`
	Variables         map[string]interface{} `json:"variables,omitempty" temporal_history:"allowed"`
	SeenEventIDs      map[string]bool        `json:"seen_event_ids,omitempty" temporal_history:"allowed"`
	BufferedEvents    map[string]EventSignal `json:"buffered_events,omitempty" temporal_history:"allowed"`
}

// CompiledJourneyResult represents the terminal result of the compiled workflow.
type CompiledJourneyResult struct {
	RunID             string                 `json:"run_id" temporal_history:"allowed"`
	WorkflowID        string                 `json:"workflow_id" temporal_history:"allowed"`
	Status            TerminalStatus         `json:"status" temporal_history:"allowed"`
	CurrentNodeID     string                 `json:"current_node_id,omitempty" temporal_history:"allowed"`
	VisitCounts       map[string]int         `json:"visit_counts,omitempty" temporal_history:"allowed"`
	NodeOutputs       map[string]interface{} `json:"node_outputs,omitempty" temporal_history:"allowed"`
	ExperimentContext map[string]string      `json:"experiment_context,omitempty" temporal_history:"allowed"`
	ErrorMessage      string                 `json:"error_message,omitempty" temporal_history:"allowed"`
}

// CompiledJourneyWorkflow executes a compiled journey graph deterministically.
func CompiledJourneyWorkflow(ctx workflow.Context, input CompiledJourneyInput) (*CompiledJourneyResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting CompiledJourneyWorkflow", "workflowID", input.WorkflowID, "runID", input.RunID, "contentHash", input.ContentHash)

	if input.WorkflowID == "" || input.RunID == "" || input.ContentHash == "" {
		return &CompiledJourneyResult{
			RunID:        input.RunID,
			WorkflowID:   input.WorkflowID,
			Status:       StatusInvalidArtifact,
			ErrorMessage: "invalid input: missing required workflowID, runID, or contentHash",
		}, nil
	}

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    100 * time.Millisecond,
			BackoffCoefficient: 2.0,
			MaximumInterval:    1 * time.Second,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	act := &activities.Activities{}

	// Load Compiled IR by SHA-256 hash & verify integrity
	loadInput := activities.LoadCompiledIRInput{
		ContentHash: input.ContentHash,
		IRID:        input.IRID,
		TenantID:    input.TenantID,
	}
	var ir domain.CompiledIR
	loadAo := workflow.ActivityOptions{
		Summary:             fmt.Sprintf("Load compiled IR graph artifact for draft %s", input.ContentHash),
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    100 * time.Millisecond,
			BackoffCoefficient: 2.0,
			MaximumInterval:    1 * time.Second,
			MaximumAttempts:    3,
		},
	}
	loadCtx := workflow.WithActivityOptions(ctx, loadAo)
	err := workflow.ExecuteActivity(loadCtx, act.LoadCompiledIR, loadInput).Get(loadCtx, &ir)
	if err != nil {
		logger.Error("Failed to load compiled IR", "error", err)
		_ = workflow.ExecuteActivity(WithActivitySummary(ctx, "Emit invalid_artifact lifecycle event"), act.EmitLifecycleEvent, activities.LifecycleEvent{
			SchemaVersion: domain.DefaultSchemaVersion,
			RunID:         input.RunID,
			TenantID:      input.TenantID,
			WorkflowID:    input.WorkflowID,
			EventType:     "invalid_artifact",
			Status:        string(StatusInvalidArtifact),
			Timestamp:     workflow.Now(ctx),
			Metadata:      map[string]interface{}{"error": err.Error()},
		}).Get(ctx, nil)
		return &CompiledJourneyResult{
			RunID:        input.RunID,
			WorkflowID:   input.WorkflowID,
			Status:       StatusInvalidArtifact,
			ErrorMessage: fmt.Sprintf("failed to load compiled IR: %v", err),
		}, fmt.Errorf("failed to load compiled IR: %w", err)
	}

	// Initialize Compact State
	generation := input.Generation
	if generation <= 0 {
		generation = 1
	}

	var state *CompiledJourneyState
	if input.InitialState != nil {
		state = input.InitialState
		if state.Generation <= 0 {
			state.Generation = generation
		}
		if state.VisitCounts == nil {
			state.VisitCounts = make(map[string]int)
		}
		if state.NodeOutputs == nil {
			state.NodeOutputs = make(map[string]interface{})
		}
		if state.ExperimentContext == nil {
			state.ExperimentContext = make(map[string]string)
		}
		if state.Variables == nil {
			state.Variables = make(map[string]interface{})
		}
		if state.SeenEventIDs == nil {
			state.SeenEventIDs = make(map[string]bool)
		}
		if state.BufferedEvents == nil {
			state.BufferedEvents = make(map[string]EventSignal)
		}
	} else {
		state = &CompiledJourneyState{
			CurrentNodeID:     ir.EntryNodeID,
			VisitCounts:       make(map[string]int),
			NodeOutputs:       make(map[string]interface{}),
			ExperimentContext: make(map[string]string),
			ExecutionMode:     input.ExecutionMode,
			Status:            StatusSucceeded,
			Generation:        generation,
			Variables:         make(map[string]interface{}),
			SeenEventIDs:      make(map[string]bool),
			BufferedEvents:    make(map[string]EventSignal),
		}
	}

	if state.ExecutionMode == "" {
		state.ExecutionMode = ExecutionModeProduction
	}
	if input.ExperimentContext != nil {
		for k, v := range input.ExperimentContext {
			state.ExperimentContext[k] = v
		}
	}
	if input.InputPayload != nil {
		for k, v := range input.InputPayload {
			if _, ok := state.Variables[k]; !ok {
				state.Variables[k] = v
			}
		}
	}

	// Query Handlers
	err = workflow.SetQueryHandler(ctx, QueryTypeStatus, func() (*CompiledJourneyResult, error) {
		return &CompiledJourneyResult{
			RunID:             input.RunID,
			WorkflowID:        input.WorkflowID,
			Status:            state.Status,
			CurrentNodeID:     state.CurrentNodeID,
			VisitCounts:       state.VisitCounts,
			NodeOutputs:       state.NodeOutputs,
			ExperimentContext: state.ExperimentContext,
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set status query handler: %w", err)
	}

	err = workflow.SetQueryHandler(ctx, QueryTypeVariables, func() (map[string]interface{}, error) {
		varsCopy := make(map[string]interface{})
		if state.Variables != nil {
			for k, v := range state.Variables {
				varsCopy[k] = v
			}
		}
		return varsCopy, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set variables query handler: %w", err)
	}

	err = workflow.SetQueryHandler(ctx, QueryTypeState, func() (*CompiledJourneyState, error) {
		stateCopy := *state
		return &stateCopy, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set state query handler: %w", err)
	}

	// Signal Channels
	cancelSignalChan := workflow.GetSignalChannel(ctx, SignalNameCancel)
	isCancelled := false

	// Emit workflow started lifecycle event
	_ = workflow.ExecuteActivity(WithActivitySummary(ctx, "Emit workflow_started lifecycle event"), act.EmitLifecycleEvent, activities.LifecycleEvent{
		SchemaVersion: domain.DefaultSchemaVersion,
		RunID:         input.RunID,
		TenantID:      input.TenantID,
		WorkflowID:    input.WorkflowID,
		EventType:     "workflow_started",
		Status:        "running",
		Timestamp:     workflow.Now(ctx),
	}).Get(ctx, nil)

	// Resolve entry node
	currentNodeID := ir.EntryNodeID
	if currentNodeID == "" && len(ir.Nodes) > 0 {
		for _, n := range ir.Nodes {
			t := strings.ToLower(strings.TrimSpace(n.Type))
			if t == "event_start" || t == "start" || t == "trigger" {
				currentNodeID = n.ID
				break
			}
		}
		if currentNodeID == "" {
			currentNodeID = ir.Nodes[0].ID
		}
	}

	nodeMap := make(map[string]domain.IRNode, len(ir.Nodes))
	for _, n := range ir.Nodes {
		nodeMap[n.ID] = n
	}

	// Graph execution loop
	for currentNodeID != "" {
		// Check cancellation signal or context cancellation
		if cancelSignalChan.ReceiveAsync(&domain.WorkflowSignal{}) {
			isCancelled = true
		}
		if isCancelled {
			state.Status = StatusCancelled
			_ = workflow.ExecuteActivity(WithActivitySummary(ctx, "Emit workflow_cancelled lifecycle event"), act.EmitLifecycleEvent, activities.LifecycleEvent{
				SchemaVersion: domain.DefaultSchemaVersion,
				RunID:         input.RunID,
				TenantID:      input.TenantID,
				WorkflowID:    input.WorkflowID,
				EventType:     "workflow_cancelled",
				Status:        string(StatusCancelled),
				Timestamp:     workflow.Now(ctx),
			}).Get(ctx, nil)

			return &CompiledJourneyResult{
				RunID:             input.RunID,
				WorkflowID:        input.WorkflowID,
				Status:            StatusCancelled,
				CurrentNodeID:     currentNodeID,
				VisitCounts:       state.VisitCounts,
				NodeOutputs:       state.NodeOutputs,
				ExperimentContext: state.ExperimentContext,
			}, nil
		}

		node, ok := nodeMap[currentNodeID]
		if !ok {
			logger.Error("Node not found in IR", "nodeID", currentNodeID)
			state.Status = StatusFailed
			_ = workflow.ExecuteActivity(WithActivitySummary(ctx, fmt.Sprintf("Emit workflow_failed lifecycle event for node %s", currentNodeID)), act.EmitLifecycleEvent, activities.LifecycleEvent{
				SchemaVersion: domain.DefaultSchemaVersion,
				RunID:         input.RunID,
				TenantID:      input.TenantID,
				WorkflowID:    input.WorkflowID,
				EventType:     "workflow_failed",
				NodeID:        currentNodeID,
				Status:        string(StatusFailed),
				Timestamp:     workflow.Now(ctx),
				Metadata:      map[string]interface{}{"error": fmt.Sprintf("node not found: %s", currentNodeID)},
			}).Get(ctx, nil)

			return &CompiledJourneyResult{
				RunID:             input.RunID,
				WorkflowID:        input.WorkflowID,
				Status:            StatusFailed,
				CurrentNodeID:     currentNodeID,
				VisitCounts:       state.VisitCounts,
				ExperimentContext: state.ExperimentContext,
				ErrorMessage:      fmt.Sprintf("node not found in IR graph: %s", currentNodeID),
			}, fmt.Errorf("node not found in IR graph: %s", currentNodeID)
		}

		state.CurrentNodeID = node.ID
		state.VisitCounts[node.ID]++

		// Loop guard safety check
		if state.VisitCounts[node.ID] > 100 {
			state.Status = StatusFailed
			_ = workflow.ExecuteActivity(WithActivitySummary(ctx, fmt.Sprintf("Emit workflow_failed lifecycle event for node %s", node.ID)), act.EmitLifecycleEvent, activities.LifecycleEvent{
				SchemaVersion: domain.DefaultSchemaVersion,
				RunID:         input.RunID,
				TenantID:      input.TenantID,
				WorkflowID:    input.WorkflowID,
				EventType:     "workflow_failed",
				NodeID:        node.ID,
				Status:        string(StatusFailed),
				Timestamp:     workflow.Now(ctx),
				Metadata:      map[string]interface{}{"error": "max node visit limit exceeded"},
			}).Get(ctx, nil)

			return &CompiledJourneyResult{
				RunID:             input.RunID,
				WorkflowID:        input.WorkflowID,
				Status:            StatusFailed,
				CurrentNodeID:     node.ID,
				VisitCounts:       state.VisitCounts,
				ExperimentContext: state.ExperimentContext,
				ErrorMessage:      fmt.Sprintf("execution loop threshold exceeded on node: %s", node.ID),
			}, fmt.Errorf("execution loop threshold exceeded on node: %s", node.ID)
		}

		// Emit node_entered lifecycle event
		_ = workflow.ExecuteActivity(WithActivitySummary(ctx, fmt.Sprintf("Emit node_entered lifecycle event for node %s", node.ID)), act.EmitLifecycleEvent, activities.LifecycleEvent{
			SchemaVersion: domain.DefaultSchemaVersion,
			RunID:         input.RunID,
			TenantID:      input.TenantID,
			WorkflowID:    input.WorkflowID,
			EventType:     "node_entered",
			NodeID:        node.ID,
			Status:        "running",
			Timestamp:     workflow.Now(ctx),
		}).Get(ctx, nil)

		nodeType := strings.ToLower(strings.TrimSpace(node.Type))
		var nextNodeID string

		switch {
		case nodeType == "event_start" || nodeType == "eventstart" || nodeType == "start" || nodeType == "trigger":
			outgoing := findOutgoingEdges(ir.Edges, node.ID)
			if len(outgoing) > 0 {
				nextNodeID = outgoing[0].TargetID
			}

		case nodeType == "condition" || nodeType == "branch" || nodeType == "if_else":
			// Condition handler
			outgoing := findOutgoingEdges(ir.Edges, node.ID)
			var chosenEdge *domain.IREdge

			evalCtx := domain.FlattenPayloadContext(input.InputPayload)
			for k, v := range state.NodeOutputs {
				evalCtx[k] = v
			}

			// Extract condition expression from node params/data
			nodeExpr := ""
			for _, k := range []string{"expression", "condition_expression", "condition", "name", "label"} {
				if v, ok := node.Params[k].(string); ok && strings.TrimSpace(v) != "" {
					strVal := strings.TrimSpace(v)
					if strVal != "Condition" && strVal != "condition" {
						nodeExpr = strVal
						break
					}
				}
			}

			if nodeExpr != "" {
				condInput := activities.EvaluateConditionInput{
					ConditionExpression: nodeExpr,
					Context:             evalCtx,
				}
				var condResult bool
				condAo := workflow.ActivityOptions{
					Summary:             fmt.Sprintf("Evaluate condition '%s' on node %s", nodeExpr, node.ID),
					StartToCloseTimeout: 10 * time.Second,
					RetryPolicy: &temporal.RetryPolicy{
						InitialInterval:    100 * time.Millisecond,
						BackoffCoefficient: 2.0,
						MaximumInterval:    1 * time.Second,
						MaximumAttempts:    3,
					},
				}
				condCtx := workflow.WithActivityOptions(ctx, condAo)
				err := workflow.ExecuteActivity(condCtx, act.EvaluateCondition, condInput).Get(condCtx, &condResult)
				if err == nil {
					targetBranch := "false"
					if condResult {
						targetBranch = "true"
					}
					for i := range outgoing {
						edge := outgoing[i]
						edgeIDLower := strings.ToLower(edge.ID)
						if strings.Contains(edgeIDLower, targetBranch) {
							chosenEdge = &edge
							break
						}
					}
					if chosenEdge == nil && condResult && len(outgoing) > 0 {
						chosenEdge = &outgoing[0]
					} else if chosenEdge == nil && !condResult && len(outgoing) > 1 {
						chosenEdge = &outgoing[1]
					} else if chosenEdge == nil && len(outgoing) > 0 {
						chosenEdge = &outgoing[0]
					}
				}
			}

			if chosenEdge != nil {
				state.NodeOutputs[node.ID] = chosenEdge.TargetID
				nextNodeID = chosenEdge.TargetID
			} else {
				state.NodeOutputs[node.ID] = "no_branch_matched"
				nextNodeID = ""
			}

		case nodeType == "wait_for_event" || nodeType == "waitforevent" || nodeType == "event_wait":
			cfg := ExtractWaitForEventConfig(node, input.TenantID, input.WorkflowID, input.RunID, state.Generation)
			res, err := ExecuteWaitForEventNode(ctx, act, cfg, state.BufferedEvents, state.SeenEventIDs, state.Variables)
			if err != nil {
				logger.Error("WaitForEvent execution failed", "nodeID", node.ID, "error", err)
				state.Status = StatusFailed
				_ = workflow.ExecuteActivity(WithActivitySummary(ctx, fmt.Sprintf("Emit workflow_failed lifecycle event for node %s", node.ID)), act.EmitLifecycleEvent, activities.LifecycleEvent{
					SchemaVersion: domain.DefaultSchemaVersion,
					RunID:         input.RunID,
					TenantID:      input.TenantID,
					WorkflowID:    input.WorkflowID,
					EventType:     "workflow_failed",
					NodeID:        node.ID,
					Status:        string(StatusFailed),
					Timestamp:     workflow.Now(ctx),
					Metadata:      map[string]interface{}{"error": err.Error()},
				}).Get(ctx, nil)
				return &CompiledJourneyResult{
					RunID:             input.RunID,
					WorkflowID:        input.WorkflowID,
					Status:            StatusFailed,
					CurrentNodeID:     node.ID,
					VisitCounts:       state.VisitCounts,
					ExperimentContext: state.ExperimentContext,
					ErrorMessage:      err.Error(),
				}, fmt.Errorf("WaitForEvent node failed: %w", err)
			}

			if res.Status == "cancelled" {
				state.Status = StatusCancelled
				_ = workflow.ExecuteActivity(WithActivitySummary(ctx, fmt.Sprintf("Emit workflow_cancelled lifecycle event for node %s", node.ID)), act.EmitLifecycleEvent, activities.LifecycleEvent{
					SchemaVersion: domain.DefaultSchemaVersion,
					RunID:         input.RunID,
					TenantID:      input.TenantID,
					WorkflowID:    input.WorkflowID,
					EventType:     "workflow_cancelled",
					Status:        string(StatusCancelled),
					Timestamp:     workflow.Now(ctx),
				}).Get(ctx, nil)
				return &CompiledJourneyResult{
					RunID:             input.RunID,
					WorkflowID:        input.WorkflowID,
					Status:            StatusCancelled,
					CurrentNodeID:     node.ID,
					VisitCounts:       state.VisitCounts,
					NodeOutputs:       state.NodeOutputs,
					ExperimentContext: state.ExperimentContext,
				}, nil
			}

			if res.Status == "continue_as_new" || (node.Params != nil && node.Params["continue_as_new"] == true) {
				state.Generation++
				newInput := input
				newInput.Generation = state.Generation
				newInput.InitialState = state
				return nil, workflow.NewContinueAsNewError(ctx, CompiledJourneyWorkflow, newInput)
			}

			state.NodeOutputs[node.ID] = map[string]interface{}{
				"status":      res.Status,
				"event_id":    res.EventID,
				"event_type":  res.EventType,
				"payload_ref": res.PayloadRef,
			}

			if res.Payload != nil {
				for k, v := range res.Payload {
					state.Variables[k] = v
				}
			}

			outgoing := findOutgoingEdges(ir.Edges, node.ID)
			var chosenEdge *domain.IREdge
			for i := range outgoing {
				edge := outgoing[i]
				expr := strings.TrimSpace(edge.ConditionExpression)
				if (res.Status == "matched" && (expr == "" || expr == "true" || expr == "matched" || strings.Contains(expr, "matched"))) ||
					(res.Status == "timed_out" && (expr == "timeout" || expr == "timed_out" || strings.Contains(expr, "timeout"))) {
					chosenEdge = &edge
					break
				}
			}

			if chosenEdge != nil {
				nextNodeID = chosenEdge.TargetID
			} else if len(outgoing) > 0 {
				nextNodeID = outgoing[0].TargetID
			}

		case nodeType == "delay" || nodeType == "timer" || nodeType == "wait":
			// Delay handler (via workflow.Sleep / Timer)
			delaySec := extractDelaySeconds(node.Params)
			if delaySec > 0 {
				timer := workflow.NewTimer(ctx, time.Duration(delaySec)*time.Second)
				selector := workflow.NewSelector(ctx)
				cancelledDuringDelay := false

				selector.AddFuture(timer, func(f workflow.Future) {})
				selector.AddReceive(cancelSignalChan, func(c workflow.ReceiveChannel, more bool) {
					var sig domain.WorkflowSignal
					c.Receive(ctx, &sig)
					cancelledDuringDelay = true
				})
				selector.Select(ctx)

				if cancelledDuringDelay {
					state.Status = StatusCancelled
					_ = workflow.ExecuteActivity(WithActivitySummary(ctx, fmt.Sprintf("Emit workflow_cancelled lifecycle event for node %s", node.ID)), act.EmitLifecycleEvent, activities.LifecycleEvent{
						SchemaVersion: domain.DefaultSchemaVersion,
						RunID:         input.RunID,
						TenantID:      input.TenantID,
						WorkflowID:    input.WorkflowID,
						EventType:     "workflow_cancelled",
						Status:        string(StatusCancelled),
						Timestamp:     workflow.Now(ctx),
					}).Get(ctx, nil)
					return &CompiledJourneyResult{
						RunID:             input.RunID,
						WorkflowID:        input.WorkflowID,
						Status:            StatusCancelled,
						CurrentNodeID:     node.ID,
						VisitCounts:       state.VisitCounts,
						NodeOutputs:       state.NodeOutputs,
						ExperimentContext: state.ExperimentContext,
					}, nil
				}
			}
			state.NodeOutputs[node.ID] = fmt.Sprintf("delayed_%ds", delaySec)
			outgoing := findOutgoingEdges(ir.Edges, node.ID)
			if len(outgoing) > 0 {
				nextNodeID = outgoing[0].TargetID
			}

		case nodeType == "exit" || nodeType == "end" || nodeType == "terminate" || nodeType == "stop":
			// Exit handler
			termStatus := extractExitStatus(node.Params)
			state.Status = termStatus
			state.NodeOutputs[node.ID] = string(termStatus)

			_ = workflow.ExecuteActivity(WithActivitySummary(ctx, fmt.Sprintf("Emit workflow_%s lifecycle event for node %s", string(termStatus), node.ID)), act.EmitLifecycleEvent, activities.LifecycleEvent{
				SchemaVersion: domain.DefaultSchemaVersion,
				RunID:         input.RunID,
				TenantID:      input.TenantID,
				WorkflowID:    input.WorkflowID,
				EventType:     "workflow_" + string(termStatus),
				NodeID:        node.ID,
				Status:        string(termStatus),
				Timestamp:     workflow.Now(ctx),
			}).Get(ctx, nil)
			return &CompiledJourneyResult{
				RunID:             input.RunID,
				WorkflowID:        input.WorkflowID,
				Status:            termStatus,
				CurrentNodeID:     node.ID,
				VisitCounts:       state.VisitCounts,
				NodeOutputs:       state.NodeOutputs,
				ExperimentContext: state.ExperimentContext,
			}, nil

		case nodeType == "experiment" || nodeType == "ab_test" || nodeType == "multivariate":
			targetID, err := ExecuteExperimentNode(ctx, act, node, &ir, state, input, cancelSignalChan)
			if err != nil {
				if err == ErrWorkflowCancelled {
					state.Status = StatusCancelled
					_ = workflow.ExecuteActivity(WithActivitySummary(ctx, fmt.Sprintf("Emit workflow_cancelled lifecycle event for node %s", node.ID)), act.EmitLifecycleEvent, activities.LifecycleEvent{
						SchemaVersion: domain.DefaultSchemaVersion,
						RunID:         input.RunID,
						TenantID:      input.TenantID,
						WorkflowID:    input.WorkflowID,
						EventType:     "workflow_cancelled",
						Status:        string(StatusCancelled),
						Timestamp:     workflow.Now(ctx),
					}).Get(ctx, nil)
					return &CompiledJourneyResult{
						RunID:             input.RunID,
						WorkflowID:        input.WorkflowID,
						Status:            StatusCancelled,
						CurrentNodeID:     node.ID,
						VisitCounts:       state.VisitCounts,
						NodeOutputs:       state.NodeOutputs,
						ExperimentContext: state.ExperimentContext,
					}, nil
				}
				logger.Error("Experiment node execution failed", "nodeID", node.ID, "error", err)
				state.Status = StatusFailed
				_ = workflow.ExecuteActivity(WithActivitySummary(ctx, fmt.Sprintf("Emit workflow_failed lifecycle event for node %s", node.ID)), act.EmitLifecycleEvent, activities.LifecycleEvent{
					SchemaVersion: domain.DefaultSchemaVersion,
					RunID:         input.RunID,
					TenantID:      input.TenantID,
					WorkflowID:    input.WorkflowID,
					EventType:     "workflow_failed",
					NodeID:        node.ID,
					Status:        string(StatusFailed),
					Timestamp:     workflow.Now(ctx),
					Metadata:      map[string]interface{}{"error": err.Error()},
				}).Get(ctx, nil)
				return &CompiledJourneyResult{
					RunID:             input.RunID,
					WorkflowID:        input.WorkflowID,
					Status:            StatusFailed,
					CurrentNodeID:     node.ID,
					VisitCounts:       state.VisitCounts,
					ExperimentContext: state.ExperimentContext,
					ErrorMessage:      err.Error(),
				}, fmt.Errorf("Experiment node failed: %w", err)
			}
			nextNodeID = targetID

		case nodeType == "email" || nodeType == "sms" || nodeType == "push" || nodeType == "inapp" || nodeType == "in_app" || nodeType == "webhook" || nodeType == "action" || nodeType == "channel_action":
			targetID, err := ExecuteChannelActionNode(ctx, act, node, &ir, state, input)
			if err != nil {
				logger.Error("Channel action node execution failed", "nodeID", node.ID, "error", err)
				state.Status = StatusFailed
				_ = workflow.ExecuteActivity(WithActivitySummary(ctx, fmt.Sprintf("Emit workflow_failed lifecycle event for node %s", node.ID)), act.EmitLifecycleEvent, activities.LifecycleEvent{
					SchemaVersion: domain.DefaultSchemaVersion,
					RunID:         input.RunID,
					TenantID:      input.TenantID,
					WorkflowID:    input.WorkflowID,
					EventType:     "workflow_failed",
					NodeID:        node.ID,
					Status:        string(StatusFailed),
					Timestamp:     workflow.Now(ctx),
					Metadata:      map[string]interface{}{"error": err.Error()},
				}).Get(ctx, nil)
				return &CompiledJourneyResult{
					RunID:             input.RunID,
					WorkflowID:        input.WorkflowID,
					Status:            StatusFailed,
					CurrentNodeID:     node.ID,
					VisitCounts:       state.VisitCounts,
					ExperimentContext: state.ExperimentContext,
					ErrorMessage:      err.Error(),
				}, fmt.Errorf("Channel action node failed: %w", err)
			}
			nextNodeID = targetID

		default:
			// Generic activity node
			execInput := activities.ExecuteNodeInput{
				RunID:  input.RunID,
				Node:   node,
				Params: input.InputPayload,
			}
			var actResult domain.ActionResult
			err := workflow.ExecuteActivity(WithActivitySummary(ctx, fmt.Sprintf("Execute activity node %s (%s)", node.ID, node.ActivityName)), act.ExecuteNode, execInput).Get(ctx, &actResult)
			if err != nil {
				logger.Error("Node activity execution failed", "nodeID", node.ID, "error", err)
				state.Status = StatusFailed
				_ = workflow.ExecuteActivity(WithActivitySummary(ctx, fmt.Sprintf("Emit workflow_failed lifecycle event for node %s", node.ID)), act.EmitLifecycleEvent, activities.LifecycleEvent{
					SchemaVersion: domain.DefaultSchemaVersion,
					RunID:         input.RunID,
					TenantID:      input.TenantID,
					WorkflowID:    input.WorkflowID,
					EventType:     "workflow_failed",
					NodeID:        node.ID,
					Status:        string(StatusFailed),
					Timestamp:     workflow.Now(ctx),
					Metadata:      map[string]interface{}{"error": err.Error()},
				}).Get(ctx, nil)
				return &CompiledJourneyResult{
					RunID:             input.RunID,
					WorkflowID:        input.WorkflowID,
					Status:            StatusFailed,
					CurrentNodeID:     node.ID,
					VisitCounts:       state.VisitCounts,
					ExperimentContext: state.ExperimentContext,
					ErrorMessage:      err.Error(),
				}, fmt.Errorf("Node activity execution failed: %w", err)
			}
			state.NodeOutputs[node.ID] = "completed"

			outgoing := findOutgoingEdges(ir.Edges, node.ID)
			if len(outgoing) > 0 {
				nextNodeID = outgoing[0].TargetID
			}
		}

		currentNodeID = nextNodeID
	}

	// Default completion if graph ends without explicit exit node
	_ = workflow.ExecuteActivity(WithActivitySummary(ctx, "Emit workflow_completed lifecycle event"), act.EmitLifecycleEvent, activities.LifecycleEvent{
		SchemaVersion: domain.DefaultSchemaVersion,
		RunID:         input.RunID,
		TenantID:      input.TenantID,
		WorkflowID:    input.WorkflowID,
		EventType:     "workflow_completed",
		Status:        string(state.Status),
		Timestamp:     workflow.Now(ctx),
	}).Get(ctx, nil)
	return &CompiledJourneyResult{
		RunID:             input.RunID,
		WorkflowID:        input.WorkflowID,
		Status:            state.Status,
		CurrentNodeID:     state.CurrentNodeID,
		VisitCounts:       state.VisitCounts,
		NodeOutputs:       state.NodeOutputs,
		ExperimentContext: state.ExperimentContext,
	}, nil
}

// Helpers

func findOutgoingEdges(edges []domain.IREdge, sourceID string) []domain.IREdge {
	var res []domain.IREdge
	for _, e := range edges {
		if e.SourceID == sourceID {
			res = append(res, e)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		if res[i].ID != res[j].ID {
			return res[i].ID < res[j].ID
		}
		return res[i].TargetID < res[j].TargetID
	})
	return res
}

func extractDelaySeconds(params map[string]interface{}) int64 {
	if params == nil {
		return 0
	}
	keys := []string{"duration_seconds", "delay_seconds", "duration", "seconds"}
	for _, k := range keys {
		if val, ok := params[k]; ok {
			return toInt64(val)
		}
	}
	return 0
}

func extractExitStatus(params map[string]interface{}) TerminalStatus {
	if params == nil {
		return StatusSucceeded
	}
	keys := []string{"status", "terminal_status", "exit_status"}
	for _, k := range keys {
		if val, ok := params[k]; ok {
			str := strings.ToLower(fmt.Sprintf("%v", val))
			switch str {
			case "suppressed":
				return StatusSuppressed
			case "failed":
				return StatusFailed
			case "cancelled":
				return StatusCancelled
			case "invalid_artifact":
				return StatusInvalidArtifact
			case "succeeded", "success":
				return StatusSucceeded
			}
		}
	}
	return StatusSucceeded
}

func toInt64(val interface{}) int64 {
	switch v := val.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case string:
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return 0
}
