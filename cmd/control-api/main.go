package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	"github.com/validated-pattern/journey-platform/internal/api/handlers"
	appMiddleware "github.com/validated-pattern/journey-platform/internal/api/middleware"
	"github.com/validated-pattern/journey-platform/internal/compiler"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
	"github.com/validated-pattern/journey-platform/internal/telemetry/logging"
	"github.com/validated-pattern/journey-platform/internal/telemetry/tracing"
	"go.temporal.io/sdk/client"
)

func main() {
	logging.Init(false)
	ctx := context.Background()
	shutdownTracer, _ := tracing.InitTracerProvider(ctx, "control-api")
	if shutdownTracer != nil {
		defer func() {
			sCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = shutdownTracer(sCtx)
		}()
	}
	port := os.Getenv("CONTROL_API_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8087"
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
		tracedDB := postgres.NewTracedDB(db)
		storeRepo = postgres.NewPostgresRepository(tracedDB)
	} else {
		logging.Debug().Err(err).Msg("Postgres repository connection error, falling back to in-memory repository")
		storeRepo = postgres.NewMemoryRepository()
	}

	temporalHost := os.Getenv("TEMPORAL_HOST_PORT")
	if temporalHost == "" {
		temporalHost = "127.0.0.1:7233"
	}
	tc, err := client.Dial(client.Options{
		HostPort:  temporalHost,
		Namespace: "default",
		Logger:    logging.NewTemporalLogger(),
	})
	if err != nil {
		logging.Debug().Err(err).Msg("Temporal client connection notice")
	} else {
		defer tc.Close()
	}

	h := handlers.New(storeRepo, compiler.New(), nil)
	if tc != nil {
		h.SetTemporalClient(tc)
	}

	r := chi.NewRouter()
	r.Use(appMiddleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(appMiddleware.RequestLogger(nil))
	r.Use(middleware.Recoverer)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","service":"control-api"}`))
	})

	h.RegisterRoutes(r)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	logging.Info().Str("service", "control-api").Str("port", port).Msg(fmt.Sprintf("Starting control-api service on port %s...", port))

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Fatal().Err(err).Msg("Control-API server failed")
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logging.Info().Str("service", "control-api").Msg("Shutting down control-api gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
