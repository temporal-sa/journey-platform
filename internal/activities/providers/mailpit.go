package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"time"

	"github.com/validated-pattern/journey-platform/internal/api/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// MailpitClient handles routing local email previews to Mailpit via HTTP or SMTP.
type MailpitClient struct {
	HTTPURL  string
	SMTPAddr string
	client   *http.Client
}

// NewMailpitClient creates a new MailpitClient using environment or default ports.
func NewMailpitClient() *MailpitClient {
	httpURL := os.Getenv("MAILPIT_HTTP_URL")
	if httpURL == "" {
		httpURL = "http://localhost:8025"
	}
	smtpAddr := os.Getenv("MAILPIT_SMTP_ADDR")
	if smtpAddr == "" {
		smtpAddr = "localhost:1025"
	}

	return &MailpitClient{
		HTTPURL:  httpURL,
		SMTPAddr: smtpAddr,
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// MailpitRecipient represents an email address object for Mailpit.
type MailpitRecipient struct {
	Email string `json:"Email"`
	Name  string `json:"Name,omitempty"`
}

// MailpitSendPayload represents the JSON body accepted by Mailpit HTTP API.
type MailpitSendPayload struct {
	From    MailpitRecipient   `json:"From"`
	To      []MailpitRecipient `json:"To"`
	Subject string             `json:"Subject"`
	Text    string             `json:"Text"`
	HTML    string             `json:"HTML,omitempty"`
}

// RoutePreview attempts to dispatch an email preview to Mailpit via HTTP API or SMTP.
func (m *MailpitClient) RoutePreview(ctx context.Context, from, to, subject, body string) error {
	if to == "" {
		to = "preview@journey.local"
	}
	if from == "" {
		from = "noreply@journey.local"
	}
	if subject == "" {
		subject = "Journey Workflow Email Preview"
	}

	// 1. Try HTTP API first
	httpEndpoint := fmt.Sprintf("%s/api/v1/send", m.HTTPURL)
	payload := MailpitSendPayload{
		From:    MailpitRecipient{Email: from, Name: "Journey Platform"},
		To:      []MailpitRecipient{{Email: to, Name: to}},
		Subject: subject,
		Text:    body,
		HTML:    fmt.Sprintf("<p>%s</p>", body),
	}

	payloadBytes, err := json.Marshal(payload)
	if err == nil {
		req, reqErr := http.NewRequestWithContext(ctx, "POST", httpEndpoint, bytes.NewBuffer(payloadBytes))
		if reqErr == nil {
			req.Header.Set("Content-Type", "application/json")
			middleware.InjectHTTPHeaders(ctx, req)
			otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
			resp, err := m.client.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode < 400 {
					return nil
				}
			}
		}
	}

	// 2. Fallback to SMTP
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body)
	err = smtp.SendMail(m.SMTPAddr, nil, from, []string{to}, []byte(msg))
	if err != nil {
		return fmt.Errorf("mailpit routing via HTTP and SMTP failed: %w", err)
	}

	return nil
}
