package middleware_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/validated-pattern/journey-platform/internal/api/middleware"
)

type samplePayload struct {
	Name string `json:"name"`
}

func TestTableMiddlewareScenarios(t *testing.T) {
	tests := []struct {
		name           string
		setupRouter    func() http.Handler
		method         string
		path           string
		headers        map[string]string
		body           string
		expectedStatus int
		verifyResponse func(t *testing.T, res *http.Response, bodyStr string)
	}{
		{
			name: "malformed JSON payload returns 400 Bad Request",
			setupRouter: func() http.Handler {
				r := chi.NewRouter()
				r.Use(middleware.RequestID)
				r.Post("/test", func(w http.ResponseWriter, r *http.Request) {
					var p samplePayload
					if err := middleware.DecodeJSON(w, r, &p); err != nil {
						return
					}
					middleware.WriteJSON(w, http.StatusOK, p)
				})
				return r
			},
			method:         http.MethodPost,
			path:           "/test",
			body:           `{"name": invalid-json}`,
			expectedStatus: http.StatusBadRequest,
			verifyResponse: func(t *testing.T, res *http.Response, bodyStr string) {
				var errResp middleware.ErrorResponse
				err := json.Unmarshal([]byte(bodyStr), &errResp)
				require.NoError(t, err)
				assert.Equal(t, "Bad Request", errResp.Error.Code)
				assert.Equal(t, "malformed JSON payload", errResp.Error.Message)
			},
		},
		{
			name: "oversized request body returns 413 Payload Too Large",
			setupRouter: func() http.Handler {
				r := chi.NewRouter()
				r.Use(middleware.RequestID)
				r.Use(middleware.BodyLimit(50)) // 50 bytes max
				r.Post("/test", func(w http.ResponseWriter, r *http.Request) {
					var p samplePayload
					if err := middleware.DecodeJSON(w, r, &p); err != nil {
						return
					}
					middleware.WriteJSON(w, http.StatusOK, p)
				})
				return r
			},
			method:         http.MethodPost,
			path:           "/test",
			body:           `{"name": "` + strings.Repeat("A", 100) + `"}`,
			expectedStatus: http.StatusRequestEntityTooLarge,
			verifyResponse: func(t *testing.T, res *http.Response, bodyStr string) {
				var errResp middleware.ErrorResponse
				err := json.Unmarshal([]byte(bodyStr), &errResp)
				require.NoError(t, err)
				assert.Equal(t, "Request Entity Too Large", errResp.Error.Code)
				assert.Equal(t, "request body exceeds size limit", errResp.Error.Message)
			},
		},
		{
			name: "panic recovery returns 500 Internal Server Error",
			setupRouter: func() http.Handler {
				r := chi.NewRouter()
				r.Use(middleware.RequestID)
				r.Use(middleware.Recoverer(nil))
				r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
					panic("unhandled exception inside handler")
				})
				return r
			},
			method:         http.MethodGet,
			path:           "/panic",
			expectedStatus: http.StatusInternalServerError,
			verifyResponse: func(t *testing.T, res *http.Response, bodyStr string) {
				var errResp middleware.ErrorResponse
				err := json.Unmarshal([]byte(bodyStr), &errResp)
				require.NoError(t, err)
				assert.Equal(t, "Internal Server Error", errResp.Error.Code)
				assert.Equal(t, "Internal Server Error", errResp.Error.Message)
			},
		},
		{
			name: "stale ETag in If-Match returns 412 Precondition Failed",
			setupRouter: func() http.Handler {
				r := chi.NewRouter()
				r.Use(middleware.RequestID)
				r.Put("/resource", func(w http.ResponseWriter, r *http.Request) {
					currentETag := middleware.GenerateETag([]byte("current-version-v1"))
					if !middleware.CheckIfMatch(w, r, currentETag) {
						return
					}
					middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "updated"})
				})
				return r
			},
			method: http.MethodPut,
			path:   "/resource",
			headers: map[string]string{
				"If-Match": `"stale-etag-999"`,
			},
			body:           `{"name":"update"}`,
			expectedStatus: http.StatusPreconditionFailed,
			verifyResponse: func(t *testing.T, res *http.Response, bodyStr string) {
				var errResp middleware.ErrorResponse
				err := json.Unmarshal([]byte(bodyStr), &errResp)
				require.NoError(t, err)
				assert.Equal(t, "Precondition Failed", errResp.Error.Code)
				assert.Equal(t, "precondition failed: ETag mismatch", errResp.Error.Message)
			},
		},
		{
			name: "matching ETag in If-Match succeeds with 200 OK",
			setupRouter: func() http.Handler {
				r := chi.NewRouter()
				r.Use(middleware.RequestID)
				r.Put("/resource", func(w http.ResponseWriter, r *http.Request) {
					currentETag := middleware.GenerateETag([]byte("current-version-v1"))
					if !middleware.CheckIfMatch(w, r, currentETag) {
						return
					}
					middleware.WriteJSON(w, http.StatusOK, map[string]string{"status": "updated"})
				})
				return r
			},
			method: http.MethodPut,
			path:   "/resource",
			headers: map[string]string{
				"If-Match": middleware.GenerateETag([]byte("current-version-v1")),
			},
			body:           `{"name":"update"}`,
			expectedStatus: http.StatusOK,
			verifyResponse: func(t *testing.T, res *http.Response, bodyStr string) {
				assert.Contains(t, bodyStr, "updated")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := tt.setupRouter()
			var bodyReader io.Reader
			if tt.body != "" {
				bodyReader = strings.NewReader(tt.body)
			}
			req := httptest.NewRequest(tt.method, tt.path, bodyReader)
			req.Header.Set("Content-Type", "application/json")
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			res := rec.Result()
			defer res.Body.Close()

			resBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedStatus, res.StatusCode)
			if tt.verifyResponse != nil {
				tt.verifyResponse(t, res, string(resBody))
			}
		})
	}
}

func TestIdempotencyMiddleware(t *testing.T) {
	store := middleware.NewInMemoryIdempotencyStore()
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Idempotency(store))
	r.Post("/orders", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := middleware.DecodeJSON(w, r, &body); err != nil {
			return
		}
		middleware.WriteJSON(w, http.StatusCreated, map[string]string{
			"order_id": "ord-1001",
			"item":     body["item"],
		})
	})

	t.Run("first request creates resource", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{"item":"laptop"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "idemp-key-1")

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Contains(t, rec.Body.String(), "ord-1001")
	})

	t.Run("repeated request with same idempotency key and identical body returns cached response", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{"item":"laptop"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "idemp-key-1")

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, "IDEMPOTENT-HIT", rec.Header().Get("X-Cache"))
		assert.Contains(t, rec.Body.String(), "ord-1001")
	})

	t.Run("key reuse with different body returns 409 Conflict", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(`{"item":"smartphone"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "idemp-key-1")

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)

		var errResp middleware.ErrorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Equal(t, "Conflict", errResp.Error.Code)
		assert.Equal(t, "idempotency key reused with different request payload", errResp.Error.Message)
	})
}

func TestHealthAndReadinessEndpoints(t *testing.T) {
	registry := middleware.NewReadinessRegistry()
	registry.Register("database", func(ctx context.Context) error {
		return nil
	})

	r := middleware.NewRouter(middleware.RouterConfig{
		ReadinessRegistry: registry,
	})

	t.Run("healthz endpoint returns 200 OK", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp middleware.HealthResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "ok", resp.Status)
	})

	t.Run("readyz endpoint returns 200 OK when all checks pass", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp middleware.HealthResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "ok", resp.Status)
		assert.Equal(t, "ok", resp.Checks["database"])
	})

	t.Run("readyz endpoint returns 503 Service Unavailable when check fails", func(t *testing.T) {
		unhealthyRegistry := middleware.NewReadinessRegistry()
		unhealthyRegistry.Register("database", func(ctx context.Context) error {
			return nil
		})
		unhealthyRegistry.Register("cache", func(ctx context.Context) error {
			return errors.New("connection refused")
		})

		unhealthyRouter := middleware.NewRouter(middleware.RouterConfig{
			ReadinessRegistry: unhealthyRegistry,
		})

		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rec := httptest.NewRecorder()
		unhealthyRouter.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

		var resp middleware.HealthResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "degraded", resp.Status)
		assert.Equal(t, "ok", resp.Checks["database"])
		assert.Equal(t, "error: connection refused", resp.Checks["cache"])
	})
}

func TestIdentityAndRequestIDMiddleware(t *testing.T) {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Identity)
	r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		middleware.WriteJSON(w, http.StatusOK, map[string]string{
			"request_id": middleware.GetRequestID(ctx),
			"tenant_id":  middleware.GetTenantID(ctx),
			"user_id":    middleware.GetUserID(ctx),
		})
	})

	t.Run("propagates provided headers into context and response", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		req.Header.Set("X-Request-ID", "custom-req-id-777")
		req.Header.Set("X-Tenant-ID", "tenant-alpha")
		req.Header.Set("X-User-ID", "user-beta")

		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "custom-req-id-777", rec.Header().Get("X-Request-ID"))

		var body map[string]string
		err := json.Unmarshal(rec.Body.Bytes(), &body)
		require.NoError(t, err)
		assert.Equal(t, "custom-req-id-777", body["request_id"])
		assert.Equal(t, "tenant-alpha", body["tenant_id"])
		assert.Equal(t, "user-beta", body["user_id"])
	})

	t.Run("generates random request ID if not supplied", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/me", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		generatedReqID := rec.Header().Get("X-Request-ID")
		assert.NotEmpty(t, generatedReqID)

		var body map[string]string
		err := json.Unmarshal(rec.Body.Bytes(), &body)
		require.NoError(t, err)
		assert.Equal(t, generatedReqID, body["request_id"])
	})
}
