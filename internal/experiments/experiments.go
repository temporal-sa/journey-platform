package experiments

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// RandomizationUnit defines supported units for experiment assignment.
type RandomizationUnit string

const (
	UnitUserID    RandomizationUnit = "user_id"
	UnitDeviceID  RandomizationUnit = "device_id"
	UnitAccountID RandomizationUnit = "account_id"
	UnitSessionID RandomizationUnit = "session_id"
	UnitSubjectID RandomizationUnit = "subject_id"
)

// ExperimentStatus defines experiment lifecycles.
type ExperimentStatus string

const (
	ExperimentStatusDraft     ExperimentStatus = "draft"
	ExperimentStatusActive    ExperimentStatus = "active"
	ExperimentStatusPaused    ExperimentStatus = "paused"
	ExperimentStatusCompleted ExperimentStatus = "completed"
)

// AssignmentMode defines whether an assignment request is for production or test mode.
type AssignmentMode string

const (
	ModeProduction AssignmentMode = "production"
	ModeTest       AssignmentMode = "test"
)

// Variant represents a single experiment treatment or control group.
type Variant struct {
	Key               string                 `json:"key"`
	Name              string                 `json:"name,omitempty"`
	WeightBasisPoints int                    `json:"weight_basis_points"`
	IsControl         bool                   `json:"is_control"`
	Config            map[string]interface{} `json:"config,omitempty"`
}

// MetricDefinition specifies a primary or secondary metric tracked during the experiment.
type MetricDefinition struct {
	Key       string `json:"key"`
	Name      string `json:"name,omitempty"`
	EventType string `json:"event_type"`
	Type      string `json:"type,omitempty"` // e.g. "conversion", "sum", "mean"
}

// Experiment defines the complete specification of an A/B or multivariate experiment version.
type Experiment struct {
	SchemaVersion            string             `json:"schema_version"`
	TenantID                 string             `json:"tenant_id"`
	ExperimentID             string             `json:"experiment_id"`
	Version                  int                `json:"version"`
	Name                     string             `json:"name"`
	Description              string             `json:"description,omitempty"`
	Status                   ExperimentStatus   `json:"status"`
	Salt                     string             `json:"salt"`
	RandomizationUnit        RandomizationUnit  `json:"randomization_unit"`
	Variants                 []Variant          `json:"variants"`
	Metrics                  []MetricDefinition `json:"metrics"`
	Filters                  map[string]string  `json:"filters,omitempty"`
	AttributionWindowSeconds int64              `json:"attribution_window_seconds"`
	TargetAudience           string             `json:"target_audience,omitempty"`
	ContentHash              string             `json:"content_hash,omitempty"`
	CreatedAt                time.Time          `json:"created_at,omitempty"`
	UpdatedAt                time.Time          `json:"updated_at,omitempty"`
}

// Clone creates a deep copy of the Experiment.
func (exp *Experiment) Clone() *Experiment {
	if exp == nil {
		return nil
	}
	cp := *exp
	if exp.Variants != nil {
		cp.Variants = make([]Variant, len(exp.Variants))
		for i, v := range exp.Variants {
			cp.Variants[i] = v
			if v.Config != nil {
				cp.Variants[i].Config = make(map[string]interface{})
				for k, val := range v.Config {
					cp.Variants[i].Config[k] = val
				}
			}
		}
	}
	if exp.Metrics != nil {
		cp.Metrics = make([]MetricDefinition, len(exp.Metrics))
		copy(cp.Metrics, exp.Metrics)
	}
	if exp.Filters != nil {
		cp.Filters = make(map[string]string)
		for k, v := range exp.Filters {
			cp.Filters[k] = v
		}
	}
	return &cp
}

// ImmutablePayload captures all fields that demand a version increment if modified.
type ImmutablePayload struct {
	Salt                     string             `json:"salt"`
	RandomizationUnit        RandomizationUnit  `json:"randomization_unit"`
	Variants                 []Variant          `json:"variants"`
	Metrics                  []MetricDefinition `json:"metrics"`
	Filters                  map[string]string  `json:"filters,omitempty"`
	AttributionWindowSeconds int64              `json:"attribution_window_seconds"`
}

var (
	ErrImmutableVersionConflict = errors.New("immutable experiment fields modified without incrementing version")
	ErrVersionMustIncrement     = errors.New("new version number must be strictly greater than current version")
)

// CalculateContentHash computes a deterministic SHA-256 hash over the experiment's immutable fields.
func (exp *Experiment) CalculateContentHash() (string, error) {
	payload := ImmutablePayload{
		Salt:                     exp.Salt,
		RandomizationUnit:        exp.RandomizationUnit,
		Variants:                 exp.Variants,
		Metrics:                  exp.Metrics,
		Filters:                  exp.Filters,
		AttributionWindowSeconds: exp.AttributionWindowSeconds,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal immutable payload: %w", err)
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// ValidateImmutability checks whether any allocation, salt, unit, metric, filter, or attribution changes
// occurred without a corresponding version increment.
func ValidateImmutability(oldExp, newExp *Experiment) error {
	if oldExp == nil || newExp == nil {
		return nil
	}

	oldHash, err := oldExp.CalculateContentHash()
	if err != nil {
		return fmt.Errorf("failed to compute old hash: %w", err)
	}

	newHash, err := newExp.CalculateContentHash()
	if err != nil {
		return fmt.Errorf("failed to compute new hash: %w", err)
	}

	if oldHash != newHash {
		if newExp.Version <= oldExp.Version {
			return fmt.Errorf("%w: old version %d, new version %d (hash mismatch)", ErrImmutableVersionConflict, oldExp.Version, newExp.Version)
		}
	}
	return nil
}

// CreateNextVersion creates a new experiment version with incremented version number and calculated content hash.
func CreateNextVersion(baseExp *Experiment, mutatedExp *Experiment) (*Experiment, error) {
	if baseExp == nil || mutatedExp == nil {
		return nil, errors.New("nil experiment provided")
	}

	nextVersion := baseExp.Version + 1
	nextExp := mutatedExp.Clone()
	nextExp.Version = nextVersion
	if nextExp.SchemaVersion == "" {
		nextExp.SchemaVersion = "1.0"
	}
	nextExp.UpdatedAt = time.Now().UTC()

	hash, err := nextExp.CalculateContentHash()
	if err != nil {
		return nil, err
	}
	nextExp.ContentHash = hash

	if err := ValidateExperiment(nextExp); err != nil {
		return nil, err
	}

	return nextExp, nil
}

// Manager manages experiment configurations and provides backward-compatible APIs.
type Manager struct {
	experiments map[string]*Experiment
}

// New creates a new Manager instance.
func New() *Manager {
	return &Manager{
		experiments: make(map[string]*Experiment),
	}
}

// Active returns true if experiments are active.
func (m *Manager) Active() bool {
	return true
}
