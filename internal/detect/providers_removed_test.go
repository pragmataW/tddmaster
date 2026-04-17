package detect

import "testing"

// TestProviderDetectionRemoved documents that provider detection has been removed.
// The providers.go file is deleted; any reference to DetectProviders or
// GetAvailableProviderNames will fail to compile — which is the intended guard.
func TestProviderDetectionRemoved(t *testing.T) {
	// Intentional: this test file exists only to anchor the deletion.
	// The enforcement is at compile time — any caller that still references
	// the removed symbols will fail to compile.
	t.Log("providers.go removed; symbols no longer accessible")
}
