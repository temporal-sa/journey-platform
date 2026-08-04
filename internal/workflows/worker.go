package workflows

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/validated-pattern/journey-platform/internal/activities"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// WorkerConfig holds configuration for the Temporal journey worker.
type WorkerConfig struct {
	HostPort                string
	Namespace               string
	TaskQueue               string
	Identity                string
	MaxConcurrentActivities int
	MaxConcurrentWorkflows  int
	ShutdownTimeout         time.Duration
}

// DefaultWorkerConfig returns sensible defaults for WorkerConfig.
func DefaultWorkerConfig() WorkerConfig {
	hostPort := os.Getenv("TEMPORAL_HOST_PORT")
	if hostPort == "" {
		hostPort = "localhost:7233"
	}
	namespace := os.Getenv("TEMPORAL_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "local"
	}
	identity := fmt.Sprintf("journey-worker@%s-pid%d", hostname, os.Getpid())

	return WorkerConfig{
		HostPort:                hostPort,
		Namespace:               namespace,
		TaskQueue:               JOURNEY_TASK_QUEUE,
		Identity:                identity,
		MaxConcurrentActivities: 100,
		MaxConcurrentWorkflows:  100,
		ShutdownTimeout:         30 * time.Second,
	}
}

// BootstrapWorker initializes the Temporal Client with data converter isolation and registers all workflows and activities.
func BootstrapWorker(cfg WorkerConfig) (worker.Worker, client.Client, error) {
	isoConverter := NewIsolatedDataConverter(nil)

	clientOpts := client.Options{
		HostPort:      cfg.HostPort,
		Namespace:     cfg.Namespace,
		DataConverter: isoConverter,
		Identity:      cfg.Identity,
	}

	c, err := client.Dial(clientOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Temporal client: %w", err)
	}

	workerOpts := worker.Options{
		Identity:                              cfg.Identity,
		MaxConcurrentActivityExecutionSize:     cfg.MaxConcurrentActivities,
		MaxConcurrentWorkflowTaskExecutionSize: cfg.MaxConcurrentWorkflows,
	}

	w := worker.New(c, cfg.TaskQueue, workerOpts)

	// Register Workflow
	w.RegisterWorkflow(JourneyWorkflow)
	w.RegisterWorkflow(CompiledJourneyWorkflow)

	// Register Activities
	act := activities.NewActivities()
	RegisterAllActivities(w, act)
	return w, c, nil
}

// ActivityRegistrar abstracts worker.Worker and testsuite.TestWorkflowEnvironment.
type ActivityRegistrar interface {
	RegisterActivity(a interface{})
}

// RegisterAllActivities registers all engine activity definitions on a Worker or TestWorkflowEnvironment.
func RegisterAllActivities(r ActivityRegistrar, act *activities.Activities) {
	if r == nil || act == nil {
		return
	}
	r.RegisterActivity(act.ExecuteNode)
	r.RegisterActivity(act.EvaluateCondition)
	r.RegisterActivity(act.EmitOutcome)
	r.RegisterActivity(act.ExposeAssignment)
	r.RegisterActivity(act.LoadCompiledIR)
	r.RegisterActivity(act.EmitLifecycleEvent)
	r.RegisterActivity(act.ExecuteActionGateway)
	r.RegisterActivity(act.GetOrAssign)
	r.RegisterActivity(act.RecordExposure)
	r.RegisterActivity(act.CreateSubscription)
	r.RegisterActivity(act.CloseSubscription)
	r.RegisterActivity(act.EvaluatePolicy)
	r.RegisterActivity(act.ResolveAttributes)
	r.RegisterActivity(act.ResolveParameters)
}

// RunWorkerWithGracefulDrain starts the worker and handles OS signals for graceful shutdown/drain.
func RunWorkerWithGracefulDrain(ctx context.Context, w worker.Worker) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Run(worker.InterruptCh())
	}()

	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		return err
	case <-sigCtx.Done():
		w.Stop()
		return nil
	}
}
