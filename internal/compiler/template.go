package compiler

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/validated-pattern/journey-platform/internal/compiler/expression"
)

type Channel string

const (
	ChannelEmail   Channel = "email"
	ChannelSMS     Channel = "sms"
	ChannelPush    Channel = "push"
	ChannelInApp   Channel = "in_app"
	ChannelWebhook Channel = "webhook"
)

var (
	ErrDisallowedURL        = errors.New("URL violates allowlist or safe protocol rules")
	ErrOutputLengthExceeded = errors.New("rendered template output exceeds maximum length bound")
	ErrInvalidHeader        = errors.New("header contains prohibited control characters")
	ErrProhibitedToken      = errors.New("template contains prohibited injection token")
)

// DefaultMaxOutputLengths defines default upper bounds on rendered template length per channel.
var DefaultMaxOutputLengths = map[Channel]int{
	ChannelEmail:   65536,
	ChannelSMS:     1600,
	ChannelPush:    4096,
	ChannelInApp:   10000,
	ChannelWebhook: 65536,
}

// DefaultSensitiveKeys defines keywords used to redact sensitive values in previews.
var DefaultSensitiveKeys = []string{
	"password", "secret", "token", "auth", "api_key", "apikey",
	"ssn", "credit_card", "card_number", "cvv", "private_key",
}

// RawTemplate represents raw unparsed template inputs.
type RawTemplate struct {
	Channel Channel           `json:"channel"`
	Subject string            `json:"subject,omitempty"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers,omitempty"`
	URL     string            `json:"url,omitempty"`
}

// TemplateToken represents a parsed template segment.
type TemplateToken struct {
	IsParam bool   `json:"is_param"`
	Value   string `json:"value"` // static text or parameter path
}

// RenderPlan is an immutable compiled execution plan for rendering templates safely.
type RenderPlan struct {
	channel         Channel
	subjectTokens   []TemplateToken
	bodyTokens      []TemplateToken
	urlTokens       []TemplateToken
	headerTokens    map[string][]TemplateToken
	parameters      []string
	maxOutputLength int
	urlAllowlist    []string
}

// Channel returns the target channel for this RenderPlan.
func (rp *RenderPlan) Channel() Channel {
	return rp.channel
}

// Parameters returns a sorted list of unique parameter paths referenced in the template.
func (rp *RenderPlan) Parameters() []string {
	res := make([]string, len(rp.parameters))
	copy(res, rp.parameters)
	return res
}

// RenderedMessage represents the output of evaluating a RenderPlan.
type RenderedMessage struct {
	Channel Channel           `json:"channel"`
	Subject string            `json:"subject,omitempty"`
	Body    string            `json:"body"`
	Headers map[string]string `json:"headers,omitempty"`
	URL     string            `json:"url,omitempty"`
}

// TemplateCompiler compiles RawTemplate structs into immutable RenderPlans.
type TemplateCompiler struct {
	maxOutputLengths map[Channel]int
	urlAllowlist     []string
}

// TemplateOption configures a TemplateCompiler.
type TemplateOption func(*TemplateCompiler)

func WithMaxOutputLength(ch Channel, maxLen int) TemplateOption {
	return func(tc *TemplateCompiler) {
		tc.maxOutputLengths[ch] = maxLen
	}
}

func WithURLAllowlist(allowed []string) TemplateOption {
	return func(tc *TemplateCompiler) {
		tc.urlAllowlist = allowed
	}
}

func NewTemplateCompiler(opts ...TemplateOption) *TemplateCompiler {
	tc := &TemplateCompiler{
		maxOutputLengths: make(map[Channel]int),
	}
	for k, v := range DefaultMaxOutputLengths {
		tc.maxOutputLengths[k] = v
	}
	for _, opt := range opts {
		if opt != nil {
			opt(tc)
		}
	}
	return tc
}

var placeholderRegexp = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_\.]+)\s*\}\}`)

// CompileRaw compiles a RawTemplate into an immutable RenderPlan.
func (tc *TemplateCompiler) CompileRaw(raw RawTemplate) (*RenderPlan, error) {
	channel := raw.Channel
	if channel == "" {
		channel = ChannelEmail
	}

	// Check for prohibited injection tokens in body/subject
	if containsProhibitedTokens(raw.Body) || containsProhibitedTokens(raw.Subject) {
		return nil, fmt.Errorf("%w: prohibited script/injection token detected", ErrProhibitedToken)
	}

	maxLen, ok := tc.maxOutputLengths[channel]
	if !ok || maxLen <= 0 {
		maxLen = DefaultMaxOutputLengths[channel]
	}

	paramMap := make(map[string]bool)

	subjectTokens := parseTemplateTokens(raw.Subject, paramMap)
	bodyTokens := parseTemplateTokens(raw.Body, paramMap)
	urlTokens := parseTemplateTokens(raw.URL, paramMap)

	headerTokens := make(map[string][]TemplateToken)
	for k, v := range raw.Headers {
		// Clean header keys
		cleanKey := sanitizeHeaderString(k)
		headerTokens[cleanKey] = parseTemplateTokens(v, paramMap)
	}

	var params []string
	for p := range paramMap {
		params = append(params, p)
	}
	sort.Strings(params)

	return &RenderPlan{
		channel:         channel,
		subjectTokens:   subjectTokens,
		bodyTokens:      bodyTokens,
		urlTokens:       urlTokens,
		headerTokens:    headerTokens,
		parameters:      params,
		maxOutputLength: maxLen,
		urlAllowlist:    tc.urlAllowlist,
	}, nil
}

func parseTemplateTokens(templateStr string, paramMap map[string]bool) []TemplateToken {
	if templateStr == "" {
		return nil
	}

	matches := placeholderRegexp.FindAllStringSubmatchIndex(templateStr, -1)
	if len(matches) == 0 {
		return []TemplateToken{{IsParam: false, Value: templateStr}}
	}

	var tokens []TemplateToken
	lastIdx := 0

	for _, m := range matches {
		fullStart, fullEnd := m[0], m[1]
		paramStart, paramEnd := m[2], m[3]

		if fullStart > lastIdx {
			tokens = append(tokens, TemplateToken{IsParam: false, Value: templateStr[lastIdx:fullStart]})
		}

		paramPath := strings.TrimSpace(templateStr[paramStart:paramEnd])
		paramMap[paramPath] = true
		tokens = append(tokens, TemplateToken{IsParam: true, Value: paramPath})

		lastIdx = fullEnd
	}

	if lastIdx < len(templateStr) {
		tokens = append(tokens, TemplateToken{IsParam: false, Value: templateStr[lastIdx:]})
	}

	return tokens
}

func containsProhibitedTokens(s string) bool {
	lower := strings.ToLower(s)
	if strings.Contains(lower, "<script") || strings.Contains(lower, "javascript:") || strings.Contains(lower, "vbscript:") {
		return true
	}
	return false
}

func sanitizeHeaderString(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return s
}

// Render evaluates the RenderPlan against the given expression.EvalContext and options.
func (rp *RenderPlan) Render(ctx *expression.EvalContext, fakeAttrs map[string]interface{}) (*RenderedMessage, map[string]interface{}, error) {
	if ctx == nil {
		ctx = expression.NewEvalContext()
	}

	resolvedParams := make(map[string]interface{})

	evalTokenList := func(tokens []TemplateToken, escapeForChannel bool) (string, error) {
		var sb strings.Builder
		for _, tok := range tokens {
			if !tok.IsParam {
				sb.WriteString(tok.Value)
				continue
			}

			val, err := resolveParamValue(tok.Value, ctx, fakeAttrs)
			if err != nil {
				return "", err
			}
			resolvedParams[tok.Value] = val

			strVal := formatValue(val)

			if escapeForChannel {
				switch rp.channel {
				case ChannelEmail:
					strVal = html.EscapeString(strVal)
				case ChannelWebhook:
					// If formatting a string inside json, escape JSON special chars
					strVal = jsonEscapeString(strVal)
				case ChannelSMS, ChannelPush, ChannelInApp:
					// Plain text formatting - keep original characters
				}
			}

			sb.WriteString(strVal)
		}
		return sb.String(), nil
	}

	// Render Subject
	renderedSubject, err := evalTokenList(rp.subjectTokens, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to render subject: %w", err)
	}
	renderedSubject = sanitizeHeaderString(renderedSubject)

	// Render Body
	renderedBody, err := evalTokenList(rp.bodyTokens, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to render body: %w", err)
	}

	// Render Headers
	renderedHeaders := make(map[string]string)
	for k, hTokens := range rp.headerTokens {
		hVal, err := evalTokenList(hTokens, false)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to render header %s: %w", k, err)
		}
		renderedHeaders[k] = sanitizeHeaderString(hVal)
	}

	// Render URL
	renderedURL, err := evalTokenList(rp.urlTokens, false)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to render URL: %w", err)
	}
	renderedURL = sanitizeHeaderString(renderedURL)

	// Validate URL against allowlist if URL is non-empty
	if renderedURL != "" {
		if err := validateURL(renderedURL, rp.urlAllowlist); err != nil {
			return nil, nil, err
		}
	}

	// Output length bound check
	totalLen := len(renderedSubject) + len(renderedBody) + len(renderedURL)
	for k, v := range renderedHeaders {
		totalLen += len(k) + len(v)
	}

	if rp.maxOutputLength > 0 && totalLen > rp.maxOutputLength {
		return nil, nil, fmt.Errorf("%w: rendered length %d exceeds max %d for channel %s",
			ErrOutputLengthExceeded, totalLen, rp.maxOutputLength, rp.channel)
	}

	msg := &RenderedMessage{
		Channel: rp.channel,
		Subject: renderedSubject,
		Body:    renderedBody,
		Headers: renderedHeaders,
		URL:     renderedURL,
	}

	return msg, resolvedParams, nil
}

func resolveParamValue(paramPath string, ctx *expression.EvalContext, fakeAttrs map[string]interface{}) (interface{}, error) {
	// First, try expression AST evaluation
	var rawIdent string
	if strings.Contains(paramPath, ".") {
		rawIdent = paramPath
	} else {
		rawIdent = "parameter." + paramPath
	}

	identNode, err := expression.NewIdentifierNode(rawIdent)
	if err == nil {
		val, err := expression.Evaluate(identNode, ctx)
		if err == nil && val.Type != expression.ValMissing {
			return unpackValue(val), nil
		}
	}

	// If missing from expression context, check fakeAttrs
	if fakeAttrs != nil {
		if val, exists := fakeAttrs[paramPath]; exists {
			return val, nil
		}
		if val, exists := fakeAttrs[rawIdent]; exists {
			return val, nil
		}
	}

	// Fallback to empty string for missing template parameters
	return "", nil
}

func unpackValue(val expression.Value) interface{} {
	switch val.Type {
	case expression.ValString:
		return val.Str
	case expression.ValNumber:
		return val.Num
	case expression.ValBool:
		return val.Bool
	case expression.ValArray:
		var res []interface{}
		for _, item := range val.Array {
			res = append(res, unpackValue(item))
		}
		return res
	case expression.ValNull, expression.ValMissing:
		return ""
	default:
		return val.String()
	}
}

func formatValue(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case bool:
		return fmt.Sprintf("%t", v)
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%v", v)
	case int, int64, int32:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func jsonEscapeString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return s
	}
	// json.Marshal adds outer quotes, strip them
	res := string(b)
	if len(res) >= 2 && res[0] == '"' && res[len(res)-1] == '"' {
		return res[1 : len(res)-1]
	}
	return res
}

func validateURL(rawURL string, allowlist []string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil
	}

	lower := strings.ToLower(rawURL)
	if strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "data:") || strings.HasPrefix(lower, "vbscript:") {
		return fmt.Errorf("%w: unsafe URL scheme in %s", ErrDisallowedURL, rawURL)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: invalid URL format: %v", ErrDisallowedURL, err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "" && scheme != "http" && scheme != "https" && scheme != "mailto" && scheme != "tel" {
		return fmt.Errorf("%w: unsupported URL scheme '%s'", ErrDisallowedURL, scheme)
	}

	if len(allowlist) > 0 {
		matched := false
		for _, allowed := range allowlist {
			allowedLower := strings.ToLower(allowed)
			if strings.HasPrefix(lower, allowedLower) {
				matched = true
				break
			}
			if parsed.Host != "" && strings.EqualFold(parsed.Host, allowed) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("%w: URL '%s' is not in allowlist %v", ErrDisallowedURL, rawURL, allowlist)
		}
	}

	return nil
}

// RedactSensitiveText replaces sensitive keys and pattern matches with [REDACTED].
func RedactSensitiveText(text string, sensitiveKeys []string) (string, []string) {
	if text == "" {
		return text, nil
	}

	if len(sensitiveKeys) == 0 {
		sensitiveKeys = DefaultSensitiveKeys
	}

	var redacted []string

	// Credit card regex: 13-16 digits with optional spaces or dashes
	ccReg := regexp.MustCompile(`\b(?:\d[ -]*?){13,16}\b`)
	if ccReg.MatchString(text) {
		text = ccReg.ReplaceAllString(text, "[REDACTED:CREDIT_CARD]")
		redacted = append(redacted, "credit_card")
	}

	// SSN regex: XXX-XX-XXXX
	ssnReg := regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	if ssnReg.MatchString(text) {
		text = ssnReg.ReplaceAllString(text, "[REDACTED:SSN]")
		redacted = append(redacted, "ssn")
	}

	// Bearer Token regex: Bearer <token>
	bearerReg := regexp.MustCompile(`(?i)Bearer\s+[^\s,;\.\r\n\[\]]+`)
	if bearerReg.MatchString(text) {
		text = bearerReg.ReplaceAllString(text, "Bearer [REDACTED]")
		redacted = append(redacted, "auth_token")
	}

	// Key-value sensitive redactions (e.g., password=..., token: ...)
	for _, key := range sensitiveKeys {
		pattern := fmt.Sprintf(`(?i)\b(%s\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;\.\r\n\[\]]+)`, regexp.QuoteMeta(key))
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(text, -1)
		for _, m := range matches {
			if len(m) > 2 {
				val := strings.TrimSpace(m[2])
				if val != "" && !strings.EqualFold(val, "Bearer") && !strings.HasPrefix(val, "[REDACTED") {
					redacted = append(redacted, key)
				}
			}
		}
		text = re.ReplaceAllStringFunc(text, func(match string) string {
			parts := re.FindStringSubmatch(match)
			if len(parts) > 2 {
				val := strings.TrimSpace(parts[2])
				if strings.EqualFold(val, "Bearer") || strings.HasPrefix(val, "[REDACTED") {
					return match
				}
				return parts[1] + "[REDACTED]"
			}
			return match
		})
	}

	// Sort and deduplicate redacted fields
	if len(redacted) > 0 {
		sort.Strings(redacted)
		unique := make([]string, 0, len(redacted))
		for i, r := range redacted {
			if i == 0 || r != redacted[i-1] {
				unique = append(unique, r)
			}
		}
		redacted = unique
	}

	return text, redacted
}
