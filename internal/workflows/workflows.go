package workflows

import (
	"fmt"
	"time"

	"github.com/validated-pattern/journey-platform/internal/activities"
	"github.com/validated-pattern/journey-platform/internal/domain"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	JOURNEY_TASK_QUEUE = "journey-engine-task-queue"
	TaskQueue          = JOURNEY_TASK_QUEUE

	// Signal Names
	SignalNameEvent  = "journey.signal.event"
	SignalNamePause  = "journey.signal.pause"
	SignalNameResume = "journey.signal.resume"
	SignalNameCancel = "journey.signal.cancel"

	// Query Names
	QueryTypeStatus    = "journey.query.status"
	QueryTypeVariables = "journey.query.variables"

	// Update Names
	UpdateTypeState  = "journey.update.state"
	UpdateTypeConfig = "journey.update.config"
)

// Engine executes workflow definitions.
type Engine struct{}

// New creates a new Engine.
func New() *Engine {
	return &Engine{}
}

// Execute starts execution of a workflow.
func (e *Engine) Execute(workflowID string) error {
	if workflowID == "" {
		return fmt.Errorf("workflowID cannot be empty")
	}
	return nil
}

// JourneyWorkflow is the main Temporal workflow executing domain journeys deterministically.
func JourneyWorkflow(ctx workflow.Context, input domain.WorkflowInput) (*domain.RunProjection, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting JourneyWorkflow", "workflowID", input.WorkflowID, "runID", input.RunID)

	if input.WorkflowID == "" {
		return nil, fmt.Errorf("invalid input: WorkflowID is required")
	}

	startTime := workflow.Now(ctx)

	// Initialize RunProjection state
	projection := &domain.RunProjection{
		SchemaVersion: domain.DefaultSchemaVersion,
		RunID:         input.RunID,
		TenantID:      input.TenantID,
		WorkflowID:    input.WorkflowID,
		Status:        domain.RunStatusRunning,
		CurrentNodes:  []string{"start"},
		Variables:     make(map[string]interface{}),
		StartedAt:     startTime,
		UpdatedAt:     startTime,
	}

	// Copy input payload variables to projection variables
	if input.InputPayload != nil {
		for k, v := range input.InputPayload {
			projection.Variables[k] = v
		}
	}

	isPaused := false

	// Register Query Handler: Status
	err := workflow.SetQueryHandler(ctx, QueryTypeStatus, func() (*domain.RunProjection, error) {
		pCopy := *projection
		pCopy.UpdatedAt = workflow.Now(ctx)
		return &pCopy, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set status query handler: %w", err)
	}

	// Register Query Handler: Variables
	err = workflow.SetQueryHandler(ctx, QueryTypeVariables, func() (map[string]interface{}, error) {
		varsCopy := make(map[string]interface{})
		for k, v := range projection.Variables {
			varsCopy[k] = v
		}
		return varsCopy, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set variables query handler: %w", err)
	}

	// Register Update Handler: State
	err = workflow.SetUpdateHandler(ctx, UpdateTypeState, func(ctx workflow.Context, updatedProj domain.RunProjection) (domain.RunProjection, error) {
		if updatedProj.Status != "" {
			projection.Status = updatedProj.Status
		}
		if len(updatedProj.CurrentNodes) > 0 {
			projection.CurrentNodes = updatedProj.CurrentNodes
		}
		projection.UpdatedAt = workflow.Now(ctx)
		return *projection, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set state update handler: %w", err)
	}

	// Register Update Handler: Config
	err = workflow.SetUpdateHandler(ctx, UpdateTypeConfig, func(ctx workflow.Context, config map[string]interface{}) (map[string]interface{}, error) {
		for k, v := range config {
			projection.Variables[k] = v
		}
		projection.UpdatedAt = workflow.Now(ctx)
		return projection.Variables, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set config update handler: %w", err)
	}

	// Setup Activity Options
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

	// Signal channels
	eventSignalChan := workflow.GetSignalChannel(ctx, SignalNameEvent)
	pauseSignalChan := workflow.GetSignalChannel(ctx, SignalNamePause)
	resumeSignalChan := workflow.GetSignalChannel(ctx, SignalNameResume)
	cancelSignalChan := workflow.GetSignalChannel(ctx, SignalNameCancel)

	// Execute start activity node
	act := &activities.Activities{}
	var actResult domain.ActionResult
	execInput := activities.ExecuteNodeInput{
		RunID: input.RunID,
		Node: domain.IRNode{
			ID:           "start-node",
			Type:         "activity",
			ActivityName: "StartJourney",
			Params:       input.InputPayload,
		},
	}

	err = workflow.ExecuteActivity(WithActivitySummary(ctx, "Execute start activity node StartJourney"), act.ExecuteNode, execInput).Get(ctx, &actResult)
	if err != nil {
		logger.Error("Start activity failed", "error", err)
		projection.Status = domain.RunStatusFailed
		now := workflow.Now(ctx)
		projection.CompletedAt = &now
		return projection, err
	}

	// Main workflow execution event/signal loop with selector
	selector := workflow.NewSelector(ctx)

	isCancelled := false

	selector.AddReceive(eventSignalChan, func(c workflow.ReceiveChannel, more bool) {
		var sig domain.WorkflowSignal
		c.Receive(ctx, &sig)
		logger.Info("Received event signal", "signalName", sig.SignalName)
		if sig.Payload != nil {
			for k, v := range sig.Payload {
				projection.Variables[k] = v
			}
		}
		projection.UpdatedAt = workflow.Now(ctx)
	})

	selector.AddReceive(pauseSignalChan, func(c workflow.ReceiveChannel, more bool) {
		var sig domain.WorkflowSignal
		c.Receive(ctx, &sig)
		logger.Info("Received pause signal")
		isPaused = true
		projection.Status = domain.RunStatusRunning
		projection.UpdatedAt = workflow.Now(ctx)
	})

	selector.AddReceive(resumeSignalChan, func(c workflow.ReceiveChannel, more bool) {
		var sig domain.WorkflowSignal
		c.Receive(ctx, &sig)
		logger.Info("Received resume signal")
		isPaused = false
		projection.Status = domain.RunStatusRunning
		projection.UpdatedAt = workflow.Now(ctx)
	})

	selector.AddReceive(cancelSignalChan, func(c workflow.ReceiveChannel, more bool) {
		var sig domain.WorkflowSignal
		c.Receive(ctx, &sig)
		logger.Info("Received cancel signal")
		isCancelled = true
		projection.Status = domain.RunStatusTerminated
		projection.UpdatedAt = workflow.Now(ctx)
	})

	// Process any buffered signals or state changes
	for selector.HasPending() {
		selector.Select(ctx)
	}

	if isCancelled {
		now := workflow.Now(ctx)
		projection.CompletedAt = &now
		return projection, nil
	}

	if !isPaused {
		// Evaluate step condition via activity
		var conditionMet bool
		condInput := activities.EvaluateConditionInput{
			ConditionExpression: "true",
			Context:             projection.Variables,
		}
		err = workflow.ExecuteActivity(WithActivitySummary(ctx, "Evaluate step condition"), act.EvaluateCondition, condInput).Get(ctx, &conditionMet)
		if err != nil {
			logger.Error("Condition evaluation activity failed", "error", err)
		}

		if conditionMet {
			projection.CurrentNodes = []string{"end-node"}
		}
	}

	endTime := workflow.Now(ctx)
	projection.Status = domain.RunStatusCompleted
	projection.UpdatedAt = endTime
	projection.CompletedAt = &endTime

	logger.Info("JourneyWorkflow completed successfully", "runID", input.RunID)
	return projection, nil
}
