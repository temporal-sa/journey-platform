package security

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	emailRegexp      = regexp.MustCompile(`(?i)[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}`)
	phoneRegexp      = regexp.MustCompile(`(?:\+?\d{1,3}[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}`)
	bearerRegexp     = regexp.MustCompile(`(?i)Bearer\s+[^\s,;\.\r\n\[\]]+`)
	creditCardRegexp = regexp.MustCompile(`\b(?:\d[ -]*?){13,16}\b`)
	ssnRegexp        = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)

	// Sensitive Key/Value Pattern Matcher for inline key-value pairs in logs/errors
	sensitivePairRegexp = regexp.MustCompile(`(?i)\b(email|phone|phone_number|body|message_body|message|password|secret|api_key|apikey|credential|credentials|hmac|hmac_key|tracking_secret|auth_token|authorization|private_key|token|pii|ssn|credit_card)\s*[:=]\s*("[^"]*"|'[^']*'|[^\s,;\.\r\n\[\]]+)`)
)

// Sensitive keys that must be redacted when processing map keys.
var DefaultSensitiveMapKeys = map[string]bool{
	"email":           true,
	"email_address":   true,
	"phone":           true,
	"phone_number":    true,
	"body":            true,
	"message_body":    true,
	"message":         true,
	"password":        true,
	"secret":          true,
	"api_key":         true,
	"apikey":          true,
	"credential":      true,
	"credentials":     true,
	"hmac":            true,
	"hmac_key":        true,
	"tracking_secret": true,
	"auth_token":      true,
	"authorization":   true,
	"private_key":     true,
	"token":           true,
	"ssn":             true,
	"credit_card":     true,
	"payload":         true,
	"pii":             true,
}

// RedactString sanitizes raw text by masking emails, phones, credentials, secrets, message bodies, and PII.
func RedactString(input string) string {
	if input == "" {
		return input
	}

	result := input

	// 1. Redact Email addresses
	result = emailRegexp.ReplaceAllString(result, "[REDACTED_EMAIL]")

	// 2. Redact Phone numbers
	result = phoneRegexp.ReplaceAllString(result, "[REDACTED_PHONE]")

	// 3. Redact Bearer tokens
	result = bearerRegexp.ReplaceAllString(result, "Bearer [REDACTED_TOKEN]")

	// 4. Redact SSN & Credit Card
	result = ssnRegexp.ReplaceAllString(result, "[REDACTED_SSN]")
	result = creditCardRegexp.ReplaceAllString(result, "[REDACTED_CARD]")

	// 5. Redact Key-Value pairs (e.g. password=..., api_key: ..., hmac_key=...)
	result = sensitivePairRegexp.ReplaceAllStringFunc(result, func(match string) string {
		parts := sensitivePairRegexp.FindStringSubmatch(match)
		if len(parts) >= 3 {
			key := parts[1]
			val := strings.TrimSpace(parts[2])
			if strings.EqualFold(val, "Bearer") || strings.HasPrefix(val, "[REDACTED") {
				return match
			}
			return fmt.Sprintf("%s=[REDACTED]", key)
		}
		return match
	})

	return result
}

// RedactMap recursively scrubs sensitive keys and values from a structured map.
func RedactMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}

	out := make(map[string]interface{}, len(m))

	for k, v := range m {
		lowerKey := strings.ToLower(k)
		if DefaultSensitiveMapKeys[lowerKey] {
			out[k] = "[REDACTED]"
			continue
		}

		switch val := v.(type) {
		case string:
			out[k] = RedactString(val)
		case map[string]interface{}:
			out[k] = RedactMap(val)
		case []interface{}:
			out[k] = redactSlice(val)
		default:
			out[k] = v
		}
	}

	return out
}

func redactSlice(s []interface{}) []interface{} {
	out := make([]interface{}, len(s))
	for i, elem := range s {
		switch v := elem.(type) {
		case string:
			out[i] = RedactString(v)
		case map[string]interface{}:
			out[i] = RedactMap(v)
		case []interface{}:
			out[i] = redactSlice(v)
		default:
			out[i] = v
		}
	}
	return out
}

// RedactError sanitizes an error message, stripping any embedded emails, phones, credentials, or secrets.
func RedactError(err error) error {
	if err == nil {
		return nil
	}
	redactedMsg := RedactString(err.Error())
	return errors.New(redactedMsg)
}

// FormatRedactedKeys returns sorted list of redacted keys found in a string (helper utility).
func FormatRedactedKeys(keys []string) []string {
	if len(keys) == 0 {
		return nil
	}
	sort.Strings(keys)
	res := make([]string, 0, len(keys))
	for i, k := range keys {
		if i == 0 || k != keys[i-1] {
			res = append(res, k)
		}
	}
	return res
}
