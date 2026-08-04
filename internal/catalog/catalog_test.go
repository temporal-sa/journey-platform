package catalog

import "testing"

func TestCatalogVersion(t *testing.T) {
	svc := New()
	if svc.Version() == "" {
		t.Error("expected non-empty version")
	}
}
