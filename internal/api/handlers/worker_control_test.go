package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWorkerControlEndpoints(t *testing.T) {
	router, _ := setupTestRouter()

	t.Run("GET /api/v1/worker/status initial stopped state", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/worker/status", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var res map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if res["status"] != "stopped" {
			t.Errorf("expected status 'stopped', got '%v'", res["status"])
		}
		if pid, ok := res["pid"].(float64); !ok || pid != 0 {
			t.Errorf("expected pid 0, got '%v'", res["pid"])
		}
		if uptime, ok := res["uptime_seconds"].(float64); !ok || uptime != 0 {
			t.Errorf("expected uptime_seconds 0, got '%v'", res["uptime_seconds"])
		}
	})

	t.Run("POST /api/v1/worker/stop when stopped", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/worker/stop", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		var res map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if res["status"] != "stopped" {
			t.Errorf("expected status 'stopped', got '%v'", res["status"])
		}
	})
}
