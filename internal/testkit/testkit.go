package testkit

// TestKit provides helper utilities for tests.
type TestKit struct{}

// New creates a new TestKit instance.
func New() *TestKit {
	return &TestKit{}
}

// Ready returns true when test helpers are ready.
func (t *TestKit) Ready() bool {
	return true
}
