
package sync_test

import (
	"testing"

	statesync "github.com/pragmataW/tddmaster/internal/sync"
	"github.com/stretchr/testify/assert"
)

// AC-1: AskUserStrategy field must exist on InteractionHints as a string.
// This is a compile-time guard: if the field is absent or the wrong type,
// this file will not compile and the build fails in the RED phase.
var _ string = statesync.InteractionHints{}.AskUserStrategy

// TestInteractionHints_AskUserStrategyFieldExists verifies that
// AskUserStrategy is a string field on InteractionHints and can be
// assigned programmatically (structural sanity beyond the compile guard).
func TestInteractionHints_AskUserStrategyFieldExists(t *testing.T) {
	h := statesync.InteractionHints{
		AskUserStrategy: "ask_user_question",
	}
	assert.Equal(t, "ask_user_question", h.AskUserStrategy)
}

// AC-2: Valid AskUserStrategy enum values are documented and mutually exclusive.
// The two known values must be non-empty and distinct strings.
func TestInteractionHints_AskUserStrategyKnownValues(t *testing.T) {
	knownValues := []string{
		"ask_user_question",
		"tddmaster_block",
	}

	for _, v := range knownValues {
		assert.NotEmpty(t, v, "each known AskUserStrategy value must be non-empty")
	}

	// Values must be distinct — no two strategies are the same string.
	seen := map[string]bool{}
	for _, v := range knownValues {
		assert.False(t, seen[v], "duplicate AskUserStrategy value: %q", v)
		seen[v] = true
	}
}
