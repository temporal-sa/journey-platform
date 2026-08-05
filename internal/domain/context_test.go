package domain_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/domain"
)

// generateRandomKey returns a randomized column header string.
func generateRandomKey(rng *rand.Rand, prefix string) string {
	chars := "abcdefghijklmnopqrstuvwxyz0123456789_"
	b := make([]byte, 10)
	for i := range b {
		b[i] = chars[rng.Intn(len(chars))]
	}
	return fmt.Sprintf("%s_%s", prefix, string(b))
}

// generateRandomValue returns a randomized value of varying types (string, int, bool, float).
func generateRandomValue(rng *rand.Rand) interface{} {
	switch rng.Intn(4) {
	case 0:
		return fmt.Sprintf("val_%d", rng.Intn(10000))
	case 1:
		return rng.Intn(5000)
	case 2:
		return rng.Float64() * 100.0
	default:
		return rng.Intn(2) == 1
	}
}

func TestFlattenPayloadContext_RandomizedColumns(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Run multiple randomized iterations to ensure robustness
	iterations := 100

	for i := 0; i < iterations; i++ {
		t.Run(fmt.Sprintf("Iteration_%d", i+1), func(t *testing.T) {
			expectedMap := make(map[string]interface{})
			topLevelPayload := make(map[string]interface{})

			// 1. Generate random top-level fields
			numTopLevel := rng.Intn(5) + 3 // 3 to 7 fields
			for j := 0; j < numTopLevel; j++ {
				k := generateRandomKey(rng, "top")
				v := generateRandomValue(rng)
				topLevelPayload[k] = v
				expectedMap[k] = v
			}

			// 2. Generate random attributes (nested map)
			numAttributes := rng.Intn(10) + 5 // 5 to 14 fields
			attributesMap := make(map[string]interface{})
			attributesStrMap := make(map[string]string)

			useStringMap := rng.Intn(2) == 1

			for j := 0; j < numAttributes; j++ {
				k := generateRandomKey(rng, "attr")
				if useStringMap {
					v := fmt.Sprintf("str_val_%d", rng.Intn(9999))
					attributesStrMap[k] = v
					expectedMap[k] = v
				} else {
					v := generateRandomValue(rng)
					attributesMap[k] = v
					expectedMap[k] = v
				}
			}

			if useStringMap {
				topLevelPayload["attributes"] = attributesStrMap
				expectedMap["attributes"] = attributesStrMap
			} else {
				topLevelPayload["attributes"] = attributesMap
				expectedMap["attributes"] = attributesMap
			}

			// 3. Execute FlattenPayloadContext
			result := domain.FlattenPayloadContext(topLevelPayload)

			// 4. Assert that all expected keys and values exist in the output map
			if len(result) < len(expectedMap) {
				t.Fatalf("Iteration %d: Result length %d is smaller than expected %d", i+1, len(result), len(expectedMap))
			}

			for expectedKey, expectedVal := range expectedMap {
				actualVal, exists := result[expectedKey]
				if !exists {
					t.Errorf("Iteration %d: Expected key %q not found in flattened result", i+1, expectedKey)
					continue
				}
				if fmt.Sprintf("%v", actualVal) != fmt.Sprintf("%v", expectedVal) {
					t.Errorf("Iteration %d: Mismatch for key %q. Expected: %v (%T), Got: %v (%T)",
						i+1, expectedKey, expectedVal, expectedVal, actualVal, actualVal)
				}
			}
		})
	}
}

func TestFlattenPayloadContext_NilAndEmpty(t *testing.T) {
	t.Run("Nil Input", func(t *testing.T) {
		res := domain.FlattenPayloadContext(nil)
		if res == nil {
			t.Fatal("Expected non-nil map for nil input")
		}
		if len(res) != 0 {
			t.Fatalf("Expected empty map for nil input, got len %d", len(res))
		}
	})

	t.Run("Empty Map Input", func(t *testing.T) {
		res := domain.FlattenPayloadContext(map[string]interface{}{})
		if len(res) != 0 {
			t.Fatalf("Expected empty map for empty input, got len %d", len(res))
		}
	})
}
