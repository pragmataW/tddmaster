package cmd

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeTDDExecutingState produces a minimal EXECUTING state parked in the given
// TDD cycle. Mirrors the private helper in internal/state/machine_tdd_test.go.
func makeTDDExecutingState(cycle string) state.StateFile {
	st := state.CreateInitialState()
	specName := "spec-x"
	st.Phase = state.PhaseExecuting
	st.Spec = &specName
	st.Execution.TDDCycle = cycle
	return st
}

// --- Açık A: applyVerifierReport submit-loss guard ---

func TestApplyVerifierReport_GreenPassWithHintInOutputButNoRefactorNotes_Rejects(t *testing.T) {
	st := makeTDDExecutingState(state.TDDCycleGreen)
	report := map[string]interface{}{
		"passed": true,
		"output": "all tests pass. consider extracting this helper into a shared util.",
	}

	newState, err := applyVerifierReport(st, nil, report)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "`refactorNotes` field is empty")
	assert.Equal(t, state.TDDCycleGreen, newState.Execution.TDDCycle, "state must not mutate on guard rejection")
}

func TestApplyVerifierReport_GreenPassWithExplicitEmptyRefactorNotes_Passes(t *testing.T) {
	st := makeTDDExecutingState(state.TDDCycleGreen)
	report := map[string]interface{}{
		"passed":        true,
		"output":        "clean, consider future refactor maybe — but nothing needed now",
		"refactorNotes": []interface{}{},
	}

	newState, err := applyVerifierReport(st, nil, report)

	require.NoError(t, err, "explicit empty array must bypass the hint regex")
	assert.Equal(t, "", newState.Execution.TDDCycle, "GREEN PASS with no notes → cycle clears")
}

func TestApplyVerifierReport_GreenPassWithNotesIncluded_AdvancesToRefactor(t *testing.T) {
	st := makeTDDExecutingState(state.TDDCycleGreen)
	report := map[string]interface{}{
		"passed": true,
		"output": "tests pass; found duplication to extract",
		"refactorNotes": []interface{}{
			map[string]interface{}{
				"file":       "a.go",
				"suggestion": "extract helper",
				"rationale":  "dedupe",
			},
		},
	}

	newState, err := applyVerifierReport(st, nil, report)

	require.NoError(t, err)
	assert.Equal(t, state.TDDCycleRefactor, newState.Execution.TDDCycle)
	require.NotNil(t, newState.Execution.LastVerification)
	require.Len(t, newState.Execution.LastVerification.RefactorNotes, 1)
	assert.Equal(t, "a.go", newState.Execution.LastVerification.RefactorNotes[0].File)
}

func TestApplyVerifierReport_GreenFailure_NotEvaluatedForHint(t *testing.T) {
	st := makeTDDExecutingState(state.TDDCycleGreen)
	report := map[string]interface{}{
		"passed":    false,
		"output":    "tests failed, also consider a refactor later",
		"failedACs": []interface{}{"ac-1"},
	}

	_, err := applyVerifierReport(st, nil, report)

	require.NoError(t, err, "guard fires only on passed:true; failures take the retry path")
}

func TestApplyVerifierReport_RedPass_NotEvaluatedForHint(t *testing.T) {
	st := makeTDDExecutingState(state.TDDCycleRed)
	report := map[string]interface{}{
		"passed": true,
		"output": "tests fail as expected; should refactor the scaffold eventually",
	}

	newState, err := applyVerifierReport(st, nil, report)

	require.NoError(t, err, "guard is gated by TDDCycle==green")
	assert.Equal(t, state.TDDCycleGreen, newState.Execution.TDDCycle, "RED PASS advances to GREEN regardless of output prose")
}

func TestApplyVerifierReport_NonTDD_NotEvaluatedForHint(t *testing.T) {
	st := state.CreateInitialState()
	specName := "spec-y"
	st.Phase = state.PhaseExecuting
	st.Spec = &specName
	// TDDCycle stays empty → non-TDD path
	report := map[string]interface{}{
		"passed": true,
		"output": "please refactor the whole thing",
	}

	_, err := applyVerifierReport(st, nil, report)

	require.NoError(t, err, "non-TDD submits must not be evaluated by the refactor hint guard")
}

// --- Açık B: applyExecutorReport bypass guard ---

func makeRefactorPhaseState(withPendingNotes bool) state.StateFile {
	st := makeTDDExecutingState(state.TDDCycleRefactor)
	st.Execution.RefactorApplied = false
	st.Execution.LastVerification = &state.VerificationResult{
		Passed:    true,
		Output:    "green scan produced notes",
		Phase:     state.TDDCycleGreen,
		Timestamp: "2026-04-17T00:00:00Z",
	}
	if withPendingNotes {
		st.Execution.LastVerification.RefactorNotes = []state.RefactorNote{
			{File: "engine.go", Suggestion: "extract defaultInteractionHints", Rationale: "duplication"},
		}
	}
	return st
}

func TestApplyExecutorReport_InRefactorWithPendingNotesAndCompleted_Rejects(t *testing.T) {
	st := makeRefactorPhaseState(true)
	prevIter := st.Execution.Iteration

	report := map[string]interface{}{
		"completed": []interface{}{"task-1"},
	}

	newState, err := applyExecutorReport(st, nil, report)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "apply the refactor notes first")
	assert.Equal(t, prevIter, newState.Execution.Iteration, "iteration must not bump on rejection")
	assert.Empty(t, newState.Execution.CompletedTasks, "task must not be appended to CompletedTasks on rejection")
	assert.Equal(t, state.TDDCycleRefactor, newState.Execution.TDDCycle, "cycle must remain in REFACTOR")
}

func TestApplyExecutorReport_InRefactorWithRefactorAppliedTrue_Passes(t *testing.T) {
	st := makeRefactorPhaseState(true)

	report := map[string]interface{}{
		"refactorApplied": true,
	}

	newState, err := applyExecutorReport(st, nil, report)

	require.NoError(t, err)
	assert.True(t, newState.Execution.RefactorApplied, "MarkRefactorApplied must flip the flag")
	assert.Equal(t, state.TDDCycleRefactor, newState.Execution.TDDCycle)
}

func TestApplyExecutorReport_InRefactorWithRefactorAppliedAndCompleted_Passes(t *testing.T) {
	st := makeRefactorPhaseState(true)

	report := map[string]interface{}{
		"refactorApplied": true,
		"completed":       []interface{}{"task-1"},
	}

	newState, err := applyExecutorReport(st, nil, report)

	require.NoError(t, err, "same-submit apply+complete is legal — verifier will re-scan next")
	assert.Contains(t, newState.Execution.CompletedTasks, "task-1")
	// Note: RefactorApplied may be reset by reseedTDDCycleIfNeeded once the task
	// advances (nil config → non-TDD reseed clears the flag). The invariant we
	// care about is: no error, and the task landed in CompletedTasks.
}

func TestApplyExecutorReport_InRefactorNoPendingNotes_Passes(t *testing.T) {
	st := makeRefactorPhaseState(false)

	report := map[string]interface{}{
		"completed": []interface{}{"task-1"},
	}

	_, err := applyExecutorReport(st, nil, report)

	require.NoError(t, err, "no pending notes → guard does not fire")
}

func TestApplyExecutorReport_GreenPhaseCompleted_NoGuardApplies(t *testing.T) {
	st := makeTDDExecutingState(state.TDDCycleGreen)

	report := map[string]interface{}{
		"completed": []interface{}{"task-1"},
	}

	_, err := applyExecutorReport(st, nil, report)

	require.NoError(t, err, "guard is gated by TDDCycle==refactor")
}

func TestApplyExecutorReport_NonTDDCompleted_NoGuardApplies(t *testing.T) {
	st := state.CreateInitialState()
	specName := "spec-z"
	st.Phase = state.PhaseExecuting
	st.Spec = &specName
	// TDDCycle stays empty → non-TDD path

	report := map[string]interface{}{
		"completed": []interface{}{"task-1"},
	}

	_, err := applyExecutorReport(st, nil, report)

	require.NoError(t, err, "non-TDD submits must not be evaluated by the refactor bypass guard")
}
