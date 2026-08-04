package providers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/validated-pattern/journey-platform/internal/activities"
)

// Controllable behavior constants.
const (
	BehaviorSuccess                = "success"
	BehaviorRetryableFailure       = "retryable_failure"
	BehaviorPermanentFailure       = "permanent_failure"
	BehaviorTimeoutAfterAcceptance = "timeout_after_acceptance"
	BehaviorCallbackDelay          = "callback_delay"
	BehaviorDuplicateCallback      = "duplicate_callback"
)

// FakeProviderRegistry manages channel fake providers with controllable behaviors and ledger logging.
type FakeProviderRegistry struct {
	Ledger        *RequestLedger
	Mailpit       *MailpitClient
	SafeWebhook   *SafeWebhookClient
	LinkManager   *TrackedLinkManager
	DefaultBehavior string
}

// NewFakeProviderRegistry initializes the fake provider registry with default dependencies.
func NewFakeProviderRegistry(ledger *RequestLedger) *FakeProviderRegistry {
	if ledger == nil {
		ledger = NewRequestLedger()
	}
	return &FakeProviderRegistry{
		Ledger:          ledger,
		Mailpit:         NewMailpitClient(),
		SafeWebhook:     NewSafeWebhookClient(nil, 5*time.Second),
		LinkManager:     NewTrackedLinkManager(nil),
		DefaultBehavior: BehaviorSuccess,
	}
}

// ExtractBehavior determines the target behavior for a request based on metadata or defaults.
func (r *FakeProviderRegistry) ExtractBehavior(req *activities.ProviderRequest) (string, time.Duration, bool) {
	behavior := r.DefaultBehavior
	var delay time.Duration
	duplicate := false

	if req == nil || req.Metadata == nil {
		return behavior, delay, duplicate
	}

	if b, ok := req.Metadata["behavior"].(string); ok && b != "" {
		behavior = b
	}

	if dStr, ok := req.Metadata["callback_delay"].(string); ok && dStr != "" {
		if parsed, err := time.ParseDuration(dStr); err == nil {
			delay = parsed
		}
	} else if dMs, ok := req.Metadata["callback_delay_ms"].(float64); ok {
		delay = time.Duration(dMs) * time.Millisecond
	} else if dInt, ok := req.Metadata["callback_delay_ms"].(int); ok {
		delay = time.Duration(dInt) * time.Millisecond
	}

	if dup, ok := req.Metadata["duplicate_callback"].(bool); ok {
		duplicate = dup
	} else if dupStr, ok := req.Metadata["duplicate_callback"].(string); ok {
		duplicate = strings.ToLower(dupStr) == "true"
	}

	// Address-based triggers for convenience in testing
	if strings.Contains(req.RecipientAddress, "fail-retry") {
		behavior = BehaviorRetryableFailure
	} else if strings.Contains(req.RecipientAddress, "fail-perm") {
		behavior = BehaviorPermanentFailure
	} else if strings.Contains(req.RecipientAddress, "timeout-accept") {
		behavior = BehaviorTimeoutAfterAcceptance
	}

	return behavior, delay, duplicate
}

// Dispatch satisfies activities.ActionProvider interface.
func (r *FakeProviderRegistry) Dispatch(ctx context.Context, req *activities.ProviderRequest) (*activities.ProviderResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil provider request")
	}

	behavior, delay, duplicate := r.ExtractBehavior(req)

	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	channel := strings.ToLower(req.Channel)
	if channel == "" {
		channel = "email"
	}

	var resp *activities.ProviderResponse
	var mailpitRouted bool
	var webhookURL string
	var err error

	switch behavior {
	case BehaviorRetryableFailure:
		resp = &activities.ProviderResponse{
			Status:       activities.LedgerStatusRetryableFailure,
			ProviderID:   fmt.Sprintf("fake_%s_%d", channel, time.Now().UnixNano()),
			ReasonCode:   "SERVICE_UNAVAILABLE",
			ErrorMessage: "Simulated retryable transient failure",
			Metadata:     map[string]interface{}{"behavior": behavior, "channel": channel},
		}

	case BehaviorPermanentFailure:
		resp = &activities.ProviderResponse{
			Status:       activities.LedgerStatusPermanentFailure,
			ProviderID:   fmt.Sprintf("fake_%s_%d", channel, time.Now().UnixNano()),
			ReasonCode:   "INVALID_RECIPIENT",
			ErrorMessage: "Simulated permanent failure for address",
			Metadata:     map[string]interface{}{"behavior": behavior, "channel": channel},
		}

	case BehaviorTimeoutAfterAcceptance:
		resp = &activities.ProviderResponse{
			Status:       activities.LedgerStatusAccepted,
			ProviderID:   fmt.Sprintf("fake_%s_%d", channel, time.Now().UnixNano()),
			ReasonCode:   "ACCEPTED_PENDING",
			ErrorMessage: "Accepted but async processing timed out",
			Metadata:     map[string]interface{}{"behavior": behavior, "channel": channel, "timeout_after_acceptance": true},
		}

	case BehaviorSuccess, BehaviorCallbackDelay, BehaviorDuplicateCallback, "":
		resp = &activities.ProviderResponse{
			Status:       activities.LedgerStatusAccepted,
			ProviderID:   fmt.Sprintf("fake_%s_%d", channel, time.Now().UnixNano()),
			ReasonCode:   "DELIVERED",
			ErrorMessage: "",
			Metadata:     map[string]interface{}{"behavior": behavior, "channel": channel},
		}

		// Perform channel-specific actions
		if channel == "email" {
			err = r.Mailpit.RoutePreview(ctx, "noreply@journey.local", req.RecipientAddress, "Workflow Notification", req.RenderedContent)
			if err == nil {
				mailpitRouted = true
			}
		} else if channel == "webhook" {
			webhookURL = req.RecipientAddress
			if urlMeta, ok := req.Metadata["webhook_url"].(string); ok && urlMeta != "" {
				webhookURL = urlMeta
			}
			destID, _ := req.Metadata["destination_id"].(string)

			code, body, wErr := r.SafeWebhook.SendWebhook(ctx, webhookURL, destID, []byte(req.RenderedContent))
			if wErr != nil {
				resp.Status = activities.LedgerStatusRetryableFailure
				resp.ReasonCode = "WEBHOOK_FAILED"
				resp.ErrorMessage = wErr.Error()
			} else if code >= 400 {
				resp.Status = activities.LedgerStatusPermanentFailure
				resp.ReasonCode = fmt.Sprintf("HTTP_%d", code)
				resp.ErrorMessage = fmt.Sprintf("Webhook returned status code %d: %s", code, body)
			} else {
				resp.Metadata["http_status"] = code
				resp.Metadata["response_body"] = body
			}
		}

	default:
		resp = &activities.ProviderResponse{
			Status:       activities.LedgerStatusAccepted,
			ProviderID:   fmt.Sprintf("fake_%s_%d", channel, time.Now().UnixNano()),
			ReasonCode:   "DELIVERED",
			Metadata:     map[string]interface{}{"behavior": behavior, "channel": channel},
		}
	}

	// Record in Ledger
	entry := LedgerEntry{
		ID:               resp.ProviderID,
		TenantID:         req.TenantID,
		IdempotencyKey:   req.IdempotencyKey,
		Channel:          channel,
		SubjectRef:       req.SubjectRef,
		RecipientAddress: req.RecipientAddress,
		Status:           resp.Status,
		ReasonCode:       resp.ReasonCode,
		ErrorMessage:     resp.ErrorMessage,
		RenderedContent:  req.RenderedContent,
		Behavior:         behavior,
		MailpitRouted:    mailpitRouted,
		WebhookURL:       webhookURL,
		Metadata:         req.Metadata,
		Timestamp:        time.Now().UTC(),
	}
	r.Ledger.Record(entry)

	// Simulate duplicate callback if requested
	if duplicate || behavior == BehaviorDuplicateCallback {
		dupEntry := entry
		dupEntry.ID = entry.ID + "_dup"
		dupEntry.Metadata = make(map[string]interface{})
		for k, v := range entry.Metadata {
			dupEntry.Metadata[k] = v
		}
		dupEntry.Metadata["is_duplicate"] = true
		r.Ledger.Record(dupEntry)
		resp.Metadata["duplicate_recorded"] = true
	}

	return resp, nil
}

// Reconcile satisfies activities.ActionProvider interface.
func (r *FakeProviderRegistry) Reconcile(ctx context.Context, idempotencyKey string) (*activities.ProviderResponse, error) {
	entries := r.Ledger.AllEntries()
	for _, e := range entries {
		if e.IdempotencyKey == idempotencyKey {
			if e.Behavior == BehaviorTimeoutAfterAcceptance {
				return &activities.ProviderResponse{
					Status:       activities.LedgerStatusRetryableFailure,
					ProviderID:   e.ID,
					ReasonCode:   "TIMEOUT_AFTER_ACCEPTANCE",
					ErrorMessage: "Reconciliation timed out following acceptance",
					Metadata:     e.Metadata,
				}, nil
			}
			return &activities.ProviderResponse{
				Status:       e.Status,
				ProviderID:   e.ID,
				ReasonCode:   e.ReasonCode,
				ErrorMessage: e.ErrorMessage,
				Metadata:     e.Metadata,
			}, nil
		}
	}

	return &activities.ProviderResponse{
		Status:       activities.LedgerStatusUnknown,
		ReasonCode:   "NOT_FOUND",
		ErrorMessage: fmt.Sprintf("no ledger entry found for idempotency key %s", idempotencyKey),
	}, nil
}

// Helper types for individual channel providers implementing activities.ActionProvider
type EmailProvider struct{ Registry *FakeProviderRegistry }
func (p *EmailProvider) Dispatch(ctx context.Context, req *activities.ProviderRequest) (*activities.ProviderResponse, error) {
	req.Channel = "email"
	return p.Registry.Dispatch(ctx, req)
}
func (p *EmailProvider) Reconcile(ctx context.Context, idempotencyKey string) (*activities.ProviderResponse, error) {
	return p.Registry.Reconcile(ctx, idempotencyKey)
}

type SMSProvider struct{ Registry *FakeProviderRegistry }
func (p *SMSProvider) Dispatch(ctx context.Context, req *activities.ProviderRequest) (*activities.ProviderResponse, error) {
	req.Channel = "sms"
	return p.Registry.Dispatch(ctx, req)
}
func (p *SMSProvider) Reconcile(ctx context.Context, idempotencyKey string) (*activities.ProviderResponse, error) {
	return p.Registry.Reconcile(ctx, idempotencyKey)
}

type PushProvider struct{ Registry *FakeProviderRegistry }
func (p *PushProvider) Dispatch(ctx context.Context, req *activities.ProviderRequest) (*activities.ProviderResponse, error) {
	req.Channel = "push"
	return p.Registry.Dispatch(ctx, req)
}
func (p *PushProvider) Reconcile(ctx context.Context, idempotencyKey string) (*activities.ProviderResponse, error) {
	return p.Registry.Reconcile(ctx, idempotencyKey)
}

type InAppProvider struct{ Registry *FakeProviderRegistry }
func (p *InAppProvider) Dispatch(ctx context.Context, req *activities.ProviderRequest) (*activities.ProviderResponse, error) {
	req.Channel = "in_app"
	return p.Registry.Dispatch(ctx, req)
}
func (p *InAppProvider) Reconcile(ctx context.Context, idempotencyKey string) (*activities.ProviderResponse, error) {
	return p.Registry.Reconcile(ctx, idempotencyKey)
}

type WebhookProvider struct{ Registry *FakeProviderRegistry }
func (p *WebhookProvider) Dispatch(ctx context.Context, req *activities.ProviderRequest) (*activities.ProviderResponse, error) {
	req.Channel = "webhook"
	return p.Registry.Dispatch(ctx, req)
}
func (p *WebhookProvider) Reconcile(ctx context.Context, idempotencyKey string) (*activities.ProviderResponse, error) {
	return p.Registry.Reconcile(ctx, idempotencyKey)
}

// ParseIntDuration utility
func ParseIntDuration(ms string) time.Duration {
	val, err := strconv.Atoi(ms)
	if err != nil {
		return 0
	}
	return time.Duration(val) * time.Millisecond
}
