package security_test

import (
	"reflect"
	"testing"

	"github.com/validated-pattern/journey-platform/internal/domain"
)

// TestDataClassificationAnnotations asserts that every domain contract struct field
// has explicit annotations for storage, logging, temporal_history, and api_exposure.
func TestDataClassificationAnnotations(t *testing.T) {
	domainStructs := []interface{}{
		domain.EventEnvelope{},
		domain.NodePosition{},
		domain.GraphNode{},
		domain.GraphEdge{},
		domain.GraphDraft{},
		domain.RetryPolicy{},
		domain.IRNode{},
		domain.IREdge{},
		domain.CompiledIR{},
		domain.CatalogRecord{},
		domain.ExperimentVariant{},
		domain.ExperimentDefinition{},
		domain.AssignmentExposure{},
		domain.NormalizedOutcome{},
		domain.MetricSummary{},
		domain.AggregateReport{},
		domain.StaticList{},
		domain.TestRun{},
		domain.WorkflowInput{},
		domain.WorkflowSignal{},
		domain.WorkflowQuery{},
		domain.WorkflowActivityResult{},
		domain.ActionResult{},
		domain.ValidationIssue{},
		domain.RunProjection{},
	}

	requiredTagKeys := []string{
		"storage",
		"logging",
		"temporal_history",
		"api_exposure",
	}

	for _, s := range domainStructs {
		typ := reflect.TypeOf(s)
		t.Run(typ.Name(), func(t *testing.T) {
			numFields := typ.NumField()
			if numFields == 0 {
				t.Fatalf("struct %s has no fields", typ.Name())
			}

			for i := 0; i < numFields; i++ {
				field := typ.Field(i)

				for _, tagKey := range requiredTagKeys {
					tagVal := field.Tag.Get(tagKey)
					if tagVal == "" {
						t.Errorf("field '%s' in struct '%s' is missing explicit '%s' annotation tag",
							field.Name, typ.Name(), tagKey)
					}
				}
			}
		})
	}
}
