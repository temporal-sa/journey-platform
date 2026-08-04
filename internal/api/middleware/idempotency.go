package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	HeaderIdempotencyKey  = "Idempotency-Key"
	HeaderXIdempotencyKey = "X-Idempotency-Key"
)

// IdempotencyRecord stores cached HTTP response details for an idempotency key.
type IdempotencyRecord struct {
	Key         string      `json:"key"`
	RequestHash string      `json:"request_hash"`
	StatusCode  int         `json:"status_code"`
	Headers     http.Header `json:"headers"`
	Body        []byte      `json:"body"`
	CreatedAt   time.Time   `json:"created_at"`
}

// IdempotencyStore defines storage interface for idempotency records.
type IdempotencyStore interface {
	Get(ctx context.Context, key string) (*IdempotencyRecord, bool, error)
	Save(ctx context.Context, record *IdempotencyRecord) error
}

// InMemoryIdempotencyStore is an in-memory thread-safe implementation of IdempotencyStore.
type InMemoryIdempotencyStore struct {
	mu      sync.RWMutex
	records map[string]*IdempotencyRecord
}

func NewInMemoryIdempotencyStore() *InMemoryIdempotencyStore {
	return &InMemoryIdempotencyStore{
		records: make(map[string]*IdempotencyRecord),
	}
}

func (s *InMemoryIdempotencyStore) Get(ctx context.Context, key string) (*IdempotencyRecord, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, exists := s.records[key]
	if !exists {
		return nil, false, nil
	}
	return rec, true, nil
}

func (s *InMemoryIdempotencyStore) Save(ctx context.Context, record *IdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[record.Key] = record
	return nil
}

type idempotencyResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (rw *idempotencyResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *idempotencyResponseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

// Idempotency middleware enforces transactional idempotency using request-hash conflict detection.
func Idempotency(store IdempotencyStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if store == nil {
				next.ServeHTTP(w, r)
				return
			}

			key := r.Header.Get(HeaderIdempotencyKey)
			if key == "" {
				key = r.Header.Get(HeaderXIdempotencyKey)
			}
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			var bodyBytes []byte
			if r.Body != nil {
				var err error
				bodyBytes, err = io.ReadAll(r.Body)
				if err != nil {
					WriteError(w, r, http.StatusBadRequest, "failed to read request body")
					return
				}
				r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}

			h := sha256.New()
			h.Write([]byte(r.Method))
			h.Write([]byte(":"))
			h.Write([]byte(r.URL.Path))
			h.Write([]byte(":"))
			h.Write(bodyBytes)
			reqHash := hex.EncodeToString(h.Sum(nil))

			ctx := r.Context()
			rec, found, err := store.Get(ctx, key)
			if err != nil {
				WriteError(w, r, http.StatusInternalServerError, "idempotency store error")
				return
			}

			if found && rec != nil {
				if rec.RequestHash == reqHash {
					for k, vv := range rec.Headers {
						for _, v := range vv {
							w.Header().Add(k, v)
						}
					}
					w.Header().Set("X-Cache", "IDEMPOTENT-HIT")
					w.WriteHeader(rec.StatusCode)
					_, _ = w.Write(rec.Body)
					return
				}

				WriteError(w, r, http.StatusConflict, "idempotency key reused with different request payload")
				return
			}

			rw := &idempotencyResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rw, r)

			// Cache response on non-server-error HTTP status codes
			if rw.statusCode < 500 {
				newRec := &IdempotencyRecord{
					Key:         key,
					RequestHash: reqHash,
					StatusCode:  rw.statusCode,
					Headers:     rw.Header().Clone(),
					Body:        rw.body.Bytes(),
					CreatedAt:   time.Now(),
				}
				_ = store.Save(ctx, newRec)
			}
		})
	}
}
