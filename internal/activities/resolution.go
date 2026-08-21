package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/store/object"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

// DefaultLargePayloadThreshold specifies the default size boundary in bytes above which resolved values are offloaded to object storage.
const DefaultLargePayloadThreshold = 1024

// ResolutionStatus represents the explicit status of a resolved key.
type ResolutionStatus string

const (
	ResolutionStatusResolved    ResolutionStatus = "resolved"
	ResolutionStatusMissing     ResolutionStatus = "missing"
	ResolutionStatusNull        ResolutionStatus = "null"
	ResolutionStatusStale       ResolutionStatus = "stale"
	ResolutionStatusUnavailable ResolutionStatus = "unavailable"
	ResolutionStatusDisallowed  ResolutionStatus = "disallowed"
)

// FreshnessStatus describes the freshness state of the resolved value.
type FreshnessStatus string

const (
	FreshnessFresh   FreshnessStatus = "fresh"
	FreshnessStale   FreshnessStatus = "stale"
	FreshnessUnknown FreshnessStatus = "unknown"
)

// ResolutionKey defines a single requested attribute or parameter key specification.
type ResolutionKey struct {
	Key         string        `json:"key"`
	Type        string        `json:"type,omitempty"`
	MaxAge      time.Duration `json:"max_age,omitempty"`
	Fallback    interface{}   `json:"fallback,omitempty"`
	HasFallback bool          `json:"has_fallback,omitempty"`
}

// ResolveAttributeInput represents activity inputs for subject attribute resolution.
type ResolveAttributeInput struct {
	SubjectRef string          `json:"subject_ref"`
	PayloadRef string          `json:"payload_ref,omitempty"`
	Keys       []ResolutionKey `json:"keys"`
}

// ResolveParameterInput represents activity inputs for shared parameter resolution.
type ResolveParameterInput struct {
	SubjectRef string          `json:"subject_ref"`
	PayloadRef string          `json:"payload_ref,omitempty"`
	Keys       []ResolutionKey `json:"keys"`
}

// ResolvedValue contains value resolution details, data metadata, freshness, and offload references.
type ResolvedValue struct {
	Key                string                    `json:"key"`
	Value              interface{}               `json:"value,omitempty"`
	Status             ResolutionStatus          `json:"status"`
	Freshness          FreshnessStatus           `json:"freshness"`
	ObservedAt         time.Time                 `json:"observed_at"`
	SourceVersion      string                    `json:"source_version,omitempty"`
	DataClassification domain.DataClassification `json:"data_classification,omitempty"`
	ValueRef           string                    `json:"value_ref,omitempty"`
	IsRef              bool                      `json:"is_ref"`
	ErrorMessage       string                    `json:"error_message,omitempty"`
}

// ResolveResult represents the collection of resolved attribute or parameter outputs.
type ResolveResult struct {
	SubjectRef string                   `json:"subject_ref"`
	PayloadRef string                   `json:"payload_ref,omitempty"`
	Values     map[string]ResolvedValue `json:"values"`
}

// RawResolvedValue holds low-level resolution data provided by underlying resolvers.
type RawResolvedValue struct {
	Value              interface{}
	IsNull             bool
	ObservedAt         time.Time
	SourceVersion      string
	DataClassification domain.DataClassification
	IsDisallowed       bool
	IsUnavailable      bool
	IsMissing          bool
}

// AttributeResolver resolves local subject attributes.
type AttributeResolver interface {
	ResolveAttribute(ctx context.Context, subjectRef string, payloadRef string, key string) (*RawResolvedValue, error)
}

// ParameterResolver resolves shared parameters.
type ParameterResolver interface {
	ResolveParameter(ctx context.Context, subjectRef string, payloadRef string, key string) (*RawResolvedValue, error)
}

// MockAttributeResolver provides an in-memory AttributeResolver implementation for testing.
type MockAttributeResolver struct {
	mu         sync.RWMutex
	attributes map[string]map[string]*RawResolvedValue
}

// NewMockAttributeResolver constructs a new MockAttributeResolver.
func NewMockAttributeResolver() *MockAttributeResolver {
	return &MockAttributeResolver{
		attributes: make(map[string]map[string]*RawResolvedValue),
	}
}

// SetAttribute sets or overrides an attribute value for a subject in the mock resolver.
func (m *MockAttributeResolver) SetAttribute(subjectRef, key string, val *RawResolvedValue) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.attributes[subjectRef] == nil {
		m.attributes[subjectRef] = make(map[string]*RawResolvedValue)
	}
	m.attributes[subjectRef][key] = val
}

// ResolveAttribute resolves subject attributes from in-memory storage.
func (m *MockAttributeResolver) ResolveAttribute(ctx context.Context, subjectRef, payloadRef, key string) (*RawResolvedValue, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if subMap, found := m.attributes[subjectRef]; found {
		if val, ok := subMap[key]; ok {
			return val, nil
		}
	}
	return &RawResolvedValue{IsMissing: true}, nil
}

// MockParameterResolver provides an in-memory ParameterResolver implementation for testing.
type MockParameterResolver struct {
	mu         sync.RWMutex
	parameters map[string]*RawResolvedValue
}

// NewMockParameterResolver constructs a new MockParameterResolver.
func NewMockParameterResolver() *MockParameterResolver {
	return &MockParameterResolver{
		parameters: make(map[string]*RawResolvedValue),
	}
}

// SetParameter sets or overrides a shared parameter value in the mock resolver.
func (m *MockParameterResolver) SetParameter(key string, val *RawResolvedValue) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.parameters[key] = val
}

// ResolveParameter resolves shared parameters from in-memory storage.
func (m *MockParameterResolver) ResolveParameter(ctx context.Context, subjectRef, payloadRef, key string) (*RawResolvedValue, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if val, ok := m.parameters[key]; ok {
		return val, nil
	}
	return &RawResolvedValue{IsMissing: true}, nil
}

// ResolveAttributes resolves requested local subject attributes.
func (a *Activities) ResolveAttributes(ctx context.Context, input ResolveAttributeInput) (ResolveResult, error) {
	if activity.IsActivity(ctx) {
		logger := activity.GetLogger(ctx)
		logger.Info("Resolving attributes", "subjectRef", input.SubjectRef, "count", len(input.Keys))
	}
	info := ActivityInfo{
		ActivityName: "ResolveAttributes",
		NodeID:       input.SubjectRef,
	}
	if err := a.getBehaviorInjector().Intercept(ctx, info); err != nil {
		return ResolveResult{}, err
	}
	return a.resolveKeys(ctx, input.SubjectRef, input.PayloadRef, input.Keys, true)
}

// ResolveParameters resolves requested shared parameters.
func (a *Activities) ResolveParameters(ctx context.Context, input ResolveParameterInput) (ResolveResult, error) {
	if activity.IsActivity(ctx) {
		logger := activity.GetLogger(ctx)
		logger.Info("Resolving parameters", "subjectRef", input.SubjectRef, "count", len(input.Keys))
	}
	info := ActivityInfo{
		ActivityName: "ResolveParameters",
		NodeID:       input.SubjectRef,
	}
	if err := a.getBehaviorInjector().Intercept(ctx, info); err != nil {
		return ResolveResult{}, err
	}
	return a.resolveKeys(ctx, input.SubjectRef, input.PayloadRef, input.Keys, false)
}

func (a *Activities) resolveKeys(ctx context.Context, subjectRef, payloadRef string, keys []ResolutionKey, isAttribute bool) (ResolveResult, error) {
	res := ResolveResult{
		SubjectRef: subjectRef,
		PayloadRef: payloadRef,
		Values:     make(map[string]ResolvedValue),
	}

	var payloadData map[string]interface{}
	if payloadRef != "" && a.objectStore != nil {
		if rc, _, err := a.objectStore.GetObject(ctx, payloadRef); err == nil {
			buf, readErr := io.ReadAll(rc)
			_ = rc.Close()
			if readErr == nil {
				_ = json.Unmarshal(buf, &payloadData)
			}
		}
	}

	for _, keyReq := range keys {
		var raw *RawResolvedValue
		var err error

		if isAttribute {
			if a.attributeResolver != nil {
				raw, err = a.attributeResolver.ResolveAttribute(ctx, subjectRef, payloadRef, keyReq.Key)
			}
		} else {
			if a.parameterResolver != nil {
				raw, err = a.parameterResolver.ResolveParameter(ctx, subjectRef, payloadRef, keyReq.Key)
			}
		}

		if err != nil {
			raw = &RawResolvedValue{IsUnavailable: true}
		}

		if raw == nil {
			raw = &RawResolvedValue{IsMissing: true}
		}

		// Fallback to payloadData if missing from primary resolver
		if (raw.IsMissing || raw.IsNull) && payloadData != nil {
			if val, ok := payloadData[keyReq.Key]; ok {
				raw = &RawResolvedValue{
					Value:              val,
					IsNull:             val == nil,
					ObservedAt:         time.Now().UTC(),
					SourceVersion:      "payload",
					DataClassification: domain.DataClassificationNonPII,
				}
			}
		}

		// Disallowed field requests fail non-retryably without data exposure
		if raw.IsDisallowed {
			return ResolveResult{}, temporal.NewNonRetryableApplicationError(
				fmt.Sprintf("access disallowed for requested key: %s", keyReq.Key),
				"ErrDisallowedKey",
				nil,
			)
		}

		var status ResolutionStatus
		var freshness FreshnessStatus = FreshnessUnknown
		var val interface{}

		switch {
		case raw.IsMissing:
			status = ResolutionStatusMissing
			freshness = FreshnessUnknown
			if keyReq.HasFallback {
				val = keyReq.Fallback
			} else {
				val = nil
			}
		case raw.IsUnavailable:
			status = ResolutionStatusUnavailable
			freshness = FreshnessUnknown
			if keyReq.HasFallback {
				val = keyReq.Fallback
			} else {
				val = nil
			}
		case raw.IsNull:
			status = ResolutionStatusNull
			if !raw.ObservedAt.IsZero() {
				if keyReq.MaxAge > 0 && time.Since(raw.ObservedAt) > keyReq.MaxAge {
					freshness = FreshnessStale
				} else {
					freshness = FreshnessFresh
				}
			} else {
				freshness = FreshnessFresh
			}
			if keyReq.HasFallback {
				val = keyReq.Fallback
			} else {
				val = nil
			}
		default:
			// Check staleness if observed timestamp and max age are present
			if !raw.ObservedAt.IsZero() && keyReq.MaxAge > 0 && time.Since(raw.ObservedAt) > keyReq.MaxAge {
				status = ResolutionStatusStale
				freshness = FreshnessStale
				if keyReq.HasFallback {
					val = keyReq.Fallback
				} else {
					val = raw.Value
				}
			} else {
				status = ResolutionStatusResolved
				if !raw.ObservedAt.IsZero() {
					freshness = FreshnessFresh
				} else {
					freshness = FreshnessFresh
				}
				val = raw.Value
			}
		}

		// Handle offloading large payloads to ObjectStore
		threshold := a.largeThreshold
		if threshold <= 0 {
			threshold = DefaultLargePayloadThreshold
		}

		var isRef bool
		var valueRef string

		if val != nil && a.objectStore != nil {
			jsonBytes, err := json.Marshal(val)
			if err == nil && len(jsonBytes) > threshold {
				refKey := fmt.Sprintf("resolution/%s/%s-%d.json", subjectRef, keyReq.Key, time.Now().UnixNano())
				info, putErr := a.objectStore.PutObject(ctx, refKey, bytes.NewReader(jsonBytes), object.PutOptions{
					TenantID:    "resolution",
					ContentType: "application/json",
				})
				if putErr == nil && info != nil {
					isRef = true
					valueRef = info.Key
					val = nil
				}
			}
		}

		resVal := ResolvedValue{
			Key:                keyReq.Key,
			Value:              val,
			Status:             status,
			Freshness:          freshness,
			ObservedAt:         raw.ObservedAt,
			SourceVersion:      raw.SourceVersion,
			DataClassification: raw.DataClassification,
			IsRef:              isRef,
			ValueRef:           valueRef,
		}

		res.Values[keyReq.Key] = resVal
	}

	return res, nil
}
