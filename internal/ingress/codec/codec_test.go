package codec_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/ingress/codec"
)

func TestCodec_EncodeDecodeValid(t *testing.T) {
	c := codec.NewCodec()
	now := time.Now().Truncate(time.Second)

	original := &codec.EventEnvelope{
		EventID:            "evt-1001",
		OccurredAt:         now,
		SchemaVersion:      "1.0",
		TenantID:           "tenant-alpha",
		SubjectRef:         "user-55",
		CorrelationID:      "corr-99",
		EventType:          "user.signup.v1",
		Source:             "auth-service",
		DataClassification: "PII",
		Payload: map[string]interface{}{
			"email": "test@example.com",
			"tier":  "gold",
		},
	}

	encoded, err := c.Encode(original)
	if err != nil {
		t.Fatalf("unexpected encode error: %v", err)
	}

	decoded, err := c.Decode(encoded)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}

	if decoded.EventID != original.EventID {
		t.Errorf("expected EventID %s, got %s", original.EventID, decoded.EventID)
	}
	if decoded.TenantID != original.TenantID {
		t.Errorf("expected TenantID %s, got %s", original.TenantID, decoded.TenantID)
	}
	if decoded.SubjectRef != original.SubjectRef {
		t.Errorf("expected SubjectRef %s, got %s", original.SubjectRef, decoded.SubjectRef)
	}
	if decoded.CorrelationID != original.CorrelationID {
		t.Errorf("expected CorrelationID %s, got %s", original.CorrelationID, decoded.CorrelationID)
	}
	if decoded.Payload["email"] != "test@example.com" {
		t.Errorf("expected email payload 'test@example.com', got %v", decoded.Payload["email"])
	}
}

func TestCodec_AliasFieldMapping(t *testing.T) {
	c := codec.NewCodec()

	// JSON payload using legacy / alternative domain alias keys (timestamp, trace_id, subject, data)
	rawJSON := `{
		"schema_version": "1.0",
		"event_id": "evt-alias-1",
		"timestamp": "2026-07-27T14:00:00Z",
		"tenant_id": "tenant-beta",
		"subject": "sub-777",
		"trace_id": "tr-888",
		"data": {
			"key": "value"
		}
	}`

	decoded, err := c.Decode([]byte(rawJSON))
	if err != nil {
		t.Fatalf("failed to decode alias JSON: %v", err)
	}

	if decoded.SubjectRef != "sub-777" {
		t.Errorf("expected SubjectRef 'sub-777' mapped from subject, got '%s'", decoded.SubjectRef)
	}
	if decoded.CorrelationID != "tr-888" {
		t.Errorf("expected CorrelationID 'tr-888' mapped from trace_id, got '%s'", decoded.CorrelationID)
	}
	if decoded.Payload["key"] != "value" {
		t.Errorf("expected Payload key 'value' mapped from data, got '%v'", decoded.Payload["key"])
	}
	if decoded.OccurredAt.IsZero() {
		t.Errorf("expected OccurredAt timestamp to be populated")
	}
}

func TestCodec_UnsupportedSchemaVersion(t *testing.T) {
	c := codec.NewCodec(codec.WithSupportedVersions("1.0", "1.1"))

	env := &codec.EventEnvelope{
		EventID:       "evt-unsupported",
		OccurredAt:    time.Now(),
		SchemaVersion: "99.0",
		TenantID:      "tenant-1",
		SubjectRef:    "sub-1",
		CorrelationID: "corr-1",
		Payload:       map[string]interface{}{"foo": "bar"},
	}

	_, err := c.Encode(env)
	if !errors.Is(err, codec.ErrUnsupportedSchema) {
		t.Fatalf("expected ErrUnsupportedSchema, got: %v", err)
	}

	invalidVersionJSON := `{
		"schema_version": "99.0",
		"event_id": "evt-unsupported",
		"occurred_at": "2026-07-27T14:00:00Z",
		"tenant_id": "tenant-1",
		"subject_ref": "sub-1",
		"payload": {"foo": "bar"}
	}`

	_, err = c.Decode([]byte(invalidVersionJSON))
	if !errors.Is(err, codec.ErrUnsupportedSchema) {
		t.Fatalf("expected ErrUnsupportedSchema decoding invalid version, got: %v", err)
	}
}

func TestCodec_MissingRequiredFields(t *testing.T) {
	c := codec.NewCodec()

	testCases := []struct {
		name string
		env  *codec.EventEnvelope
	}{
		{
			name: "missing_event_id",
			env: &codec.EventEnvelope{
				OccurredAt:    time.Now(),
				SchemaVersion: "1.0",
				TenantID:      "t-1",
				SubjectRef:    "s-1",
				Payload:       map[string]interface{}{"a": 1},
			},
		},
		{
			name: "missing_tenant_id",
			env: &codec.EventEnvelope{
				EventID:       "evt-1",
				OccurredAt:    time.Now(),
				SchemaVersion: "1.0",
				SubjectRef:    "s-1",
				Payload:       map[string]interface{}{"a": 1},
			},
		},
		{
			name: "missing_subject_ref",
			env: &codec.EventEnvelope{
				EventID:       "evt-1",
				OccurredAt:    time.Now(),
				SchemaVersion: "1.0",
				TenantID:      "t-1",
				Payload:       map[string]interface{}{"a": 1},
			},
		},
		{
			name: "zero_occurred_at",
			env: &codec.EventEnvelope{
				EventID:       "evt-1",
				SchemaVersion: "1.0",
				TenantID:      "t-1",
				SubjectRef:    "s-1",
				Payload:       map[string]interface{}{"a": 1},
			},
		},
		{
			name: "nil_payload",
			env: &codec.EventEnvelope{
				EventID:       "evt-1",
				OccurredAt:    time.Now(),
				SchemaVersion: "1.0",
				TenantID:      "t-1",
				SubjectRef:    "s-1",
				Payload:       nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := c.Encode(tc.env)
			if !errors.Is(err, codec.ErrMissingField) {
				t.Errorf("expected ErrMissingField for case %s, got: %v", tc.name, err)
			}
		})
	}
}

func TestCodec_PayloadTooLarge(t *testing.T) {
	c := codec.NewCodec(codec.WithMaxSizeBytes(100))

	largeString := strings.Repeat("A", 200)
	env := &codec.EventEnvelope{
		EventID:       "evt-large",
		OccurredAt:    time.Now(),
		SchemaVersion: "1.0",
		TenantID:      "t-1",
		SubjectRef:    "s-1",
		Payload:       map[string]interface{}{"data": largeString},
	}

	_, err := c.Encode(env)
	if !errors.Is(err, codec.ErrPayloadTooLarge) {
		t.Fatalf("expected ErrPayloadTooLarge, got: %v", err)
	}
}

func TestCodec_MalformedJSON(t *testing.T) {
	c := codec.NewCodec()
	badJSON := []byte(`{ invalid json syntax `)

	_, err := c.Decode(badJSON)
	if !errors.Is(err, codec.ErrInvalidJSON) {
		t.Fatalf("expected ErrInvalidJSON, got: %v", err)
	}
}

func TestCodec_CalculateContentHash(t *testing.T) {
	c := codec.NewCodec()
	env := &codec.EventEnvelope{
		Payload: map[string]interface{}{
			"foo": "bar",
			"num": 42,
		},
	}

	hash1, err := c.CalculateContentHash(env)
	if err != nil {
		t.Fatalf("unexpected hashing error: %v", err)
	}
	if hash1 == "" {
		t.Fatalf("expected non-empty hash string")
	}

	hash2, _ := c.CalculateContentHash(env)
	if hash1 != hash2 {
		t.Errorf("expected hash calculation to be deterministic, got %s vs %s", hash1, hash2)
	}
}

func TestCodec_DomainConversion(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	codecEnv := &codec.EventEnvelope{
		EventID:            "evt-domain-1",
		OccurredAt:         now,
		SchemaVersion:      "1.0",
		TenantID:           "tenant-xyz",
		SubjectRef:         "user-77",
		CorrelationID:      "corr-55",
		EventType:          "user.signup.v1",
		Source:             "web-client",
		DataClassification: "PII",
		Payload:            map[string]interface{}{"action": "test"},
	}

	domEnv := codecEnv.ToDomain()
	if domEnv.EventID != codecEnv.EventID {
		t.Errorf("domain EventID mismatch")
	}
	if domEnv.Subject != codecEnv.SubjectRef {
		t.Errorf("domain Subject mismatch")
	}
	if domEnv.TraceID != codecEnv.CorrelationID {
		t.Errorf("domain TraceID mismatch")
	}
	if domEnv.DataClassification != domain.DataClassificationPII {
		t.Errorf("domain DataClassification mismatch, got %v", domEnv.DataClassification)
	}

	convertedCodecEnv := codec.FromDomain(domEnv, "tenant-xyz")
	if convertedCodecEnv.TenantID != "tenant-xyz" {
		t.Errorf("converted TenantID mismatch")
	}
	if convertedCodecEnv.SubjectRef != codecEnv.SubjectRef {
		t.Errorf("converted SubjectRef mismatch")
	}
}
