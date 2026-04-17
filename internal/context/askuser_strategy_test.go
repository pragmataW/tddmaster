package context

// RED phase: task-10 — AskUserStrategy field + prompt compiler wiring
//
// These tests are in package `context` (internal) to access unexported helpers.
// They will NOT compile until task-10 GREEN adds:
//   - AskUserStrategy field to InteractionHints
//   - buildBehavioral logic gated on AskUserStrategy
//   - golden files in testdata/golden/
//
// AC-1: compile-time guard — asserts AskUserStrategy exists on InteractionHints
// AC-2: DefaultHints.AskUserStrategy == "ask_user_question"
// AC-3: tddmaster_block strategy injects "tddmaster block" guidance in Rules
// AC-4: ask_user_question strategy does NOT inject "use tddmaster block instead"
// AC-5: tddmaster_block strategy never emits "AskUserQuestion" in any rule (EC-5 core)
// AC-6: golden snapshot — Codex-like hints, EXECUTING phase
// AC-7: golden snapshot — Claude-like hints, EXECUTING phase

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AC-1: compile-time guard — this line will cause a compile error until
// AskUserStrategy is added to InteractionHints.
var _ string = InteractionHints{}.AskUserStrategy

// update flag enables golden file refresh: go test -run=Golden -update
var update = flag.Bool("update", false, "update golden files")

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// makeExecutingState returns a minimal StateFile in EXECUTING phase.
func makeExecutingState() state.StateFile {
	st := state.CreateInitialState()
	specName := "test-spec"
	st.Phase = state.PhaseExecuting
	st.Spec = &specName
	st.Execution.Iteration = 1
	return st
}

// callCompileWithHints calls Compile with a minimal EXECUTING state and the
// given hints. All other optional parameters are nil / zero.
func callCompileWithHints(t *testing.T, hints InteractionHints) NextOutput {
	t.Helper()
	st := makeExecutingState()
	return Compile(st, nil, nil, nil, nil, nil, nil, &hints, nil, 0)
}

// ---------------------------------------------------------------------------
// AC-2: DefaultHints.AskUserStrategy
// ---------------------------------------------------------------------------

func TestInteractionHints_AskUserStrategyDefault(t *testing.T) {
	assert.Equal(t, "ask_user_question", DefaultHints.AskUserStrategy,
		"DefaultHints.AskUserStrategy must be 'ask_user_question' (Claude Code default)")
}

// ---------------------------------------------------------------------------
// AC-3: tddmaster_block strategy injects block guidance into Rules
// ---------------------------------------------------------------------------

func TestBuildBehavioral_TddmasterBlockStrategy_InjectsBlockGuidance(t *testing.T) {
	hints := InteractionHints{
		HasAskUserTool:        false,
		OptionPresentation:    "prose",
		HasSubAgentDelegation: true,
		SubAgentMethod:        "spawn",
		AskUserStrategy:       "tddmaster_block",
	}

	out := callCompileWithHints(t, hints)

	allRules := strings.Join(out.Behavioral.Rules, "\n")
	allRulesLower := strings.ToLower(allRules)

	// At least one rule must mention tddmaster block (case-insensitive)
	assert.Contains(t, allRulesLower, "tddmaster block",
		"When AskUserStrategy='tddmaster_block', at least one rule must reference 'tddmaster block'")

	// That rule must also convey "do not ask inline"
	foundBlockRule := false
	for _, r := range out.Behavioral.Rules {
		rLower := strings.ToLower(r)
		if strings.Contains(rLower, "tddmaster block") {
			// The rule should discourage inline asking
			if strings.Contains(rLower, "not") || strings.Contains(rLower, "never") ||
				strings.Contains(rLower, "instead") || strings.Contains(rLower, "use tddmaster block") {
				foundBlockRule = true
				break
			}
		}
	}
	assert.True(t, foundBlockRule,
		"When AskUserStrategy='tddmaster_block', a rule must instruct to use tddmaster block instead of asking inline")
}

// ---------------------------------------------------------------------------
// AC-4: ask_user_question strategy does NOT inject block-guidance text
// ---------------------------------------------------------------------------

func TestBuildBehavioral_AskUserQuestionStrategy_NoBlockGuidance(t *testing.T) {
	hints := InteractionHints{
		HasAskUserTool:        true,
		OptionPresentation:    "tool",
		HasSubAgentDelegation: true,
		SubAgentMethod:        "task",
		AskUserStrategy:       "ask_user_question",
	}

	out := callCompileWithHints(t, hints)

	for _, r := range out.Behavioral.Rules {
		rLower := strings.ToLower(r)
		assert.NotContains(t, rLower, "use tddmaster block instead",
			"ask_user_question strategy must not inject 'use tddmaster block instead' in rules")
	}
}

// ---------------------------------------------------------------------------
// AC-5: tddmaster_block strategy must never emit "AskUserQuestion" in rules (EC-5)
// ---------------------------------------------------------------------------

func TestBuildBehavioral_TddmasterBlockStrategy_NoAskUserQuestionText(t *testing.T) {
	hints := InteractionHints{
		HasAskUserTool:        false,
		OptionPresentation:    "prose",
		HasSubAgentDelegation: true,
		SubAgentMethod:        "spawn",
		AskUserStrategy:       "tddmaster_block",
	}

	out := callCompileWithHints(t, hints)

	for _, r := range out.Behavioral.Rules {
		assert.NotContains(t, r, "AskUserQuestion",
			"Rules sent to tddmaster_block adapters (Codex/OpenCode) must never mention AskUserQuestion tool — EC-5 core assertion")
	}
}

// ---------------------------------------------------------------------------
// AC-6: golden snapshot — Codex-like hints (tddmaster_block), EXECUTING phase
// ---------------------------------------------------------------------------

func TestBuildBehavioral_GoldenSnapshot_TddmasterBlock(t *testing.T) {
	hints := InteractionHints{
		HasAskUserTool:        false,
		OptionPresentation:    "prose",
		HasSubAgentDelegation: true,
		SubAgentMethod:        "spawn",
		AskUserStrategy:       "tddmaster_block",
	}

	out := callCompileWithHints(t, hints)
	rules := out.Behavioral.Rules

	goldenPath := filepath.Join("testdata", "golden", "behavioral_tddmaster_block.json")

	data, err := json.MarshalIndent(rules, "", "  ")
	require.NoError(t, err, "marshalling rules to JSON must not fail")

	if *update {
		require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0o755))
		require.NoError(t, os.WriteFile(goldenPath, data, 0o644))
		t.Logf("golden file updated: %s", goldenPath)
		return
	}

	golden, err := os.ReadFile(goldenPath)
	require.NoError(t, err,
		"golden file %s must exist; run with -update to create it", goldenPath)

	assert.JSONEq(t, string(golden), string(data),
		"behavioral rules for tddmaster_block strategy must match golden snapshot")
}

// ---------------------------------------------------------------------------
// AC-7: golden snapshot — Claude-like hints (ask_user_question), EXECUTING phase
// ---------------------------------------------------------------------------

func TestBuildBehavioral_GoldenSnapshot_AskUserQuestion(t *testing.T) {
	hints := InteractionHints{
		HasAskUserTool:        true,
		OptionPresentation:    "tool",
		HasSubAgentDelegation: true,
		SubAgentMethod:        "task",
		AskUserStrategy:       "ask_user_question",
	}

	out := callCompileWithHints(t, hints)
	rules := out.Behavioral.Rules

	goldenPath := filepath.Join("testdata", "golden", "behavioral_ask_user_question.json")

	data, err := json.MarshalIndent(rules, "", "  ")
	require.NoError(t, err, "marshalling rules to JSON must not fail")

	if *update {
		require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0o755))
		require.NoError(t, os.WriteFile(goldenPath, data, 0o644))
		t.Logf("golden file updated: %s", goldenPath)
		return
	}

	golden, err := os.ReadFile(goldenPath)
	require.NoError(t, err,
		"golden file %s must exist; run with -update to create it", goldenPath)

	assert.JSONEq(t, string(golden), string(data),
		"behavioral rules for ask_user_question strategy must match golden snapshot")
}
