package middleware

import (
	"time"

	"github.com/go-chi/chi/v5"
)

// RouterConfig holds configuration options for initializing the Chi router.
type RouterConfig struct {
	Logger            any
	ReadinessRegistry *ReadinessRegistry
	IdempotencyStore  IdempotencyStore
	MaxBodyBytes      int64
	RequestTimeout    time.Duration
}

// NewRouter constructs a Chi router with standard middleware and health endpoints.
func NewRouter(cfg RouterConfig) chi.Router {
	r := chi.NewRouter()

	logger := cfg.Logger

	maxBodyBytes := cfg.MaxBodyBytes
	if maxBodyBytes <= 0 {
		maxBodyBytes = 1024 * 1024 // 1 MB default
	}

	timeout := cfg.RequestTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// Standard middleware pipeline
	r.Use(RequestID)
	r.Use(RequestLogger(logger))
	r.Use(Recoverer(logger))
	r.Use(Timeout(timeout))
	r.Use(BodyLimit(maxBodyBytes))
	r.Use(Identity)

	if cfg.IdempotencyStore != nil {
		r.Use(Idempotency(cfg.IdempotencyStore))
	}

	// Liveness and Readiness endpoints
	r.Get("/healthz", Healthz())
	r.Get("/readyz", Readyz(cfg.ReadinessRegistry))

	return r
}
