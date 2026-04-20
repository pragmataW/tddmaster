package context_test

import (
	"testing"

	ctx "github.com/pragmataW/tddmaster/internal/context"
	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/state"
)

// executingState builds a minimal StateFile in PhaseExecuting with one TDD-
// enabled task and the given TDDCycle set.
func executingState(tddCycle string) state.StateFile {
	st := state.CreateInitialState()
	st.Phase = state.PhaseExecuting
	tddOn := true
	st.OverrideTasks = []state.SpecTask{
		{ID: "task-1", Title: "task one", TDDEnabled: &tddOn},
	}
	st.Execution.TDDCycle = tddCycle
	return st
}

// manifestTDD builds a NosManifest with TddMode and SkipVerify set.
func manifestTDD(tddMode, skipVerify bool) *state.NosManifest {
	return &state.NosManifest{
		Tdd: &state.Manifest{
			TddMode:    tddMode,
			SkipVerify: skipVerify,
		},
	}
}

// compileExec is a helper that runs ctx.Compile for the executing phase and
// returns the ExecutionOutput (asserts it is non-nil inside).
func compileExec(t *testing.T, st state.StateFile, cfg *state.NosManifest) model.ExecutionOutput {
	t.Helper()
	out := ctx.Compile(model.CompileInput{
		State:  st,
		Config: cfg,
	})
	if out.ExecutionData == nil {
		t.Fatal("expected ExecutionData to be non-nil for EXECUTING phase")
	}
	return *out.ExecutionData
}

// TestCompileExecution_VerifierRequired_SkipVerifyFalse_BackwardCompat covers AC-6 and AC-8.
// When skipVerify is false (default), VerifierRequired must be true for all TDD phases
// and TDDVerificationContext must be populated, preserving backward compatibility.
func TestCompileExecution_VerifierRequired_SkipVerifyFalse_BackwardCompat(t *testing.T) {
	t.Parallel()
	phases := []string{
		state.TDDCycleRed,
		state.TDDCycleGreen,
		state.TDDCycleRefactor,
	}
	for _, phase := range phases {
		phase := phase
		t.Run("phase="+phase, func(t *testing.T) {
			t.Parallel()
			st := executingState(phase)
			cfg := manifestTDD(true, false) // skipVerify=false (default)
			exec := compileExec(t, st, cfg)

			// AC-6: VerifierRequired set by helper (skipVerify=false → always true).
			if !exec.VerifierRequired {
				t.Errorf("phase=%s: VerifierRequired = false; want true when skipVerify=false", phase)
			}
			// AC-7: TDDVerificationContext populated when VerifierRequired=true.
			if exec.TDDVerificationContext == nil {
				t.Errorf("phase=%s: TDDVerificationContext = nil; want non-nil when VerifierRequired=true", phase)
			}
			// AC-8: TDDPhase is always set (not gated by VerifierRequired).
			if exec.TDDPhase == nil || *exec.TDDPhase != phase {
				t.Errorf("phase=%s: TDDPhase = %v; want %q", phase, exec.TDDPhase, phase)
			}
		})
	}
}

// TestCompileExecution_VerifierRequired_SkipVerifyTrue_TDDOff covers AC-6, AC-7, AC-8.
// skipVerify=true + TDD=off → VerifierRequired=false, TDDVerificationContext=nil.
// TDDPhase is not applicable here because TDD cycle won't be running.
func TestCompileExecution_VerifierRequired_SkipVerifyTrue_TDDOff(t *testing.T) {
	t.Parallel()
	// When TDD is off, ShouldRunTDDForCurrentTask returns false → TDDPhase not set,
	// but VerifierRequired must reflect skip logic.
	st := state.CreateInitialState()
	st.Phase = state.PhaseExecuting
	tddOff := false
	st.OverrideTasks = []state.SpecTask{
		{ID: "task-1", Title: "task one", TDDEnabled: &tddOff},
	}
	// No TDDCycle set since TDD is off.
	cfg := manifestTDD(false, true) // TDD=off, skipVerify=true

	exec := compileExec(t, st, cfg)

	// AC-6: VerifierRequired=false when skipVerify=true and TDD=off.
	if exec.VerifierRequired {
		t.Error("VerifierRequired = true; want false when skipVerify=true and TDD=off")
	}
	// AC-7: TDDVerificationContext=nil when VerifierRequired=false.
	if exec.TDDVerificationContext != nil {
		t.Error("TDDVerificationContext non-nil; want nil when VerifierRequired=false")
	}
}

// TestCompileExecution_VerifierRequired_SkipVerifyTrue_TDDOn_PhaseGreen covers AC-3, AC-6, AC-7, AC-8.
// skipVerify=true + TDD=on + phase=green → VerifierRequired=true, TDDVerificationContext populated.
func TestCompileExecution_VerifierRequired_SkipVerifyTrue_TDDOn_PhaseGreen(t *testing.T) {
	t.Parallel()
	st := executingState(state.TDDCycleGreen)
	cfg := manifestTDD(true, true) // TDD=on, skipVerify=true

	exec := compileExec(t, st, cfg)

	// AC-3 / AC-6: green phase is the exception — verifier still required.
	if !exec.VerifierRequired {
		t.Error("VerifierRequired = false; want true for skipVerify=true + TDD=on + phase=green")
	}
	// AC-7: TDDVerificationContext must be populated because VerifierRequired=true.
	if exec.TDDVerificationContext == nil {
		t.Error("TDDVerificationContext = nil; want non-nil when VerifierRequired=true (green phase)")
	}
	// AC-8: TDDPhase still set.
	if exec.TDDPhase == nil || *exec.TDDPhase != state.TDDCycleGreen {
		t.Errorf("TDDPhase = %v; want %q", exec.TDDPhase, state.TDDCycleGreen)
	}
}

// TestCompileExecution_VerifierRequired_SkipVerifyTrue_TDDOn_PhaseRed covers AC-4, AC-6, AC-7, AC-8.
// skipVerify=true + TDD=on + phase=red → VerifierRequired=false, TDDVerificationContext=nil,
// but TDDPhase="red" is still set.
func TestCompileExecution_VerifierRequired_SkipVerifyTrue_TDDOn_PhaseRed(t *testing.T) {
	t.Parallel()
	st := executingState(state.TDDCycleRed)
	cfg := manifestTDD(true, true) // TDD=on, skipVerify=true

	exec := compileExec(t, st, cfg)

	// AC-4 / AC-6: red phase → verifier not required.
	if exec.VerifierRequired {
		t.Error("VerifierRequired = true; want false for skipVerify=true + TDD=on + phase=red")
	}
	// AC-7: TDDVerificationContext must be nil when VerifierRequired=false.
	if exec.TDDVerificationContext != nil {
		t.Error("TDDVerificationContext non-nil; want nil when VerifierRequired=false (red phase)")
	}
	// AC-8: TDDPhase must still be set regardless of VerifierRequired.
	if exec.TDDPhase == nil || *exec.TDDPhase != state.TDDCycleRed {
		t.Errorf("TDDPhase = %v; want %q", exec.TDDPhase, state.TDDCycleRed)
	}
}

// TestCompileExecution_VerifierRequired_SkipVerifyTrue_TDDOn_PhaseRefactor covers AC-4, AC-6, AC-7, AC-8.
// skipVerify=true + TDD=on + phase=refactor → VerifierRequired=false, TDDVerificationContext=nil,
// but TDDPhase="refactor" is still set.
func TestCompileExecution_VerifierRequired_SkipVerifyTrue_TDDOn_PhaseRefactor(t *testing.T) {
	t.Parallel()
	st := executingState(state.TDDCycleRefactor)
	cfg := manifestTDD(true, true) // TDD=on, skipVerify=true

	exec := compileExec(t, st, cfg)

	// AC-4 / AC-6: refactor phase → verifier not required.
	if exec.VerifierRequired {
		t.Error("VerifierRequired = true; want false for skipVerify=true + TDD=on + phase=refactor")
	}
	// AC-7: TDDVerificationContext must be nil when VerifierRequired=false.
	if exec.TDDVerificationContext != nil {
		t.Error("TDDVerificationContext non-nil; want nil when VerifierRequired=false (refactor phase)")
	}
	// AC-8: TDDPhase must still be set.
	if exec.TDDPhase == nil || *exec.TDDPhase != state.TDDCycleRefactor {
		t.Errorf("TDDPhase = %v; want %q", exec.TDDPhase, state.TDDCycleRefactor)
	}
}
