package experiments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/validated-pattern/journey-platform/internal/store/postgres"
)

// AssignmentRequest specifies parameters for sticky assignment evaluation.
type AssignmentRequest struct {
	TenantID        string                 `json:"tenant_id"`
	Experiment      *Experiment            `json:"experiment"`
	SubjectID       string                 `json:"subject_id"`
	Mode            AssignmentMode         `json:"mode"`
	ForcedVariantID string                 `json:"forced_variant_id,omitempty"`
	Context         map[string]interface{} `json:"context,omitempty"`
}

// AssignmentRecord represents a persisted sticky assignment.
type AssignmentRecord struct {
	AssignmentID      string                 `json:"assignment_id"`
	TenantID          string                 `json:"tenant_id"`
	ExperimentID      string                 `json:"experiment_id"`
	Version           int                    `json:"version"`
	SubjectID         string                 `json:"subject_id"`
	VariantID         string                 `json:"variant_id"`
	WeightBasisPoints int                    `json:"weight_basis_points"`
	Bucket            uint16                 `json:"bucket"`
	IsTestMode        bool                   `json:"is_test_mode"`
	Context           map[string]interface{} `json:"context,omitempty"`
	AssignedAt        time.Time              `json:"assigned_at"`
	CreatedAt         time.Time              `json:"created_at"`
}

// AssignmentService manages experiment variant sticky assignment resolution.
type AssignmentService struct {
	repo       postgres.Repository
	mu         sync.Mutex
	seqCounter uint64
}

// NewAssignmentService creates a new AssignmentService using the provided repository.
func NewAssignmentService(repo postgres.Repository) *AssignmentService {
	return &AssignmentService{
		repo: repo,
	}
}

// GetOrAssign retrieves an existing sticky assignment or creates a new deterministic assignment in a single transaction.
// Production and test mode assignments are strictly segregated.
func (s *AssignmentService) GetOrAssign(ctx context.Context, req AssignmentRequest) (*AssignmentRecord, error) {
	if s.repo == nil {
		return nil, errors.New("repository not initialized")
	}

	if req.TenantID == "" {
		req.TenantID = "default"
	}
	if req.SubjectID == "" {
		return nil, errors.New("subject_id is required")
	}
	if req.Experiment == nil {
		return nil, errors.New("experiment is required")
	}
	if req.Mode == "" {
		req.Mode = ModeProduction
	}

	isTest := req.Mode == ModeTest
	rawExpID := req.Experiment.ExperimentID
	if strings.TrimSpace(rawExpID) == "" {
		return nil, ErrMissingExperimentID
	}

	// Segregate production vs test mode by scoping test mode experiment IDs
	targetExpID := rawExpID
	if isTest {
		targetExpID = rawExpID + ":test"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var result *AssignmentRecord

	err := s.repo.WithTx(ctx, func(txRepo postgres.Repository) error {
		// 1. Find existing sticky assignment in one transaction
		existing, err := txRepo.GetAssignmentByExperimentSubject(ctx, req.TenantID, targetExpID, req.SubjectID)
		if err == nil && existing != nil {
			rec := s.toAssignmentRecord(existing, req.Experiment.Version, isTest)
			if req.Context != nil {
				rec.Context = req.Context
			}
			result = rec
			return nil
		}
		if err != nil && !errors.Is(err, postgres.ErrNotFound) {
			return fmt.Errorf("failed to query existing assignment: %w", err)
		}

		// 2. Determine variant and bucket index
		var selectedVariant *Variant
		var bucket uint16

		if isTest && req.ForcedVariantID != "" {
			for i := range req.Experiment.Variants {
				if req.Experiment.Variants[i].Key == req.ForcedVariantID {
					selectedVariant = &req.Experiment.Variants[i]
					break
				}
			}
			if selectedVariant == nil {
				return fmt.Errorf("forced variant %q not found in experiment %q", req.ForcedVariantID, rawExpID)
			}
			bucket = 0
		} else {
			var selectErr error
			selectedVariant, bucket, selectErr = SelectVariant(req.Experiment, req.SubjectID)
			if selectErr != nil {
				return selectErr
			}
		}

		// 3. Create atomic sticky assignment record
		now := time.Now().UTC()
		seq := atomic.AddUint64(&s.seqCounter, 1)
		prefix := "asg"
		if isTest {
			prefix = "asg-test"
		}
		asgID := fmt.Sprintf("%s-%s-%s-%d-%d", prefix, req.TenantID, rawExpID, now.UnixNano(), seq)

		dbAsg := &postgres.Assignment{
			TenantID:          req.TenantID,
			AssignmentID:      asgID,
			ExperimentID:      targetExpID,
			SubjectID:         req.SubjectID,
			VariantID:         selectedVariant.Key,
			WeightBasisPoints: int32(selectedVariant.WeightBasisPoints),
			AssignedAt:        now,
			CreatedAt:         now,
		}

		inserted, err := txRepo.CreateAssignment(ctx, dbAsg)
		if err != nil {
			if errors.Is(err, postgres.ErrConflict) || errors.Is(err, postgres.ErrAlreadyExists) || strings.Contains(err.Error(), "already assigned") {
				// Retry fetching concurrently created sticky assignment
				existingRetry, fetchErr := txRepo.GetAssignmentByExperimentSubject(ctx, req.TenantID, targetExpID, req.SubjectID)
				if fetchErr == nil && existingRetry != nil {
					rec := s.toAssignmentRecord(existingRetry, req.Experiment.Version, isTest)
					if req.Context != nil {
						rec.Context = req.Context
					}
					result = rec
					return nil
				}
			}
			return fmt.Errorf("failed to insert assignment: %w", err)
		}

		rec := s.toAssignmentRecord(inserted, req.Experiment.Version, isTest)
		rec.Bucket = bucket
		if req.Context != nil {
			rec.Context = req.Context
		}
		result = rec
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *AssignmentService) toAssignmentRecord(a *postgres.Assignment, version int, isTestMode bool) *AssignmentRecord {
	rawExpID := strings.TrimSuffix(a.ExperimentID, ":test")
	return &AssignmentRecord{
		AssignmentID:      a.AssignmentID,
		TenantID:          a.TenantID,
		ExperimentID:      rawExpID,
		Version:           version,
		SubjectID:         a.SubjectID,
		VariantID:         a.VariantID,
		WeightBasisPoints: int(a.WeightBasisPoints),
		IsTestMode:        isTestMode,
		AssignedAt:        a.AssignedAt,
		CreatedAt:         a.CreatedAt,
	}
}
