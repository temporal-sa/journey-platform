package workflows

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/validated-pattern/journey-platform/internal/activities"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
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
func BootstrapWorker(cfg WorkerConfig, repos ...postgres.Repository) (worker.Worker, client.Client, error) {
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
	w.RegisterWorkflow(CompiledJourneyWorkflow)

	// Register Activities
	act := activities.NewActivities()
	if len(repos) > 0 && repos[0] != nil {
		act.SetRepository(repos[0])
	}

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
}

// RunWorkerWithGracefulDrain blocks until an OS signal is received, then initiates graceful worker shutdown.
func RunWorkerWithGracefulDrain(ctx context.Context, w worker.Worker) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Run(worker.InterruptCh())
	}()

	select {
	case sig := <-sigCh:
		fmt.Printf("\n[Worker] Received signal %v, starting graceful drain...\n", sig)
		w.Stop()
		return nil
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("worker execution failed: %w", err)
		}
		return nil
	case <-ctx.Done():
		fmt.Println("\n[Worker] Context canceled, stopping worker...")
		w.Stop()
		return ctx.Err()
	}
}
