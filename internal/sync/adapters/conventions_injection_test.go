package adapters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/state"
)

const (
	preambleHeader    = "## Project Conventions"
	activeRulesHeader = "### Active Rules"
)

func writeProjectFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func manifestInject(enabled bool) *state.Manifest {
	v := enabled
	return &state.Manifest{
		TddMode:                  true,
		InjectProjectConventions: &v,
	}
}

// ---------------------------------------------------------------------------
// Claude Code — preamble presence/absence across executor, verifier, test-writer
// ---------------------------------------------------------------------------

func TestClaudeCode_Executor_InjectsPreamble(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeProjectFile(t, dir, "CLAUDE.md", "Project rule A\n<!-- tddmaster:start -->\nTDDMASTER\n<!-- tddmaster:end -->\nProject rule B")

	if err := generateAgentFile(dir, "tddmaster", []string{"rule-X"}, manifestInject(true)); err != nil {
		t.Fatalf("generateAgentFile: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".claude", "agents", "tddmaster-executor.md"))
	content := string(data)

	if !strings.Contains(content, preambleHeader) {
		t.Fatalf("preamble header missing: %s", content)
	}
	if strings.Contains(content, "TDDMASTER") {
		t.Fatalf("tddmaster block leaked into preamble: %s", content)
	}
	if !strings.Contains(content, "Project rule A") || !strings.Contains(content, "Project rule B") {
		t.Fatalf("project content not injected: %s", content)
	}
	if !strings.Contains(content, activeRulesHeader) || !strings.Contains(content, "- rule-X") {
		t.Fatalf("active rules not injected: %s", content)
	}
}

func TestClaudeCode_Executor_RespectsInjectFalse(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeProjectFile(t, dir, "CLAUDE.md", "Project body")

	if err := generateAgentFile(dir, "tddmaster", []string{"rule-X"}, manifestInject(false)); err != nil {
		t.Fatalf("generateAgentFile: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".claude", "agents", "tddmaster-executor.md"))
	content := string(data)

	if strings.Contains(content, preambleHeader) {
		t.Fatalf("preamble should be absent when inject=false: %s", content)
	}
}

func TestClaudeCode_TestWriter_InjectsPreamble(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeProjectFile(t, dir, "CLAUDE.md", "Test-writer conventions body")

	if err := generateClaudeCodeTestWriterFile(dir, []string{"rule-Y"}, manifestInject(true)); err != nil {
		t.Fatalf("generateClaudeCodeTestWriterFile: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".claude", "agents", "test-writer.md"))
	content := string(data)

	if !strings.Contains(content, preambleHeader) {
		t.Fatalf("preamble header missing: %s", content)
	}
	if !strings.Contains(content, "Test-writer conventions body") {
		t.Fatalf("conventions missing: %s", content)
	}
	if !strings.Contains(content, "- rule-Y") {
		t.Fatalf("rule missing: %s", content)
	}
}

func TestClaudeCode_Verifier_InjectsPreamble(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeProjectFile(t, dir, "CLAUDE.md", "Verifier conventions")

	if err := generateVerifierFile(dir, []string{"rule-Z"}, manifestInject(true)); err != nil {
		t.Fatalf("generateVerifierFile: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, ".claude", "agents", "tddmaster-verifier.md"))
	content := string(data)

	if !strings.Contains(content, preambleHeader) {
		t.Fatalf("preamble header missing: %s", content)
	}
	if !strings.Contains(content, "- rule-Z") {
		t.Fatalf("rule missing: %s", content)
	}
}

// ---------------------------------------------------------------------------
// Codex — three agents
// ---------------------------------------------------------------------------

func TestCodex_Executor_InjectsPreamble(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeProjectFile(t, dir, "AGENTS.md", "Codex exec body")

	got := buildCodexExecutorAgentToml(dir, "tddmaster", []string{"rule-A"}, manifestInject(true))

	if !strings.Contains(got, preambleHeader) {
		t.Fatalf("preamble missing: %s", got)
	}
	if !strings.Contains(got, "Codex exec body") {
		t.Fatalf("conventions missing: %s", got)
	}
	if !strings.Contains(got, "- rule-A") {
		t.Fatalf("rule missing: %s", got)
	}
}

func TestCodex_Executor_RespectsInjectFalse(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeProjectFile(t, dir, "AGENTS.md", "whatever")

	got := buildCodexExecutorAgentToml(dir, "tddmaster", []string{"rule-A"}, manifestInject(false))

	if strings.Contains(got, preambleHeader) {
		t.Fatalf("preamble should be absent: %s", got)
	}
}

func TestCodex_TestWriter_InjectsPreamble(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeProjectFile(t, dir, "AGENTS.md", "Codex tw body")

	got := buildCodexTestWriterAgentToml(dir, []string{"rule-B"}, manifestInject(true))

	if !strings.Contains(got, preambleHeader) || !strings.Contains(got, "Codex tw body") || !strings.Contains(got, "- rule-B") {
		t.Fatalf("preamble content missing: %s", got)
	}
}

func TestCodex_Verifier_InjectsPreamble(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeProjectFile(t, dir, "AGENTS.md", "Codex verifier body")

	got := buildCodexVerifierAgentToml(dir, []string{"rule-C"}, manifestInject(true))

	if !strings.Contains(got, preambleHeader) || !strings.Contains(got, "- rule-C") {
		t.Fatalf("preamble content missing: %s", got)
	}
}

// ---------------------------------------------------------------------------
// OpenCode — three agents
// ---------------------------------------------------------------------------

func TestOpenCode_Executor_InjectsPreamble(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeProjectFile(t, dir, "AGENTS.md", "OC exec body")

	got := buildOpenCodeExecutorAgentMd(dir, "tddmaster", []string{"rule-D"}, manifestInject(true))

	if !strings.Contains(got, preambleHeader) || !strings.Contains(got, "OC exec body") || !strings.Contains(got, "- rule-D") {
		t.Fatalf("preamble content missing: %s", got)
	}
}

func TestOpenCode_Executor_RespectsInjectFalse(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeProjectFile(t, dir, "AGENTS.md", "whatever")

	got := buildOpenCodeExecutorAgentMd(dir, "tddmaster", []string{"rule-D"}, manifestInject(false))

	if strings.Contains(got, preambleHeader) {
		t.Fatalf("preamble should be absent: %s", got)
	}
}

func TestOpenCode_TestWriter_InjectsPreamble(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeProjectFile(t, dir, "AGENTS.md", "OC tw body")

	got := buildOpenCodeTestWriterAgentMd(dir, []string{"rule-E"}, manifestInject(true))

	if !strings.Contains(got, preambleHeader) || !strings.Contains(got, "- rule-E") {
		t.Fatalf("preamble content missing: %s", got)
	}
}

func TestOpenCode_Verifier_InjectsPreamble(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeProjectFile(t, dir, "AGENTS.md", "OC verifier body")

	got := buildOpenCodeVerifierAgentMd(dir, []string{"rule-F"}, manifestInject(true))

	if !strings.Contains(got, preambleHeader) || !strings.Contains(got, "- rule-F") {
		t.Fatalf("preamble content missing: %s", got)
	}
}
