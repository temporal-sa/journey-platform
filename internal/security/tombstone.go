package security

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrSubjectTombstoned = errors.New("operation blocked: subject is tombstoned")
)

type CleanupStatus struct {
	SubjectID             string    `json:"subject_id"`
	TenantID              string    `json:"tenant_id"`
	TombstonedAt          time.Time `json:"tombstoned_at"`
	SubscriptionsCleaned  bool      `json:"subscriptions_cleaned"`
	TargetsCleaned        bool      `json:"targets_cleaned"`
	PayloadRefsCleaned    bool      `json:"payload_refs_cleaned"`
	ProjectionsCleaned    bool      `json:"projections_cleaned"`
	AsyncCleanupCompleted bool      `json:"async_cleanup_completed"`
}

type TombstoneService struct {
	mu            sync.RWMutex
	tombstones    map[string]time.Time      // key: tenantID:subjectID
	cleanupStatus map[string]*CleanupStatus // key: tenantID:subjectID

	// Registered cleanup handlers
	subscriptionHandlers []func(ctx context.Context, tenantID, subjectID string) error
	targetHandlers       []func(ctx context.Context, tenantID, subjectID string) error
	payloadHandlers      []func(ctx context.Context, tenantID, subjectID string) error
	projectionHandlers   []func(ctx context.Context, tenantID, subjectID string) error
}

var defaultTombstoneService = NewTombstoneService()

func NewTombstoneService() *TombstoneService {
	return &TombstoneService{
		tombstones:    make(map[string]time.Time),
		cleanupStatus: make(map[string]*CleanupStatus),
	}
}

// DefaultService returns the package-level tombstone service instance.
func DefaultService() *TombstoneService {
	return defaultTombstoneService
}

func makeKey(tenantID, subjectID string) string {
	if tenantID == "" {
		return subjectID
	}
	return tenantID + ":" + subjectID
}

// TombstoneSubject commits a tombstone instantly blocking all operations, then launches async propagation.
func (s *TombstoneService) TombstoneSubject(ctx context.Context, tenantID, subjectID string) error {
	if subjectID == "" {
		return errors.New("subjectID required for tombstone")
	}

	key := makeKey(tenantID, subjectID)
	now := time.Now()

	s.mu.Lock()
	s.tombstones[key] = now
	status := &CleanupStatus{
		SubjectID:    subjectID,
		TenantID:     tenantID,
		TombstonedAt: now,
	}
	s.cleanupStatus[key] = status
	s.mu.Unlock()

	// Launch async cleanup propagation
	go s.performAsyncCleanup(context.Background(), tenantID, subjectID, status)

	return nil
}

// IsTombstoned returns true if a subject has been tombstoned.
func (s *TombstoneService) IsTombstoned(ctx context.Context, tenantID, subjectID string) bool {
	if subjectID == "" {
		return false
	}
	key := makeKey(tenantID, subjectID)

	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.tombstones[key]
	if exists {
		return true
	}
	// Fallback check by subjectID alone if tenantID was omitted
	if tenantID != "" {
		_, exists = s.tombstones[subjectID]
	}
	return exists
}

func (s *TombstoneService) performAsyncCleanup(ctx context.Context, tenantID, subjectID string, status *CleanupStatus) {
	s.mu.RLock()
	subHandlers := append([]func(context.Context, string, string) error(nil), s.subscriptionHandlers...)
	tgtHandlers := append([]func(context.Context, string, string) error(nil), s.targetHandlers...)
	payHandlers := append([]func(context.Context, string, string) error(nil), s.payloadHandlers...)
	prjHandlers := append([]func(context.Context, string, string) error(nil), s.projectionHandlers...)
	s.mu.RUnlock()

	// 1. Cleanup open subscriptions
	for _, h := range subHandlers {
		_ = h(ctx, tenantID, subjectID)
	}
	s.mu.Lock()
	status.SubscriptionsCleaned = true
	s.mu.Unlock()

	// 2. Cleanup undispatched targets
	for _, h := range tgtHandlers {
		_ = h(ctx, tenantID, subjectID)
	}
	s.mu.Lock()
	status.TargetsCleaned = true
	s.mu.Unlock()

	// 3. Cleanup payload references
	for _, h := range payHandlers {
		_ = h(ctx, tenantID, subjectID)
	}
	s.mu.Lock()
	status.PayloadRefsCleaned = true
	s.mu.Unlock()

	// 4. Cleanup projections
	for _, h := range prjHandlers {
		_ = h(ctx, tenantID, subjectID)
	}
	s.mu.Lock()
	status.ProjectionsCleaned = true
	s.mu.Unlock()

	s.mu.Lock()
	status.AsyncCleanupCompleted = true
	s.mu.Unlock()
}

// GetCleanupStatus returns current status of asynchronous cleanup propagation.
func (s *TombstoneService) GetCleanupStatus(tenantID, subjectID string) *CleanupStatus {
	key := makeKey(tenantID, subjectID)
	s.mu.RLock()
	defer s.mu.RUnlock()

	if st, ok := s.cleanupStatus[key]; ok {
		cp := *st
		return &cp
	}
	return nil
}

// Reset clears all tombstone states (useful for testing).
func (s *TombstoneService) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tombstones = make(map[string]time.Time)
	s.cleanupStatus = make(map[string]*CleanupStatus)
}

// -----------------------------------------------------------------------------
// Package-level helpers forwarding to DefaultService
// -----------------------------------------------------------------------------

func TombstoneSubject(ctx context.Context, tenantID, subjectID string) error {
	return defaultTombstoneService.TombstoneSubject(ctx, tenantID, subjectID)
}

func IsTombstoned(ctx context.Context, tenantID, subjectID string) bool {
	return defaultTombstoneService.IsTombstoned(ctx, tenantID, subjectID)
}

// -----------------------------------------------------------------------------
// Immediate Operation Blocking Guards
// -----------------------------------------------------------------------------

// BlockKafkaManifest blocks Kafka manifest creation if subject is tombstoned.
func BlockKafkaManifest(ctx context.Context, tenantID, subjectID string) error {
	if IsTombstoned(ctx, tenantID, subjectID) {
		return fmt.Errorf("%w: Kafka manifest creation blocked for subject '%s'", ErrSubjectTombstoned, subjectID)
	}
	return nil
}

// BlockTemporalStart blocks Temporal workflow start if subject is tombstoned.
func BlockTemporalStart(ctx context.Context, tenantID, subjectID string) error {
	if IsTombstoned(ctx, tenantID, subjectID) {
		return fmt.Errorf("%w: Temporal workflow start blocked for subject '%s'", ErrSubjectTombstoned, subjectID)
	}
	return nil
}

// BlockEventReplay blocks event replay if subject is tombstoned.
func BlockEventReplay(ctx context.Context, tenantID, subjectID string) error {
	if IsTombstoned(ctx, tenantID, subjectID) {
		return fmt.Errorf("%w: Event replay blocked for subject '%s'", ErrSubjectTombstoned, subjectID)
	}
	return nil
}

// BlockAttributeResolution blocks attribute resolution if subject is tombstoned.
func BlockAttributeResolution(ctx context.Context, tenantID, subjectID string) error {
	if IsTombstoned(ctx, tenantID, subjectID) {
		return fmt.Errorf("%w: Attribute resolution blocked for subject '%s'", ErrSubjectTombstoned, subjectID)
	}
	return nil
}

// BlockAddressResolution blocks address resolution if subject is tombstoned.
func BlockAddressResolution(ctx context.Context, tenantID, subjectID string) error {
	if IsTombstoned(ctx, tenantID, subjectID) {
		return fmt.Errorf("%w: Address resolution blocked for subject '%s'", ErrSubjectTombstoned, subjectID)
	}
	return nil
}

// BlockProviderDispatch blocks provider dispatch if subject is tombstoned.
func BlockProviderDispatch(ctx context.Context, tenantID, subjectID string) error {
	if IsTombstoned(ctx, tenantID, subjectID) {
		return fmt.Errorf("%w: Provider dispatch blocked for subject '%s'", ErrSubjectTombstoned, subjectID)
	}
	return nil
}
