// Package behavioral_test exercises the conditional verifier-delegation
// rendering introduced by task-4 of the skip-verify-zellii spec.
//
// The tests call behavioral.Build with an extra verifierRequired bool
// parameter that does not exist yet — every test in this file will fail
// to compile until the GREEN implementation adds that parameter.  That is
// the expected RED state.
package behavioral_test

import (
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/context/service/behavioral"
	"github.com/pragmataW/tddmaster/internal/state"
)

// fakeRenderer implements behavioral.Renderer with predictable output so
// tests are independent of command-prefix configuration.
type fakeRenderer struct{}

func (fakeRenderer) C(sub string) string              { return "tddmaster " + sub }
func (fakeRenderer) CS(sub string, _ *string) string  { return "tddmaster " + sub }

// executingStateWithCycle builds the minimal StateFile for PhaseExecuting
// with a single TDD-enabled task and the given TDDCycle value.
func executingStateWithCycle(tddCycle string) state.StateFile {
	st := state.CreateInitialState()
	st.Phase = state.PhaseExecuting
	tddOn := true
	st.OverrideTasks = []state.SpecTask{
		{ID: "task-1", Title: "task one", TDDEnabled: &tddOn},
	}
	st.Execution.TDDCycle = tddCycle
	return st
}

// executingStateNoTDD builds the minimal StateFile for PhaseExecuting
// without a TDD cycle (TDD off).
func executingStateNoTDD() state.StateFile {
	st := state.CreateInitialState()
	st.Phase = state.PhaseExecuting
	tddOff := false
	st.OverrideTasks = []state.SpecTask{
		{ID: "task-1", Title: "task one", TDDEnabled: &tddOff},
	}
	return st
}

// rulesContain returns true when any rule in the slice contains substr
// (case-sensitive).
func rulesContain(rules []string, substr string) bool {
	for _, r := range rules {
		if strings.Contains(r, substr) {
			return true
		}
	}
	return false
}

// buildExecuting invokes behavioral.Build for the EXECUTING phase using
// the provided state and the verifierRequired flag that the GREEN
// implementation must accept.
//
// NOTE: This call will fail to compile until behavioral.Build gains the
// verifierRequired parameter — that is the RED failure.
func buildExecuting(t *testing.T, st state.StateFile, verifierRequired bool) model.BehavioralBlock {
	t.Helper()
	return behavioral.Build(
		fakeRenderer{},
		st,
		10,    // maxIterationsBeforeRestart
		false, // allowGit
		nil,   // activeConcerns
		nil,   // parsedSpec
		model.DefaultHints,
		verifierRequired, // new parameter — does not exist yet → RED
	)
}

// ---------------------------------------------------------------------------
// AC-1: VerifierRequired=true → verifier-delegation sentences are present
// (regression: existing behavior must be preserved).
// ---------------------------------------------------------------------------

// TestBehavioral_VerifierRequired_True_VerifierSentencesPresent asserts that
// when verifierRequired=true the rendered rules include a verifier-delegation
// sentence.  This is the regression guard for AC-1.
func TestBehavioral_VerifierRequired_True_VerifierSentencesPresent(t *testing.T) {
	t.Parallel()
	st := executingStateWithCycle(state.TDDCycleGreen)
	block := buildExecuting(t, st, true)

	if !rulesContain(block.Rules, "verifier") {
		t.Error("expected at least one rule containing 'verifier' when verifierRequired=true, found none")
	}
}

// TestBehavioral_VerifierRequired_True_NoTDD_VerifierSentencesPresent covers
// AC-1 for the non-TDD case: even without a TDD cycle, verifierRequired=true
// must still produce verifier-delegation language.
func TestBehavioral_VerifierRequired_True_NoTDD_VerifierSentencesPresent(t *testing.T) {
	t.Parallel()
	st := executingStateNoTDD()
	block := buildExecuting(t, st, true)

	if !rulesContain(block.Rules, "verifier") {
		t.Error("expected verifier sentence when verifierRequired=true (TDD off), found none")
	}
}

// ---------------------------------------------------------------------------
// AC-2: VerifierRequired=false + TDD=off → no "verifier" word in executor
// or test-writer sections of the prompt.
// ---------------------------------------------------------------------------

// TestBehavioral_VerifierRequired_False_TDDOff_NoVerifierWord asserts that
// when verifierRequired=false and TDD is not active the rendered rules contain
// no reference to "verifier" in the executor/delegation sentences.
func TestBehavioral_VerifierRequired_False_TDDOff_NoVerifierWord(t *testing.T) {
	t.Parallel()
	st := executingStateNoTDD()
	block := buildExecuting(t, st, false)

	for _, r := range block.Rules {
		// The TDD behavioural rules (e.g. "tddmaster-verifier") are injected
		// separately; we only check the executor-delegation sentences produced
		// by phaseExecutingBehavioral itself.  Filter out TDD-prefixed rules.
		if strings.HasPrefix(r, "TDD REQUIRED") {
			continue
		}
		if strings.Contains(r, "tddmaster-verifier") || strings.Contains(r, "spawn tddmaster-verifier") {
			t.Errorf("rule must not mention verifier when verifierRequired=false (TDD off): %q", r)
		}
	}

	// The "VERIFICATION REQUIRED: … spawn tddmaster-verifier" sentence must be
	// absent from delegation rules.
	if rulesContain(block.Rules, "spawn tddmaster-verifier") {
		t.Error("spawn-verifier sentence must be absent when verifierRequired=false (TDD off)")
	}
}

// ---------------------------------------------------------------------------
// AC-3: VerifierRequired=false + TDD=on + phase=red/refactor →
// prompt includes "do not spawn verifier" or "verifier çağırma" directive
// with "report directly to next".
// ---------------------------------------------------------------------------

// TestBehavioral_VerifierRequired_False_TDDOn_PhaseRed_SkipVerifierDirective
// asserts that a skip-verifier directive appears when verifierRequired=false
// and the TDD cycle is in the RED phase.
func TestBehavioral_VerifierRequired_False_TDDOn_PhaseRed_SkipVerifierDirective(t *testing.T) {
	t.Parallel()
	st := executingStateWithCycle(state.TDDCycleRed)
	block := buildExecuting(t, st, false)

	// The prompt must tell the orchestrator not to spawn the verifier and to
	// report directly.  Accept any of the canonical phrasings.
	hasSkipDirective := rulesContain(block.Rules, "do not spawn verifier") ||
		rulesContain(block.Rules, "verifier çağırma") ||
		rulesContain(block.Rules, "skip verifier") ||
		rulesContain(block.Rules, "verifier atla")
	if !hasSkipDirective {
		t.Error("expected a skip-verifier directive when verifierRequired=false + TDD=on + phase=red, found none")
	}

	hasReportDirectly := rulesContain(block.Rules, "report directly") ||
		rulesContain(block.Rules, "doğrudan")
	if !hasReportDirectly {
		t.Error("expected 'report directly to next' language when verifierRequired=false + TDD=on + phase=red, found none")
	}
}

// TestBehavioral_VerifierRequired_False_TDDOn_PhaseRefactor_SkipVerifierDirective
// mirrors the RED phase test for the REFACTOR phase (also verifier-skipped per
// VerifierRequired logic).
func TestBehavioral_VerifierRequired_False_TDDOn_PhaseRefactor_SkipVerifierDirective(t *testing.T) {
	t.Parallel()
	st := executingStateWithCycle(state.TDDCycleRefactor)
	block := buildExecuting(t, st, false)

	hasSkipDirective := rulesContain(block.Rules, "do not spawn verifier") ||
		rulesContain(block.Rules, "verifier çağırma") ||
		rulesContain(block.Rules, "skip verifier") ||
		rulesContain(block.Rules, "verifier atla")
	if !hasSkipDirective {
		t.Error("expected a skip-verifier directive when verifierRequired=false + TDD=on + phase=refactor, found none")
	}

	hasReportDirectly := rulesContain(block.Rules, "report directly") ||
		rulesContain(block.Rules, "doğrudan")
	if !hasReportDirectly {
		t.Error("expected 'report directly to next' language when verifierRequired=false + TDD=on + phase=refactor, found none")
	}
}

// ---------------------------------------------------------------------------
// AC-4: VerifierRequired=true + TDD=on + phase=green + skipVerify=true →
// GREEN segment strengthens: "refactorNotes ZORUNLU" or equivalent mandatory
// language appears.
// ---------------------------------------------------------------------------

// TestBehavioral_VerifierRequired_True_TDDOn_PhaseGreen_RefactorNotesMandatory
// asserts that when verifierRequired=true AND the phase is green the rendered
// rules include mandatory language about refactorNotes.  This is the
// "strengthened GREEN" requirement from AC-4.
func TestBehavioral_VerifierRequired_True_TDDOn_PhaseGreen_RefactorNotesMandatory(t *testing.T) {
	t.Parallel()
	st := executingStateWithCycle(state.TDDCycleGreen)
	block := buildExecuting(t, st, true)

	hasMandatoryRefactorNotes := rulesContain(block.Rules, "refactorNotes ZORUNLU") ||
		rulesContain(block.Rules, "refactorNotes mandatory") ||
		rulesContain(block.Rules, "refactorNotes MANDATORY") ||
		rulesContain(block.Rules, "ZORUNLU") ||
		rulesContain(block.Rules, "refactorNotes are mandatory")
	if !hasMandatoryRefactorNotes {
		t.Error("expected mandatory refactorNotes language in GREEN segment when verifierRequired=true + TDD=on + phase=green, found none")
	}
}

// ---------------------------------------------------------------------------
// AC-5: Verifier-delegation sentences at specific lines are gated on
// verifierRequired condition — when false they must be absent, when true
// they must be present.
// ---------------------------------------------------------------------------

// TestBehavioral_VerifierDelegationGating_SubAgentTask_VerifierRequired_True
// checks that verifier delegation is included when verifierRequired=true and
// the sub-agent method is "task".
func TestBehavioral_VerifierDelegationGating_SubAgentTask_VerifierRequired_True(t *testing.T) {
	t.Parallel()
	st := executingStateNoTDD()
	hints := model.DefaultHints // SubAgentMethod = "task"

	block := behavioral.Build(
		fakeRenderer{},
		st,
		10,
		false,
		nil,
		nil,
		hints,
		true, // verifierRequired
	)

	// The verifier delegation sentence for "task" method contains "spawn
	// tddmaster-verifier".
	if !rulesContain(block.Rules, "spawn tddmaster-verifier") {
		t.Error("expected 'spawn tddmaster-verifier' delegation sentence when verifierRequired=true (subAgentMethod=task)")
	}
}

// TestBehavioral_VerifierDelegationGating_SubAgentTask_VerifierRequired_False
// checks that verifier delegation is suppressed when verifierRequired=false
// even with subAgentMethod=task.
func TestBehavioral_VerifierDelegationGating_SubAgentTask_VerifierRequired_False(t *testing.T) {
	t.Parallel()
	st := executingStateNoTDD()
	hints := model.DefaultHints // SubAgentMethod = "task"

	block := behavioral.Build(
		fakeRenderer{},
		st,
		10,
		false,
		nil,
		nil,
		hints,
		false, // verifierRequired
	)

	if rulesContain(block.Rules, "spawn tddmaster-verifier") {
		t.Error("must not contain 'spawn tddmaster-verifier' when verifierRequired=false (subAgentMethod=task)")
	}
}

// TestBehavioral_VerifierDelegationGating_SubAgentDelegation_VerifierRequired_True
// checks that "delegate to tddmaster-verifier" appears when verifierRequired=true
// and subAgentMethod=delegation.
func TestBehavioral_VerifierDelegationGating_SubAgentDelegation_VerifierRequired_True(t *testing.T) {
	t.Parallel()
	st := executingStateNoTDD()
	hints := model.InteractionHints{
		HasAskUserTool:        true,
		OptionPresentation:    "tool",
		HasSubAgentDelegation: true,
		SubAgentMethod:        "delegation",
		AskUserStrategy:       "ask_user_question",
	}

	block := behavioral.Build(
		fakeRenderer{},
		st,
		10,
		false,
		nil,
		nil,
		hints,
		true, // verifierRequired
	)

	if !rulesContain(block.Rules, "tddmaster-verifier") {
		t.Error("expected tddmaster-verifier delegation sentence when verifierRequired=true (subAgentMethod=delegation)")
	}
}

// TestBehavioral_VerifierDelegationGating_SubAgentDelegation_VerifierRequired_False
// checks that verifier delegation is absent when verifierRequired=false even
// with subAgentMethod=delegation.
func TestBehavioral_VerifierDelegationGating_SubAgentDelegation_VerifierRequired_False(t *testing.T) {
	t.Parallel()
	st := executingStateNoTDD()
	hints := model.InteractionHints{
		HasAskUserTool:        true,
		OptionPresentation:    "tool",
		HasSubAgentDelegation: true,
		SubAgentMethod:        "delegation",
		AskUserStrategy:       "ask_user_question",
	}

	block := behavioral.Build(
		fakeRenderer{},
		st,
		10,
		false,
		nil,
		nil,
		hints,
		false, // verifierRequired
	)

	// For delegation method the sentence is "delegate to tddmaster-verifier".
	if rulesContain(block.Rules, "delegate to tddmaster-verifier") {
		t.Error("must not contain 'delegate to tddmaster-verifier' when verifierRequired=false (subAgentMethod=delegation)")
	}
}

// TestBehavioral_VerifierDelegationGating_NoSubAgents_VerifierRequired_True
// checks that the no-sub-agent VERIFICATION REQUIRED sentence references
// type-check + tests when verifierRequired=true (no sub-agents path).
func TestBehavioral_VerifierDelegationGating_NoSubAgents_VerifierRequired_True(t *testing.T) {
	t.Parallel()
	st := executingStateNoTDD()
	hints := model.InteractionHints{
		SubAgentMethod:  "none",
		AskUserStrategy: "ask_user_question",
	}

	block := behavioral.Build(
		fakeRenderer{},
		st,
		10,
		false,
		nil,
		nil,
		hints,
		true, // verifierRequired
	)

	if !rulesContain(block.Rules, "VERIFICATION REQUIRED") {
		t.Error("expected VERIFICATION REQUIRED sentence when verifierRequired=true (no sub-agents)")
	}
}

// TestBehavioral_VerifierDelegationGating_NoSubAgents_VerifierRequired_False
// checks that the VERIFICATION REQUIRED sentence is absent when
// verifierRequired=false and no sub-agents are used.
func TestBehavioral_VerifierDelegationGating_NoSubAgents_VerifierRequired_False(t *testing.T) {
	t.Parallel()
	st := executingStateNoTDD()
	hints := model.InteractionHints{
		SubAgentMethod:  "none",
		AskUserStrategy: "ask_user_question",
	}

	block := behavioral.Build(
		fakeRenderer{},
		st,
		10,
		false,
		nil,
		nil,
		hints,
		false, // verifierRequired
	)

	if rulesContain(block.Rules, "VERIFICATION REQUIRED") {
		t.Error("must not contain VERIFICATION REQUIRED when verifierRequired=false (no sub-agents)")
	}
}

// ---------------------------------------------------------------------------
// AC-1 regression: existing table of sub-agent methods with verifierRequired=true
// ---------------------------------------------------------------------------

// TestBehavioral_VerifierRequired_True_AllSubAgentMethods_VerifierPresent is a
// table-driven regression test ensuring no existing sub-agent method loses its
// verifier sentence when verifierRequired=true.
func TestBehavioral_VerifierRequired_True_AllSubAgentMethods_VerifierPresent(t *testing.T) {
	t.Parallel()
	methods := []string{"task", "spawn", "fleet", "delegation"}
	for _, method := range methods {
		method := method
		t.Run("method="+method, func(t *testing.T) {
			t.Parallel()
			st := executingStateNoTDD()
			hints := model.InteractionHints{
				HasAskUserTool:        true,
				OptionPresentation:    "tool",
				HasSubAgentDelegation: true,
				SubAgentMethod:        method,
				AskUserStrategy:       "ask_user_question",
			}
			block := behavioral.Build(
				fakeRenderer{},
				st,
				10,
				false,
				nil,
				nil,
				hints,
				true, // verifierRequired
			)
			if !rulesContain(block.Rules, "verifier") {
				t.Errorf("method=%s: expected verifier sentence when verifierRequired=true, found none", method)
			}
		})
	}
}

// TestBehavioral_VerifierRequired_False_AllSubAgentMethods_NoSpawnVerifier
// asserts that no sub-agent method produces a spawn-verifier sentence when
// verifierRequired=false.
func TestBehavioral_VerifierRequired_False_AllSubAgentMethods_NoSpawnVerifier(t *testing.T) {
	t.Parallel()
	methods := []string{"task", "spawn", "fleet", "delegation"}
	for _, method := range methods {
		method := method
		t.Run("method="+method, func(t *testing.T) {
			t.Parallel()
			st := executingStateNoTDD()
			hints := model.InteractionHints{
				HasAskUserTool:        true,
				OptionPresentation:    "tool",
				HasSubAgentDelegation: true,
				SubAgentMethod:        method,
				AskUserStrategy:       "ask_user_question",
			}
			block := behavioral.Build(
				fakeRenderer{},
				st,
				10,
				false,
				nil,
				nil,
				hints,
				false, // verifierRequired
			)
			// Spawn/delegate verifier sentences must be absent.
			for _, r := range block.Rules {
				if strings.HasPrefix(r, "TDD REQUIRED") {
					continue
				}
				if strings.Contains(r, "spawn tddmaster-verifier") ||
					strings.Contains(r, "delegate to tddmaster-verifier") {
					t.Errorf("method=%s: verifier delegation sentence must be absent when verifierRequired=false: %q", method, r)
				}
			}
		})
	}
}
