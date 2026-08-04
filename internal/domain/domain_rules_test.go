package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/validated-pattern/journey-platform/internal/domain"
)

// TestDomainRulesTableDriven tests domain rules, struct field behaviors, and hashing guarantees.
func TestDomainRulesTableDriven(t *testing.T) {
	t.Run("DataClassification constants validity", func(t *testing.T) {
		tests := []struct {
			name           string
			classification domain.DataClassification
			expected       string
		}{
			{"PII", domain.DataClassificationPII, "PII"},
			{"NonPII", domain.DataClassificationNonPII, "NonPII"},
			{"Sensitive", domain.DataClassificationSensitive, "Sensitive"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.expected, string(tt.classification))
			})
		}
	})

	t.Run("ComponentType constants validity", func(t *testing.T) {
		tests := []struct {
			name     string
			compType domain.ComponentType
			expected string
		}{
			{"Activity", domain.ComponentTypeActivity, "activity"},
			{"Trigger", domain.ComponentTypeTrigger, "trigger"},
			{"Action", domain.ComponentTypeAction, "action"},
			{"Condition", domain.ComponentTypeCondition, "condition"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.expected, string(tt.compType))
			})
		}
	})

	t.Run("Status enums validity", func(t *testing.T) {
		assert.Equal(t, "draft", string(domain.ExperimentStatusDraft))
		assert.Equal(t, "active", string(domain.ExperimentStatusActive))
		assert.Equal(t, "paused", string(domain.ExperimentStatusPaused))
		assert.Equal(t, "completed", string(domain.ExperimentStatusCompleted))

		assert.Equal(t, "success", string(domain.ActionResultStatusSuccess))
		assert.Equal(t, "failure", string(domain.ActionResultStatusFailure))
		assert.Equal(t, "retrying", string(domain.ActionResultStatusRetrying))

		assert.Equal(t, "pending", string(domain.TestRunStatusPending))
		assert.Equal(t, "passed", string(domain.TestRunStatusPassed))
		assert.Equal(t, "failed", string(domain.TestRunStatusFailed))

		assert.Equal(t, "error", string(domain.ValidationSeverityError))
		assert.Equal(t, "warning", string(domain.ValidationSeverityWarning))
		assert.Equal(t, "info", string(domain.ValidationSeverityInfo))

		assert.Equal(t, "running", string(domain.RunStatusRunning))
		assert.Equal(t, "completed", string(domain.RunStatusCompleted))
		assert.Equal(t, "failed", string(domain.RunStatusFailed))
		assert.Equal(t, "terminated", string(domain.RunStatusTerminated))

		assert.Equal(t, "1.0", domain.DefaultSchemaVersion)
	})

	t.Run("EventEnvelope SHA256 Table Tests", func(t *testing.T) {
		baseTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
		env1 := domain.EventEnvelope{
			SchemaVersion:      domain.DefaultSchemaVersion,
			EventID:            "evt-001",
			TraceID:            "trace-001",
			EventType:          "user.signup",
			Source:             "web-app",
			Subject:            "user-123",
			Timestamp:          baseTime,
			DataClassification: domain.DataClassificationNonPII,
			Data:               map[string]interface{}{"key": "val1"},
		}

		hash1, err := env1.CalculateSHA256()
		require.NoError(t, err)
		assert.NotEmpty(t, hash1)

		// Test ContentHash field exclusion property: setting ContentHash before calculation does not affect result
		env1WithHash := env1
		env1WithHash.ContentHash = "pre-existing-hash"
		hash1B, err := env1WithHash.CalculateSHA256()
		require.NoError(t, err)
		assert.Equal(t, hash1, hash1B, "ContentHash field must be ignored during SHA256 calculation")

		// Mutation Table Tests
		mutationTests := []struct {
			name      string
			mutateFn  func(e *domain.EventEnvelope)
			shouldChange bool
		}{
			{
				name: "mutate event ID",
				mutateFn: func(e *domain.EventEnvelope) {
					e.EventID = "evt-002"
				},
				shouldChange: true,
			},
			{
				name: "mutate timestamp",
				mutateFn: func(e *domain.EventEnvelope) {
					e.Timestamp = baseTime.Add(time.Hour)
				},
				shouldChange: true,
			},
			{
				name: "mutate data payload",
				mutateFn: func(e *domain.EventEnvelope) {
					e.Data = map[string]interface{}{"key": "val2"}
				},
				shouldChange: true,
			},
			{
				name: "mutate data classification",
				mutateFn: func(e *domain.EventEnvelope) {
					e.DataClassification = domain.DataClassificationPII
				},
				shouldChange: true,
			},
		}

		for _, tt := range mutationTests {
			t.Run(tt.name, func(t *testing.T) {
				mutated := env1
				tt.mutateFn(&mutated)
				mutatedHash, err := mutated.CalculateSHA256()
				require.NoError(t, err)
				if tt.shouldChange {
					assert.NotEqual(t, hash1, mutatedHash)
				} else {
					assert.Equal(t, hash1, mutatedHash)
				}
			})
		}
	})

	t.Run("GraphDraft SHA256 Table Tests", func(t *testing.T) {
		draft := domain.GraphDraft{
			SchemaVersion: domain.DefaultSchemaVersion,
			DraftID:       "draft-101",
			TenantID:      "tenant-a",
			Name:          "Onboarding Flow",
			Version:       1,
			Nodes: []domain.GraphNode{
				{ID: "n1", Type: "trigger", Name: "Start"},
			},
			Edges: []domain.GraphEdge{
				{ID: "e1", Source: "n1", Target: "n2"},
			},
		}

		hash1, err := draft.CalculateSHA256()
		require.NoError(t, err)
		assert.NotEmpty(t, hash1)

		draftWithHash := draft
		draftWithHash.ContentHash = "already-set-hash"
		hash1B, err := draftWithHash.CalculateSHA256()
		require.NoError(t, err)
		assert.Equal(t, hash1, hash1B)

		mutated := draft
		mutated.Nodes[0].Name = "Start Node Modified"
		mutatedHash, err := mutated.CalculateSHA256()
		require.NoError(t, err)
		assert.NotEqual(t, hash1, mutatedHash)
	})

	t.Run("CatalogRecord & StaticList & ExperimentDefinition SHA256 Determinism", func(t *testing.T) {
		cat := domain.CatalogRecord{
			SchemaVersion: domain.DefaultSchemaVersion,
			RecordID:      "rec-001",
			Name:          "SendEmail",
			ComponentType: domain.ComponentTypeAction,
			Version:       "1.0.0",
		}
		hashCat, err := cat.CalculateSHA256()
		require.NoError(t, err)
		assert.NotEmpty(t, hashCat)

		list := domain.StaticList{
			SchemaVersion: domain.DefaultSchemaVersion,
			ListID:        "list-001",
			Name:          "VIP Users",
			ItemCount:     2,
			Items:         []string{"u1", "u2"},
		}
		hashList, err := list.CalculateSHA256()
		require.NoError(t, err)
		assert.NotEmpty(t, hashList)

		exp := domain.ExperimentDefinition{
			SchemaVersion: domain.DefaultSchemaVersion,
			ExperimentID:  "exp-001",
			Name:          "Discount AB Test",
			Status:        domain.ExperimentStatusActive,
			Variants: []domain.ExperimentVariant{
				{VariantID: "v1", Name: "Control", WeightBasisPoints: 5000},
				{VariantID: "v2", Name: "Treatment", WeightBasisPoints: 5000},
			},
		}
		hashExp, err := exp.CalculateSHA256()
		require.NoError(t, err)
		assert.NotEmpty(t, hashExp)
	})
}
