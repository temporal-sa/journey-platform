package api_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// simpleYAMLNode represents a lightweight parsed YAML map/structure for testing without external C dependencies.
type simpleYAMLMap map[string]interface{}

func parseSimpleYAML(content string) (simpleYAMLMap, error) {
	lines := strings.Split(content, "\n")
	result := make(simpleYAMLMap)
	
	// We build a simplified map based on indentation levels for top-level keys and nested maps/sections
	var stack []map[string]interface{}
	var keyStack []string
	var indentStack []int

	stack = append(stack, result)
	indentStack = append(indentStack, -1)

	for lineIdx, rawLine := range lines {
		// Strip comments
		line := rawLine
		if idx := strings.Index(line, "#"); idx >= 0 {
			// Ignore '#' if inside quotes (basic check)
			quoteCount := strings.Count(line[:idx], "\"") + strings.Count(line[:idx], "'")
			if quoteCount%2 == 0 {
				line = line[:idx]
			}
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		indent := len(rawLine) - len(strings.TrimLeft(rawLine, " "))

		// Adjust stack based on indentation
		for len(indentStack) > 1 && indent <= indentStack[len(indentStack)-1] {
			stack = stack[:len(stack)-1]
			indentStack = indentStack[:len(indentStack)-1]
			if len(keyStack) > 0 {
				keyStack = keyStack[:len(keyStack)-1]
			}
		}

		currentMap := stack[len(stack)-1]

		if strings.HasPrefix(trimmed, "- ") {
			// Array item line
			itemVal := strings.TrimSpace(trimmed[2:])
			arrKey := ""
			if len(keyStack) > 0 {
				arrKey = keyStack[len(keyStack)-1]
			}
			if arrKey != "" {
				var list []interface{}
				if existing, ok := currentMap[arrKey].([]interface{}); ok {
					list = existing
				}
				if strings.Contains(itemVal, ":") {
					// Map item in array
					parts := strings.SplitN(itemVal, ":", 2)
					subMap := map[string]interface{}{strings.TrimSpace(parts[0]): strings.TrimSpace(parts[1])}
					list = append(list, subMap)
				} else {
					list = append(list, itemVal)
				}
				currentMap[arrKey] = list
			}
			continue
		}

		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) < 1 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		var val interface{}
		if len(parts) > 1 {
			valStr := strings.TrimSpace(parts[1])
			if valStr != "" {
				val = valStr
			}
		}

		if val == nil {
			// Section start
			newMap := make(map[string]interface{})
			currentMap[key] = newMap
			stack = append(stack, newMap)
			indentStack = append(indentStack, indent)
			keyStack = append(keyStack, key)
		} else {
			currentMap[key] = val
		}
		_ = lineIdx
	}

	return result, nil
}

func TestOpenAPISyntaxAndStructure(t *testing.T) {
	apiBytes, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatalf("Failed to read api/openapi.yaml: %v", err)
	}

	content := string(apiBytes)
	if len(content) == 0 {
		t.Fatal("openapi.yaml is empty")
	}

	// 1. Verify OpenAPI version
	if !strings.Contains(content, "openapi: 3.0.") {
		t.Errorf("openapi.yaml must declare OpenAPI 3.0.x version")
	}

	// 2. Parse YAML
	doc, err := parseSimpleYAML(content)
	if err != nil {
		t.Fatalf("Failed to parse openapi.yaml: %v", err)
	}

	if _, ok := doc["info"]; !ok {
		t.Errorf("openapi.yaml is missing required top-level 'info' field")
	}

	if _, ok := doc["paths"]; !ok {
		t.Errorf("openapi.yaml is missing required top-level 'paths' field")
	}

	if _, ok := doc["components"]; !ok {
		t.Errorf("openapi.yaml is missing required top-level 'components' field")
	}

	// 3. Verify all required endpoints exist in paths
	requiredEndpoints := []string{
		"/catalogs/events",
		"/catalogs/actions",
		"/catalogs/attributes",
		"/catalogs/parameters",
		"/catalogs/metrics",
		"/catalogs/templates",
		"/journeys/drafts",
		"/journeys/drafts/{draft_id}",
		"/journeys/drafts/{draft_id}/validate",
		"/journeys/drafts/{draft_id}/simulate",
		"/journeys/drafts/{draft_id}/versions",
		"/journeys/drafts/{draft_id}/publish",
		"/journeys/drafts/{draft_id}/activate-local",
		"/journeys/drafts/{draft_id}/pause-local",
		"/experiments",
		"/experiments/{experiment_id}/versions/{version_id}",
		"/reports/aggregate",
		"/exports/csv",
		"/static-lists/upload",
		"/static-lists/{list_id}/versions",
		"/test-runs",
		"/test-runs/{test_run_id}",
		"/test-runs/{test_run_id}/cancel",
		"/journeys/runs",
		"/journeys/runs/{run_id}/timeline",
		"/events/emit",
		"/callbacks/outcomes",
	}

	pathsMap, ok := doc["paths"].(map[string]interface{})
	if !ok {
		t.Fatalf("paths section is invalid or not a map")
	}

	for _, ep := range requiredEndpoints {
		if _, found := pathsMap[ep]; !found {
			t.Errorf("Missing required OpenAPI path: %s", ep)
		}
	}

	// 4. Check header parameters and ErrorResponse schema
	compMap, ok := doc["components"].(map[string]interface{})
	if !ok {
		t.Fatalf("components section is invalid")
	}

	schemasMap, ok := compMap["schemas"].(map[string]interface{})
	if !ok {
		t.Fatalf("components.schemas section is invalid")
	}

	if _, found := schemasMap["ErrorResponse"]; !found {
		t.Errorf("components.schemas missing required 'ErrorResponse' schema")
	}

	if !strings.Contains(content, "Request-ID") {
		t.Errorf("openapi.yaml missing standard header 'Request-ID'")
	}
	if !strings.Contains(content, "Idempotency-Key") {
		t.Errorf("openapi.yaml missing standard header 'Idempotency-Key'")
	}
	if !strings.Contains(content, "If-Match") {
		t.Errorf("openapi.yaml missing standard header 'If-Match'")
	}
	if !strings.Contains(content, "ETag") {
		t.Errorf("openapi.yaml missing standard header 'ETag'")
	}
}

func TestOpenAPISchemaCompatibility(t *testing.T) {
	apiBytes, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatalf("Failed to read openapi.yaml: %v", err)
	}

	apiContent := string(apiBytes)
	doc, err := parseSimpleYAML(apiContent)
	if err != nil {
		t.Fatalf("Failed to parse openapi.yaml: %v", err)
	}

	compMap, ok := doc["components"].(map[string]interface{})
	if !ok {
		t.Fatalf("components section is invalid")
	}
	schemasMap, ok := compMap["schemas"].(map[string]interface{})
	if !ok {
		t.Fatalf("components.schemas section is invalid")
	}

	schemaFiles, err := filepath.Glob("schemas/*.json")
	if err != nil || len(schemaFiles) == 0 {
		t.Fatalf("No JSON schema files found in api/schemas/: %v", err)
	}

	for _, schemaPath := range schemaFiles {
		raw, err := os.ReadFile(schemaPath)
		if err != nil {
			t.Errorf("Failed to read %s: %v", schemaPath, err)
			continue
		}

		var jsonSchema struct {
			Title      string                 `json:"title"`
			Required   []string               `json:"required"`
			Properties map[string]interface{} `json:"properties"`
		}
		if err := json.Unmarshal(raw, &jsonSchema); err != nil {
			t.Errorf("Failed to unmarshal JSON schema %s: %v", schemaPath, err)
			continue
		}

		title := jsonSchema.Title
		if title == "" {
			continue
		}

		// Ensure corresponding OpenAPI schema exists under components/schemas
		openAPISchema, found := schemasMap[title]
		if !found {
			t.Errorf("JSON schema %s (Title: %s) is missing from OpenAPI components.schemas", schemaPath, title)
			continue
		}

		_ = openAPISchema

		// Check required properties from JSON schema are referenced in openapi.yaml
		for _, reqProp := range jsonSchema.Required {
			if !strings.Contains(apiContent, reqProp) {
				t.Errorf("Required property %s from schema %s is missing in openapi.yaml", reqProp, title)
			}
		}
	}
}

func TestOpenAPINoBreakingChangesAgainstBaseline(t *testing.T) {
	currentBytes, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatalf("Failed to read openapi.yaml: %v", err)
	}

	baselineBytes, err := os.ReadFile("openapi_baseline.yaml")
	if err != nil {
		t.Fatalf("Failed to read openapi_baseline.yaml baseline: %v", err)
	}

	currentDoc, err := parseSimpleYAML(string(currentBytes))
	if err != nil {
		t.Fatalf("Failed to parse openapi.yaml: %v", err)
	}

	baselineDoc, err := parseSimpleYAML(string(baselineBytes))
	if err != nil {
		t.Fatalf("Failed to parse openapi_baseline.yaml: %v", err)
	}

	// 1. Verify all baseline paths still exist in current OpenAPI spec
	baselinePaths, ok := baselineDoc["paths"].(map[string]interface{})
	if !ok {
		t.Fatalf("Baseline paths section invalid")
	}

	currentPaths, ok := currentDoc["paths"].(map[string]interface{})
	if !ok {
		t.Fatalf("Current paths section invalid")
	}

	for path, baseVal := range baselinePaths {
		curVal, found := currentPaths[path]
		if !found {
			t.Errorf("BREAKING CHANGE: Path %s present in baseline was removed from openapi.yaml", path)
			continue
		}

		baseMethodMap, bOk := baseVal.(map[string]interface{})
		curMethodMap, cOk := curVal.(map[string]interface{})
		if bOk && cOk {
			for method := range baseMethodMap {
				if _, mFound := curMethodMap[method]; !mFound {
					t.Errorf("BREAKING CHANGE: Method %s for path %s present in baseline was removed from openapi.yaml", strings.ToUpper(method), path)
				}
			}
		}
	}

	// 2. Verify all baseline component schemas still exist in current OpenAPI spec
	baselineComp, ok := baselineDoc["components"].(map[string]interface{})
	if !ok {
		t.Fatalf("Baseline components section invalid")
	}
	baselineSchemas, ok := baselineComp["schemas"].(map[string]interface{})
	if !ok {
		t.Fatalf("Baseline schemas section invalid")
	}

	currentComp, ok := currentDoc["components"].(map[string]interface{})
	if !ok {
		t.Fatalf("Current components section invalid")
	}
	currentSchemas, ok := currentComp["schemas"].(map[string]interface{})
	if !ok {
		t.Fatalf("Current schemas section invalid")
	}

	for schemaName := range baselineSchemas {
		if _, found := currentSchemas[schemaName]; !found {
			t.Errorf("BREAKING CHANGE: Schema %s present in baseline was removed from openapi.yaml", schemaName)
		}
	}
}
