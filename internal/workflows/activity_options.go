package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// WithActivitySummary returns a workflow Context configured with ActivityOptions including a human-readable Summary.
func WithActivitySummary(ctx workflow.Context, summary string) workflow.Context {
	ao := workflow.ActivityOptions{
		Summary:             summary,
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    100 * time.Millisecond,
			BackoffCoefficient: 2.0,
			MaximumInterval:    1 * time.Second,
			MaximumAttempts:    3,
		},
	}
	return workflow.WithActivityOptions(ctx, ao)
}
