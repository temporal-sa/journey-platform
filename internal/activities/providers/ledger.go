package providers

import (
	"sync"
	"time"

	"github.com/validated-pattern/journey-platform/internal/activities"
)

// LedgerEntry represents an inspectable request record logged by a fake provider.
type LedgerEntry struct {
	ID               string                 `json:"id"`
	TenantID         string                 `json:"tenant_id"`
	WorkflowID       string                 `json:"workflow_id,omitempty"`
	IdempotencyKey   string                 `json:"idempotency_key,omitempty"`
	Channel          string                 `json:"channel"`
	SubjectRef       string                 `json:"subject_ref,omitempty"`
	RecipientAddress string                 `json:"recipient_address,omitempty"`
	Status           activities.LedgerStatus `json:"status"`
	ReasonCode       string                 `json:"reason_code,omitempty"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
	RenderedContent  string                 `json:"rendered_content,omitempty"`
	Behavior         string                 `json:"behavior,omitempty"`
	MailpitRouted    bool                   `json:"mailpit_routed,omitempty"`
	WebhookURL       string                 `json:"webhook_url,omitempty"`
	DestinationID    string                 `json:"destination_id,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	Timestamp        time.Time              `json:"timestamp"`
}

// RequestLedger stores and provides query capabilities for fake provider executions.
type RequestLedger struct {
	mu      sync.RWMutex
	entries []LedgerEntry
}

// NewRequestLedger creates a new empty RequestLedger.
func NewRequestLedger() *RequestLedger {
	return &RequestLedger{
		entries: make([]LedgerEntry, 0),
	}
}

// Record adds an entry to the request ledger.
func (l *RequestLedger) Record(entry LedgerEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	l.entries = append(l.entries, entry)
}

// GetEntries returns entries, optionally filtered by tenantID and channel.
func (l *RequestLedger) GetEntries(tenantID, channel string) []LedgerEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	res := make([]LedgerEntry, 0, len(l.entries))
	for _, e := range l.entries {
		if (tenantID == "" || e.TenantID == tenantID) && (channel == "" || e.Channel == channel) {
			res = append(res, e)
		}
	}
	return res
}

// AllEntries returns all entries in the ledger.
func (l *RequestLedger) AllEntries() []LedgerEntry {
	return l.GetEntries("", "")
}

// Clear resets all stored entries in the ledger.
func (l *RequestLedger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = make([]LedgerEntry, 0)
}
