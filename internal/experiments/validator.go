package experiments

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrMissingExperimentID           = errors.New("experiment ID is required")
	ErrMissingSalt                   = errors.New("experiment salt is required")
	ErrInvalidVariantCount           = errors.New("experiment must have between 2 and 5 unique variants")
	ErrDuplicateVariantKey           = errors.New("variant keys must be unique")
	ErrEmptyVariantKey               = errors.New("variant key cannot be empty")
	ErrInvalidControlVariant         = errors.New("experiment must have exactly 1 control variant")
	ErrInvalidWeights                = errors.New("variant weights must be positive and sum to 10000 basis points")
	ErrUnsupportedRandomizationUnit = errors.New("unsupported randomization unit")
	ErrMissingMetrics                = errors.New("experiment must define at least one metric")
	ErrInvalidMetricDefinition       = errors.New("metric definition requires non-empty key and event_type")
	ErrInvalidAttributionWindow       = errors.New("attribution window must be positive")
)

// ValidateExperiment validates an experiment specification according to engine rules:
// - 2-5 unique variant keys
// - 1 control variant
// - Positive basis-point weights totaling 10000 (100%)
// - Supported randomization unit
// - Metric definitions
// - Attribution window > 0
func ValidateExperiment(exp *Experiment) error {
	if exp == nil {
		return errors.New("nil experiment provided")
	}

	if strings.TrimSpace(exp.ExperimentID) == "" {
		return ErrMissingExperimentID
	}

	if strings.TrimSpace(exp.Salt) == "" {
		return ErrMissingSalt
	}

	// 1. Variant count check (2-5 variants)
	if len(exp.Variants) < 2 || len(exp.Variants) > 5 {
		return fmt.Errorf("%w: got %d variants", ErrInvalidVariantCount, len(exp.Variants))
	}

	// 2. Variant keys uniqueness, non-emptiness, weights & control variant check
	seenKeys := make(map[string]bool)
	controlCount := 0
	totalWeight := 0

	for i := range exp.Variants {
		v := &exp.Variants[i]
		key := strings.TrimSpace(v.Key)
		if key == "" {
			return ErrEmptyVariantKey
		}
		if seenKeys[key] {
			return fmt.Errorf("%w: duplicate key %q", ErrDuplicateVariantKey, key)
		}
		seenKeys[key] = true

		if v.IsControl {
			controlCount++
		}

		if v.WeightBasisPoints <= 0 {
			return fmt.Errorf("%w: variant %q weight must be positive, got %d", ErrInvalidWeights, key, v.WeightBasisPoints)
		}
		totalWeight += v.WeightBasisPoints
	}

	// If no variant was explicitly marked as control, check if exactly 1 variant key is "control" or "var-control"
	if controlCount == 0 {
		for i := range exp.Variants {
			v := &exp.Variants[i]
			k := strings.ToLower(strings.TrimSpace(v.Key))
			if k == "control" || k == "var-control" || k == "c" {
				v.IsControl = true
				controlCount++
			}
		}
	}

	if controlCount != 1 {
		return fmt.Errorf("%w: expected 1 control variant, found %d", ErrInvalidControlVariant, controlCount)
	}

	if totalWeight != 10000 {
		return fmt.Errorf("%w: total weight must be 10000 basis points, got %d", ErrInvalidWeights, totalWeight)
	}

	// 3. Supported randomization unit check
	switch exp.RandomizationUnit {
	case UnitUserID, UnitDeviceID, UnitAccountID, UnitSessionID, UnitSubjectID:
		// valid
	default:
		return fmt.Errorf("%w: %q", ErrUnsupportedRandomizationUnit, exp.RandomizationUnit)
	}

	// 4. Metric definitions check
	if len(exp.Metrics) == 0 {
		return ErrMissingMetrics
	}
	for _, m := range exp.Metrics {
		if strings.TrimSpace(m.Key) == "" || strings.TrimSpace(m.EventType) == "" {
			return ErrInvalidMetricDefinition
		}
	}

	// 5. Attribution window check
	if exp.AttributionWindowSeconds <= 0 {
		return ErrInvalidAttributionWindow
	}

	return nil
}
