package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	"github.com/validated-pattern/journey-platform/internal/api/handlers"
	"github.com/validated-pattern/journey-platform/internal/compiler"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
	"go.temporal.io/sdk/client"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("EVENT_INGRESS_PORT")
	}
	if port == "" {
		port = "8084"
	}

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

	temporalHost := os.Getenv("TEMPORAL_HOST_PORT")
	if temporalHost == "" {
		temporalHost = "127.0.0.1:7233"
	}
	tc, _ := client.Dial(client.Options{
		HostPort:  temporalHost,
		Namespace: "default",
	})
	if tc != nil {
		defer tc.Close()
	}

	h := handlers.New(storeRepo, compiler.New(), nil)
	if tc != nil {
		h.SetTemporalClient(tc)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","service":"event-ingress"}`))
	})

	r.Post("/api/v1/events/emit", h.EmitKafkaTestEvent)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	log.Printf("Starting event-ingress service on port %s...", port)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Event-Ingress server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down event-ingress gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
