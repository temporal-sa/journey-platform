package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/validated-pattern/journey-platform/internal/activities"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
	"github.com/validated-pattern/journey-platform/internal/telemetry/logging"
	"github.com/validated-pattern/journey-platform/internal/workflows"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"
)

func main() {
	logging.Init(false)

	temporalHost := os.Getenv("TEMPORAL_HOST_PORT")
	if temporalHost == "" {
		temporalHost = "127.0.0.1:7233"
	}
	namespace := os.Getenv("TEMPORAL_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}
	tracingInterceptor, _ := workflows.CreateTemporalTracingInterceptor()
	var clientInterceptors []interceptor.ClientInterceptor
	if tracingInterceptor != nil {
		clientInterceptors = append(clientInterceptors, tracingInterceptor)
	}

	c, err := client.Dial(client.Options{
		HostPort:     temporalHost,
		Namespace:    namespace,
		Interceptors: clientInterceptors,
	})
	if err != nil {
		logging.Fatal().Err(err).Msg("Failed to create Temporal client")
	}
	defer c.Close()

	taskQueue := "journey-engine-task-queue"
	wOpts := worker.Options{}
	if tracingInterceptor != nil {
		wOpts.Interceptors = []interceptor.WorkerInterceptor{tracingInterceptor}
	}
	w := worker.New(c, taskQueue, wOpts)
	w.RegisterWorkflow(workflows.CompiledJourneyWorkflow)

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		host := os.Getenv("POSTGRES_HOST")
		if host == "" {
			host = "127.0.0.1"
		}
		pgPort := os.Getenv("POSTGRES_PORT")
		if pgPort == "" {
			pgPort = "5432"
		}
		user := os.Getenv("POSTGRES_USER")
		if user == "" {
			user = "journey"
		}
		pass := os.Getenv("POSTGRES_PASSWORD")
		if pass == "" {
			pass = "journey_pass"
		}
		dbName := os.Getenv("POSTGRES_DB")
		if dbName == "" {
			dbName = "journeydb"
		}
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, pgPort, dbName)
	}

	var storeRepo postgres.Repository
	db, err := sql.Open("postgres", dsn)
	if err == nil && db != nil {
		tracedDB := postgres.NewTracedDB(db)
		storeRepo = postgres.NewPostgresRepository(tracedDB)
	} else {
		logging.Debug().Err(err).Msg("Postgres repository connection error, falling back to in-memory repository")
		storeRepo = postgres.NewMemoryRepository()
	}

	act := activities.NewActivities()
	act.SetRepository(storeRepo)

	w.RegisterActivity(act.LoadCompiledIR)
	w.RegisterActivity(act.EvaluateCondition)
	w.RegisterActivity(act.ExecuteActionGateway)
	w.RegisterActivity(act.EmitLifecycleEvent)
	w.RegisterActivity(act.CreateSubscription)
	w.RegisterActivity(act.CloseSubscription)
	w.RegisterActivity(act.GetOrAssign)
	w.RegisterActivity(act.RecordExposure)
	w.RegisterActivity(act.ExecuteNode)

	logging.Info().Str("service", "journey-worker").Str("temporal_host", temporalHost).Str("task_queue", taskQueue).Msg(fmt.Sprintf("Starting journey-worker connecting to Temporal at %s (TaskQueue: %s)...", temporalHost, taskQueue))

	err = w.Start()
	if err != nil {
		logging.Fatal().Err(err).Msg("Failed to start Temporal worker")
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logging.Info().Str("service", "journey-worker").Msg("Stopping journey-worker...")
	w.Stop()
}
