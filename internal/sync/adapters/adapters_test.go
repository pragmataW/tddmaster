
package adapters_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/sync/adapters"
	statesync "github.com/pragmataW/tddmaster/internal/sync"
	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Adapter interface tests
// =============================================================================

func allAdapters() []statesync.ToolAdapter {
	return []statesync.ToolAdapter{
		&adapters.ClaudeCodeAdapter{},
		&adapters.OpenCodeAdapter{},
		&adapters.CodexAdapter{},
	}
}

func TestAdapters_ImplementInterface(t *testing.T) {
	for _, a := range allAdapters() {
		assert.NotEmpty(t, string(a.ID()), "adapter %T should have non-empty ID", a)
		caps := a.Capabilities()
		assert.True(t, caps.Rules, "all adapters must support rules")
	}
}

func TestAdapters_UniqueIDs(t *testing.T) {
	seen := map[state.CodingToolId]bool{}
	for _, a := range allAdapters() {
		id := a.ID()
		assert.False(t, seen[id], "duplicate adapter ID: %s", id)
		seen[id] = true
	}
}

func TestAdapters_KnownIDs(t *testing.T) {
	expected := map[state.CodingToolId]bool{
		state.CodingToolClaudeCode: true,
		state.CodingToolOpencode:   true,
		state.CodingToolCodex:      true,
	}
	for _, a := range allAdapters() {
		assert.True(t, expected[a.ID()], "unknown adapter ID: %s", a.ID())
	}
}

// =============================================================================
// ClaudeCodeAdapter tests
// =============================================================================

func TestClaudeCodeAdapter_SyncRules(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         []string{"Use Go idioms"},
		CommandPrefix: "tddmaster",
	}
	a := &adapters.ClaudeCodeAdapter{}
	err := a.SyncRules(ctx, nil)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "tddmaster orchestrator")
	assert.Contains(t, content, "Use Go idioms")
	assert.Contains(t, content, "<!-- tddmaster:start -->")
	assert.Contains(t, content, "<!-- tddmaster:end -->")
}

func TestClaudeCodeAdapter_SyncRules_AllowGit(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         []string{},
		CommandPrefix: "tddmaster",
	}
	a := &adapters.ClaudeCodeAdapter{}

	// Without allowGit, should contain git section
	err := a.SyncRules(ctx, &statesync.SyncOptions{AllowGit: false})
	require.NoError(t, err)
	data, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	assert.Contains(t, string(data), "### Git")

	// With allowGit, should NOT contain git section
	err = a.SyncRules(ctx, &statesync.SyncOptions{AllowGit: true})
	require.NoError(t, err)
	data, _ = os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	assert.NotContains(t, string(data), "### Git")
}

func TestClaudeCodeAdapter_SyncRules_PreservesExistingContent(t *testing.T) {
	dir := t.TempDir()

	// Pre-existing CLAUDE.md with non-tddmaster content
	existingContent := "# My Project\n\nSome existing content.\n\n<!-- tddmaster:start -->\nold content\n<!-- tddmaster:end -->\n\nMore content.\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(existingContent), 0o644))

	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         []string{"New rule"},
		CommandPrefix: "tddmaster",
	}
	a := &adapters.ClaudeCodeAdapter{}
	err := a.SyncRules(ctx, nil)
	require.NoError(t, err)

	data, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	content := string(data)
	assert.Contains(t, content, "# My Project")
	assert.Contains(t, content, "New rule")
	assert.NotContains(t, content, "old content")
}

func TestClaudeCodeAdapter_SyncHooks(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         nil,
		CommandPrefix: "tddmaster",
	}
	a := &adapters.ClaudeCodeAdapter{}
	err := a.SyncHooks(ctx, nil)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	require.NoError(t, err)

	var settings map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &settings))
	assert.Contains(t, settings, "hooks")
}

func TestClaudeCodeAdapter_SyncAgents(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         nil,
		CommandPrefix: "tddmaster",
	}
	a := &adapters.ClaudeCodeAdapter{}
	err := a.SyncAgents(ctx, nil)
	require.NoError(t, err)

	agentDir := filepath.Join(dir, ".claude", "agents")
	_, err = os.Stat(filepath.Join(agentDir, "tddmaster-executor.md"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(agentDir, "tddmaster-verifier.md"))
	assert.NoError(t, err)
}

func TestClaudeCodeAdapter_SyncAgents_DefaultDeno(t *testing.T) {
	// When Manifest is nil, verifier/executor templates must use deno commands.
	dir := t.TempDir()
	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         nil,
		CommandPrefix: "tddmaster",
		Manifest:      nil,
	}
	a := &adapters.ClaudeCodeAdapter{}
	require.NoError(t, a.SyncAgents(ctx, nil))

	agentDir := filepath.Join(dir, ".claude", "agents")

	execData, err := os.ReadFile(filepath.Join(agentDir, "tddmaster-executor.md"))
	require.NoError(t, err)
	assert.NotContains(t, string(execData), "Self-Verification", "executor must not self-verify")
	assert.NotContains(t, string(execData), `"verification"`, "executor report must not have verification field")

	verifData, err := os.ReadFile(filepath.Join(agentDir, "tddmaster-verifier.md"))
	require.NoError(t, err)
	assert.Contains(t, string(verifData), "deno check", "verifier should use deno check when manifest is nil")
	assert.Contains(t, string(verifData), "deno test", "verifier should use deno test when manifest is nil")
}

func TestClaudeCodeAdapter_SyncAgents_CustomTestRunner(t *testing.T) {
	// When Manifest.TestRunner is set, verifier/executor templates must use that command.
	dir := t.TempDir()
	runner := "go test ./..."
	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         nil,
		CommandPrefix: "tddmaster",
		Manifest:      &state.Manifest{TestRunner: &runner},
	}
	a := &adapters.ClaudeCodeAdapter{}
	require.NoError(t, a.SyncAgents(ctx, nil))

	agentDir := filepath.Join(dir, ".claude", "agents")

	execData, err := os.ReadFile(filepath.Join(agentDir, "tddmaster-executor.md"))
	require.NoError(t, err)
	assert.NotContains(t, string(execData), "Self-Verification", "executor must not self-verify")
	assert.NotContains(t, string(execData), `"verification"`, "executor report must not have verification field")

	verifData, err := os.ReadFile(filepath.Join(agentDir, "tddmaster-verifier.md"))
	require.NoError(t, err)
	assert.Contains(t, string(verifData), "go test ./...", "verifier should use custom test runner")
	assert.NotContains(t, string(verifData), "deno check", "verifier must not use deno when test runner is configured")
}

func TestCodexAdapter_SyncAgents_DefaultDeno(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{Root: dir, Rules: nil, CommandPrefix: "tddmaster", Manifest: nil}
	a := &adapters.CodexAdapter{}
	require.NoError(t, a.SyncAgents(ctx, nil))

	agentsDir := filepath.Join(dir, ".codex", "agents")

	execData, err := os.ReadFile(filepath.Join(agentsDir, "tddmaster-executor.toml"))
	require.NoError(t, err)
	assert.NotContains(t, string(execData), "Self-Verification", "executor must not self-verify")
	assert.NotContains(t, string(execData), "verification", "executor report must not have verification field")

	verifData, err := os.ReadFile(filepath.Join(agentsDir, "tddmaster-verifier.toml"))
	require.NoError(t, err)
	assert.Contains(t, string(verifData), "deno check")
	assert.Contains(t, string(verifData), "deno test")
}

func TestCodexAdapter_SyncAgents_CustomTestRunner(t *testing.T) {
	dir := t.TempDir()
	runner := "pytest"
	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         nil,
		CommandPrefix: "tddmaster",
		Manifest:      &state.Manifest{TestRunner: &runner},
	}
	a := &adapters.CodexAdapter{}
	require.NoError(t, a.SyncAgents(ctx, nil))

	agentsDir := filepath.Join(dir, ".codex", "agents")

	execData, err := os.ReadFile(filepath.Join(agentsDir, "tddmaster-executor.toml"))
	require.NoError(t, err)
	assert.NotContains(t, string(execData), "Self-Verification", "executor must not self-verify")

	verifData, err := os.ReadFile(filepath.Join(agentsDir, "tddmaster-verifier.toml"))
	require.NoError(t, err)
	assert.Contains(t, string(verifData), "pytest")
	assert.NotContains(t, string(verifData), "deno")
}

func TestOpenCodeAdapter_SyncAgents_DefaultDeno(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{Root: dir, Rules: nil, CommandPrefix: "tddmaster", Manifest: nil}
	a := &adapters.OpenCodeAdapter{}
	require.NoError(t, a.SyncAgents(ctx, nil))

	agentsDir := filepath.Join(dir, ".opencode", "agents")

	execData, err := os.ReadFile(filepath.Join(agentsDir, "tddmaster-executor.md"))
	require.NoError(t, err)
	assert.NotContains(t, string(execData), "Self-Verification", "executor must not self-verify")
	assert.NotContains(t, string(execData), `"verification"`, "executor report must not have verification field")

	verifData, err := os.ReadFile(filepath.Join(agentsDir, "tddmaster-verifier.md"))
	require.NoError(t, err)
	assert.Contains(t, string(verifData), "deno check")
	assert.Contains(t, string(verifData), "deno test")
}

func TestOpenCodeAdapter_SyncAgents_CustomTestRunner(t *testing.T) {
	dir := t.TempDir()
	runner := "mvn test"
	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         nil,
		CommandPrefix: "tddmaster",
		Manifest:      &state.Manifest{TestRunner: &runner},
	}
	a := &adapters.OpenCodeAdapter{}
	require.NoError(t, a.SyncAgents(ctx, nil))

	agentsDir := filepath.Join(dir, ".opencode", "agents")

	execData, err := os.ReadFile(filepath.Join(agentsDir, "tddmaster-executor.md"))
	require.NoError(t, err)
	assert.NotContains(t, string(execData), "Self-Verification", "executor must not self-verify")

	verifData, err := os.ReadFile(filepath.Join(agentsDir, "tddmaster-verifier.md"))
	require.NoError(t, err)
	assert.Contains(t, string(verifData), "mvn test")
	assert.NotContains(t, string(verifData), "deno")
}

func TestClaudeCodeAdapter_Capabilities(t *testing.T) {
	a := &adapters.ClaudeCodeAdapter{}
	caps := a.Capabilities()
	assert.True(t, caps.Rules)
	assert.True(t, caps.Hooks)
	assert.True(t, caps.Agents)
	assert.False(t, caps.Specs)
	assert.False(t, caps.Mcp)
	assert.True(t, caps.Interaction.HasAskUserTool)
	assert.Equal(t, "tool", caps.Interaction.OptionPresentation)
	assert.Equal(t, "task", caps.Interaction.SubAgentMethod)
}

// =============================================================================
// CodexAdapter tests
// =============================================================================

func TestCodexAdapter_SyncRules(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         []string{"Codex rule"},
		CommandPrefix: "tddmaster",
	}
	a := &adapters.CodexAdapter{}
	err := a.SyncRules(ctx, nil)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "tddmaster")
}

func TestCodexAdapter_SyncHooks(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{Root: dir, Rules: nil, CommandPrefix: "tddmaster"}
	a := &adapters.CodexAdapter{}
	require.NoError(t, a.SyncHooks(ctx, nil))

	data, err := os.ReadFile(filepath.Join(dir, ".codex", "hooks.json"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "tddmaster")
}

func TestCodexAdapter_SyncAgents(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{Root: dir, Rules: nil, CommandPrefix: "tddmaster"}
	a := &adapters.CodexAdapter{}
	require.NoError(t, a.SyncAgents(ctx, nil))

	_, err := os.Stat(filepath.Join(dir, ".codex", "agents", "tddmaster-executor.toml"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, ".codex", "agents", "tddmaster-verifier.toml"))
	assert.NoError(t, err)
}

func TestCodexAdapter_SyncMcp(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{Root: dir, Rules: nil, CommandPrefix: "tddmaster"}
	a := &adapters.CodexAdapter{}
	require.NoError(t, a.SyncMcp(ctx))

	data, err := os.ReadFile(filepath.Join(dir, ".codex", "config.toml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "[mcp_servers.tddmaster]")
}

// =============================================================================
// OpenCodeAdapter tests
// =============================================================================

func TestOpenCodeAdapter_SyncRules(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         []string{"OpenCode rule"},
		CommandPrefix: "tddmaster",
	}
	a := &adapters.OpenCodeAdapter{}
	err := a.SyncRules(ctx, nil)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "tddmaster")
}

func TestOpenCodeAdapter_SyncHooks(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{Root: dir, Rules: nil, CommandPrefix: "tddmaster"}
	a := &adapters.OpenCodeAdapter{}
	require.NoError(t, a.SyncHooks(ctx, nil))

	data, err := os.ReadFile(filepath.Join(dir, ".opencode", "plugins", "tddmaster.ts"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "execSync")
	assert.Contains(t, string(data), "tddmaster invoke-hook")
}

func TestOpenCodeAdapter_SyncAgents(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{Root: dir, Rules: nil, CommandPrefix: "tddmaster"}
	a := &adapters.OpenCodeAdapter{}
	require.NoError(t, a.SyncAgents(ctx, nil))

	_, err := os.Stat(filepath.Join(dir, ".opencode", "agents", "tddmaster-executor.md"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, ".opencode", "agents", "tddmaster-verifier.md"))
	assert.NoError(t, err)
}

func TestOpenCodeAdapter_SyncMcp(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{Root: dir, Rules: nil, CommandPrefix: "tddmaster"}
	a := &adapters.OpenCodeAdapter{}
	require.NoError(t, a.SyncMcp(ctx))

	data, err := os.ReadFile(filepath.Join(dir, "opencode.json"))
	require.NoError(t, err)
	var config map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &config))
	assert.Contains(t, config, "mcp")
}

func TestOpenCodeAdapter_SyncSpecs(t *testing.T) {
	dir := t.TempDir()

	specContent := `# Spec: My Skill

## Discovery Answers

Overview content here.

## Tasks

Tasks here.
`
	specDir := filepath.Join(dir, ".tddmaster", "specs", "my-skill")
	require.NoError(t, os.MkdirAll(specDir, 0o755))
	specPath := filepath.Join(specDir, "spec.md")
	require.NoError(t, os.WriteFile(specPath, []byte(specContent), 0o644))

	ctx := statesync.SyncContext{Root: dir, Rules: nil, CommandPrefix: "tddmaster"}
	a := &adapters.OpenCodeAdapter{}
	require.NoError(t, a.SyncSpecs(ctx, specPath))

	data, err := os.ReadFile(filepath.Join(dir, ".opencode", "skills", "my-skill.md"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "My Skill")
}

// =============================================================================
// Shared: agents_md section replacement
// =============================================================================

func TestAgentsMdSectionReplacement(t *testing.T) {
	dir := t.TempDir()

	// Write initial AGENTS.md with existing content
	initial := "# My Project\n\nSome user content.\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(initial), 0o644))

	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         []string{"Rule 1"},
		CommandPrefix: "tddmaster",
	}
	a := &adapters.CodexAdapter{}
	require.NoError(t, a.SyncRules(ctx, nil))

	data, _ := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	content := string(data)
	assert.Contains(t, content, "# My Project")
	assert.Contains(t, content, "<!-- tddmaster:start -->")
	assert.Contains(t, content, "<!-- tddmaster:end -->")
	assert.Contains(t, content, "Rule 1")

	// Second sync — replace section
	ctx.Rules = []string{"Rule 2"}
	require.NoError(t, a.SyncRules(ctx, nil))

	data, _ = os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	content = string(data)
	assert.Contains(t, content, "# My Project")
	assert.Contains(t, content, "Rule 2")
	assert.NotContains(t, content, "Rule 1")
	assert.Equal(t, 1, strings.Count(content, "<!-- tddmaster:start -->"))
}

// =============================================================================
// Test-writer agent generation tests
// =============================================================================

func TestClaudeCodeAdapter_SyncAgents_TestWriterWhenTddModeTrue(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         nil,
		CommandPrefix: "tddmaster",
		Manifest:      &state.Manifest{TddMode: true},
	}
	a := &adapters.ClaudeCodeAdapter{}
	require.NoError(t, a.SyncAgents(ctx, nil))

	agentDir := filepath.Join(dir, ".claude", "agents")
	data, err := os.ReadFile(filepath.Join(agentDir, "test-writer.md"))
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "name: test-writer")
	assert.Contains(t, content, "TDD")
	assert.Contains(t, content, ".claude/rules/")
	assert.Contains(t, content, "You do NOT write implementation code")
}

func TestClaudeCodeAdapter_SyncAgents_NoTestWriterWhenTddModeFalse(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         nil,
		CommandPrefix: "tddmaster",
		Manifest:      &state.Manifest{TddMode: false},
	}
	a := &adapters.ClaudeCodeAdapter{}
	require.NoError(t, a.SyncAgents(ctx, nil))

	agentDir := filepath.Join(dir, ".claude", "agents")
	_, err := os.Stat(filepath.Join(agentDir, "test-writer.md"))
	assert.True(t, os.IsNotExist(err), "test-writer.md should NOT be created when TddMode is false")
}

func TestClaudeCodeAdapter_SyncAgents_NoTestWriterWhenManifestNil(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         nil,
		CommandPrefix: "tddmaster",
		Manifest:      nil,
	}
	a := &adapters.ClaudeCodeAdapter{}
	require.NoError(t, a.SyncAgents(ctx, nil))

	agentDir := filepath.Join(dir, ".claude", "agents")
	_, err := os.Stat(filepath.Join(agentDir, "test-writer.md"))
	assert.True(t, os.IsNotExist(err), "test-writer.md should NOT be created when Manifest is nil")
}

func TestCodexAdapter_SyncAgents_TestWriterWhenTddModeTrue(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         nil,
		CommandPrefix: "tddmaster",
		Manifest:      &state.Manifest{TddMode: true},
	}
	a := &adapters.CodexAdapter{}
	require.NoError(t, a.SyncAgents(ctx, nil))

	agentsDir := filepath.Join(dir, ".codex", "agents")
	data, err := os.ReadFile(filepath.Join(agentsDir, "test-writer.toml"))
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, `name = "test-writer"`)
	assert.Contains(t, content, "TDD")
	assert.Contains(t, content, "AGENTS.md")
	assert.Contains(t, content, "You do NOT write implementation code")
}

func TestCodexAdapter_SyncAgents_NoTestWriterWhenTddModeFalse(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         nil,
		CommandPrefix: "tddmaster",
		Manifest:      &state.Manifest{TddMode: false},
	}
	a := &adapters.CodexAdapter{}
	require.NoError(t, a.SyncAgents(ctx, nil))

	agentsDir := filepath.Join(dir, ".codex", "agents")
	_, err := os.Stat(filepath.Join(agentsDir, "test-writer.toml"))
	assert.True(t, os.IsNotExist(err), "test-writer.toml should NOT be created when TddMode is false")
}

func TestCodexAdapter_SyncAgents_NoTestWriterWhenManifestNil(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         nil,
		CommandPrefix: "tddmaster",
		Manifest:      nil,
	}
	a := &adapters.CodexAdapter{}
	require.NoError(t, a.SyncAgents(ctx, nil))

	agentsDir := filepath.Join(dir, ".codex", "agents")
	_, err := os.Stat(filepath.Join(agentsDir, "test-writer.toml"))
	assert.True(t, os.IsNotExist(err), "test-writer.toml should NOT be created when Manifest is nil")
}

func TestOpenCodeAdapter_SyncAgents_TestWriterWhenTddModeTrue(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         nil,
		CommandPrefix: "tddmaster",
		Manifest:      &state.Manifest{TddMode: true},
	}
	a := &adapters.OpenCodeAdapter{}
	require.NoError(t, a.SyncAgents(ctx, nil))

	agentsDir := filepath.Join(dir, ".opencode", "agents")
	data, err := os.ReadFile(filepath.Join(agentsDir, "test-writer.md"))
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "name: test-writer")
	assert.Contains(t, content, "TDD")
	assert.Contains(t, content, ".opencode/skills/")
	assert.Contains(t, content, "AGENTS.md")
	assert.Contains(t, content, "You do NOT write implementation code")
}

func TestOpenCodeAdapter_SyncAgents_NoTestWriterWhenTddModeFalse(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         nil,
		CommandPrefix: "tddmaster",
		Manifest:      &state.Manifest{TddMode: false},
	}
	a := &adapters.OpenCodeAdapter{}
	require.NoError(t, a.SyncAgents(ctx, nil))

	agentsDir := filepath.Join(dir, ".opencode", "agents")
	_, err := os.Stat(filepath.Join(agentsDir, "test-writer.md"))
	assert.True(t, os.IsNotExist(err), "test-writer.md should NOT be created when TddMode is false")
}

func TestOpenCodeAdapter_SyncAgents_NoTestWriterWhenManifestNil(t *testing.T) {
	dir := t.TempDir()
	ctx := statesync.SyncContext{
		Root:          dir,
		Rules:         nil,
		CommandPrefix: "tddmaster",
		Manifest:      nil,
	}
	a := &adapters.OpenCodeAdapter{}
	require.NoError(t, a.SyncAgents(ctx, nil))

	agentsDir := filepath.Join(dir, ".opencode", "agents")
	_, err := os.Stat(filepath.Join(agentsDir, "test-writer.md"))
	assert.True(t, os.IsNotExist(err), "test-writer.md should NOT be created when Manifest is nil")
}

// =============================================================================
// AskUserStrategy tests (AC-3 through AC-8)
// These tests intentionally reference the not-yet-existing AskUserStrategy
// field on InteractionHints. The build will fail until the field is added
// (RED phase). Do NOT add the field here — the executor does that.
// =============================================================================

// AC-3: ClaudeCodeAdapter must report AskUserStrategy == "ask_user_question".
func TestClaudeCodeAdapter_Capabilities_AskUserStrategy(t *testing.T) {
	a := &adapters.ClaudeCodeAdapter{}
	caps := a.Capabilities()
	assert.Equal(t, "ask_user_question", caps.Interaction.AskUserStrategy,
		"ClaudeCode has native AskUserQuestion tool so strategy must be ask_user_question")
}

// AC-4: ClaudeCodeAdapter invariant — HasAskUserTool==true implies AskUserStrategy=="ask_user_question".
func TestClaudeCodeAdapter_Capabilities_AskUserStrategyConsistency(t *testing.T) {
	a := &adapters.ClaudeCodeAdapter{}
	caps := a.Capabilities()
	if caps.Interaction.HasAskUserTool {
		assert.Equal(t, "ask_user_question", caps.Interaction.AskUserStrategy,
			"when HasAskUserTool is true, AskUserStrategy must be ask_user_question")
	}
}

// AC-5: CodexAdapter must report AskUserStrategy == "tddmaster_block".
func TestCodexAdapter_Capabilities_AskUserStrategy(t *testing.T) {
	a := &adapters.CodexAdapter{}
	caps := a.Capabilities()
	assert.Equal(t, "tddmaster_block", caps.Interaction.AskUserStrategy,
		"Codex has no native ask-user tool; must use tddmaster block transition")
}

// AC-6: CodexAdapter invariant — HasAskUserTool==false implies AskUserStrategy=="tddmaster_block".
func TestCodexAdapter_Capabilities_AskUserStrategyConsistency(t *testing.T) {
	a := &adapters.CodexAdapter{}
	caps := a.Capabilities()
	if !caps.Interaction.HasAskUserTool {
		assert.Equal(t, "tddmaster_block", caps.Interaction.AskUserStrategy,
			"when HasAskUserTool is false, AskUserStrategy must be tddmaster_block")
	}
}

// AC-7: OpenCodeAdapter must report AskUserStrategy == "tddmaster_block".
func TestOpenCodeAdapter_Capabilities_AskUserStrategy(t *testing.T) {
	a := &adapters.OpenCodeAdapter{}
	caps := a.Capabilities()
	assert.Equal(t, "tddmaster_block", caps.Interaction.AskUserStrategy,
		"OpenCode has no native ask-user tool; must use tddmaster block transition")
}

// AC-8: OpenCodeAdapter invariant — HasAskUserTool==false implies AskUserStrategy=="tddmaster_block".
func TestOpenCodeAdapter_Capabilities_AskUserStrategyConsistency(t *testing.T) {
	a := &adapters.OpenCodeAdapter{}
	caps := a.Capabilities()
	if !caps.Interaction.HasAskUserTool {
		assert.Equal(t, "tddmaster_block", caps.Interaction.AskUserStrategy,
			"when HasAskUserTool is false, AskUserStrategy must be tddmaster_block")
	}
}
