package activities

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Decision represents the policy decision outcome.
type Decision string

const (
	DecisionAllow      Decision = "allow"
	DecisionDelayUntil Decision = "delay_until"
	DecisionSuppress   Decision = "suppress"
)

// ExecutionMode represents the engine execution mode.
type ExecutionMode string

const (
	ExecutionModeProduction ExecutionMode = "production"
	ExecutionModeTest       ExecutionMode = "test"
)

// Non-sensitive reason codes.
const (
	ReasonAllowed              = "ALLOWED"
	ReasonMissingContext       = "MISSING_CONTEXT"
	ReasonContextUnavailable   = "CONTEXT_UNAVAILABLE"
	ReasonNotInTestList        = "NOT_IN_TEST_LIST"
	ReasonConsentWithdrawn     = "CONSENT_WITHDRAWN"
	ReasonSuppressed           = "SUPPRESSED"
	ReasonNotContactable       = "NOT_CONTACTABLE"
	ReasonQuietHours           = "QUIET_HOURS"
	ReasonFrequencyCapExceeded = "FREQUENCY_CAP_EXCEEDED"
)

// DefaultPolicyVersion represents the active policy version.
const DefaultPolicyVersion = "1.0"

// ErrFrequencyCapExceeded indicates frequency capacity limit reached.
var ErrFrequencyCapExceeded = errors.New("frequency cap capacity exceeded")

// PolicyContext contains evaluation input data for communication policy decisions.
type PolicyContext struct {
	SubjectID        string        `json:"subject_id"`
	Channel          string        `json:"channel"`
	ExecutionMode    ExecutionMode `json:"execution_mode"`
	RecipientAddress string        `json:"recipient_address"`
	Timezone         string        `json:"timezone"`
}

// PolicyDecisionResult contains the outcome of a policy evaluation.
type PolicyDecisionResult struct {
	Decision      Decision   `json:"decision"`
	ReasonCode    string     `json:"reason_code"`
	PolicyVersion string     `json:"policy_version"`
	DelayUntil    *time.Time `json:"delay_until,omitempty"`
}

// TestListStore determines test membership.
type TestListStore interface {
	IsTestSubject(ctx context.Context, subjectID string) (bool, error)
}

// ConsentStore determines channel consent.
type ConsentStore interface {
	HasConsent(ctx context.Context, subjectID string, channel string) (bool, error)
}

// SuppressionStore determines if subject/address is suppressed.
type SuppressionStore interface {
	IsSuppressed(ctx context.Context, subjectID string, recipientAddress string, channel string) (bool, error)
}

// ContactabilityStore determines contactability of subject/address.
type ContactabilityStore interface {
	IsContactable(ctx context.Context, subjectID string, recipientAddress string) (bool, error)
}

// MemoryPolicyStore provides an in-memory implementation of policy stores.
type MemoryPolicyStore struct {
	mu           sync.RWMutex
	testSubjects map[string]bool
	consents     map[string]bool // key: "subjectID:channel"
	suppressions map[string]bool // key: "subjectID:address:channel"
	contactable  map[string]bool // key: "subjectID:address"
}

// NewMemoryPolicyStore creates a new MemoryPolicyStore.
func NewMemoryPolicyStore() *MemoryPolicyStore {
	return &MemoryPolicyStore{
		testSubjects: make(map[string]bool),
		consents:     make(map[string]bool),
		suppressions: make(map[string]bool),
		contactable:  make(map[string]bool),
	}
}

func (m *MemoryPolicyStore) SetTestSubject(subjectID string, isTest bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.testSubjects[subjectID] = isTest
}

func (m *MemoryPolicyStore) IsTestSubject(ctx context.Context, subjectID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.testSubjects[subjectID], nil
}

func (m *MemoryPolicyStore) SetConsent(subjectID, channel string, hasConsent bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.consents[fmt.Sprintf("%s:%s", subjectID, channel)] = hasConsent
}

func (m *MemoryPolicyStore) HasConsent(ctx context.Context, subjectID, channel string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.consents[fmt.Sprintf("%s:%s", subjectID, channel)]
	if !ok {
		return true, nil // default consent true if unconfigured
	}
	return val, nil
}

func (m *MemoryPolicyStore) SetSuppressed(subjectID, recipientAddress, channel string, suppressed bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%s:%s", subjectID, recipientAddress, channel)
	m.suppressions[key] = suppressed
}

func (m *MemoryPolicyStore) IsSuppressed(ctx context.Context, subjectID, recipientAddress, channel string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := fmt.Sprintf("%s:%s:%s", subjectID, recipientAddress, channel)
	if m.suppressions[key] {
		return true, nil
	}
	// Also check wildcard address/channel
	if m.suppressions[fmt.Sprintf("%s::", subjectID)] || m.suppressions[fmt.Sprintf(":%s:", recipientAddress)] {
		return true, nil
	}
	return false, nil
}

func (m *MemoryPolicyStore) SetContactable(subjectID, recipientAddress string, contactable bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s:%s", subjectID, recipientAddress)
	m.contactable[key] = contactable
}

func (m *MemoryPolicyStore) IsContactable(ctx context.Context, subjectID, recipientAddress string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := fmt.Sprintf("%s:%s", subjectID, recipientAddress)
	val, ok := m.contactable[key]
	if !ok {
		return true, nil // default contactable if unconfigured
	}
	return val, nil
}

// Reservation represents an active frequency capacity reservation.
type Reservation struct {
	ID        string
	SubjectID string
	Channel   string
	Amount    int
	CreatedAt time.Time
}

// FrequencyLimiter manages transactional capacity reservation.
type FrequencyLimiter struct {
	mu           sync.Mutex
	defaultCap   int
	caps         map[string]int // key: "subjectID:channel" or "channel"
	used         map[string]int // key: "subjectID:channel"
	reservations map[string]*Reservation
	nextSeq      uint64
}

// NewFrequencyLimiter creates a new FrequencyLimiter.
func NewFrequencyLimiter(defaultCap int) *FrequencyLimiter {
	if defaultCap <= 0 {
		defaultCap = 10
	}
	return &FrequencyLimiter{
		defaultCap:   defaultCap,
		caps:         make(map[string]int),
		used:         make(map[string]int),
		reservations: make(map[string]*Reservation),
	}
}

func (fl *FrequencyLimiter) SetCap(subjectID, channel string, capLimit int) {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	fl.caps[fmt.Sprintf("%s:%s", subjectID, channel)] = capLimit
}

func (fl *FrequencyLimiter) SetDefaultCap(defaultCap int) {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	fl.defaultCap = defaultCap
}

func (fl *FrequencyLimiter) getCapLocked(key, channel string) int {
	if capLimit, ok := fl.caps[key]; ok {
		return capLimit
	}
	if capLimit, ok := fl.caps[channel]; ok {
		return capLimit
	}
	return fl.defaultCap
}

func (fl *FrequencyLimiter) HasCapacity(ctx context.Context, subjectID, channel string, amount int) (bool, error) {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	key := fmt.Sprintf("%s:%s", subjectID, channel)
	capLimit := fl.getCapLocked(key, channel)
	return fl.used[key]+amount <= capLimit, nil
}

func (fl *FrequencyLimiter) ReserveFrequencyCapacity(ctx context.Context, subjectID, channel string, amount int) (string, error) {
	if subjectID == "" || channel == "" || amount <= 0 {
		return "", errors.New("invalid reservation parameters")
	}

	fl.mu.Lock()
	defer fl.mu.Unlock()

	key := fmt.Sprintf("%s:%s", subjectID, channel)
	capLimit := fl.getCapLocked(key, channel)

	currentUsed := fl.used[key]
	if currentUsed+amount > capLimit {
		return "", ErrFrequencyCapExceeded
	}

	fl.nextSeq++
	resID := fmt.Sprintf("res-%d-%d", time.Now().UnixNano(), fl.nextSeq)
	res := &Reservation{
		ID:        resID,
		SubjectID: subjectID,
		Channel:   channel,
		Amount:    amount,
		CreatedAt: time.Now(),
	}

	fl.used[key] = currentUsed + amount
	fl.reservations[resID] = res

	return resID, nil
}

func (fl *FrequencyLimiter) ReleaseFrequencyCapacity(ctx context.Context, reservationID string) error {
	if reservationID == "" {
		return errors.New("invalid reservation ID")
	}

	fl.mu.Lock()
	defer fl.mu.Unlock()

	res, exists := fl.reservations[reservationID]
	if !exists {
		return errors.New("reservation not found")
	}

	key := fmt.Sprintf("%s:%s", res.SubjectID, res.Channel)
	fl.used[key] -= res.Amount
	if fl.used[key] < 0 {
		fl.used[key] = 0
	}

	delete(fl.reservations, reservationID)
	return nil
}

// QuietHoursConfig defines quiet hours window in recipient local time.
type QuietHoursConfig struct {
	Enabled   bool
	StartHour int // 0-23
	EndHour   int // 0-23
}

// PolicyService evaluates deterministic communication policies.
type PolicyService struct {
	version             string
	clock               func() time.Time
	locationLoader      func(name string) (*time.Location, error)
	testListStore       TestListStore
	consentStore        ConsentStore
	suppressionStore    SuppressionStore
	contactabilityStore ContactabilityStore
	quietHours          QuietHoursConfig
	frequencyLimiter    *FrequencyLimiter
}

// PolicyServiceOption configures PolicyService.
type PolicyServiceOption func(*PolicyService)

func WithClock(clock func() time.Time) PolicyServiceOption {
	return func(s *PolicyService) {
		s.clock = clock
	}
}

func WithLocationLoader(loader func(name string) (*time.Location, error)) PolicyServiceOption {
	return func(s *PolicyService) {
		s.locationLoader = loader
	}
}

func WithTestListStore(store TestListStore) PolicyServiceOption {
	return func(s *PolicyService) {
		s.testListStore = store
	}
}

func WithConsentStore(store ConsentStore) PolicyServiceOption {
	return func(s *PolicyService) {
		s.consentStore = store
	}
}

func WithSuppressionStore(store SuppressionStore) PolicyServiceOption {
	return func(s *PolicyService) {
		s.suppressionStore = store
	}
}

func WithContactabilityStore(store ContactabilityStore) PolicyServiceOption {
	return func(s *PolicyService) {
		s.contactabilityStore = store
	}
}

func WithQuietHours(config QuietHoursConfig) PolicyServiceOption {
	return func(s *PolicyService) {
		s.quietHours = config
	}
}

func WithFrequencyLimiter(limiter *FrequencyLimiter) PolicyServiceOption {
	return func(s *PolicyService) {
		s.frequencyLimiter = limiter
	}
}

func WithPolicyVersion(version string) PolicyServiceOption {
	return func(s *PolicyService) {
		s.version = version
	}
}

// NewPolicyService creates a new PolicyService with sensible defaults.
func NewPolicyService(opts ...PolicyServiceOption) *PolicyService {
	memStore := NewMemoryPolicyStore()
	s := &PolicyService{
		version:             DefaultPolicyVersion,
		clock:               time.Now,
		locationLoader:      time.LoadLocation,
		testListStore:       memStore,
		consentStore:        memStore,
		suppressionStore:    memStore,
		contactabilityStore: memStore,
		quietHours: QuietHoursConfig{
			Enabled:   true,
			StartHour: 21, // 9:00 PM
			EndHour:   9,  // 9:00 AM
		},
		frequencyLimiter: NewFrequencyLimiter(10),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// ReserveFrequencyCapacity transactionally reserves capacity.
func (s *PolicyService) ReserveFrequencyCapacity(ctx context.Context, subjectID, channel string, amount int) (string, error) {
	return s.frequencyLimiter.ReserveFrequencyCapacity(ctx, subjectID, channel, amount)
}

// ReleaseFrequencyCapacity releases reserved capacity.
func (s *PolicyService) ReleaseFrequencyCapacity(ctx context.Context, reservationID string) error {
	return s.frequencyLimiter.ReleaseFrequencyCapacity(ctx, reservationID)
}

// Evaluate evaluates communication policy deterministically based on context.
func (s *PolicyService) Evaluate(ctx context.Context, pctx *PolicyContext) PolicyDecisionResult {
	// Fail-closed check for missing or nil context
	if pctx == nil || pctx.SubjectID == "" || pctx.Channel == "" || pctx.ExecutionMode == "" {
		return PolicyDecisionResult{
			Decision:      DecisionSuppress,
			ReasonCode:    ReasonMissingContext,
			PolicyVersion: s.version,
		}
	}

	// Fail-closed check for missing or invalid timezone
	if pctx.Timezone == "" {
		return PolicyDecisionResult{
			Decision:      DecisionSuppress,
			ReasonCode:    ReasonMissingContext,
			PolicyVersion: s.version,
		}
	}

	loc, err := s.locationLoader(pctx.Timezone)
	if err != nil || loc == nil {
		return PolicyDecisionResult{
			Decision:      DecisionSuppress,
			ReasonCode:    ReasonMissingContext,
			PolicyVersion: s.version,
		}
	}

	// 1. Execution Mode & Test List Membership
	if pctx.ExecutionMode == ExecutionModeTest {
		if s.testListStore == nil {
			return PolicyDecisionResult{
				Decision:      DecisionSuppress,
				ReasonCode:    ReasonContextUnavailable,
				PolicyVersion: s.version,
			}
		}
		isTest, err := s.testListStore.IsTestSubject(ctx, pctx.SubjectID)
		if err != nil {
			return PolicyDecisionResult{
				Decision:      DecisionSuppress,
				ReasonCode:    ReasonContextUnavailable,
				PolicyVersion: s.version,
			}
		}
		if !isTest {
			return PolicyDecisionResult{
				Decision:      DecisionSuppress,
				ReasonCode:    ReasonNotInTestList,
				PolicyVersion: s.version,
			}
		}
	}

	// 2. Channel Consent Fixtures
	if s.consentStore == nil {
		return PolicyDecisionResult{
			Decision:      DecisionSuppress,
			ReasonCode:    ReasonContextUnavailable,
			PolicyVersion: s.version,
		}
	}
	hasConsent, err := s.consentStore.HasConsent(ctx, pctx.SubjectID, pctx.Channel)
	if err != nil {
		return PolicyDecisionResult{
			Decision:      DecisionSuppress,
			ReasonCode:    ReasonContextUnavailable,
			PolicyVersion: s.version,
		}
	}
	if !hasConsent {
		return PolicyDecisionResult{
			Decision:      DecisionSuppress,
			ReasonCode:    ReasonConsentWithdrawn,
			PolicyVersion: s.version,
		}
	}

	// 3. Suppression Lists
	if s.suppressionStore == nil {
		return PolicyDecisionResult{
			Decision:      DecisionSuppress,
			ReasonCode:    ReasonContextUnavailable,
			PolicyVersion: s.version,
		}
	}
	isSuppressed, err := s.suppressionStore.IsSuppressed(ctx, pctx.SubjectID, pctx.RecipientAddress, pctx.Channel)
	if err != nil {
		return PolicyDecisionResult{
			Decision:      DecisionSuppress,
			ReasonCode:    ReasonContextUnavailable,
			PolicyVersion: s.version,
		}
	}
	if isSuppressed {
		return PolicyDecisionResult{
			Decision:      DecisionSuppress,
			ReasonCode:    ReasonSuppressed,
			PolicyVersion: s.version,
		}
	}

	// 4. Contactability
	if s.contactabilityStore == nil {
		return PolicyDecisionResult{
			Decision:      DecisionSuppress,
			ReasonCode:    ReasonContextUnavailable,
			PolicyVersion: s.version,
		}
	}
	isContactable, err := s.contactabilityStore.IsContactable(ctx, pctx.SubjectID, pctx.RecipientAddress)
	if err != nil {
		return PolicyDecisionResult{
			Decision:      DecisionSuppress,
			ReasonCode:    ReasonContextUnavailable,
			PolicyVersion: s.version,
		}
	}
	if !isContactable {
		return PolicyDecisionResult{
			Decision:      DecisionSuppress,
			ReasonCode:    ReasonNotContactable,
			PolicyVersion: s.version,
		}
	}

	// 5. Quiet Hours (with Timezone Evaluation)
	if s.quietHours.Enabled {
		nowLocal := s.clock().In(loc)
		if inQuiet, delayUntil := InQuietHours(nowLocal, s.quietHours.StartHour, s.quietHours.EndHour); inQuiet {
			return PolicyDecisionResult{
				Decision:      DecisionDelayUntil,
				ReasonCode:    ReasonQuietHours,
				PolicyVersion: s.version,
				DelayUntil:    &delayUntil,
			}
		}
	}

	// 6. Frequency Caps
	if s.frequencyLimiter == nil {
		return PolicyDecisionResult{
			Decision:      DecisionSuppress,
			ReasonCode:    ReasonContextUnavailable,
			PolicyVersion: s.version,
		}
	}
	hasCap, err := s.frequencyLimiter.HasCapacity(ctx, pctx.SubjectID, pctx.Channel, 1)
	if err != nil {
		return PolicyDecisionResult{
			Decision:      DecisionSuppress,
			ReasonCode:    ReasonContextUnavailable,
			PolicyVersion: s.version,
		}
	}
	if !hasCap {
		return PolicyDecisionResult{
			Decision:      DecisionSuppress,
			ReasonCode:    ReasonFrequencyCapExceeded,
			PolicyVersion: s.version,
		}
	}

	// All evaluations passed
	return PolicyDecisionResult{
		Decision:      DecisionAllow,
		ReasonCode:    ReasonAllowed,
		PolicyVersion: s.version,
	}
}

// InQuietHours determines if current local time is within quiet hours.
func InQuietHours(now time.Time, startHour, endHour int) (bool, time.Time) {
	if startHour == endHour {
		return false, time.Time{}
	}

	h := now.Hour()
	loc := now.Location()

	if startHour > endHour {
		// Overnight quiet hours, e.g., 21 to 9 (9 PM to 9 AM)
		if h >= startHour {
			tomorrow := now.AddDate(0, 0, 1)
			delayUntil := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), endHour, 0, 0, 0, loc)
			return true, delayUntil
		}
		if h < endHour {
			delayUntil := time.Date(now.Year(), now.Month(), now.Day(), endHour, 0, 0, 0, loc)
			return true, delayUntil
		}
	} else {
		// Daytime quiet hours, e.g., 1 to 5
		if h >= startHour && h < endHour {
			delayUntil := time.Date(now.Year(), now.Month(), now.Day(), endHour, 0, 0, 0, loc)
			return true, delayUntil
		}
	}

	return false, time.Time{}
}
