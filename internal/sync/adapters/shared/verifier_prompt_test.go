package shared

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerifierInstructions_ContainsGenericSteps(t *testing.T) {
	out := VerifierInstructions("deno check", "deno test")
	assert.Contains(t, out, "Read the changed files")
	assert.Contains(t, out, "deno check")
	assert.Contains(t, out, "deno test")
	assert.Contains(t, out, "You CANNOT edit files")
}

func TestVerifierInstructionsAllPhases_ContainsAllPhaseBlocks(t *testing.T) {
	out := VerifierInstructionsAllPhases("deno check", "deno test")
	assert.Contains(t, out, "TDD RED Phase")
	assert.Contains(t, out, "TDD GREEN Phase")
	assert.Contains(t, out, "TDD REFACTOR Phase")
	assert.Contains(t, out, "Report Format")
	assert.Contains(t, out, "refactorNotes")
}

func TestVerifierRedPhaseBlock_IsReadOnly(t *testing.T) {
	out := VerifierRedPhaseBlock()
	assert.Contains(t, out, "READ-ONLY")
	assert.Contains(t, out, "DO NOT run tests")
	assert.Contains(t, out, "readOnly")
	assert.NotContains(t, out, "Exit code MUST be non-zero", "RED must not require test execution")
	assert.NotContains(t, out, "Run the target test file", "RED must not run tests")
	assert.NotContains(t, strings.ToLower(out), "refactornotes", "RED must not mention refactorNotes emission")
}

func TestVerifierRedPhaseBlock_DoesNotReferenceTestCommand(t *testing.T) {
	out := VerifierRedPhaseBlock()
	assert.NotContains(t, out, "deno test", "RED phase must not reference test command")
	assert.NotContains(t, out, "go test", "RED phase must not reference test command")
}

func TestVerifierGreenPhaseBlock_SignalsExpectedPass(t *testing.T) {
	out := VerifierGreenPhaseBlock("deno check", "deno test")
	assert.Contains(t, out, "Exit code MUST be zero")
	assert.Contains(t, out, "expected-pass-but-failed")
	assert.Contains(t, out, "deno test", "GREEN must include test command")
	assert.Contains(t, out, "deno check", "GREEN must include type check command")
}

func TestVerifierRefactorPhaseBlock_MentionsNotesContract(t *testing.T) {
	out := VerifierRefactorPhaseBlock("deno check", "deno test")
	assert.Contains(t, out, "behavior-changed")
	assert.Contains(t, out, "refactorNotes")
	assert.Contains(t, out, "{file, suggestion, rationale}")
	assert.Contains(t, out, "empty", "empty notes array must be described as valid")
	assert.Contains(t, out, "deno test", "REFACTOR must include test command")
}
