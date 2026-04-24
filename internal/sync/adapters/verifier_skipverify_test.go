package adapters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/state"
)

// mandatoryKeywords are the tokens that signal strong/mandatory language in
// verifier prompts when skipVerify=true.
var mandatoryKeywords = []string{"ZORUNLU", "MANDATORY", "mandatory", "required", "REQUIRED"}

func containsMandatory(s string) bool {
	for _, kw := range mandatoryKeywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

func manifestWithTDD(tddMode bool, skipVerify bool) *state.Manifest {
	return &state.Manifest{
		TddMode:    tddMode,
		SkipVerify: skipVerify,
	}
}

// ---------------------------------------------------------------------------
// TestClaudeCode_GenerateVerifierFile_PassesSkipVerify
//
// generateVerifierFile must accept a skipVerify parameter (or derive it from
// manifest.SkipVerify) and forward it to shared.VerifierInstructionsAllPhases.
// The resulting .claude/agents/tddmaster-verifier.md must contain mandatory
// language when skipVerify=true.
// ---------------------------------------------------------------------------
func TestClaudeCode_GenerateVerifierFile_PassesSkipVerify(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	m := manifestWithTDD(true, true) // TDD on, skipVerify on
	if err := generateVerifierFile(dir, nil, m); err != nil {
		t.Fatalf("generateVerifierFile returned unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".claude", "agents", "tddmaster-verifier.md"))
	if err != nil {
		t.Fatalf("verifier file not created: %v", err)
	}
	content := string(data)

	if !containsMandatory(content) {
		t.Errorf(
			"tddmaster-verifier.md with skipVerify=true must contain mandatory language for refactorNotes;\ngot content:\n%s",
			content,
		)
	}
}

// TestClaudeCode_GenerateVerifierFile_SkipVerifyFalse_NoMandatoryLanguage asserts
// that skipVerify=false does not inject the mandatory-language override, so the
// existing prompt wording is preserved (regression guard).
func TestClaudeCode_GenerateVerifierFile_SkipVerifyFalse_NoMandatoryLanguage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	m := manifestWithTDD(true, false) // TDD on, skipVerify off
	if err := generateVerifierFile(dir, nil, m); err != nil {
		t.Fatalf("generateVerifierFile returned unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".claude", "agents", "tddmaster-verifier.md"))
	if err != nil {
		t.Fatalf("verifier file not created: %v", err)
	}
	content := string(data)

	// With skipVerify=false the mandatory override must NOT be present
	// (it would confuse the verifier when full verification is enabled).
	for _, kw := range []string{"ZORUNLU", "MANDATORY"} {
		if strings.Contains(content, kw) {
			t.Errorf(
				"tddmaster-verifier.md with skipVerify=false must NOT contain uppercase mandatory keyword %q;\ngot content:\n%s",
				kw, content,
			)
		}
	}
}

// ---------------------------------------------------------------------------
// TestCodex_BuildVerifierAgentToml_PassesSkipVerify
//
// buildCodexVerifierAgentToml must forward manifest.SkipVerify to the shared
// helper. The resulting TOML must contain mandatory language when skipVerify=true.
// ---------------------------------------------------------------------------
func TestCodex_BuildVerifierAgentToml_PassesSkipVerify(t *testing.T) {
	t.Parallel()

	m := manifestWithTDD(true, true)
	got := buildCodexVerifierAgentToml(t.TempDir(), nil, m)

	if !containsMandatory(got) {
		t.Errorf(
			"codex verifier TOML with skipVerify=true must contain mandatory language for refactorNotes;\ngot:\n%s",
			got,
		)
	}
}

// TestCodex_BuildVerifierAgentToml_SkipVerifyFalse_Regression asserts that
// buildCodexVerifierAgentToml with skipVerify=false does not inject mandatory language.
func TestCodex_BuildVerifierAgentToml_SkipVerifyFalse_Regression(t *testing.T) {
	t.Parallel()

	m := manifestWithTDD(true, false)
	got := buildCodexVerifierAgentToml(t.TempDir(), nil, m)

	for _, kw := range []string{"ZORUNLU", "MANDATORY"} {
		if strings.Contains(got, kw) {
			t.Errorf(
				"codex verifier TOML with skipVerify=false must NOT contain uppercase mandatory keyword %q;\ngot:\n%s",
				kw, got,
			)
		}
	}
}

// ---------------------------------------------------------------------------
// TestOpenCode_BuildVerifierAgentMd_PassesSkipVerify
//
// buildOpenCodeVerifierAgentMd must forward manifest.SkipVerify to the shared
// helper. The resulting markdown must contain mandatory language when skipVerify=true.
// ---------------------------------------------------------------------------
func TestOpenCode_BuildVerifierAgentMd_PassesSkipVerify(t *testing.T) {
	t.Parallel()

	m := manifestWithTDD(true, true)
	got := buildOpenCodeVerifierAgentMd(t.TempDir(), nil, m)

	if !containsMandatory(got) {
		t.Errorf(
			"opencode verifier MD with skipVerify=true must contain mandatory language for refactorNotes;\ngot:\n%s",
			got,
		)
	}
}

// TestOpenCode_BuildVerifierAgentMd_SkipVerifyFalse_Regression asserts that
// buildOpenCodeVerifierAgentMd with skipVerify=false does not inject mandatory language.
func TestOpenCode_BuildVerifierAgentMd_SkipVerifyFalse_Regression(t *testing.T) {
	t.Parallel()

	m := manifestWithTDD(true, false)
	got := buildOpenCodeVerifierAgentMd(t.TempDir(), nil, m)

	for _, kw := range []string{"ZORUNLU", "MANDATORY"} {
		if strings.Contains(got, kw) {
			t.Errorf(
				"opencode verifier MD with skipVerify=false must NOT contain uppercase mandatory keyword %q;\ngot:\n%s",
				kw, got,
			)
		}
	}
}
