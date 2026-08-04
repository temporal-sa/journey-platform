package security

import (
	"context"
	"errors"

	"github.com/validated-pattern/journey-platform/internal/api/middleware"
)

type tenantCtxKey string

const (
	TenantContextKey tenantCtxKey = "tenant_id"

	HeaderTenantID = "X-Tenant-ID"
)

var (
	ErrMissingTenantContext = errors.New("tenant context required: missing tenant ID in context")
	ErrTenantMismatch       = errors.New("tenant scope mismatch: cross-tenant access denied")
)

// WithTenantContext injects a tenant ID into the context.
func WithTenantContext(ctx context.Context, tenantID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = middleware.WithTenantID(ctx, tenantID)
	return context.WithValue(ctx, TenantContextKey, tenantID)
}

// GetTenantID retrieves the tenant ID from context if present.
func GetTenantID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if val := middleware.GetTenantID(ctx); val != "" {
		return val
	}
	if val, ok := ctx.Value(TenantContextKey).(string); ok && val != "" {
		return val
	}
	if val, ok := ctx.Value("tenant_id").(string); ok && val != "" {
		return val
	}
	return ""
}

// RequireTenantContext verifies that context contains a non-empty tenant ID.
func RequireTenantContext(ctx context.Context) (string, error) {
	tenantID := GetTenantID(ctx)
	if tenantID == "" {
		return "", ErrMissingTenantContext
	}
	return tenantID, nil
}

// VerifyTenantScope checks that context contains a valid tenant ID and matches targetTenantID (if target is specified).
func VerifyTenantScope(ctx context.Context, targetTenantID string) error {
	tenantID, err := RequireTenantContext(ctx)
	if err != nil {
		return err
	}
	if targetTenantID != "" && tenantID != targetTenantID {
		return ErrTenantMismatch
	}
	return nil
}

// -----------------------------------------------------------------------------
// Component Boundary Verification Guards
// -----------------------------------------------------------------------------

// VerifyRepositoryAccess enforces tenant scope for database/repository operations.
func VerifyRepositoryAccess(ctx context.Context, targetTenantID string) error {
	return VerifyTenantScope(ctx, targetTenantID)
}

// VerifyKafkaManifest enforces tenant scope for Kafka manifest creation.
func VerifyKafkaManifest(ctx context.Context, targetTenantID string) error {
	return VerifyTenantScope(ctx, targetTenantID)
}

// VerifyTemporalStart enforces tenant scope for Temporal workflow starts.
func VerifyTemporalStart(ctx context.Context, targetTenantID string) error {
	return VerifyTenantScope(ctx, targetTenantID)
}

// VerifyAssignmentAccess enforces tenant scope for experiment assignment operations.
func VerifyAssignmentAccess(ctx context.Context, targetTenantID string) error {
	return VerifyTenantScope(ctx, targetTenantID)
}

// VerifyExposureAccess enforces tenant scope for exposure logging operations.
func VerifyExposureAccess(ctx context.Context, targetTenantID string) error {
	return VerifyTenantScope(ctx, targetTenantID)
}

// VerifyActionAccess enforces tenant scope for action dispatch operations.
func VerifyActionAccess(ctx context.Context, targetTenantID string) error {
	return VerifyTenantScope(ctx, targetTenantID)
}

// VerifyReportAccess enforces tenant scope for aggregate report operations.
func VerifyReportAccess(ctx context.Context, targetTenantID string) error {
	return VerifyTenantScope(ctx, targetTenantID)
}

// VerifyExperimentAccess enforces tenant scope for experiment definition operations.
func VerifyExperimentAccess(ctx context.Context, targetTenantID string) error {
	return VerifyTenantScope(ctx, targetTenantID)
}

// VerifyStaticListAccess enforces tenant scope for static audience list operations.
func VerifyStaticListAccess(ctx context.Context, targetTenantID string) error {
	return VerifyTenantScope(ctx, targetTenantID)
}
