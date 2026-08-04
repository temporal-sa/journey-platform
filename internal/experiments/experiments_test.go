package experiments

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/validated-pattern/journey-platform/internal/store/postgres"
)

// Helper to construct a valid base experiment for testing.
func helperValidExperiment() *Experiment {
	return &Experiment{
		SchemaVersion:     "1.0",
		TenantID:          "tenant-test",
		ExperimentID:      "exp-onboarding-v1",
		Version:           1,
		Name:              "Onboarding Optimization Experiment",
		Description:       "Testing new onboarding flow variations",
		Status:            ExperimentStatusActive,
		Salt:              "exp-onboarding-v1",
		RandomizationUnit: UnitUserID,
		Variants: []Variant{
			{
				Key:               "var-control",
				Name:              "Control",
				WeightBasisPoints: 5000,
				IsControl:         true,
			},
			{
				Key:               "var-treatment",
				Name:              "Treatment Video",
				WeightBasisPoints: 5000,
				IsControl:         false,
			},
		},
		Metrics: []MetricDefinition{
			{
				Key:       "met-conversion",
				Name:      "User Signup Conversion",
				EventType: "user_signup",
				Type:      "conversion",
			},
		},
		Filters: map[string]string{
			"country": "US",
		},
		AttributionWindowSeconds: 86400,
	}
}

// 1. Test HMAC bucketing against golden test vectors matching Go, TypeScript, and SQL semantics.
func TestHMACBucketing_GoldenVectors(t *testing.T) {
	salt := "exp-onboarding-v1"
	version := 1
	versionedKey := VersionedKey(salt, version)

	if versionedKey != "exp-onboarding-v1:v1" {
		t.Fatalf("expected versioned key 'exp-onboarding-v1:v1', got %q", versionedKey)
	}

	testCases := []struct {
		unitID          string
		expectedBucket  uint16
		expectedVariant string
	}{
		{unitID: "usr-0001", expectedBucket: 526, expectedVariant: "var-control"},
		{unitID: "usr-0002", expectedBucket: 7606, expectedVariant: "var-treatment"},
		{unitID: "usr-0003", expectedBucket: 7018, expectedVariant: "var-treatment"},
		{unitID: "usr-9999", expectedBucket: 4689, expectedVariant: "var-control"},
	}

	exp := helperValidExperiment()

	for _, tc := range testCases {
		t.Run(tc.unitID, func(t *testing.T) {
			bucket := BucketUnit(versionedKey, tc.unitID)
			if bucket != tc.expectedBucket {
				t.Errorf("Unit %s: expected bucket %d, got %d", tc.unitID, tc.expectedBucket, bucket)
			}

			variant, selBucket, err := SelectVariant(exp, tc.unitID)
			if err != nil {
				t.Fatalf("SelectVariant failed: %v", err)
			}
			if selBucket != tc.expectedBucket {
				t.Errorf("SelectVariant bucket mismatch: expected %d, got %d", tc.expectedBucket, selBucket)
			}
			if variant.Key != tc.expectedVariant {
				t.Errorf("Unit %s: expected variant %q, got %q", tc.unitID, tc.expectedVariant, variant.Key)
			}
		})
	}
}

// 2. Test distribution of 1,000,000 units meeting configured basis-point weight tolerances.
func TestDistribution_1MillionUnits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 1,000,000 unit distribution test in short mode")
	}

	exp := &Experiment{
		SchemaVersion:     "1.0",
		TenantID:          "tenant-dist",
		ExperimentID:      "exp-distribution-1m",
		Version:           1,
		Name:              "1M Distribution Scale Test",
		Status:            ExperimentStatusActive,
		Salt:              "salt-dist-1m",
		RandomizationUnit: UnitUserID,
		Variants: []Variant{
			{Key: "var-a", WeightBasisPoints: 5000, IsControl: true},  // 50% = 500,000
			{Key: "var-b", WeightBasisPoints: 3000, IsControl: false}, // 30% = 300,000
			{Key: "var-c", WeightBasisPoints: 2000, IsControl: false}, // 20% = 200,000
		},
		Metrics: []MetricDefinition{
			{Key: "m1", EventType: "click"},
		},
		AttributionWindowSeconds: 3600,
	}

	if err := ValidateExperiment(exp); err != nil {
		t.Fatalf("experiment setup invalid: %v", err)
	}

	counts := make(map[string]int)
	totalUnits := 1_000_000

	for i := 0; i < totalUnits; i++ {
		unitID := fmt.Sprintf("unit-user-%d", i)
		v, _, err := SelectVariant(exp, unitID)
		if err != nil {
			t.Fatalf("SelectVariant failed at iteration %d: %v", i, err)
		}
		counts[v.Key]++
	}

	// Verify basis-point weight tolerances (+/- 0.5% of total, i.e., +/- 5,000 units out of 1,000,000)
	tolerance := 5000 // 50 basis points (0.5%)

	expected := map[string]int{
		"var-a": 500_000,
		"var-b": 300_000,
		"var-c": 200_000,
	}

	t.Logf("Distribution outcome over %d units:", totalUnits)
	for varKey, expCount := range expected {
		actual := counts[varKey]
		diff := actual - expCount
		if diff < 0 {
			diff = -diff
		}
		pct := float64(actual) / float64(totalUnits) * 100.0
		t.Logf("  Variant %s: actual=%d (%.2f%%), expected=%d, diff=%d", varKey, actual, pct, expCount, diff)

		if diff > tolerance {
			t.Errorf("Variant %s count %d outside tolerance of %d (+/- %d)", varKey, actual, expCount, tolerance)
		}
	}
}

// 3. Test concurrent retry idempotency.
func TestAssignment_ConcurrentRetryIdempotency(t *testing.T) {
	repo := postgres.NewMemoryRepository()
	svc := NewAssignmentService(repo)
	exp := helperValidExperiment()

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	results := make([]*AssignmentRecord, numGoroutines)
	errs := make([]error, numGoroutines)

	ctx := context.Background()
	req := AssignmentRequest{
		TenantID:   "tenant-concurrent",
		Experiment: exp,
		SubjectID:  "subject-concurrent-user-42",
		Mode:       ModeProduction,
	}

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			asg, err := svc.GetOrAssign(ctx, req)
			results[idx] = asg
			errs[idx] = err
		}(i)
	}

	wg.Wait()

	// Verify all requests succeeded without error
	var firstAssigmentID string
	var firstVariantID string

	for i := 0; i < numGoroutines; i++ {
		if errs[i] != nil {
			t.Fatalf("goroutine %d returned error: %v", i, errs[i])
		}
		if results[i] == nil {
			t.Fatalf("goroutine %d returned nil assignment", i)
		}
		if i == 0 {
			firstAssigmentID = results[0].AssignmentID
			firstVariantID = results[0].VariantID
		} else {
			if results[i].AssignmentID != firstAssigmentID {
				t.Errorf("goroutine %d assignment ID mismatch: got %s, expected %s", i, results[i].AssignmentID, firstAssigmentID)
			}
			if results[i].VariantID != firstVariantID {
				t.Errorf("goroutine %d variant ID mismatch: got %s, expected %s", i, results[i].VariantID, firstVariantID)
			}
		}
	}

	// Verify DB contains exactly 1 assignment for this subject
	stored, err := repo.GetAssignmentByExperimentSubject(ctx, "tenant-concurrent", exp.ExperimentID, "subject-concurrent-user-42")
	if err != nil {
		t.Fatalf("failed to retrieve stored assignment from repo: %v", err)
	}
	if stored.VariantID != firstVariantID {
		t.Errorf("stored variant ID mismatch: got %s, expected %s", stored.VariantID, firstVariantID)
	}
}

// 4. Test test-mode vs production-mode segregation.
func TestAssignment_TestModeVsProductionSegregation(t *testing.T) {
	repo := postgres.NewMemoryRepository()
	svc := NewAssignmentService(repo)
	exp := helperValidExperiment()
	ctx := context.Background()
	subjectID := "subject-segregated-99"

	// Step 1: Assign in Production Mode
	prodReq := AssignmentRequest{
		TenantID:   "tenant-seg",
		Experiment: exp,
		SubjectID:  subjectID,
		Mode:       ModeProduction,
	}

	prodAsg, err := svc.GetOrAssign(ctx, prodReq)
	if err != nil {
		t.Fatalf("production GetOrAssign failed: %v", err)
	}
	if prodAsg.IsTestMode {
		t.Errorf("expected production assignment IsTestMode = false")
	}

	// Step 2: Assign in Test Mode with a Forced Variant
	testReq := AssignmentRequest{
		TenantID:        "tenant-seg",
		Experiment:      exp,
		SubjectID:       subjectID,
		Mode:            ModeTest,
		ForcedVariantID: "var-treatment",
	}

	testAsg, err := svc.GetOrAssign(ctx, testReq)
	if err != nil {
		t.Fatalf("test mode GetOrAssign failed: %v", err)
	}
	if !testAsg.IsTestMode {
		t.Errorf("expected test assignment IsTestMode = true")
	}
	if testAsg.VariantID != "var-treatment" {
		t.Errorf("expected test mode forced variant 'var-treatment', got %q", testAsg.VariantID)
	}
	if testAsg.AssignmentID == prodAsg.AssignmentID {
		t.Errorf("test mode assignment ID collided with production assignment ID")
	}

	// Step 3: Re-query Production Mode to confirm sticky assignment is untouched
	prodAsgRequery, err := svc.GetOrAssign(ctx, prodReq)
	if err != nil {
		t.Fatalf("production re-query failed: %v", err)
	}
	if prodAsgRequery.AssignmentID != prodAsg.AssignmentID {
		t.Errorf("production sticky assignment ID changed after test mode execution: expected %s, got %s", prodAsg.AssignmentID, prodAsgRequery.AssignmentID)
	}
	if prodAsgRequery.VariantID != prodAsg.VariantID {
		t.Errorf("production sticky variant changed: expected %s, got %s", prodAsg.VariantID, prodAsgRequery.VariantID)
	}
}

// 5. Additional unit tests for validator rules.
func TestValidator_Rules(t *testing.T) {
	t.Run("Valid Experiment", func(t *testing.T) {
		exp := helperValidExperiment()
		if err := ValidateExperiment(exp); err != nil {
			t.Errorf("expected valid experiment, got: %v", err)
		}
	})

	t.Run("Invalid Variant Count - Too Few", func(t *testing.T) {
		exp := helperValidExperiment()
		exp.Variants = exp.Variants[:1]
		if err := ValidateExperiment(exp); err == nil {
			t.Error("expected error for 1 variant, got nil")
		}
	})

	t.Run("Invalid Variant Count - Too Many", func(t *testing.T) {
		exp := helperValidExperiment()
		exp.Variants = []Variant{
			{Key: "v1", WeightBasisPoints: 2000, IsControl: true},
			{Key: "v2", WeightBasisPoints: 2000},
			{Key: "v3", WeightBasisPoints: 2000},
			{Key: "v4", WeightBasisPoints: 2000},
			{Key: "v5", WeightBasisPoints: 1000},
			{Key: "v6", WeightBasisPoints: 1000},
		}
		if err := ValidateExperiment(exp); err == nil {
			t.Error("expected error for 6 variants, got nil")
		}
	})

	t.Run("Duplicate Variant Key", func(t *testing.T) {
		exp := helperValidExperiment()
		exp.Variants[1].Key = exp.Variants[0].Key
		if err := ValidateExperiment(exp); err == nil {
			t.Error("expected error for duplicate variant key, got nil")
		}
	})

	t.Run("No Control Variant", func(t *testing.T) {
		exp := helperValidExperiment()
		exp.Variants[0].IsControl = false
		exp.Variants[0].Key = "var-alpha"
		exp.Variants[1].IsControl = false
		exp.Variants[1].Key = "var-beta"
		if err := ValidateExperiment(exp); err == nil {
			t.Error("expected error for missing control variant, got nil")
		}
	})

	t.Run("Multiple Control Variants", func(t *testing.T) {
		exp := helperValidExperiment()
		exp.Variants[0].IsControl = true
		exp.Variants[1].IsControl = true
		if err := ValidateExperiment(exp); err == nil {
			t.Error("expected error for multiple control variants, got nil")
		}
	})

	t.Run("Invalid Weights Sum", func(t *testing.T) {
		exp := helperValidExperiment()
		exp.Variants[0].WeightBasisPoints = 5000
		exp.Variants[1].WeightBasisPoints = 4000 // Sum 9000
		if err := ValidateExperiment(exp); err == nil {
			t.Error("expected error for weight sum != 10000, got nil")
		}
	})

	t.Run("Unsupported Randomization Unit", func(t *testing.T) {
		exp := helperValidExperiment()
		exp.RandomizationUnit = RandomizationUnit("ip_address")
		if err := ValidateExperiment(exp); err == nil {
			t.Error("expected error for unsupported randomization unit, got nil")
		}
	})

	t.Run("Missing Metrics", func(t *testing.T) {
		exp := helperValidExperiment()
		exp.Metrics = nil
		if err := ValidateExperiment(exp); err == nil {
			t.Error("expected error for missing metrics, got nil")
		}
	})

	t.Run("Zero Attribution Window", func(t *testing.T) {
		exp := helperValidExperiment()
		exp.AttributionWindowSeconds = 0
		if err := ValidateExperiment(exp); err == nil {
			t.Error("expected error for zero attribution window, got nil")
		}
	})
}

// 6. Test immutable versioning and version increment rules.
func TestImmutability_VersionEnforcement(t *testing.T) {
	v1 := helperValidExperiment()
	v1.Version = 1

	hash1, err := v1.CalculateContentHash()
	if err != nil {
		t.Fatalf("failed to calculate hash: %v", err)
	}
	v1.ContentHash = hash1

	// Modify allocation without incrementing version -> should fail validation
	v2SameVersion := v1.Clone()
	v2SameVersion.Variants[0].WeightBasisPoints = 6000
	v2SameVersion.Variants[1].WeightBasisPoints = 4000

	if err := ValidateImmutability(v1, v2SameVersion); err == nil {
		t.Error("expected error when modifying allocation without incrementing version")
	}

	// Modify salt without version increment -> should fail
	v2SaltChange := v1.Clone()
	v2SaltChange.Salt = "new-salt"
	if err := ValidateImmutability(v1, v2SaltChange); err == nil {
		t.Error("expected error when modifying salt without incrementing version")
	}

	// CreateNextVersion should properly increment version and recompute hash
	v2Valid, err := CreateNextVersion(v1, v2SameVersion)
	if err != nil {
		t.Fatalf("CreateNextVersion failed: %v", err)
	}
	if v2Valid.Version != 2 {
		t.Errorf("expected version 2, got %d", v2Valid.Version)
	}
	if v2Valid.ContentHash == v1.ContentHash {
		t.Errorf("expected content hash to change between versions")
	}

	if err := ValidateImmutability(v1, v2Valid); err != nil {
		t.Errorf("ValidateImmutability failed for valid version increment: %v", err)
	}
}

// 7. Backward compatibility check for Manager API.
func TestManager(t *testing.T) {
	m := New()
	if !m.Active() {
		t.Error("expected active manager")
	}
}
