package ingress

import "testing"

func TestIngest(t *testing.T) {
	ing := New()
	if err := ing.Ingest([]byte("data")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
