package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/validated-pattern/journey-platform/internal/activities"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
	"github.com/validated-pattern/journey-platform/internal/workflows"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	temporalHost := os.Getenv("TEMPORAL_HOST_PORT")
	if temporalHost == "" {
		temporalHost = "127.0.0.1:7233"
	}
	namespace := os.Getenv("TEMPORAL_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	c, err := client.Dial(client.Options{
		HostPort:  temporalHost,
		Namespace: namespace,
	})
	if err != nil {
		log.Fatalf("Failed to create Temporal client: %v", err)
	}
	defer c.Close()

	taskQueue := "journey-engine-task-queue"
	w := worker.New(c, taskQueue, worker.Options{})

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
		storeRepo = postgres.NewPostgresRepository(db)
	} else {
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

	log.Printf("Starting journey-worker connecting to Temporal at %s (TaskQueue: %s)...", temporalHost, taskQueue)

	err = w.Start()
	if err != nil {
		log.Fatalf("Failed to start Temporal worker: %v", err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Stopping journey-worker...")
	w.Stop()
}
