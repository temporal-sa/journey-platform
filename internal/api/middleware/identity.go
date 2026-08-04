package middleware

import (
	"context"
	"net/http"
)

type contextKey string

const (
	tenantIDKey contextKey = "tenant_id"
	userIDKey   contextKey = "user_id"

	HeaderTenantID = "X-Tenant-ID"
	HeaderUserID   = "X-User-ID"
)

// WithTenantID returns a new context with the tenant ID.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// GetTenantID retrieves the tenant ID from context.
func GetTenantID(ctx context.Context) string {
	if val, ok := ctx.Value(tenantIDKey).(string); ok {
		return val
	}
	return ""
}

// WithUserID returns a new context with the user ID.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID retrieves the user ID from context.
func GetUserID(ctx context.Context) string {
	if val, ok := ctx.Value(userIDKey).(string); ok {
		return val
	}
	return ""
}

// Identity middleware extracts X-Tenant-ID and X-User-ID from headers and injects them into request context.
func Identity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if tenantID := r.Header.Get(HeaderTenantID); tenantID != "" {
			ctx = WithTenantID(ctx, tenantID)
		}
		if userID := r.Header.Get(HeaderUserID); userID != "" {
			ctx = WithUserID(ctx, userID)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
