package compiler

import "testing"

func TestCompile(t *testing.T) {
	c := New()
	res, err := c.Compile("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != "test" {
		t.Errorf("expected 'test', got '%s'", res)
	}
}
