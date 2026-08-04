package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/validated-pattern/journey-platform/internal/domain"
)

var (
	ErrInvalidTrackingToken  = errors.New("forged or invalid tracking token signature")
	ErrUnsafeWebhookURL      = errors.New("unsafe webhook destination: loopback, private IP, metadata, or prohibited protocol")
	ErrTemplateInjection     = errors.New("template injection attack detected")
	ErrCSVFormulaInjection   = errors.New("CSV formula injection attack detected")
	ErrSmallCellReconstruct = errors.New("small-cell report reconstruction risk: count below differential privacy threshold")
)

// -----------------------------------------------------------------------------
// 1. Forged Tracking Token Validation
// -----------------------------------------------------------------------------

// ValidateTrackingToken verifies HMAC signature of a tracking token formatted as "payload.signature".
func ValidateTrackingToken(token string, hmacSecret string) error {
	if token == "" || hmacSecret == "" {
		return ErrInvalidTrackingToken
	}

	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return fmt.Errorf("%w: malformed token format", ErrInvalidTrackingToken)
	}

	payload, sigHex := parts[0], parts[1]
	expectedSig := computeHMACSHA256(payload, hmacSecret)

	if !hmac.Equal([]byte(sigHex), []byte(expectedSig)) {
		return ErrInvalidTrackingToken
	}

	return nil
}

// GenerateTrackingToken creates a valid HMAC-signed tracking token.
func GenerateTrackingToken(payload string, hmacSecret string) string {
	sigHex := computeHMACSHA256(payload, hmacSecret)
	return payload + "." + sigHex
}

func computeHMACSHA256(data string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// -----------------------------------------------------------------------------
// 2. Webhook SSRF Safety Validation
// -----------------------------------------------------------------------------

var restrictedCIDRs []*net.IPNet

func init() {
	cidrStrings := []string{
		"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8",
		"169.254.0.0/16", "172.16.0.0/12", "192.0.0.0/24", "192.0.2.0/24",
		"192.168.0.0/16", "198.51.100.0/24", "203.0.113.0/24", "224.0.0.0/4",
		"240.0.0.0/4", "255.255.255.255/32", "::/128", "::1/128",
		"fc00::/7", "fe80::/10", "ff00::/8",
	}
	for _, s := range cidrStrings {
		_, block, err := net.ParseCIDR(s)
		if err == nil {
			restrictedCIDRs = append(restrictedCIDRs, block)
		}
	}
}

// ValidateWebhookDestination rejects unsafe URLs targeting private, loopback, link-local, metadata IPs or non-HTTP/HTTPS schemes.
func ValidateWebhookDestination(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("%w: empty URL", ErrUnsafeWebhookURL)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: invalid URL: %v", ErrUnsafeWebhookURL, err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("%w: unsupported scheme '%s'", ErrUnsafeWebhookURL, scheme)
	}

	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("%w: empty host", ErrUnsafeWebhookURL)
	}

	lowerHost := strings.ToLower(host)
	if lowerHost == "localhost" || strings.HasSuffix(lowerHost, ".localhost") || lowerHost == "localhost.localdomain" {
		return fmt.Errorf("%w: localhost destinations prohibited", ErrUnsafeWebhookURL)
	}

	// Check if direct IP address
	if ip := net.ParseIP(host); ip != nil {
		if isPrivateOrRestrictedIP(ip) {
			return fmt.Errorf("%w: IP %s is in restricted private/loopback/metadata range", ErrUnsafeWebhookURL, host)
		}
		return nil
	}

	// Check dotted quad or integer IP representation
	if val, err := strconv.ParseUint(host, 0, 32); err == nil {
		ip := net.IPv4(byte(val>>24), byte(val>>16), byte(val>>8), byte(val))
		if isPrivateOrRestrictedIP(ip) {
			return fmt.Errorf("%w: resolved integer IP is restricted", ErrUnsafeWebhookURL)
		}
	}

	return nil
}

func isPrivateOrRestrictedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() || ip.IsPrivate() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}
	for _, cidr := range restrictedCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// -----------------------------------------------------------------------------
// 3. Template Injection Sanity & Validation
// -----------------------------------------------------------------------------

var prohibitedTemplateTokens = []string{
	"<script", "javascript:", "vbscript:", "exec(", "eval(",
	"system(", "popen(", "import os", "process.exit", "cmd.exe",
}

// ValidateTemplateInput scans templates for malicious injection payloads.
func ValidateTemplateInput(templateStr string) error {
	lower := strings.ToLower(templateStr)
	for _, tok := range prohibitedTemplateTokens {
		if strings.Contains(lower, tok) {
			return fmt.Errorf("%w: template contains prohibited token '%s'", ErrTemplateInjection, tok)
		}
	}
	// Check for dangerous unescaped ${...} expression injection
	if regexp.MustCompile(`\$\{\s*exec|\$\{\s*eval|\$\{\s*system`).MatchString(lower) {
		return fmt.Errorf("%w: dangerous code execution expression in template", ErrTemplateInjection)
	}
	return nil
}

// -----------------------------------------------------------------------------
// 4. CSV Formula Injection Defense
// -----------------------------------------------------------------------------

var csvFormulaTrigger = regexp.MustCompile(`^[\s=+\-@\t\r]`)

// SanitizeCSVCell escapes cells starting with formula triggers (=, +, -, @, tab, CR) to prevent CSV injection.
func SanitizeCSVCell(cell string) string {
	if cell == "" {
		return cell
	}
	// If cell starts with formula trigger characters, prepend single quote `'`
	if csvFormulaTrigger.MatchString(cell) {
		return "'" + cell
	}
	return cell
}

// SanitizeCSVRecord sanitizes an entire row of CSV fields.
func SanitizeCSVRecord(record []string) []string {
	out := make([]string, len(record))
	for i, cell := range record {
		out[i] = SanitizeCSVCell(cell)
	}
	return out
}

// -----------------------------------------------------------------------------
// 5. Differential Privacy & Small-Cell Reconstruction Guard
// -----------------------------------------------------------------------------

const DefaultMinCellSize = 5 // Minimum sample count per cell/metric to prevent reconstruction attacks

// ValidateReportCellSize checks that a sample count meets minimum differential privacy threshold.
func ValidateReportCellSize(totalCount int64, minCellSize int64) error {
	if minCellSize <= 0 {
		minCellSize = DefaultMinCellSize
	}
	if totalCount > 0 && totalCount < minCellSize {
		return fmt.Errorf("%w: sample size %d is below minimum threshold %d", ErrSmallCellReconstruct, totalCount, minCellSize)
	}
	return nil
}

// EnforceReportDifferentialPrivacy redacts or suppresses small-cell metrics in an AggregateReport.
func EnforceReportDifferentialPrivacy(report *domain.AggregateReport, minCellSize int64) (*domain.AggregateReport, error) {
	if report == nil {
		return nil, nil
	}
	if minCellSize <= 0 {
		minCellSize = DefaultMinCellSize
	}

	cloned := *report
	var safeMetrics []domain.MetricSummary

	for _, m := range report.Metrics {
		if m.TotalCount > 0 && m.TotalCount < minCellSize {
			return nil, fmt.Errorf("%w: metric '%s' with count %d violates differential privacy threshold %d",
				ErrSmallCellReconstruct, m.MetricName, m.TotalCount, minCellSize)
		}
		safeMetrics = append(safeMetrics, m)
	}

	cloned.Metrics = safeMetrics
	return &cloned, nil
}
