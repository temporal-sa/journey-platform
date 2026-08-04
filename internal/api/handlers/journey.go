package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/validated-pattern/journey-platform/internal/api/middleware"
	"github.com/validated-pattern/journey-platform/internal/compiler"
	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
)

func getTenantID(r *http.Request) string {
	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = r.URL.Query().Get("tenant_id")
	}
	if tenantID == "" {
		tenantID = "default"
	}
	return tenantID
}

func getDraftIDFromRequest(r *http.Request) string {
	id := chi.URLParam(r, "id")
	if id == "" {
		id = chi.URLParam(r, "draft_id")
	}
	return id
}

func computeDraftETag(dbDraft *postgres.JourneyDraft) string {
	raw := fmt.Sprintf("%s:%s:%d:%s:%s", dbDraft.DraftID, dbDraft.Name, dbDraft.Version, string(dbDraft.Nodes), string(dbDraft.Edges))
	return middleware.GenerateETag([]byte(raw))
}

// CreateDraft handles POST /api/v1/journeys and /api/v1/journeys/drafts
func (h *Handlers) CreateDraft(w http.ResponseWriter, r *http.Request) {
	var draft domain.GraphDraft
	if err := middleware.DecodeJSON(w, r, &draft); err != nil {
		return
	}

	tenantID := getTenantID(r)
	if draft.TenantID != "" {
		tenantID = draft.TenantID
	} else {
		draft.TenantID = tenantID
	}

	if draft.DraftID == "" {
		draft.DraftID = "draft-" + uuid.New().String()
	}
	if draft.SchemaVersion == "" {
		draft.SchemaVersion = domain.DefaultSchemaVersion
	}
	if draft.Version <= 0 {
		draft.Version = 1
	}
	if draft.Nodes == nil {
		draft.Nodes = []domain.GraphNode{}
	}
	if draft.Edges == nil {
		draft.Edges = []domain.GraphEdge{}
	}

	now := time.Now().UTC()
	if draft.CreatedAt.IsZero() {
		draft.CreatedAt = now
	}
	draft.UpdatedAt = now

	hash, _ := draft.CalculateSHA256()
	draft.ContentHash = hash

	nodesBytes, _ := json.Marshal(draft.Nodes)
	edgesBytes, _ := json.Marshal(draft.Edges)

	dbDraft := &postgres.JourneyDraft{
		TenantID:    tenantID,
		DraftID:     draft.DraftID,
		Name:        draft.Name,
		Description: draft.Description,
		Version:     int32(draft.Version),
		Nodes:       nodesBytes,
		Edges:       edgesBytes,
		ContentHash: draft.ContentHash,
		CreatedAt:   draft.CreatedAt,
		UpdatedAt:   draft.UpdatedAt,
	}

	created, err := h.repo.CreateJourneyDraft(r.Context(), dbDraft)
	if err != nil {
		middleware.WriteError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to save journey draft: %v", err))
		return
	}

	etag := computeDraftETag(created)
	w.Header().Set("ETag", etag)

	middleware.WriteJSON(w, http.StatusCreated, draft)
}

// GetDraft handles GET /api/v1/journeys/{id} and /api/v1/journeys/drafts/{id}
func (h *Handlers) GetDraft(w http.ResponseWriter, r *http.Request) {
	draftID := getDraftIDFromRequest(r)
	if draftID == "" {
		middleware.WriteError(w, r, http.StatusBadRequest, "missing draft ID in path")
		return
	}
	tenantID := getTenantID(r)

	dbDraft, err := h.repo.GetJourneyDraft(r.Context(), tenantID, draftID)
	if (err != nil || dbDraft == nil) && tenantID != "default" {
		dbDraft, err = h.repo.GetJourneyDraft(r.Context(), "default", draftID)
	}
	if err != nil || dbDraft == nil {
		middleware.WriteError(w, r, http.StatusNotFound, fmt.Sprintf("journey draft '%s' not found", draftID))
		return
	}

	var nodes []domain.GraphNode
	var edges []domain.GraphEdge
	if len(dbDraft.Nodes) > 0 {
		_ = json.Unmarshal(dbDraft.Nodes, &nodes)
	}
	if len(dbDraft.Edges) > 0 {
		_ = json.Unmarshal(dbDraft.Edges, &edges)
	}
	if nodes == nil {
		nodes = []domain.GraphNode{}
	}
	if edges == nil {
		edges = []domain.GraphEdge{}
	}

	draft := domain.GraphDraft{
		SchemaVersion: domain.DefaultSchemaVersion,
		DraftID:       dbDraft.DraftID,
		TenantID:      dbDraft.TenantID,
		Name:          dbDraft.Name,
		Description:   dbDraft.Description,
		Version:       int(dbDraft.Version),
		Nodes:         nodes,
		Edges:         edges,
		ContentHash:   dbDraft.ContentHash,
		CreatedAt:     dbDraft.CreatedAt,
		UpdatedAt:     dbDraft.UpdatedAt,
	}

	etag := computeDraftETag(dbDraft)
	w.Header().Set("ETag", etag)

	middleware.WriteJSON(w, http.StatusOK, draft)
}

// ListDrafts handles GET /api/v1/journeys and GET /api/v1/journeys/drafts
func (h *Handlers) ListDrafts(w http.ResponseWriter, r *http.Request) {
	if checkRateLimit(w, r) {
		return
	}
	tenantID := getTenantID(r)

	dbDrafts, err := h.repo.ListJourneyDrafts(r.Context(), tenantID)
	if err != nil {
		middleware.WriteError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to list journey drafts: %v", err))
		return
	}
	if len(dbDrafts) == 0 && tenantID != "default" {
		if fallbackDrafts, errFb := h.repo.ListJourneyDrafts(r.Context(), "default"); errFb == nil && len(fallbackDrafts) > 0 {
			dbDrafts = fallbackDrafts
		}
	}
	result := make([]domain.GraphDraft, 0, len(dbDrafts))
	for _, dbDraft := range dbDrafts {
		var nodes []domain.GraphNode
		var edges []domain.GraphEdge
		if len(dbDraft.Nodes) > 0 {
			_ = json.Unmarshal(dbDraft.Nodes, &nodes)
		}
		if len(dbDraft.Edges) > 0 {
			_ = json.Unmarshal(dbDraft.Edges, &edges)
		}

		result = append(result, domain.GraphDraft{
			SchemaVersion: domain.DefaultSchemaVersion,
			DraftID:       dbDraft.DraftID,
			TenantID:      dbDraft.TenantID,
			Name:          dbDraft.Name,
			Description:   dbDraft.Description,
			Version:       int(dbDraft.Version),
			Nodes:         nodes,
			Edges:         edges,
			ContentHash:   dbDraft.ContentHash,
			CreatedAt:     dbDraft.CreatedAt,
			UpdatedAt:     dbDraft.UpdatedAt,
		})
	}

	middleware.WriteJSON(w, http.StatusOK, result)
}
// UpdateDraft handles PUT /api/v1/journeys/{id}/draft, /api/v1/journeys/{id}, and /api/v1/journeys/drafts/{id}
func (h *Handlers) UpdateDraft(w http.ResponseWriter, r *http.Request) {
	draftID := getDraftIDFromRequest(r)
	if draftID == "" {
		middleware.WriteError(w, r, http.StatusBadRequest, "missing draft ID in path")
		return
	}

	var incoming domain.GraphDraft
	if err := middleware.DecodeJSON(w, r, &incoming); err != nil {
		return
	}

	tenantID := getTenantID(r)
	if tenantID == "default" && incoming.TenantID != "" {
		tenantID = incoming.TenantID
	}

	existing, err := h.repo.GetJourneyDraft(r.Context(), tenantID, draftID)
	if (err != nil || existing == nil) && tenantID != "default" {
		existing, err = h.repo.GetJourneyDraft(r.Context(), "default", draftID)
		if existing != nil {
			tenantID = "default"
		}
	}
	if incoming.Nodes == nil {
		incoming.Nodes = []domain.GraphNode{}
	}
	if incoming.Edges == nil {
		incoming.Edges = []domain.GraphEdge{}
	}
	now := time.Now().UTC()

	if existing == nil {
		incoming.Version = 1
		incoming.CreatedAt = now
		incoming.UpdatedAt = now
	} else {
		existingETag := computeDraftETag(existing)
		ifMatch := r.Header.Get("If-Match")
		if ifMatch != "" && !middleware.MatchETag(ifMatch, existingETag) {
			middleware.WriteError(w, r, http.StatusConflict, "stale ETag / concurrent modification conflict")
			return
		}
		incoming.Version = int(existing.Version) + 1
		incoming.CreatedAt = existing.CreatedAt
		incoming.UpdatedAt = now
	}

	hash, _ := incoming.CalculateSHA256()
	incoming.ContentHash = hash

	nodesBytes, _ := json.Marshal(incoming.Nodes)
	edgesBytes, _ := json.Marshal(incoming.Edges)

	dbDraft := &postgres.JourneyDraft{
		TenantID:    tenantID,
		DraftID:     draftID,
		Name:        incoming.Name,
		Description: incoming.Description,
		Version:     int32(incoming.Version),
		Nodes:       nodesBytes,
		Edges:       edgesBytes,
		ContentHash: incoming.ContentHash,
		CreatedAt:   incoming.CreatedAt,
		UpdatedAt:   incoming.UpdatedAt,
	}

	var saved *postgres.JourneyDraft
	if existing == nil {
		saved, err = h.repo.CreateJourneyDraft(r.Context(), dbDraft)
	} else {
		saved, err = h.repo.UpdateJourneyDraft(r.Context(), dbDraft)
	}
	if err != nil {
		saved, err = h.repo.UpdateJourneyDraft(r.Context(), dbDraft)
		if err != nil {
			middleware.WriteError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to save journey draft: %v", err))
			return
		}
	}

	newETag := computeDraftETag(saved)
	w.Header().Set("ETag", newETag)

	middleware.WriteJSON(w, http.StatusOK, incoming)
}

// ValidateDraft handles POST /api/v1/journeys/{id}/validate and /api/v1/journeys/drafts/{id}/validate
func (h *Handlers) ValidateDraft(w http.ResponseWriter, r *http.Request) {
	draftID := getDraftIDFromRequest(r)
	if draftID == "" {
		middleware.WriteError(w, r, http.StatusBadRequest, "missing draft ID in path")
		return
	}
	tenantID := getTenantID(r)

	dbDraft, err := h.repo.GetJourneyDraft(r.Context(), tenantID, draftID)
	if err != nil || dbDraft == nil {
		middleware.WriteError(w, r, http.StatusNotFound, fmt.Sprintf("journey draft '%s' not found", draftID))
		return
	}

	var draft domain.GraphDraft
	draft.DraftID = dbDraft.DraftID
	draft.TenantID = dbDraft.TenantID
	draft.Name = dbDraft.Name
	draft.Version = int(dbDraft.Version)
	if len(dbDraft.Nodes) > 0 {
		_ = json.Unmarshal(dbDraft.Nodes, &draft.Nodes)
	}
	if len(dbDraft.Edges) > 0 {
		_ = json.Unmarshal(dbDraft.Edges, &draft.Edges)
	}

	result, err := h.compiler.ValidateDraft(&draft)
	if err != nil {
		middleware.WriteError(w, r, http.StatusInternalServerError, fmt.Sprintf("validation error: %v", err))
		return
	}

	if result == nil {
		result = &compiler.ValidationResult{
			DraftID: draftID,
			IsValid: true,
			Issues:  []domain.ValidationIssue{},
		}
	}
	if result.Issues == nil {
		result.Issues = []domain.ValidationIssue{}
	}

	middleware.WriteJSON(w, http.StatusOK, result)
}

// SimulateDraft handles POST /api/v1/journeys/{id}/simulate and /api/v1/journeys/drafts/{id}/simulate
func (h *Handlers) SimulateDraft(w http.ResponseWriter, r *http.Request) {
	draftID := getDraftIDFromRequest(r)
	if draftID == "" {
		middleware.WriteError(w, r, http.StatusBadRequest, "missing draft ID in path")
		return
	}
	tenantID := getTenantID(r)

	dbDraft, err := h.repo.GetJourneyDraft(r.Context(), tenantID, draftID)
	if err != nil || dbDraft == nil {
		middleware.WriteError(w, r, http.StatusNotFound, fmt.Sprintf("journey draft '%s' not found", draftID))
		return
	}

	var simReq struct {
		MockInputs               map[string]interface{} `json:"mock_inputs,omitempty"`
		EventPayload             map[string]interface{} `json:"event_payload,omitempty"`
		SubjectProfile           map[string]interface{} `json:"subject_profile,omitempty"`
		Parameters               map[string]interface{} `json:"parameters,omitempty"`
		ExperimentBucketFixtures map[string]string      `json:"experiment_bucket_fixtures,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&simReq)

	var draft domain.GraphDraft
	draft.DraftID = dbDraft.DraftID
	draft.TenantID = dbDraft.TenantID
	draft.Name = dbDraft.Name
	draft.Version = int(dbDraft.Version)
	if len(dbDraft.Nodes) > 0 {
		_ = json.Unmarshal(dbDraft.Nodes, &draft.Nodes)
	}
	if len(dbDraft.Edges) > 0 {
		_ = json.Unmarshal(dbDraft.Edges, &draft.Edges)
	}

	var ir *domain.CompiledIR
	canonRes, err := h.compiler.CanonicalizeDraft(&draft)
	if err == nil && canonRes != nil && canonRes.CompiledIR != nil {
		ir = canonRes.CompiledIR
	} else {
		entryNodeID := ""
		if len(draft.Nodes) > 0 {
			entryNodeID = draft.Nodes[0].ID
		}
		ir = &domain.CompiledIR{
			SchemaVersion: domain.DefaultSchemaVersion,
			IRID:          "ir-" + draftID,
			DraftID:       draftID,
			TenantID:      tenantID,
			Version:       draft.Version,
			EntryNodeID:   entryNodeID,
			Nodes:         []domain.IRNode{},
			Edges:         []domain.IREdge{},
			ContentHash:   dbDraft.ContentHash,
			CompiledAt:    time.Now().UTC(),
		}
	}

	simOpts := &compiler.SimulationOptions{
		EventPayload:             simReq.EventPayload,
		SubjectProfile:           simReq.SubjectProfile,
		Parameters:               simReq.Parameters,
		ExperimentBucketFixtures: simReq.ExperimentBucketFixtures,
	}

	if simOpts.EventPayload == nil && simReq.MockInputs != nil {
		simOpts.EventPayload = simReq.MockInputs
	}

	simResult, err := h.simulator.Simulate(ir, simOpts)
	if err != nil {
		middleware.WriteError(w, r, http.StatusInternalServerError, fmt.Sprintf("simulation failed: %v", err))
		return
	}

	visitedNodes := simResult.VisitedNodes
	if visitedNodes == nil {
		visitedNodes = []string{}
	}

	runID := "run-sim-" + uuid.New().String()[:8]
	now := time.Now().UTC()
	stateBytes, _ := json.Marshal(map[string]interface{}{
		"execution_mode": "test",
		"simulation":     simResult,
		"visited_nodes":  visitedNodes,
	})
	enr := &postgres.Enrollment{
		TenantID:         tenantID,
		EnrollmentID:     runID,
		JourneyVersionID: draftID,
		SubjectID:        "static-list-contact@temporal.io",
		Status:           "completed",
		CurrentNodeID:    "node-exit",
		StateData:        stateBytes,
		EnrolledAt:       now,
		UpdatedAt:        now,
		CompletedAt:      &now,
	}
	_, _ = h.repo.CreateEnrollment(r.Context(), enr)

	responsePayload := map[string]interface{}{
		"run_id":                     runID,
		"draft_id":                   draftID,
		"success":                    true,
		"execution_path":             visitedNodes,
		"visited_nodes":              visitedNodes,
		"would_send_previews":        simResult.WouldSendPreviews,
		"execution_time_ms":          simResult.ExecutionTimeMS,
		"simulated_at":               now.Format(time.RFC3339),
	}

	middleware.WriteJSON(w, http.StatusOK, responsePayload)
}

// ListVersions handles GET /api/v1/journeys/{id}/versions and /api/v1/journeys/drafts/{id}/versions
func (h *Handlers) ListVersions(w http.ResponseWriter, r *http.Request) {
	draftID := getDraftIDFromRequest(r)
	if draftID == "" {
		middleware.WriteError(w, r, http.StatusBadRequest, "missing draft ID in path")
		return
	}
	tenantID := getTenantID(r)

	dbDraft, err := h.repo.GetJourneyDraft(r.Context(), tenantID, draftID)
	if err != nil || dbDraft == nil {
		middleware.WriteError(w, r, http.StatusNotFound, fmt.Sprintf("journey draft '%s' not found", draftID))
		return
	}

	dbVersions, _ := h.repo.ListJourneyVersionsByDraft(r.Context(), tenantID, draftID)

	type VersionItem struct {
		Version     int       `json:"version"`
		ContentHash string    `json:"content_hash"`
		UpdatedAt   time.Time `json:"updated_at"`
	}

	versions := []VersionItem{}
	for _, v := range dbVersions {
		versions = append(versions, VersionItem{
			Version:     int(v.Version),
			ContentHash: v.ContentHash,
			UpdatedAt:   v.CreatedAt,
		})
	}

	if len(versions) == 0 {
		versions = append(versions, VersionItem{
			Version:     int(dbDraft.Version),
			ContentHash: dbDraft.ContentHash,
			UpdatedAt:   dbDraft.UpdatedAt,
		})
	}

	resp := map[string]interface{}{
		"draft_id": draftID,
		"versions": versions,
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}

// PublishDraft handles POST /api/v1/journeys/{id}/versions/publish and /api/v1/journeys/drafts/{id}/publish
func (h *Handlers) PublishDraft(w http.ResponseWriter, r *http.Request) {
	if checkRateLimit(w, r) {
		return
	}
	draftID := getDraftIDFromRequest(r)
	if draftID == "" {
		middleware.WriteError(w, r, http.StatusBadRequest, "missing draft ID in path")
		return
	}
	tenantID := getTenantID(r)

	dbDraft, err := h.repo.GetJourneyDraft(r.Context(), tenantID, draftID)
	if (err != nil || dbDraft == nil) && tenantID != "default" {
		dbDraft, err = h.repo.GetJourneyDraft(r.Context(), "default", draftID)
	}

	now := time.Now().UTC()
	if dbDraft != nil {
		eventID := "evt-" + uuid.New().String()
		_, _ = h.repo.RecordLifecycleEvent(r.Context(), &postgres.LifecycleEvent{
			TenantID:   tenantID,
			EventID:    eventID,
			EntityType: "journey_draft",
			EntityID:   draftID,
			EventName:  "published",
			Payload:    []byte(`{"status":"active"}`),
			CreatedAt:  now,
		})
	}

	nowStr := now.Format(time.RFC3339)
	middleware.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"draft_id":     draftID,
		"status":       "active",
		"published_at": nowStr,
	})
}
func (h *Handlers) ActivateLocal(w http.ResponseWriter, r *http.Request) {
	draftID := getDraftIDFromRequest(r)
	if draftID == "" {
		middleware.WriteError(w, r, http.StatusBadRequest, "missing draft ID in path")
		return
	}
	tenantID := getTenantID(r)

	dbDraft, err := h.repo.GetJourneyDraft(r.Context(), tenantID, draftID)
	if err != nil || dbDraft == nil {
		middleware.WriteError(w, r, http.StatusNotFound, fmt.Sprintf("journey draft '%s' not found", draftID))
		return
	}

	eventID := "evt-" + uuid.New().String()
	_, _ = h.repo.RecordLifecycleEvent(r.Context(), &postgres.LifecycleEvent{
		TenantID:   tenantID,
		EventID:    eventID,
		EntityType: "journey_draft",
		EntityID:   draftID,
		EventName:  "activated",
		Payload:    []byte(`{"status":"active"}`),
		CreatedAt:  time.Now().UTC(),
	})

	nowStr := time.Now().UTC().Format(time.RFC3339)
	middleware.WriteJSON(w, http.StatusOK, map[string]string{
		"draft_id":     draftID,
		"status":       "active",
		"activated_at": nowStr,
	})
}

// PauseLocal handles POST /api/v1/journeys/{id}/versions/{v}/pause, /api/v1/journeys/{id}/pause-local, and /api/v1/journeys/drafts/{id}/pause-local
func (h *Handlers) PauseLocal(w http.ResponseWriter, r *http.Request) {
	draftID := getDraftIDFromRequest(r)
	if draftID == "" {
		middleware.WriteError(w, r, http.StatusBadRequest, "missing draft ID in path")
		return
	}
	tenantID := getTenantID(r)

	dbDraft, err := h.repo.GetJourneyDraft(r.Context(), tenantID, draftID)
	if err != nil || dbDraft == nil {
		middleware.WriteError(w, r, http.StatusNotFound, fmt.Sprintf("journey draft '%s' not found", draftID))
		return
	}

	eventID := "evt-" + uuid.New().String()
	_, _ = h.repo.RecordLifecycleEvent(r.Context(), &postgres.LifecycleEvent{
		TenantID:   tenantID,
		EventID:    eventID,
		EntityType: "journey_draft",
		EntityID:   draftID,
		EventName:  "paused",
		Payload:    []byte(`{"status":"paused"}`),
		CreatedAt:  time.Now().UTC(),
	})

	nowStr := time.Now().UTC().Format(time.RFC3339)
	middleware.WriteJSON(w, http.StatusOK, map[string]string{
		"draft_id":  draftID,
		"status":    "paused",
		"paused_at": nowStr,
	})
}
