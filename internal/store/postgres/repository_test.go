package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/store/postgres"
)

const testTenant = "tenant-test-01"

func TestRepository_HappyPaths(t *testing.T) {
	ctx := context.Background()
	repo := postgres.NewMemoryRepository()

	// 1. Catalog
	t.Run("Catalog CRUD", func(t *testing.T) {
		cat := &postgres.Catalog{
			TenantID:         testTenant,
			RecordID:         "cat-001",
			Name:             "Email Trigger",
			ComponentType:    "trigger",
			Version:          "1.0.0",
			Description:      "Fires when email received",
			SchemaDefinition: []byte(`{"type":"object"}`),
			ContentHash:      "hash-123",
			Tags:             []string{"email", "trigger"},
			IsDeprecated:     false,
		}
		created, err := repo.CreateCatalog(ctx, cat)
		if err != nil {
			t.Fatalf("CreateCatalog failed: %v", err)
		}
		if created.RecordID != cat.RecordID {
			t.Errorf("expected RecordID %s, got %s", cat.RecordID, created.RecordID)
		}

		fetched, err := repo.GetCatalog(ctx, testTenant, "cat-001")
		if err != nil {
			t.Fatalf("GetCatalog failed: %v", err)
		}
		if fetched.Name != "Email Trigger" {
			t.Errorf("expected Name 'Email Trigger', got '%s'", fetched.Name)
		}

		list, err := repo.ListCatalogs(ctx, testTenant)
		if err != nil || len(list) != 1 {
			t.Fatalf("ListCatalogs failed: %v, count: %d", err, len(list))
		}

		cat.Name = "Email Trigger Updated"
		updated, err := repo.UpdateCatalog(ctx, cat)
		if err != nil || updated.Name != "Email Trigger Updated" {
			t.Fatalf("UpdateCatalog failed: %v", err)
		}

		err = repo.DeleteCatalog(ctx, testTenant, "cat-001")
		if err != nil {
			t.Fatalf("DeleteCatalog failed: %v", err)
		}
	})

	// 2. Journey Draft
	t.Run("JourneyDraft CRUD", func(t *testing.T) {
		draft := &postgres.JourneyDraft{
			TenantID:    testTenant,
			DraftID:     "draft-001",
			Name:        "Onboarding Flow",
			Description: "Initial user onboarding",
			Version:     1,
			Nodes:       []byte(`[{"id":"node-1"}]`),
			Edges:       []byte(`[]`),
			ContentHash: "hash-draft-1",
		}
		created, err := repo.CreateJourneyDraft(ctx, draft)
		if err != nil || created.DraftID != "draft-001" {
			t.Fatalf("CreateJourneyDraft failed: %v", err)
		}

		fetched, err := repo.GetJourneyDraft(ctx, testTenant, "draft-001")
		if err != nil || fetched.Name != "Onboarding Flow" {
			t.Fatalf("GetJourneyDraft failed: %v", err)
		}

		list, err := repo.ListJourneyDrafts(ctx, testTenant)
		if err != nil || len(list) != 1 {
			t.Fatalf("ListJourneyDrafts failed: %v", err)
		}

		draft.Name = "Onboarding Flow v2"
		updated, err := repo.UpdateJourneyDraft(ctx, draft)
		if err != nil || updated.Name != "Onboarding Flow v2" {
			t.Fatalf("UpdateJourneyDraft failed: %v", err)
		}

		err = repo.DeleteJourneyDraft(ctx, testTenant, "draft-001")
		if err != nil {
			t.Fatalf("DeleteJourneyDraft failed: %v", err)
		}
	})

	// 3. Journey Version
	t.Run("JourneyVersion CRUD", func(t *testing.T) {
		jv := &postgres.JourneyVersion{
			TenantID:    testTenant,
			VersionID:   "jv-001",
			DraftID:     "draft-100",
			Version:     1,
			EntryNodeID: "node-start",
			Nodes:       []byte(`[{"id":"node-start"}]`),
			Edges:       []byte(`[]`),
			ContentHash: "hash-jv-1",
		}
		created, err := repo.CreateJourneyVersion(ctx, jv)
		if err != nil || created.VersionID != "jv-001" {
			t.Fatalf("CreateJourneyVersion failed: %v", err)
		}

		fetched, err := repo.GetJourneyVersion(ctx, testTenant, "jv-001")
		if err != nil || fetched.EntryNodeID != "node-start" {
			t.Fatalf("GetJourneyVersion failed: %v", err)
		}

		byDraft, err := repo.GetJourneyVersionByDraftAndVersion(ctx, testTenant, "draft-100", 1)
		if err != nil || byDraft.VersionID != "jv-001" {
			t.Fatalf("GetJourneyVersionByDraftAndVersion failed: %v", err)
		}

		list, err := repo.ListJourneyVersionsByDraft(ctx, testTenant, "draft-100")
		if err != nil || len(list) != 1 {
			t.Fatalf("ListJourneyVersionsByDraft failed: %v", err)
		}
	})

	// 4. Idempotency Key
	t.Run("IdempotencyKey Operations", func(t *testing.T) {
		ik := &postgres.IdempotencyKey{
			TenantID:        testTenant,
			Key:             "key-abc-123",
			Scope:           "order_created",
			Status:          "pending",
			ResponsePayload: []byte(`{}`),
			ExpiresAt:       time.Now().Add(1 * time.Hour),
		}
		created, err := repo.CreateIdempotencyKey(ctx, ik)
		if err != nil || created.Key != "key-abc-123" {
			t.Fatalf("CreateIdempotencyKey failed: %v", err)
		}

		fetched, err := repo.GetIdempotencyKey(ctx, testTenant, "order_created", "key-abc-123")
		if err != nil || fetched.Status != "pending" {
			t.Fatalf("GetIdempotencyKey failed: %v", err)
		}

		ik.Status = "completed"
		ik.ResponsePayload = []byte(`{"status":"ok"}`)
		updated, err := repo.UpdateIdempotencyKey(ctx, ik)
		if err != nil || updated.Status != "completed" {
			t.Fatalf("UpdateIdempotencyKey failed: %v", err)
		}
	})

	// 5. Kafka Inbox
	t.Run("KafkaInbox Operations", func(t *testing.T) {
		msg := &postgres.KafkaInbox{
			TenantID:  testTenant,
			MessageID: "msg-001",
			EventID:   "evt-100",
			Topic:     "user-events",
			Partition: 0,
			OffsetVal: 105,
			Payload:   []byte(`{"event":"signup"}`),
			Status:    "received",
		}
		created, err := repo.SaveKafkaInboxMessage(ctx, msg)
		if err != nil || created.MessageID != "msg-001" {
			t.Fatalf("SaveKafkaInboxMessage failed: %v", err)
		}

		fetched, err := repo.GetKafkaInboxMessage(ctx, testTenant, "msg-001")
		if err != nil || fetched.Topic != "user-events" {
			t.Fatalf("GetKafkaInboxMessage failed: %v", err)
		}

		marked, err := repo.MarkKafkaInboxProcessed(ctx, testTenant, "msg-001", time.Now().UTC())
		if err != nil || marked.Status != "processed" {
			t.Fatalf("MarkKafkaInboxProcessed failed: %v", err)
		}
	})

	// 6. Target Manifests
	t.Run("TargetManifest Operations", func(t *testing.T) {
		tm := &postgres.TargetManifest{
			TenantID:    testTenant,
			ManifestID:  "man-001",
			Name:        "Active Customers Q3",
			QuerySpec:   []byte(`{"status":"active"}`),
			TotalCount:  5000,
			ContentHash: "hash-man-1",
		}
		created, err := repo.CreateTargetManifest(ctx, tm)
		if err != nil || created.ManifestID != "man-001" {
			t.Fatalf("CreateTargetManifest failed: %v", err)
		}

		fetched, err := repo.GetTargetManifest(ctx, testTenant, "man-001")
		if err != nil || fetched.TotalCount != 5000 {
			t.Fatalf("GetTargetManifest failed: %v", err)
		}
	})

	// 7. Dispatch Ledger
	t.Run("DispatchLedger Operations", func(t *testing.T) {
		dl := &postgres.DispatchLedger{
			TenantID:         testTenant,
			DispatchID:       "disp-001",
			ManifestID:       "man-001",
			JourneyVersionID: "jv-001",
			SubjectID:        "subj-99",
			Status:           "dispatched",
		}
		created, err := repo.CreateDispatchLedger(ctx, dl)
		if err != nil || created.DispatchID != "disp-001" {
			t.Fatalf("CreateDispatchLedger failed: %v", err)
		}

		fetched, err := repo.GetDispatchLedger(ctx, testTenant, "disp-001")
		if err != nil || fetched.SubjectID != "subj-99" {
			t.Fatalf("GetDispatchLedger failed: %v", err)
		}
	})

	// 8. Enrollments
	t.Run("Enrollments Operations", func(t *testing.T) {
		e := &postgres.Enrollment{
			TenantID:         testTenant,
			EnrollmentID:     "enr-001",
			JourneyVersionID: "jv-001",
			SubjectID:        "user-100",
			Status:           "active",
			CurrentNodeID:    "node-1",
			StateData:        []byte(`{"step":1}`),
		}
		created, err := repo.CreateEnrollment(ctx, e)
		if err != nil || created.EnrollmentID != "enr-001" {
			t.Fatalf("CreateEnrollment failed: %v", err)
		}

		fetched, err := repo.GetEnrollment(ctx, testTenant, "enr-001")
		if err != nil || fetched.Status != "active" {
			t.Fatalf("GetEnrollment failed: %v", err)
		}

		now := time.Now().UTC()
		updated, err := repo.UpdateEnrollmentStatus(ctx, testTenant, "enr-001", "completed", "node-end", []byte(`{"step":2}`), &now)
		if err != nil || updated.Status != "completed" {
			t.Fatalf("UpdateEnrollmentStatus failed: %v", err)
		}
	})

	// 9. Subscriptions
	t.Run("Subscriptions Operations", func(t *testing.T) {
		s := &postgres.Subscription{
			TenantID:       testTenant,
			SubscriptionID: "sub-001",
			EnrollmentID:   "enr-001",
			EventType:      "order_completed",
			ConditionExpr:  "amount > 100",
			Status:         "active",
		}
		created, err := repo.CreateSubscription(ctx, s)
		if err != nil || created.SubscriptionID != "sub-001" {
			t.Fatalf("CreateSubscription failed: %v", err)
		}

		list, err := repo.ListSubscriptionsByEnrollment(ctx, testTenant, "enr-001")
		if err != nil || len(list) != 1 {
			t.Fatalf("ListSubscriptionsByEnrollment failed: %v", err)
		}
	})

	// 10. Action Ledger
	t.Run("ActionLedger Operations", func(t *testing.T) {
		al := &postgres.ActionLedger{
			TenantID:            testTenant,
			ActionID:            "act-001",
			EnrollmentID:        "enr-001",
			NodeID:              "node-email",
			ActivityType:        "send_email",
			Status:              "pending",
			Output:              []byte(`{}`),
			ExecutionDurationMS: 0,
		}
		created, err := repo.CreateActionLedger(ctx, al)
		if err != nil || created.ActionID != "act-001" {
			t.Fatalf("CreateActionLedger failed: %v", err)
		}

		fetched, err := repo.GetActionLedger(ctx, testTenant, "act-001")
		if err != nil || fetched.ActivityType != "send_email" {
			t.Fatalf("GetActionLedger failed: %v", err)
		}

		now := time.Now().UTC()
		updated, err := repo.UpdateActionLedgerStatus(ctx, testTenant, "act-001", "success", []byte(`{"sent":true}`), "", 120, &now)
		if err != nil || updated.Status != "success" {
			t.Fatalf("UpdateActionLedgerStatus failed: %v", err)
		}
	})

	// 11. Experiment Definitions
	t.Run("ExperimentDefinitions Operations", func(t *testing.T) {
		exp := &postgres.ExperimentDefinition{
			TenantID:       testTenant,
			ExperimentID:   "exp-001",
			Name:           "Discount Test A/B",
			Description:    "10% vs 20% discount",
			Status:         "active",
			Variants:       []byte(`[{"id":"A"},{"id":"B"}]`),
			TargetAudience: "all",
			ContentHash:    "hash-exp-1",
		}
		created, err := repo.CreateExperimentDefinition(ctx, exp)
		if err != nil || created.ExperimentID != "exp-001" {
			t.Fatalf("CreateExperimentDefinition failed: %v", err)
		}

		fetched, err := repo.GetExperimentDefinition(ctx, testTenant, "exp-001")
		if err != nil || fetched.Name != "Discount Test A/B" {
			t.Fatalf("GetExperimentDefinition failed: %v", err)
		}

		exp.Status = "completed"
		updated, err := repo.UpdateExperimentDefinition(ctx, exp)
		if err != nil || updated.Status != "completed" {
			t.Fatalf("UpdateExperimentDefinition failed: %v", err)
		}
	})

	// 12. Assignments
	t.Run("Assignments Operations", func(t *testing.T) {
		a := &postgres.Assignment{
			TenantID:          testTenant,
			AssignmentID:      "asgn-001",
			ExperimentID:      "exp-001",
			SubjectID:         "user-200",
			VariantID:         "variant-A",
			WeightBasisPoints: 5000,
		}
		created, err := repo.CreateAssignment(ctx, a)
		if err != nil || created.AssignmentID != "asgn-001" {
			t.Fatalf("CreateAssignment failed: %v", err)
		}

		fetched, err := repo.GetAssignment(ctx, testTenant, "asgn-001")
		if err != nil || fetched.VariantID != "variant-A" {
			t.Fatalf("GetAssignment failed: %v", err)
		}

		byExpSubj, err := repo.GetAssignmentByExperimentSubject(ctx, testTenant, "exp-001", "user-200")
		if err != nil || byExpSubj.AssignmentID != "asgn-001" {
			t.Fatalf("GetAssignmentByExperimentSubject failed: %v", err)
		}
	})

	// 13. Exposures
	t.Run("Exposures Operations", func(t *testing.T) {
		ex := &postgres.Exposure{
			TenantID:          testTenant,
			ExposureID:        "expo-001",
			ExperimentID:      "exp-001",
			AssignmentID:      "asgn-001",
			SubjectID:         "user-200",
			VariantID:         "variant-A",
			WeightBasisPoints: 5000,
			Context:           []byte(`{"device":"mobile"}`),
		}
		created, err := repo.CreateExposure(ctx, ex)
		if err != nil || created.ExposureID != "expo-001" {
			t.Fatalf("CreateExposure failed: %v", err)
		}

		fetched, err := repo.GetExposure(ctx, testTenant, "expo-001")
		if err != nil || fetched.SubjectID != "user-200" {
			t.Fatalf("GetExposure failed: %v", err)
		}
	})

	// 14. Outbox
	t.Run("Outbox Operations", func(t *testing.T) {
		o := &postgres.Outbox{
			TenantID:      testTenant,
			ID:            "out-001",
			AggregateType: "enrollment",
			AggregateID:   "enr-001",
			EventType:     "EnrollmentCompleted",
			Payload:       []byte(`{"enrolled":true}`),
			Headers:       []byte(`{}`),
			Status:        "pending",
		}
		created, err := repo.CreateOutboxEvent(ctx, o)
		if err != nil || created.ID != "out-001" {
			t.Fatalf("CreateOutboxEvent failed: %v", err)
		}

		fetched, err := repo.GetOutboxEvent(ctx, testTenant, "out-001")
		if err != nil || fetched.EventType != "EnrollmentCompleted" {
			t.Fatalf("GetOutboxEvent failed: %v", err)
		}

		pending, err := repo.ListPendingOutboxEvents(ctx, testTenant, 10)
		if err != nil || len(pending) != 1 {
			t.Fatalf("ListPendingOutboxEvents failed: %v", err)
		}

		marked, err := repo.MarkOutboxProcessed(ctx, testTenant, "out-001", time.Now().UTC())
		if err != nil || marked.Status != "processed" {
			t.Fatalf("MarkOutboxProcessed failed: %v", err)
		}
	})

	// 15. Static Lists
	t.Run("StaticLists Operations", func(t *testing.T) {
		sl := &postgres.StaticList{
			TenantID:           testTenant,
			ListID:             "list-001",
			Name:               "VIP Users",
			Description:        "High LTV accounts",
			ItemCount:          2,
			DataClassification: "NonPII",
			Items:              []byte(`["u1","u2"]`),
			ContentHash:        "hash-list-1",
		}
		created, err := repo.CreateStaticList(ctx, sl)
		if err != nil || created.ListID != "list-001" {
			t.Fatalf("CreateStaticList failed: %v", err)
		}

		fetched, err := repo.GetStaticList(ctx, testTenant, "list-001")
		if err != nil || fetched.Name != "VIP Users" {
			t.Fatalf("GetStaticList failed: %v", err)
		}

		sl.Name = "VIP Users Updated"
		updated, err := repo.UpdateStaticList(ctx, sl)
		if err != nil || updated.Name != "VIP Users Updated" {
			t.Fatalf("UpdateStaticList failed: %v", err)
		}

		err = repo.DeleteStaticList(ctx, testTenant, "list-001")
		if err != nil {
			t.Fatalf("DeleteStaticList failed: %v", err)
		}
	})

	// 16. Test Runs
	t.Run("TestRuns Operations", func(t *testing.T) {
		tr := &postgres.TestRun{
			TenantID:         testTenant,
			TestRunID:        "tr-001",
			DraftID:          "draft-001",
			IRID:             "ir-001",
			Status:           "pending",
			MockInputs:       []byte(`{}`),
			ExpectedOutcomes: []byte(`{}`),
			ActualOutcomes:   []byte(`{}`),
		}
		created, err := repo.CreateTestRun(ctx, tr)
		if err != nil || created.TestRunID != "tr-001" {
			t.Fatalf("CreateTestRun failed: %v", err)
		}

		fetched, err := repo.GetTestRun(ctx, testTenant, "tr-001")
		if err != nil || fetched.Status != "pending" {
			t.Fatalf("GetTestRun failed: %v", err)
		}

		updated, err := repo.UpdateTestRun(ctx, testTenant, "tr-001", "passed", []byte(`{"passed":true}`), 350)
		if err != nil || updated.Status != "passed" {
			t.Fatalf("UpdateTestRun failed: %v", err)
		}
	})

	// 17. Lifecycle Events
	t.Run("LifecycleEvents Operations", func(t *testing.T) {
		le := &postgres.LifecycleEvent{
			TenantID:   testTenant,
			EventID:    "le-001",
			EntityType: "journey_draft",
			EntityID:   "draft-001",
			EventName:  "DraftPublished",
			Payload:    []byte(`{"published_by":"admin"}`),
		}
		created, err := repo.RecordLifecycleEvent(ctx, le)
		if err != nil || created.EventID != "le-001" {
			t.Fatalf("RecordLifecycleEvent failed: %v", err)
		}

		list, err := repo.ListLifecycleEventsByEntity(ctx, testTenant, "journey_draft", "draft-001")
		if err != nil || len(list) != 1 {
			t.Fatalf("ListLifecycleEventsByEntity failed: %v", err)
		}
	})

	// 18. Tombstones
	t.Run("Tombstones Operations", func(t *testing.T) {
		tb := &postgres.Tombstone{
			TenantID:    testTenant,
			TombstoneID: "tomb-001",
			EntityType:  "catalog",
			EntityID:    "cat-999",
			DeletedBy:   "user-admin",
			Reason:      "Deprecated asset cleanup",
		}
		created, err := repo.CreateTombstone(ctx, tb)
		if err != nil || created.TombstoneID != "tomb-001" {
			t.Fatalf("CreateTombstone failed: %v", err)
		}

		fetched, err := repo.GetTombstone(ctx, testTenant, "tomb-001")
		if err != nil || fetched.Reason != "Deprecated asset cleanup" {
			t.Fatalf("GetTombstone failed: %v", err)
		}

		byEntity, err := repo.GetTombstoneByEntity(ctx, testTenant, "catalog", "cat-999")
		if err != nil || byEntity.TombstoneID != "tomb-001" {
			t.Fatalf("GetTombstoneByEntity failed: %v", err)
		}
	})
}

func TestRepository_NotFoundErrors(t *testing.T) {
	ctx := context.Background()
	repo := postgres.NewMemoryRepository()

	_, err := repo.GetCatalog(ctx, testTenant, "nonexistent")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("expected ErrNotFound for GetCatalog, got %v", err)
	}

	_, err = repo.GetJourneyDraft(ctx, testTenant, "nonexistent")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("expected ErrNotFound for GetJourneyDraft, got %v", err)
	}

	_, err = repo.GetJourneyVersion(ctx, testTenant, "nonexistent")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("expected ErrNotFound for GetJourneyVersion, got %v", err)
	}

	_, err = repo.GetIdempotencyKey(ctx, testTenant, "scope", "nonexistent")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("expected ErrNotFound for GetIdempotencyKey, got %v", err)
	}

	_, err = repo.GetKafkaInboxMessage(ctx, testTenant, "nonexistent")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("expected ErrNotFound for GetKafkaInboxMessage, got %v", err)
	}

	_, err = repo.GetEnrollment(ctx, testTenant, "nonexistent")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("expected ErrNotFound for GetEnrollment, got %v", err)
	}

	_, err = repo.GetTombstone(ctx, testTenant, "nonexistent")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Errorf("expected ErrNotFound for GetTombstone, got %v", err)
	}
}

func TestRepository_UniqueConstraintConflicts(t *testing.T) {
	ctx := context.Background()
	repo := postgres.NewMemoryRepository()

	// Primary Key Conflict
	cat := &postgres.Catalog{
		TenantID:      testTenant,
		RecordID:      "cat-dup",
		Name:          "Cat Dup",
		ComponentType: "action",
		Version:       "1.0",
	}
	_, err := repo.CreateCatalog(ctx, cat)
	if err != nil {
		t.Fatalf("first CreateCatalog failed: %v", err)
	}
	_, err = repo.CreateCatalog(ctx, cat)
	if !errors.Is(err, postgres.ErrAlreadyExists) {
		t.Errorf("expected ErrAlreadyExists for duplicate catalog PK, got %v", err)
	}

	// Idempotency Key Conflict
	ik := &postgres.IdempotencyKey{
		TenantID:  testTenant,
		Key:       "dup-key",
		Scope:     "payments",
		Status:    "pending",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	_, err = repo.CreateIdempotencyKey(ctx, ik)
	if err != nil {
		t.Fatalf("first CreateIdempotencyKey failed: %v", err)
	}
	_, err = repo.CreateIdempotencyKey(ctx, ik)
	if !errors.Is(err, postgres.ErrConflict) {
		t.Errorf("expected ErrConflict for duplicate idempotency key, got %v", err)
	}

	// Journey Version Draft/Version Unique Boundary Conflict
	jv1 := &postgres.JourneyVersion{
		TenantID:  testTenant,
		VersionID: "jv-unique-1",
		DraftID:   "draft-55",
		Version:   1,
	}
	jv2 := &postgres.JourneyVersion{
		TenantID:  testTenant,
		VersionID: "jv-unique-2",
		DraftID:   "draft-55",
		Version:   1, // Same draft & version
	}
	_, err = repo.CreateJourneyVersion(ctx, jv1)
	if err != nil {
		t.Fatalf("first CreateJourneyVersion failed: %v", err)
	}
	_, err = repo.CreateJourneyVersion(ctx, jv2)
	if !errors.Is(err, postgres.ErrConflict) {
		t.Errorf("expected ErrConflict for duplicate journey draft version, got %v", err)
	}

	// Kafka Inbox Topic/Partition/Offset Boundary Conflict
	m1 := &postgres.KafkaInbox{
		TenantID:  testTenant,
		MessageID: "msg-111",
		EventID:   "evt-1",
		Topic:     "orders",
		Partition: 1,
		OffsetVal: 500,
	}
	m2 := &postgres.KafkaInbox{
		TenantID:  testTenant,
		MessageID: "msg-222",
		EventID:   "evt-2",
		Topic:     "orders",
		Partition: 1,
		OffsetVal: 500, // Same topic, partition, offset
	}
	_, err = repo.SaveKafkaInboxMessage(ctx, m1)
	if err != nil {
		t.Fatalf("first SaveKafkaInboxMessage failed: %v", err)
	}
	_, err = repo.SaveKafkaInboxMessage(ctx, m2)
	if !errors.Is(err, postgres.ErrConflict) {
		t.Errorf("expected ErrConflict for duplicate kafka inbox offset, got %v", err)
	}
}

func TestRepository_TransactionCommitAndRollback(t *testing.T) {
	ctx := context.Background()
	repo := postgres.NewMemoryRepository()

	// 1. Transaction Commit
	err := repo.WithTx(ctx, func(txRepo postgres.Repository) error {
		_, err := txRepo.CreateCatalog(ctx, &postgres.Catalog{
			TenantID:      testTenant,
			RecordID:      "tx-cat-1",
			Name:          "TX Cat 1",
			ComponentType: "condition",
			Version:       "1.0",
		})
		if err != nil {
			return err
		}
		_, err = txRepo.CreateJourneyDraft(ctx, &postgres.JourneyDraft{
			TenantID: testTenant,
			DraftID:  "tx-draft-1",
			Name:     "TX Draft 1",
			Version:  1,
		})
		return err
	})
	if err != nil {
		t.Fatalf("WithTx commit failed: %v", err)
	}

	// Verify committed state
	_, err = repo.GetCatalog(ctx, testTenant, "tx-cat-1")
	if err != nil {
		t.Fatalf("GetCatalog after tx commit failed: %v", err)
	}
	_, err = repo.GetJourneyDraft(ctx, testTenant, "tx-draft-1")
	if err != nil {
		t.Fatalf("GetJourneyDraft after tx commit failed: %v", err)
	}

	// 2. Transaction Rollback
	errRollback := errors.New("simulated error forcing rollback")
	err = repo.WithTx(ctx, func(txRepo postgres.Repository) error {
		_, err := txRepo.CreateCatalog(ctx, &postgres.Catalog{
			TenantID:      testTenant,
			RecordID:      "tx-cat-rollback",
			Name:          "TX Cat Rollback",
			ComponentType: "activity",
			Version:       "1.0",
		})
		if err != nil {
			return err
		}
		return errRollback
	})

	if !errors.Is(err, errRollback) {
		t.Fatalf("expected errRollback, got %v", err)
	}

	// Verify rolled back state (should NOT exist in main repo)
	_, err = repo.GetCatalog(ctx, testTenant, "tx-cat-rollback")
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for rolled back item, got %v", err)
	}
}

func TestRepository_ConcurrentInsertion(t *testing.T) {
	ctx := context.Background()
	repo := postgres.NewMemoryRepository()

	numWorkers := 20
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(id int) {
			defer wg.Done()
			recordID := fmt.Sprintf("concurrent-cat-%d", id)
			cat := &postgres.Catalog{
				TenantID:      testTenant,
				RecordID:      recordID,
				Name:          fmt.Sprintf("Concurrent Cat %d", id),
				ComponentType: "action",
				Version:       "1.0",
			}
			_, err := repo.CreateCatalog(ctx, cat)
			if err != nil {
				t.Errorf("worker %d CreateCatalog failed: %v", id, err)
			}

			fetched, err := repo.GetCatalog(ctx, testTenant, recordID)
			if err != nil || fetched.RecordID != recordID {
				t.Errorf("worker %d GetCatalog failed: %v", id, err)
			}
		}(i)
	}

	wg.Wait()

	list, err := repo.ListCatalogs(ctx, testTenant)
	if err != nil {
		t.Fatalf("ListCatalogs failed: %v", err)
	}
	if len(list) != numWorkers {
		t.Fatalf("expected %d catalogs after concurrent insertion, got %d", numWorkers, len(list))
	}
}
