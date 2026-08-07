package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
)


// -----------------------------------------------------------------------------
// 1. Run Endpoints Tests
// -----------------------------------------------------------------------------

func TestRunEndpoints_SuccessAndNotFound(t *testing.T) {
	router, repo := setupTestRouter()

	// Seed an enrollment (run)
	ctx := t.Context()
	enrolledAt := time.Now().UTC()
	_, err := repo.CreateEnrollment(ctx, &postgres.Enrollment{
		TenantID:         "test-tenant",
		EnrollmentID:     "run-101",
		JourneyVersionID: "jv-101",
		SubjectID:        "subj-1",
		Status:           "active",
		CurrentNodeID:    "node-start",
		StateData:        []byte(`{"counter": 1}`),
		EnrolledAt:       enrolledAt,
		UpdatedAt:        enrolledAt,
	})
	if err != nil {
		t.Fatalf("failed to seed enrollment: %v", err)
	}

	// GET /api/v1/runs
	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs", nil)
	req.Header.Set("X-Tenant-ID", "test-tenant")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for ListRuns, got %d: %s", rec.Code, rec.Body.String())
	}

	var runs []domain.RunProjection
	if err := json.Unmarshal(rec.Body.Bytes(), &runs); err != nil {
		t.Fatalf("failed to parse runs response: %v", err)
	}
	if len(runs) != 1 || runs[0].RunID != "run-101" {
		t.Fatalf("expected 1 run with ID 'run-101', got %+v", runs)
	}

	// GET /api/v1/runs/run-101
	reqDetail := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-101", nil)
	reqDetail.Header.Set("X-Tenant-ID", "test-tenant")
	recDetail := httptest.NewRecorder()
	router.ServeHTTP(recDetail, reqDetail)

	if recDetail.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for GetRunTimeline, got %d: %s", recDetail.Code, recDetail.Body.String())
	}

	// GET /api/v1/runs/nonexistent -> 404
	req404 := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-nonexistent", nil)
	req404.Header.Set("X-Tenant-ID", "test-tenant")
	rec404 := httptest.NewRecorder()
	router.ServeHTTP(rec404, req404)

	if rec404.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for non-existent run, got %d", rec404.Code)
	}
}
func TestGetRunTimeline_RecordedLifecycleEvents(t *testing.T) {
	router, repo := setupTestRouter()
	ctx := t.Context()
	enrolledAt := time.Now().UTC()

	_, err := repo.CreateEnrollment(ctx, &postgres.Enrollment{
		TenantID:         "test-tenant",
		EnrollmentID:     "run-202",
		JourneyVersionID: "jv-202",
		SubjectID:        "subj-2",
		Status:           "completed",
		CurrentNodeID:    "node-exit",
		EnrolledAt:       enrolledAt,
		UpdatedAt:        enrolledAt,
	})
	if err != nil {
		t.Fatalf("failed to seed enrollment: %v", err)
	}

	events := []*postgres.LifecycleEvent{
		{
			TenantID:   "test-tenant",
			EventID:    "evt-1",
			EntityType: "workflow_run",
			EntityID:   "run-202",
			EventName:  "workflow_started",
			Payload:    []byte(`{"status":"running","workflow_id":"jv-202"}`),
			CreatedAt:  enrolledAt,
		},
		{
			TenantID:   "test-tenant",
			EventID:    "evt-2",
			EntityType: "workflow_run",
			EntityID:   "run-202",
			EventName:  "node_entered",
			Payload:    []byte(`{"node_id":"node-email","status":"running"}`),
			CreatedAt:  enrolledAt.Add(time.Second),
		},
		{
			TenantID:   "test-tenant",
			EventID:    "evt-3",
			EntityType: "workflow_run",
			EntityID:   "run-202",
			EventName:  "workflow_completed",
			Payload:    []byte(`{"node_id":"node-exit","status":"completed"}`),
			CreatedAt:  enrolledAt.Add(2 * time.Second),
		},
	}
	for _, le := range events {
		if _, err := repo.RecordLifecycleEvent(ctx, le); err != nil {
			t.Fatalf("failed to record lifecycle event: %v", err)
		}
	}

	reqDetail := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-202/timeline", nil)
	reqDetail.Header.Set("X-Tenant-ID", "test-tenant")
	recDetail := httptest.NewRecorder()
	router.ServeHTTP(recDetail, reqDetail)

	if recDetail.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for GetRunTimeline, got %d: %s", recDetail.Code, recDetail.Body.String())
	}

	var res map[string]interface{}
	if err := json.Unmarshal(recDetail.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse timeline response: %v", err)
	}

	timeline, ok := res["timeline"].([]interface{})
	if !ok || len(timeline) != 3 {
		t.Fatalf("expected timeline with 3 recorded lifecycle events, got %v", res["timeline"])
	}

	firstEvent := timeline[0].(map[string]interface{})
	if firstEvent["event_type"] != "workflow_started" || firstEvent["event_id"] != "evt-1" {
		t.Fatalf("expected first event to be workflow_started, got %+v", firstEvent)
	}

	secondEvent := timeline[1].(map[string]interface{})
	if secondEvent["event_type"] != "node_entered" || secondEvent["node_id"] != "node-email" {
		t.Fatalf("expected second event to be node_entered for node-email, got %+v", secondEvent)
	}
}

// -----------------------------------------------------------------------------
// 2. Experiment Endpoints Tests
// -----------------------------------------------------------------------------

func TestExperimentEndpoints(t *testing.T) {
	router, _ := setupTestRouter()

	// 1. Create valid experiment -> 201 Created
	expReq := domain.ExperimentDefinition{
		ExperimentID: "exp-001",
		Name:         "Checkout Button Color Test",
		Description:  "Testing blue vs green CTA",
		Variants: []domain.ExperimentVariant{
			{
				VariantID:         "control",
				Name:              "Control",
				WeightBasisPoints: 5000,
			},
			{
				VariantID:         "variant_green",
				Name:              "Green CTA",
				WeightBasisPoints: 5000,
			},
		},
		TargetAudience: "all_users",
	}

	bodyBytes, _ := json.Marshal(expReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/experiments", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "test-tenant")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for CreateExperiment, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("ETag") == "" {
		t.Fatalf("expected ETag header in CreateExperiment response")
	}

	// 2. GET /api/v1/experiments/exp-001 -> 200 OK
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/experiments/exp-001", nil)
	reqGet.Header.Set("X-Tenant-ID", "test-tenant")
	recGet := httptest.NewRecorder()
	router.ServeHTTP(recGet, reqGet)

	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for GetExperiment, got %d: %s", recGet.Code, recGet.Body.String())
	}

	// 3. GET /api/v1/experiments/exp-001/versions/v1 -> 200 OK
	reqVer := httptest.NewRequest(http.MethodGet, "/api/v1/experiments/exp-001/versions/v1", nil)
	reqVer.Header.Set("X-Tenant-ID", "test-tenant")
	recVer := httptest.NewRecorder()
	router.ServeHTTP(recVer, reqVer)

	if recVer.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for GetExperimentVersion, got %d", recVer.Code)
	}

	// 4. Invalid variant weights (8000 != 10000) -> 422 Unprocessable Entity
	badExpReq := domain.ExperimentDefinition{
		ExperimentID: "exp-bad-weights",
		Name:         "Invalid Weights Test",
		Variants: []domain.ExperimentVariant{
			{VariantID: "control", WeightBasisPoints: 4000},
			{VariantID: "v1", WeightBasisPoints: 4000},
		},
	}
	badBody, _ := json.Marshal(badExpReq)
	req422 := httptest.NewRequest(http.MethodPost, "/api/v1/experiments", bytes.NewReader(badBody))
	req422.Header.Set("Content-Type", "application/json")
	req422.Header.Set("X-Tenant-ID", "test-tenant")
	rec422 := httptest.NewRecorder()
	router.ServeHTTP(rec422, req422)

	if rec422.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity for invalid weights, got %d: %s", rec422.Code, rec422.Body.String())
	}
}

// -----------------------------------------------------------------------------
// 3. Report Endpoints & Privacy Rejections Tests
// -----------------------------------------------------------------------------

func TestReportEndpoints_RawFactAndSmallCellRejections(t *testing.T) {
	router, repo := setupTestRouter()

	// Seed experiment
	ctx := t.Context()
	variantsBytes, _ := json.Marshal([]domain.ExperimentVariant{
		{VariantID: "control", WeightBasisPoints: 5000},
		{VariantID: "variant_a", WeightBasisPoints: 5000},
	})
	_, _ = repo.CreateExperimentDefinition(ctx, &postgres.ExperimentDefinition{
		TenantID:     "test-tenant",
		ExperimentID: "exp-report-1",
		Name:         "Report Test Experiment",
		Status:       "active",
		Variants:     variantsBytes,
		CreatedAt:    time.Now().UTC(),
	})

	// 1. Success Path: GET /api/v1/reports/experiments/exp-report-1 -> 200 OK
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/experiments/exp-report-1", nil)
	req.Header.Set("X-Tenant-ID", "test-tenant")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for GetExperimentReport, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. CSV Export Success Path: GET /api/v1/reports/experiments/exp-report-1/export -> 200 OK text/csv
	reqCSV := httptest.NewRequest(http.MethodGet, "/api/v1/reports/experiments/exp-report-1/export", nil)
	reqCSV.Header.Set("X-Tenant-ID", "test-tenant")
	recCSV := httptest.NewRecorder()
	router.ServeHTTP(recCSV, reqCSV)

	if recCSV.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for ExportExperimentReportCSV, got %d: %s", recCSV.Code, recCSV.Body.String())
	}
	if ct := recCSV.Header().Get("Content-Type"); ct == "" || ct[:8] != "text/csv" {
		t.Fatalf("expected text/csv Content-Type, got %s", ct)
	}

	// 3. Raw Fact Rejection Path: raw=true -> 422 Unprocessable Entity
	reqRaw := httptest.NewRequest(http.MethodGet, "/api/v1/reports/experiments/exp-report-1?raw=true", nil)
	reqRaw.Header.Set("X-Tenant-ID", "test-tenant")
	recRaw := httptest.NewRecorder()
	router.ServeHTTP(recRaw, reqRaw)

	if recRaw.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity for raw fact query rejection, got %d: %s", recRaw.Code, recRaw.Body.String())
	}

	// 4. Raw Fact CSV Export Rejection Path: type=raw -> 422 Unprocessable Entity
	reqRawCSV := httptest.NewRequest(http.MethodGet, "/api/v1/reports/experiments/exp-report-1/export?type=raw", nil)
	reqRawCSV.Header.Set("X-Tenant-ID", "test-tenant")
	recRawCSV := httptest.NewRecorder()
	router.ServeHTTP(recRawCSV, reqRawCSV)

	if recRawCSV.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity for raw fact CSV export rejection, got %d: %s", recRawCSV.Code, recRawCSV.Body.String())
	}

	// 5. Small-Cell Rejection Path: simulate_small_cell=true -> 422 Unprocessable Entity
	reqSmallCell := httptest.NewRequest(http.MethodGet, "/api/v1/reports/experiments/exp-report-1?simulate_small_cell=true", nil)
	reqSmallCell.Header.Set("X-Tenant-ID", "test-tenant")
	recSmallCell := httptest.NewRecorder()
	router.ServeHTTP(recSmallCell, reqSmallCell)

	if recSmallCell.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 Unprocessable Entity for small-cell differential privacy rejection, got %d: %s", recSmallCell.Code, recSmallCell.Body.String())
	}
}

// -----------------------------------------------------------------------------
// 4. Static-List Endpoints & Deletion Tests
// -----------------------------------------------------------------------------

func TestStaticListEndpoints_UploadFinalizeGetDelete(t *testing.T) {
	router, _ := setupTestRouter()

	// 1. Upload CSV Static List -> 201 Created
	csvPayload := "member_id,recipient\nm-1,user1@example.com\nm-2,user2@example.com\n"
	reqUpload := httptest.NewRequest(http.MethodPost, "/api/v1/static-lists/upload", bytes.NewBufferString(csvPayload))
	reqUpload.Header.Set("Content-Type", "text/csv")
	reqUpload.Header.Set("X-Tenant-ID", "test-tenant")
	recUpload := httptest.NewRecorder()
	router.ServeHTTP(recUpload, reqUpload)

	if recUpload.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for UploadStaticList, got %d: %s", recUpload.Code, recUpload.Body.String())
	}

	var listResp domain.StaticList
	if err := json.Unmarshal(recUpload.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to unmarshal static list response: %v", err)
	}
	listID := listResp.ListID

	// 2. GET /api/v1/static-lists/{id} -> 200 OK
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/static-lists/"+listID, nil)
	reqGet.Header.Set("X-Tenant-ID", "test-tenant")
	recGet := httptest.NewRecorder()
	router.ServeHTTP(recGet, reqGet)

	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for GetStaticList, got %d", recGet.Code)
	}

	// 3. Finalize Static List -> 200 OK
	reqFinal := httptest.NewRequest(http.MethodPost, "/api/v1/static-lists/"+listID+"/finalize", nil)
	reqFinal.Header.Set("X-Tenant-ID", "test-tenant")
	recFinal := httptest.NewRecorder()
	router.ServeHTTP(recFinal, reqFinal)

	if recFinal.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for FinalizeStaticList, got %d: %s", recFinal.Code, recFinal.Body.String())
	}

	// 4. Finalize AGAIN -> 409 Conflict
	reqFinal2 := httptest.NewRequest(http.MethodPost, "/api/v1/static-lists/"+listID+"/finalize", nil)
	reqFinal2.Header.Set("X-Tenant-ID", "test-tenant")
	recFinal2 := httptest.NewRecorder()
	router.ServeHTTP(recFinal2, reqFinal2)

	if recFinal2.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict when re-finalizing static list, got %d: %s", recFinal2.Code, recFinal2.Body.String())
	}

	// 5. Idempotent Deletion: DELETE /api/v1/static-lists/{id} -> 204 No Content
	reqDel1 := httptest.NewRequest(http.MethodDelete, "/api/v1/static-lists/"+listID, nil)
	reqDel1.Header.Set("X-Tenant-ID", "test-tenant")
	recDel1 := httptest.NewRecorder()
	router.ServeHTTP(recDel1, reqDel1)

	if recDel1.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content for DeleteStaticList, got %d", recDel1.Code)
	}

	// Calling DELETE a second time must also return 204 No Content (Idempotent)
	reqDel2 := httptest.NewRequest(http.MethodDelete, "/api/v1/static-lists/"+listID, nil)
	reqDel2.Header.Set("X-Tenant-ID", "test-tenant")
	recDel2 := httptest.NewRecorder()
	router.ServeHTTP(recDel2, reqDel2)

	if recDel2.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content for second idempotent DeleteStaticList call, got %d", recDel2.Code)
	}
}

// -----------------------------------------------------------------------------
// 5. Test-Run Endpoints & Idempotency Tests
// -----------------------------------------------------------------------------

func TestTestRunEndpoints_StartCancelIdempotency(t *testing.T) {
	router, repo := setupTestRouter()
	ctx := t.Context()

	trPayload := map[string]interface{}{
		"draft_id": "draft-101",
		"ir_id":    "ir-101",
		"mock_inputs": map[string]interface{}{
			"user_id": "usr-test-1",
		},
	}
	bodyBytes, _ := json.Marshal(trPayload)

	// 1. Idempotent Test Run Creation (First Call) -> 201 Created
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/test-runs", bytes.NewReader(bodyBytes))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("X-Tenant-ID", "test-tenant")
	req1.Header.Set("Idempotency-Key", "idem-tr-key-999")
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for StartTestRun, got %d: %s", rec1.Code, rec1.Body.String())
	}

	var trResp1 domain.TestRun
	if err := json.Unmarshal(rec1.Body.Bytes(), &trResp1); err != nil {
		t.Fatalf("failed to unmarshal test run response: %v", err)
	}
	testRunID := trResp1.TestRunID

	// 2. Idempotent Test Run Creation (Second Call with SAME Idempotency-Key) -> 200 OK or 201 Created with SAME TestRunID
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/test-runs", bytes.NewReader(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Tenant-ID", "test-tenant")
	req2.Header.Set("Idempotency-Key", "idem-tr-key-999")
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK && rec2.Code != http.StatusCreated {
		t.Fatalf("expected 200/201 for idempotent StartTestRun, got %d: %s", rec2.Code, rec2.Body.String())
	}

	var trResp2 domain.TestRun
	_ = json.Unmarshal(rec2.Body.Bytes(), &trResp2)
	if trResp2.TestRunID != testRunID {
		t.Fatalf("expected idempotent response to match test_run_id '%s', got '%s'", testRunID, trResp2.TestRunID)
	}

	// 3. GET /api/v1/test-runs/{id} -> 200 OK with status and member results
	reqGet := httptest.NewRequest(http.MethodGet, "/api/v1/test-runs/"+testRunID, nil)
	reqGet.Header.Set("X-Tenant-ID", "test-tenant")
	recGet := httptest.NewRecorder()
	router.ServeHTTP(recGet, reqGet)

	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for GetTestRunStatus, got %d: %s", recGet.Code, recGet.Body.String())
	}

	// 4. Cancel Test Run -> 200 OK (Seed active/pending test run first)
	_, _ = repo.CreateTestRun(ctx, &postgres.TestRun{
		TenantID:  "test-tenant",
		TestRunID: "tr-active-cancel",
		Status:    "pending",
	})

	reqCancel := httptest.NewRequest(http.MethodPost, "/api/v1/test-runs/tr-active-cancel/cancel", nil)
	reqCancel.Header.Set("X-Tenant-ID", "test-tenant")
	recCancel := httptest.NewRecorder()
	router.ServeHTTP(recCancel, reqCancel)

	if recCancel.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for CancelTestRun, got %d: %s", recCancel.Code, recCancel.Body.String())
	}

	// 5. Cancel ALREADY Cancelled Test Run -> 409 Conflict
	reqCancel2 := httptest.NewRequest(http.MethodPost, "/api/v1/test-runs/tr-active-cancel/cancel", nil)
	reqCancel2.Header.Set("X-Tenant-ID", "test-tenant")
	recCancel2 := httptest.NewRecorder()
	router.ServeHTTP(recCancel2, reqCancel2)

	if recCancel2.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict when cancelling terminal test run, got %d: %s", recCancel2.Code, recCancel2.Body.String())
	}
}
func TestStartTestRun_WithStaticList(t *testing.T) {
	router, repo := setupTestRouter()
	ctx := context.Background()

	// Seed static list under "default" tenant
	itemsJSON, _ := json.Marshal([]map[string]interface{}{
		{"member_id": "m-001", "recipient": "alice@example.com", "tier": "gold"},
		{"member_id": "m-002", "recipient": "bob@example.com", "tier": "silver"},
	})
	_, err := repo.CreateStaticList(ctx, &postgres.StaticList{
		TenantID:           "default",
		ListID:             "list-test-audience",
		Name:               "Test Audience List",
		ItemCount:          2,
		DataClassification: "NonPII",
		Items:              itemsJSON,
		ContentHash:        "hash-test-aud",
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("failed to seed static list: %v", err)
	}

	// 1. Execute Test Run with Static List under non-default tenant ("tenant-custom")
	trPayload := map[string]interface{}{
		"draft_id":       "draft-101",
		"static_list_id": "list-test-audience",
		"mock_inputs": map[string]interface{}{
			"execution_mode": "realistic",
		},
	}
	bodyBytes, _ := json.Marshal(trPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/test-runs", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant-custom")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for StartTestRun with static list, got %d: %s", rec.Code, rec.Body.String())
	}

	var trResp domain.TestRun
	if err := json.Unmarshal(rec.Body.Bytes(), &trResp); err != nil {
		t.Fatalf("failed to unmarshal test run response: %v", err)
	}

	if trResp.MockInputs["static_list_id"] != "list-test-audience" {
		t.Fatalf("expected mock_inputs.static_list_id to be 'list-test-audience', got %v", trResp.MockInputs["static_list_id"])
	}
}

// -----------------------------------------------------------------------------
// 6. Additional Error Codes & Rate Limit Tests (400, 413, 429)
// -----------------------------------------------------------------------------

func TestPublicAPI_ErrorCodes(t *testing.T) {
	router, _ := setupTestRouter()

	// 400 Bad Request: Malformed JSON
	req400 := httptest.NewRequest(http.MethodPost, "/api/v1/experiments", bytes.NewBufferString("{invalid json"))
	req400.Header.Set("Content-Type", "application/json")
	req400.Header.Set("X-Tenant-ID", "test-tenant")
	rec400 := httptest.NewRecorder()
	router.ServeHTTP(rec400, req400)

	if rec400.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for malformed JSON, got %d: %s", rec400.Code, rec400.Body.String())
	}

	// 413 Payload Too Large
	req413 := httptest.NewRequest(http.MethodPost, "/api/v1/static-lists/upload", bytes.NewBufferString("oversized data"))
	req413.Header.Set("Content-Type", "text/csv")
	req413.Header.Set("X-Tenant-ID", "test-tenant")
	req413.Header.Set("X-Simulate-Oversized", "true")
	rec413 := httptest.NewRecorder()
	router.ServeHTTP(rec413, req413)

	if rec413.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 Request Entity Too Large, got %d: %s", rec413.Code, rec413.Body.String())
	}

	// 429 Too Many Requests
	req429 := httptest.NewRequest(http.MethodGet, "/api/v1/runs", nil)
	req429.Header.Set("X-Tenant-ID", "test-tenant")
	req429.Header.Set("X-Test-Rate-Limit", "true")
	rec429 := httptest.NewRecorder()
	router.ServeHTTP(rec429, req429)

	if rec429.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 Too Many Requests, got %d: %s", rec429.Code, rec429.Body.String())
	}
}
