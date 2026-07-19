package promptregistry

import (
	"strings"
	"testing"
)

func TestSpecProposalInstruction_MentionsGivenWhenThen(t *testing.T) {
	text, ok := Instruction(KeySpecTaskGen)
	if !ok {
		t.Fatalf("Instruction(%q): expected ok=true, got false", KeySpecTaskGen)
	}
	upper := strings.ToUpper(text)
	for _, keyword := range []string{"GIVEN", "WHEN", "THEN"} {
		if !strings.Contains(upper, keyword) {
			t.Errorf("spec-proposal instruction does not mention %q; instruction text: %q", keyword, text)
		}
	}
}

func TestSpecProposalInstruction_ReferencesVerificationAnswer(t *testing.T) {
	text, ok := Instruction(KeySpecTaskGen)
	if !ok {
		t.Fatalf("Instruction(%q): expected ok=true, got false", KeySpecTaskGen)
	}
	lower := strings.ToLower(text)
	if !strings.Contains(lower, "verif") {
		t.Errorf("spec-proposal instruction does not reference verification answer; instruction text: %q", text)
	}
}

func TestRefinePromptInstruction_MentionsDependsOn(t *testing.T) {
	text, ok := Instruction(KeyRefinePrompt)
	if !ok {
		t.Fatalf("Instruction(%q): expected ok=true, got false", KeyRefinePrompt)
	}
	if !strings.Contains(text, "dependsOn") {
		t.Errorf("refine prompt does not mention dependsOn; instruction text: %q", text)
	}
	lower := strings.ToLower(text)
	for _, keyword := range []string{"parallel", "cycle", "self-dependencies", "unknown task ids", "removing"} {
		if !strings.Contains(lower, keyword) {
			t.Errorf("refine prompt does not mention %q; instruction text: %q", keyword, text)
		}
	}
}
