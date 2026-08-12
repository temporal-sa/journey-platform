package workflows

import (
	"fmt"
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

