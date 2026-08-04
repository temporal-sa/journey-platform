package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
)

const (
	HeaderIfMatch     = "If-Match"
	HeaderIfNoneMatch = "If-None-Match"
	HeaderETag        = "ETag"
	etagKey           contextKey = "resource_etag"
)

// GenerateETag computes a strong SHA-256 ETag formatted with double quotes.
func GenerateETag(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf(`"%s"`, hex.EncodeToString(hash[:]))
}

// MatchETag evaluates whether ifMatch header matches the target resource ETag.
func MatchETag(ifMatchHeader, resourceETag string) bool {
	ifMatchHeader = strings.TrimSpace(ifMatchHeader)
	if ifMatchHeader == "" {
		return true
	}
	if ifMatchHeader == "*" {
		return true
	}
	resourceETag = strings.TrimSpace(resourceETag)

	tags := strings.Split(ifMatchHeader, ",")
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == resourceETag || strings.Trim(tag, `"`) == strings.Trim(resourceETag, `"`) {
			return true
		}
	}
	return false
}

// WithResourceETag attaches the target resource ETag to request context.
func WithResourceETag(ctx context.Context, etag string) context.Context {
	return context.WithValue(ctx, etagKey, etag)
}

// GetResourceETag retrieves the target resource ETag from context.
func GetResourceETag(ctx context.Context) string {
	if val, ok := ctx.Value(etagKey).(string); ok {
		return val
	}
	return ""
}

// CheckIfMatch checks if request's If-Match header matches resourceETag.
// If it fails to match, it writes a 412 Precondition Failed error and returns false.
func CheckIfMatch(w http.ResponseWriter, r *http.Request, resourceETag string) bool {
	ifMatch := r.Header.Get(HeaderIfMatch)
	if ifMatch != "" && !MatchETag(ifMatch, resourceETag) {
		WriteError(w, r, http.StatusPreconditionFailed, "precondition failed: ETag mismatch")
		return false
	}
	return true
}

// IfMatch returns a middleware that checks If-Match against context resource ETag if set.
func IfMatch(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ifMatch := r.Header.Get(HeaderIfMatch)
		if ifMatch != "" {
			ctxETag := GetResourceETag(r.Context())
			if ctxETag != "" && !MatchETag(ifMatch, ctxETag) {
				WriteError(w, r, http.StatusPreconditionFailed, "precondition failed: ETag mismatch")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
