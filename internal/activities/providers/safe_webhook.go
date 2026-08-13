package providers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/validated-pattern/journey-platform/internal/api/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

var (
	ErrUnapprovedDestination = errors.New("webhook target is not in approved destination IDs list")
	ErrRestrictedIPAddress   = errors.New("webhook destination resolved to a loopback, private, link-local, or metadata IP range")
	ErrInvalidWebhookScheme  = errors.New("webhook target scheme must be http or https")
	ErrMaxRedirectsExceeded  = errors.New("webhook redirect limit exceeded")
	ErrWebhookTimeout        = errors.New("webhook request timed out")
)

var restrictedCIDRs []*net.IPNet

func init() {
	cidrStrings := []string{
		"0.0.0.0/8",
		"10.0.0.0/8",
		"100.64.0.0/10",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"172.16.0.0/12",
		"192.0.0.0/24",
		"192.0.2.0/24",
		"192.168.0.0/16",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"224.0.0.0/4",
		"240.0.0.0/4",
		"255.255.255.255/32",
		"::/128",
		"::1/128",
		"100::/64",
		"2001:db8::/32",
		"fc00::/7",
		"fe80::/10",
		"ff00::/8",
	}

	for _, s := range cidrStrings {
		_, block, err := net.ParseCIDR(s)
		if err == nil {
			restrictedCIDRs = append(restrictedCIDRs, block)
		}
	}
}

func ParseIPOrBypass(host string) (net.IP, bool) {
	host = strings.TrimSpace(host)
	host = strings.TrimPrefix(host, "[")
	host = strings.TrimSuffix(host, "]")

	if ip := net.ParseIP(host); ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			return ip4, true
		}
		return ip, true
	}

	if val, err := strconv.ParseUint(host, 0, 32); err == nil {
		ip4 := net.IPv4(byte(val>>24), byte(val>>16), byte(val>>8), byte(val))
		return ip4, true
	}

	parts := strings.Split(host, ".")
	if len(parts) == 4 {
		var octets [4]byte
		valid := true
		for i, p := range parts {
			val, err := strconv.ParseUint(p, 0, 8)
			if err != nil {
				valid = false
				break
			}
			octets[i] = byte(val)
		}
		if valid {
			ip4 := net.IPv4(octets[0], octets[1], octets[2], octets[3])
			return ip4, true
		}
	}

	return nil, false
}

func IsRestrictedIP(ip net.IP) bool {
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

type DNSResolver interface {
	LookupIP(ctx context.Context, network, host string) ([]net.IP, error)
}

type DefaultDNSResolver struct{}

func (d *DefaultDNSResolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	return net.DefaultResolver.LookupIP(ctx, network, host)
}

type SafeWebhookClient struct {
	ApprovedDestinationIDs map[string]bool
	Resolver               DNSResolver
	Timeout                time.Duration
	AllowLocalForTesting   bool
	mu                     sync.RWMutex
}

func NewSafeWebhookClient(approvedIDs []string, timeout time.Duration) *SafeWebhookClient {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	approvedMap := make(map[string]bool)
	for _, id := range approvedIDs {
		if id != "" {
			approvedMap[id] = true
		}
	}
	return &SafeWebhookClient{
		ApprovedDestinationIDs: approvedMap,
		Resolver:               &DefaultDNSResolver{},
		Timeout:                timeout,
	}
}

func (c *SafeWebhookClient) SetApprovedDestinations(ids []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ApprovedDestinationIDs = make(map[string]bool)
	for _, id := range ids {
		if id != "" {
			c.ApprovedDestinationIDs[id] = true
		}
	}
}

func (c *SafeWebhookClient) ValidateTarget(ctx context.Context, targetURL string, destinationID string) (*url.URL, net.IP, error) {
	return c.validateTargetInternal(ctx, targetURL, destinationID, false)
}

func (c *SafeWebhookClient) ValidateTargetStrict(ctx context.Context, targetURL string, destinationID string) (*url.URL, net.IP, error) {
	return c.validateTargetInternal(ctx, targetURL, destinationID, true)
}

func (c *SafeWebhookClient) validateTargetInternal(ctx context.Context, targetURL string, destinationID string, strict bool) (*url.URL, net.IP, error) {
	c.mu.RLock()
	hasApprovedList := len(c.ApprovedDestinationIDs) > 0
	isApproved := c.ApprovedDestinationIDs[destinationID]
	c.mu.RUnlock()

	if hasApprovedList && destinationID != "" && !isApproved {
		return nil, nil, ErrUnapprovedDestination
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid webhook URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, nil, ErrInvalidWebhookScheme
	}

	host := parsedURL.Hostname()
	if host == "" {
		return nil, nil, errors.New("empty host in webhook URL")
	}

	allowLocal := c.AllowLocalForTesting && !strict

	// 1. Direct IP check
	if parsedIP, ok := ParseIPOrBypass(host); ok {
		if !allowLocal && IsRestrictedIP(parsedIP) {
			return nil, nil, ErrRestrictedIPAddress
		}
		return parsedURL, parsedIP, nil
	}

	lowerHost := strings.ToLower(host)
	if lowerHost == "localhost" || strings.HasSuffix(lowerHost, ".localhost") || lowerHost == "localhost.localdomain" {
		if !allowLocal {
			return nil, nil, ErrRestrictedIPAddress
		}
	}

	// 2. DNS Resolution
	ips, err := c.Resolver.LookupIP(ctx, "ip", host)
	if err != nil || len(ips) == 0 {
		return nil, nil, fmt.Errorf("failed DNS lookup for host %s: %v", host, err)
	}

	for _, ip := range ips {
		if !allowLocal && IsRestrictedIP(ip) {
			return nil, nil, ErrRestrictedIPAddress
		}
	}

	return parsedURL, ips[0], nil
}

func (c *SafeWebhookClient) BuildHTTPClient() *http.Client {
	dialer := &net.Dialer{
		Timeout:   c.Timeout,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			parsedURL := fmt.Sprintf("http://%s:%s", host, port)
			_, safeIP, err := c.ValidateTarget(ctx, parsedURL, "")
			if err != nil {
				return nil, err
			}

			dialAddr := net.JoinHostPort(safeIP.String(), port)
			return dialer.DialContext(ctx, network, dialAddr)
		},
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: c.Timeout,
	}

	return &http.Client{
		Timeout:   c.Timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return ErrMaxRedirectsExceeded
			}

			_, _, err := c.ValidateTargetStrict(req.Context(), req.URL.String(), "")
			if err != nil {
				return fmt.Errorf("redirect security error: %w", err)
			}

			return nil
		},
	}
}

func (c *SafeWebhookClient) SendWebhook(ctx context.Context, targetURL string, destinationID string, payload []byte) (int, string, error) {
	_, _, err := c.ValidateTarget(ctx, targetURL, destinationID)
	if err != nil {
		return 0, "", err
	}

	client := c.BuildHTTPClient()
	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, strings.NewReader(string(payload)))
	if err != nil {
		return 0, "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "JourneyEngine-SafeWebhook/1.0")
	middleware.InjectHTTPHeaders(ctx, req)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return 0, "", ErrWebhookTimeout
		}
		return 0, "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*64))
	return resp.StatusCode, string(bodyBytes), nil
}
