package testaudience

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Member represents an individual contact/member in a static list.
type Member struct {
	MemberKey          string            `json:"member_key"`           // Opaque immutable key: SHA-256(tenant_id + member_identifier)
	MemberID           string            `json:"member_id"`            // Original member identifier
	Recipient          string            `json:"recipient"`            // Validated recipient address (email/phone)
	MaskedDisplayValue string            `json:"masked_display_value"` // Masked display value for privacy
	Attributes         map[string]string `json:"attributes,omitempty"` // Optional row attributes
}

// StaticList represents an immutable static list version with metadata and members.
type StaticList struct {
	mu          sync.RWMutex
	ListID      string            `json:"list_id"`
	TenantID    string            `json:"tenant_id"`
	Name        string            `json:"name,omitempty"`
	Version     string            `json:"version,omitempty"`
	ContentHash string            `json:"content_hash"` // SHA-256 of list rows
	ItemCount   int               `json:"item_count"`
	Members     []Member          `json:"members"`
	MemberMap   map[string]Member `json:"-"` // Fast lookup by MemberKey or MemberID
	CreatedAt   time.Time         `json:"created_at"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
	IsDeleted   bool              `json:"is_deleted"`
	DeletedAt   *time.Time        `json:"deleted_at,omitempty"`
}

// ResolveMemberKey generates the opaque immutable key for a tenant and member ID.
// SHA-256(tenant_id + member_identifier)
func ResolveMemberKey(tenantID, memberID string) string {
	h := sha256.Sum256([]byte(tenantID + memberID))
	return hex.EncodeToString(h[:])
}

// MaskDisplayValue returns a masked string for privacy compliance.
func MaskDisplayValue(val string) string {
	val = strings.TrimSpace(val)
	if val == "" {
		return ""
	}
	// Check if email address
	if strings.Contains(val, "@") {
		parts := strings.SplitN(val, "@", 2)
		local, domain := parts[0], parts[1]
		if len(local) <= 2 {
			return local[:1] + "***@" + domain
		}
		return local[:1] + "***" + local[len(local)-1:] + "@" + domain
	}
	// Check if phone number (starts with + or digits)
	if strings.HasPrefix(val, "+") || isNumeric(val) {
		if len(val) <= 5 {
			return val[:1] + "***"
		}
		prefix := val[:2]
		suffix := val[len(val)-2:]
		return prefix + "***" + suffix
	}
	// Generic string masking
	if len(val) <= 2 {
		return val[:1] + "*"
	}
	return val[:1] + "***" + val[len(val)-1:]
}

func isNumeric(s string) bool {
	for _, r := range s {
		if (r < '0' || r > '9') && r != '+' && r != '-' && r != ' ' {
			return false
		}
	}
	return true
}

// CalculateContentHash computes SHA-256 content hash over canonical list rows.
func (s *StaticList) CalculateContentHash() string {
	if len(s.Members) == 0 {
		h := sha256.Sum256([]byte(""))
		return hex.EncodeToString(h[:])
	}
	// Sort member rows deterministically by MemberKey
	rows := make([]string, len(s.Members))
	for i, m := range s.Members {
		rows[i] = fmt.Sprintf("%s:%s:%s", m.MemberKey, m.MemberID, m.Recipient)
	}
	sort.Strings(rows)
	canonicalData := strings.Join(rows, "\n")
	h := sha256.Sum256([]byte(canonicalData))
	return hex.EncodeToString(h[:])
}

// Finalize sets the content hash and initializes member lookups for fast retrieval.
func (s *StaticList) Finalize() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ContentHash = s.CalculateContentHash()
	s.ItemCount = len(s.Members)
	if s.MemberMap == nil {
		s.MemberMap = make(map[string]Member, len(s.Members)*2)
	}
	for _, m := range s.Members {
		s.MemberMap[m.MemberKey] = m
		s.MemberMap[m.MemberID] = m
	}
}

// IsExpired checks whether the list has passed its expiry time.
func (s *StaticList) IsExpired(now time.Time) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.ExpiresAt == nil {
		return false
	}
	return now.After(*s.ExpiresAt) || now.Equal(*s.ExpiresAt)
}

// Delete idempotently marks the static list as deleted.
// Returns true if state changed from active to deleted, false if already deleted.
func (s *StaticList) Delete(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.IsDeleted {
		return false
	}
	s.IsDeleted = true
	s.DeletedAt = &now
	return true
}

// Contains checks if a member key or ID is present in the static list.
// Returns false if the list is deleted or expired.
func (s *StaticList) Contains(memberKeyOrID string, now time.Time) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.IsDeleted || (s.ExpiresAt != nil && (now.After(*s.ExpiresAt) || now.Equal(*s.ExpiresAt))) {
		return false
	}
	if s.MemberMap == nil {
		return false
	}
	_, found := s.MemberMap[memberKeyOrID]
	return found
}
