package outcomes

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/validated-pattern/journey-platform/internal/store/analytics"
	"github.com/validated-pattern/journey-platform/internal/store/postgres"
)

type OutcomeEvent struct {
	EventID           string                 `json:"event_id"`
	Source            string                 `json:"source"`
	SourceIdentity    string                 `json:"source_identity,omitempty"`
	EventType         string                 `json:"event_type"` // "delivered", "open", "click", "conversion"
	TenantID          string                 `json:"tenant_id"`
	ActionID          string                 `json:"action_id,omitempty"`
	AssignmentID      string                 `json:"assignment_id,omitempty"`
	ExposureID        string                 `json:"exposure_id,omitempty"`
	ExperimentID      string                 `json:"experiment_id,omitempty"`
	ExperimentVersion int                    `json:"experiment_version,omitempty"`
	JourneyID         string                 `json:"journey_id,omitempty"`
	JourneyVersion    int                    `json:"journey_version,omitempty"`
	ExecutionMode     string                 `json:"execution_mode,omitempty"` // "production" or "test"
	SubjectID         string                 `json:"subject_id,omitempty"`
	VariantID         string                 `json:"variant_id,omitempty"`
	MetricName        string                 `json:"metric_name,omitempty"`
	Value             float64                `json:"value,omitempty"`
	Unit              string                 `json:"unit,omitempty"`
	TargetURL         string                 `json:"target_url,omitempty"`
	UserAgent         string                 `json:"user_agent,omitempty"`
	IPAddress         string                 `json:"ip_address,omitempty"`
	Timestamp         time.Time              `json:"timestamp"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

type ProcessedOutcome struct {
	OutcomeID         string                 `json:"outcome_id"`
	EventID           string                 `json:"event_id"`
	Source            string                 `json:"source"`
	SourceIdentity    string                 `json:"source_identity"`
	EventType         string                 `json:"event_type"`
	TenantID          string                 `json:"tenant_id"`
	ActionID          string                 `json:"action_id"`
	AssignmentID      string                 `json:"assignment_id"`
	ExposureID        string                 `json:"exposure_id"`
	ExperimentID      string                 `json:"experiment_id"`
	ExperimentVersion int                    `json:"experiment_version"`
	JourneyID         string                 `json:"journey_id"`
	JourneyVersion    int                    `json:"journey_version"`
	ExecutionMode     string                 `json:"execution_mode"`
	SubjectID         string                 `json:"subject_id"`
	VariantID         string                 `json:"variant_id"`
	MetricName        string                 `json:"metric_name"`
	Value             float64                `json:"value"`
	Unit              string                 `json:"unit"`
	TargetURL         string                 `json:"target_url"`
	UserAgent         string                 `json:"user_agent"`
	IPAddress         string                 `json:"ip_address"`
	Timestamp         time.Time              `json:"timestamp"`
	QualityTag        QualityTag             `json:"quality_tag"`
	Metadata          map[string]interface{} `json:"metadata"`
	ProcessedAt       time.Time              `json:"processed_at"`
}

type IngestResult struct {
	EventID     string     `json:"event_id"`
	OutcomeID   string     `json:"outcome_id"`
	IsDuplicate bool       `json:"is_duplicate"`
	QualityTag  QualityTag `json:"quality_tag"`
	RawCount    int64      `json:"raw_count"`
	UniqueCount int64      `json:"unique_count"`
	ProcessedAt time.Time  `json:"processed_at"`
}

type Handler struct {
	mu                 sync.RWMutex
	pgRepo             postgres.Repository
	analyticsRepo      analytics.Repository
	classifier         *QualityClassifier
	seenEvents         map[string]bool
	rawEventCounts     map[string]int64
	rawCallbackCount   int64
	uniqueOutcomeCount int64
	outcomes           []*ProcessedOutcome
	seqCounter         uint64
}

type HandlerOption func(*Handler)

func WithPostgresRepository(repo postgres.Repository) HandlerOption {
	return func(h *Handler) {
		h.pgRepo = repo
	}
}

func WithAnalyticsRepository(repo analytics.Repository) HandlerOption {
	return func(h *Handler) {
		h.analyticsRepo = repo
	}
}

func WithQualityClassifier(qc *QualityClassifier) HandlerOption {
	return func(h *Handler) {
		h.classifier = qc
	}
}

func New(opts ...HandlerOption) *Handler {
	h := &Handler{
		classifier:     NewQualityClassifier(),
		seenEvents:     make(map[string]bool),
		rawEventCounts: make(map[string]int64),
		outcomes:       make([]*ProcessedOutcome, 0),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Process handles an outcome event payload by eventID.
func (h *Handler) Process(eventID string) bool {
	if eventID == "" {
		return false
	}
	_, err := h.Ingest(context.Background(), &OutcomeEvent{
		EventID:   eventID,
		EventType: "conversion",
		Source:    "default",
	})
	return err == nil
}

// Ingest ingests and correlates an incoming outcome event callback or tracking pixel.
func (h *Handler) Ingest(ctx context.Context, event *OutcomeEvent) (*IngestResult, error) {
	if event == nil {
		return nil, fmt.Errorf("outcome event cannot be nil")
	}

	if event.TenantID == "" {
		event.TenantID = "default"
	}
	if event.EventID == "" {
		event.EventID = fmt.Sprintf("evt-%d", time.Now().UnixNano())
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.EventType == "" {
		event.EventType = "conversion"
	}

	dedupKey := event.EventID
	if event.SourceIdentity != "" {
		dedupKey = event.SourceIdentity
	} else if event.Source != "" {
		dedupKey = fmt.Sprintf("%s:%s", event.Source, event.EventID)
	}

	h.mu.Lock()
	h.rawCallbackCount++
	isDuplicate := h.seenEvents[dedupKey]
	if isDuplicate {
		h.rawEventCounts[dedupKey]++
	} else {
		h.seenEvents[dedupKey] = true
		h.rawEventCounts[dedupKey] = 1
		h.uniqueOutcomeCount++
	}
	rawCount := h.rawCallbackCount
	uniqueCount := h.uniqueOutcomeCount
	seq := atomic.AddUint64(&h.seqCounter, 1)
	h.mu.Unlock()

	now := time.Now().UTC()
	outcomeID := fmt.Sprintf("outc-%d-%d", now.UnixNano(), seq)

	qualityTag := h.classifier.Classify(event, isDuplicate)

	processed := &ProcessedOutcome{
		OutcomeID:         outcomeID,
		EventID:           event.EventID,
		Source:            event.Source,
		SourceIdentity:    event.SourceIdentity,
		EventType:         event.EventType,
		TenantID:          event.TenantID,
		ActionID:          event.ActionID,
		AssignmentID:      event.AssignmentID,
		ExposureID:        event.ExposureID,
		ExperimentID:      event.ExperimentID,
		ExperimentVersion: event.ExperimentVersion,
		JourneyID:         event.JourneyID,
		JourneyVersion:    event.JourneyVersion,
		ExecutionMode:     event.ExecutionMode,
		SubjectID:         event.SubjectID,
		VariantID:         event.VariantID,
		MetricName:        event.MetricName,
		Value:             event.Value,
		Unit:              event.Unit,
		TargetURL:         event.TargetURL,
		UserAgent:         event.UserAgent,
		IPAddress:         event.IPAddress,
		Timestamp:         event.Timestamp,
		QualityTag:        qualityTag,
		Metadata:          event.Metadata,
		ProcessedAt:       now,
	}

	h.mu.Lock()
	h.outcomes = append(h.outcomes, processed)
	h.mu.Unlock()

	if h.analyticsRepo != nil {
		isTest := strings.EqualFold(event.ExecutionMode, "test")
		expVer := uint32(event.ExperimentVersion)
		if expVer == 0 {
			expVer = 1
		}
		jVer := uint32(event.JourneyVersion)
		if jVer == 0 {
			jVer = 1
		}

		if qualityTag.IsFiltered {
			_ = h.analyticsRepo.InsertDataQuality(ctx, []analytics.DataQualityFact{
				{
					QualityEventID:    fmt.Sprintf("dq-%s", outcomeID),
					TenantID:          event.TenantID,
					ExperimentID:      event.ExperimentID,
					ExperimentVersion: expVer,
					JourneyID:         event.JourneyID,
					JourneyVersion:    jVer,
					EntityType:        "outcome",
					EntityID:          outcomeID,
					RuleID:            qualityTag.RuleID,
					IssueType:         string(qualityTag.Classification),
					Severity:          "warning",
					IsTest:            isTest,
					OccurredAt:        event.Timestamp,
					IngestedAt:        now,
					Details:           qualityTag.Reason,
				},
			})
		} else {
			if strings.EqualFold(event.EventType, "conversion") {
				_ = h.analyticsRepo.InsertConversions(ctx, []analytics.ConversionFact{
					{
						ConversionID:      outcomeID,
						OutcomeID:         event.EventID,
						TenantID:          event.TenantID,
						ExperimentID:      event.ExperimentID,
						ExperimentVersion: expVer,
						JourneyID:         event.JourneyID,
						JourneyVersion:    jVer,
						VariantID:         event.VariantID,
						SubjectID:         event.SubjectID,
						MetricName:        event.MetricName,
						Value:             event.Value,
						Unit:              event.Unit,
						IsTest:            isTest,
						ConvertedAt:       event.Timestamp,
						IngestedAt:        now,
					},
				})
			}
		}
	}

	return &IngestResult{
		EventID:     event.EventID,
		OutcomeID:   outcomeID,
		IsDuplicate: isDuplicate,
		QualityTag:  qualityTag,
		RawCount:    rawCount,
		UniqueCount: uniqueCount,
		ProcessedAt: now,
	}, nil
}

func (h *Handler) GetOutcomes() []*ProcessedOutcome {
	h.mu.RLock()
	defer h.mu.RUnlock()
	res := make([]*ProcessedOutcome, len(h.outcomes))
	copy(res, h.outcomes)
	return res
}

func (h *Handler) GetRawCount() int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.rawCallbackCount
}

func (h *Handler) GetUniqueCount() int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.uniqueOutcomeCount
}
