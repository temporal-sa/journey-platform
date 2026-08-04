package outcomes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/validated-pattern/journey-platform/internal/store/postgres"
)

var (
	ErrMissingTenantID     = errors.New("tenant_id is required")
	ErrMissingExperimentID = errors.New("experiment_id is required")
	ErrMissingAssignmentID = errors.New("assignment_id is required")
	ErrMissingSubjectID    = errors.New("subject_id is required")
	ErrMissingVariantID    = errors.New("variant_id is required")
)

// ExposureRequest contains fields needed to record an exposure event.
type ExposureRequest struct {
	TenantID          string                 `json:"tenant_id"`
	ExposureID        string                 `json:"exposure_id,omitempty"`
	ExperimentID      string                 `json:"experiment_id"`
	ExperimentVersion int                    `json:"experiment_version,omitempty"`
	JourneyID         string                 `json:"journey_id,omitempty"`
	JourneyVersion    int                    `json:"journey_version,omitempty"`
	AssignmentID      string                 `json:"assignment_id"`
	SubjectID         string                 `json:"subject_id"`
	VariantID         string                 `json:"variant_id"`
	WeightBasisPoints int                    `json:"weight_basis_points,omitempty"`
	ExecutionMode     string                 `json:"execution_mode,omitempty"` // "production" or "test"
	Context           map[string]interface{} `json:"context,omitempty"`
	ExposedAt         time.Time              `json:"exposed_at,omitempty"`
}

// ExposureRecord represents a persisted experiment exposure record.
type ExposureRecord struct {
	ExposureID        string                 `json:"exposure_id"`
	TenantID          string                 `json:"tenant_id"`
	ExperimentID      string                 `json:"experiment_id"`
	ExperimentVersion int                    `json:"experiment_version"`
	JourneyID         string                 `json:"journey_id,omitempty"`
	JourneyVersion    int                    `json:"journey_version,omitempty"`
	AssignmentID      string                 `json:"assignment_id"`
	SubjectID         string                 `json:"subject_id"`
	VariantID         string                 `json:"variant_id"`
	WeightBasisPoints int                    `json:"weight_basis_points"`
	IsTestMode        bool                   `json:"is_test_mode"`
	Context           map[string]interface{} `json:"context,omitempty"`
	ExposedAt         time.Time              `json:"exposed_at"`
	CreatedAt         time.Time              `json:"created_at"`
}

// ExposureService manages recording and querying experiment exposures.
type ExposureService struct {
	repo       postgres.Repository
	seqCounter uint64
}

// NewExposureService creates a new ExposureService with the provided PostgreSQL repository.
func NewExposureService(repo postgres.Repository) *ExposureService {
	return &ExposureService{
		repo: repo,
	}
}

// RecordExposure records 1 exposure per Workflow/experiment node/visit immediately before variant branch entry
// via an idempotent transaction & outbox pattern.
func (s *ExposureService) RecordExposure(ctx context.Context, req ExposureRequest) (*ExposureRecord, error) {
	if s.repo == nil {
		return nil, errors.New("repository not initialized")
	}

	if req.TenantID == "" {
		req.TenantID = "default"
	}
	if req.ExperimentID == "" {
		return nil, ErrMissingExperimentID
	}
	if req.AssignmentID == "" {
		return nil, ErrMissingAssignmentID
	}
	if req.SubjectID == "" {
		return nil, ErrMissingSubjectID
	}
	if req.VariantID == "" {
		return nil, ErrMissingVariantID
	}

	isTest := strings.EqualFold(req.ExecutionMode, "test") || strings.HasSuffix(req.ExperimentID, ":test")
	rawExpID := strings.TrimSuffix(req.ExperimentID, ":test")

	targetExpID := rawExpID
	if isTest {
		targetExpID = rawExpID + ":test"
	}

	now := req.ExposedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var result *ExposureRecord

	err := s.repo.WithTx(ctx, func(txRepo postgres.Repository) error {
		expID := req.ExposureID
		if expID == "" {
			seq := atomic.AddUint64(&s.seqCounter, 1)
			prefix := "exp"
			if isTest {
				prefix = "exp-test"
			}
			expID = fmt.Sprintf("%s-%s-%s-%s-%d", prefix, req.TenantID, rawExpID, req.SubjectID, seq)
		}

		ctxBytes, marshalErr := json.Marshal(req.Context)
		if marshalErr != nil {
			ctxBytes = []byte("{}")
		}

		dbExp := &postgres.Exposure{
			TenantID:          req.TenantID,
			ExposureID:        expID,
			ExperimentID:      targetExpID,
			AssignmentID:      req.AssignmentID,
			SubjectID:         req.SubjectID,
			VariantID:         req.VariantID,
			WeightBasisPoints: int32(req.WeightBasisPoints),
			Context:           ctxBytes,
			ExposedAt:         now,
			CreatedAt:         now,
		}

		inserted, err := txRepo.CreateExposure(ctx, dbExp)
		if err != nil {
			if errors.Is(err, postgres.ErrConflict) || errors.Is(err, postgres.ErrAlreadyExists) || strings.Contains(err.Error(), "duplicate exposure") || strings.Contains(err.Error(), "already exists") {
				existing, getErr := txRepo.GetExposure(ctx, req.TenantID, expID)
				if getErr == nil && existing != nil {
					result = s.toExposureRecord(existing, req, isTest)
					return nil
				}
				result = &ExposureRecord{
					ExposureID:        expID,
					TenantID:          req.TenantID,
					ExperimentID:      rawExpID,
					ExperimentVersion: req.ExperimentVersion,
					JourneyID:         req.JourneyID,
					JourneyVersion:    req.JourneyVersion,
					AssignmentID:      req.AssignmentID,
					SubjectID:         req.SubjectID,
					VariantID:         req.VariantID,
					WeightBasisPoints: req.WeightBasisPoints,
					IsTestMode:        isTest,
					Context:           req.Context,
					ExposedAt:         now,
					CreatedAt:         now,
				}
				return nil
			}
			return fmt.Errorf("failed to create exposure: %w", err)
		}

		outboxPayload := map[string]interface{}{
			"exposure_id":        inserted.ExposureID,
			"tenant_id":          inserted.TenantID,
			"experiment_id":      rawExpID,
			"experiment_version": req.ExperimentVersion,
			"journey_id":         req.JourneyID,
			"journey_version":    req.JourneyVersion,
			"assignment_id":      inserted.AssignmentID,
			"subject_id":         inserted.SubjectID,
			"variant_id":         inserted.VariantID,
			"weight_basis_points": inserted.WeightBasisPoints,
			"is_test_mode":       isTest,
			"context":            req.Context,
			"exposed_at":         inserted.ExposedAt,
		}
		payloadBytes, _ := json.Marshal(outboxPayload)
		headersBytes, _ := json.Marshal(map[string]string{
			"source":         "exposure_service",
			"execution_mode": req.ExecutionMode,
		})

		outboxEvent := &postgres.Outbox{
			TenantID:      req.TenantID,
			ID:            fmt.Sprintf("outbox-exp-%s", inserted.ExposureID),
			AggregateType: "exposure",
			AggregateID:   inserted.ExposureID,
			EventType:     "experiment.exposure_recorded",
			Payload:       payloadBytes,
			Headers:       headersBytes,
			Status:        "pending",
			CreatedAt:     now,
		}

		if _, outboxErr := txRepo.CreateOutboxEvent(ctx, outboxEvent); outboxErr != nil {
			if !errors.Is(outboxErr, postgres.ErrAlreadyExists) && !errors.Is(outboxErr, postgres.ErrConflict) {
				return fmt.Errorf("failed to create outbox event for exposure: %w", outboxErr)
			}
		}

		result = s.toExposureRecord(inserted, req, isTest)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetExposure retrieves an exposure record by ID.
func (s *ExposureService) GetExposure(ctx context.Context, tenantID, exposureID string) (*ExposureRecord, error) {
	if s.repo == nil {
		return nil, errors.New("repository not initialized")
	}
	if tenantID == "" {
		tenantID = "default"
	}
	dbExp, err := s.repo.GetExposure(ctx, tenantID, exposureID)
	if err != nil {
		return nil, err
	}
	isTest := strings.HasSuffix(dbExp.ExperimentID, ":test")
	rawExpID := strings.TrimSuffix(dbExp.ExperimentID, ":test")
	var ctxMap map[string]interface{}
	if len(dbExp.Context) > 0 {
		_ = json.Unmarshal(dbExp.Context, &ctxMap)
	}
	return &ExposureRecord{
		ExposureID:        dbExp.ExposureID,
		TenantID:          dbExp.TenantID,
		ExperimentID:      rawExpID,
		AssignmentID:      dbExp.AssignmentID,
		SubjectID:         dbExp.SubjectID,
		VariantID:         dbExp.VariantID,
		WeightBasisPoints: int(dbExp.WeightBasisPoints),
		IsTestMode:        isTest,
		Context:           ctxMap,
		ExposedAt:         dbExp.ExposedAt,
		CreatedAt:         dbExp.CreatedAt,
	}, nil
}

func (s *ExposureService) toExposureRecord(exp *postgres.Exposure, req ExposureRequest, isTest bool) *ExposureRecord {
	rawExpID := strings.TrimSuffix(exp.ExperimentID, ":test")
	var ctxMap map[string]interface{}
	if req.Context != nil {
		ctxMap = req.Context
	} else if len(exp.Context) > 0 {
		_ = json.Unmarshal(exp.Context, &ctxMap)
	}
	return &ExposureRecord{
		ExposureID:        exp.ExposureID,
		TenantID:          exp.TenantID,
		ExperimentID:      rawExpID,
		ExperimentVersion: req.ExperimentVersion,
		JourneyID:         req.JourneyID,
		JourneyVersion:    req.JourneyVersion,
		AssignmentID:      exp.AssignmentID,
		SubjectID:         exp.SubjectID,
		VariantID:         exp.VariantID,
		WeightBasisPoints: int(exp.WeightBasisPoints),
		IsTestMode:        isTest,
		Context:           ctxMap,
		ExposedAt:         exp.ExposedAt,
		CreatedAt:         exp.CreatedAt,
	}
}
