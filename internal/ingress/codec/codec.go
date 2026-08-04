package codec

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/validated-pattern/journey-platform/internal/domain"
)

var (
	ErrInvalidJSON       = errors.New("invalid JSON payload")
	ErrUnsupportedSchema = errors.New("unsupported schema version")
	ErrMissingField      = errors.New("missing required envelope field")
	ErrInvalidTenant     = errors.New("invalid or empty tenant ID")
	ErrPayloadTooLarge   = errors.New("payload exceeds maximum allowed size")
)

const (
	DefaultSchemaVersion = "1.0"
	DefaultMaxSizeBytes  = 1024 * 1024 // 1 MB
)

// EventEnvelope represents the standardized versioned event container.
type EventEnvelope struct {
	EventID            string                 `json:"event_id"`
	OccurredAt         time.Time              `json:"occurred_at"`
	SchemaVersion      string                 `json:"schema_version"`
	TenantID           string                 `json:"tenant_id"`
	SubjectRef         string                 `json:"subject_ref"`
	CorrelationID      string                 `json:"correlation_id"`
	Payload            map[string]interface{} `json:"payload"`
	EventType          string                 `json:"event_type,omitempty"`
	Source             string                 `json:"source,omitempty"`
	DataClassification string                 `json:"data_classification,omitempty"`
	ContentHash        string                 `json:"content_hash,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshaling to support field alias mappings:
// occurred_at / timestamp, correlation_id / trace_id, subject_ref / subject, payload / data.
func (e *EventEnvelope) UnmarshalJSON(data []byte) error {
	type rawEnvelope struct {
		EventID            string                 `json:"event_id"`
		OccurredAt         *time.Time             `json:"occurred_at"`
		Timestamp          *time.Time             `json:"timestamp"`
		SchemaVersion      string                 `json:"schema_version"`
		TenantID           string                 `json:"tenant_id"`
		SubjectRef         string                 `json:"subject_ref"`
		Subject            string                 `json:"subject"`
		CorrelationID      string                 `json:"correlation_id"`
		TraceID            string                 `json:"trace_id"`
		Payload            map[string]interface{} `json:"payload"`
		Data               map[string]interface{} `json:"data"`
		EventType          string                 `json:"event_type"`
		Source             string                 `json:"source"`
		DataClassification string                 `json:"data_classification"`
		ContentHash        string                 `json:"content_hash"`
	}

	var raw rawEnvelope
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	e.EventID = raw.EventID
	e.SchemaVersion = raw.SchemaVersion
	e.TenantID = raw.TenantID
	e.EventType = raw.EventType
	e.Source = raw.Source
	e.DataClassification = raw.DataClassification
	e.ContentHash = raw.ContentHash

	if raw.SubjectRef != "" {
		e.SubjectRef = raw.SubjectRef
	} else {
		e.SubjectRef = raw.Subject
	}

	if raw.CorrelationID != "" {
		e.CorrelationID = raw.CorrelationID
	} else {
		e.CorrelationID = raw.TraceID
	}

	if raw.Payload != nil {
		e.Payload = raw.Payload
	} else {
		e.Payload = raw.Data
	}

	if raw.OccurredAt != nil {
		e.OccurredAt = *raw.OccurredAt
	} else if raw.Timestamp != nil {
		e.OccurredAt = *raw.Timestamp
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling to ensure ISO8601 formatting for OccurredAt and standard field keys.
func (e EventEnvelope) MarshalJSON() ([]byte, error) {
	type aliasEnvelope struct {
		EventID            string                 `json:"event_id"`
		OccurredAt         string                 `json:"occurred_at"`
		SchemaVersion      string                 `json:"schema_version"`
		TenantID           string                 `json:"tenant_id"`
		SubjectRef         string                 `json:"subject_ref"`
		CorrelationID      string                 `json:"correlation_id"`
		Payload            map[string]interface{} `json:"payload"`
		EventType          string                 `json:"event_type,omitempty"`
		Source             string                 `json:"source,omitempty"`
		DataClassification string                 `json:"data_classification,omitempty"`
		ContentHash        string                 `json:"content_hash,omitempty"`
	}

	var occurredAtStr string
	if !e.OccurredAt.IsZero() {
		occurredAtStr = e.OccurredAt.UTC().Format(time.RFC3339)
	}

	payload := e.Payload
	if payload == nil {
		payload = make(map[string]interface{})
	}

	return json.Marshal(aliasEnvelope{
		EventID:            e.EventID,
		OccurredAt:         occurredAtStr,
		SchemaVersion:      e.SchemaVersion,
		TenantID:           e.TenantID,
		SubjectRef:         e.SubjectRef,
		CorrelationID:      e.CorrelationID,
		Payload:            payload,
		EventType:          e.EventType,
		Source:             e.Source,
		DataClassification: e.DataClassification,
		ContentHash:        e.ContentHash,
	})
}

// ToDomain converts codec.EventEnvelope to domain.EventEnvelope.
func (e *EventEnvelope) ToDomain() *domain.EventEnvelope {
	return &domain.EventEnvelope{
		SchemaVersion:      e.SchemaVersion,
		EventID:            e.EventID,
		TraceID:            e.CorrelationID,
		EventType:          e.EventType,
		Source:             e.Source,
		Subject:            e.SubjectRef,
		Timestamp:          e.OccurredAt,
		ContentHash:        e.ContentHash,
		DataClassification: domain.DataClassification(e.DataClassification),
		Data:               e.Payload,
	}
}

// FromDomain creates a codec.EventEnvelope from domain.EventEnvelope.
func FromDomain(d *domain.EventEnvelope, tenantID string) *EventEnvelope {
	if d == nil {
		return nil
	}
	return &EventEnvelope{
		SchemaVersion:      d.SchemaVersion,
		EventID:            d.EventID,
		CorrelationID:      d.TraceID,
		EventType:          d.EventType,
		Source:             d.Source,
		SubjectRef:         d.Subject,
		OccurredAt:         d.Timestamp,
		ContentHash:        d.ContentHash,
		DataClassification: string(d.DataClassification),
		Payload:            d.Data,
		TenantID:           tenantID,
	}
}

// Codec manages encoding, decoding, and validation for EventEnvelopes.
type Codec struct {
	supportedVersions map[string]bool
	maxSizeBytes      int
}

// Option configures Codec options.
type Option func(*Codec)

// WithSupportedVersions sets allowed schema versions.
func WithSupportedVersions(versions ...string) Option {
	return func(c *Codec) {
		m := make(map[string]bool)
		for _, v := range versions {
			m[v] = true
		}
		c.supportedVersions = m
	}
}

// WithMaxSizeBytes sets maximum allowed payload size in bytes.
func WithMaxSizeBytes(size int) Option {
	return func(c *Codec) {
		if size > 0 {
			c.maxSizeBytes = size
		}
	}
}

// NewCodec constructs a Codec with default or custom options.
func NewCodec(opts ...Option) *Codec {
	c := &Codec{
		supportedVersions: map[string]bool{
			"1.0": true,
			"1.1": true,
			"v1":  true,
		},
		maxSizeBytes: DefaultMaxSizeBytes,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Encode validates and serializes an EventEnvelope to JSON.
func (c *Codec) Encode(env *EventEnvelope) ([]byte, error) {
	if env == nil {
		return nil, fmt.Errorf("%w: envelope is nil", ErrMissingField)
	}

	if env.SchemaVersion == "" {
		env.SchemaVersion = DefaultSchemaVersion
	}

	if env.ContentHash == "" && env.Payload != nil {
		hash, err := c.CalculateContentHash(env)
		if err == nil {
			env.ContentHash = hash
		}
	}

	if err := c.Validate(env); err != nil {
		return nil, err
	}

	data, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}

	if len(data) > c.maxSizeBytes {
		return nil, fmt.Errorf("%w: payload %d bytes exceeds max %d", ErrPayloadTooLarge, len(data), c.maxSizeBytes)
	}

	return data, nil
}

// Decode deserializes JSON data into an EventEnvelope and validates it.
func (c *Codec) Decode(data []byte) (*EventEnvelope, error) {
	if len(data) > c.maxSizeBytes {
		return nil, fmt.Errorf("%w: payload %d bytes exceeds max %d", ErrPayloadTooLarge, len(data), c.maxSizeBytes)
	}

	var env EventEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}

	if err := c.Validate(&env); err != nil {
		return nil, err
	}

	return &env, nil
}

// Validate checks that all required fields and version constraints are met.
func (c *Codec) Validate(env *EventEnvelope) error {
	if env == nil {
		return fmt.Errorf("%w: envelope is nil", ErrMissingField)
	}

	if env.SchemaVersion == "" {
		return fmt.Errorf("%w: schema_version is required", ErrMissingField)
	}

	if !c.supportedVersions[env.SchemaVersion] {
		return fmt.Errorf("%w: '%s'", ErrUnsupportedSchema, env.SchemaVersion)
	}

	if env.EventID == "" {
		return fmt.Errorf("%w: event_id is required", ErrMissingField)
	}

	if env.TenantID == "" {
		return fmt.Errorf("%w: tenant_id is required", ErrMissingField)
	}

	if env.SubjectRef == "" {
		return fmt.Errorf("%w: subject_ref is required", ErrMissingField)
	}

	if env.OccurredAt.IsZero() {
		return fmt.Errorf("%w: occurred_at timestamp is required", ErrMissingField)
	}

	if env.Payload == nil {
		return fmt.Errorf("%w: payload is required", ErrMissingField)
	}

	return nil
}

// CalculateContentHash produces a SHA256 hex string of the envelope payload.
func (c *Codec) CalculateContentHash(env *EventEnvelope) (string, error) {
	if env == nil || env.Payload == nil {
		return "", errors.New("cannot calculate content hash for nil payload")
	}

	payloadBytes, err := json.Marshal(env.Payload)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(payloadBytes)
	return hex.EncodeToString(hash[:]), nil
}
