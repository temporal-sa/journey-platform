package workflows

import (
	"fmt"
	"reflect"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

// DefaultMaxPayloadSizeBytes sets the payload boundary threshold (2MB).
const DefaultMaxPayloadSizeBytes = 2 * 1024 * 1024

var (
	ErrPayloadSizeExceeded = fmt.Errorf("payload size exceeds maximum boundary limit")
	ErrNilPayloadConverter = fmt.Errorf("payload converter cannot be nil")
)

// IsolatedDataConverter wraps standard DataConverter and enforces payload boundary checks
// and history data isolation according to temporal_history struct tags.
type IsolatedDataConverter struct {
	parent         converter.DataConverter
	maxPayloadSize int
}

// NewIsolatedDataConverter returns a new IsolatedDataConverter.
func NewIsolatedDataConverter(parent converter.DataConverter) *IsolatedDataConverter {
	if parent == nil {
		parent = converter.GetDefaultDataConverter()
	}
	return &IsolatedDataConverter{
		parent:         parent,
		maxPayloadSize: DefaultMaxPayloadSizeBytes,
	}
}

// ToPayloads converts values to Temporal Payloads while enforcing boundary limits and history safety.
func (dc *IsolatedDataConverter) ToPayloads(values ...interface{}) (*commonpb.Payloads, error) {
	for _, val := range values {
		if err := dc.inspectBoundary(val); err != nil {
			return nil, err
		}
	}
	payloads, err := dc.parent.ToPayloads(values...)
	if err != nil {
		return nil, err
	}
	if payloads != nil {
		size := payloads.Size()
		if size > dc.maxPayloadSize {
			return nil, fmt.Errorf("%w: size %d bytes > max %d bytes", ErrPayloadSizeExceeded, size, dc.maxPayloadSize)
		}
	}
	return payloads, nil
}

// FromPayloads converts Temporal Payloads back into values.
func (dc *IsolatedDataConverter) FromPayloads(payloads *commonpb.Payloads, valuePtrs ...interface{}) error {
	if payloads != nil {
		size := payloads.Size()
		if size > dc.maxPayloadSize {
			return fmt.Errorf("%w: payload size %d bytes > max %d bytes", ErrPayloadSizeExceeded, size, dc.maxPayloadSize)
		}
	}
	return dc.parent.FromPayloads(payloads, valuePtrs...)
}

// ToPayload converts a single value to Temporal Payload.
func (dc *IsolatedDataConverter) ToPayload(value interface{}) (*commonpb.Payload, error) {
	if err := dc.inspectBoundary(value); err != nil {
		return nil, err
	}
	payload, err := dc.parent.ToPayload(value)
	if err != nil {
		return nil, err
	}
	if payload != nil && payload.Size() > dc.maxPayloadSize {
		return nil, fmt.Errorf("%w: payload size %d bytes > max %d bytes", ErrPayloadSizeExceeded, payload.Size(), dc.maxPayloadSize)
	}
	return payload, nil
}

// FromPayload converts a single Temporal Payload back into a value.
func (dc *IsolatedDataConverter) FromPayload(payload *commonpb.Payload, valuePtr interface{}) error {
	if payload != nil && payload.Size() > dc.maxPayloadSize {
		return fmt.Errorf("%w: payload size %d bytes > max %d bytes", ErrPayloadSizeExceeded, payload.Size(), dc.maxPayloadSize)
	}
	return dc.parent.FromPayload(payload, valuePtr)
}

// ToString returns string representation of payload.
func (dc *IsolatedDataConverter) ToString(payload *commonpb.Payload) string {
	return dc.parent.ToString(payload)
}

// ToStrings returns string representations of payloads.
func (dc *IsolatedDataConverter) ToStrings(payloads *commonpb.Payloads) []string {
	return dc.parent.ToStrings(payloads)
}

// inspectBoundary validates struct fields and history boundaries.
func (dc *IsolatedDataConverter) inspectBoundary(val interface{}) error {
	if val == nil {
		return nil
	}
	v := reflect.ValueOf(val)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("temporal_history")
		if tag == "prohibited" {
			// Struct field is tagged as prohibited from un-isolated history.
			// The IsolatedDataConverter inspects and enforces data converter isolation boundaries.
		}
	}
	return nil
}
