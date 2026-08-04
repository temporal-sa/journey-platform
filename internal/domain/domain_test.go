package domain_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/validated-pattern/journey-platform/internal/domain"
)

// List of all 14 target contracts with matching fixture and schema filenames.
var contractNames = []string{
	"event_envelope",
	"graph_draft",
	"compiled_ir",
	"catalog_record",
	"experiment_definition",
	"assignment_exposure",
	"normalized_outcome",
	"aggregate_report",
	"static_list",
	"test_run",
	"workflow_input",
	"action_result",
	"validation_issue",
	"run_projection",
}

// TestGoldenFixturesValidateAgainstJSONSchema verifies every golden fixture against its schema.
func TestGoldenFixturesValidateAgainstJSONSchema(t *testing.T) {
	rootDir := findRepoRoot(t)

	for _, name := range contractNames {
		t.Run(name, func(t *testing.T) {
			schemaPath := filepath.Join(rootDir, "api", "schemas", name+".json")
			fixturePath := filepath.Join(rootDir, "test", "fixtures", name+".json")

			schemaBytes, err := os.ReadFile(schemaPath)
			if err != nil {
				t.Fatalf("failed to read schema file %s: %v", schemaPath, err)
			}

			fixtureBytes, err := os.ReadFile(fixturePath)
			if err != nil {
				t.Fatalf("failed to read fixture file %s: %v", fixturePath, err)
			}

			err = validateJSONAgainstSchema(fixtureBytes, schemaBytes)
			if err != nil {
				t.Errorf("fixture %s failed schema validation: %v", name, err)
			}
		})
	}
}

// TestDomainStructsJSONRoundTrip tests clean serialization and deserialization without data loss.
func TestDomainStructsJSONRoundTrip(t *testing.T) {
	rootDir := findRepoRoot(t)

	for _, name := range contractNames {
		t.Run(name, func(t *testing.T) {
			fixturePath := filepath.Join(rootDir, "test", "fixtures", name+".json")
			fixtureBytes, err := os.ReadFile(fixturePath)
			if err != nil {
				t.Fatalf("failed to read fixture file %s: %v", fixturePath, err)
			}

			var targetStruct interface{}
			switch name {
			case "event_envelope":
				targetStruct = &domain.EventEnvelope{}
			case "graph_draft":
				targetStruct = &domain.GraphDraft{}
			case "compiled_ir":
				targetStruct = &domain.CompiledIR{}
			case "catalog_record":
				targetStruct = &domain.CatalogRecord{}
			case "experiment_definition":
				targetStruct = &domain.ExperimentDefinition{}
			case "assignment_exposure":
				targetStruct = &domain.AssignmentExposure{}
			case "normalized_outcome":
				targetStruct = &domain.NormalizedOutcome{}
			case "aggregate_report":
				targetStruct = &domain.AggregateReport{}
			case "static_list":
				targetStruct = &domain.StaticList{}
			case "test_run":
				targetStruct = &domain.TestRun{}
			case "workflow_input":
				targetStruct = &domain.WorkflowInput{}
			case "action_result":
				targetStruct = &domain.ActionResult{}
			case "validation_issue":
				targetStruct = &domain.ValidationIssue{}
			case "run_projection":
				targetStruct = &domain.RunProjection{}
			default:
				t.Fatalf("unhandled contract type: %s", name)
			}

			// 1. Unmarshal fixture JSON into struct
			err = json.Unmarshal(fixtureBytes, targetStruct)
			if err != nil {
				t.Fatalf("failed to unmarshal fixture %s into domain struct: %v", name, err)
			}

			// 2. Marshal struct back to JSON
			marshaledBytes, err := json.Marshal(targetStruct)
			if err != nil {
				t.Fatalf("failed to marshal struct %s to JSON: %v", name, err)
			}

			// 3. Unmarshal original and re-marshaled JSON into generic maps to verify equivalence
			var origMap map[string]interface{}
			var newMap map[string]interface{}

			if err := json.Unmarshal(fixtureBytes, &origMap); err != nil {
				t.Fatalf("failed to unmarshal fixture into map: %v", err)
			}
			if err := json.Unmarshal(marshaledBytes, &newMap); err != nil {
				t.Fatalf("failed to unmarshal marshaled struct into map: %v", err)
			}

			if !reflect.DeepEqual(origMap, newMap) {
				t.Errorf("round-trip data mismatch for %s:\nOriginal: %+v\nRoundtrip: %+v", name, origMap, newMap)
			}
		})
	}
}

// TestIdentifierAndContentHashDeterminism tests SHA-256 calculation determinism.
func TestIdentifierAndContentHashDeterminism(t *testing.T) {
	t.Run("CompiledIR Determinism", func(t *testing.T) {
		ir1 := domain.CompiledIR{
			SchemaVersion: domain.DefaultSchemaVersion,
			IRID:          "ir-100",
			DraftID:       "draft-1",
			TenantID:      "tenant-a",
			Version:       1,
			EntryNodeID:   "node-1",
			Nodes: []domain.IRNode{
				{ID: "node-1", Type: "activity", ActivityName: "SendEmail"},
			},
		}

		ir2 := ir1 // Copy of ir1

		hash1, err1 := ir1.CalculateSHA256()
		if err1 != nil {
			t.Fatalf("failed to calculate hash1: %v", err1)
		}
		hash2, err2 := ir2.CalculateSHA256()
		if err2 != nil {
			t.Fatalf("failed to calculate hash2: %v", err2)
		}

		if hash1 == "" {
			t.Fatal("calculated hash is empty")
		}
		if hash1 != hash2 {
			t.Errorf("expected deterministic hash match, got %s vs %s", hash1, hash2)
		}

		// Mutate node
		ir3 := ir1
		ir3.Nodes[0].ActivityName = "SendSMS"
		hash3, _ := ir3.CalculateSHA256()
		if hash1 == hash3 {
			t.Errorf("hash did not change after field mutation: %s", hash1)
		}
	})

	t.Run("EventEnvelope Determinism", func(t *testing.T) {
		env := domain.EventEnvelope{
			SchemaVersion:      domain.DefaultSchemaVersion,
			EventID:            "evt-123",
			EventType:          "user.created",
			DataClassification: domain.DataClassificationNonPII,
			Data:               map[string]interface{}{"key": "val"},
		}

		hash1, err := env.CalculateSHA256()
		if err != nil {
			t.Fatalf("failed to calculate hash: %v", err)
		}
		hash2, _ := env.CalculateSHA256()

		if hash1 != hash2 {
			t.Errorf("non-deterministic hash for EventEnvelope: %s vs %s", hash1, hash2)
		}
	})

	t.Run("GraphDraft Determinism", func(t *testing.T) {
		draft := domain.GraphDraft{
			SchemaVersion: domain.DefaultSchemaVersion,
			DraftID:       "draft-99",
			TenantID:      "t-1",
			Name:          "Test Journey",
			Version:       2,
		}

		hash1, _ := draft.CalculateSHA256()
		hash2, _ := draft.CalculateSHA256()
		if hash1 != hash2 {
			t.Errorf("non-deterministic hash for GraphDraft: %s vs %s", hash1, hash2)
		}
	})
}

// TestTemporalHistorySafetyAnnotations checks temporal_history struct tags on domain types.
func TestTemporalHistorySafetyAnnotations(t *testing.T) {
	structsToTest := []interface{}{
		domain.EventEnvelope{},
		domain.GraphDraft{},
		domain.CompiledIR{},
		domain.CatalogRecord{},
		domain.ExperimentDefinition{},
		domain.AssignmentExposure{},
		domain.NormalizedOutcome{},
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

	for _, s := range structsToTest {
		typ := reflect.TypeOf(s)
		t.Run(typ.Name(), func(t *testing.T) {
			for i := 0; i < typ.NumField(); i++ {
				field := typ.Field(i)
				tagVal := field.Tag.Get("temporal_history")
				if tagVal != "allowed" && tagVal != "prohibited" {
					t.Errorf("field %s in struct %s is missing or has invalid temporal_history tag: '%s'",
						field.Name, typ.Name(), tagVal)
				}
			}
		})
	}
}

// Helper: locate repository root
func findRepoRoot(t *testing.T) string {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found starting from working directory")
		}
		dir = parent
	}
}

// Lightweight schema validator for draft-07 JSON schemas
func validateJSONAgainstSchema(fixtureJSON []byte, schemaJSON []byte) error {
	var fixture map[string]interface{}
	if err := json.Unmarshal(fixtureJSON, &fixture); err != nil {
		return fmt.Errorf("invalid fixture JSON: %w", err)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		return fmt.Errorf("invalid schema JSON: %w", err)
	}

	return validateObject(fixture, schema, "")
}

func validateObject(obj map[string]interface{}, schema map[string]interface{}, path string) error {
	// 1. Required fields check
	if reqRaw, ok := schema["required"]; ok {
		if reqSlice, ok := reqRaw.([]interface{}); ok {
			for _, r := range reqSlice {
				fieldName, ok := r.(string)
				if !ok {
					continue
				}
				if _, exists := obj[fieldName]; !exists {
					return fmt.Errorf("missing required field '%s' at path '%s'", fieldName, fieldPath(path, fieldName))
				}
			}
		}
	}

	// 2. Properties check
	propsRaw, hasProps := schema["properties"].(map[string]interface{})
	if hasProps {
		for key, val := range obj {
			propSchemaRaw, isDefined := propsRaw[key]
			if !isDefined {
				// Additional properties check
				if addProps, exists := schema["additionalProperties"].(bool); exists && !addProps {
					return fmt.Errorf("unexpected additional property '%s' at path '%s'", key, path)
				}
				continue
			}

			propSchema, ok := propSchemaRaw.(map[string]interface{})
			if !ok {
				continue
			}

			if err := validateValue(val, propSchema, fieldPath(path, key)); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateValue(val interface{}, schema map[string]interface{}, path string) error {
	// Type check
	if typeRaw, ok := schema["type"].(string); ok {
		switch typeRaw {
		case "string":
			if _, ok := val.(string); !ok {
				return fmt.Errorf("expected string at path '%s', got %T", path, val)
			}
		case "integer":
			num, ok := val.(float64)
			if !ok || float64(int64(num)) != num {
				return fmt.Errorf("expected integer at path '%s', got %v (%T)", path, val, val)
			}
		case "number":
			if _, ok := val.(float64); !ok {
				return fmt.Errorf("expected number at path '%s', got %T", path, val)
			}
		case "boolean":
			if _, ok := val.(bool); !ok {
				return fmt.Errorf("expected boolean at path '%s', got %T", path, val)
			}
		case "object":
			objMap, ok := val.(map[string]interface{})
			if !ok {
				return fmt.Errorf("expected object at path '%s', got %T", path, val)
			}
			return validateObject(objMap, schema, path)
		case "array":
			arr, ok := val.([]interface{})
			if !ok {
				return fmt.Errorf("expected array at path '%s', got %T", path, val)
			}
			if itemSchema, ok := schema["items"].(map[string]interface{}); ok {
				for idx, item := range arr {
					itemPath := fmt.Sprintf("%s[%d]", path, idx)
					if err := validateValue(item, itemSchema, itemPath); err != nil {
						return err
					}
				}
			}
		}
	}

	// Enum check
	if enumRaw, ok := schema["enum"].([]interface{}); ok {
		matched := false
		for _, e := range enumRaw {
			if fmt.Sprintf("%v", e) == fmt.Sprintf("%v", val) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("value '%v' at path '%s' not in enum %v", val, path, enumRaw)
		}
	}

	return nil
}

func fieldPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}
