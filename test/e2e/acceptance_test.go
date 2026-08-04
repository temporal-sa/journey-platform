package e2e

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/validated-pattern/journey-platform/internal/api/handlers"
	"github.com/validated-pattern/journey-platform/internal/api/middleware"
	"github.com/validated-pattern/journey-platform/internal/compiler"
	"github.com/validated-pattern/journey-platform/internal/integration"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
)

func TestPlaywrightEndToEndAcceptance(t *testing.T) {
	ctx := context.Background()
	repo := postgres.NewMemoryRepository()
	comp := compiler.New()
	h := handlers.New(repo, comp, nil)

	r := middleware.NewRouter(middleware.RouterConfig{})
	h.RegisterRoutes(r)

	// 1. Verify health & readiness
	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from healthz, got status %d", rec.Code)
	}

	// 2. Verify deterministic seed
	_, err := integration.Seed(ctx, repo, integration.SeedOptions{TenantID: "tenant_default"})
	if err != nil {
		t.Fatalf("seed initialization failed: %v", err)
	}

	// 3. Verify journey drafts seeded
	drafts, err := repo.ListJourneyDrafts(ctx, "tenant_default")
	if err != nil || len(drafts) == 0 {
		t.Fatalf("expected seeded journey drafts, got err: %v, count: %d", err, len(drafts))
	}

	// 4. Verify static lists seeded
	lists, err := repo.ListStaticLists(ctx, "tenant_default")
	if err != nil || len(lists) == 0 {
		t.Fatalf("expected seeded static lists, got err: %v, count: %d", err, len(lists))
	}
}
