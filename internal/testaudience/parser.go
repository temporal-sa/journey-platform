package testaudience

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/validated-pattern/journey-platform/internal/store/object"
)

var (
	// ErrMalformedEncoding indicates invalid UTF-8 or malformed CSV formatting.
	ErrMalformedEncoding = errors.New("malformed encoding or invalid UTF-8")

	// ErrMissingHeader indicates a required CSV column header is missing.
	ErrMissingHeader = errors.New("missing required CSV header")

	// ErrUnexpectedHeader indicates an unrecognized column header when StrictHeaders is enabled.
	ErrUnexpectedHeader = errors.New("unexpected CSV header")

	// ErrDuplicateMemberID indicates that a duplicate member identifier was detected in the input.
	ErrDuplicateMemberID = errors.New("duplicate member ID detected")

	// ErrFormulaInjection indicates potential CSV formula injection (=, @, +, -).
	ErrFormulaInjection = errors.New("CSV formula injection detected")

	// ErrOversizedField indicates a field value exceeded the maximum allowed size.
	ErrOversizedField = errors.New("oversized field value")

	// ErrOversizedRow indicates the row count exceeded the maximum allowed limit.
	ErrOversizedRow = errors.New("max rows limit exceeded")

	// ErrFileSizeExceeded indicates total file size exceeded maximum allowed limit.
	ErrFileSizeExceeded = errors.New("max file size limit exceeded")

	// ErrUnverifiedRecipient indicates an invalid recipient email or phone address.
	ErrUnverifiedRecipient = errors.New("unverified recipient address")

	// ErrEmptyCSV indicates an empty CSV input or header-only file with no data rows.
	ErrEmptyCSV = errors.New("empty CSV input")
)

const (
	DefaultMaxRows      = 10000
	DefaultMaxFileSize  = 10 * 1024 * 1024 // 10 MB
	DefaultMaxFieldSize = 1024             // 1 KB
)

// ParserOptions configures the static-list parser and validator.
type ParserOptions struct {
	MaxRows         int
	MaxFileSize     int64
	MaxFieldSize    int
	RequiredHeaders []string
	AllowedHeaders  []string
	StrictHeaders   bool
	Delimiter       rune
	TTL             time.Duration
	ExpiresAt       *time.Time
}

// DefaultParserOptions returns sensible defaults for static list parsing.
func DefaultParserOptions() ParserOptions {
	return ParserOptions{
		MaxRows:         DefaultMaxRows,
		MaxFileSize:     DefaultMaxFileSize,
		MaxFieldSize:    DefaultMaxFieldSize,
		RequiredHeaders: []string{"member_id", "recipient"},
		Delimiter:       0, // Auto-detect
	}
}

// Parser parses and validates static-list CSV streams.
type Parser struct {
	opts ParserOptions
}

// NewParser creates a new Parser with the provided options.
func NewParser(opts ParserOptions) *Parser {
	if opts.MaxRows <= 0 {
		opts.MaxRows = DefaultMaxRows
	}
	if opts.MaxFileSize <= 0 {
		opts.MaxFileSize = DefaultMaxFileSize
	}
	if opts.MaxFieldSize <= 0 {
		opts.MaxFieldSize = DefaultMaxFieldSize
	}
	if len(opts.RequiredHeaders) == 0 {
		opts.RequiredHeaders = []string{"member_id", "recipient"}
	}
	return &Parser{opts: opts}
}

// ParseReader parses a CSV stream from an io.Reader into an immutable StaticList.
func (p *Parser) ParseReader(r io.Reader, tenantID, listID string) (*StaticList, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, errors.New("tenant_id is required")
	}
	if strings.TrimSpace(listID) == "" {
		return nil, errors.New("list_id is required")
	}

	// 1. Enforce Bounded File Size Reading
	lr := io.LimitReader(r, p.opts.MaxFileSize+1)
	rawBytes, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("error reading input stream: %w", err)
	}
	if int64(len(rawBytes)) > p.opts.MaxFileSize {
		return nil, ErrFileSizeExceeded
	}

	// Strip UTF-8 BOM if present
	rawBytes = bytes.TrimPrefix(rawBytes, []byte("\xef\xbb\xbf"))

	// 2. Validate Encoding (UTF-8 check)
	if !utf8.Valid(rawBytes) {
		return nil, ErrMalformedEncoding
	}

	if len(bytes.TrimSpace(rawBytes)) == 0 {
		return nil, ErrEmptyCSV
	}

	// 3. Auto-detect Delimiter if not explicitly specified
	delimiter := p.opts.Delimiter
	if delimiter == 0 {
		delimiter = detectDelimiter(rawBytes)
	}

	// 4. Initialize CSV Reader
	csvReader := csv.NewReader(bytes.NewReader(rawBytes))
	csvReader.Comma = delimiter
	csvReader.FieldsPerRecord = -1
	csvReader.LazyQuotes = false

	// Read Header Row
	header, err := csvReader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, ErrEmptyCSV
		}
		return nil, fmt.Errorf("%w: %v", ErrMalformedEncoding, err)
	}

	// Validate Header Fields
	normalizedHeaders := make([]string, len(header))
	for i, h := range header {
		if len(h) > p.opts.MaxFieldSize {
			return nil, ErrOversizedField
		}
		if hasFormulaInjection(h) {
			return nil, ErrFormulaInjection
		}
		normalizedHeaders[i] = strings.ToLower(strings.TrimSpace(h))
	}

	// Locate required column indices
	memberIDIdx := -1
	recipientIdx := -1

	for idx, h := range normalizedHeaders {
		switch h {
		case "member_id", "memberid", "id", "user_id", "userid":
			if memberIDIdx == -1 {
				memberIDIdx = idx
			}
		case "recipient", "email", "phone", "contact", "recipient_address":
			if recipientIdx == -1 {
				recipientIdx = idx
			}
		}
	}

	// Check if explicit required headers are present
	for _, req := range p.opts.RequiredHeaders {
		reqNorm := strings.ToLower(strings.TrimSpace(req))
		found := false
		for _, h := range normalizedHeaders {
			if h == reqNorm {
				found = true
				break
			}
		}
		if !found {
			// Check mapped aliases
			if (reqNorm == "member_id" && memberIDIdx != -1) || (reqNorm == "recipient" && recipientIdx != -1) {
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("%w: missing %q", ErrMissingHeader, req)
		}
	}

	if memberIDIdx == -1 || recipientIdx == -1 {
		return nil, ErrMissingHeader
	}

	// Validate strict headers if configured
	if p.opts.StrictHeaders && len(p.opts.AllowedHeaders) > 0 {
		allowedMap := make(map[string]bool)
		for _, a := range p.opts.AllowedHeaders {
			allowedMap[strings.ToLower(strings.TrimSpace(a))] = true
		}
		for _, h := range normalizedHeaders {
			if !allowedMap[h] {
				return nil, fmt.Errorf("%w: header %q not allowed", ErrUnexpectedHeader, h)
			}
		}
	}

	// 5. Read & Validate Data Rows
	seenIDs := make(map[string]bool)
	var members []Member
	rowCount := 0

	for {
		record, err := csvReader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrMalformedEncoding, err)
		}

		rowCount++
		if rowCount > p.opts.MaxRows {
			return nil, ErrOversizedRow
		}

		if memberIDIdx >= len(record) || recipientIdx >= len(record) {
			return nil, fmt.Errorf("%w: row %d missing required fields", ErrMalformedEncoding, rowCount)
		}

		// Field validation
		for _, field := range record {
			if len(field) > p.opts.MaxFieldSize {
				return nil, fmt.Errorf("%w: field size %d exceeds limit %d", ErrOversizedField, len(field), p.opts.MaxFieldSize)
			}
			if hasFormulaInjection(field) {
				return nil, ErrFormulaInjection
			}
		}

		memberID := strings.TrimSpace(record[memberIDIdx])
		recipient := strings.TrimSpace(record[recipientIdx])

		if memberID == "" {
			return nil, fmt.Errorf("%w: empty member ID at row %d", ErrMalformedEncoding, rowCount)
		}

		// Check unverified recipient address
		if !isValidRecipient(recipient) {
			return nil, fmt.Errorf("%w: %q at row %d", ErrUnverifiedRecipient, recipient, rowCount)
		}

		// Check duplicate member ID
		if seenIDs[memberID] {
			return nil, fmt.Errorf("%w: duplicate member ID %q at row %d", ErrDuplicateMemberID, memberID, rowCount)
		}
		seenIDs[memberID] = true

		// Extra attributes
		attributes := make(map[string]string)
		for idx, val := range record {
			if idx != memberIDIdx && idx != recipientIdx && idx < len(normalizedHeaders) {
				key := normalizedHeaders[idx]
				if key != "" && val != "" {
					attributes[key] = val
				}
			}
		}

		memberKey := ResolveMemberKey(tenantID, memberID)
		masked := MaskDisplayValue(recipient)

		members = append(members, Member{
			MemberKey:          memberKey,
			MemberID:           memberID,
			Recipient:          recipient,
			MaskedDisplayValue: masked,
			Attributes:         attributes,
		})
	}

	if len(members) == 0 {
		return nil, ErrEmptyCSV
	}

	// 6. Build & Finalize Immutable Static List
	now := time.Now()
	var expiresAt *time.Time
	if p.opts.ExpiresAt != nil {
		expiresAt = p.opts.ExpiresAt
	} else if p.opts.TTL > 0 {
		exp := now.Add(p.opts.TTL)
		expiresAt = &exp
	}

	list := &StaticList{
		ListID:    listID,
		TenantID:  tenantID,
		Members:   members,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	}

	list.Finalize()
	return list, nil
}

// ParseFromObjectStore fetches and parses a static list directly from an ObjectStore.
// If the object is stored as JSON, it unmarshals the StaticList directly.
// Otherwise, it parses the object content as a CSV stream.
func (p *Parser) ParseFromObjectStore(ctx context.Context, store object.ObjectStore, key, tenantID, listID string) (*StaticList, error) {
	rc, info, err := store.GetObject(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve object %q from store: %w", key, err)
	}
	defer rc.Close()

	if info != nil && (info.ContentType == "application/json" || strings.HasSuffix(key, ".json")) {
		var list StaticList
		if err := json.NewDecoder(rc).Decode(&list); err != nil {
			return nil, fmt.Errorf("failed to decode JSON static list from store: %w", err)
		}
		if tenantID != "" {
			list.TenantID = tenantID
		}
		if listID != "" {
			list.ListID = listID
		}
		list.Finalize()
		return &list, nil
	}

	return p.ParseReader(rc, tenantID, listID)
}

// SaveToObjectStore serializes and stores a finalized StaticList into an ObjectStore.
func SaveToObjectStore(ctx context.Context, store object.ObjectStore, list *StaticList, opts object.PutOptions) (*object.ObjectInfo, error) {
	if list == nil {
		return nil, errors.New("nil static list provided")
	}
	list.Finalize()

	data, err := json.Marshal(list)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal static list: %w", err)
	}

	key := fmt.Sprintf("static-lists/%s/%s/%s.json", list.TenantID, list.ListID, list.ContentHash)
	if opts.TenantID == "" {
		opts.TenantID = list.TenantID
	}
	if opts.Metadata == nil {
		opts.Metadata = make(map[string]string)
	}
	opts.Metadata["content_hash"] = list.ContentHash
	opts.Finalized = true
	opts.ContentType = "application/json"

	info, err := store.PutObject(ctx, key, bytes.NewReader(data), opts)
	if err != nil {
		return nil, fmt.Errorf("failed to put static list object into store: %w", err)
	}
	return info, nil
}

// Detect delimiter based on character frequencies in first line
func detectDelimiter(rawBytes []byte) rune {
	lines := bytes.SplitN(rawBytes, []byte("\n"), 2)
	if len(lines) == 0 {
		return ','
	}
	firstLine := string(lines[0])

	counts := map[rune]int{
		',':  strings.Count(firstLine, ","),
		';':  strings.Count(firstLine, ";"),
		'\t': strings.Count(firstLine, "\t"),
		'|':  strings.Count(firstLine, "|"),
	}

	bestDelim := ','
	maxCount := 0
	for delim, count := range counts {
		if count > maxCount {
			maxCount = count
			bestDelim = delim
		}
	}
	return bestDelim
}

// hasFormulaInjection returns true if a cell starts with =, @, +, - (excluding valid phone numbers)
func hasFormulaInjection(val string) bool {
	trimmed := strings.TrimSpace(val)
	if len(trimmed) == 0 {
		return false
	}
	c := trimmed[0]
	if c == '=' || c == '@' {
		return true
	}
	if c == '+' || c == '-' {
		if c == '+' && isPhoneWithPlus(trimmed) {
			return false
		}
		return true
	}
	return false
}

var phonePlusRegex = regexp.MustCompile(`^\+[1-9]\d{6,14}$`)

func isPhoneWithPlus(val string) bool {
	return phonePlusRegex.MatchString(val)
}

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{6,14}$`)
)

func isValidRecipient(val string) bool {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return false
	}
	if strings.Contains(trimmed, "*") {
		return true // Allow masked recipient addresses in test audience streams
	}
	if strings.Contains(trimmed, "@") {
		_, err := mail.ParseAddress(trimmed)
		if err != nil {
			return false
		}
		return emailRegex.MatchString(trimmed)
	}
	return phoneRegex.MatchString(trimmed)
}
