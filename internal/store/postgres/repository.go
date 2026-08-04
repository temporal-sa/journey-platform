package postgres

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Repository interface defines all data access methods for the 18 PostgreSQL tables with transaction support.
type Repository interface {
	// Transaction Support
	WithTx(ctx context.Context, fn func(repo Repository) error) error

	// 1. Catalogs
	CreateCatalog(ctx context.Context, c *Catalog) (*Catalog, error)
	GetCatalog(ctx context.Context, tenantID, recordID string) (*Catalog, error)
	ListCatalogs(ctx context.Context, tenantID string) ([]Catalog, error)
	UpdateCatalog(ctx context.Context, c *Catalog) (*Catalog, error)
	DeleteCatalog(ctx context.Context, tenantID, recordID string) error

	// 2. Journey Drafts
	CreateJourneyDraft(ctx context.Context, d *JourneyDraft) (*JourneyDraft, error)
	GetJourneyDraft(ctx context.Context, tenantID, draftID string) (*JourneyDraft, error)
	ListJourneyDrafts(ctx context.Context, tenantID string) ([]JourneyDraft, error)
	UpdateJourneyDraft(ctx context.Context, d *JourneyDraft) (*JourneyDraft, error)
	DeleteJourneyDraft(ctx context.Context, tenantID, draftID string) error

	// 3. Journey Versions
	CreateJourneyVersion(ctx context.Context, v *JourneyVersion) (*JourneyVersion, error)
	GetJourneyVersion(ctx context.Context, tenantID, versionID string) (*JourneyVersion, error)
	GetJourneyVersionByDraftAndVersion(ctx context.Context, tenantID, draftID string, version int32) (*JourneyVersion, error)
	ListJourneyVersionsByDraft(ctx context.Context, tenantID, draftID string) ([]JourneyVersion, error)

	// 4. Idempotency Keys
	CreateIdempotencyKey(ctx context.Context, ik *IdempotencyKey) (*IdempotencyKey, error)
	GetIdempotencyKey(ctx context.Context, tenantID, scope, key string) (*IdempotencyKey, error)
	UpdateIdempotencyKey(ctx context.Context, ik *IdempotencyKey) (*IdempotencyKey, error)

	// 5. Kafka Inbox
	SaveKafkaInboxMessage(ctx context.Context, msg *KafkaInbox) (*KafkaInbox, error)
	GetKafkaInboxMessage(ctx context.Context, tenantID, messageID string) (*KafkaInbox, error)
	MarkKafkaInboxProcessed(ctx context.Context, tenantID, messageID string, processedAt time.Time) (*KafkaInbox, error)

	// 6. Target Manifests
	CreateTargetManifest(ctx context.Context, m *TargetManifest) (*TargetManifest, error)
	GetTargetManifest(ctx context.Context, tenantID, manifestID string) (*TargetManifest, error)

	// 7. Dispatch Ledger
	CreateDispatchLedger(ctx context.Context, dl *DispatchLedger) (*DispatchLedger, error)
	GetDispatchLedger(ctx context.Context, tenantID, dispatchID string) (*DispatchLedger, error)

	// 8. Enrollments
	CreateEnrollment(ctx context.Context, e *Enrollment) (*Enrollment, error)
	GetEnrollment(ctx context.Context, tenantID, enrollmentID string) (*Enrollment, error)
	ListEnrollments(ctx context.Context, tenantID string) ([]Enrollment, error)
	UpdateEnrollmentStatus(ctx context.Context, tenantID, enrollmentID, status, currentNodeID string, stateData []byte, completedAt *time.Time) (*Enrollment, error)

	// 9. Subscriptions
	CreateSubscription(ctx context.Context, s *Subscription) (*Subscription, error)
	ListSubscriptionsByEnrollment(ctx context.Context, tenantID, enrollmentID string) ([]Subscription, error)

	// 10. Action Ledger
	CreateActionLedger(ctx context.Context, al *ActionLedger) (*ActionLedger, error)
	GetActionLedger(ctx context.Context, tenantID, actionID string) (*ActionLedger, error)
	UpdateActionLedgerStatus(ctx context.Context, tenantID, actionID, status string, output []byte, errMsg string, durationMS int64, completedAt *time.Time) (*ActionLedger, error)

	// 11. Experiment Definitions
	CreateExperimentDefinition(ctx context.Context, exp *ExperimentDefinition) (*ExperimentDefinition, error)
	GetExperimentDefinition(ctx context.Context, tenantID, experimentID string) (*ExperimentDefinition, error)
	ListExperimentDefinitions(ctx context.Context, tenantID string) ([]ExperimentDefinition, error)
	UpdateExperimentDefinition(ctx context.Context, exp *ExperimentDefinition) (*ExperimentDefinition, error)

	// 12. Assignments
	CreateAssignment(ctx context.Context, a *Assignment) (*Assignment, error)
	GetAssignment(ctx context.Context, tenantID, assignmentID string) (*Assignment, error)
	GetAssignmentByExperimentSubject(ctx context.Context, tenantID, experimentID, subjectID string) (*Assignment, error)

	// 13. Exposures
	CreateExposure(ctx context.Context, ex *Exposure) (*Exposure, error)
	GetExposure(ctx context.Context, tenantID, exposureID string) (*Exposure, error)

	// 14. Outbox
	CreateOutboxEvent(ctx context.Context, o *Outbox) (*Outbox, error)
	GetOutboxEvent(ctx context.Context, tenantID, id string) (*Outbox, error)
	ListPendingOutboxEvents(ctx context.Context, tenantID string, limit int) ([]Outbox, error)
	MarkOutboxProcessed(ctx context.Context, tenantID, id string, processedAt time.Time) (*Outbox, error)

	// 15. Static Lists
	CreateStaticList(ctx context.Context, l *StaticList) (*StaticList, error)
	GetStaticList(ctx context.Context, tenantID, listID string) (*StaticList, error)
	ListStaticLists(ctx context.Context, tenantID string) ([]StaticList, error)
	UpdateStaticList(ctx context.Context, l *StaticList) (*StaticList, error)
	DeleteStaticList(ctx context.Context, tenantID, listID string) error

	// 16. Test Runs
	CreateTestRun(ctx context.Context, tr *TestRun) (*TestRun, error)
	GetTestRun(ctx context.Context, tenantID, testRunID string) (*TestRun, error)
	ListTestRuns(ctx context.Context, tenantID string) ([]TestRun, error)
	UpdateTestRun(ctx context.Context, tenantID, testRunID, status string, actualOutcomes []byte, durationMS int64) (*TestRun, error)

	// 17. Lifecycle Events
	RecordLifecycleEvent(ctx context.Context, le *LifecycleEvent) (*LifecycleEvent, error)
	ListLifecycleEventsByEntity(ctx context.Context, tenantID, entityType, entityID string) ([]LifecycleEvent, error)

	// 18. Tombstones
	CreateTombstone(ctx context.Context, t *Tombstone) (*Tombstone, error)
	GetTombstone(ctx context.Context, tenantID, tombstoneID string) (*Tombstone, error)
	GetTombstoneByEntity(ctx context.Context, tenantID, entityType, entityID string) (*Tombstone, error)
}

// MemoryRepository implements Repository interface in-memory with complete transaction & constraint support for testing.
type MemoryRepository struct {
	mu sync.RWMutex

	catalogs           map[string]Catalog
	journeyDrafts      map[string]JourneyDraft
	journeyVersions    map[string]JourneyVersion
	idempotencyKeys    map[string]IdempotencyKey
	kafkaInbox         map[string]KafkaInbox
	targetManifests    map[string]TargetManifest
	dispatchLedger     map[string]DispatchLedger
	enrollments        map[string]Enrollment
	subscriptions      map[string]Subscription
	actionLedger       map[string]ActionLedger
	experiments        map[string]ExperimentDefinition
	assignments        map[string]Assignment
	exposures          map[string]Exposure
	outbox             map[string]Outbox
	staticLists        map[string]StaticList
	testRuns           map[string]TestRun
	lifecycleEvents    map[string]LifecycleEvent
	tombstones         map[string]Tombstone

	// Unique constraint indexes
	journeyVersionDraftIdx map[string]string // tenantID:draftID:version -> versionID
	kafkaInboxOffsetIdx    map[string]string // tenantID:topic:partition:offset -> messageID
	dispatchBoundaryIdx    map[string]string // tenantID:manifestID:subjectID:journeyVersionID -> dispatchID
	enrollmentSubjectIdx   map[string]string // tenantID:journeyVersionID:subjectID -> enrollmentID
	subscriptionEventIdx   map[string]string // tenantID:enrollmentID:eventType -> subscriptionID
	actionNodeIdx          map[string]string // tenantID:enrollmentID:nodeID -> actionID
	assignmentSubjectIdx   map[string]string // tenantID:experimentID:subjectID -> assignmentID
	exposureSubjectIdx     map[string]string // tenantID:experimentID:subjectID:assignmentID -> exposureID
	tombstoneEntityIdx     map[string]string // tenantID:entityType:entityID -> tombstoneID
}

// NewMemoryRepository creates a new thread-safe in-memory PostgreSQL repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		catalogs:               make(map[string]Catalog),
		journeyDrafts:          make(map[string]JourneyDraft),
		journeyVersions:        make(map[string]JourneyVersion),
		idempotencyKeys:        make(map[string]IdempotencyKey),
		kafkaInbox:             make(map[string]KafkaInbox),
		targetManifests:        make(map[string]TargetManifest),
		dispatchLedger:         make(map[string]DispatchLedger),
		enrollments:            make(map[string]Enrollment),
		subscriptions:          make(map[string]Subscription),
		actionLedger:           make(map[string]ActionLedger),
		experiments:            make(map[string]ExperimentDefinition),
		assignments:            make(map[string]Assignment),
		exposures:              make(map[string]Exposure),
		outbox:                 make(map[string]Outbox),
		staticLists:            make(map[string]StaticList),
		testRuns:               make(map[string]TestRun),
		lifecycleEvents:        make(map[string]LifecycleEvent),
		tombstones:             make(map[string]Tombstone),
		journeyVersionDraftIdx: make(map[string]string),
		kafkaInboxOffsetIdx:    make(map[string]string),
		dispatchBoundaryIdx:    make(map[string]string),
		enrollmentSubjectIdx:   make(map[string]string),
		subscriptionEventIdx:   make(map[string]string),
		actionNodeIdx:          make(map[string]string),
		assignmentSubjectIdx:   make(map[string]string),
		exposureSubjectIdx:     make(map[string]string),
		tombstoneEntityIdx:     make(map[string]string),
	}
}

func (m *MemoryRepository) clone() *MemoryRepository {
	cp := NewMemoryRepository()
	for k, v := range m.catalogs {
		cp.catalogs[k] = v
	}
	for k, v := range m.journeyDrafts {
		cp.journeyDrafts[k] = v
	}
	for k, v := range m.journeyVersions {
		cp.journeyVersions[k] = v
	}
	for k, v := range m.idempotencyKeys {
		cp.idempotencyKeys[k] = v
	}
	for k, v := range m.kafkaInbox {
		cp.kafkaInbox[k] = v
	}
	for k, v := range m.targetManifests {
		cp.targetManifests[k] = v
	}
	for k, v := range m.dispatchLedger {
		cp.dispatchLedger[k] = v
	}
	for k, v := range m.enrollments {
		cp.enrollments[k] = v
	}
	for k, v := range m.subscriptions {
		cp.subscriptions[k] = v
	}
	for k, v := range m.actionLedger {
		cp.actionLedger[k] = v
	}
	for k, v := range m.experiments {
		cp.experiments[k] = v
	}
	for k, v := range m.assignments {
		cp.assignments[k] = v
	}
	for k, v := range m.exposures {
		cp.exposures[k] = v
	}
	for k, v := range m.outbox {
		cp.outbox[k] = v
	}
	for k, v := range m.staticLists {
		cp.staticLists[k] = v
	}
	for k, v := range m.testRuns {
		cp.testRuns[k] = v
	}
	for k, v := range m.lifecycleEvents {
		cp.lifecycleEvents[k] = v
	}
	for k, v := range m.tombstones {
		cp.tombstones[k] = v
	}
	for k, v := range m.journeyVersionDraftIdx {
		cp.journeyVersionDraftIdx[k] = v
	}
	for k, v := range m.kafkaInboxOffsetIdx {
		cp.kafkaInboxOffsetIdx[k] = v
	}
	for k, v := range m.dispatchBoundaryIdx {
		cp.dispatchBoundaryIdx[k] = v
	}
	for k, v := range m.enrollmentSubjectIdx {
		cp.enrollmentSubjectIdx[k] = v
	}
	for k, v := range m.subscriptionEventIdx {
		cp.subscriptionEventIdx[k] = v
	}
	for k, v := range m.actionNodeIdx {
		cp.actionNodeIdx[k] = v
	}
	for k, v := range m.assignmentSubjectIdx {
		cp.assignmentSubjectIdx[k] = v
	}
	for k, v := range m.exposureSubjectIdx {
		cp.exposureSubjectIdx[k] = v
	}
	for k, v := range m.tombstoneEntityIdx {
		cp.tombstoneEntityIdx[k] = v
	}
	return cp
}

// WithTx simulates a transactional block. If fn returns an error, changes are rolled back.
func (m *MemoryRepository) WithTx(ctx context.Context, fn func(repo Repository) error) error {
	m.mu.Lock()
	txStore := m.clone()
	m.mu.Unlock()

	err := fn(txStore)
	if err != nil {
		return err // Rollback: discard txStore changes
	}

	// Commit: merge txStore state back into m under write lock
	m.mu.Lock()
	defer m.mu.Unlock()
	m.catalogs = txStore.catalogs
	m.journeyDrafts = txStore.journeyDrafts
	m.journeyVersions = txStore.journeyVersions
	m.idempotencyKeys = txStore.idempotencyKeys
	m.kafkaInbox = txStore.kafkaInbox
	m.targetManifests = txStore.targetManifests
	m.dispatchLedger = txStore.dispatchLedger
	m.enrollments = txStore.enrollments
	m.subscriptions = txStore.subscriptions
	m.actionLedger = txStore.actionLedger
	m.experiments = txStore.experiments
	m.assignments = txStore.assignments
	m.exposures = txStore.exposures
	m.outbox = txStore.outbox
	m.staticLists = txStore.staticLists
	m.testRuns = txStore.testRuns
	m.lifecycleEvents = txStore.lifecycleEvents
	m.tombstones = txStore.tombstones

	m.journeyVersionDraftIdx = txStore.journeyVersionDraftIdx
	m.kafkaInboxOffsetIdx = txStore.kafkaInboxOffsetIdx
	m.dispatchBoundaryIdx = txStore.dispatchBoundaryIdx
	m.enrollmentSubjectIdx = txStore.enrollmentSubjectIdx
	m.subscriptionEventIdx = txStore.subscriptionEventIdx
	m.actionNodeIdx = txStore.actionNodeIdx
	m.assignmentSubjectIdx = txStore.assignmentSubjectIdx
	m.exposureSubjectIdx = txStore.exposureSubjectIdx
	m.tombstoneEntityIdx = txStore.tombstoneEntityIdx

	return nil
}

// 1. Catalogs
func (m *MemoryRepository) CreateCatalog(ctx context.Context, c *Catalog) (*Catalog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", c.TenantID, c.RecordID)
	if _, exists := m.catalogs[key]; exists {
		return nil, fmt.Errorf("%w: catalog %s", ErrAlreadyExists, c.RecordID)
	}
	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	item := *c
	m.catalogs[key] = item
	return &item, nil
}

func (m *MemoryRepository) GetCatalog(ctx context.Context, tenantID, recordID string) (*Catalog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", tenantID, recordID)
	c, exists := m.catalogs[key]
	if !exists {
		return nil, ErrNotFound
	}
	return &c, nil
}

func (m *MemoryRepository) ListCatalogs(ctx context.Context, tenantID string) ([]Catalog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var res []Catalog
	for _, c := range m.catalogs {
		if c.TenantID == tenantID {
			res = append(res, c)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].CreatedAt.After(res[j].CreatedAt)
	})
	return res, nil
}

func (m *MemoryRepository) UpdateCatalog(ctx context.Context, c *Catalog) (*Catalog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", c.TenantID, c.RecordID)
	if _, exists := m.catalogs[key]; !exists {
		return nil, ErrNotFound
	}
	c.UpdatedAt = time.Now().UTC()
	item := *c
	m.catalogs[key] = item
	return &item, nil
}

func (m *MemoryRepository) DeleteCatalog(ctx context.Context, tenantID, recordID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", tenantID, recordID)
	if _, exists := m.catalogs[key]; !exists {
		return ErrNotFound
	}
	delete(m.catalogs, key)
	return nil
}

// 2. Journey Drafts
func (m *MemoryRepository) CreateJourneyDraft(ctx context.Context, d *JourneyDraft) (*JourneyDraft, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", d.TenantID, d.DraftID)
	if _, exists := m.journeyDrafts[key]; exists {
		return nil, fmt.Errorf("%w: draft %s", ErrAlreadyExists, d.DraftID)
	}
	now := time.Now().UTC()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = now
	}
	item := *d
	m.journeyDrafts[key] = item
	return &item, nil
}

func (m *MemoryRepository) GetJourneyDraft(ctx context.Context, tenantID, draftID string) (*JourneyDraft, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", tenantID, draftID)
	d, exists := m.journeyDrafts[key]
	if !exists {
		return nil, ErrNotFound
	}
	return &d, nil
}

func (m *MemoryRepository) ListJourneyDrafts(ctx context.Context, tenantID string) ([]JourneyDraft, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var res []JourneyDraft
	for _, d := range m.journeyDrafts {
		if d.TenantID == tenantID {
			res = append(res, d)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].UpdatedAt.After(res[j].UpdatedAt)
	})
	return res, nil
}

func (m *MemoryRepository) UpdateJourneyDraft(ctx context.Context, d *JourneyDraft) (*JourneyDraft, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", d.TenantID, d.DraftID)
	if _, exists := m.journeyDrafts[key]; !exists {
		return nil, ErrNotFound
	}
	d.UpdatedAt = time.Now().UTC()
	item := *d
	m.journeyDrafts[key] = item
	return &item, nil
}

func (m *MemoryRepository) DeleteJourneyDraft(ctx context.Context, tenantID, draftID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", tenantID, draftID)
	if _, exists := m.journeyDrafts[key]; !exists {
		return ErrNotFound
	}
	delete(m.journeyDrafts, key)
	return nil
}

// 3. Journey Versions
func (m *MemoryRepository) CreateJourneyVersion(ctx context.Context, v *JourneyVersion) (*JourneyVersion, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pk := fmt.Sprintf("%s:%s", v.TenantID, v.VersionID)
	if _, exists := m.journeyVersions[pk]; exists {
		return nil, fmt.Errorf("%w: journey version %s", ErrAlreadyExists, v.VersionID)
	}

	uk := fmt.Sprintf("%s:%s:%d", v.TenantID, v.DraftID, v.Version)
	if _, exists := m.journeyVersionDraftIdx[uk]; exists {
		return nil, fmt.Errorf("%w: version %d for draft %s", ErrConflict, v.Version, v.DraftID)
	}

	now := time.Now().UTC()
	if v.CompiledAt.IsZero() {
		v.CompiledAt = now
	}
	if v.CreatedAt.IsZero() {
		v.CreatedAt = now
	}
	item := *v
	m.journeyVersions[pk] = item
	m.journeyVersionDraftIdx[uk] = v.VersionID
	return &item, nil
}

func (m *MemoryRepository) GetJourneyVersion(ctx context.Context, tenantID, versionID string) (*JourneyVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pk := fmt.Sprintf("%s:%s", tenantID, versionID)
	v, exists := m.journeyVersions[pk]
	if !exists {
		return nil, ErrNotFound
	}
	return &v, nil
}

func (m *MemoryRepository) GetJourneyVersionByDraftAndVersion(ctx context.Context, tenantID, draftID string, version int32) (*JourneyVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uk := fmt.Sprintf("%s:%s:%d", tenantID, draftID, version)
	vID, exists := m.journeyVersionDraftIdx[uk]
	if !exists {
		return nil, ErrNotFound
	}
	pk := fmt.Sprintf("%s:%s", tenantID, vID)
	v := m.journeyVersions[pk]
	return &v, nil
}

func (m *MemoryRepository) ListJourneyVersionsByDraft(ctx context.Context, tenantID, draftID string) ([]JourneyVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var res []JourneyVersion
	for _, v := range m.journeyVersions {
		if v.TenantID == tenantID && v.DraftID == draftID {
			res = append(res, v)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Version > res[j].Version
	})
	return res, nil
}

// 4. Idempotency Keys
func (m *MemoryRepository) CreateIdempotencyKey(ctx context.Context, ik *IdempotencyKey) (*IdempotencyKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%s", ik.TenantID, ik.Scope, ik.Key)
	if _, exists := m.idempotencyKeys[key]; exists {
		return nil, fmt.Errorf("%w: idempotency key %s", ErrConflict, ik.Key)
	}
	now := time.Now().UTC()
	if ik.CreatedAt.IsZero() {
		ik.CreatedAt = now
	}
	if ik.UpdatedAt.IsZero() {
		ik.UpdatedAt = now
	}
	item := *ik
	m.idempotencyKeys[key] = item
	return &item, nil
}

func (m *MemoryRepository) GetIdempotencyKey(ctx context.Context, tenantID, scope, key string) (*IdempotencyKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	k := fmt.Sprintf("%s:%s:%s", tenantID, scope, key)
	ik, exists := m.idempotencyKeys[k]
	if !exists {
		return nil, ErrNotFound
	}
	return &ik, nil
}

func (m *MemoryRepository) UpdateIdempotencyKey(ctx context.Context, ik *IdempotencyKey) (*IdempotencyKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%s", ik.TenantID, ik.Scope, ik.Key)
	if _, exists := m.idempotencyKeys[key]; !exists {
		return nil, ErrNotFound
	}
	ik.UpdatedAt = time.Now().UTC()
	item := *ik
	m.idempotencyKeys[key] = item
	return &item, nil
}

// 5. Kafka Inbox
func (m *MemoryRepository) SaveKafkaInboxMessage(ctx context.Context, msg *KafkaInbox) (*KafkaInbox, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pk := fmt.Sprintf("%s:%s", msg.TenantID, msg.MessageID)
	if _, exists := m.kafkaInbox[pk]; exists {
		return nil, fmt.Errorf("%w: message %s", ErrAlreadyExists, msg.MessageID)
	}

	uk := fmt.Sprintf("%s:%s:%d:%d", msg.TenantID, msg.Topic, msg.Partition, msg.OffsetVal)
	if _, exists := m.kafkaInboxOffsetIdx[uk]; exists {
		return nil, fmt.Errorf("%w: topic %s partition %d offset %d", ErrConflict, msg.Topic, msg.Partition, msg.OffsetVal)
	}

	now := time.Now().UTC()
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = now
	}
	item := *msg
	m.kafkaInbox[pk] = item
	m.kafkaInboxOffsetIdx[uk] = msg.MessageID
	return &item, nil
}

func (m *MemoryRepository) GetKafkaInboxMessage(ctx context.Context, tenantID, messageID string) (*KafkaInbox, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pk := fmt.Sprintf("%s:%s", tenantID, messageID)
	msg, exists := m.kafkaInbox[pk]
	if !exists {
		return nil, ErrNotFound
	}
	return &msg, nil
}

func (m *MemoryRepository) MarkKafkaInboxProcessed(ctx context.Context, tenantID, messageID string, processedAt time.Time) (*KafkaInbox, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pk := fmt.Sprintf("%s:%s", tenantID, messageID)
	msg, exists := m.kafkaInbox[pk]
	if !exists {
		return nil, ErrNotFound
	}
	msg.Status = "processed"
	msg.ProcessedAt = &processedAt
	m.kafkaInbox[pk] = msg
	return &msg, nil
}

// 6. Target Manifests
func (m *MemoryRepository) CreateTargetManifest(ctx context.Context, tm *TargetManifest) (*TargetManifest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", tm.TenantID, tm.ManifestID)
	if _, exists := m.targetManifests[key]; exists {
		return nil, fmt.Errorf("%w: target manifest %s", ErrAlreadyExists, tm.ManifestID)
	}
	now := time.Now().UTC()
	if tm.CreatedAt.IsZero() {
		tm.CreatedAt = now
	}
	if tm.UpdatedAt.IsZero() {
		tm.UpdatedAt = now
	}
	item := *tm
	m.targetManifests[key] = item
	return &item, nil
}

func (m *MemoryRepository) GetTargetManifest(ctx context.Context, tenantID, manifestID string) (*TargetManifest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", tenantID, manifestID)
	tm, exists := m.targetManifests[key]
	if !exists {
		return nil, ErrNotFound
	}
	return &tm, nil
}

// 7. Dispatch Ledger
func (m *MemoryRepository) CreateDispatchLedger(ctx context.Context, dl *DispatchLedger) (*DispatchLedger, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pk := fmt.Sprintf("%s:%s", dl.TenantID, dl.DispatchID)
	if _, exists := m.dispatchLedger[pk]; exists {
		return nil, fmt.Errorf("%w: dispatch %s", ErrAlreadyExists, dl.DispatchID)
	}

	uk := fmt.Sprintf("%s:%s:%s:%s", dl.TenantID, dl.ManifestID, dl.SubjectID, dl.JourneyVersionID)
	if _, exists := m.dispatchBoundaryIdx[uk]; exists {
		return nil, fmt.Errorf("%w: duplicate dispatch boundary", ErrConflict)
	}

	now := time.Now().UTC()
	if dl.DispatchedAt.IsZero() {
		dl.DispatchedAt = now
	}
	if dl.CreatedAt.IsZero() {
		dl.CreatedAt = now
	}
	item := *dl
	m.dispatchLedger[pk] = item
	m.dispatchBoundaryIdx[uk] = dl.DispatchID
	return &item, nil
}

func (m *MemoryRepository) GetDispatchLedger(ctx context.Context, tenantID, dispatchID string) (*DispatchLedger, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pk := fmt.Sprintf("%s:%s", tenantID, dispatchID)
	dl, exists := m.dispatchLedger[pk]
	if !exists {
		return nil, ErrNotFound
	}
	return &dl, nil
}

// 8. Enrollments
func (m *MemoryRepository) CreateEnrollment(ctx context.Context, e *Enrollment) (*Enrollment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pk := fmt.Sprintf("%s:%s", e.TenantID, e.EnrollmentID)
	if _, exists := m.enrollments[pk]; exists {
		return nil, fmt.Errorf("%w: enrollment %s", ErrAlreadyExists, e.EnrollmentID)
	}

	uk := fmt.Sprintf("%s:%s:%s", e.TenantID, e.JourneyVersionID, e.SubjectID)
	if _, exists := m.enrollmentSubjectIdx[uk]; exists {
		return nil, fmt.Errorf("%w: subject %s already enrolled in journey version %s", ErrConflict, e.SubjectID, e.JourneyVersionID)
	}

	now := time.Now().UTC()
	if e.EnrolledAt.IsZero() {
		e.EnrolledAt = now
	}
	if e.UpdatedAt.IsZero() {
		e.UpdatedAt = now
	}
	item := *e
	m.enrollments[pk] = item
	m.enrollmentSubjectIdx[uk] = e.EnrollmentID
	return &item, nil
}

func (m *MemoryRepository) GetEnrollment(ctx context.Context, tenantID, enrollmentID string) (*Enrollment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pk := fmt.Sprintf("%s:%s", tenantID, enrollmentID)
	e, exists := m.enrollments[pk]
	if !exists {
		return nil, ErrNotFound
	}
	return &e, nil
}

func (m *MemoryRepository) ListEnrollments(ctx context.Context, tenantID string) ([]Enrollment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var res []Enrollment
	for _, item := range m.enrollments {
		if item.TenantID == tenantID {
			res = append(res, item)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].EnrolledAt.After(res[j].EnrolledAt)
	})
	return res, nil
}

func (m *MemoryRepository) UpdateEnrollmentStatus(ctx context.Context, tenantID, enrollmentID, status, currentNodeID string, stateData []byte, completedAt *time.Time) (*Enrollment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pk := fmt.Sprintf("%s:%s", tenantID, enrollmentID)
	e, exists := m.enrollments[pk]
	if !exists {
		return nil, ErrNotFound
	}
	e.Status = status
	if currentNodeID != "" {
		e.CurrentNodeID = currentNodeID
	}
	if len(stateData) > 0 {
		e.StateData = stateData
	}
	e.UpdatedAt = time.Now().UTC()
	if completedAt != nil {
		e.CompletedAt = completedAt
	}
	m.enrollments[pk] = e
	return &e, nil
}

// 9. Subscriptions
func (m *MemoryRepository) CreateSubscription(ctx context.Context, s *Subscription) (*Subscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pk := fmt.Sprintf("%s:%s", s.TenantID, s.SubscriptionID)
	if _, exists := m.subscriptions[pk]; exists {
		return nil, fmt.Errorf("%w: subscription %s", ErrAlreadyExists, s.SubscriptionID)
	}

	uk := fmt.Sprintf("%s:%s:%s", s.TenantID, s.EnrollmentID, s.EventType)
	if _, exists := m.subscriptionEventIdx[uk]; exists {
		return nil, fmt.Errorf("%w: active subscription for event %s on enrollment %s", ErrConflict, s.EventType, s.EnrollmentID)
	}

	now := time.Now().UTC()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = now
	}
	item := *s
	m.subscriptions[pk] = item
	m.subscriptionEventIdx[uk] = s.SubscriptionID
	return &item, nil
}

func (m *MemoryRepository) ListSubscriptionsByEnrollment(ctx context.Context, tenantID, enrollmentID string) ([]Subscription, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var res []Subscription
	for _, s := range m.subscriptions {
		if s.TenantID == tenantID && s.EnrollmentID == enrollmentID && s.Status == "active" {
			res = append(res, s)
		}
	}
	return res, nil
}

// 10. Action Ledger
func (m *MemoryRepository) CreateActionLedger(ctx context.Context, al *ActionLedger) (*ActionLedger, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pk := fmt.Sprintf("%s:%s", al.TenantID, al.ActionID)
	if _, exists := m.actionLedger[pk]; exists {
		return nil, fmt.Errorf("%w: action %s", ErrAlreadyExists, al.ActionID)
	}

	uk := fmt.Sprintf("%s:%s:%s", al.TenantID, al.EnrollmentID, al.NodeID)
	if _, exists := m.actionNodeIdx[uk]; exists {
		return nil, fmt.Errorf("%w: action for node %s on enrollment %s", ErrConflict, al.NodeID, al.EnrollmentID)
	}

	now := time.Now().UTC()
	if al.CreatedAt.IsZero() {
		al.CreatedAt = now
	}
	item := *al
	m.actionLedger[pk] = item
	m.actionNodeIdx[uk] = al.ActionID
	return &item, nil
}

func (m *MemoryRepository) GetActionLedger(ctx context.Context, tenantID, actionID string) (*ActionLedger, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pk := fmt.Sprintf("%s:%s", tenantID, actionID)
	al, exists := m.actionLedger[pk]
	if !exists {
		return nil, ErrNotFound
	}
	return &al, nil
}

func (m *MemoryRepository) UpdateActionLedgerStatus(ctx context.Context, tenantID, actionID, status string, output []byte, errMsg string, durationMS int64, completedAt *time.Time) (*ActionLedger, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pk := fmt.Sprintf("%s:%s", tenantID, actionID)
	al, exists := m.actionLedger[pk]
	if !exists {
		return nil, ErrNotFound
	}
	al.Status = status
	if len(output) > 0 {
		al.Output = output
	}
	if errMsg != "" {
		al.ErrorMessage = errMsg
	}
	if durationMS > 0 {
		al.ExecutionDurationMS = durationMS
	}
	if completedAt != nil {
		al.CompletedAt = completedAt
	}
	m.actionLedger[pk] = al
	return &al, nil
}

// 11. Experiment Definitions
func (m *MemoryRepository) CreateExperimentDefinition(ctx context.Context, exp *ExperimentDefinition) (*ExperimentDefinition, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", exp.TenantID, exp.ExperimentID)
	if _, exists := m.experiments[key]; exists {
		return nil, fmt.Errorf("%w: experiment %s", ErrAlreadyExists, exp.ExperimentID)
	}
	now := time.Now().UTC()
	if exp.CreatedAt.IsZero() {
		exp.CreatedAt = now
	}
	if exp.UpdatedAt.IsZero() {
		exp.UpdatedAt = now
	}
	item := *exp
	m.experiments[key] = item
	return &item, nil
}

func (m *MemoryRepository) GetExperimentDefinition(ctx context.Context, tenantID, experimentID string) (*ExperimentDefinition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", tenantID, experimentID)
	exp, exists := m.experiments[key]
	if !exists {
		return nil, ErrNotFound
	}
	return &exp, nil
}

func (m *MemoryRepository) ListExperimentDefinitions(ctx context.Context, tenantID string) ([]ExperimentDefinition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var res []ExperimentDefinition
	for _, item := range m.experiments {
		if item.TenantID == tenantID {
			res = append(res, item)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].CreatedAt.After(res[j].CreatedAt)
	})
	return res, nil
}

func (m *MemoryRepository) UpdateExperimentDefinition(ctx context.Context, exp *ExperimentDefinition) (*ExperimentDefinition, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", exp.TenantID, exp.ExperimentID)
	if _, exists := m.experiments[key]; !exists {
		return nil, ErrNotFound
	}
	exp.UpdatedAt = time.Now().UTC()
	item := *exp
	m.experiments[key] = item
	return &item, nil
}

// 12. Assignments
func (m *MemoryRepository) CreateAssignment(ctx context.Context, a *Assignment) (*Assignment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pk := fmt.Sprintf("%s:%s", a.TenantID, a.AssignmentID)
	if _, exists := m.assignments[pk]; exists {
		return nil, fmt.Errorf("%w: assignment %s", ErrAlreadyExists, a.AssignmentID)
	}

	uk := fmt.Sprintf("%s:%s:%s", a.TenantID, a.ExperimentID, a.SubjectID)
	if _, exists := m.assignmentSubjectIdx[uk]; exists {
		return nil, fmt.Errorf("%w: subject %s already assigned in experiment %s", ErrConflict, a.SubjectID, a.ExperimentID)
	}

	now := time.Now().UTC()
	if a.AssignedAt.IsZero() {
		a.AssignedAt = now
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	item := *a
	m.assignments[pk] = item
	m.assignmentSubjectIdx[uk] = a.AssignmentID
	return &item, nil
}

func (m *MemoryRepository) GetAssignment(ctx context.Context, tenantID, assignmentID string) (*Assignment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pk := fmt.Sprintf("%s:%s", tenantID, assignmentID)
	a, exists := m.assignments[pk]
	if !exists {
		return nil, ErrNotFound
	}
	return &a, nil
}

func (m *MemoryRepository) GetAssignmentByExperimentSubject(ctx context.Context, tenantID, experimentID, subjectID string) (*Assignment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uk := fmt.Sprintf("%s:%s:%s", tenantID, experimentID, subjectID)
	aID, exists := m.assignmentSubjectIdx[uk]
	if !exists {
		return nil, ErrNotFound
	}
	pk := fmt.Sprintf("%s:%s", tenantID, aID)
	a := m.assignments[pk]
	return &a, nil
}

// 13. Exposures
func (m *MemoryRepository) CreateExposure(ctx context.Context, ex *Exposure) (*Exposure, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pk := fmt.Sprintf("%s:%s", ex.TenantID, ex.ExposureID)
	if _, exists := m.exposures[pk]; exists {
		return nil, fmt.Errorf("%w: exposure %s", ErrAlreadyExists, ex.ExposureID)
	}

	uk := fmt.Sprintf("%s:%s:%s:%s", ex.TenantID, ex.ExperimentID, ex.SubjectID, ex.AssignmentID)
	if _, exists := m.exposureSubjectIdx[uk]; exists {
		return nil, fmt.Errorf("%w: duplicate exposure for assignment %s", ErrConflict, ex.AssignmentID)
	}

	now := time.Now().UTC()
	if ex.ExposedAt.IsZero() {
		ex.ExposedAt = now
	}
	if ex.CreatedAt.IsZero() {
		ex.CreatedAt = now
	}
	item := *ex
	m.exposures[pk] = item
	m.exposureSubjectIdx[uk] = ex.ExposureID
	return &item, nil
}

func (m *MemoryRepository) GetExposure(ctx context.Context, tenantID, exposureID string) (*Exposure, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pk := fmt.Sprintf("%s:%s", tenantID, exposureID)
	ex, exists := m.exposures[pk]
	if !exists {
		return nil, ErrNotFound
	}
	return &ex, nil
}

// 14. Outbox
func (m *MemoryRepository) CreateOutboxEvent(ctx context.Context, o *Outbox) (*Outbox, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", o.TenantID, o.ID)
	if _, exists := m.outbox[key]; exists {
		return nil, fmt.Errorf("%w: outbox event %s", ErrAlreadyExists, o.ID)
	}
	now := time.Now().UTC()
	if o.CreatedAt.IsZero() {
		o.CreatedAt = now
	}
	if o.Status == "" {
		o.Status = "pending"
	}
	item := *o
	m.outbox[key] = item
	return &item, nil
}

func (m *MemoryRepository) GetOutboxEvent(ctx context.Context, tenantID, id string) (*Outbox, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", tenantID, id)
	o, exists := m.outbox[key]
	if !exists {
		return nil, ErrNotFound
	}
	return &o, nil
}

func (m *MemoryRepository) ListPendingOutboxEvents(ctx context.Context, tenantID string, limit int) ([]Outbox, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var res []Outbox
	for _, o := range m.outbox {
		if o.TenantID == tenantID && o.Status == "pending" {
			res = append(res, o)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].CreatedAt.Before(res[j].CreatedAt)
	})
	if limit > 0 && len(res) > limit {
		res = res[:limit]
	}
	return res, nil
}

func (m *MemoryRepository) MarkOutboxProcessed(ctx context.Context, tenantID, id string, processedAt time.Time) (*Outbox, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", tenantID, id)
	o, exists := m.outbox[key]
	if !exists {
		return nil, ErrNotFound
	}
	o.Status = "processed"
	o.ProcessedAt = &processedAt
	m.outbox[key] = o
	return &o, nil
}

// 15. Static Lists
func (m *MemoryRepository) CreateStaticList(ctx context.Context, l *StaticList) (*StaticList, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", l.TenantID, l.ListID)
	if _, exists := m.staticLists[key]; exists {
		return nil, fmt.Errorf("%w: static list %s", ErrAlreadyExists, l.ListID)
	}
	now := time.Now().UTC()
	if l.CreatedAt.IsZero() {
		l.CreatedAt = now
	}
	if l.UpdatedAt.IsZero() {
		l.UpdatedAt = now
	}
	item := *l
	m.staticLists[key] = item
	return &item, nil
}

func (m *MemoryRepository) GetStaticList(ctx context.Context, tenantID, listID string) (*StaticList, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", tenantID, listID)
	l, exists := m.staticLists[key]
	if !exists {
		return nil, ErrNotFound
	}
	return &l, nil
}

func (m *MemoryRepository) ListStaticLists(ctx context.Context, tenantID string) ([]StaticList, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var res []StaticList
	for _, item := range m.staticLists {
		if item.TenantID == tenantID {
			res = append(res, item)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].CreatedAt.After(res[j].CreatedAt)
	})
	return res, nil
}

func (m *MemoryRepository) UpdateStaticList(ctx context.Context, l *StaticList) (*StaticList, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", l.TenantID, l.ListID)
	if _, exists := m.staticLists[key]; !exists {
		return nil, ErrNotFound
	}
	l.UpdatedAt = time.Now().UTC()
	item := *l
	m.staticLists[key] = item
	return &item, nil
}

func (m *MemoryRepository) DeleteStaticList(ctx context.Context, tenantID, listID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", tenantID, listID)
	if _, exists := m.staticLists[key]; !exists {
		return ErrNotFound
	}
	delete(m.staticLists, key)
	return nil
}

// 16. Test Runs
func (m *MemoryRepository) CreateTestRun(ctx context.Context, tr *TestRun) (*TestRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", tr.TenantID, tr.TestRunID)
	if _, exists := m.testRuns[key]; exists {
		return nil, fmt.Errorf("%w: test run %s", ErrAlreadyExists, tr.TestRunID)
	}
	now := time.Now().UTC()
	if tr.CreatedAt.IsZero() {
		tr.CreatedAt = now
	}
	if tr.UpdatedAt.IsZero() {
		tr.UpdatedAt = now
	}
	item := *tr
	m.testRuns[key] = item
	return &item, nil
}

func (m *MemoryRepository) GetTestRun(ctx context.Context, tenantID, testRunID string) (*TestRun, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", tenantID, testRunID)
	tr, exists := m.testRuns[key]
	if !exists {
		return nil, ErrNotFound
	}
	return &tr, nil
}

func (m *MemoryRepository) ListTestRuns(ctx context.Context, tenantID string) ([]TestRun, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var res []TestRun
	for _, item := range m.testRuns {
		if item.TenantID == tenantID {
			res = append(res, item)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].CreatedAt.After(res[j].CreatedAt)
	})
	return res, nil
}

func (m *MemoryRepository) UpdateTestRun(ctx context.Context, tenantID, testRunID, status string, actualOutcomes []byte, durationMS int64) (*TestRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", tenantID, testRunID)
	tr, exists := m.testRuns[key]
	if !exists {
		return nil, ErrNotFound
	}
	tr.Status = status
	if len(actualOutcomes) > 0 {
		tr.ActualOutcomes = actualOutcomes
	}
	if durationMS > 0 {
		tr.ExecutionTimeMS = durationMS
	}
	tr.UpdatedAt = time.Now().UTC()
	m.testRuns[key] = tr
	return &tr, nil
}

// 17. Lifecycle Events
func (m *MemoryRepository) RecordLifecycleEvent(ctx context.Context, le *LifecycleEvent) (*LifecycleEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", le.TenantID, le.EventID)
	if _, exists := m.lifecycleEvents[key]; exists {
		return nil, fmt.Errorf("%w: event %s", ErrAlreadyExists, le.EventID)
	}
	now := time.Now().UTC()
	if le.CreatedAt.IsZero() {
		le.CreatedAt = now
	}
	item := *le
	m.lifecycleEvents[key] = item
	return &item, nil
}

func (m *MemoryRepository) ListLifecycleEventsByEntity(ctx context.Context, tenantID, entityType, entityID string) ([]LifecycleEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var res []LifecycleEvent
	for _, le := range m.lifecycleEvents {
		if le.TenantID == tenantID && le.EntityType == entityType && le.EntityID == entityID {
			res = append(res, le)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].CreatedAt.Before(res[j].CreatedAt)
	})
	return res, nil
}

// 18. Tombstones
func (m *MemoryRepository) CreateTombstone(ctx context.Context, t *Tombstone) (*Tombstone, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pk := fmt.Sprintf("%s:%s", t.TenantID, t.TombstoneID)
	if _, exists := m.tombstones[pk]; exists {
		return nil, fmt.Errorf("%w: tombstone %s", ErrAlreadyExists, t.TombstoneID)
	}

	uk := fmt.Sprintf("%s:%s:%s", t.TenantID, t.EntityType, t.EntityID)
	if _, exists := m.tombstoneEntityIdx[uk]; exists {
		return nil, fmt.Errorf("%w: tombstone for entity %s of type %s", ErrConflict, t.EntityID, t.EntityType)
	}

	now := time.Now().UTC()
	if t.DeletedAt.IsZero() {
		t.DeletedAt = now
	}
	item := *t
	m.tombstones[pk] = item
	m.tombstoneEntityIdx[uk] = t.TombstoneID
	return &item, nil
}

func (m *MemoryRepository) GetTombstone(ctx context.Context, tenantID, tombstoneID string) (*Tombstone, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pk := fmt.Sprintf("%s:%s", tenantID, tombstoneID)
	t, exists := m.tombstones[pk]
	if !exists {
		return nil, ErrNotFound
	}
	return &t, nil
}

func (m *MemoryRepository) GetTombstoneByEntity(ctx context.Context, tenantID, entityType, entityID string) (*Tombstone, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uk := fmt.Sprintf("%s:%s:%s", tenantID, entityType, entityID)
	tID, exists := m.tombstoneEntityIdx[uk]
	if !exists {
		return nil, ErrNotFound
	}
	pk := fmt.Sprintf("%s:%s", tenantID, tID)
	t := m.tombstones[pk]
	return &t, nil
}
