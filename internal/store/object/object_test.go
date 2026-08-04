package object

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func TestMemoryStore_Conformance(t *testing.T) {
	store := NewMemoryStore()
	runConformanceTestSuite(t, store)
}

func TestMinIOStore_Conformance(t *testing.T) {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "127.0.0.1:9000"
	}

	accessKey := os.Getenv("MINIO_ROOT_USER")
	if accessKey == "" {
		accessKey = "minioadmin"
	}

	secretKey := os.Getenv("MINIO_ROOT_PASSWORD")
	if secretKey == "" {
		secretKey = "minioadmin"
	}

	ctx := context.Background()
	bucket := "journey-test-conformance"

	cfg := MinIOConfig{
		Endpoint:        endpoint,
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		UseSSL:          false,
		Bucket:          bucket,
		AutoCreate:      true,
	}

	store, err := NewMinIOStore(ctx, cfg)
	if err != nil {
		t.Skipf("MinIO endpoint %s not reachable or network restricted (%v). Skipping MinIO conformance tests.", endpoint, err)
		return
	}

	runConformanceTestSuite(t, store)
}

func runConformanceTestSuite(t *testing.T, store ObjectStore) {
	ctx := context.Background()

	t.Run("Basic Put, Get, Exists, Delete", func(t *testing.T) {
		key := "conformance/basic.txt"
		data := []byte("hello world conformance")
		hash := sha256.Sum256(data)
		expectedChecksum := hex.EncodeToString(hash[:])

		info, err := store.PutObject(ctx, key, bytes.NewReader(data), PutOptions{
			ContentType: "text/plain",
			Metadata:    map[string]string{"env": "test"},
		})
		if err != nil {
			t.Fatalf("PutObject failed: %v", err)
		}

		if info.Key != key {
			t.Errorf("expected key %s, got %s", key, info.Key)
		}
		if info.Size != int64(len(data)) {
			t.Errorf("expected size %d, got %d", len(data), info.Size)
		}
		if info.Checksum != expectedChecksum {
			t.Errorf("expected checksum %s, got %s", expectedChecksum, info.Checksum)
		}
		if info.Finalized {
			t.Errorf("expected object not to be finalized")
		}

		exists, err := store.ObjectExists(ctx, key)
		if err != nil || !exists {
			t.Fatalf("ObjectExists expected true, got %v, err %v", exists, err)
		}

		rc, getInfo, err := store.GetObject(ctx, key)
		if err != nil {
			t.Fatalf("GetObject failed: %v", err)
		}
		defer rc.Close()

		readBuf, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("reading object body failed: %v", err)
		}
		if !bytes.Equal(readBuf, data) {
			t.Errorf("expected content %q, got %q", string(data), string(readBuf))
		}
		if getInfo.Checksum != expectedChecksum {
			t.Errorf("expected getInfo checksum %s, got %s", expectedChecksum, getInfo.Checksum)
		}
		if getInfo.Metadata["env"] != "test" {
			t.Errorf("expected metadata env=test, got %v", getInfo.Metadata)
		}

		if err := store.DeleteObject(ctx, key); err != nil {
			t.Fatalf("DeleteObject failed: %v", err)
		}

		exists, err = store.ObjectExists(ctx, key)
		if err != nil || exists {
			t.Fatalf("ObjectExists expected false after delete, got %v, err %v", exists, err)
		}

		_, _, err = store.GetObject(ctx, key)
		if !errors.Is(err, ErrObjectNotFound) {
			t.Errorf("expected ErrObjectNotFound, got %v", err)
		}
	})

	t.Run("Tenant and Test-Run Prefixes", func(t *testing.T) {
		data := []byte("tenant data")
		info, err := store.PutObject(ctx, "artifact.log", bytes.NewReader(data), PutOptions{
			TenantID:  "tenant-100",
			TestRunID: "run-200",
		})
		if err != nil {
			t.Fatalf("PutObject failed: %v", err)
		}

		expectedKey := "tenant-100/run-200/artifact.log"
		if info.Key != expectedKey {
			t.Fatalf("expected key %s, got %s", expectedKey, info.Key)
		}

		rc, _, err := store.GetObject(ctx, expectedKey)
		if err != nil {
			t.Fatalf("GetObject with full prefixed key failed: %v", err)
		}
		rc.Close()

		_ = store.DeleteObject(ctx, expectedKey)
	})

	t.Run("Content-Addressed Keys", func(t *testing.T) {
		data := []byte("sha256 content addressing test")
		hash := sha256.Sum256(data)
		expectedHash := hex.EncodeToString(hash[:])

		info, err := store.PutObject(ctx, "", bytes.NewReader(data), PutOptions{
			TenantID:         "tenant-ca",
			ContentAddressed: true,
		})
		if err != nil {
			t.Fatalf("PutObject content-addressed failed: %v", err)
		}

		expectedKey := "tenant-ca/" + expectedHash
		if info.Key != expectedKey {
			t.Fatalf("expected key %s, got %s", expectedKey, info.Key)
		}
		if info.Checksum != expectedHash {
			t.Errorf("expected checksum %s, got %s", expectedHash, info.Checksum)
		}

		exists, err := store.ObjectExists(ctx, expectedKey)
		if err != nil || !exists {
			t.Fatalf("expected content-addressed key to exist")
		}

		_ = store.DeleteObject(ctx, expectedKey)
	})

	t.Run("Size Limit Enforcement", func(t *testing.T) {
		key := "conformance/oversized.bin"
		data := bytes.Repeat([]byte("A"), 100)

		_, err := store.PutObject(ctx, key, bytes.NewReader(data), PutOptions{
			MaxSize: 50,
		})
		if !errors.Is(err, ErrSizeLimitExceeded) {
			t.Fatalf("expected ErrSizeLimitExceeded, got %v", err)
		}

		exists, err := store.ObjectExists(ctx, key)
		if err != nil || exists {
			t.Fatalf("oversized object should not exist, exists=%v", exists)
		}
	})

	t.Run("Checksum Verification", func(t *testing.T) {
		key := "conformance/checksum.txt"
		data := []byte("checksum verification test")

		_, err := store.PutObject(ctx, key, bytes.NewReader(data), PutOptions{
			Checksum: "0000000000000000000000000000000000000000000000000000000000000000",
		})
		if !errors.Is(err, ErrChecksumMismatch) {
			t.Fatalf("expected ErrChecksumMismatch, got %v", err)
		}

		hash := sha256.Sum256(data)
		validChecksum := hex.EncodeToString(hash[:])

		info, err := store.PutObject(ctx, key, bytes.NewReader(data), PutOptions{
			Checksum: validChecksum,
		})
		if err != nil {
			t.Fatalf("PutObject with valid checksum failed: %v", err)
		}
		_ = store.DeleteObject(ctx, info.Key)
	})

	t.Run("Immutability and Finalization", func(t *testing.T) {
		key := "conformance/immutable.txt"
		dataV1 := []byte("version 1")
		dataV2 := []byte("version 2")

		info, err := store.PutObject(ctx, key, bytes.NewReader(dataV1), PutOptions{
			Finalized: false,
		})
		if err != nil {
			t.Fatalf("PutObject failed: %v", err)
		}

		if err := store.FinalizeObject(ctx, info.Key); err != nil {
			t.Fatalf("FinalizeObject failed: %v", err)
		}

		rc, getInfo, err := store.GetObject(ctx, info.Key)
		if err != nil {
			t.Fatalf("GetObject failed after finalize: %v", err)
		}
		rc.Close()
		if !getInfo.Finalized {
			t.Errorf("expected getInfo.Finalized to be true")
		}

		_, err = store.PutObject(ctx, info.Key, bytes.NewReader(dataV2), PutOptions{})
		if !errors.Is(err, ErrObjectFinalized) {
			t.Fatalf("expected ErrObjectFinalized on PutObject overwrite, got %v", err)
		}

		err = store.DeleteObject(ctx, info.Key)
		if !errors.Is(err, ErrObjectFinalized) {
			t.Fatalf("expected ErrObjectFinalized on DeleteObject, got %v", err)
		}

		key2 := "conformance/immutable-direct.txt"
		info2, err := store.PutObject(ctx, key2, bytes.NewReader(dataV1), PutOptions{
			Finalized: true,
		})
		if err != nil {
			t.Fatalf("PutObject with Finalized=true failed: %v", err)
		}

		_, err = store.PutObject(ctx, info2.Key, bytes.NewReader(dataV2), PutOptions{})
		if !errors.Is(err, ErrObjectFinalized) {
			t.Fatalf("expected ErrObjectFinalized on direct finalized overwrite, got %v", err)
		}
	})

	t.Run("Expiry Metadata", func(t *testing.T) {
		key := "conformance/expiring.txt"
		data := []byte("ephemeral data")

		_, err := store.PutObject(ctx, key, bytes.NewReader(data), PutOptions{
			TTL: 100 * time.Millisecond,
		})
		if err != nil {
			t.Fatalf("PutObject failed: %v", err)
		}

		exists, err := store.ObjectExists(ctx, key)
		if err != nil || !exists {
			t.Fatalf("expected object to exist immediately after Put")
		}

		time.Sleep(150 * time.Millisecond)

		exists, err = store.ObjectExists(ctx, key)
		if err != nil || exists {
			t.Fatalf("expected object to be expired, exists=%v", exists)
		}

		_, _, err = store.GetObject(ctx, key)
		if !errors.Is(err, ErrObjectNotFound) {
			t.Fatalf("expected ErrObjectNotFound for expired object, got %v", err)
		}
	})

	t.Run("Path Traversal Rejection", func(t *testing.T) {
		traversalKeys := []string{
			"../etc/passwd",
			"foo/../../bar",
			"/absolute/path",
			"foo\\bar",
			"folder/..",
			"folder/.",
			`..\win.ini`,
		}

		data := []byte("malicious attempt")

		for _, tk := range traversalKeys {
			_, err := store.PutObject(ctx, tk, bytes.NewReader(data), PutOptions{})
			if !errors.Is(err, ErrPathTraversal) && !errors.Is(err, ErrInvalidKey) {
				t.Errorf("PutObject(%q) expected ErrPathTraversal or ErrInvalidKey, got %v", tk, err)
			}

			_, _, err = store.GetObject(ctx, tk)
			if !errors.Is(err, ErrPathTraversal) && !errors.Is(err, ErrInvalidKey) {
				t.Errorf("GetObject(%q) expected ErrPathTraversal or ErrInvalidKey, got %v", tk, err)
			}

			err = store.DeleteObject(ctx, tk)
			if !errors.Is(err, ErrPathTraversal) && !errors.Is(err, ErrInvalidKey) {
				t.Errorf("DeleteObject(%q) expected ErrPathTraversal or ErrInvalidKey, got %v", tk, err)
			}

			err = store.FinalizeObject(ctx, tk)
			if !errors.Is(err, ErrPathTraversal) && !errors.Is(err, ErrInvalidKey) {
				t.Errorf("FinalizeObject(%q) expected ErrPathTraversal or ErrInvalidKey, got %v", tk, err)
			}

			_, err = store.ObjectExists(ctx, tk)
			if !errors.Is(err, ErrPathTraversal) && !errors.Is(err, ErrInvalidKey) {
				t.Errorf("ObjectExists(%q) expected ErrPathTraversal or ErrInvalidKey, got %v", tk, err)
			}
		}

		_, err := store.PutObject(ctx, "valid.txt", bytes.NewReader(data), PutOptions{
			TenantID: "../badtenant",
		})
		if !errors.Is(err, ErrPathTraversal) {
			t.Errorf("PutObject with invalid tenant prefix expected ErrPathTraversal, got %v", err)
		}
	})
}

func TestKeyValidationHelpers(t *testing.T) {
	if err := ValidateKey("valid/file.txt"); err != nil {
		t.Errorf("unexpected error for valid key: %v", err)
	}

	if err := ValidateKey(""); !errors.Is(err, ErrInvalidKey) {
		t.Errorf("expected ErrInvalidKey for empty key, got %v", err)
	}

	if err := ValidateKey("../etc/passwd"); !errors.Is(err, ErrPathTraversal) {
		t.Errorf("expected ErrPathTraversal, got %v", err)
	}

	key, err := BuildKey("tenant1", "run1", "file.log")
	if err != nil {
		t.Fatalf("BuildKey failed: %v", err)
	}
	if key != "tenant1/run1/file.log" {
		t.Errorf("unexpected key %s", key)
	}

	_, err = BuildKey("../badtenant", "run1", "file.log")
	if !errors.Is(err, ErrPathTraversal) {
		t.Errorf("expected ErrPathTraversal from BuildKey, got %v", err)
	}
}

func TestMinIOStoreConstructor(t *testing.T) {
	ctx := context.Background()

	client, err := minio.New("invalid.endpoint.local:9000", &minio.Options{
		Creds: credentials.NewStaticV4("key", "secret", ""),
	})
	if err != nil {
		t.Fatalf("minio.New unexpectedly failed: %v", err)
	}

	store := NewMinIOStoreWithClient(client, "my-bucket", 1024)
	if store.bucket != "my-bucket" || store.defaultMaxSize != 1024 {
		t.Errorf("unexpected store settings")
	}

	_, err = NewMinIOStore(ctx, MinIOConfig{
		Endpoint:   "127.0.0.1:59999",
		Bucket:     "nonexistent-bucket",
		AutoCreate: true,
	})
	if err == nil {
		t.Errorf("expected error connecting to invalid endpoint with AutoCreate=true, got nil")
	}
}
