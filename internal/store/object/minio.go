package object

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOConfig holds configuration for initializing MinIO object storage.
type MinIOConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	Bucket          string
	AutoCreate      bool
	DefaultMaxSize  int64
}

// MinIOStore implements ObjectStore backed by MinIO / S3-compatible storage.
type MinIOStore struct {
	client         *minio.Client
	bucket         string
	defaultMaxSize int64
}

// NewMinIOStore creates a new MinIOStore instance with the specified configuration.
func NewMinIOStore(ctx context.Context, cfg MinIOConfig) (*MinIOStore, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	store := &MinIOStore{
		client:         client,
		bucket:         cfg.Bucket,
		defaultMaxSize: cfg.DefaultMaxSize,
	}

	if cfg.AutoCreate {
		exists, err := client.BucketExists(ctx, cfg.Bucket)
		if err != nil {
			return nil, fmt.Errorf("failed to check bucket existence: %w", err)
		}
		if !exists {
			err = client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to create bucket: %w", err)
			}
		}
	}

	return store, nil
}

// NewMinIOStoreWithClient constructs a MinIOStore using an existing minio.Client.
func NewMinIOStoreWithClient(client *minio.Client, bucket string, defaultMaxSize int64) *MinIOStore {
	return &MinIOStore{
		client:         client,
		bucket:         bucket,
		defaultMaxSize: defaultMaxSize,
	}
}

func (m *MinIOStore) PutObject(ctx context.Context, key string, data io.Reader, opts PutOptions) (*ObjectInfo, error) {
	buf, err := io.ReadAll(data)
	if err != nil {
		return nil, err
	}

	maxAllowed := opts.MaxSize
	if maxAllowed == 0 {
		maxAllowed = m.defaultMaxSize
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

	// Check if existing object exists and is finalized
	stat, err := m.client.StatObject(ctx, m.bucket, fullKey, minio.StatObjectOptions{})
	if err == nil {
		_, _, _, finalized, expiresAt, _ := parseUserMetadata(stat.UserMetadata)
		if expiresAt == nil || time.Now().Before(*expiresAt) {
			if finalized {
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

	userMeta := map[string]string{
		"tenant-id":   opts.TenantID,
		"test-run-id": opts.TestRunID,
		"checksum":    sha256Hex,
		"finalized":   fmt.Sprintf("%t", opts.Finalized),
	}
	if expiresAt != nil {
		userMeta["expires-at"] = expiresAt.Format(time.RFC3339Nano)
	}
	if len(opts.Metadata) > 0 {
		metaBytes, _ := json.Marshal(opts.Metadata)
		userMeta["custom-metadata"] = string(metaBytes)
	}

	putOpts := minio.PutObjectOptions{
		ContentType:  opts.ContentType,
		UserMetadata: userMeta,
	}

	_, err = m.client.PutObject(ctx, m.bucket, fullKey, bytes.NewReader(buf), int64(len(buf)), putOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to put object in minio: %w", err)
	}

	info := &ObjectInfo{
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

	return info, nil
}

func (m *MinIOStore) GetObject(ctx context.Context, key string) (io.ReadCloser, *ObjectInfo, error) {
	if err := ValidateKey(key); err != nil {
		return nil, nil, err
	}

	stat, err := m.client.StatObject(ctx, m.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if isMinIONotFound(err) {
			return nil, nil, ErrObjectNotFound
		}
		return nil, nil, fmt.Errorf("failed to stat object: %w", err)
	}

	tenantID, testRunID, checksum, finalized, expiresAt, customMeta := parseUserMetadata(stat.UserMetadata)
	if expiresAt != nil && time.Now().After(*expiresAt) {
		return nil, nil, ErrObjectNotFound
	}

	obj, err := m.client.GetObject(ctx, m.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		if isMinIONotFound(err) {
			return nil, nil, ErrObjectNotFound
		}
		return nil, nil, fmt.Errorf("failed to get object: %w", err)
	}

	info := &ObjectInfo{
		Key:         key,
		Size:        stat.Size,
		Checksum:    checksum,
		TenantID:    tenantID,
		TestRunID:   testRunID,
		ExpiresAt:   expiresAt,
		Finalized:   finalized,
		CreatedAt:   stat.LastModified.UTC(),
		ContentType: stat.ContentType,
		Metadata:    customMeta,
	}

	return obj, info, nil
}

func (m *MinIOStore) DeleteObject(ctx context.Context, key string) error {
	if err := ValidateKey(key); err != nil {
		return err
	}

	stat, err := m.client.StatObject(ctx, m.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if isMinIONotFound(err) {
			return ErrObjectNotFound
		}
		return fmt.Errorf("failed to stat object before deletion: %w", err)
	}

	_, _, _, finalized, expiresAt, _ := parseUserMetadata(stat.UserMetadata)
	if expiresAt != nil && time.Now().After(*expiresAt) {
		_ = m.client.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{})
		return ErrObjectNotFound
	}

	if finalized {
		return ErrObjectFinalized
	}

	err = m.client.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		if isMinIONotFound(err) {
			return ErrObjectNotFound
		}
		return fmt.Errorf("failed to remove object: %w", err)
	}

	return nil
}

func (m *MinIOStore) FinalizeObject(ctx context.Context, key string) error {
	if err := ValidateKey(key); err != nil {
		return err
	}

	stat, err := m.client.StatObject(ctx, m.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if isMinIONotFound(err) {
			return ErrObjectNotFound
		}
		return fmt.Errorf("failed to stat object for finalization: %w", err)
	}

	_, _, _, finalized, expiresAt, _ := parseUserMetadata(stat.UserMetadata)
	if expiresAt != nil && time.Now().After(*expiresAt) {
		_ = m.client.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{})
		return ErrObjectNotFound
	}

	if finalized {
		return nil
	}

	updatedUserMeta := make(map[string]string)
	for k, v := range stat.UserMetadata {
		updatedUserMeta[k] = v
	}
	updatedUserMeta["finalized"] = "true"

	src := minio.CopySrcOptions{
		Bucket: m.bucket,
		Object: key,
	}
	dst := minio.CopyDestOptions{
		Bucket:       m.bucket,
		Object:       key,
		UserMetadata: updatedUserMeta,
	}

	_, err = m.client.CopyObject(ctx, dst, src)
	if err != nil {
		return fmt.Errorf("failed to copy object for finalization: %w", err)
	}

	return nil
}

func (m *MinIOStore) ObjectExists(ctx context.Context, key string) (bool, error) {
	if err := ValidateKey(key); err != nil {
		return false, err
	}

	stat, err := m.client.StatObject(ctx, m.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if isMinIONotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat object: %w", err)
	}

	_, _, _, _, expiresAt, _ := parseUserMetadata(stat.UserMetadata)
	if expiresAt != nil && time.Now().After(*expiresAt) {
		return false, nil
	}

	return true, nil
}

func isMinIONotFound(err error) bool {
	if err == nil {
		return false
	}
	errResp := minio.ToErrorResponse(err)
	if errResp.StatusCode == http.StatusNotFound {
		return true
	}
	code := strings.ToLower(errResp.Code)
	return code == "nosuchkey" || code == "notfound" || code == "resourcenotfound"
}

func getCaseInsensitive(m map[string]string, target string) (string, bool) {
	for k, v := range m {
		if strings.EqualFold(k, target) {
			return v, true
		}
	}
	return "", false
}

func parseUserMetadata(userMeta map[string]string) (tenantID, testRunID, checksum string, finalized bool, expiresAt *time.Time, customMeta map[string]string) {
	if v, ok := getCaseInsensitive(userMeta, "tenant-id"); ok {
		tenantID = v
	}
	if v, ok := getCaseInsensitive(userMeta, "test-run-id"); ok {
		testRunID = v
	}
	if v, ok := getCaseInsensitive(userMeta, "checksum"); ok {
		checksum = v
	}
	if v, ok := getCaseInsensitive(userMeta, "finalized"); ok {
		finalized = strings.EqualFold(v, "true")
	}
	if v, ok := getCaseInsensitive(userMeta, "expires-at"); ok && v != "" {
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			expiresAt = &t
		}
	}
	if v, ok := getCaseInsensitive(userMeta, "custom-metadata"); ok && v != "" {
		var meta map[string]string
		if err := json.Unmarshal([]byte(v), &meta); err == nil {
			customMeta = meta
		}
	}
	return
}
