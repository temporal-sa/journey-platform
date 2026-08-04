package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/api/handlers"
	"github.com/validated-pattern/journey-platform/internal/api/middleware"
	"github.com/validated-pattern/journey-platform/internal/compiler"
	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
)

func TestOpenAPIContractConformance(t *testing.T) {
	repo := postgres.NewMemoryRepository()
	comp := compiler.New()
	h := handlers.New(repo, comp, nil)

	r := middleware.NewRouter(middleware.RouterConfig{})
	h.RegisterRoutes(r)

	t.Run("Catalogs - 200 Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/catalogs/events", nil)
		req.Header.Set("X-Tenant-ID", "tenant_default")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rec.Code)
		}
	})

	t.Run("Journey Draft CRUD & Concurrency", func(t *testing.T) {
		// 1. Create Draft
		createBody := map[string]interface{}{
			"tenant_id":   "tenant_default",
			"name":        "Test Journey",
			"description": "Integration test journey",
		}
		b, _ := json.Marshal(createBody)
		req := httptest.NewRequest("POST", "/api/v1/journeys", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", "tenant_default")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}
		etag := rec.Header().Get("ETag")
		if etag == "" {
			t.Fatal("expected ETag header on created draft")
		}

		var draft domain.GraphDraft
		json.Unmarshal(rec.Body.Bytes(), &draft)

		// 2. Update Draft with valid ETag
		updateBody := map[string]interface{}{
			"name":        "Updated Test Journey",
			"description": "Updated description",
		}
		b, _ = json.Marshal(updateBody)
		req = httptest.NewRequest("PUT", "/api/v1/journeys/"+draft.DraftID+"/draft", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", "tenant_default")
		req.Header.Set("If-Match", etag)
		rec = httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on update, got %d: %s", rec.Code, rec.Body.String())
		}

		// 3. Update Draft with stale ETag -> 409 Conflict
		req = httptest.NewRequest("PUT", "/api/v1/journeys/"+draft.DraftID+"/draft", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", "tenant_default")
		req.Header.Set("If-Match", etag) // old stale etag
		rec = httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict for stale ETag, got %d", rec.Code)
		}
	})

	t.Run("Experiment Creation and Validation Failure", func(t *testing.T) {
		invalidExp := map[string]interface{}{
			"tenant_id":       "tenant_default",
			"name":            "Invalid Weight Exp",
			"control_variant": "control",
			"variants": []map[string]interface{}{
				{"key": "control", "name": "Control", "is_control": true, "weight_basis_points": 5000},
				{"key": "v1", "name": "V1", "is_control": false, "weight_basis_points": 4000}, // Total 9000 != 10000
			},
		}
		b, _ := json.Marshal(invalidExp)
		req := httptest.NewRequest("POST", "/api/v1/experiments", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", "tenant_default")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest && rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 400 or 422 for bad weights, got %d", rec.Code)
		}
	})

	t.Run("Report Differential Privacy Rejection", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/reports/experiments/non_existent_exp", nil)
		req.Header.Set("X-Tenant-ID", "tenant_default")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound && rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 404/422 for unprocessable small-cell/missing report, got %d", rec.Code)
		}
	})
}

func TestPersistenceRepositoryIntegration(t *testing.T) {
	ctx := context.Background()
	repo := postgres.NewMemoryRepository()

	draft := &postgres.JourneyDraft{
		DraftID:     "draft_integration_1",
		TenantID:    "tenant_1",
		Name:        "Integration Draft",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := repo.CreateJourneyDraft(ctx, draft)
	if err != nil {
		t.Fatalf("failed to create journey draft: %v", err)
	}

	got, err := repo.GetJourneyDraft(ctx, "tenant_1", "draft_integration_1")
	if err != nil {
		t.Fatalf("failed to get journey draft: %v", err)
	}
	if got.Name != "Integration Draft" {
		t.Fatalf("expected 'Integration Draft', got '%s'", got.Name)
	}

	// Idempotency constraint test
	idem := &postgres.IdempotencyKey{
		Key:       "idem_key_1",
		TenantID:  "tenant_1",
		Scope:     "test",
		Status:    "200",
		CreatedAt: time.Now(),
	}
	_, err = repo.CreateIdempotencyKey(ctx, idem)
	if err != nil {
		t.Fatalf("failed to create idempotency key: %v", err)
	}

	_, err = repo.CreateIdempotencyKey(ctx, idem)
	if err == nil {
		t.Fatal("expected error on duplicate idempotency key insert")
	}
}
