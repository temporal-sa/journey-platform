package activities

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/store/object"
	"go.temporal.io/sdk/temporal"
)

func TestResolution_DistinguishStates(t *testing.T) {
	ctx := context.Background()
	mockAttr := NewMockAttributeResolver()
	now := time.Now().UTC()

	// 1. Setup mock attribute states
	mockAttr.SetAttribute("sub-1", "attr_false", &RawResolvedValue{
		Value:              false,
		ObservedAt:         now,
		SourceVersion:      "v1",
		DataClassification: domain.DataClassificationNonPII,
	})

	mockAttr.SetAttribute("sub-1", "attr_zero_int", &RawResolvedValue{
		Value:              0,
		ObservedAt:         now,
		SourceVersion:      "v1",
		DataClassification: domain.DataClassificationNonPII,
	})

	mockAttr.SetAttribute("sub-1", "attr_zero_float", &RawResolvedValue{
		Value:              0.0,
		ObservedAt:         now,
		SourceVersion:      "v1",
		DataClassification: domain.DataClassificationNonPII,
	})

	mockAttr.SetAttribute("sub-1", "attr_empty_str", &RawResolvedValue{
		Value:              "",
		ObservedAt:         now,
		SourceVersion:      "v1",
		DataClassification: domain.DataClassificationNonPII,
	})

	mockAttr.SetAttribute("sub-1", "attr_null", &RawResolvedValue{
		Value:              nil,
		IsNull:             true,
		ObservedAt:         now,
		SourceVersion:      "v1",
		DataClassification: domain.DataClassificationPII,
	})

	mockAttr.SetAttribute("sub-1", "attr_stale", &RawResolvedValue{
		Value:              "cached_value",
		ObservedAt:         now.Add(-2 * time.Hour),
		SourceVersion:      "v1",
		DataClassification: domain.DataClassificationSensitive,
	})

	mockAttr.SetAttribute("sub-1", "attr_unavail", &RawResolvedValue{
		IsUnavailable: true,
	})

	act := NewActivities().WithAttributeResolver(mockAttr)

	// 2. Execute ResolveAttributes
	input := ResolveAttributeInput{
		SubjectRef: "sub-1",
		Keys: []ResolutionKey{
			{Key: "attr_false"},
			{Key: "attr_zero_int"},
			{Key: "attr_zero_float"},
			{Key: "attr_empty_str"},
			{Key: "attr_null", Fallback: "fallback_null", HasFallback: true},
			{Key: "attr_missing", Fallback: "fallback_missing", HasFallback: true},
			{Key: "attr_stale", MaxAge: 1 * time.Hour, Fallback: "fallback_stale", HasFallback: true},
			{Key: "attr_unavail", Fallback: "fallback_unavail", HasFallback: true},
		},
	}

	res, err := act.ResolveAttributes(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error resolving attributes: %v", err)
	}

	// 3. Verify attr_false (must be resolved, false, non-nil)
	vFalse, ok := res.Values["attr_false"]
	if !ok {
		t.Fatalf("attr_false missing from output")
	}
	if vFalse.Status != ResolutionStatusResolved {
		t.Errorf("attr_false status = %s; want %s", vFalse.Status, ResolutionStatusResolved)
	}
	if vFalse.Value != false {
		t.Errorf("attr_false value = %v; want false", vFalse.Value)
	}
	if vFalse.Freshness != FreshnessFresh {
		t.Errorf("attr_false freshness = %s; want %s", vFalse.Freshness, FreshnessFresh)
	}

	// 4. Verify attr_zero_int (must be resolved, 0, non-nil)
	vZeroInt, ok := res.Values["attr_zero_int"]
	if !ok {
		t.Fatalf("attr_zero_int missing from output")
	}
	if vZeroInt.Status != ResolutionStatusResolved {
		t.Errorf("attr_zero_int status = %s; want %s", vZeroInt.Status, ResolutionStatusResolved)
	}
	if vZeroInt.Value != 0 {
		t.Errorf("attr_zero_int value = %v; want 0", vZeroInt.Value)
	}

	// 5. Verify attr_zero_float (must be resolved, 0.0, non-nil)
	vZeroFloat, ok := res.Values["attr_zero_float"]
	if !ok {
		t.Fatalf("attr_zero_float missing from output")
	}
	if vZeroFloat.Status != ResolutionStatusResolved {
		t.Errorf("attr_zero_float status = %s; want %s", vZeroFloat.Status, ResolutionStatusResolved)
	}
	if vZeroFloat.Value != 0.0 {
		t.Errorf("attr_zero_float value = %v; want 0.0", vZeroFloat.Value)
	}

	// 6. Verify attr_empty_str (must be resolved, "")
	vEmptyStr, ok := res.Values["attr_empty_str"]
	if !ok {
		t.Fatalf("attr_empty_str missing from output")
	}
	if vEmptyStr.Status != ResolutionStatusResolved {
		t.Errorf("attr_empty_str status = %s; want %s", vEmptyStr.Status, ResolutionStatusResolved)
	}
	if vEmptyStr.Value != "" {
		t.Errorf("attr_empty_str value = %v; want empty string", vEmptyStr.Value)
	}

	// 7. Verify attr_null (must be null status with explicit fallback value)
	vNull, ok := res.Values["attr_null"]
	if !ok {
		t.Fatalf("attr_null missing from output")
	}
	if vNull.Status != ResolutionStatusNull {
		t.Errorf("attr_null status = %s; want %s", vNull.Status, ResolutionStatusNull)
	}
	if vNull.Value != "fallback_null" {
		t.Errorf("attr_null value = %v; want fallback_null", vNull.Value)
	}

	// 8. Verify attr_missing (must be missing status with explicit fallback value)
	vMissing, ok := res.Values["attr_missing"]
	if !ok {
		t.Fatalf("attr_missing missing from output")
	}
	if vMissing.Status != ResolutionStatusMissing {
		t.Errorf("attr_missing status = %s; want %s", vMissing.Status, ResolutionStatusMissing)
	}
	if vMissing.Value != "fallback_missing" {
		t.Errorf("attr_missing value = %v; want fallback_missing", vMissing.Value)
	}

	// 9. Verify attr_stale (must be stale status with explicit fallback value and stale freshness)
	vStale, ok := res.Values["attr_stale"]
	if !ok {
		t.Fatalf("attr_stale missing from output")
	}
	if vStale.Status != ResolutionStatusStale {
		t.Errorf("attr_stale status = %s; want %s", vStale.Status, ResolutionStatusStale)
	}
	if vStale.Freshness != FreshnessStale {
		t.Errorf("attr_stale freshness = %s; want %s", vStale.Freshness, FreshnessStale)
	}
	if vStale.Value != "fallback_stale" {
		t.Errorf("attr_stale value = %v; want fallback_stale", vStale.Value)
	}

	// 10. Verify attr_unavail (must be unavailable status with fallback)
	vUnavail, ok := res.Values["attr_unavail"]
	if !ok {
		t.Fatalf("attr_unavail missing from output")
	}
	if vUnavail.Status != ResolutionStatusUnavailable {
		t.Errorf("attr_unavail status = %s; want %s", vUnavail.Status, ResolutionStatusUnavailable)
	}
	if vUnavail.Value != "fallback_unavail" {
		t.Errorf("attr_unavail value = %v; want fallback_unavail", vUnavail.Value)
	}
}

func TestResolution_Parameters(t *testing.T) {
	ctx := context.Background()
	mockParam := NewMockParameterResolver()
	now := time.Now().UTC()

	mockParam.SetParameter("param_discount", &RawResolvedValue{
		Value:              15.5,
		ObservedAt:         now,
		SourceVersion:      "v2",
		DataClassification: domain.DataClassificationNonPII,
	})

	act := NewActivities().WithParameterResolver(mockParam)

	res, err := act.ResolveParameters(ctx, ResolveParameterInput{
		SubjectRef: "global",
		Keys: []ResolutionKey{
			{Key: "param_discount"},
			{Key: "param_missing", Fallback: 0.0, HasFallback: true},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error resolving parameters: %v", err)
	}

	pDiscount := res.Values["param_discount"]
	if pDiscount.Status != ResolutionStatusResolved || pDiscount.Value != 15.5 {
		t.Errorf("param_discount resolution failed: status=%s, val=%v", pDiscount.Status, pDiscount.Value)
	}

	pMissing := res.Values["param_missing"]
	if pMissing.Status != ResolutionStatusMissing || pMissing.Value != 0.0 {
		t.Errorf("param_missing resolution failed: status=%s, val=%v", pMissing.Status, pMissing.Value)
	}
}

func TestResolution_DisallowedNonRetryable(t *testing.T) {
	ctx := context.Background()
	mockAttr := NewMockAttributeResolver()

	mockAttr.SetAttribute("sub-secret", "ssn_raw", &RawResolvedValue{
		Value:              "999-00-1234",
		IsDisallowed:       true,
		DataClassification: domain.DataClassificationPII,
	})

	act := NewActivities().WithAttributeResolver(mockAttr)

	_, err := act.ResolveAttributes(ctx, ResolveAttributeInput{
		SubjectRef: "sub-secret",
		Keys: []ResolutionKey{
			{Key: "ssn_raw"},
		},
	})

	if err == nil {
		t.Fatalf("expected error for disallowed field request, got nil")
	}

	var appErr *temporal.ApplicationError
	if !temporal.IsApplicationError(err) {
		t.Fatalf("expected temporal ApplicationError, got %T: %v", err, err)
	}
	if !errors.As(err, &appErr) {
		t.Fatalf("failed to cast to ApplicationError")
	}

	if !appErr.NonRetryable() {
		t.Errorf("expected disallowed error to be non-retryable")
	}

	// Verify error message does not expose the underlying secret data
	errMsg := err.Error()
	if strings.Contains(errMsg, "999-00-1234") {
		t.Errorf("error message leaked secret data: %s", errMsg)
	}
	if !strings.Contains(errMsg, "disallowed") {
		t.Errorf("error message should indicate disallowed access: %s", errMsg)
	}
}

func TestResolution_LargePayloadReference(t *testing.T) {
	ctx := context.Background()
	mockAttr := NewMockAttributeResolver()
	memStore := object.NewMemoryStore()
	now := time.Now().UTC()

	// Create a large object value (>50 bytes)
	largeData := map[string]interface{}{
		"history":     "User logged in from 192.168.1.1, accessed catalog, updated preferences, generated 10 events.",
		"tags":        []string{"vip", "active", "beta_tester", "enterprise", "high_volume"},
		"payload_num": 1000293,
	}

	mockAttr.SetAttribute("sub-large", "user_profile_large", &RawResolvedValue{
		Value:              largeData,
		ObservedAt:         now,
		SourceVersion:      "v1",
		DataClassification: domain.DataClassificationPII,
	})

	// Configure activities with small threshold (50 bytes)
	act := NewActivities().
		WithAttributeResolver(mockAttr).
		WithObjectStore(memStore).
		WithLargePayloadThreshold(50)

	res, err := act.ResolveAttributes(ctx, ResolveAttributeInput{
		SubjectRef: "sub-large",
		Keys: []ResolutionKey{
			{Key: "user_profile_large"},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, ok := res.Values["user_profile_large"]
	if !ok {
		t.Fatalf("missing user_profile_large in output")
	}

	if !val.IsRef {
		t.Errorf("expected IsRef to be true for large payload")
	}
	if val.ValueRef == "" {
		t.Errorf("expected ValueRef to contain object storage key")
	}
	if val.Value != nil {
		t.Errorf("expected inline Value to be nil when offloaded to object store, got %v", val.Value)
	}

	// Verify object store contains the offloaded data
	rc, info, err := memStore.GetObject(ctx, val.ValueRef)
	if err != nil {
		t.Fatalf("failed to retrieve offloaded object from store using ref %s: %v", val.ValueRef, err)
	}
	defer rc.Close()

	if info == nil || info.Size <= 50 {
		t.Errorf("stored object size expected > 50 bytes, got %v", info)
	}

	body, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("failed to read offloaded object bytes: %v", err)
	}

	var fetched map[string]interface{}
	if err := json.Unmarshal(body, &fetched); err != nil {
		t.Fatalf("failed to unmarshal offloaded payload JSON: %v", err)
	}

	if fetched["history"] != largeData["history"] {
		t.Errorf("offloaded object history mismatch: got %v", fetched["history"])
	}
}

func TestResolution_PayloadRefResolution(t *testing.T) {
	ctx := context.Background()
	memStore := object.NewMemoryStore()

	payloadMap := map[string]interface{}{
		"event_name":  "item_purchased",
		"total_price": 99.95,
	}
	payloadBytes, _ := json.Marshal(payloadMap)

	info, err := memStore.PutObject(ctx, "payloads/p123.json", strings.NewReader(string(payloadBytes)), object.PutOptions{
		TenantID: "tenant-1",
	})
	if err != nil {
		t.Fatalf("failed to put payload object: %v", err)
	}

	act := NewActivities().WithObjectStore(memStore)

	res, err := act.ResolveAttributes(ctx, ResolveAttributeInput{
		SubjectRef: "sub-1",
		PayloadRef: info.Key,
		Keys: []ResolutionKey{
			{Key: "event_name"},
			{Key: "total_price"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Values["event_name"].Value != "item_purchased" {
		t.Errorf("expected event_name = item_purchased, got %v", res.Values["event_name"].Value)
	}
	if res.Values["total_price"].Value != 99.95 {
		t.Errorf("expected total_price = 99.95, got %v", res.Values["total_price"].Value)
	}
}
