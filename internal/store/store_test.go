package store

import "testing"

func TestPing(t *testing.T) {
	st := New()
	if err := st.Ping(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
