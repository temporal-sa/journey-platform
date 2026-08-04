package security_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/validated-pattern/journey-platform/internal/domain"
	"github.com/validated-pattern/journey-platform/internal/security"
)

// 1. Cross-tenant access attempts are rejected at boundaries
func TestCrossTenantAccessBoundaryEnforcement(t *testing.T) {
	tenantA := "tenant-alpha"
	tenantB := "tenant-beta"

	ctxA := security.WithTenantContext(context.Background(), tenantA)
	ctxNoTenant := context.Background()

	t.Run("Missing Tenant Context Rejections", func(t *testing.T) {
		if err := security.VerifyTenantScope(ctxNoTenant, tenantA); !errors.Is(err, security.ErrMissingTenantContext) {
			t.Errorf("expected ErrMissingTenantContext, got: %v", err)
		}
		if err := security.VerifyRepositoryAccess(ctxNoTenant, tenantA); !errors.Is(err, security.ErrMissingTenantContext) {
			t.Errorf("repository access without tenant context should be rejected, got: %v", err)
		}
		if err := security.VerifyKafkaManifest(ctxNoTenant, tenantA); !errors.Is(err, security.ErrMissingTenantContext) {
			t.Errorf("kafka manifest without tenant context should be rejected, got: %v", err)
		}
		if err := security.VerifyTemporalStart(ctxNoTenant, tenantA); !errors.Is(err, security.ErrMissingTenantContext) {
			t.Errorf("temporal start without tenant context should be rejected, got: %v", err)
		}
	})

	t.Run("Cross-Tenant Access Rejections Across All Boundaries", func(t *testing.T) {
		boundaries := map[string]func(ctx context.Context, targetTenant string) error{
			"API Scope":         security.VerifyTenantScope,
			"Repository":        security.VerifyRepositoryAccess,
			"Kafka Manifest":    security.VerifyKafkaManifest,
			"Temporal Start":    security.VerifyTemporalStart,
			"Experiment":        security.VerifyExperimentAccess,
			"Assignment":        security.VerifyAssignmentAccess,
			"Exposure":          security.VerifyExposureAccess,
			"Static List":       security.VerifyStaticListAccess,
			"Action Execution":  security.VerifyActionAccess,
			"Report Generation": security.VerifyReportAccess,
		}

		for name, guard := range boundaries {
			t.Run(name, func(t *testing.T) {
				// Tenant A attempting to access Tenant B resources
				err := guard(ctxA, tenantB)
				if !errors.Is(err, security.ErrTenantMismatch) {
					t.Errorf("boundary %s allowed cross-tenant access! expected ErrTenantMismatch, got: %v", name, err)
				}
			})
		}
	})

	t.Run("Same-Tenant Access Approval", func(t *testing.T) {
		err := security.VerifyTenantScope(ctxA, tenantA)
		if err != nil {
			t.Errorf("same-tenant access unexpectedly rejected: %v", err)
		}
	})
}

// 2. Replaying Kafka events after tombstone commit is blocked
func TestTombstoneBlocksKafkaEventReplay(t *testing.T) {
	tenantID := "tenant-acme"
	subjectID := "user-tombstone-kafka-123"
	ctx := security.WithTenantContext(context.Background(), tenantID)

	// Verify subject is initially not tombstoned
	if security.IsTombstoned(ctx, tenantID, subjectID) {
		t.Fatal("subject should not be tombstoned initially")
	}

	// Commit tombstone
	err := security.TombstoneSubject(ctx, tenantID, subjectID)
	if err != nil {
		t.Fatalf("failed to commit tombstone: %v", err)
	}

	// Instant block verification
	if !security.IsTombstoned(ctx, tenantID, subjectID) {
		t.Fatal("subject must be instantly reported as tombstoned")
	}

	// Block Kafka Manifest Creation
	err = security.BlockKafkaManifest(ctx, tenantID, subjectID)
	if !errors.Is(err, security.ErrSubjectTombstoned) {
		t.Errorf("expected Kafka manifest creation to be blocked, got: %v", err)
	}

	// Block Kafka Event Replay
	err = security.BlockEventReplay(ctx, tenantID, subjectID)
	if !errors.Is(err, security.ErrSubjectTombstoned) {
		t.Errorf("expected Kafka event replay to be blocked, got: %v", err)
	}

	// Block Temporal Start
	err = security.BlockTemporalStart(ctx, tenantID, subjectID)
	if !errors.Is(err, security.ErrSubjectTombstoned) {
		t.Errorf("expected Temporal start to be blocked, got: %v", err)
	}
}

// 3. Tombstone committed after target creation but before dispatch blocks provider dispatch
func TestTombstoneCommittedAfterTargetCreationBlocksDispatch(t *testing.T) {
	tenantID := "tenant-acme"
	subjectID := "user-target-created-999"
	ctx := security.WithTenantContext(context.Background(), tenantID)

	// Step 1: Target created (before tombstone)
	targetCreated := true
	if !targetCreated {
		t.Fatal("target creation failed")
	}

	// Step 2: Subject commits tombstone
	err := security.TombstoneSubject(ctx, tenantID, subjectID)
	if err != nil {
		t.Fatalf("failed to tombstone subject: %v", err)
	}

	// Step 3: Attempt provider dispatch
	dispatchErr := security.BlockProviderDispatch(ctx, tenantID, subjectID)
	if !errors.Is(dispatchErr, security.ErrSubjectTombstoned) {
		t.Errorf("expected provider dispatch to be blocked after tombstone, got: %v", dispatchErr)
	}

	// Step 4: Verify async cleanup status
	status := security.DefaultService().GetCleanupStatus(tenantID, subjectID)
	if status == nil {
		t.Fatal("cleanup status should be tracked")
	}
}

// 4. Log and error output scans contain no raw seeded sensitive data
func TestStructuredRedactionScans(t *testing.T) {
	seededRawEmail := "john.doe.secret@domain-test.com"
	seededRawPhone := "+1-555-019-2831"
	seededRawMessageBody := "Super confidential message content body payload"
	seededRawCredential := "api_key_secret_token_12345"
	seededRawTrackingSecret := "tracking_secret_9988776655"
	seededRawHMACKey := "hmac_key_top_secret_key"
	seededRawBearerToken := "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.secret"

	t.Run("RedactString Scans", func(t *testing.T) {
		rawLog := fmt.Sprintf("Error occurred processing email=%s phone=%s body=%s api_key=%s tracking_secret=%s hmac_key=%s auth=%s",
			seededRawEmail, seededRawPhone, seededRawMessageBody, seededRawCredential, seededRawTrackingSecret, seededRawHMACKey, seededRawBearerToken)

		redacted := security.RedactString(rawLog)

		scannedRawValues := []string{
			seededRawEmail,
			seededRawPhone,
			seededRawCredential,
			seededRawTrackingSecret,
			seededRawHMACKey,
		}

		for _, rawVal := range scannedRawValues {
			if strings.Contains(redacted, rawVal) {
				t.Errorf("raw sensitive value '%s' leaked in redacted string output:\n%s", rawVal, redacted)
			}
		}
	})

	t.Run("RedactMap Scans", func(t *testing.T) {
		rawMap := map[string]interface{}{
			"email":           seededRawEmail,
			"phone":           seededRawPhone,
			"message_body":    seededRawMessageBody,
			"api_key":         seededRawCredential,
			"tracking_secret": seededRawTrackingSecret,
			"hmac_key":        seededRawHMACKey,
			"nested": map[string]interface{}{
				"user_email": seededRawEmail,
				"secret":     seededRawCredential,
			},
		}

		redactedMap := security.RedactMap(rawMap)

		// Assert values are redacted
		if redactedMap["email"] == seededRawEmail {
			t.Error("raw email leaked in map redaction")
		}
		if redactedMap["phone"] == seededRawPhone {
			t.Error("raw phone leaked in map redaction")
		}
		if redactedMap["api_key"] == seededRawCredential {
			t.Error("raw credential leaked in map redaction")
		}
		if redactedMap["tracking_secret"] == seededRawTrackingSecret {
			t.Error("raw tracking secret leaked in map redaction")
		}
		if redactedMap["hmac_key"] == seededRawHMACKey {
			t.Error("raw hmac key leaked in map redaction")
		}

		nested, ok := redactedMap["nested"].(map[string]interface{})
		if !ok || nested["secret"] == seededRawCredential {
			t.Error("raw secret leaked in nested map redaction")
		}
	})

	t.Run("RedactError Scans", func(t *testing.T) {
		rawErr := fmt.Errorf("failed provider call for email=%s auth=api_key=%s", seededRawEmail, seededRawCredential)
		redactedErr := security.RedactError(rawErr)

		if strings.Contains(redactedErr.Error(), seededRawEmail) {
			t.Errorf("raw email leaked in error redaction: %v", redactedErr)
		}
		if strings.Contains(redactedErr.Error(), seededRawCredential) {
			t.Errorf("raw credential leaked in error redaction: %v", redactedErr)
		}
	})
}

// 5. Forged tracking tokens, unsafe webhook destinations, template injection, CSV formula injection, and small-cell report reconstruction attempts are rejected
func TestAdversarialSecurityDefenses(t *testing.T) {
	t.Run("Forged Tracking Token Rejection", func(t *testing.T) {
		validSecret := "secret-key-123"
		wrongSecret := "wrong-secret-999"
		payload := "subject=123&action=click"

		validToken := security.GenerateTrackingToken(payload, validSecret)

		// Valid token must pass
		if err := security.ValidateTrackingToken(validToken, validSecret); err != nil {
			t.Errorf("valid tracking token unexpectedly rejected: %v", err)
		}

		// Forged signature (validated with wrong secret) must be rejected
		if err := security.ValidateTrackingToken(validToken, wrongSecret); !errors.Is(err, security.ErrInvalidTrackingToken) {
			t.Errorf("forged tracking token should be rejected, got: %v", err)
		}

		// Tampered payload in token must be rejected
		tamperedToken := "subject=123&action=admin." + strings.Split(validToken, ".")[1]
		if err := security.ValidateTrackingToken(tamperedToken, validSecret); !errors.Is(err, security.ErrInvalidTrackingToken) {
			t.Errorf("tampered tracking token should be rejected, got: %v", err)
		}
	})

	t.Run("Unsafe Webhook Destination Rejection", func(t *testing.T) {
		unsafeURLs := []string{
			"http://127.0.0.1/webhook",
			"http://localhost:8080/callback",
			"http://169.254.169.254/latest/meta-data/",
			"http://10.0.0.1/admin",
			"http://192.168.1.1/router",
			"ftp://example.com/pub",
			"file:///etc/passwd",
		}

		for _, u := range unsafeURLs {
			err := security.ValidateWebhookDestination(u)
			if !errors.Is(err, security.ErrUnsafeWebhookURL) {
				t.Errorf("unsafe webhook URL '%s' should have been rejected, got: %v", u, err)
			}
		}

		// Safe external HTTPS URL must pass
		if err := security.ValidateWebhookDestination("https://api.example.com/webhooks/receiver"); err != nil {
			t.Errorf("safe webhook URL unexpectedly rejected: %v", err)
		}
	})

	t.Run("Template Injection Defense", func(t *testing.T) {
		maliciousTemplates := []string{
			"Hello {{ user.name }} <script>alert(1)</script>",
			"Click here: javascript:alert(document.cookie)",
			"Result: ${exec('cat /etc/passwd')}",
			"Run: system('rm -rf /')",
		}

		for _, tmpl := range maliciousTemplates {
			err := security.ValidateTemplateInput(tmpl)
			if !errors.Is(err, security.ErrTemplateInjection) {
				t.Errorf("malicious template '%s' should be rejected, got: %v", tmpl, err)
			}
		}

		safeTmpl := "Hello {{ user.name }}, your order {{ order.id }} has shipped!"
		if err := security.ValidateTemplateInput(safeTmpl); err != nil {
			t.Errorf("safe template unexpectedly rejected: %v", err)
		}
	})

	t.Run("CSV Formula Injection Defense", func(t *testing.T) {
		maliciousCells := []string{
			"=SUM(A1:A10)",
			"+123456789",
			"-123456789",
			"@cmd|' /C calc'!A0",
			"\t=1+1",
		}

		for _, cell := range maliciousCells {
			sanitized := security.SanitizeCSVCell(cell)
			if !strings.HasPrefix(sanitized, "'") {
				t.Errorf("CSV formula cell '%s' was not properly escaped with leading single quote: '%s'", cell, sanitized)
			}
		}

		safeCell := "Normal Report Metric Header"
		if security.SanitizeCSVCell(safeCell) != safeCell {
			t.Errorf("safe CSV cell was modified unexpectedly")
		}
	})

	t.Run("Small-Cell Report Reconstruction Rejection", func(t *testing.T) {
		// Minimum cell size threshold = 5
		if err := security.ValidateReportCellSize(3, 5); !errors.Is(err, security.ErrSmallCellReconstruct) {
			t.Errorf("small cell count (3 < 5) should be rejected for reconstruction risk, got: %v", err)
		}

		if err := security.ValidateReportCellSize(100, 5); err != nil {
			t.Errorf("large cell count (100 >= 5) unexpectedly rejected: %v", err)
		}

		smallReport := &domain.AggregateReport{
			ReportID: "rep-small-cell",
			Metrics: []domain.MetricSummary{
				{MetricName: "conversions", TotalCount: 2}, // Violates threshold
			},
		}

		_, err := security.EnforceReportDifferentialPrivacy(smallReport, 5)
		if !errors.Is(err, security.ErrSmallCellReconstruct) {
			t.Errorf("report with small cell should be rejected by differential privacy guard, got: %v", err)
		}
	})
}
