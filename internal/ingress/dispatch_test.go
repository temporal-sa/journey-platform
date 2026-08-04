package ingress_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/ingress"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
)

func TestOffsetWatermarkTracker_ContiguousWatermarkAndGaps(t *testing.T) {
	tracker := ingress.NewOffsetWatermarkTracker()
	topic := "events-user"
	partition := int32(0)

	// Initial watermark should be -1
	if wm := tracker.Watermark(topic, partition); wm != -1 {
		t.Fatalf("expected initial watermark -1, got %d", wm)
	}

	// 1. Process offset 0 -> watermark 0
	wm := tracker.MarkProcessed(topic, partition, 0)
	if wm != 0 {
		t.Fatalf("expected watermark 0 after offset 0, got %d", wm)
	}

	// 2. Process offset 1 -> watermark 1
	wm = tracker.MarkProcessed(topic, partition, 1)
	if wm != 1 {
		t.Fatalf("expected watermark 1 after offset 1, got %d", wm)
	}

	// 3. Process offset 3 (gap at offset 2) -> watermark remains 1
	wm = tracker.MarkProcessed(topic, partition, 3)
	if wm != 1 {
		t.Fatalf("expected watermark to stay 1 due to gap at 2, got %d", wm)
	}

	// 4. Process offset 4 -> watermark remains 1
	wm = tracker.MarkProcessed(topic, partition, 4)
	if wm != 1 {
		t.Fatalf("expected watermark to stay 1 due to gap at 2, got %d", wm)
	}

	// 5. Fill gap by processing offset 2 -> watermark advances contiguously to 4
	wm = tracker.MarkProcessed(topic, partition, 2)
	if wm != 4 {
		t.Fatalf("expected watermark to jump to 4 after filling gap at 2, got %d", wm)
	}

	if finalWM := tracker.Watermark(topic, partition); finalWM != 4 {
		t.Fatalf("expected final watermark 4, got %d", finalWM)
	}
}

func TestPartitionFencer_AcquireAndValidateOwnership(t *testing.T) {
	fencer := ingress.NewPartitionFencer()
	topic := "events-orders"
	partition := int32(1)
	owner1 := "worker-1"
	owner2 := "worker-2"
	gen1 := int64(1)
	gen2 := int64(2)
	ttl := 100 * time.Millisecond

	// 1. Validate ownership without lease -> ErrNoLease
	err := fencer.ValidateOwnership(topic, partition, owner1, gen1)
	if !errors.Is(err, ingress.ErrNoLease) {
		t.Fatalf("expected ErrNoLease, got %v", err)
	}

	// 2. Acquire lease worker-1, gen 1
	lease, err := fencer.AcquireLease(topic, partition, owner1, gen1, ttl)
	if err != nil {
		t.Fatalf("failed to acquire lease: %v", err)
	}
	if lease.OwnerID != owner1 || lease.Generation != gen1 {
		t.Fatalf("unexpected lease data: %+v", lease)
	}

	// 3. Validate ownership valid -> success
	if err := fencer.ValidateOwnership(topic, partition, owner1, gen1); err != nil {
		t.Fatalf("expected valid ownership, got %v", err)
	}

	// 4. Stale owner rejection (worker-2 trying with gen 1)
	err = fencer.ValidateOwnership(topic, partition, owner2, gen1)
	if !errors.Is(err, ingress.ErrFencedStaleOwner) {
		t.Fatalf("expected ErrFencedStaleOwner, got %v", err)
	}

	// 5. Stale generation rejection (worker-1 trying with gen 0)
	err = fencer.ValidateOwnership(topic, partition, owner1, 0)
	if !errors.Is(err, ingress.ErrFencedStaleGeneration) {
		t.Fatalf("expected ErrFencedStaleGeneration, got %v", err)
	}

	// 6. Worker-2 acquires higher generation (gen 2) -> fences worker-1
	_, err = fencer.AcquireLease(topic, partition, owner2, gen2, ttl)
	if err != nil {
		t.Fatalf("failed worker-2 lease acquisition: %v", err)
	}

	// Worker-1 with gen 1 is now fenced out
	err = fencer.ValidateOwnership(topic, partition, owner1, gen1)
	if !errors.Is(err, ingress.ErrFencedStaleGeneration) {
		t.Fatalf("expected ErrFencedStaleGeneration for fenced worker-1, got %v", err)
	}

	// Worker-2 with gen 2 succeeds
	if err := fencer.ValidateOwnership(topic, partition, owner2, gen2); err != nil {
		t.Fatalf("expected valid ownership for worker-2, got %v", err)
	}

	// 7. Test lease expiration
	time.Sleep(150 * time.Millisecond)
	err = fencer.ValidateOwnership(topic, partition, owner2, gen2)
	if !errors.Is(err, ingress.ErrLeaseExpired) {
		t.Fatalf("expected ErrLeaseExpired, got %v", err)
	}
}

func TestTargetDispatcher_IdempotentDispatchAndLedgerResults(t *testing.T) {
	ctx := context.Background()
	repo := postgres.NewMemoryRepository()
	mockTemporal := ingress.NewMockTemporalClient()
	fencer := ingress.NewPartitionFencer()
	tracker := ingress.NewOffsetWatermarkTracker()

	dispatcher := ingress.NewTargetDispatcher(repo, mockTemporal, fencer, tracker)

	tenantID := "tenant-test"
	manifestID := "man-001"
	topic := "targets-topic"
	partition := int32(0)
	ownerID := "worker-alpha"
	gen := int64(10)

	// Acquire partition lease
	_, err := fencer.AcquireLease(topic, partition, ownerID, gen, 5*time.Minute)
	if err != nil {
		t.Fatalf("failed to acquire partition lease: %v", err)
	}

	// Create target manifest in DB
	querySpecMap := map[string]interface{}{
		"event_id":           "evt-100",
		"event_type":         "user.onboarding",
		"journey_version_id": "jv-v1",
		"targets":            []string{"subj-1", "subj-2", "subj-3"},
		"target_lane":        "transactional",
	}
	specBytes, _ := json.Marshal(querySpecMap)

	tm := &postgres.TargetManifest{
		TenantID:   tenantID,
		ManifestID: manifestID,
		Name:       "onboarding-manifest",
		QuerySpec:  specBytes,
		TotalCount: 3,
	}
	_, err = repo.CreateTargetManifest(ctx, tm)
	if err != nil {
		t.Fatalf("failed creating target manifest: %v", err)
	}

	offsetVal := int64(100)
	req := ingress.DispatchRequest{
		TenantID:   tenantID,
		ManifestID: manifestID,
		OwnerID:    ownerID,
		Generation: gen,
		Topic:      topic,
		Partition:  partition,
		Offset:     &offsetVal,
	}

	// 1. Initial Dispatch -> 3 Accepted
	res, err := dispatcher.DispatchManifest(ctx, req)
	if err != nil {
		t.Fatalf("dispatch manifest failed: %v", err)
	}

	if res.Accepted != 3 {
		t.Fatalf("expected 3 accepted targets, got %d", res.Accepted)
	}
	if res.Lane != ingress.LaneTransactional {
		t.Fatalf("expected LaneTransactional, got %s", res.Lane)
	}
	if res.TaskQueue != "JOURNEY_TASK_QUEUE_TRANSACTIONAL" {
		t.Fatalf("expected task queue JOURNEY_TASK_QUEUE_TRANSACTIONAL, got %s", res.TaskQueue)
	}
	if res.Watermark != 100 {
		t.Fatalf("expected watermark 100, got %d", res.Watermark)
	}

	// Verify Temporal received 3 workflow starts
	if len(mockTemporal.Workflows) != 3 {
		t.Fatalf("expected 3 Temporal workflows started, got %d", len(mockTemporal.Workflows))
	}

	// 2. Re-dispatch same manifest -> Idempotent, 3 AlreadyExists
	resDup, err := dispatcher.DispatchManifest(ctx, req)
	if err != nil {
		t.Fatalf("re-dispatch manifest failed: %v", err)
	}
	if resDup.AlreadyExists != 3 {
		t.Fatalf("expected 3 already-exists results on duplicate dispatch, got %d", resDup.AlreadyExists)
	}
	if resDup.Accepted != 0 {
		t.Fatalf("expected 0 new accepted dispatches on duplicate dispatch, got %d", resDup.Accepted)
	}
}

func TestTargetDispatcher_QuarantinedAndErrorResults(t *testing.T) {
	ctx := context.Background()
	repo := postgres.NewMemoryRepository()
	mockTemporal := ingress.NewMockTemporalClient()

	dispatcher := ingress.NewTargetDispatcher(repo, mockTemporal, nil, nil)

	tenantID := "tenant-err"
	manifestID := "man-err"

	// Create tombstone for subject-poison
	_, err := repo.CreateTombstone(ctx, &postgres.Tombstone{
		TenantID:    tenantID,
		TombstoneID: "tomb-1",
		EntityType:  "subject",
		EntityID:    "subject-poison",
		Reason:      "compliance block",
	})
	if err != nil {
		t.Fatalf("failed creating tombstone: %v", err)
	}

	querySpecMap := map[string]interface{}{
		"event_id":           "evt-err",
		"journey_version_id": "jv-err",
		"targets":            []string{"subject-poison", "subject-ok"},
		"target_lane":        "production",
	}
	specBytes, _ := json.Marshal(querySpecMap)

	_, _ = repo.CreateTargetManifest(ctx, &postgres.TargetManifest{
		TenantID:   tenantID,
		ManifestID: manifestID,
		Name:       "error-manifest",
		QuerySpec:  specBytes,
		TotalCount: 2,
	})

	req := ingress.DispatchRequest{
		TenantID:   tenantID,
		ManifestID: manifestID,
	}

	res, err := dispatcher.DispatchManifest(ctx, req)
	if err != nil {
		t.Fatalf("dispatch manifest failed: %v", err)
	}

	if res.Quarantined != 1 {
		t.Fatalf("expected 1 quarantined result for tombstoned subject, got %d", res.Quarantined)
	}
	if res.Accepted != 1 {
		t.Fatalf("expected 1 accepted result for valid subject, got %d", res.Accepted)
	}
}

func TestTargetDispatcher_LaneSegregationAndValidation(t *testing.T) {
	ctx := context.Background()
	repo := postgres.NewMemoryRepository()
	dispatcher := ingress.NewTargetDispatcher(repo, nil, nil, nil)

	tenantID := "tenant-lane"
	manifestID := "man-promo"

	querySpecMap := map[string]interface{}{
		"event_id":    "evt-promo",
		"targets":     []string{"subj-promo"},
		"target_lane": "promotional",
	}
	specBytes, _ := json.Marshal(querySpecMap)

	_, _ = repo.CreateTargetManifest(ctx, &postgres.TargetManifest{
		TenantID:   tenantID,
		ManifestID: manifestID,
		QuerySpec:  specBytes,
	})

	// Cross-lane dispatch attempt (claiming promotional manifest is production)
	reqMismatch := ingress.DispatchRequest{
		TenantID:   tenantID,
		ManifestID: manifestID,
		TargetLane: ingress.LaneProduction,
	}

	_, err := dispatcher.DispatchManifest(ctx, reqMismatch)
	if !errors.Is(err, ingress.ErrInvalidTargetLane) {
		t.Fatalf("expected ErrInvalidTargetLane on cross-lane dispatch, got %v", err)
	}

	// Valid promotional dispatch
	reqValid := ingress.DispatchRequest{
		TenantID:   tenantID,
		ManifestID: manifestID,
		TargetLane: ingress.LanePromotional,
	}

	res, err := dispatcher.DispatchManifest(ctx, reqValid)
	if err != nil {
		t.Fatalf("expected clean dispatch, got %v", err)
	}
	if res.Lane != ingress.LanePromotional {
		t.Fatalf("expected LanePromotional, got %s", res.Lane)
	}
	if res.TaskQueue != "JOURNEY_TASK_QUEUE_PROMOTIONAL" {
		t.Fatalf("expected JOURNEY_TASK_QUEUE_PROMOTIONAL, got %s", res.TaskQueue)
	}
}
