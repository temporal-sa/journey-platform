package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/validated-pattern/journey-platform/internal/domain"
)

// NodeFieldMapping maps IR node fields back to UI node IDs and field names.
type NodeFieldMapping struct {
	UINodeID string            `json:"ui_node_id"`
	Fields   map[string]string `json:"fields,omitempty"`
}

// SourceMap maps compiled IR instructions back to UI node and field IDs.
type SourceMap struct {
	Version string                      `json:"version"`
	Nodes   map[string]NodeFieldMapping `json:"nodes"`
	Edges   map[string]string           `json:"edges,omitempty"`
}

// EncodeIR encodes a CompiledIR struct into canonical JSON bytes.
func EncodeIR(ir *domain.CompiledIR) ([]byte, error) {
	if ir == nil {
		return nil, fmt.Errorf("compiled IR is nil")
	}
	return EncodeCanonicalJSON(ir)
}

// DecodeIR decodes canonical JSON bytes into a CompiledIR struct and validates required fields.
func DecodeIR(data []byte) (*domain.CompiledIR, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data for IR decoding")
	}
	var ir domain.CompiledIR
	if err := json.Unmarshal(data, &ir); err != nil {
		return nil, fmt.Errorf("failed to decode IR JSON: %w", err)
	}

	// Basic validation of required IR fields
	if ir.SchemaVersion == "" {
		ir.SchemaVersion = domain.DefaultSchemaVersion
	}
	if ir.IRID == "" {
		return nil, fmt.Errorf("invalid IR: missing ir_id")
	}
	if ir.DraftID == "" {
		return nil, fmt.Errorf("invalid IR: missing draft_id")
	}
	if ir.TenantID == "" {
		return nil, fmt.Errorf("invalid IR: missing tenant_id")
	}
	if ir.Nodes == nil {
		ir.Nodes = []domain.IRNode{}
	}
	if ir.Edges == nil {
		ir.Edges = []domain.IREdge{}
	}

	return &ir, nil
}

// ComputeIRHash computes the deterministic SHA-256 hash of a CompiledIR (excluding content_hash field).
func ComputeIRHash(ir *domain.CompiledIR) (string, error) {
	if ir == nil {
		return "", fmt.Errorf("compiled IR is nil")
	}
	clone := *ir
	clone.ContentHash = ""
	bytes, err := EncodeCanonicalJSON(clone)
	if err != nil {
		return "", fmt.Errorf("failed to marshal IR for hash calculation: %w", err)
	}
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:]), nil
}

// VerifyIRHash verifies whether ir.ContentHash matches the computed SHA-256 hash.
func VerifyIRHash(ir *domain.CompiledIR) (bool, error) {
	if ir == nil {
		return false, fmt.Errorf("compiled IR is nil")
	}
	computed, err := ComputeIRHash(ir)
	if err != nil {
		return false, err
	}
	return computed == ir.ContentHash, nil
}

// EncodeSourceMap encodes a SourceMap into canonical JSON bytes.
func EncodeSourceMap(sm *SourceMap) ([]byte, error) {
	if sm == nil {
		return nil, fmt.Errorf("source map is nil")
	}
	return EncodeCanonicalJSON(sm)
}

// DecodeSourceMap decodes JSON bytes into a SourceMap struct.
func DecodeSourceMap(data []byte) (*SourceMap, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data for source map decoding")
	}
	var sm SourceMap
	if err := json.Unmarshal(data, &sm); err != nil {
		return nil, fmt.Errorf("failed to decode SourceMap JSON: %w", err)
	}
	if sm.Nodes == nil {
		sm.Nodes = make(map[string]NodeFieldMapping)
	}
	if sm.Edges == nil {
		sm.Edges = make(map[string]string)
	}
	return &sm, nil
}

// EncodeCanonicalJSON encodes any value to JSON with deterministic key sorting.
func EncodeCanonicalJSON(v interface{}) ([]byte, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal canonical JSON: %w", err)
	}
	return bytes, nil
}
