package testaudience

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"
)

// AssignmentMode defines the assignment distribution mode for test runs.
type AssignmentMode string

const (
	AssignmentModeRealistic            AssignmentMode = "realistic"
	AssignmentModeForcedVariantCoverage AssignmentMode = "forced_variant_coverage"
)

// TestRunStatus defines execution status of test runs.
type TestRunStatus string

const (
	TestRunStatusPending   TestRunStatus = "pending"
	TestRunStatusRunning   TestRunStatus = "running"
	TestRunStatusCompleted TestRunStatus = "completed"
	TestRunStatusCancelled TestRunStatus = "cancelled"
	TestRunStatusExpired   TestRunStatus = "expired"
)

// TargetStatus defines state of individual frozen test targets.
type TargetStatus string

const (
	TargetStatusPending    TargetStatus = "pending"
	TargetStatusDispatched TargetStatus = "dispatched"
	TargetStatusFailed     TargetStatus = "failed"
	TargetStatusCancelled  TargetStatus = "cancelled"
	TargetStatusExpired    TargetStatus = "expired"
)

// Sentinel errors.
var (
	ErrInvalidInput      = errors.New("invalid test run input")
	ErrListNotFound      = errors.New("static list not found")
	ErrListExpired       = errors.New("static list is expired")
	ErrListDeleted       = errors.New("static list is deleted")
	ErrTestRunNotFound   = errors.New("test run not found")
	ErrTargetNotFound    = errors.New("test target not found")
	ErrTargetNotEligible = errors.New("test target is not eligible for dispatch")
	ErrTestRunTerminated = errors.New("test run is cancelled or expired")
	ErrHashMismatch      = errors.New("list version hash mismatch")
)

// TestRunInput contains immutable pinned parameters for initializing a test run.
type TestRunInput struct {
	TenantID        string                 `json:"tenant_id"`
	TestRunID       string                 `json:"test_run_id"`
	CandidateIRHash string                 `json:"candidate_ir_hash"`
	ListVersionHash string                 `json:"list_version_hash"`
	StaticListID    string                 `json:"static_list_id"`
	FixturePackHash string                 `json:"fixture_pack_hash"`
	AssignmentMode  AssignmentMode         `json:"assignment_mode"`
	FakeProvider    string                 `json:"fake_provider"`
	ExpiryDuration  time.Duration          `json:"expiry_duration"`
	Variants        []string               `json:"variants,omitempty"`
	MockInputs      map[string]interface{} `json:"mock_inputs,omitempty"`
}

// TestRunRecord represents an active or completed test run metadata record.
type TestRunRecord struct {
	TestRunID          string                 `json:"test_run_id"`
	TenantID           string                 `json:"tenant_id"`
	CandidateIRHash    string                 `json:"candidate_ir_hash"`
	ListVersionHash    string                 `json:"list_version_hash"`
	StaticListID       string                 `json:"static_list_id"`
	FixturePackHash    string                 `json:"fixture_pack_hash"`
	AssignmentMode     AssignmentMode         `json:"assignment_mode"`
	FakeProvider       string                 `json:"fake_provider"`
	Status             TestRunStatus          `json:"status"`
	IsTest             bool                   `json:"is_test"`
	TotalTargets       int                    `json:"total_targets"`
	DispatchedCount    int                    `json:"dispatched_count"`
	CancelledCount     int                    `json:"cancelled_count"`
	ExpiredCount       int                    `json:"expired_count"`
	CreatedAt          time.Time              `json:"created_at"`
	ExpiresAt          time.Time              `json:"expires_at"`
	CompletedAt        *time.Time             `json:"completed_at,omitempty"`
	CancellationReason string                 `json:"cancellation_reason,omitempty"`
	MockInputs         map[string]interface{} `json:"mock_inputs,omitempty"`
}

// TestTarget represents a frozen target derived directly from static list members.
type TestTarget struct {
	TargetID           string            `json:"target_id"`
	TestRunID          string            `json:"test_run_id"`
	TenantID           string            `json:"tenant_id"`
	MemberKey          string            `json:"member_key"`
	MemberID           string            `json:"member_id"`
	Recipient          string            `json:"recipient"`
	MaskedDisplayValue string            `json:"masked_display_value"`
	Attributes         map[string]string `json:"attributes,omitempty"`
	VariantID          string            `json:"variant_id"`
	Status             TargetStatus      `json:"status"`
	IsTest             bool              `json:"is_test"`
	CreatedAt          time.Time         `json:"created_at"`
	ExpiresAt          time.Time         `json:"expires_at"`
	DispatchedAt       *time.Time        `json:"dispatched_at,omitempty"`
}

// TestAssignment represents a sticky experiment assignment for a test subject.
type TestAssignment struct {
	AssignmentID    string         `json:"assignment_id"`
	TestRunID       string         `json:"test_run_id"`
	TenantID        string         `json:"tenant_id"`
	SubjectID       string         `json:"subject_id"`
	VariantID       string         `json:"variant_id"`
	CandidateIRHash string         `json:"candidate_ir_hash"`
	ListVersionHash string         `json:"list_version_hash"`
	FixturePackHash string         `json:"fixture_pack_hash"`
	AssignmentMode  AssignmentMode `json:"assignment_mode"`
	IsTest          bool           `json:"is_test"`
	AssignedAt      time.Time      `json:"assigned_at"`
}

// TestActionLedger records the execution of an action within a test run.
type TestActionLedger struct {
	LedgerID            string                 `json:"ledger_id"`
	TestRunID           string                 `json:"test_run_id"`
	TargetID            string                 `json:"target_id"`
	TenantID            string                 `json:"tenant_id"`
	NodeID              string                 `json:"node_id"`
	ActivityType        string                 `json:"activity_type"`
	FakeProvider        string                 `json:"fake_provider"`
	Status              string                 `json:"status"`
	Output              map[string]interface{} `json:"output,omitempty"`
	ErrorMessage        string                 `json:"error_message,omitempty"`
	ExecutionDurationMS int64                  `json:"execution_duration_ms"`
	IsTest              bool                   `json:"is_test"`
	ExecutedAt          time.Time              `json:"executed_at"`
}

// TestAnalyticsFact represents an analytical fact stored in the isolated test analytics partition.
type TestAnalyticsFact struct {
	FactID          string                 `json:"fact_id"`
	TestRunID       string                 `json:"test_run_id"`
	TenantID        string                 `json:"tenant_id"`
	EventType       string                 `json:"event_type"`
	SubjectID       string                 `json:"subject_id"`
	VariantID       string                 `json:"variant_id"`
	IsTest          bool                   `json:"is_test"`
	CandidateIRHash string                 `json:"candidate_ir_hash"`
	ListVersionHash string                 `json:"list_version_hash"`
	FixturePackHash string                 `json:"fixture_pack_hash"`
	OccurredAt      time.Time              `json:"occurred_at"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// TestRunDiagnostics provides a complete snapshot of test run state and diagnostic output.
type TestRunDiagnostics struct {
	Run         *TestRunRecord       `json:"run"`
	Targets     []*TestTarget        `json:"targets"`
	Assignments []*TestAssignment    `json:"assignments"`
	Actions     []*TestActionLedger  `json:"actions"`
	Analytics   []*TestAnalyticsFact `json:"analytics"`
}

// Storage interface abstracts isolated persistent or in-memory test data stores.
type Storage interface {
	SaveTestRun(ctx context.Context, run *TestRunRecord) error
	GetTestRun(ctx context.Context, testRunID string) (*TestRunRecord, bool, error)
	UpdateTestRun(ctx context.Context, run *TestRunRecord) error

	EnqueueTestTargets(ctx context.Context, targets []*TestTarget) error
	GetTestTargets(ctx context.Context, testRunID string) ([]*TestTarget, error)
	GetTestTarget(ctx context.Context, testRunID, targetID string) (*TestTarget, error)
	UpdateTestTarget(ctx context.Context, target *TestTarget) error

	SaveTestAssignment(ctx context.Context, assignment *TestAssignment) error
	GetTestAssignments(ctx context.Context, testRunID string) ([]*TestAssignment, error)

	RecordTestAction(ctx context.Context, ledger *TestActionLedger) error
	GetTestActions(ctx context.Context, testRunID string) ([]*TestActionLedger, error)

	RecordTestAnalytics(ctx context.Context, fact *TestAnalyticsFact) error
	GetTestAnalytics(ctx context.Context, testRunID string) ([]*TestAnalyticsFact, error)

	IsProductionTableTouched() bool
}

// InMemoryStorage implements Storage with strict isolation tracking.
type InMemoryStorage struct {
	mu                sync.RWMutex
	runs              map[string]*TestRunRecord
	targets           map[string][]*TestTarget
	assignments       map[string][]*TestAssignment
	ledgers           map[string][]*TestActionLedger
	facts             map[string][]*TestAnalyticsFact
	productionTouched bool
}

// NewInMemoryStorage creates a new isolated in-memory storage engine.
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		runs:        make(map[string]*TestRunRecord),
		targets:     make(map[string][]*TestTarget),
		assignments: make(map[string][]*TestAssignment),
		ledgers:     make(map[string][]*TestActionLedger),
		facts:       make(map[string][]*TestAnalyticsFact),
	}
}

func (s *InMemoryStorage) IsProductionTableTouched() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.productionTouched
}

func (s *InMemoryStorage) SaveTestRun(ctx context.Context, run *TestRunRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *run
	s.runs[run.TestRunID] = &cp
	return nil
}

func (s *InMemoryStorage) GetTestRun(ctx context.Context, testRunID string) (*TestRunRecord, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, found := s.runs[testRunID]
	if !found {
		return nil, false, nil
	}
	cp := *r
	return &cp, true, nil
}

func (s *InMemoryStorage) UpdateTestRun(ctx context.Context, run *TestRunRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, found := s.runs[run.TestRunID]; !found {
		return ErrTestRunNotFound
	}
	cp := *run
	s.runs[run.TestRunID] = &cp
	return nil
}

func (s *InMemoryStorage) EnqueueTestTargets(ctx context.Context, targets []*TestTarget) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, tgt := range targets {
		cp := *tgt
		s.targets[tgt.TestRunID] = append(s.targets[tgt.TestRunID], &cp)
	}
	return nil
}

func (s *InMemoryStorage) GetTestTargets(ctx context.Context, testRunID string) ([]*TestTarget, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tgts := s.targets[testRunID]
	res := make([]*TestTarget, len(tgts))
	for i, t := range tgts {
		cp := *t
		res[i] = &cp
	}
	return res, nil
}

func (s *InMemoryStorage) GetTestTarget(ctx context.Context, testRunID, targetID string) (*TestTarget, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, t := range s.targets[testRunID] {
		if t.TargetID == targetID {
			cp := *t
			return &cp, nil
		}
	}
	return nil, ErrTargetNotFound
}

func (s *InMemoryStorage) UpdateTestTarget(ctx context.Context, target *TestTarget) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tgts := s.targets[target.TestRunID]
	for i, t := range tgts {
		if t.TargetID == target.TargetID {
			cp := *target
			tgts[i] = &cp
			return nil
		}
	}
	return ErrTargetNotFound
}

func (s *InMemoryStorage) SaveTestAssignment(ctx context.Context, assignment *TestAssignment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *assignment
	s.assignments[assignment.TestRunID] = append(s.assignments[assignment.TestRunID], &cp)
	return nil
}

func (s *InMemoryStorage) GetTestAssignments(ctx context.Context, testRunID string) ([]*TestAssignment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	asgs := s.assignments[testRunID]
	res := make([]*TestAssignment, len(asgs))
	for i, a := range asgs {
		cp := *a
		res[i] = &cp
	}
	return res, nil
}

func (s *InMemoryStorage) RecordTestAction(ctx context.Context, ledger *TestActionLedger) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *ledger
	s.ledgers[ledger.TestRunID] = append(s.ledgers[ledger.TestRunID], &cp)
	return nil
}

func (s *InMemoryStorage) GetTestActions(ctx context.Context, testRunID string) ([]*TestActionLedger, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	leds := s.ledgers[testRunID]
	res := make([]*TestActionLedger, len(leds))
	for i, l := range leds {
		cp := *l
		res[i] = &cp
	}
	return res, nil
}

func (s *InMemoryStorage) RecordTestAnalytics(ctx context.Context, fact *TestAnalyticsFact) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *fact
	s.facts[fact.TestRunID] = append(s.facts[fact.TestRunID], &cp)
	return nil
}

func (s *InMemoryStorage) GetTestAnalytics(ctx context.Context, testRunID string) ([]*TestAnalyticsFact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	facts := s.facts[testRunID]
	res := make([]*TestAnalyticsFact, len(facts))
	for i, f := range facts {
		cp := *f
		res[i] = &cp
	}
	return res, nil
}

// Coordinator coordinates static-list test runs in full isolation from production workflows.
type Coordinator struct {
	evaluator *Evaluator
	store     Storage
}

// NewCoordinator creates a new static-list test-run coordinator.
func NewCoordinator(evaluator *Evaluator, store Storage) *Coordinator {
	if store == nil {
		store = NewInMemoryStorage()
	}
	return &Coordinator{
		evaluator: evaluator,
		store:     store,
	}
}

// StartTestRun initializes an isolated static-list test run with pinned parameters.
func (c *Coordinator) StartTestRun(ctx context.Context, input TestRunInput) (*TestRunRecord, []*TestTarget, error) {
	if input.TenantID == "" {
		return nil, nil, fmt.Errorf("%w: tenant_id is required", ErrInvalidInput)
	}
	if input.CandidateIRHash == "" {
		return nil, nil, fmt.Errorf("%w: candidate_ir_hash is required", ErrInvalidInput)
	}
	if input.StaticListID == "" {
		return nil, nil, fmt.Errorf("%w: static_list_id is required", ErrInvalidInput)
	}
	if input.FixturePackHash == "" {
		return nil, nil, fmt.Errorf("%w: fixture_pack_hash is required", ErrInvalidInput)
	}
	if input.TestRunID == "" {
		input.TestRunID = fmt.Sprintf("tr-%s-%d", input.StaticListID, time.Now().UnixNano())
	}
	if input.AssignmentMode == "" {
		input.AssignmentMode = AssignmentModeRealistic
	}
	if input.FakeProvider == "" {
		input.FakeProvider = "fake_default_provider"
	}
	if input.ExpiryDuration <= 0 {
		input.ExpiryDuration = 1 * time.Hour
	}
	if len(input.Variants) == 0 {
		input.Variants = []string{"control", "variant_a"}
	}

	// Idempotency check: if run already exists, return existing run and targets
	existingRun, found, err := c.store.GetTestRun(ctx, input.TestRunID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query test run: %w", err)
	}
	if found {
		targets, err := c.store.GetTestTargets(ctx, input.TestRunID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to query test targets: %w", err)
		}
		return existingRun, targets, nil
	}

	// Fetch static list from Evaluator
	list, ok := c.evaluator.GetList(input.StaticListID)
	if !ok {
		return nil, nil, fmt.Errorf("%w: list ID %s", ErrListNotFound, input.StaticListID)
	}
	now := time.Now()
	if list.IsDeleted {
		return nil, nil, fmt.Errorf("%w: list ID %s", ErrListDeleted, input.StaticListID)
	}
	if list.IsExpired(now) {
		return nil, nil, fmt.Errorf("%w: list ID %s", ErrListExpired, input.StaticListID)
	}

	listHash := list.ContentHash
	if listHash == "" {
		listHash = list.CalculateContentHash()
	}
	if input.ListVersionHash != "" && input.ListVersionHash != listHash {
		return nil, nil, fmt.Errorf("%w: expected %s, got %s", ErrHashMismatch, input.ListVersionHash, listHash)
	}
	input.ListVersionHash = listHash

	// Frozen Test Target Generator
	expiresAt := now.Add(input.ExpiryDuration)
	targets := make([]*TestTarget, 0, len(list.Members))
	assignments := make([]*TestAssignment, 0, len(list.Members))
	facts := make([]*TestAnalyticsFact, 0, len(list.Members))

	for i, m := range list.Members {
		var selectedVariant string
		if input.AssignmentMode == AssignmentModeForcedVariantCoverage {
			selectedVariant = input.Variants[i%len(input.Variants)]
		} else {
			selectedVariant = selectDeterministicVariant(m.MemberKey, input.CandidateIRHash, input.Variants)
		}

		targetID := fmt.Sprintf("tgt-%s-%s", input.TestRunID, m.MemberKey)
		tgt := &TestTarget{
			TargetID:           targetID,
			TestRunID:          input.TestRunID,
			TenantID:           input.TenantID,
			MemberKey:          m.MemberKey,
			MemberID:           m.MemberID,
			Recipient:          m.Recipient,
			MaskedDisplayValue: m.MaskedDisplayValue,
			Attributes:         m.Attributes,
			VariantID:          selectedVariant,
			Status:             TargetStatusPending,
			IsTest:             true,
			CreatedAt:          now,
			ExpiresAt:          expiresAt,
		}
		targets = append(targets, tgt)

		asgID := fmt.Sprintf("asg-test-%s-%s", input.TestRunID, m.MemberKey)
		asg := &TestAssignment{
			AssignmentID:    asgID,
			TestRunID:       input.TestRunID,
			TenantID:        input.TenantID,
			SubjectID:       m.MemberKey,
			VariantID:       selectedVariant,
			CandidateIRHash: input.CandidateIRHash,
			ListVersionHash: input.ListVersionHash,
			FixturePackHash: input.FixturePackHash,
			AssignmentMode:  input.AssignmentMode,
			IsTest:          true,
			AssignedAt:      now,
		}
		assignments = append(assignments, asg)

		factID := fmt.Sprintf("fact-asg-%s-%s", input.TestRunID, m.MemberKey)
		fact := &TestAnalyticsFact{
			FactID:          factID,
			TestRunID:       input.TestRunID,
			TenantID:        input.TenantID,
			EventType:       "test_assignment",
			SubjectID:       m.MemberKey,
			VariantID:       selectedVariant,
			IsTest:          true,
			CandidateIRHash: input.CandidateIRHash,
			ListVersionHash: input.ListVersionHash,
			FixturePackHash: input.FixturePackHash,
			OccurredAt:      now,
		}
		facts = append(facts, fact)
	}

	runRec := &TestRunRecord{
		TestRunID:       input.TestRunID,
		TenantID:        input.TenantID,
		CandidateIRHash: input.CandidateIRHash,
		ListVersionHash: input.ListVersionHash,
		StaticListID:    input.StaticListID,
		FixturePackHash: input.FixturePackHash,
		AssignmentMode:  input.AssignmentMode,
		FakeProvider:    input.FakeProvider,
		Status:          TestRunStatusRunning,
		IsTest:          true,
		TotalTargets:    len(targets),
		CreatedAt:       now,
		ExpiresAt:       expiresAt,
		MockInputs:      input.MockInputs,
	}

	// Write directly to test target queues, test assignment keys, and test analytical partitions
	if err := c.store.SaveTestRun(ctx, runRec); err != nil {
		return nil, nil, fmt.Errorf("failed to save test run: %w", err)
	}
	if err := c.store.EnqueueTestTargets(ctx, targets); err != nil {
		return nil, nil, fmt.Errorf("failed to enqueue test targets: %w", err)
	}
	for _, asg := range assignments {
		if err := c.store.SaveTestAssignment(ctx, asg); err != nil {
			return nil, nil, fmt.Errorf("failed to save test assignment: %w", err)
		}
	}
	for _, fact := range facts {
		if err := c.store.RecordTestAnalytics(ctx, fact); err != nil {
			return nil, nil, fmt.Errorf("failed to record test analytics fact: %w", err)
		}
	}

	return runRec, targets, nil
}

// DispatchTarget dispatches an individual target, recording test action ledgers and analytical facts.
func (c *Coordinator) DispatchTarget(ctx context.Context, testRunID, targetID, nodeID, activityType string) (*TestActionLedger, error) {
	run, found, err := c.store.GetTestRun(ctx, testRunID)
	if err != nil || !found {
		return nil, ErrTestRunNotFound
	}

	if run.Status == TestRunStatusCancelled || run.Status == TestRunStatusExpired {
		return nil, fmt.Errorf("%w: test run status is %s", ErrTestRunTerminated, run.Status)
	}

	now := time.Now()
	if now.After(run.ExpiresAt) || now.Equal(run.ExpiresAt) {
		c.HandleExpiry(ctx, testRunID, now)
		return nil, fmt.Errorf("%w: test run has expired", ErrTestRunTerminated)
	}

	target, err := c.store.GetTestTarget(ctx, testRunID, targetID)
	if err != nil {
		return nil, ErrTargetNotFound
	}
	if target.Status != TargetStatusPending {
		return nil, fmt.Errorf("%w: target status is %s", ErrTargetNotEligible, target.Status)
	}

	target.Status = TargetStatusDispatched
	target.DispatchedAt = &now
	if err := c.store.UpdateTestTarget(ctx, target); err != nil {
		return nil, fmt.Errorf("failed to update target: %w", err)
	}

	run.DispatchedCount++
	if run.DispatchedCount == run.TotalTargets {
		run.Status = TestRunStatusCompleted
		run.CompletedAt = &now
	}
	if err := c.store.UpdateTestRun(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to update test run: %w", err)
	}

	ledgerID := fmt.Sprintf("led-%s-%s", testRunID, target.MemberKey)
	ledger := &TestActionLedger{
		LedgerID:            ledgerID,
		TestRunID:           testRunID,
		TargetID:            targetID,
		TenantID:            target.TenantID,
		NodeID:              nodeID,
		ActivityType:        activityType,
		FakeProvider:        run.FakeProvider,
		Status:              "success",
		Output:              map[string]interface{}{"status": "delivered", "recipient": target.Recipient, "is_test": true},
		ExecutionDurationMS: 12,
		IsTest:              true,
		ExecutedAt:          now,
	}

	if err := c.store.RecordTestAction(ctx, ledger); err != nil {
		return nil, fmt.Errorf("failed to record action ledger: %w", err)
	}

	factID := fmt.Sprintf("fact-act-%s-%s", testRunID, target.MemberKey)
	fact := &TestAnalyticsFact{
		FactID:          factID,
		TestRunID:       testRunID,
		TenantID:        target.TenantID,
		EventType:       "test_action_dispatch",
		SubjectID:       target.MemberKey,
		VariantID:       target.VariantID,
		IsTest:          true,
		CandidateIRHash: run.CandidateIRHash,
		ListVersionHash: run.ListVersionHash,
		FixturePackHash: run.FixturePackHash,
		OccurredAt:      now,
		Metadata:        map[string]interface{}{"activity_type": activityType, "status": "success"},
	}
	if err := c.store.RecordTestAnalytics(ctx, fact); err != nil {
		return nil, fmt.Errorf("failed to record analytics fact: %w", err)
	}

	return ledger, nil
}

// CancelTestRun cancels undispatched test targets while preserving diagnostic results.
func (c *Coordinator) CancelTestRun(ctx context.Context, testRunID, reason string) (*TestRunRecord, error) {
	run, found, err := c.store.GetTestRun(ctx, testRunID)
	if err != nil || !found {
		return nil, ErrTestRunNotFound
	}

	if run.Status == TestRunStatusCancelled || run.Status == TestRunStatusExpired {
		return run, nil
	}

	now := time.Now()
	run.Status = TestRunStatusCancelled
	run.CancellationReason = reason
	run.CompletedAt = &now

	targets, err := c.store.GetTestTargets(ctx, testRunID)
	if err != nil {
		return nil, fmt.Errorf("failed to get targets: %w", err)
	}

	cancelledCount := 0
	for _, tgt := range targets {
		if tgt.Status == TargetStatusPending {
			tgt.Status = TargetStatusCancelled
			if err := c.store.UpdateTestTarget(ctx, tgt); err != nil {
				return nil, fmt.Errorf("failed to cancel target: %w", err)
			}
			cancelledCount++
		}
	}
	run.CancelledCount = cancelledCount

	if err := c.store.UpdateTestRun(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to update test run: %w", err)
	}

	return run, nil
}

// HandleExpiry cancels undispatched test targets when test run duration has expired.
func (c *Coordinator) HandleExpiry(ctx context.Context, testRunID string, now time.Time) (*TestRunRecord, error) {
	run, found, err := c.store.GetTestRun(ctx, testRunID)
	if err != nil || !found {
		return nil, ErrTestRunNotFound
	}

	if run.Status == TestRunStatusCancelled || run.Status == TestRunStatusExpired {
		return run, nil
	}

	if now.Before(run.ExpiresAt) {
		return run, nil
	}

	run.Status = TestRunStatusExpired
	run.CompletedAt = &now

	targets, err := c.store.GetTestTargets(ctx, testRunID)
	if err != nil {
		return nil, fmt.Errorf("failed to get targets: %w", err)
	}

	expiredCount := 0
	for _, tgt := range targets {
		if tgt.Status == TargetStatusPending {
			tgt.Status = TargetStatusExpired
			if err := c.store.UpdateTestTarget(ctx, tgt); err != nil {
				return nil, fmt.Errorf("failed to expire target: %w", err)
			}
			expiredCount++
		}
	}
	run.ExpiredCount = expiredCount

	if err := c.store.UpdateTestRun(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to update test run: %w", err)
	}

	return run, nil
}

// GetDiagnostics retrieves all diagnostic data for a given test run.
func (c *Coordinator) GetDiagnostics(ctx context.Context, testRunID string) (*TestRunDiagnostics, error) {
	run, found, err := c.store.GetTestRun(ctx, testRunID)
	if err != nil || !found {
		return nil, ErrTestRunNotFound
	}
	targets, err := c.store.GetTestTargets(ctx, testRunID)
	if err != nil {
		return nil, err
	}
	assignments, err := c.store.GetTestAssignments(ctx, testRunID)
	if err != nil {
		return nil, err
	}
	actions, err := c.store.GetTestActions(ctx, testRunID)
	if err != nil {
		return nil, err
	}
	analytics, err := c.store.GetTestAnalytics(ctx, testRunID)
	if err != nil {
		return nil, err
	}

	return &TestRunDiagnostics{
		Run:         run,
		Targets:     targets,
		Assignments: assignments,
		Actions:     actions,
		Analytics:   analytics,
	}, nil
}

func selectDeterministicVariant(memberKey, irHash string, variants []string) string {
	if len(variants) == 0 {
		return "control"
	}
	h := sha256.Sum256([]byte(memberKey + ":" + irHash))
	val := binary.BigEndian.Uint64(h[:8])
	idx := val % uint64(len(variants))
	return variants[idx]
}
