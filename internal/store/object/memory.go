package object

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"sync"
	"time"
)

type storedObject struct {
	info ObjectInfo
	data []byte
}

// MemoryStore implements an in-memory ObjectStore adapter for testing and light local execution.
type MemoryStore struct {
	mu      sync.RWMutex
	objects map[string]*storedObject
	maxSize int64
}

// NewMemoryStore constructs a new in-memory object store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		objects: make(map[string]*storedObject),
	}
}

// NewMemoryStoreWithMaxSize constructs a new in-memory object store with a default size limit.
func NewMemoryStoreWithMaxSize(maxSize int64) *MemoryStore {
	return &MemoryStore{
		objects: make(map[string]*storedObject),
		maxSize: maxSize,
	}
}

func (m *MemoryStore) PutObject(ctx context.Context, key string, data io.Reader, opts PutOptions) (*ObjectInfo, error) {
	buf, err := io.ReadAll(data)
	if err != nil {
		return nil, err
	}

	maxAllowed := opts.MaxSize
	if maxAllowed == 0 {
		maxAllowed = m.maxSize
	}
	if maxAllowed > 0 && int64(len(buf)) > maxAllowed {
		return nil, ErrSizeLimitExceeded
	}

	hash := sha256.Sum256(buf)
	sha256Hex := hex.EncodeToString(hash[:])

	if opts.Checksum != "" {
		if !strings.EqualFold(sha256Hex, opts.Checksum) {
			return nil, ErrChecksumMismatch
		}
	}

	baseKey := key
	if opts.ContentAddressed || key == "" {
		baseKey = sha256Hex
	}

	fullKey, err := BuildKey(opts.TenantID, opts.TestRunID, baseKey)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, found := m.objects[fullKey]; found {
		if existing.info.ExpiresAt == nil || time.Now().Before(*existing.info.ExpiresAt) {
			if existing.info.Finalized {
				return nil, ErrObjectFinalized
			}
		}
	}

	var expiresAt *time.Time
	if opts.ExpiresAt != nil {
		expiresAt = opts.ExpiresAt
	} else if opts.TTL > 0 {
		t := time.Now().Add(opts.TTL)
		expiresAt = &t
	}

	info := ObjectInfo{
		Key:         fullKey,
		Size:        int64(len(buf)),
		Checksum:    sha256Hex,
		TenantID:    opts.TenantID,
		TestRunID:   opts.TestRunID,
		ExpiresAt:   expiresAt,
		Finalized:   opts.Finalized,
		CreatedAt:   time.Now().UTC(),
		ContentType: opts.ContentType,
		Metadata:    copyMetadata(opts.Metadata),
	}

	m.objects[fullKey] = &storedObject{
		info: info,
		data: buf,
	}

	infoCopy := copyObjectInfo(&info)
	return &infoCopy, nil
}

func (m *MemoryStore) GetObject(ctx context.Context, key string) (io.ReadCloser, *ObjectInfo, error) {
	if err := ValidateKey(key); err != nil {
		return nil, nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	obj, found := m.objects[key]
	if !found {
		return nil, nil, ErrObjectNotFound
	}

	if obj.info.ExpiresAt != nil && time.Now().After(*obj.info.ExpiresAt) {
		return nil, nil, ErrObjectNotFound
	}

	infoCopy := copyObjectInfo(&obj.info)
	return io.NopCloser(bytes.NewReader(obj.data)), &infoCopy, nil
}

func (m *MemoryStore) DeleteObject(ctx context.Context, key string) error {
	if err := ValidateKey(key); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	obj, found := m.objects[key]
	if !found {
		return ErrObjectNotFound
	}

	if obj.info.ExpiresAt != nil && time.Now().After(*obj.info.ExpiresAt) {
		delete(m.objects, key)
		return ErrObjectNotFound
	}

	if obj.info.Finalized {
		return ErrObjectFinalized
	}

	delete(m.objects, key)
	return nil
}

func (m *MemoryStore) FinalizeObject(ctx context.Context, key string) error {
	if err := ValidateKey(key); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	obj, found := m.objects[key]
	if !found {
		return ErrObjectNotFound
	}

	if obj.info.ExpiresAt != nil && time.Now().After(*obj.info.ExpiresAt) {
		delete(m.objects, key)
		return ErrObjectNotFound
	}

	obj.info.Finalized = true
	return nil
}

func (m *MemoryStore) ObjectExists(ctx context.Context, key string) (bool, error) {
	if err := ValidateKey(key); err != nil {
		return false, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	obj, found := m.objects[key]
	if !found {
		return false, nil
	}

	if obj.info.ExpiresAt != nil && time.Now().After(*obj.info.ExpiresAt) {
		return false, nil
	}

	return true, nil
}

func copyMetadata(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	cp := make(map[string]string, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

func copyObjectInfo(info *ObjectInfo) ObjectInfo {
	cp := *info
	if info.ExpiresAt != nil {
		t := *info.ExpiresAt
		cp.ExpiresAt = &t
	}
	cp.Metadata = copyMetadata(info.Metadata)
	return cp
}
