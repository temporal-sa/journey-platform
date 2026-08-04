package testaudience

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/validated-pattern/journey-platform/internal/store/object"
)

func TestEvaluator(t *testing.T) {
	e := New()
	if !e.Matches("user-1") {
		t.Error("expected user-1 to match empty evaluator")
	}
	if e.Matches("") {
		t.Error("expected empty string not to match")
	}

	// Register static list
	parser := NewParser(DefaultParserOptions())
	csvData := "member_id,recipient\nuser-1,user1@example.com\nuser-2,+14155552671\n"
	list, err := parser.ParseReader(strings.NewReader(csvData), "tenant-1", "list-1")
	if err != nil {
		t.Fatalf("failed to parse list: %v", err)
	}

	e.RegisterList(list)

	key1 := ResolveMemberKey("tenant-1", "user-1")
	if !e.Matches(key1) {
		t.Errorf("expected member key %s to match evaluator", key1)
	}
	if !e.MatchesList("list-1", key1, time.Now()) {
		t.Errorf("expected member key %s to match list-1", key1)
	}
	if e.MatchesList("list-1", "unknown-user", time.Now()) {
		t.Error("expected unknown-user not to match list-1")
	}

	// Remove list
	e.RemoveList("list-1")
	if _, found := e.GetList("list-1"); found {
		t.Error("expected list-1 to be removed")
	}
}

func TestBOMsAndEncoding(t *testing.T) {
	parser := NewParser(DefaultParserOptions())

	t.Run("UTF-8 BOM header", func(t *testing.T) {
		bomCSV := "\xef\xbb\xbfmember_id,recipient\nm-1,m1@example.com\n"
		list, err := parser.ParseReader(strings.NewReader(bomCSV), "t-1", "l-1")
		if err != nil {
			t.Fatalf("expected UTF-8 BOM to be stripped, got error: %v", err)
		}
		if list.ItemCount != 1 {
			t.Errorf("expected 1 member, got %d", list.ItemCount)
		}
	})

	t.Run("Invalid UTF-8 encoding", func(t *testing.T) {
		invalidUTF8 := []byte{0x6d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x2c, 0x72, 0x65, 0x63, 0x69, 0x70, 0x69, 0x65, 0x6e, 0x74, 0x0a, 0xff, 0xfe, 0xfd, 0x0a}
		_, err := parser.ParseReader(bytes.NewReader(invalidUTF8), "t-1", "l-1")
		if !errors.Is(err, ErrMalformedEncoding) {
			t.Errorf("expected ErrMalformedEncoding, got %v", err)
		}
	})
}

func TestDelimiters(t *testing.T) {
	parser := NewParser(DefaultParserOptions())

	delimiters := []struct {
		name      string
		csvData   string
		expectCount int
	}{
		{"Comma", "member_id,recipient\nm-1,user1@example.com\nm-2,user2@example.com\n", 2},
		{"Semicolon", "member_id;recipient\nm-1;user1@example.com\nm-2;user2@example.com\n", 2},
		{"Tab", "member_id\trecipient\nm-1\tuser1@example.com\nm-2\tuser2@example.com\n", 2},
		{"Pipe", "member_id|recipient\nm-1|user1@example.com\nm-2|user2@example.com\n", 2},
	}

	for _, tt := range delimiters {
		t.Run(tt.name, func(t *testing.T) {
			list, err := parser.ParseReader(strings.NewReader(tt.csvData), "t-1", "l-delim-"+tt.name)
			if err != nil {
				t.Fatalf("failed to parse %s delimited CSV: %v", tt.name, err)
			}
			if list.ItemCount != tt.expectCount {
				t.Errorf("expected %d items, got %d", tt.expectCount, list.ItemCount)
			}
		})
	}
}

func TestCSVFormulas(t *testing.T) {
	parser := NewParser(DefaultParserOptions())

	formulaCases := []struct {
		name    string
		csvData string
	}{
		{"Equals formula", "member_id,recipient\n=SUM(A1:A10),user@example.com\n"},
		{"At formula", "member_id,recipient\nm-1,@SUM(1+1)\n"},
		{"Minus formula", "member_id,recipient\n-1+1,user@example.com\n"},
		{"Plus formula non-phone", "member_id,recipient\n+cmd|' /C calc'!A0,user@example.com\n"},
		{"Header equals formula", "=member_id,recipient\nm-1,user@example.com\n"},
	}

	for _, tt := range formulaCases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseReader(strings.NewReader(tt.csvData), "t-1", "l-formula")
			if !errors.Is(err, ErrFormulaInjection) {
				t.Errorf("expected ErrFormulaInjection for case %q, got: %v", tt.name, err)
			}
		})
	}

	t.Run("Valid E.164 phone with plus sign", func(t *testing.T) {
		validPlusCSV := "member_id,recipient\nm-1,+14155552671\n"
		list, err := parser.ParseReader(strings.NewReader(validPlusCSV), "t-1", "l-phone")
		if err != nil {
			t.Fatalf("expected valid phone with plus to be accepted, got: %v", err)
		}
		if list.ItemCount != 1 {
			t.Errorf("expected 1 item, got %d", list.ItemCount)
		}
	})
}

func TestOversizedRowsAndFields(t *testing.T) {
	t.Run("Max rows limit", func(t *testing.T) {
		opts := DefaultParserOptions()
		opts.MaxRows = 3
		parser := NewParser(opts)

		csvData := "member_id,recipient\nm-1,u1@example.com\nm-2,u2@example.com\nm-3,u3@example.com\nm-4,u4@example.com\n"
		_, err := parser.ParseReader(strings.NewReader(csvData), "t-1", "l-rows")
		if !errors.Is(err, ErrOversizedRow) {
			t.Errorf("expected ErrOversizedRow, got %v", err)
		}
	})

	t.Run("Max field size limit", func(t *testing.T) {
		opts := DefaultParserOptions()
		opts.MaxFieldSize = 15
		parser := NewParser(opts)

		csvData := "member_id,recipient\nthis_member_id_is_way_too_long,u1@example.com\n"
		_, err := parser.ParseReader(strings.NewReader(csvData), "t-1", "l-field")
		if !errors.Is(err, ErrOversizedField) {
			t.Errorf("expected ErrOversizedField, got %v", err)
		}
	})

	t.Run("Max file size limit", func(t *testing.T) {
		opts := DefaultParserOptions()
		opts.MaxFileSize = 30
		parser := NewParser(opts)

		csvData := "member_id,recipient\nm-1,u1@example.com\nm-2,u2@example.com\n"
		_, err := parser.ParseReader(strings.NewReader(csvData), "t-1", "l-file")
		if !errors.Is(err, ErrFileSizeExceeded) {
			t.Errorf("expected ErrFileSizeExceeded, got %v", err)
		}
	})
}

func TestDuplicateEntries(t *testing.T) {
	parser := NewParser(DefaultParserOptions())

	csvData := "member_id,recipient\nm-1,user1@example.com\nm-1,user2@example.com\n"
	_, err := parser.ParseReader(strings.NewReader(csvData), "t-1", "l-dupe")
	if !errors.Is(err, ErrDuplicateMemberID) {
		t.Errorf("expected ErrDuplicateMemberID, got %v", err)
	}
}

func TestUnverifiedRecipients(t *testing.T) {
	parser := NewParser(DefaultParserOptions())

	invalidCases := []struct {
		name      string
		recipient string
	}{
		{"Invalid email string", "not-an-email"},
		{"Missing domain TLD", "user@domain"},
		{"Invalid phone number", "123"},
	}

	for _, tt := range invalidCases {
		t.Run(tt.name, func(t *testing.T) {
			csvData := "member_id,recipient\nm-1," + tt.recipient + "\n"
			_, err := parser.ParseReader(strings.NewReader(csvData), "t-1", "l-unverified")
			if !errors.Is(err, ErrUnverifiedRecipient) {
				t.Errorf("expected ErrUnverifiedRecipient for %q, got: %v", tt.recipient, err)
			}
		})
	}
}

func TestHeaderValidation(t *testing.T) {
	t.Run("Missing required header", func(t *testing.T) {
		parser := NewParser(DefaultParserOptions())
		csvData := "member_id,other_column\nm-1,val\n"
		_, err := parser.ParseReader(strings.NewReader(csvData), "t-1", "l-hdr-missing")
		if !errors.Is(err, ErrMissingHeader) {
			t.Errorf("expected ErrMissingHeader, got %v", err)
		}
	})

	t.Run("Unexpected header under strict rules", func(t *testing.T) {
		opts := DefaultParserOptions()
		opts.StrictHeaders = true
		opts.AllowedHeaders = []string{"member_id", "recipient"}
		parser := NewParser(opts)

		csvData := "member_id,recipient,unexpected_header\nm-1,u1@example.com,val\n"
		_, err := parser.ParseReader(strings.NewReader(csvData), "t-1", "l-hdr-unexpected")
		if !errors.Is(err, ErrUnexpectedHeader) {
			t.Errorf("expected ErrUnexpectedHeader, got %v", err)
		}
	})
}

func TestExpiredAndDeletedMembers(t *testing.T) {
	parser := NewParser(DefaultParserOptions())
	csvData := "member_id,recipient\nm-1,u1@example.com\n"

	now := time.Now()
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	t.Run("Expired static list", func(t *testing.T) {
		opts := DefaultParserOptions()
		opts.ExpiresAt = &past
		p := NewParser(opts)

		list, err := p.ParseReader(strings.NewReader(csvData), "t-1", "l-expired")
		if err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}

		if !list.IsExpired(now) {
			t.Error("expected static list to be expired")
		}

		key := ResolveMemberKey("t-1", "m-1")
		if list.Contains(key, now) {
			t.Error("expected expired list.Contains to return false")
		}
	})

	t.Run("Active static list not expired", func(t *testing.T) {
		opts := DefaultParserOptions()
		opts.ExpiresAt = &future
		p := NewParser(opts)

		list, err := p.ParseReader(strings.NewReader(csvData), "t-1", "l-active")
		if err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}

		if list.IsExpired(now) {
			t.Error("expected active static list not to be expired")
		}

		key := ResolveMemberKey("t-1", "m-1")
		if !list.Contains(key, now) {
			t.Error("expected active list.Contains to return true")
		}
	})

	t.Run("Idempotent deletion handler", func(t *testing.T) {
		list, err := parser.ParseReader(strings.NewReader(csvData), "t-1", "l-delete")
		if err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}

		key := ResolveMemberKey("t-1", "m-1")
		if !list.Contains(key, now) {
			t.Error("expected active member to be present before deletion")
		}

		// First delete call
		changed := list.Delete(now)
		if !changed {
			t.Error("expected first Delete() call to return true")
		}
		if !list.IsDeleted {
			t.Error("expected list.IsDeleted to be true")
		}
		if list.Contains(key, now) {
			t.Error("expected list.Contains to return false after deletion")
		}

		// Second delete call (idempotent no-op)
		changedAgain := list.Delete(now)
		if changedAgain {
			t.Error("expected second Delete() call to return false (idempotent)")
		}
	})
}

func TestObjectStoreIntegration(t *testing.T) {
	ctx := context.Background()
	store := object.NewMemoryStore()
	parser := NewParser(DefaultParserOptions())

	csvData := "member_id,recipient\nm-100,user100@example.com\nm-101,+14155551234\n"
	list, err := parser.ParseReader(strings.NewReader(csvData), "tenant-alpha", "list-100")
	if err != nil {
		t.Fatalf("failed to parse CSV: %v", err)
	}

	info, err := SaveToObjectStore(ctx, store, list, object.PutOptions{})
	if err != nil {
		t.Fatalf("failed to save list to ObjectStore: %v", err)
	}

	if info.Metadata["content_hash"] != list.ContentHash {
		t.Errorf("expected stored content_hash %s, got %s", list.ContentHash, info.Metadata["content_hash"])
	}

	// Retrieve object and check content
	fetchedList, err := parser.ParseFromObjectStore(ctx, store, info.Key, "tenant-alpha", "list-100")
	if err != nil {
		t.Fatalf("failed to parse list from ObjectStore: %v", err)
	}

	if fetchedList.ContentHash != list.ContentHash {
		t.Errorf("expected content hash %s, got %s", list.ContentHash, fetchedList.ContentHash)
	}
	if fetchedList.ItemCount != 2 {
		t.Errorf("expected item count 2, got %d", fetchedList.ItemCount)
	}
}

func TestMemberKeyResolutionAndMasking(t *testing.T) {
	key1 := ResolveMemberKey("tenant-A", "user-123")
	key2 := ResolveMemberKey("tenant-A", "user-123")
	key3 := ResolveMemberKey("tenant-B", "user-123")

	if key1 != key2 {
		t.Errorf("expected deterministic member keys, got %s and %s", key1, key2)
	}
	if key1 == key3 {
		t.Errorf("expected different keys for different tenants")
	}

	maskedEmail := MaskDisplayValue("john.doe@example.com")
	if !strings.HasPrefix(maskedEmail, "j***e@") {
		t.Errorf("unexpected email masking result: %s", maskedEmail)
	}

	maskedPhone := MaskDisplayValue("+14155552671")
	if !strings.HasPrefix(maskedPhone, "+1***") || !strings.HasSuffix(maskedPhone, "71") {
		t.Errorf("unexpected phone masking result: %s", maskedPhone)
	}
}

// FuzzParseCSV fuzz tests arbitrary byte input to ensure the parser never panics.
func FuzzParseCSV(f *testing.F) {
	seeds := [][]byte{
		[]byte("member_id,recipient\nm-1,u1@example.com\n"),
		[]byte("\xef\xbb\xbfmember_id;recipient\nm-2;+14155552671\n"),
		[]byte("member_id|recipient\n=SUM(1),u3@example.com\n"),
		[]byte("\x00\xff\xfe\xfd\x12\x34"),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		parser := NewParser(DefaultParserOptions())
		// Ensure no panic occurs for arbitrary byte parsing
		_, _ = parser.ParseReader(bytes.NewReader(input), "fuzz-tenant", "fuzz-list")
	})
}
