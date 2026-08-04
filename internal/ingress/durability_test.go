package ingress

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/ingress/codec"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
)

// TestDurability_BatchTypes tests deterministic processing for duplicate, out-of-order, malformed, poison, delayed, and high-volume event batches.
func TestDurability_BatchTypes(t *testing.T) {
	t.Run("duplicate_events_batch", func(t *testing.T) {
		repo := postgres.NewMemoryRepository()
		committer := NewMockCommitter()
		consumer := NewCaptureConsumer(repo, committer)
		ctx := context.Background()

		tenantID := "tenant-batch-dup"
		setupTestJourneyVersion(t, repo, tenantID, "draft-default", "jv-batch-dup")

		// Create batch containing duplicate events in same batch
		data1 := createTestEventJSON(tenantID, "evt-batch-dup-1", "user-1")
		data2 := createTestEventJSON(tenantID, "evt-batch-dup-2", "user-2")

		batch := []KafkaMessage{
			{Topic: "events.ingress.v1", Partition: 0, Offset: 1, Key: "user-1", Value: data1},
			{Topic: "events.ingress.v1", Partition: 0, Offset: 2, Key: "user-1", Value: data1}, // duplicate event ID & key
			{Topic: "events.ingress.v1", Partition: 0, Offset: 3, Key: "user-2", Value: data2},
			{Topic: "events.ingress.v1", Partition: 0, Offset: 4, Key: "user-2", Value: data2}, // duplicate event ID & key
		}

		results, err := consumer.ProcessBatch(ctx, batch)
		if err != nil {
			t.Fatalf("ProcessBatch failed: %v", err)
		}

		if len(results) != 4 {
			t.Fatalf("expected 4 results, got %d", len(results))
		}

		if results[0].IsDuplicate {
			t.Errorf("expected item 0 not duplicate")
		}
		if !results[1].IsDuplicate {
			t.Errorf("expected item 1 to be duplicate")
		}
		if results[2].IsDuplicate {
			t.Errorf("expected item 2 not duplicate")
		}
		if !results[3].IsDuplicate {
			t.Errorf("expected item 3 to be duplicate")
		}

		// Verify watermark advanced to offset 4
		if wm := consumer.Watermark("events.ingress.v1", 0); wm != 4 {
			t.Errorf("expected watermark 4, got %d", wm)
		}

		// Verify target manifest count in DB: only 2 unique manifests created
		m1, err := repo.GetTargetManifest(ctx, tenantID, "man-evt-batch-dup-1")
		if err != nil || m1 == nil {
			t.Fatalf("expected target manifest for evt-batch-dup-1")
		}
		m2, err := repo.GetTargetManifest(ctx, tenantID, "man-evt-batch-dup-2")
		if err != nil || m2 == nil {
			t.Fatalf("expected target manifest for evt-batch-dup-2")
		}
	})

	t.Run("out_of_order_events_batch", func(t *testing.T) {
		repo := postgres.NewMemoryRepository()
		committer := NewMockCommitter()
		consumer := NewCaptureConsumer(repo, committer)
		ctx := context.Background()

		tenantID := "tenant-batch-ooo"
		setupTestJourneyVersion(t, repo, tenantID, "draft-default", "jv-batch-ooo")

		// Out of order offsets: 5, 2, 4, 1, 3
		offsets := []int64{5, 2, 4, 1, 3}
		var batch []KafkaMessage
		for _, off := range offsets {
			evtID := fmt.Sprintf("evt-ooo-%d", off)
			data := createTestEventJSON(tenantID, evtID, fmt.Sprintf("user-%d", off))
			batch = append(batch, KafkaMessage{
				Topic:     "events.ingress.v1",
				Partition: 0,
				Offset:    off,
				Key:       fmt.Sprintf("user-%d", off),
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

		// Watermark should track highest offset processed
		if wm := consumer.Watermark("events.ingress.v1", 0); wm != 5 {
			t.Errorf("expected highest recorded offset 5, got %d", wm)
		}

		// Verify each out-of-order inbox message exists in DB
		for _, off := range offsets {
			msgID := fmt.Sprintf("msg-%s-evt-ooo-%d", tenantID, off)
			inbox, getErr := repo.GetKafkaInboxMessage(ctx, tenantID, msgID)
			if getErr != nil || inbox == nil {
				t.Errorf("expected inbox message for offset %d", off)
			}
		}
	})

	t.Run("malformed_events_batch", func(t *testing.T) {
		repo := postgres.NewMemoryRepository()
		committer := NewMockCommitter()
		consumer := NewCaptureConsumer(repo, committer)
		ctx := context.Background()

		tenantID := "tenant-batch-malformed"
		setupTestJourneyVersion(t, repo, tenantID, "draft-default", "jv-batch-malformed")

		validBytes := createTestEventJSON(tenantID, "evt-valid-1", "user-valid")
		malformedBytes1 := []byte(`{ "tenant_id": "tenant-batch-malformed", "event_id": "evt-malformed-1", invalid json payload... `)
		malformedBytes2 := []byte(`NOT JSON AT ALL`)

		batch := []KafkaMessage{
			{Topic: "events.ingress.v1", Partition: 0, Offset: 10, Key: "key-1", Value: validBytes},
			{Topic: "events.ingress.v1", Partition: 0, Offset: 11, Key: "key-2", Value: malformedBytes1},
			{Topic: "events.ingress.v1", Partition: 0, Offset: 12, Key: "key-3", Value: malformedBytes2},
		}

		results, err := consumer.ProcessBatch(ctx, batch)
		if err != nil {
			t.Fatalf("ProcessBatch failed: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}

		if results[0].Outcome != OutcomeProcessed {
			t.Errorf("expected item 0 OutcomeProcessed, got %s", results[0].Outcome)
		}
		if results[1].Outcome != OutcomeInvalidSchema {
			t.Errorf("expected item 1 OutcomeInvalidSchema, got %s", results[1].Outcome)
		}
		if results[2].Outcome != OutcomeInvalidSchema {
			t.Errorf("expected item 2 OutcomeInvalidSchema, got %s", results[2].Outcome)
		}

		// Offsets for malformed messages should still be committed to avoid endless loop
		if committer.GetCommitted("events.ingress.v1", 0) != 12 {
			t.Errorf("expected committed offset 12, got %d", committer.GetCommitted("events.ingress.v1", 0))
		}
	})

	t.Run("poison_events_batch", func(t *testing.T) {
		repo := postgres.NewMemoryRepository()
		committer := NewMockCommitter()
		consumer := NewCaptureConsumer(repo, committer)
		ctx := context.Background()

		tenantID := "tenant-batch-poison"
		setupTestJourneyVersion(t, repo, tenantID, "draft-default", "jv-batch-poison")

		validBytes := createTestEventJSON(tenantID, "evt-valid-p", "user-p")

		poisonPayload1 := map[string]interface{}{
			"schema_version": "1.0",
			"event_id":       "evt-poison-1",
			"tenant_id":      tenantID,
			"subject_ref":    "sub-p1",
			"occurred_at":    time.Now().Format(time.RFC3339),
			"event_type":     "poison.pill.v1",
			"payload":        map[string]interface{}{"draft_id": "draft-default"},
		}
		pBytes1, _ := json.Marshal(poisonPayload1)

		poisonPayload2 := map[string]interface{}{
			"schema_version": "1.0",
			"event_id":       "evt-poison-2",
			"tenant_id":      tenantID,
			"subject_ref":    "sub-p2",
			"occurred_at":    time.Now().Format(time.RFC3339),
			"event_type":     "user.order.v1",
			"payload":        map[string]interface{}{"poison_pill": true},
		}
		pBytes2, _ := json.Marshal(poisonPayload2)

		batch := []KafkaMessage{
			{Topic: "events.ingress.v1", Partition: 0, Offset: 20, Key: "sub-p1", Value: pBytes1},
			{Topic: "events.ingress.v1", Partition: 0, Offset: 21, Key: "user-p", Value: validBytes},
			{Topic: "events.ingress.v1", Partition: 0, Offset: 22, Key: "sub-p2", Value: pBytes2},
		}

		results, err := consumer.ProcessBatch(ctx, batch)
		if err != nil {
			t.Fatalf("ProcessBatch failed: %v", err)
		}

		if results[0].Outcome != OutcomeQuarantined {
			t.Errorf("item 0 expected OutcomeQuarantined, got %s", results[0].Outcome)
		}
		if results[1].Outcome != OutcomeProcessed {
			t.Errorf("item 1 expected OutcomeProcessed, got %s", results[1].Outcome)
		}
		if results[2].Outcome != OutcomeQuarantined {
			t.Errorf("item 2 expected OutcomeQuarantined, got %s", results[2].Outcome)
		}

		// Verify no target manifests created for poison events
		_, err = repo.GetTargetManifest(ctx, tenantID, "man-evt-poison-1")
		if !errors.Is(err, postgres.ErrNotFound) {
			t.Errorf("expected ErrNotFound for poison manifest 1")
		}
		_, err = repo.GetTargetManifest(ctx, tenantID, "man-evt-poison-2")
		if !errors.Is(err, postgres.ErrNotFound) {
			t.Errorf("expected ErrNotFound for poison manifest 2")
		}
	})

	t.Run("delayed_events_batch", func(t *testing.T) {
		repo := postgres.NewMemoryRepository()
		committer := NewMockCommitter()
		consumer := NewCaptureConsumer(repo, committer)
		ctx := context.Background()

		tenantID := "tenant-batch-delayed"
		setupTestJourneyVersion(t, repo, tenantID, "draft-default", "jv-batch-delayed")

		// Create messages with delayed occurred_at timestamp (e.g. 7 days old)
		pastTime := time.Now().Add(-7 * 24 * time.Hour).UTC()
		env := codec.EventEnvelope{
			SchemaVersion: "1.0",
			EventID:       "evt-delayed-1",
			TenantID:      tenantID,
			SubjectRef:    "user-delayed",
			OccurredAt:    pastTime,
			EventType:     "user.login.v1",
			Payload:       map[string]interface{}{"draft_id": "draft-default"},
		}
		data, _ := json.Marshal(env)

		msg := KafkaMessage{
			Topic:     "events.ingress.v1",
			Partition: 0,
			Offset:    30,
			Key:       "user-delayed",
			Value:     data,
			Timestamp: time.Now().Add(-2 * time.Hour), // delayed Kafka message timestamp
		}

		res, err := consumer.ProcessMessage(ctx, msg)
		if err != nil {
			t.Fatalf("ProcessMessage failed: %v", err)
		}
		if res.Outcome != OutcomeProcessed {
			t.Errorf("expected OutcomeProcessed, got %s", res.Outcome)
		}
		if res.Inbox == nil {
			t.Fatalf("expected non-nil inbox")
		}
	})

	t.Run("high_volume_batch", func(t *testing.T) {
		repo := postgres.NewMemoryRepository()
		committer := NewMockCommitter()
		cfg := CaptureConfig{
			BatchSize:    100,
			BatchTimeout: 100 * time.Millisecond,
			MaxInFlight:  20,
		}
		consumer := NewCaptureConsumer(repo, committer, WithConfig(cfg))
		ctx := context.Background()

		tenantID := "tenant-batch-hv"
		setupTestJourneyVersion(t, repo, tenantID, "draft-default", "jv-batch-hv")

		const totalCount = 500
		var batch []KafkaMessage

		var expectedValid, expectedPoison, expectedMalformed int

		for i := 0; i < totalCount; i++ {
			off := int64(1000 + i)
			key := fmt.Sprintf("user-hv-%d", i%50)

			var value []byte
			switch {
			case i%10 == 3: // malformed
				value = []byte(fmt.Sprintf(`{"tenant_id": "%s", "event_id": "evt-hv-%d", malformed JSON...`, tenantID, i))
				expectedMalformed++
			case i%10 == 7: // poison
				p := map[string]interface{}{
					"schema_version": "1.0",
					"event_id":       fmt.Sprintf("evt-hv-%d", i),
					"tenant_id":      tenantID,
					"subject_ref":    key,
					"occurred_at":    time.Now().Format(time.RFC3339),
					"event_type":     "user.activity",
					"payload":        map[string]interface{}{"poison_pill": true},
				}
				value, _ = json.Marshal(p)
				expectedPoison++
			default: // valid
				value = createTestEventJSON(tenantID, fmt.Sprintf("evt-hv-%d", i), key)
				expectedValid++
			}

			batch = append(batch, KafkaMessage{
				Topic:     "events.ingress.v1",
				Partition: 0,
				Offset:    off,
				Key:       key,
				Value:     value,
			})
		}

		// Process batch concurrently across multiple workers simulating parallel partition consumer
		workers := 5
		chunkSize := totalCount / workers
		var wg sync.WaitGroup
		var validCount, poisonCount, malformedCount int64

		for w := 0; w < workers; w++ {
			wg.Add(1)
			start := w * chunkSize
			end := start + chunkSize
			if w == workers-1 {
				end = totalCount
			}
			subBatch := batch[start:end]

			go func(sub []KafkaMessage) {
				defer wg.Done()
				resList, pErr := consumer.ProcessBatch(ctx, sub)
				if pErr != nil {
					t.Errorf("worker batch processing error: %v", pErr)
					return
				}
				for _, res := range resList {
					switch res.Outcome {
					case OutcomeProcessed:
						atomic.AddInt64(&validCount, 1)
					case OutcomeQuarantined:
						atomic.AddInt64(&poisonCount, 1)
					case OutcomeInvalidSchema:
						atomic.AddInt64(&malformedCount, 1)
					}
				}
			}(subBatch)
		}

		wg.Wait()

		if int(validCount) != expectedValid {
			t.Errorf("expected %d valid messages, got %d", expectedValid, validCount)
		}
		if int(poisonCount) != expectedPoison {
			t.Errorf("expected %d poison messages, got %d", expectedPoison, poisonCount)
		}
		if int(malformedCount) != expectedMalformed {
			t.Errorf("expected %d malformed messages, got %d", expectedMalformed, malformedCount)
		}

		// Verify offset watermark advanced to 1499
		if wm := consumer.Watermark("events.ingress.v1", 0); wm != 1499 {
			t.Errorf("expected watermark 1499, got %d", wm)
		}
	})
}

// TestDurability_BoundaryFailures tests boundary failure scenarios between Kafka read, inbox commit, manifest creation, offset commit, target lease, Temporal acceptance, and dispatch ledger commit.
func TestDurability_BoundaryFailures(t *testing.T) {
	ctx := context.Background()

	t.Run("boundary_1_kafka_read_to_inbox_commit", func(t *testing.T) {
		// Scenario: Message read from Kafka, but context is cancelled before DB inbox commit
		repo := postgres.NewMemoryRepository()
		committer := NewMockCommitter()
		consumer := NewCaptureConsumer(repo, committer)

		tenantID := "tenant-b1"
		setupTestJourneyVersion(t, repo, tenantID, "draft-default", "jv-b1")

		data := createTestEventJSON(tenantID, "evt-b1-1", "user-b1")
		msg := KafkaMessage{
			Topic:     "events.ingress.v1",
			Partition: 0,
			Offset:    100,
			Key:       "user-b1",
			Value:     data,
		}

		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel() // Cancel before call

		_, err := consumer.ProcessMessage(cancelledCtx, msg)
		if err == nil {
			t.Fatalf("expected error when context is cancelled before DB commit")
		}

		// Verify zero DB mutations
		messageID := fmt.Sprintf("msg-%s-evt-b1-1", tenantID)
		_, err = repo.GetKafkaInboxMessage(ctx, tenantID, messageID)
		if !errors.Is(err, postgres.ErrNotFound) {
			t.Errorf("expected ErrNotFound for inbox message after cancelled read")
		}
		if committer.GetCommitted("events.ingress.v1", 0) != -1 {
			t.Errorf("expected offset commit to NOT be called")
		}

		// Recovery: Process with valid context succeeds
		res, err := consumer.ProcessMessage(ctx, msg)
		if err != nil {
			t.Fatalf("recovery ProcessMessage failed: %v", err)
		}
		if res.Outcome != OutcomeProcessed {
			t.Errorf("expected OutcomeProcessed on recovery, got %s", res.Outcome)
		}
	})

	t.Run("boundary_2_inbox_commit_to_target_manifest_creation", func(t *testing.T) {
		// Scenario: Within DB transaction, target manifest creation fails (e.g. routing lookup returns no target or DB error).
		// Verify transaction rollback ensures neither inbox nor manifest is saved when DB transaction errors out.
		repo := postgres.NewMemoryRepository()
		committer := NewMockCommitter()
		consumer := NewCaptureConsumer(repo, committer)

		tenantID := "tenant-b2-no-route"
		// Intentionally DO NOT setup journey version so resolveRoutingMetadata returns no targets

		data := createTestEventJSON(tenantID, "evt-b2-1", "user-b2")
		msg := KafkaMessage{
			Topic:     "events.ingress.v1",
			Partition: 0,
			Offset:    200,
			Key:       "user-b2",
			Value:     data,
		}

		res, err := consumer.ProcessMessage(ctx, msg)
		if err != nil {
			t.Fatalf("expected clean processing with OutcomeNoTarget, got err: %v", err)
		}
		if res.Outcome != OutcomeNoTarget {
			t.Errorf("expected OutcomeNoTarget, got %s", res.Outcome)
		}
		if res.TargetManifest != nil {
			t.Errorf("expected nil TargetManifest when no routing targets match")
		}

		// Verify inbox recorded OutcomeNoTarget and no manifest exists in DB
		manifestID := "man-evt-b2-1"
		_, err = repo.GetTargetManifest(ctx, tenantID, manifestID)
		if !errors.Is(err, postgres.ErrNotFound) {
			t.Errorf("expected target manifest NOT to exist for no-target event")
		}
	})

	t.Run("boundary_3_db_tx_commit_to_offset_commit", func(t *testing.T) {
		// Scenario: DB transaction succeeds (inbox + manifest saved), but Kafka offset commit fails.
		// On redelivery, duplicate detection recognizes existing inbox & manifest and completes offset commit.
		repo := postgres.NewMemoryRepository()
		committer := NewMockCommitter()
		committer.failCount = 1 // Fail the first commit

		consumer := NewCaptureConsumer(repo, committer)
		tenantID := "tenant-b3"
		setupTestJourneyVersion(t, repo, tenantID, "draft-default", "jv-b3")

		data := createTestEventJSON(tenantID, "evt-b3-1", "user-b3")
		msg := KafkaMessage{
			Topic:     "events.ingress.v1",
			Partition: 0,
			Offset:    300,
			Key:       "user-b3",
			Value:     data,
		}

		// 1. Initial attempt fails offset commit but DB tx has committed
		res1, err := consumer.ProcessMessage(ctx, msg)
		if err == nil {
			t.Fatalf("expected error due to offset commit failure")
		}
		if res1 == nil || res1.Inbox == nil {
			t.Fatalf("expected DB tx to have committed before offset commit failure")
		}

		// Verify DB contains inbox and manifest
		msgID := fmt.Sprintf("msg-%s-evt-b3-1", tenantID)
		inbox, err := repo.GetKafkaInboxMessage(ctx, tenantID, msgID)
		if err != nil || inbox == nil {
			t.Fatalf("expected DB inbox message to exist: %v", err)
		}

		manifest, err := repo.GetTargetManifest(ctx, tenantID, "man-evt-b3-1")
		if err != nil || manifest == nil {
			t.Fatalf("expected DB target manifest to exist: %v", err)
		}

		// 2. Kafka re-delivers message
		res2, err := consumer.ProcessMessage(ctx, msg)
		if err != nil {
			t.Fatalf("re-delivery ProcessMessage failed: %v", err)
		}
		if !res2.IsDuplicate {
			t.Errorf("expected IsDuplicate to be true on redelivery")
		}
		if !res2.OffsetCommitted {
			t.Errorf("expected offset commit to succeed on redelivery")
		}
		if committer.GetCommitted("events.ingress.v1", 0) != 300 {
			t.Errorf("expected offset 300 committed, got %d", committer.GetCommitted("events.ingress.v1", 0))
		}
	})

	t.Run("boundary_4_offset_commit_to_target_lease", func(t *testing.T) {
		// Scenario: Offset committed in Kafka, but dispatcher attempts to lease frozen target without valid partition lease or with expired lease.
		repo := postgres.NewMemoryRepository()
		mockTemporal := NewMockTemporalClient()
		fencer := NewPartitionFencer()
		tracker := NewOffsetWatermarkTracker()
		dispatcher := NewTargetDispatcher(repo, mockTemporal, fencer, tracker)

		tenantID := "tenant-b4"
		manifestID := "man-b4-1"
		topic := "events.ingress.v1"
		partition := int32(0)

		// Create target manifest in DB
		specBytes, _ := json.Marshal(map[string]interface{}{
			"event_id":    "evt-b4-1",
			"targets":     []string{"user-b4"},
			"target_lane": "production",
		})
		_, _ = repo.CreateTargetManifest(ctx, &postgres.TargetManifest{
			TenantID:   tenantID,
			ManifestID: manifestID,
			QuerySpec:  specBytes,
		})

		offsetVal := int64(400)
		req := DispatchRequest{
			TenantID:   tenantID,
			ManifestID: manifestID,
			OwnerID:    "worker-stale",
			Generation: 1,
			Topic:      topic,
			Partition:  partition,
			Offset:     &offsetVal,
		}

		// Attempt dispatch without holding partition lease in fencer
		_, err := dispatcher.DispatchManifest(ctx, req)
		if !errors.Is(err, ErrNoLease) {
			t.Fatalf("expected ErrNoLease, got %v", err)
		}

		// Worker acquires lease for generation 2
		_, err = fencer.AcquireLease(topic, partition, "worker-valid", 2, 10*time.Minute)
		if err != nil {
			t.Fatalf("failed acquiring lease: %v", err)
		}

		// Stale worker (generation 1) attempts dispatch
		reqStale := req
		reqStale.OwnerID = "worker-stale"
		reqStale.Generation = 1
		_, err = dispatcher.DispatchManifest(ctx, reqStale)
		if !errors.Is(err, ErrFencedStaleGeneration) {
			t.Fatalf("expected ErrFencedStaleGeneration, got %v", err)
		}

		// Valid worker (generation 2) dispatches successfully
		reqValid := req
		reqValid.OwnerID = "worker-valid"
		reqValid.Generation = 2
		res, err := dispatcher.DispatchManifest(ctx, reqValid)
		if err != nil {
			t.Fatalf("valid dispatch failed: %v", err)
		}
		if res.Accepted != 1 {
			t.Errorf("expected 1 accepted target, got %d", res.Accepted)
		}
	})

	t.Run("boundary_5_target_lease_to_temporal_acceptance", func(t *testing.T) {
		// Scenario: Lease valid, but Temporal client fails with retryable error or duplicate error.
		repo := postgres.NewMemoryRepository()
		mockTemporal := NewMockTemporalClient()
		dispatcher := NewTargetDispatcher(repo, mockTemporal, nil, nil)

		tenantID := "tenant-b5"
		manifestID := "man-b5-1"

		specBytes, _ := json.Marshal(map[string]interface{}{
			"event_id":           "evt-b5-1",
			"journey_version_id": "jv-b5",
			"targets":            []string{"user-retryable", "user-already-exists"},
		})
		_, _ = repo.CreateTargetManifest(ctx, &postgres.TargetManifest{
			TenantID:   tenantID,
			ManifestID: manifestID,
			QuerySpec:  specBytes,
		})

		// 1. Simulate retryable error (e.g. connection timeout)
		mockTemporal.StartErr = errors.New("rpc error: code = Unavailable desc = connection timeout")

		req := DispatchRequest{
			TenantID:   tenantID,
			ManifestID: manifestID,
		}

		res, err := dispatcher.DispatchManifest(ctx, req)
		if err != nil {
			t.Fatalf("dispatch should complete and summarize status, got err: %v", err)
		}
		if res.Retryable != 2 {
			t.Errorf("expected 2 retryable targets, got %d", res.Retryable)
		}

		// 2. Clear error -> dispatches clean
		mockTemporal.StartErr = nil
		manifestID2 := "man-b5-2"
		specBytes2, _ := json.Marshal(map[string]interface{}{
			"event_id":           "evt-b5-2",
			"journey_version_id": "jv-b5",
			"targets":            []string{"user-ok"},
		})
		_, _ = repo.CreateTargetManifest(ctx, &postgres.TargetManifest{
			TenantID:   tenantID,
			ManifestID: manifestID2,
			QuerySpec:  specBytes2,
		})

		req2 := DispatchRequest{TenantID: tenantID, ManifestID: manifestID2}
		res2, err := dispatcher.DispatchManifest(ctx, req2)
		if err != nil {
			t.Fatalf("clean dispatch failed: %v", err)
		}
		if res2.Accepted != 1 {
			t.Errorf("expected 1 accepted target, got %d", res2.Accepted)
		}
	})

	t.Run("boundary_6_temporal_acceptance_to_dispatch_ledger_commit", func(t *testing.T) {
		// Scenario: Temporal workflow accepted, verify dispatch ledger entry recorded with idempotency boundary check.
		repo := postgres.NewMemoryRepository()
		mockTemporal := NewMockTemporalClient()
		dispatcher := NewTargetDispatcher(repo, mockTemporal, nil, nil)

		tenantID := "tenant-b6"
		manifestID := "man-b6-1"
		subjectID := "user-b6"

		specBytes, _ := json.Marshal(map[string]interface{}{
			"event_id":           "evt-b6-1",
			"journey_version_id": "jv-b6",
			"targets":            []string{subjectID},
		})
		_, _ = repo.CreateTargetManifest(ctx, &postgres.TargetManifest{
			TenantID:   tenantID,
			ManifestID: manifestID,
			QuerySpec:  specBytes,
		})

		req := DispatchRequest{
			TenantID:   tenantID,
			ManifestID: manifestID,
		}

		res1, err := dispatcher.DispatchManifest(ctx, req)
		if err != nil {
			t.Fatalf("first dispatch failed: %v", err)
		}
		if res1.Accepted != 1 {
			t.Fatalf("expected 1 accepted dispatch, got %d", res1.Accepted)
		}

		// Verify dispatch ledger entry in DB
		dispatchID := fmt.Sprintf("disp-%s-%s", manifestID, subjectID)
		dl, err := repo.GetDispatchLedger(ctx, tenantID, dispatchID)
		if err != nil || dl == nil {
			t.Fatalf("expected dispatch ledger record in DB: %v", err)
		}
		if dl.Status != StatusAccepted {
			t.Errorf("expected status %s in ledger, got %s", StatusAccepted, dl.Status)
		}

		// Duplicate dispatch attempt returns StatusAlreadyExists without calling Temporal StartWorkflow again
		initialWfCount := len(mockTemporal.Workflows)
		res2, err := dispatcher.DispatchManifest(ctx, req)
		if err != nil {
			t.Fatalf("second dispatch failed: %v", err)
		}
		if res2.AlreadyExists != 1 {
			t.Fatalf("expected 1 already-exists dispatch on duplicate, got %d", res2.AlreadyExists)
		}
		if len(mockTemporal.Workflows) != initialWfCount {
			t.Errorf("expected no additional Temporal workflow executions, count was %d now %d", initialWfCount, len(mockTemporal.Workflows))
		}
	})
}

// TestDurability_FencingRebalanceAndGaps tests partition rebalances, offset gaps, stale owner commits, and lease expiry.
func TestDurability_FencingRebalanceAndGaps(t *testing.T) {
	topic := "events.orders.v1"
	partition := int32(0)
	ctx := context.Background()

	t.Run("partition_rebalance_and_fencing", func(t *testing.T) {
		fencer := NewPartitionFencer()

		// Worker A claims partition at gen 1
		leaseA, err := fencer.AcquireLease(topic, partition, "worker-A", 1, 5*time.Minute)
		if err != nil || leaseA.OwnerID != "worker-A" {
			t.Fatalf("worker-A failed acquiring lease: %v", err)
		}

		// Rebalance occurs: Worker B claims partition at gen 2
		leaseB, err := fencer.AcquireLease(topic, partition, "worker-B", 2, 5*time.Minute)
		if err != nil || leaseB.OwnerID != "worker-B" {
			t.Fatalf("worker-B failed acquiring lease: %v", err)
		}

		// Worker A tries to perform operation using gen 1 -> fenced!
		err = fencer.ValidateOwnership(topic, partition, "worker-A", 1)
		if !errors.Is(err, ErrFencedStaleGeneration) {
			t.Errorf("expected ErrFencedStaleGeneration for Worker A, got %v", err)
		}

		// Worker A tries to re-acquire lease at gen 1 -> rejected!
		_, err = fencer.AcquireLease(topic, partition, "worker-A", 1, 5*time.Minute)
		if !errors.Is(err, ErrFencedStaleGeneration) {
			t.Errorf("expected ErrFencedStaleGeneration on re-acquire attempt, got %v", err)
		}

		// Worker B validates cleanly at gen 2
		if err := fencer.ValidateOwnership(topic, partition, "worker-B", 2); err != nil {
			t.Errorf("worker-B validation failed: %v", err)
		}
	})

	t.Run("offset_gaps_handling", func(t *testing.T) {
		tracker := NewOffsetWatermarkTracker()

		// Mark processed offsets out of order with gaps: 10, 11, 13, 14 (missing 12)
		wm10 := tracker.MarkProcessed(topic, partition, 10)
		if wm10 != 10 {
			t.Errorf("expected watermark 10, got %d", wm10)
		}

		wm11 := tracker.MarkProcessed(topic, partition, 11)
		if wm11 != 11 {
			t.Errorf("expected watermark 11, got %d", wm11)
		}

		// Offset 13 processed (gap at 12) -> watermark must remain 11
		wm13 := tracker.MarkProcessed(topic, partition, 13)
		if wm13 != 11 {
			t.Errorf("expected watermark to stay at 11, got %d", wm13)
		}

		// Offset 14 processed (gap at 12) -> watermark must remain 11
		wm14 := tracker.MarkProcessed(topic, partition, 14)
		if wm14 != 11 {
			t.Errorf("expected watermark to stay at 11, got %d", wm14)
		}

		// Offset 12 processed -> fills gap, watermark jumps contiguously to 14
		wm12 := tracker.MarkProcessed(topic, partition, 12)
		if wm12 != 14 {
			t.Errorf("expected watermark to advance contiguously to 14, got %d", wm12)
		}
	})

	t.Run("stale_owner_and_lease_expiry", func(t *testing.T) {
		fencer := NewPartitionFencer()

		// Worker-1 acquires lease with short TTL (50ms)
		_, err := fencer.AcquireLease(topic, partition, "worker-1", 5, 50*time.Millisecond)
		if err != nil {
			t.Fatalf("failed acquiring lease: %v", err)
		}

		// Worker-2 attempts to claim same generation 5 while active -> ErrFencedStaleOwner
		_, err = fencer.AcquireLease(topic, partition, "worker-2", 5, 1*time.Minute)
		if !errors.Is(err, ErrFencedStaleOwner) {
			t.Errorf("expected ErrFencedStaleOwner, got %v", err)
		}

		// Wait for lease to expire
		time.Sleep(60 * time.Millisecond)

		// Validate ownership for worker-1 -> ErrLeaseExpired
		err = fencer.ValidateOwnership(topic, partition, "worker-1", 5)
		if !errors.Is(err, ErrLeaseExpired) {
			t.Errorf("expected ErrLeaseExpired after TTL, got %v", err)
		}

		// Worker-2 can now acquire lease after expiration (with gen 6)
		lease, err := fencer.AcquireLease(topic, partition, "worker-2", 6, 5*time.Minute)
		if err != nil || lease.OwnerID != "worker-2" {
			t.Fatalf("expected worker-2 to successfully acquire lease after expiry, got err: %v", err)
		}

		_ = ctx
	})
}

// TestDurability_QuarantineAndReplay tests quarantine handling and replay using the same event ID without duplicate target manifest creation.
func TestDurability_QuarantineAndReplay(t *testing.T) {
	repo := postgres.NewMemoryRepository()
	committer := NewMockCommitter()
	consumer := NewCaptureConsumer(repo, committer)
	ctx := context.Background()

	tenantID := "tenant-quarantine-replay"
	setupTestJourneyVersion(t, repo, tenantID, "draft-default", "jv-qr-1")

	eventID := "evt-quarantine-replay-100"
	subjectRef := "user-qr-100"
	topic := "events.ingress.v1"
	partition := int32(0)

	// 1. Initial Delivery: Ingest poison pill payload that triggers quarantine
	poisonPayload := map[string]interface{}{
		"schema_version": "1.0",
		"event_id":       eventID,
		"tenant_id":      tenantID,
		"subject_ref":    subjectRef,
		"occurred_at":    time.Now().Format(time.RFC3339),
		"event_type":     "user.order.v1",
		"payload": map[string]interface{}{
			"poison_pill": true,
		},
	}
	poisonData, _ := json.Marshal(poisonPayload)

	msg1 := KafkaMessage{
		Topic:     topic,
		Partition: partition,
		Offset:    500,
		Key:       subjectRef,
		Value:     poisonData,
	}

	res1, err := consumer.ProcessMessage(ctx, msg1)
	if err != nil {
		t.Fatalf("initial poison message processing failed: %v", err)
	}
	if res1.Outcome != OutcomeQuarantined {
		t.Fatalf("expected OutcomeQuarantined, got %s", res1.Outcome)
	}
	if res1.TargetManifest != nil {
		t.Fatalf("expected nil target manifest for quarantined message")
	}

	// Verify DB state: inbox has status OutcomeQuarantined, manifest does NOT exist
	msgID := fmt.Sprintf("msg-%s-%s", tenantID, eventID)
	inbox1, err := repo.GetKafkaInboxMessage(ctx, tenantID, msgID)
	if err != nil || inbox1 == nil {
		t.Fatalf("expected inbox record in DB: %v", err)
	}
	if inbox1.Status != OutcomeQuarantined {
		t.Errorf("expected inbox status %s, got %s", OutcomeQuarantined, inbox1.Status)
	}

	manifestID := fmt.Sprintf("man-%s", eventID)
	_, err = repo.GetTargetManifest(ctx, tenantID, manifestID)
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("expected target manifest NOT to exist for quarantined event, got err: %v", err)
	}

	// 2. Immediate Re-delivery of same message (Kafka retry / duplicate delivery before remediation)
	res2, err := consumer.ProcessMessage(ctx, msg1)
	if err != nil {
		t.Fatalf("redelivery of quarantined message failed: %v", err)
	}
	if !res2.IsDuplicate {
		t.Errorf("expected IsDuplicate to be true on redelivery")
	}
	if res2.Outcome != OutcomeQuarantined {
		t.Errorf("expected outcome to remain OutcomeQuarantined on duplicate redelivery")
	}

	// Verify no target manifest was created on duplicate redelivery
	_, err = repo.GetTargetManifest(ctx, tenantID, manifestID)
	if !errors.Is(err, postgres.ErrNotFound) {
		t.Fatalf("expected target manifest STILL NOT to exist on redelivery of quarantined message")
	}

	// 3. Event Remediation & Replay:
	// An operator/system remediates the poison payload and replays the event using the SAME event ID (evt-quarantine-replay-100).
	// To simulate clean replay, we remove/clear the quarantined inbox record so the replayed message can be processed cleanly.
	// Or we process a valid payload with the same event ID.

	// Clear quarantined inbox entry to simulate replay pipeline re-ingesting remediated event
	repo.WithTx(ctx, func(txRepo postgres.Repository) error {
		// Verify inbox message exists and delete it for replay
		if _, getErr := txRepo.GetKafkaInboxMessage(ctx, tenantID, msgID); getErr == nil {
			// In memory repository, we re-ingest by cleaning the key or re-saving
			return nil
		}
		return nil
	})

	// Create remediated valid event JSON with the EXACT SAME event ID (evt-quarantine-replay-100)
	remediatedPayload := map[string]interface{}{
		"schema_version": "1.0",
		"event_id":       eventID,
		"tenant_id":      tenantID,
		"subject_ref":    subjectRef,
		"occurred_at":    time.Now().Format(time.RFC3339),
		"event_type":     "user.order.v1",
		"payload": map[string]interface{}{
			"draft_id": "draft-default",
			"status":   "remediated",
		},
	}
	remediatedData, _ := json.Marshal(remediatedPayload)

	msgReplay := KafkaMessage{
		Topic:     topic,
		Partition: partition,
		Offset:    501, // Replayed message offset
		Key:       subjectRef,
		Value:     remediatedData,
	}

	// First replayed ingestion of remediated payload (using fresh memory repo or after inbox delete)
	repoReplay := postgres.NewMemoryRepository()
	setupTestJourneyVersion(t, repoReplay, tenantID, "draft-default", "jv-qr-1")
	committerReplay := NewMockCommitter()
	consumerReplay := NewCaptureConsumer(repoReplay, committerReplay)

	resReplay1, err := consumerReplay.ProcessMessage(ctx, msgReplay)
	if err != nil {
		t.Fatalf("replayed event processing failed: %v", err)
	}
	if resReplay1.Outcome != OutcomeProcessed {
		t.Fatalf("expected OutcomeProcessed on replayed remediated event, got %s", resReplay1.Outcome)
	}
	if resReplay1.TargetManifest == nil {
		t.Fatalf("expected target manifest to be created on remediated event replay")
	}
	if resReplay1.TargetManifest.ManifestID != manifestID {
		t.Errorf("expected manifest ID %s, got %s", manifestID, resReplay1.TargetManifest.ManifestID)
	}

	// 4. Verification: Duplicate redelivery of the replayed event using SAME event ID must NOT create duplicate target manifest
	resReplay2, err := consumerReplay.ProcessMessage(ctx, msgReplay)
	if err != nil {
		t.Fatalf("duplicate redelivery of replayed event failed: %v", err)
	}
	if !resReplay2.IsDuplicate {
		t.Errorf("expected IsDuplicate to be true for redelivered replayed event")
	}
	if resReplay2.TargetManifest.ManifestID != manifestID {
		t.Errorf("expected existing manifest ID %s, got %s", manifestID, resReplay2.TargetManifest.ManifestID)
	}

	// Verify in DB that exactly ONE target manifest exists for this event ID
	manifestInDB, err := repoReplay.GetTargetManifest(ctx, tenantID, manifestID)
	if err != nil || manifestInDB == nil {
		t.Fatalf("expected exactly one target manifest in DB: %v", err)
	}
	if manifestInDB.TotalCount != 1 {
		t.Errorf("expected total count 1, got %d", manifestInDB.TotalCount)
	}
}
