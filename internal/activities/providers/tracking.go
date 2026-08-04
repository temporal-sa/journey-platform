package providers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidTokenFormat = errors.New("invalid token format")
	ErrInvalidSignature   = errors.New("invalid token signature (token forgery detected)")
	ErrTokenExpired       = errors.New("token has expired")
	ErrWrongTenant        = errors.New("wrong tenant ID")
	ErrTokenReplayed      = errors.New("token replay constraint violated")
)

// TrackedLinkToken holds parameters for signed opaque tracked links.
type TrackedLinkToken struct {
	TenantID         string    `json:"tenant_id"`
	Purpose          string    `json:"purpose"`
	DestinationURL   string    `json:"destination_url"`
	Version          string    `json:"version"`
	ExpiresAt        time.Time `json:"expires_at"`
	ReplayConstraint string    `json:"replay_constraint,omitempty"`
	Nonce            string    `json:"nonce,omitempty"`
}

// TrackedLinkManager generates and verifies signed HMAC SHA-256 tokens.
type TrackedLinkManager struct {
	secret     []byte
	mu         sync.RWMutex
	usedNonces map[string]time.Time
}

// NewTrackedLinkManager initializes a TrackedLinkManager with an HMAC secret key.
func NewTrackedLinkManager(secret []byte) *TrackedLinkManager {
	if len(secret) == 0 {
		secret = []byte("default-tracking-secret-key-journey-platform")
	}
	return &TrackedLinkManager{
		secret:     secret,
		usedNonces: make(map[string]time.Time),
	}
}

// Sign generates a signed opaque token string using HMAC SHA-256.
func (m *TrackedLinkManager) Sign(token TrackedLinkToken) (string, error) {
	if token.Version == "" {
		token.Version = "v1"
	}
	payloadBytes, err := json.Marshal(token)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token: %w", err)
	}

	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(payloadB64))
	sigBytes := mac.Sum(nil)
	sigB64 := base64.RawURLEncoding.EncodeToString(sigBytes)

	return fmt.Sprintf("%s.%s", payloadB64, sigB64), nil
}

// Verify decodes and validates a raw token string against signature, tenant ID, expiry, and replay constraints.
func (m *TrackedLinkManager) Verify(rawToken string, expectedTenantID string) (*TrackedLinkToken, error) {
	parts := strings.Split(rawToken, ".")
	if len(parts) != 2 {
		return nil, ErrInvalidTokenFormat
	}

	payloadB64, sigB64 := parts[0], parts[1]

	// 1. Check signature (Forgery prevention)
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(payloadB64))
	expectedSig := mac.Sum(nil)

	actualSig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil || !hmac.Equal(expectedSig, actualSig) {
		return nil, ErrInvalidSignature
	}

	// 2. Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, ErrInvalidTokenFormat
	}

	var token TrackedLinkToken
	if err := json.Unmarshal(payloadBytes, &token); err != nil {
		return nil, ErrInvalidTokenFormat
	}

	// 3. Verify tenant ID
	if expectedTenantID != "" && token.TenantID != expectedTenantID {
		return nil, ErrWrongTenant
	}

	// 4. Verify expiry
	if !token.ExpiresAt.IsZero() && time.Now().After(token.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	// 5. Verify replay constraints
	if token.ReplayConstraint == "single_use" || token.ReplayConstraint == "once" {
		nonceKey := token.Nonce
		if nonceKey == "" {
			nonceKey = payloadB64
		}

		m.mu.Lock()
		defer m.mu.Unlock()
		if _, used := m.usedNonces[nonceKey]; used {
			return nil, ErrTokenReplayed
		}
		m.usedNonces[nonceKey] = time.Now()
	}

	return &token, nil
}
