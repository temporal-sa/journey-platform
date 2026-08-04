package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/validated-pattern/journey-platform/internal/api/handlers"
	"github.com/validated-pattern/journey-platform/internal/api/middleware"
	"github.com/validated-pattern/journey-platform/internal/compiler"
	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
)

func setupTestRouter() (chi.Router, postgres.Repository) {
	r := chi.NewRouter()
	repo := postgres.NewMemoryRepository()
	h := handlers.New(repo, compiler.New(), nil)
	h.RegisterRoutes(r)
	return r, repo
}

func TestCatalogEndpoints(t *testing.T) {
	router, _ := setupTestRouter()

	endpoints := []string{
		"/api/v1/catalogs/events",
		"/api/v1/catalogs/actions",
		"/api/v1/catalogs/attributes",
		"/api/v1/catalogs/parameters",
		"/api/v1/catalogs/metrics",
		"/api/v1/catalogs/templates",
	}

	for _, ep := range endpoints {
		t.Run("GET "+ep, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, ep, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			var records []domain.CatalogRecord
			err := json.Unmarshal(rec.Body.Bytes(), &records)
			require.NoError(t, err)
			assert.NotEmpty(t, records)
		})
	}

	t.Run("Search catalog with query filter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/catalogs/events?q=cart", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var records []domain.CatalogRecord
		err := json.Unmarshal(rec.Body.Bytes(), &records)
		require.NoError(t, err)
		assert.NotEmpty(t, records)
		for _, rec := range records {
			assert.Contains(t, rec.RecordID+rec.Name+rec.Description, "Cart")
		}
	})
}

func TestJourneyDraftCRUDFlow(t *testing.T) {
	router, _ := setupTestRouter()

	var createdDraft domain.GraphDraft
	var initialETag string

	t.Run("Create draft POST /api/v1/journeys", func(t *testing.T) {
		input := domain.GraphDraft{
			Name:        "Test Journey",
			Description: "Initial test draft",
			Nodes: []domain.GraphNode{
				{ID: "node-1", Type: "trigger", Name: "Start Event"},
				{ID: "node-2", Type: "action", Name: "Send Notification"},
			},
			Edges: []domain.GraphEdge{
				{ID: "edge-1", Source: "node-1", Target: "node-2"},
			},
		}

		bodyBytes, _ := json.Marshal(input)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/journeys", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", "tenant-test")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		initialETag = rec.Header().Get("ETag")
		assert.NotEmpty(t, initialETag)

		err := json.Unmarshal(rec.Body.Bytes(), &createdDraft)
		require.NoError(t, err)
		assert.NotEmpty(t, createdDraft.DraftID)
		assert.Equal(t, "Test Journey", createdDraft.Name)
		assert.Equal(t, 1, createdDraft.Version)
	})

	t.Run("Get draft GET /api/v1/journeys/{id}", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/journeys/"+createdDraft.DraftID, nil)
		req.Header.Set("X-Tenant-ID", "tenant-test")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		etag := rec.Header().Get("ETag")
		assert.Equal(t, initialETag, etag)

		var fetchedDraft domain.GraphDraft
		err := json.Unmarshal(rec.Body.Bytes(), &fetchedDraft)
		require.NoError(t, err)
		assert.Equal(t, createdDraft.DraftID, fetchedDraft.DraftID)
		assert.Equal(t, "Test Journey", fetchedDraft.Name)
	})

	t.Run("Update draft PUT /api/v1/journeys/{id}/draft with matching If-Match", func(t *testing.T) {
		updateInput := createdDraft
		updateInput.Name = "Updated Journey Name"

		bodyBytes, _ := json.Marshal(updateInput)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/journeys/"+createdDraft.DraftID+"/draft", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Tenant-ID", "tenant-test")
		req.Header.Set("If-Match", initialETag)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		updatedETag := rec.Header().Get("ETag")
		assert.NotEmpty(t, updatedETag)
		assert.NotEqual(t, initialETag, updatedETag)

		var updatedDraft domain.GraphDraft
		err := json.Unmarshal(rec.Body.Bytes(), &updatedDraft)
		require.NoError(t, err)
		assert.Equal(t, "Updated Journey Name", updatedDraft.Name)
		assert.Equal(t, 2, updatedDraft.Version)
	})

	t.Run("Get non-existent draft returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/journeys/non-existent-id", nil)
		req.Header.Set("X-Tenant-ID", "tenant-test")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestStaleETagRejection(t *testing.T) {
	router, _ := setupTestRouter()

	input := domain.GraphDraft{
		Name: "Draft for ETag Test",
		Nodes: []domain.GraphNode{
			{ID: "node-1", Type: "trigger", Name: "Start"},
		},
	}
	bodyBytes, _ := json.Marshal(input)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/journeys", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-etag")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var created domain.GraphDraft
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	// Update with invalid / stale If-Match header
	updateInput := created
	updateInput.Name = "Stale Attempt"
	updateBytes, _ := json.Marshal(updateInput)

	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/journeys/"+created.DraftID+"/draft", bytes.NewReader(updateBytes))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("X-Tenant-ID", "tenant-etag")
	updateReq.Header.Set("If-Match", `"invalid-etag-hash-999"`)

	updateRec := httptest.NewRecorder()
	router.ServeHTTP(updateRec, updateReq)

	assert.Equal(t, http.StatusConflict, updateRec.Code)
	var errResp middleware.ErrorResponse
	err := json.Unmarshal(updateRec.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Error.Message, "ETag")
}

func TestValidationFailureHandling(t *testing.T) {
	router, _ := setupTestRouter()

	// Draft with invalid graph (e.g. disconnected nodes or invalid type)
	invalidDraft := domain.GraphDraft{
		Name: "Invalid Graph Draft",
		Nodes: []domain.GraphNode{
			{ID: "node-1", Type: "unknown_type_xxx", Name: "Bad Node"},
		},
	}

	bodyBytes, _ := json.Marshal(invalidDraft)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/journeys", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-val")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var created domain.GraphDraft
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	// Validate endpoint
	valReq := httptest.NewRequest(http.MethodPost, "/api/v1/journeys/"+created.DraftID+"/validate", nil)
	valReq.Header.Set("X-Tenant-ID", "tenant-val")

	valRec := httptest.NewRecorder()
	router.ServeHTTP(valRec, valReq)

	assert.Equal(t, http.StatusOK, valRec.Code)

	var valResult compiler.ValidationResult
	err := json.Unmarshal(valRec.Body.Bytes(), &valResult)
	require.NoError(t, err)
	assert.False(t, valResult.IsValid)
	assert.NotEmpty(t, valResult.Issues)
}

func TestSimulationPreview(t *testing.T) {
	router, _ := setupTestRouter()

	// Create valid messaging draft
	draft := domain.GraphDraft{
		Name: "Simulation Flow",
		Nodes: []domain.GraphNode{
			{
				ID:   "start-node",
				Type: "trigger",
				Name: "Start",
			},
			{
				ID:   "msg-node",
				Type: "email",
				Name: "Send Welcome Email",
				Config: map[string]interface{}{
					"subject": "Welcome to our platform!",
					"body":    "Hello {{subject.email}}, welcome!",
				},
			},
		},
		Edges: []domain.GraphEdge{
			{ID: "e1", Source: "start-node", Target: "msg-node"},
		},
	}

	bodyBytes, _ := json.Marshal(draft)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/journeys", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-sim")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var created domain.GraphDraft
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	// Simulate execution
	simPayload := map[string]interface{}{
		"event_payload": map[string]interface{}{"user_id": "usr_123"},
		"subject_profile": map[string]interface{}{"email": "test@example.com"},
	}
	simBytes, _ := json.Marshal(simPayload)

	simReq := httptest.NewRequest(http.MethodPost, "/api/v1/journeys/"+created.DraftID+"/simulate", bytes.NewReader(simBytes))
	simReq.Header.Set("Content-Type", "application/json")
	simReq.Header.Set("X-Tenant-ID", "tenant-sim")

	simRec := httptest.NewRecorder()
	router.ServeHTTP(simRec, simReq)

	assert.Equal(t, http.StatusOK, simRec.Code)

	var simResp struct {
		DraftID          string   `json:"draft_id"`
		Success          bool     `json:"success"`
		ExecutionPath    []string `json:"execution_path"`
		WouldSendPreviews []struct {
			NodeID string `json:"node_id"`
			Body   string `json:"body"`
		} `json:"would_send_previews"`
	}
	err := json.Unmarshal(simRec.Body.Bytes(), &simResp)
	require.NoError(t, err)
	assert.True(t, simResp.Success)
	assert.Equal(t, created.DraftID, simResp.DraftID)
	assert.NotEmpty(t, simResp.ExecutionPath)
}

func TestIdempotentPublishAndStateTransitions(t *testing.T) {
	router, _ := setupTestRouter()

	draft := domain.GraphDraft{
		Name: "Publish & Transition Flow",
		Nodes: []domain.GraphNode{
			{ID: "node-1", Type: "trigger", Name: "Start"},
		},
	}
	bodyBytes, _ := json.Marshal(draft)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/journeys", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-state")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var created domain.GraphDraft
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	t.Run("Publish draft returns 200 OK", func(t *testing.T) {
		pubReq := httptest.NewRequest(http.MethodPost, "/api/v1/journeys/"+created.DraftID+"/versions/publish", nil)
		pubReq.Header.Set("X-Tenant-ID", "tenant-state")

		pubRec := httptest.NewRecorder()
		router.ServeHTTP(pubRec, pubReq)

		assert.Equal(t, http.StatusOK, pubRec.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(pubRec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, created.DraftID, resp["draft_id"])
		assert.Equal(t, "active", resp["status"])
	})

	t.Run("Activate local journey", func(t *testing.T) {
		actReq := httptest.NewRequest(http.MethodPost, "/api/v1/journeys/"+created.DraftID+"/activate-local", nil)
		actReq.Header.Set("X-Tenant-ID", "tenant-state")

		actRec := httptest.NewRecorder()
		router.ServeHTTP(actRec, actReq)

		assert.Equal(t, http.StatusOK, actRec.Code)
		var resp map[string]string
		err := json.Unmarshal(actRec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, created.DraftID, resp["draft_id"])
		assert.Equal(t, "active", resp["status"])
		assert.NotEmpty(t, resp["activated_at"])
	})

	t.Run("Pause local journey", func(t *testing.T) {
		pauseReq := httptest.NewRequest(http.MethodPost, "/api/v1/journeys/"+created.DraftID+"/pause-local", nil)
		pauseReq.Header.Set("X-Tenant-ID", "tenant-state")

		pauseRec := httptest.NewRecorder()
		router.ServeHTTP(pauseRec, pauseReq)

		assert.Equal(t, http.StatusOK, pauseRec.Code)
		var resp map[string]string
		err := json.Unmarshal(pauseRec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, created.DraftID, resp["draft_id"])
		assert.Equal(t, "paused", resp["status"])
		assert.NotEmpty(t, resp["paused_at"])
	})

	t.Run("List versions returns draft version history", func(t *testing.T) {
		verReq := httptest.NewRequest(http.MethodGet, "/api/v1/journeys/"+created.DraftID+"/versions", nil)
		verReq.Header.Set("X-Tenant-ID", "tenant-state")

		verRec := httptest.NewRecorder()
		router.ServeHTTP(verRec, verReq)

		assert.Equal(t, http.StatusOK, verRec.Code)
		var resp struct {
			DraftID  string `json:"draft_id"`
			Versions []struct {
				Version int `json:"version"`
			} `json:"versions"`
		}
		err := json.Unmarshal(verRec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, created.DraftID, resp.DraftID)
		assert.NotEmpty(t, resp.Versions)
		assert.Equal(t, 1, resp.Versions[0].Version)
	})
}
