package ingress

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/ingress/codec"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
)

type MockCommitter struct {
	mu        sync.Mutex
	committed map[string]map[int32]int64
	failCount int
	callCount int
}

func NewMockCommitter() *MockCommitter {
	return &MockCommitter{
		committed: make(map[string]map[int32]int64),
	}
}

func (m *MockCommitter) CommitOffset(ctx context.Context, topic string, partition int32, offset int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++
	if m.failCount > 0 {
		m.failCount--
		return errors.New("simulated offset commit failure")
	}

	if m.committed[topic] == nil {
		m.committed[topic] = make(map[int32]int64)
	}
	m.committed[topic][partition] = offset
	return nil
}

func (m *MockCommitter) GetCommitted(topic string, partition int32) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pMap, ok := m.committed[topic]; ok {
		if off, found := pMap[partition]; found {
			return off
		}
	}
	return -1
}

func setupTestJourneyVersion(t *testing.T, repo postgres.Repository, tenantID, draftID, versionID string) {
	ctx := context.Background()
	jv := &postgres.JourneyVersion{
		TenantID:    tenantID,
		VersionID:   versionID,
		DraftID:     draftID,
		Version:     1,
		EntryNodeID: "node-start",
		Nodes:       []byte(`[{"id":"node-start","type":"trigger"}]`),
		Edges:       []byte(`[]`),
		ContentHash: "hash-jv-1",
		CompiledAt:  time.Now().UTC(),
		CreatedAt:   time.Now().UTC(),
	}

	_, err := repo.CreateJourneyVersion(ctx, jv)
	if err != nil && !errors.Is(err, postgres.ErrAlreadyExists) && !errors.Is(err, postgres.ErrConflict) {
		t.Fatalf("failed setting up journey version: %v", err)
	}
}

func createTestEventJSON(tenantID, eventID, subjectRef string) []byte {
	env := codec.EventEnvelope{
		SchemaVersion: "1.0",
		EventID:       eventID,
		TenantID:      tenantID,
		SubjectRef:    subjectRef,
		OccurredAt:    time.Now().UTC(),
		EventType:     "user.signup.v1",
		CorrelationID: "corr-100",
		Payload: map[string]interface{}{
			"draft_id": "draft-default",
			"tier":     "pro",
		},
	}
	data, _ := json.Marshal(env)
	return data
}

func TestCapture_DuplicateEventDelivery(t *testing.T) {
	repo := postgres.NewMemoryRepository()
	committer := NewMockCommitter()
	consumer := NewCaptureConsumer(repo, committer)
	ctx := context.Background()

	tenantID := "tenant-dup-test"
	setupTestJourneyVersion(t, repo, tenantID, "draft-default", "jv-dup-100")

	msgBytes := createTestEventJSON(tenantID, "evt-dup-1", "user-100")
	kafkaMsg := KafkaMessage{
		Topic:     "events.ingress.v1",
		Partition: 0,
		Offset:    10,
		Key:       "user-100",
		Value:     msgBytes,
		Timestamp: time.Now().UTC(),
	}

	// 1. Initial event ingestion
	res1, err := consumer.ProcessMessage(ctx, kafkaMsg)
	if err != nil {
		t.Fatalf("initial ProcessMessage failed: %v", err)
	}
	if res1.IsDuplicate {
		t.Errorf("expected IsDuplicate to be false on first delivery")
	}
	if res1.Outcome != OutcomeProcessed {
		t.Errorf("expected outcome %s, got %s", OutcomeProcessed, res1.Outcome)
	}
	if res1.TargetManifest == nil {
		t.Fatalf("expected target manifest to be created on first delivery")
	}
	if res1.TargetManifest.TotalCount != 1 {
		t.Errorf("expected target count 1, got %d", res1.TargetManifest.TotalCount)
	}
	if !res1.OffsetCommitted {
		t.Errorf("expected offset commit to succeed")
	}

	// Verify DB state after first ingestion
	inbox1, err := repo.GetKafkaInboxMessage(ctx, tenantID, res1.Inbox.MessageID)
	if err != nil || inbox1 == nil {
		t.Fatalf("expected inbox message in DB: %v", err)
	}

	manifest1, err := repo.GetTargetManifest(ctx, tenantID, res1.TargetManifest.ManifestID)
	if err != nil || manifest1 == nil {
		t.Fatalf("expected target manifest in DB: %v", err)
	}

	// 2. Duplicate event delivery (same Kafka message re-delivered)
	res2, err := consumer.ProcessMessage(ctx, kafkaMsg)
	if err != nil {
		t.Fatalf("duplicate ProcessMessage failed: %v", err)
	}
	if !res2.IsDuplicate {
		t.Errorf("expected IsDuplicate to be true on redelivery")
	}
	if res2.Outcome != OutcomeProcessed {
		t.Errorf("expected outcome %s for duplicate, got %s", OutcomeProcessed, res2.Outcome)
	}
	if res2.Inbox.MessageID != res1.Inbox.MessageID {
		t.Errorf("expected duplicate result to reference existing inbox ID %s, got %s", res1.Inbox.MessageID, res2.Inbox.MessageID)
	}
	if res2.TargetManifest.ManifestID != res1.TargetManifest.ManifestID {
		t.Errorf("expected duplicate result to reference existing manifest ID %s, got %s", res1.TargetManifest.ManifestID, res2.TargetManifest.ManifestID)
	}

	// Verify watermark
	wm := consumer.Watermark("events.ingress.v1", 0)
	if wm != 10 {
		t.Errorf("expected watermark 10, got %d", wm)
	}
}

func TestCapture_CrashPointRecovery(t *testing.T) {
	ctx := context.Background()

	t.Run("crash_before_transaction_commit", func(t *testing.T) {
		repo := postgres.NewMemoryRepository()
		committer := NewMockCommitter()
		consumer := NewCaptureConsumer(repo, committer)

		tenantID := "tenant-crash-1"
		setupTestJourneyVersion(t, repo, tenantID, "draft-default", "jv-crash-1")

		msgBytes := createTestEventJSON(tenantID, "evt-crash-before", "user-crash-1")
		kafkaMsg := KafkaMessage{
			Topic:     "events.ingress.v1",
			Partition: 1,
			Offset:    20,
			Key:       "user-crash-1",
			Value:     msgBytes,
		}

		// Inject simulated TX failure inside repo.WithTx by cancelling context before call
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel()

		_, err := consumer.ProcessMessage(cancelledCtx, kafkaMsg)
		if err == nil {
			t.Fatalf("expected error when context is cancelled before tx commit")
		}

		// Verify no offset committed
		if committer.GetCommitted("events.ingress.v1", 1) != -1 {
			t.Errorf("expected offset commit to NOT be called on TX crash")
		}

		// Verify no inbox saved in DB
		messageID := fmt.Sprintf("msg-%s-evt-crash-before", tenantID)
		_, err = repo.GetKafkaInboxMessage(ctx, tenantID, messageID)
		if err == nil {
			t.Errorf("expected inbox message NOT to exist after TX rollback")
		}
	})

	t.Run("crash_after_transaction_commit_before_offset_commit", func(t *testing.T) {
		repo := postgres.NewMemoryRepository()
		committer := NewMockCommitter()
		committer.failCount = 1 // Fail the first offset commit attempt

		consumer := NewCaptureConsumer(repo, committer)

		tenantID := "tenant-crash-2"
		setupTestJourneyVersion(t, repo, tenantID, "draft-default", "jv-crash-2")

		msgBytes := createTestEventJSON(tenantID, "evt-crash-after", "user-crash-2")
		kafkaMsg := KafkaMessage{
			Topic:     "events.ingress.v1",
			Partition: 2,
			Offset:    30,
			Key:       "user-crash-2",
			Value:     msgBytes,
		}

		// 1. First attempt: DB transaction commits, but offset commit fails!
		res1, err := consumer.ProcessMessage(ctx, kafkaMsg)
		if err == nil {
			t.Fatalf("expected error due to offset commit failure")
		}
		if res1 == nil || !strings.HasPrefix(res1.Inbox.MessageID, "msg-") {
			t.Fatalf("expected DB tx to have committed before offset commit failed")
		}

		// Verify DB contains the committed inbox & manifest
		messageID := fmt.Sprintf("msg-%s-evt-crash-after", tenantID)
		inbox, err := repo.GetKafkaInboxMessage(ctx, tenantID, messageID)
		if err != nil || inbox == nil {
			t.Fatalf("expected DB inbox row to exist: %v", err)
		}

		// 2. Recovery: Kafka re-delivers the message upon consumer restart
		res2, err := consumer.ProcessMessage(ctx, kafkaMsg)
		if err != nil {
			t.Fatalf("expected recovery reprocessing to succeed: %v", err)
		}
		if !res2.IsDuplicate {
			t.Errorf("expected duplicate detection on redelivery after crash recovery")
		}
		if !res2.OffsetCommitted {
			t.Errorf("expected offset commit to succeed on recovery redelivery")
		}

		// Verify committed offset
		if committer.GetCommitted("events.ingress.v1", 2) != 30 {
			t.Errorf("expected committed offset 30, got %d", committer.GetCommitted("events.ingress.v1", 2))
		}
	})
}

func TestCapture_NoTargetAndTombstonedSubject(t *testing.T) {
	ctx := context.Background()

	t.Run("no_target_handling", func(t *testing.T) {
		repo := postgres.NewMemoryRepository()
		committer := NewMockCommitter()
		consumer := NewCaptureConsumer(repo, committer)

		tenantID := "tenant-no-target-spec"
		// NOTE: Intentionally do NOT setup journey version for tenant-no-target-spec

		msgBytes := createTestEventJSON(tenantID, "evt-no-target-1", "user-200")
		kafkaMsg := KafkaMessage{
			Topic:     "events.ingress.v1",
			Partition: 0,
			Offset:    40,
			Key:       "user-200",
			Value:     msgBytes,
		}

		res, err := consumer.ProcessMessage(ctx, kafkaMsg)
		if err != nil {
			t.Fatalf("ProcessMessage failed: %v", err)
		}
		if res.Outcome != OutcomeNoTarget {
			t.Errorf("expected outcome %s, got %s", OutcomeNoTarget, res.Outcome)
		}
		if res.TargetManifest != nil {
			t.Errorf("expected nil target manifest for no-target outcome")
		}
		if res.Inbox.Status != OutcomeNoTarget {
			t.Errorf("expected inbox status %s, got %s", OutcomeNoTarget, res.Inbox.Status)
		}
	})

	t.Run("tombstoned_subject_handling", func(t *testing.T) {
		repo := postgres.NewMemoryRepository()
		committer := NewMockCommitter()
		consumer := NewCaptureConsumer(repo, committer)

		tenantID := "tenant-tombstone-spec"
		subjectRef := "user-tombstoned-999"
		setupTestJourneyVersion(t, repo, tenantID, "draft-default", "jv-tomb-1")

		// Create tombstone for subject
		tomb := &postgres.Tombstone{
			TenantID:    tenantID,
			TombstoneID: "tomb-999",
			EntityType:  "subject",
			EntityID:    subjectRef,
			Reason:      "GDPR deletion request",
			DeletedAt:   time.Now().UTC(),
		}
		if _, err := repo.CreateTombstone(ctx, tomb); err != nil {
			t.Fatalf("failed creating tombstone: %v", err)
		}

		msgBytes := createTestEventJSON(tenantID, "evt-tombstone-1", subjectRef)
		kafkaMsg := KafkaMessage{
			Topic:     "events.ingress.v1",
			Partition: 0,
			Offset:    50,
			Key:       subjectRef,
			Value:     msgBytes,
		}

		res, err := consumer.ProcessMessage(ctx, kafkaMsg)
		if err != nil {
			t.Fatalf("ProcessMessage failed: %v", err)
		}
		if res.Outcome != OutcomeTombstonedSubject {
			t.Errorf("expected outcome %s, got %s", OutcomeTombstonedSubject, res.Outcome)
		}
		if res.TargetManifest != nil {
			t.Errorf("expected nil target manifest for tombstoned-subject outcome")
		}
		if res.Inbox.Status != OutcomeTombstonedSubject {
			t.Errorf("expected inbox status %s, got %s", OutcomeTombstonedSubject, res.Inbox.Status)
		}
	})
}

func TestCapture_InvalidSchemaAndQuarantined(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid_schema_handling", func(t *testing.T) {
		repo := postgres.NewMemoryRepository()
		committer := NewMockCommitter()
		consumer := NewCaptureConsumer(repo, committer)

		kafkaMsg := KafkaMessage{
			Topic:     "events.ingress.v1",
			Partition: 0,
			Offset:    60,
			Key:       "key-malformed",
			Value:     []byte(`{ invalid json payload `),
		}

		res, err := consumer.ProcessMessage(ctx, kafkaMsg)
		if err != nil {
			t.Fatalf("ProcessMessage failed: %v", err)
		}
		if res.Outcome != OutcomeInvalidSchema {
			t.Errorf("expected outcome %s, got %s", OutcomeInvalidSchema, res.Outcome)
		}
		if res.Inbox.Status != OutcomeInvalidSchema {
			t.Errorf("expected inbox status %s, got %s", OutcomeInvalidSchema, res.Inbox.Status)
		}
		if res.TargetManifest != nil {
			t.Errorf("expected nil target manifest")
		}
	})

	t.Run("quarantined_handling", func(t *testing.T) {
		repo := postgres.NewMemoryRepository()
		committer := NewMockCommitter()
		consumer := NewCaptureConsumer(repo, committer)

		poisonPayload := map[string]interface{}{
			"schema_version": "1.0",
			"event_id":       "evt-poison-pill",
			"tenant_id":      "tenant-poison",
			"subject_ref":    "sub-poison",
			"occurred_at":    time.Now().Format(time.RFC3339),
			"event_type":     "poison.pill.v1",
			"payload": map[string]interface{}{
				"poison_pill": true,
			},
		}
		data, _ := json.Marshal(poisonPayload)

		kafkaMsg := KafkaMessage{
			Topic:     "events.ingress.v1",
			Partition: 0,
			Offset:    70,
			Key:       "sub-poison",
			Value:     data,
		}

		res, err := consumer.ProcessMessage(ctx, kafkaMsg)
		if err != nil {
			t.Fatalf("ProcessMessage failed: %v", err)
		}
		if res.Outcome != OutcomeQuarantined {
			t.Errorf("expected outcome %s, got %s", OutcomeQuarantined, res.Outcome)
		}
		if res.Inbox.Status != OutcomeQuarantined {
			t.Errorf("expected inbox status %s, got %s", OutcomeQuarantined, res.Inbox.Status)
		}
	})
}

func TestCapture_BatchingCancellationAndBackpressure(t *testing.T) {
	repo := postgres.NewMemoryRepository()
	committer := NewMockCommitter()
	cfg := CaptureConfig{
		BatchSize:    10,
		BatchTimeout: 50 * time.Millisecond,
		MaxInFlight:  2,
	}

	consumer := NewCaptureConsumer(repo, committer, WithConfig(cfg))
	tenantID := "tenant-batch"
	setupTestJourneyVersion(t, repo, tenantID, "draft-default", "jv-batch-1")

	t.Run("batch_processing", func(t *testing.T) {
		ctx := context.Background()
		var batch []KafkaMessage

		for i := 0; i < 5; i++ {
			evtID := fmt.Sprintf("evt-batch-%d", i)
			data := createTestEventJSON(tenantID, evtID, fmt.Sprintf("user-%d", i))
			batch = append(batch, KafkaMessage{
				Topic:     "events.ingress.v1",
				Partition: 0,
				Offset:    int64(100 + i),
				Key:       fmt.Sprintf("user-%d", i),
				Value:     data,
			})
		}

		results, err := consumer.ProcessBatch(ctx, batch)
		if err != nil {
			t.Fatalf("ProcessBatch failed: %v", err)
		}
		if len(results) != 5 {
			t.Fatalf("expected 5 results, got %d", len(results))
		}
		for i, res := range results {
			if res.Outcome != OutcomeProcessed {
				t.Errorf("batch item %d outcome expected %s, got %s", i, OutcomeProcessed, res.Outcome)
			}
		}
	})

	t.Run("cancellation_context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		msg := KafkaMessage{
			Topic:     "events.ingress.v1",
			Partition: 0,
			Offset:    200,
			Value:     createTestEventJSON(tenantID, "evt-cancel", "user-cancel"),
		}

		_, err := consumer.ProcessMessage(ctx, msg)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled error, got %v", err)
		}
	})
}
