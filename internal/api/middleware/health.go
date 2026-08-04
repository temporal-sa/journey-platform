package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// CheckFunc is a function type for performing readiness checks on dependencies.
type CheckFunc func(ctx context.Context) error

// ReadinessRegistry manages dependency readiness checks.
type ReadinessRegistry struct {
	mu     sync.RWMutex
	checks map[string]CheckFunc
}

// NewReadinessRegistry initializes a new ReadinessRegistry.
func NewReadinessRegistry() *ReadinessRegistry {
	return &ReadinessRegistry{
		checks: make(map[string]CheckFunc),
	}
}

// Register registers a dependency health check with a given name.
func (r *ReadinessRegistry) Register(name string, check CheckFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checks[name] = check
}

// HealthResponse represents the response structure for liveness and readiness endpoints.
type HealthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

// Healthz returns the HTTP handler for liveness checks.
func Healthz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
	}
}

// Readyz returns the HTTP handler for dependency-aware readiness checks.
func Readyz(registry *ReadinessRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		results := make(map[string]string)
		allOK := true

		if registry != nil {
			registry.mu.RLock()
			checks := make(map[string]CheckFunc, len(registry.checks))
			for name, fn := range registry.checks {
				checks[name] = fn
			}
			registry.mu.RUnlock()

			for name, check := range checks {
				if err := check(ctx); err != nil {
					results[name] = "error: " + err.Error()
					allOK = false
				} else {
					results[name] = "ok"
				}
			}
		}

		status := "ok"
		statusCode := http.StatusOK
		if !allOK {
			status = "degraded"
			statusCode = http.StatusServiceUnavailable
		}

		resp := HealthResponse{
			Status: status,
			Checks: results,
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(resp)
	}
}
