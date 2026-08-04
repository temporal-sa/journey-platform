package activities

import "testing"

func TestRegister(t *testing.T) {
	r := New()
	if !r.Register("send_email") {
		t.Error("expected successful registration")
	}
}
