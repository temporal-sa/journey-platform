package testkit

import "testing"

func TestReady(t *testing.T) {
	tk := New()
	if !tk.Ready() {
		t.Error("expected testkit to be ready")
	}
}
