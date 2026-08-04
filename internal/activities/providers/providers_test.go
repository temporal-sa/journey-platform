package providers_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/activities"
	"github.com/validated-pattern/journey-platform/internal/activities/providers"
)

// Mock DNS resolver for testing DNS rebinding scenarios
type MockDNSResolver struct {
	IPs map[string][]net.IP
}

func (m *MockDNSResolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	if ips, ok := m.IPs[host]; ok {
		return ips, nil
	}
	return net.DefaultResolver.LookupIP(ctx, network, host)
}

// ---------------------------------------------------------------------------
// 1. Provider Contract Suite Tests
// ---------------------------------------------------------------------------

func TestProviderContractSuite(t *testing.T) {
	ledger := providers.NewRequestLedger()
	reg := providers.NewFakeProviderRegistry(ledger)

	emailProvider := &providers.EmailProvider{Registry: reg}
	smsProvider := &providers.SMSProvider{Registry: reg}
	pushProvider := &providers.PushProvider{Registry: reg}
	inAppProvider := &providers.InAppProvider{Registry: reg}

	tests := []struct {
		name             string
		provider         activities.ActionProvider
		channel          string
		behavior         string
		expectedStatus   activities.LedgerStatus
		expectedReason   string
		expectReconcile  activities.LedgerStatus
	}{
		{
			name:            "Email Provider Success",
			provider:        emailProvider,
			channel:         "email",
			behavior:        providers.BehaviorSuccess,
			expectedStatus:  activities.LedgerStatusAccepted,
			expectedReason:  "DELIVERED",
			expectReconcile: activities.LedgerStatusAccepted,
		},
		{
			name:            "SMS Provider Retryable Failure",
			provider:        smsProvider,
			channel:         "sms",
			behavior:        providers.BehaviorRetryableFailure,
			expectedStatus:  activities.LedgerStatusRetryableFailure,
			expectedReason:  "SERVICE_UNAVAILABLE",
			expectReconcile: activities.LedgerStatusRetryableFailure,
		},
		{
			name:            "Push Provider Permanent Failure",
			provider:        pushProvider,
			channel:         "push",
			behavior:        providers.BehaviorPermanentFailure,
			expectedStatus:  activities.LedgerStatusPermanentFailure,
			expectedReason:  "INVALID_RECIPIENT",
			expectReconcile: activities.LedgerStatusPermanentFailure,
		},
		{
			name:            "InApp Provider Timeout After Acceptance",
			provider:        inAppProvider,
			channel:         "in_app",
			behavior:        providers.BehaviorTimeoutAfterAcceptance,
			expectedStatus:  activities.LedgerStatusAccepted,
			expectedReason:  "ACCEPTED_PENDING",
			expectReconcile: activities.LedgerStatusRetryableFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			idempotencyKey := "idempotent_" + tt.name
			req := &activities.ProviderRequest{
				IdempotencyKey:   idempotencyKey,
				TenantID:         "tenant_test_1",
				SubjectRef:       "user_123",
				RecipientAddress: "user@example.com",
				Channel:          tt.channel,
				RenderedContent:  "Hello World",
				Metadata: map[string]interface{}{
					"behavior": tt.behavior,
				},
			}

			resp, err := tt.provider.Dispatch(ctx, req)
			if err != nil {
				t.Fatalf("unexpected dispatch error: %v", err)
			}
			if resp.Status != tt.expectedStatus {
				t.Errorf("expected status %v, got %v", tt.expectedStatus, resp.Status)
			}
			if resp.ReasonCode != tt.expectedReason {
				t.Errorf("expected reason code %s, got %s", tt.expectedReason, resp.ReasonCode)
			}

			recResp, err := tt.provider.Reconcile(ctx, idempotencyKey)
			if err != nil {
				t.Fatalf("unexpected reconcile error: %v", err)
			}
			if recResp.Status != tt.expectReconcile {
				t.Errorf("expected reconcile status %v, got %v", tt.expectReconcile, recResp.Status)
			}
		})
	}
}

func TestProviderCallbackDelayAndDuplicates(t *testing.T) {
	ledger := providers.NewRequestLedger()
	reg := providers.NewFakeProviderRegistry(ledger)
	emailProvider := &providers.EmailProvider{Registry: reg}

	t.Run("Callback Delay Enforcement", func(t *testing.T) {
		ctx := context.Background()
		start := time.Now()
		req := &activities.ProviderRequest{
			IdempotencyKey:   "delay_key_1",
			TenantID:         "tenant_1",
			RecipientAddress: "user@example.com",
			Channel:          "email",
			RenderedContent:  "Test Delay",
			Metadata: map[string]interface{}{
				"behavior":       providers.BehaviorCallbackDelay,
				"callback_delay": "100ms",
			},
		}

		resp, err := emailProvider.Dispatch(ctx, req)
		elapsed := time.Since(start)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Status != activities.LedgerStatusAccepted {
			t.Errorf("expected accepted status, got %v", resp.Status)
		}
		if elapsed < 80*time.Millisecond {
			t.Errorf("expected delay of at least 80ms, got %v", elapsed)
		}
	})

	t.Run("Duplicate Callback Generation", func(t *testing.T) {
		ledger.Clear()
		ctx := context.Background()
		req := &activities.ProviderRequest{
			IdempotencyKey:   "dup_key_1",
			TenantID:         "tenant_1",
			RecipientAddress: "user@example.com",
			Channel:          "email",
			RenderedContent:  "Test Duplicate",
			Metadata: map[string]interface{}{
				"behavior":           providers.BehaviorSuccess,
				"duplicate_callback": true,
			},
		}

		_, err := emailProvider.Dispatch(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		entries := ledger.GetEntries("tenant_1", "email")
		if len(entries) != 2 {
			t.Fatalf("expected 2 ledger entries for duplicate callback, got %d", len(entries))
		}
		if entries[0].ID == entries[1].ID {
			t.Errorf("expected unique IDs for duplicate entries")
		}
	})
}

func TestInspectableRequestLedger(t *testing.T) {
	ledger := providers.NewRequestLedger()
	ledger.Record(providers.LedgerEntry{
		ID:               "msg_1",
		TenantID:         "tenant_A",
		Channel:          "email",
		RecipientAddress: "a@test.com",
		Status:           activities.LedgerStatusAccepted,
	})
	ledger.Record(providers.LedgerEntry{
		ID:               "msg_2",
		TenantID:         "tenant_B",
		Channel:          "sms",
		RecipientAddress: "+15550001",
		Status:           activities.LedgerStatusAccepted,
	})

	if len(ledger.AllEntries()) != 2 {
		t.Errorf("expected 2 entries, got %d", len(ledger.AllEntries()))
	}
	if len(ledger.GetEntries("tenant_A", "")) != 1 {
		t.Errorf("expected 1 entry for tenant_A, got %d", len(ledger.GetEntries("tenant_A", "")))
	}

	ledger.Clear()
	if len(ledger.AllEntries()) != 0 {
		t.Errorf("expected 0 entries after clear, got %d", len(ledger.AllEntries()))
	}
}

// ---------------------------------------------------------------------------
// 2. Tracked Link Token Security Tests
// ---------------------------------------------------------------------------

func TestTrackedLinkSecurity(t *testing.T) {
	manager := providers.NewTrackedLinkManager([]byte("super-secret-hmac-key"))

	t.Run("Valid Token Lifecycle", func(t *testing.T) {
		token := providers.TrackedLinkToken{
			TenantID:       "tenant_100",
			Purpose:        "email_click",
			DestinationURL: "https://example.com/promo",
			Version:        "v1",
			ExpiresAt:      time.Now().Add(1 * time.Hour),
		}

		tokenStr, err := manager.Sign(token)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		verified, err := manager.Verify(tokenStr, "tenant_100")
		if err != nil {
			t.Fatalf("expected verification success, got error: %v", err)
		}
		if verified.DestinationURL != "https://example.com/promo" {
			t.Errorf("expected destination URL https://example.com/promo, got %s", verified.DestinationURL)
		}
	})

	t.Run("Token Forgery Detection", func(t *testing.T) {
		token := providers.TrackedLinkToken{
			TenantID:       "tenant_100",
			Purpose:        "email_click",
			DestinationURL: "https://example.com/promo",
			ExpiresAt:      time.Now().Add(1 * time.Hour),
		}

		tokenStr, err := manager.Sign(token)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		parts := strings.Split(tokenStr, ".")
		tamperedTokenStr := parts[0] + ".invalidSignatureB64"

		_, err = manager.Verify(tamperedTokenStr, "tenant_100")
		if err != providers.ErrInvalidSignature {
			t.Errorf("expected ErrInvalidSignature for forged token, got %v", err)
		}
	})

	t.Run("Token Expiry Rejection", func(t *testing.T) {
		expiredToken := providers.TrackedLinkToken{
			TenantID:       "tenant_100",
			Purpose:        "email_click",
			DestinationURL: "https://example.com/promo",
			ExpiresAt:      time.Now().Add(-10 * time.Minute),
		}

		tokenStr, err := manager.Sign(expiredToken)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		_, err = manager.Verify(tokenStr, "tenant_100")
		if err != providers.ErrTokenExpired {
			t.Errorf("expected ErrTokenExpired for expired token, got %v", err)
		}
	})

	t.Run("Wrong Tenant Rejection", func(t *testing.T) {
		token := providers.TrackedLinkToken{
			TenantID:       "tenant_100",
			Purpose:        "email_click",
			DestinationURL: "https://example.com/promo",
			ExpiresAt:      time.Now().Add(1 * time.Hour),
		}

		tokenStr, err := manager.Sign(token)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		_, err = manager.Verify(tokenStr, "tenant_OTHER_200")
		if err != providers.ErrWrongTenant {
			t.Errorf("expected ErrWrongTenant when tenant ID mismatches, got %v", err)
		}
	})

	t.Run("Single-Use Replay Constraint Prevention", func(t *testing.T) {
		token := providers.TrackedLinkToken{
			TenantID:         "tenant_100",
			Purpose:          "email_click",
			DestinationURL:   "https://example.com/promo",
			ExpiresAt:        time.Now().Add(1 * time.Hour),
			ReplayConstraint: "single_use",
			Nonce:            "nonce_unique_12345",
		}

		tokenStr, err := manager.Sign(token)
		if err != nil {
			t.Fatalf("failed to sign token: %v", err)
		}

		_, err = manager.Verify(tokenStr, "tenant_100")
		if err != nil {
			t.Fatalf("first verification failed: %v", err)
		}

		_, err = manager.Verify(tokenStr, "tenant_100")
		if err != providers.ErrTokenReplayed {
			t.Errorf("expected ErrTokenReplayed on second attempt, got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// 3. Webhook Security Tests (SSRF & Safe Webhook Handler)
// ---------------------------------------------------------------------------

func TestWebhookSecurityRangesAndBypass(t *testing.T) {
	client := providers.NewSafeWebhookClient(nil, 2*time.Second)
	ctx := context.Background()

	restrictedTargets := []struct {
		name string
		url  string
	}{
		{"IPv4 Private 10.x", "http://10.0.0.1/webhook"},
		{"IPv4 Private 172.16.x", "http://172.16.0.5/webhook"},
		{"IPv4 Private 192.168.x", "http://192.168.1.100/webhook"},
		{"IPv4 Loopback 127.0.0.1", "http://127.0.0.1:8080/webhook"},
		{"IPv6 Loopback ::1", "http://[::1]:8080/webhook"},
		{"IPv6 Link-Local fe80::", "http://[fe80::1]/webhook"},
		{"IPv6 Unique Local fc00::", "http://[fc00::1]/webhook"},
		{"IPv4-Mapped IPv6 ::ffff:127.0.0.1", "http://[::ffff:127.0.0.1]/webhook"},
		{"Cloud Metadata Endpoint 169.254.169.254", "http://169.254.169.254/latest/meta-data/"},

		// Loopback Bypass Encodings
		{"Octal Loopback 0177.0.0.1", "http://0177.0.0.1/webhook"},
		{"Octal Dword 017700000001", "http://017700000001/webhook"},
		{"Hex Loopback 0x7f000001", "http://0x7f000001/webhook"},
		{"Hex Dotted 0x7f.0.0.1", "http://0x7f.0.0.1/webhook"},
		{"Integer/Dword 2130706433", "http://2130706433/webhook"},
		{"Localhost Name", "http://localhost:8080/webhook"},
	}

	for _, tt := range restrictedTargets {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := client.ValidateTarget(ctx, tt.url, "")
			if err != providers.ErrRestrictedIPAddress {
				t.Errorf("expected ErrRestrictedIPAddress for %s (%s), got %v", tt.name, tt.url, err)
			}
		})
	}
}

func TestApprovedDestinationIDs(t *testing.T) {
	client := providers.NewSafeWebhookClient([]string{"dest_approved_1", "dest_approved_2"}, 2*time.Second)
	ctx := context.Background()

	t.Run("Unapproved Destination ID Rejected", func(t *testing.T) {
		_, _, err := client.ValidateTarget(ctx, "https://93.184.215.14/webhook", "dest_UNAPPROVED_99")
		if err != providers.ErrUnapprovedDestination {
			t.Errorf("expected ErrUnapprovedDestination, got %v", err)
		}
	})

	t.Run("Approved Destination ID Allowed", func(t *testing.T) {
		_, _, err := client.ValidateTarget(ctx, "https://93.184.215.14/webhook", "dest_approved_1")
		if err != nil {
			t.Errorf("unexpected error for approved destination: %v", err)
		}
	})
}

func TestDNSRebindingProtection(t *testing.T) {
	mockResolver := &MockDNSResolver{
		IPs: map[string][]net.IP{
			"rebind.attacker.com":  {net.ParseIP("127.0.0.1")},
			"private.attacker.com": {net.ParseIP("10.0.0.15")},
		},
	}

	client := providers.NewSafeWebhookClient(nil, 2*time.Second)
	client.Resolver = mockResolver

	ctx := context.Background()

	_, _, err := client.ValidateTarget(ctx, "http://rebind.attacker.com/webhook", "")
	if err != providers.ErrRestrictedIPAddress {
		t.Errorf("expected DNS rebinding target to be blocked with ErrRestrictedIPAddress, got %v", err)
	}

	_, _, err = client.ValidateTarget(ctx, "http://private.attacker.com/webhook", "")
	if err != providers.ErrRestrictedIPAddress {
		t.Errorf("expected private DNS target to be blocked, got %v", err)
	}
}

func TestRedirectToPrivateIP(t *testing.T) {
	client := providers.NewSafeWebhookClient(nil, 2*time.Second)
	httpClient := client.BuildHTTPClient()

	redirectReq, err := http.NewRequest("GET", "http://127.0.0.1:9999/secret", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	via := []*http.Request{redirectReq}

	err = httpClient.CheckRedirect(redirectReq, via)
	if err == nil {
		t.Fatalf("expected CheckRedirect to fail for private IP redirect, but succeeded")
	}
	if !strings.Contains(err.Error(), "redirect security error") && !strings.Contains(err.Error(), "loopback") {
		t.Errorf("expected error containing redirect security error, got %v", err)
	}
}

func TestWebhookResponseTimeout(t *testing.T) {
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("skipping test due to network listener restriction: %v", err)
		return
	}
	slowServer := &httptest.Server{
		Listener: l,
		Config: &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(500 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		})},
	}
	slowServer.Start()
	defer slowServer.Close()

	client := providers.NewSafeWebhookClient(nil, 100*time.Millisecond)
	client.AllowLocalForTesting = true

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _, err = client.SendWebhook(ctx, slowServer.URL, "", []byte(`{"test":true}`))
	if err == nil {
		t.Fatalf("expected timeout error, but request succeeded")
	}
}
