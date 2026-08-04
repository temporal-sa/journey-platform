package object

import (
	"context"
	"errors"
	"io"
	"time"
)

var (
	// ErrObjectNotFound indicates that the requested object does not exist or has expired.
	ErrObjectNotFound = errors.New("object not found")

	// ErrObjectFinalized indicates that an operation cannot be performed because the object is finalized and immutable.
	ErrObjectFinalized = errors.New("object is finalized and immutable")

	// ErrChecksumMismatch indicates that the computed SHA-256 checksum does not match the expected checksum.
	ErrChecksumMismatch = errors.New("checksum mismatch")

	// ErrSizeLimitExceeded indicates that the object size exceeds the specified or default maximum limit.
	ErrSizeLimitExceeded = errors.New("size limit exceeded")

	// ErrPathTraversal indicates that the object key or prefix contains illegal path traversal components.
	ErrPathTraversal = errors.New("path traversal rejected")

	// ErrInvalidKey indicates that the provided object key is empty or invalid.
	ErrInvalidKey = errors.New("invalid object key")
)

// ObjectInfo contains metadata about a stored object.
type ObjectInfo struct {
	Key         string            `json:"key"`
	Size        int64             `json:"size"`
	Checksum    string            `json:"checksum"` // Hex-encoded SHA-256
	TenantID    string            `json:"tenant_id,omitempty"`
	TestRunID   string            `json:"test_run_id,omitempty"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
	Finalized   bool              `json:"finalized"`
	CreatedAt   time.Time         `json:"created_at"`
	ContentType string            `json:"content_type,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// PutOptions provides configurable options when putting an object into storage.
type PutOptions struct {
	TenantID         string
	TestRunID        string
	Size             int64  // Expected size (0 if unknown)
	MaxSize          int64  // Maximum allowed size in bytes (0 for store default / unlimited)
	Checksum         string // Expected hex SHA-256 checksum to verify
	ContentAddressed bool   // If true, key is generated as content SHA-256 hex string
	Finalized        bool   // If true, object is marked immutable immediately
	TTL              time.Duration
	ExpiresAt        *time.Time
	ContentType      string
	Metadata         map[string]string
}

// ObjectStore defines the unified interface for artifact and static-list object storage.
type ObjectStore interface {
	PutObject(ctx context.Context, key string, data io.Reader, opts PutOptions) (*ObjectInfo, error)
	GetObject(ctx context.Context, key string) (io.ReadCloser, *ObjectInfo, error)
	DeleteObject(ctx context.Context, key string) error
	FinalizeObject(ctx context.Context, key string) error
	ObjectExists(ctx context.Context, key string) (bool, error)
}
