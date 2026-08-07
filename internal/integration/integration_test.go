package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
)

func TestSeedIdempotency(t *testing.T) {
	ctx := context.Background()
	repo := postgres.NewMemoryRepository()

	// First Seed Run
	res1, err := Seed(ctx, repo, SeedOptions{TenantID: "default"})
	if err != nil {
		t.Fatalf("First seed run failed: %v", err)
	}

	if res1.CatalogsSeeded == 0 {
		t.Errorf("Expected catalogs to be seeded, got 0")
	}
	if res1.StaticListsSeeded != 3 {
		t.Errorf("Expected 3 static lists seeded, got %d", res1.StaticListsSeeded)
	}
	if res1.ExperimentsSeeded != 1 {
		t.Errorf("Expected 1 experiment seeded, got %d", res1.ExperimentsSeeded)
	}
	if res1.JourneyDraftsSeeded != 3 {
		t.Errorf("Expected 3 journey drafts seeded, got %d", res1.JourneyDraftsSeeded)
	}
	if res1.JourneyVersionsSeeded != 2 {
		t.Errorf("Expected 2 journey versions seeded, got %d", res1.JourneyVersionsSeeded)
	}

	// Verify Seeding Contents in Repository
	catalogs, err := repo.ListCatalogs(ctx, "default")
	if err != nil {
		t.Fatalf("ListCatalogs failed: %v", err)
	}
	catalogCountFirstRun := len(catalogs)

	// Check Verified Contacts List
	contactsList, err := repo.GetStaticList(ctx, "default", "list_verified_contacts")
	if err != nil {
		t.Fatalf("GetStaticList list_verified_contacts failed: %v", err)
	}
	var contactIDs []string
	if err := json.Unmarshal(contactsList.Items, &contactIDs); err != nil {
		t.Fatalf("Unmarshal contact items failed: %v", err)
	}
	if len(contactIDs) != 5 {
		t.Errorf("Expected 5 verified contacts, got %d", len(contactIDs))
	}
	expectedContacts := map[string]bool{"user_1": true, "user_2": true, "user_3": true, "user_4": true, "user_5": true}
	for _, c := range contactIDs {
		if !expectedContacts[c] {
			t.Errorf("Unexpected contact ID: %s", c)
		}
	}

	// Check Suppression List
	suppList, err := repo.GetStaticList(ctx, "default", "communication_suppression_list")
	if err != nil {
		t.Fatalf("GetStaticList communication_suppression_list failed: %v", err)
	}
	if suppList.ItemCount == 0 {
		t.Errorf("Expected non-empty suppression list")
	}

	// Check Onboarding Journey Draft & Version
	onboardingDraft, err := repo.GetJourneyDraft(ctx, "default", "draft_onboarding_journey")
	if err != nil || onboardingDraft == nil {
		t.Fatalf("GetJourneyDraft draft_onboarding_journey failed: %v", err)
	}
	onboardingVer, err := repo.GetJourneyVersion(ctx, "default", "ver_onboarding_journey_v1")
	if err != nil || onboardingVer == nil {
		t.Fatalf("GetJourneyVersion ver_onboarding_journey_v1 failed: %v", err)
	}

	// Check Long-running Wait Journey Draft & Version
	waitDraft, err := repo.GetJourneyDraft(ctx, "default", "draft_order_wait_journey")
	if err != nil || waitDraft == nil {
		t.Fatalf("GetJourneyDraft draft_order_wait_journey failed: %v", err)
	}
	waitVer, err := repo.GetJourneyVersion(ctx, "default", "ver_order_wait_journey_v1")
	if err != nil || waitVer == nil {
		t.Fatalf("GetJourneyVersion ver_order_wait_journey_v1 failed: %v", err)
	}

	// Second Seed Run (Testing Idempotency: 0 errors, 0 duplicate rows)
	res2, err := Seed(ctx, repo, SeedOptions{TenantID: "default"})
	if err != nil {
		t.Fatalf("Second seed run failed: %v", err)
	}

	catalogsSecondRun, err := repo.ListCatalogs(ctx, "default")
	if err != nil {
		t.Fatalf("ListCatalogs after second seed failed: %v", err)
	}
	if len(catalogsSecondRun) != catalogCountFirstRun {
		t.Errorf("Idempotency violation! Catalog count changed: before=%d, after=%d", catalogCountFirstRun, len(catalogsSecondRun))
	}

	if res2.StaticListsSeeded != 3 || res2.ExperimentsSeeded != 1 {
		t.Errorf("Unexpected seed counts on second run: %+v", res2)
	}
}

func TestOnboardingJourneyEndToEndCorrelation(t *testing.T) {
	ctx := context.Background()
	cfg := LoadConfigFromEnv()

	stack, err := NewServiceStack(ctx, cfg, nil)
	if err != nil {
		t.Fatalf("NewServiceStack failed: %v", err)
	}

	// 1. Seed deterministic golden path data
	_, err = Seed(ctx, stack.Repo, SeedOptions{TenantID: "default"})
	if err != nil {
		t.Fatalf("Seed failed: %v", err)
	}

	// 2. Setup IDs for Onboarding Flow execution
	eventID := "evt_signup_10001"
	targetID := "user_1"
	workflowID := "wf_onboarding_user_1"
	runID := "run_onboarding_10001"
	nodeVisitID := "visit_exp_node_7030"
	assignmentID := "assign_exp_7030_user_1"
	exposureID := "exp_exp_7030_user_1"
	actionID := "act_email_user_1"
	providerReqID := "freq_provider_email_1001"
	outcomeID := "out_email_opened_1001"
	reportID := "rep_onboarding_daily_1001"

	// 3. Simulate Ingress & Workflow Execution
	// Record Enrollment in Postgres Store
	enrollment := &postgres.Enrollment{
		TenantID:         "default",
		EnrollmentID:     runID,
		JourneyVersionID: "ver_onboarding_journey_v1",
		SubjectID:        targetID,
		Status:           "active",
		CurrentNodeID:    "node_signup",
		StateData:        mustJSON(map[string]interface{}{"locale": "en-US", "event_id": eventID}),
		EnrolledAt:       time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	if _, err := stack.Repo.CreateEnrollment(ctx, enrollment); err != nil {
		t.Fatalf("CreateEnrollment failed: %v", err)
	}

	// Record Experiment Assignment & Exposure
	assignment := &postgres.Assignment{
		TenantID:          "default",
		AssignmentID:      assignmentID,
		ExperimentID:      "exp_onboarding_split",
		SubjectID:         targetID,
		VariantID:         "control",
		WeightBasisPoints: 7000,
		AssignedAt:        time.Now().UTC(),
		CreatedAt:         time.Now().UTC(),
	}
	if _, err := stack.Repo.CreateAssignment(ctx, assignment); err != nil {
		t.Fatalf("CreateAssignment failed: %v", err)
	}

	exposure := &postgres.Exposure{
		TenantID:          "default",
		ExposureID:        exposureID,
		ExperimentID:      "exp_onboarding_split",
		AssignmentID:      assignmentID,
		SubjectID:         targetID,
		VariantID:         assignment.VariantID,
		WeightBasisPoints: 7000,
		Context:           mustJSON(map[string]interface{}{"workflow_id": workflowID, "node_visit": nodeVisitID}),
		ExposedAt:         time.Now().UTC(),
		CreatedAt:         time.Now().UTC(),
	}
	if _, err := stack.Repo.CreateExposure(ctx, exposure); err != nil {
		t.Fatalf("CreateExposure failed: %v", err)
	}

	// Record Action Ledger Execution
	actLedger := &postgres.ActionLedger{
		TenantID:            "default",
		ActionID:            actionID,
		EnrollmentID:        runID,
		NodeID:              "node_email_action",
		ActivityType:        "SendEmail",
		Status:              "success",
		Output:              mustJSON(map[string]interface{}{"provider_request_id": providerReqID, "status": "sent"}),
		ErrorMessage:        "",
		ExecutionDurationMS: 45,
		CompletedAt:         pointerTime(time.Now().UTC()),
		CreatedAt:           time.Now().UTC(),
	}
	if _, err := stack.Repo.CreateActionLedger(ctx, actLedger); err != nil {
		t.Fatalf("CreateActionLedger failed: %v", err)
	}

	// 4. Update Run Projection Read Model
	leStart := &postgres.LifecycleEvent{
		TenantID:   "default",
		EventID:    eventID,
		EntityType: "workflow_run",
		EntityID:   runID,
		EventName:  "WorkflowStarted",
		Payload:    mustJSON(map[string]interface{}{"node_id": "node_signup"}),
		CreatedAt:  time.Now().UTC(),
	}
	proj, err := stack.Projector.ProjectLifecycleEvent(ctx, leStart)
	if err != nil {
		t.Fatalf("ProjectLifecycleEvent WorkflowStarted failed: %v", err)
	}
	if proj.Status != domain.RunStatusRunning {
		t.Errorf("Expected run projection status running, got %s", proj.Status)
	}

	leComplete := &postgres.LifecycleEvent{
		TenantID:   "default",
		EventID:    eventID + "_comp",
		EntityType: "workflow_run",
		EntityID:   runID,
		EventName:  "WorkflowCompleted",
		Payload:    mustJSON(map[string]interface{}{"node_id": "node_exit"}),
		CreatedAt:  time.Now().UTC(),
	}
	proj, err = stack.Projector.ProjectLifecycleEvent(ctx, leComplete)
	if err != nil {
		t.Fatalf("ProjectLifecycleEvent WorkflowCompleted failed: %v", err)
	}
	if proj.Status != domain.RunStatusCompleted {
		t.Errorf("Expected run projection status completed, got %s", proj.Status)
	}

	// 5. Verify Run Projection lookup without Temporal Visibility dependency
	retrievedProj, err := stack.Projector.GetRunProjection("default", runID)
	if err != nil {
		t.Fatalf("GetRunProjection failed: %v", err)
	}
	if retrievedProj.RunID != runID || retrievedProj.Status != domain.RunStatusCompleted {
		t.Errorf("Retrieved run projection mismatch: %+v", retrievedProj)
	}

	// 6. Verify Trace Correlation across all 10 correlation fields
	tc := NewTraceCorrelation(
		eventID,
		targetID,
		workflowID,
		nodeVisitID,
		assignmentID,
		exposureID,
		actionID,
		providerReqID,
		outcomeID,
		reportID,
	)

	if err := tc.VerifyCorrelation(); err != nil {
		t.Fatalf("Trace correlation verification failed: %v", err)
	}

	tcMap := tc.Map()
	if len(tcMap) != 10 {
		t.Errorf("Expected 10 trace correlation fields, got %d", len(tcMap))
	}
	if tcMap["event_id"] != eventID || tcMap["target_id"] != targetID {
		t.Errorf("Trace correlation values mismatched: %+v", tcMap)
	}
}

func pointerTime(t time.Time) *time.Time {
	return &t
}
