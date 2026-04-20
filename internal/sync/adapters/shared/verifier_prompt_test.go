package shared_test

import (
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/sync/adapters/shared"
)

// ---------------------------------------------------------------------------
// VerifierInstructionsAllPhases — skipVerify flag
// ---------------------------------------------------------------------------

// TestVerifierInstructions_SkipVerifyTrue_GreenSegment_ContainsMandatory asserts
// that when skipVerify=true the GREEN phase segment uses mandatory/strong language
// for refactorNotes — because with no test-run gate, refactorNotes is the ONLY
// quality signal the verifier can produce.
func TestVerifierInstructions_SkipVerifyTrue_GreenSegment_ContainsMandatory(t *testing.T) {
	t.Parallel()
	got := shared.VerifierInstructionsAllPhases("go build ./...", "go test ./...", true)

	greenStart := strings.Index(got, "### TDD GREEN Phase")
	if greenStart == -1 {
		t.Fatal("GREEN phase block not found in output")
	}
	refactorStart := strings.Index(got[greenStart:], "### TDD REFACTOR Phase")
	var greenSegment string
	if refactorStart == -1 {
		greenSegment = got[greenStart:]
	} else {
		greenSegment = got[greenStart : greenStart+refactorStart]
	}

	mandatoryKeywords := []string{"ZORUNLU", "MANDATORY", "mandatory", "required", "REQUIRED"}
	found := false
	for _, kw := range mandatoryKeywords {
		if strings.Contains(greenSegment, kw) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf(
			"GREEN segment with skipVerify=true must contain mandatory/ZORUNLU language for refactorNotes;\ngot segment:\n%s",
			greenSegment,
		)
	}
}

// TestVerifierInstructions_SkipVerifyTrue_RedSegment_Soft asserts that when
// skipVerify=true the RED segment is softened: hard mandates (e.g. "DO NOT execute
// any shell command") become optional/conditional since test verification is relaxed.
func TestVerifierInstructions_SkipVerifyTrue_RedSegment_Soft(t *testing.T) {
	t.Parallel()
	got := shared.VerifierInstructionsAllPhases("go build ./...", "go test ./...", true)

	redStart := strings.Index(got, "### TDD RED Phase")
	if redStart == -1 {
		t.Fatal("RED phase block not found in output")
	}
	greenStart := strings.Index(got[redStart:], "### TDD GREEN Phase")
	var redSegment string
	if greenStart == -1 {
		redSegment = got[redStart:]
	} else {
		redSegment = got[redStart : redStart+greenStart]
	}

	// The RED block must be softened. A soft indicator OR removal of the hard mandate is acceptable.
	hardPhrase := "DO NOT run tests. DO NOT invoke type-checkers. DO NOT execute any shell command."
	softIndicators := []string{"if needed", "optional", "skip", "relaxed", "may", "soft"}
	hasSoftIndicator := false
	for _, s := range softIndicators {
		if strings.Contains(strings.ToLower(redSegment), s) {
			hasSoftIndicator = true
			break
		}
	}
	stillHard := strings.Contains(redSegment, hardPhrase)
	if stillHard && !hasSoftIndicator {
		t.Errorf(
			"RED segment with skipVerify=true must be softened (contain 'if needed'/'optional'/etc.) or remove the mandatory DO-NOT phrase;\ngot segment:\n%s",
			redSegment,
		)
	}
}

// TestVerifierInstructions_SkipVerifyTrue_RefactorSegment_Soft asserts that when
// skipVerify=true the REFACTOR segment is softened: the hard "Run the full test
// suite" mandate becomes optional/conditional.
func TestVerifierInstructions_SkipVerifyTrue_RefactorSegment_Soft(t *testing.T) {
	t.Parallel()
	got := shared.VerifierInstructionsAllPhases("go build ./...", "go test ./...", true)

	refactorStart := strings.Index(got, "### TDD REFACTOR Phase")
	if refactorStart == -1 {
		t.Fatal("REFACTOR phase block not found in output")
	}
	refactorSegment := got[refactorStart:]

	hardPhrase := "Run the full test suite"
	softIndicators := []string{"if needed", "optional", "skip", "relaxed", "may", "soft"}
	hasSoftIndicator := false
	for _, s := range softIndicators {
		if strings.Contains(strings.ToLower(refactorSegment), s) {
			hasSoftIndicator = true
			break
		}
	}
	stillHard := strings.Contains(refactorSegment, hardPhrase)
	if stillHard && !hasSoftIndicator {
		t.Errorf(
			"REFACTOR segment with skipVerify=true must be softened or remove hard run-suite mandate;\ngot segment:\n%s",
			refactorSegment,
		)
	}
}

// TestVerifierInstructions_SkipVerifyFalse_Regression asserts that when
// skipVerify=false all three phase blocks are present and the GREEN block still
// references refactorNotes, preserving pre-existing behaviour.
func TestVerifierInstructions_SkipVerifyFalse_Regression(t *testing.T) {
	t.Parallel()
	typeCheck := "go build ./..."
	testCmd := "go test ./..."

	got := shared.VerifierInstructionsAllPhases(typeCheck, testCmd, false)

	for _, header := range []string{
		"### TDD RED Phase",
		"### TDD GREEN Phase",
		"### TDD REFACTOR Phase",
	} {
		if !strings.Contains(got, header) {
			t.Errorf("skipVerify=false output missing expected phase header %q", header)
		}
	}

	if !strings.Contains(got, "refactorNotes") {
		t.Error("skipVerify=false GREEN segment must still reference refactorNotes")
	}

	// Determinism: identical inputs must produce identical outputs.
	got2 := shared.VerifierInstructionsAllPhases(typeCheck, testCmd, false)
	if got != got2 {
		t.Error("VerifierInstructionsAllPhases must be deterministic for the same inputs")
	}
}
