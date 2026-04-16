
package sync_test

import (
	"os"
	"path/filepath"
	"testing"

	statesync "github.com/pragmataW/tddmaster/internal/sync"
	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// parseFrontmatter tests (via LoadScopedRules)
// =============================================================================

func TestLoadScopedRules_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	// No .tddmaster/rules/ directory at all
	rules, err := statesync.LoadScopedRules(dir)
	require.NoError(t, err)
	assert.Empty(t, rules)
}

func TestLoadScopedRules_PlainRule(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".tddmaster", "rules")
	require.NoError(t, os.MkdirAll(rulesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "rule1.md"), []byte("Use Go idioms\n"), 0o644))

	rules, err := statesync.LoadScopedRules(dir)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	assert.Equal(t, "Use Go idioms", rules[0].Text)
	assert.Empty(t, rules[0].Phases)
	assert.Empty(t, rules[0].AppliesTo)
}

func TestLoadScopedRules_RuleWithFrontmatter(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".tddmaster", "rules")
	require.NoError(t, os.MkdirAll(rulesDir, 0o755))

	content := "---\nphases: [EXECUTING, IDLE]\napplies_to: [\"*.go\"]\n---\nWrite tests for all public functions\n"
	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "rule1.md"), []byte(content), 0o644))

	rules, err := statesync.LoadScopedRules(dir)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	assert.Equal(t, "Write tests for all public functions", rules[0].Text)
	assert.Equal(t, []string{"EXECUTING", "IDLE"}, rules[0].Phases)
	assert.Equal(t, []string{"*.go"}, rules[0].AppliesTo)
}

func TestLoadScopedRules_MultiLineRule(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".tddmaster", "rules")
	require.NoError(t, os.MkdirAll(rulesDir, 0o755))
	content := "First line\nSecond line\nThird line\n"
	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "rule1.md"), []byte(content), 0o644))

	rules, err := statesync.LoadScopedRules(dir)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	assert.Equal(t, "First line\nSecond line\nThird line", rules[0].Text)
}

func TestLoadScopedRules_SkipsTxtFiles(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".tddmaster", "rules")
	require.NoError(t, os.MkdirAll(rulesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "rule.md"), []byte("Rule 1\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "note.txt"), []byte("Note rule\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "ignore.json"), []byte(`{"rule": "ignore"}`), 0o644))

	rules, err := statesync.LoadScopedRules(dir)
	require.NoError(t, err)
	// .md and .txt are included, .json is not
	assert.Len(t, rules, 2)
}

func TestLoadRules_PlainStrings(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".tddmaster", "rules")
	require.NoError(t, os.MkdirAll(rulesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "a.md"), []byte("Rule A\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "b.md"), []byte("Rule B\n"), 0o644))

	rules, err := statesync.LoadRules(dir)
	require.NoError(t, err)
	assert.Len(t, rules, 2)
	for _, r := range rules {
		assert.Contains(t, []string{"Rule A", "Rule B"}, r)
	}
}

// =============================================================================
// SplitByTier tests
// =============================================================================

func TestSplitByTier_AllTier1(t *testing.T) {
	rules := []statesync.ScopedRule{
		{Text: "Rule 1"},
		{Text: "Rule 2"},
	}
	tier1, tier2Count := statesync.SplitByTier(rules, state.PhaseExecuting)
	assert.Equal(t, []string{"Rule 1", "Rule 2"}, tier1)
	assert.Equal(t, 0, tier2Count)
}

func TestSplitByTier_FileScopedIsTier2(t *testing.T) {
	rules := []statesync.ScopedRule{
		{Text: "Rule 1"},
		{Text: "Rule 2", AppliesTo: []string{"*.go"}},
	}
	tier1, tier2Count := statesync.SplitByTier(rules, state.PhaseExecuting)
	assert.Equal(t, []string{"Rule 1"}, tier1)
	assert.Equal(t, 1, tier2Count)
}

func TestSplitByTier_PhaseFiltered(t *testing.T) {
	rules := []statesync.ScopedRule{
		{Text: "All phases rule"},
		{Text: "Executing only", Phases: []string{"EXECUTING"}},
		{Text: "Discovery only", Phases: []string{"DISCOVERY"}},
	}
	tier1, tier2Count := statesync.SplitByTier(rules, state.PhaseExecuting)
	assert.Contains(t, tier1, "All phases rule")
	assert.Contains(t, tier1, "Executing only")
	assert.NotContains(t, tier1, "Discovery only")
	assert.Equal(t, 0, tier2Count)
}

// =============================================================================
// FilterRules tests
// =============================================================================

func TestFilterRules_NoFilter(t *testing.T) {
	rules := []statesync.ScopedRule{
		{Text: "Rule 1"},
		{Text: "Rule 2"},
	}
	result := statesync.FilterRules(rules, "EXECUTING", nil)
	assert.Equal(t, []string{"Rule 1", "Rule 2"}, result)
}

func TestFilterRules_PhaseFilter(t *testing.T) {
	rules := []statesync.ScopedRule{
		{Text: "Always"},
		{Text: "Only executing", Phases: []string{"EXECUTING"}},
		{Text: "Only discovery", Phases: []string{"DISCOVERY"}},
	}
	result := statesync.FilterRules(rules, "EXECUTING", nil)
	assert.Contains(t, result, "Always")
	assert.Contains(t, result, "Only executing")
	assert.NotContains(t, result, "Only discovery")
}

// =============================================================================
// GetTier2RulesForFile tests
// =============================================================================

func TestGetTier2RulesForFile(t *testing.T) {
	rules := []statesync.ScopedRule{
		{Text: "Global rule"},
		{Text: "Go rule", AppliesTo: []string{"*.go"}},
		{Text: "TS rule", AppliesTo: []string{"*.ts"}},
	}
	result := statesync.GetTier2RulesForFile(rules, "EXECUTING", "main.go")
	assert.Equal(t, []string{"Go rule"}, result)
}

// =============================================================================
// ResolveInteractionHints tests
// =============================================================================

func TestResolveInteractionHints_Empty(t *testing.T) {
	hints := statesync.ResolveInteractionHints(nil)
	// Returns Claude Code defaults
	assert.NotNil(t, hints)
	assert.True(t, hints.HasAskUserTool)
	assert.Equal(t, "tool", hints.OptionPresentation)
}

func TestResolveInteractionHints_ClaudeCode(t *testing.T) {
	hints := statesync.ResolveInteractionHints([]state.CodingToolId{state.CodingToolClaudeCode})
	assert.NotNil(t, hints)
	assert.True(t, hints.HasAskUserTool)
	assert.Equal(t, "tool", hints.OptionPresentation)
	assert.True(t, hints.HasSubAgentDelegation)
	assert.Equal(t, "task", hints.SubAgentMethod)
}
