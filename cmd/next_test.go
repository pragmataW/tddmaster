package cmd

import (
	"strings"
	"testing"

	ctxpkg "github.com/pragmataW/tddmaster/internal/context"
	ctxmodel "github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/state"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// skipVerifyManifest builds a NosManifest with SkipVerify + optional TDD mode.
func skipVerifyManifest(tddMode, skipVerify bool) *state.NosManifest {
	return &state.NosManifest{
		Tdd: &state.Manifest{
			TddMode:    tddMode,
			SkipVerify: skipVerify,
		},
	}
}

// baseExecutingState returns a minimal StateFile in EXECUTING phase with one
// pending task and the given TDDCycle value.
func baseExecutingState(tddCycle string) state.StateFile {
	tddEnabled := true
	return state.StateFile{
		Phase: state.PhaseExecuting,
		OverrideTasks: []state.SpecTask{
			{ID: "task-1", Title: "first task", TDDEnabled: &tddEnabled},
		},
		Execution: state.ExecutionState{
			TDDCycle:  tddCycle,
			Iteration: 1,
		},
	}
}

// executorReport builds a minimal executor status report.
func executorReport(completedIDs ...string) map[string]interface{} {
	completed := make([]interface{}, len(completedIDs))
	for i, id := range completedIDs {
		completed[i] = id
	}
	return map[string]interface{}{
		"completed": completed,
		"remaining": []interface{}{},
	}
}

// verifierReport builds a minimal verifier report with a "passed" field.
func verifierReport(passed bool) map[string]interface{} {
	return map[string]interface{}{
		"passed":       passed,
		"output":       "test output",
		"refactorNotes": []interface{}{},
	}
}

// ---------------------------------------------------------------------------
// AC-1: Guard — verifierRequired=false + hasVerifierShape → error
// ---------------------------------------------------------------------------

// TestVerifierPayloadGuard_SkipVerifyTrue_VerifierReportSubmitted asserts that
// when skipVerify=true (verifierRequired=false) but the submitted report has a
// verifier shape (contains "passed"), an error is returned containing both
// "verifier report submitted but skipVerify=true" and the current phase.
func TestVerifierPayloadGuard_SkipVerifyTrue_VerifierReportSubmitted(t *testing.T) {
	cfg := skipVerifyManifest(true, true)
	st := baseExecutingState(state.TDDCycleRed)
	report := verifierReport(true)

	_, err := verifierPayloadGuard(st, cfg, report)

	if err == nil {
		t.Fatal("expected error when verifier report submitted with skipVerify=true, got nil")
	}
	if !strings.Contains(err.Error(), "verifier report submitted but skipVerify=true") {
		t.Errorf("error message %q does not contain expected substring %q", err.Error(), "verifier report submitted but skipVerify=true")
	}
	if !strings.Contains(err.Error(), state.TDDCycleRed) {
		t.Errorf("error message %q does not contain current phase %q", err.Error(), state.TDDCycleRed)
	}
}

// TestVerifierPayloadGuard_SkipVerifyTrue_WrappedVerifierReport asserts that
// a verifier report wrapped under "tddVerification" is also caught by the guard.
func TestVerifierPayloadGuard_SkipVerifyTrue_WrappedVerifierReport(t *testing.T) {
	cfg := skipVerifyManifest(true, true)
	st := baseExecutingState(state.TDDCycleRed)
	report := map[string]interface{}{
		"tddVerification": map[string]interface{}{
			"passed": true,
			"output": "wrapped verifier output",
		},
		"completed": []interface{}{},
	}

	_, err := verifierPayloadGuard(st, cfg, report)

	if err == nil {
		t.Fatal("expected error when wrapped verifier report submitted with skipVerify=true, got nil")
	}
	if !strings.Contains(err.Error(), "verifier report submitted but skipVerify=true") {
		t.Errorf("error message %q missing expected substring", err.Error())
	}
}

// ---------------------------------------------------------------------------
// AC-2: verifierRequired=false + no verifier payload → advanceWithoutVerification
// ---------------------------------------------------------------------------

// TestVerifierPayloadGuard_SkipVerifyTrue_ExecutorReport_NoError asserts that
// when skipVerify=true and the report has no verifier shape, the guard returns
// no error.
func TestVerifierPayloadGuard_SkipVerifyTrue_ExecutorReport_NoError(t *testing.T) {
	cfg := skipVerifyManifest(true, true)
	st := baseExecutingState(state.TDDCycleRed)
	report := executorReport("task-1")

	_, err := verifierPayloadGuard(st, cfg, report)

	if err != nil {
		t.Errorf("expected no error for executor report when skipVerify=true, got: %v", err)
	}
}

// TestVerifierPayloadGuard_SkipVerifyTrue_ExecutorReport_RoutesToAdvance asserts
// that the returned state from the guard reflects advancement (not the same
// unchanged state), confirming routing to advanceWithoutVerification.
func TestVerifierPayloadGuard_SkipVerifyTrue_ExecutorReport_RoutesToAdvance(t *testing.T) {
	cfg := skipVerifyManifest(false, true) // TDD=off, skipVerify=true
	st := baseExecutingState("")           // no TDD cycle
	report := executorReport("task-1")

	newSt, err := verifierPayloadGuard(st, cfg, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The state should have advanced (iteration incremented or task completed).
	if newSt.Execution.Iteration <= st.Execution.Iteration &&
		len(newSt.Execution.CompletedTasks) == 0 {
		t.Error("expected state to advance (iteration++ or task completed) but it did not change")
	}
}

// ---------------------------------------------------------------------------
// AC-3: advanceWithoutVerification — TDD=off + skipVerify=true + executor report
// ---------------------------------------------------------------------------

// TestAdvanceWithoutVerification_TDDOff_SkipVerify_TaskCompleted asserts that
// when TDD=off and skipVerify=true, an executor report completing a task causes
// the task to appear in CompletedTasks and the iteration to increment.
func TestAdvanceWithoutVerification_TDDOff_SkipVerify_TaskCompleted(t *testing.T) {
	cfg := skipVerifyManifest(false, true) // TDD=off, skipVerify=true
	st := baseExecutingState("")           // no TDD cycle
	report := executorReport("task-1")

	newSt, err := advanceWithoutVerification(st, cfg, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !contains(newSt.Execution.CompletedTasks, "task-1") {
		t.Errorf("expected task-1 in CompletedTasks, got: %v", newSt.Execution.CompletedTasks)
	}
	if newSt.Execution.Iteration != st.Execution.Iteration+1 {
		t.Errorf("iteration not incremented: got %d, want %d", newSt.Execution.Iteration, st.Execution.Iteration+1)
	}
}

// TestAdvanceWithoutVerification_TDDOff_SkipVerify_NoTaskCompletion asserts
// that with no completed IDs in the report the function still advances
// iteration without erroring.
func TestAdvanceWithoutVerification_TDDOff_SkipVerify_NoTaskCompletion(t *testing.T) {
	cfg := skipVerifyManifest(false, true)
	st := baseExecutingState("")
	report := map[string]interface{}{
		"completed": []interface{}{},
		"remaining": []interface{}{"task-1"},
	}

	newSt, err := advanceWithoutVerification(st, cfg, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newSt.Execution.Iteration != st.Execution.Iteration+1 {
		t.Errorf("iteration not incremented: got %d, want %d", newSt.Execution.Iteration, st.Execution.Iteration+1)
	}
}

// ---------------------------------------------------------------------------
// AC-4: TDD=on + skipVerify=true + phase=red + executor report → GREEN
// ---------------------------------------------------------------------------

// TestAdvanceWithoutVerification_TDDOn_SkipVerify_PhaseRed_TransitionsToGreen
// asserts that when TDD=on and skipVerify=true and current phase=red, an
// executor (test-writer) report causes the state to transition to GREEN.
func TestAdvanceWithoutVerification_TDDOn_SkipVerify_PhaseRed_TransitionsToGreen(t *testing.T) {
	cfg := skipVerifyManifest(true, true) // TDD=on, skipVerify=true
	st := baseExecutingState(state.TDDCycleRed)
	// Executor report in RED phase: tests written, no tasks completed yet.
	report := map[string]interface{}{
		"completed": []interface{}{},
		"remaining": []interface{}{"task-1"},
	}

	newSt, err := advanceWithoutVerification(st, cfg, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newSt.Execution.TDDCycle != state.TDDCycleGreen {
		t.Errorf("expected TDDCycle to be %q (GREEN), got %q", state.TDDCycleGreen, newSt.Execution.TDDCycle)
	}
}

// ---------------------------------------------------------------------------
// AC-5: TDD=on + skipVerify=true + phase=refactor + executor report → task done, next RED
// ---------------------------------------------------------------------------

// TestAdvanceWithoutVerification_TDDOn_SkipVerify_PhaseRefactor_TaskCompleted
// asserts that when TDD=on and skipVerify=true and current phase=refactor, an
// executor report completing the task causes the task to be marked completed
// and the cycle to reset to RED for the next task.
func TestAdvanceWithoutVerification_TDDOn_SkipVerify_PhaseRefactor_TaskCompleted(t *testing.T) {
	cfg := skipVerifyManifest(true, true)
	tddEnabled := true
	st := state.StateFile{
		Phase: state.PhaseExecuting,
		OverrideTasks: []state.SpecTask{
			{ID: "task-1", Title: "first task", Completed: false, TDDEnabled: &tddEnabled},
			{ID: "task-2", Title: "second task", Completed: false, TDDEnabled: &tddEnabled},
		},
		Execution: state.ExecutionState{
			TDDCycle:        state.TDDCycleRefactor,
			Iteration:       3,
			RefactorApplied: true,
		},
	}
	report := map[string]interface{}{
		"completed":       []interface{}{"task-1"},
		"refactorApplied": true,
	}

	newSt, err := advanceWithoutVerification(st, cfg, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !contains(newSt.Execution.CompletedTasks, "task-1") {
		t.Errorf("expected task-1 in CompletedTasks, got: %v", newSt.Execution.CompletedTasks)
	}
	// Next task is also TDD-enabled, so cycle should be re-seeded to RED.
	if newSt.Execution.TDDCycle != state.TDDCycleRed {
		t.Errorf("expected TDDCycle to be %q (RED) for next task, got %q", state.TDDCycleRed, newSt.Execution.TDDCycle)
	}
}

// ---------------------------------------------------------------------------
// AC-6: TDD=on + skipVerify=true + phase=green → verifier REQUIRED, not skipped
// ---------------------------------------------------------------------------

// TestVerifierPayloadGuard_TDDOn_SkipVerify_PhaseGreen_VerifierRequired asserts
// that GREEN phase is the one exception where verifier is always required even
// when skipVerify=true; an executor report in GREEN without a verifier payload
// should NOT be routed through advanceWithoutVerification silently — the guard
// must return an error or the returned state must NOT advance the TDD cycle.
func TestVerifierPayloadGuard_TDDOn_SkipVerify_PhaseGreen_ExecutorReport_Rejected(t *testing.T) {
	cfg := skipVerifyManifest(true, true) // TDD=on, skipVerify=true
	st := baseExecutingState(state.TDDCycleGreen)
	// Executor report (no verifier payload) submitted during GREEN phase.
	report := executorReport()

	_, err := verifierPayloadGuard(st, cfg, report)

	// GREEN phase requires a verifier report; executor report alone must be rejected.
	if err == nil {
		t.Fatal("expected error when executor submits report during GREEN phase with skipVerify=true, got nil")
	}
}

// TestVerifierPayloadGuard_TDDOn_SkipVerify_PhaseGreen_VerifierReport_Accepted
// asserts that a proper verifier report IS accepted during GREEN (even with
// skipVerify=true), because GREEN always requires verification.
func TestVerifierPayloadGuard_TDDOn_SkipVerify_PhaseGreen_VerifierReport_Accepted(t *testing.T) {
	cfg := skipVerifyManifest(true, true) // TDD=on, skipVerify=true
	st := baseExecutingState(state.TDDCycleGreen)
	report := verifierReport(true)

	// In GREEN phase, the verifier report should NOT be rejected by the guard.
	// The guard should return an error ONLY for submitting verifier in non-green phases.
	// For GREEN + verifier report this should NOT hit the "skipVerify" guard path.
	_, err := verifierPayloadGuard(st, cfg, report)

	// The guard must not reject a verifier report during GREEN, because GREEN
	// always needs verifier. Either no error or it is handled by applyVerifierReport.
	// We assert that the error message (if any) does not contain the skipVerify guard message.
	if err != nil && strings.Contains(err.Error(), "verifier report submitted but skipVerify=true") {
		t.Errorf("GREEN phase verifier report was incorrectly rejected by the skipVerify guard: %v", err)
	}
}

// ---------------------------------------------------------------------------
// AC-7: applyVerifierReport is NOT called when skipVerify=true + phase≠green
// ---------------------------------------------------------------------------

// TestVerifierPayloadGuard_SkipVerify_NonGreenPhase_ExecutorReport_DoesNotCallApplyVerifier
// asserts that the full pipeline (handleExecutingAnswer-like logic) for
// skipVerify=true + phase=red + executor report does NOT route through
// applyVerifierReport. We verify this by checking the returned state does not
// carry a LastVerification result (applyVerifierReport always sets one).
func TestAdvanceWithoutVerification_DoesNotSetLastVerification(t *testing.T) {
	cfg := skipVerifyManifest(true, true)
	st := baseExecutingState(state.TDDCycleRed)
	report := map[string]interface{}{
		"completed": []interface{}{},
		"remaining": []interface{}{"task-1"},
	}

	newSt, err := advanceWithoutVerification(st, cfg, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// applyVerifierReport always sets Execution.LastVerification. If it was called,
	// LastVerification would be non-nil.
	if newSt.Execution.LastVerification != nil {
		t.Error("applyVerifierReport appears to have been called (LastVerification is set) but should not be called in skip-verify path")
	}
}

// TestAdvanceWithoutVerification_SkipVerify_RefactorPhase_DoesNotSetLastVerification
// asserts the same for refactor phase: applyVerifierReport is never called.
func TestAdvanceWithoutVerification_SkipVerify_RefactorPhase_DoesNotSetLastVerification(t *testing.T) {
	cfg := skipVerifyManifest(true, true)
	st := baseExecutingState(state.TDDCycleRefactor)
	st.Execution.RefactorApplied = true
	report := map[string]interface{}{
		"completed":       []interface{}{"task-1"},
		"refactorApplied": true,
	}

	newSt, err := advanceWithoutVerification(st, cfg, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newSt.Execution.LastVerification != nil {
		t.Error("applyVerifierReport appears to have been called in refactor/skip-verify path")
	}
}

// ---------------------------------------------------------------------------
// E2E: skipVerify=true + TDD=off — verifier is never called
// ---------------------------------------------------------------------------

// TestE2E_SkipVerifyTrue_TDDOff_VerifierNeverCalled is a full-flow integration
// test that exercises the skipVerify=true + TDD=off scenario end-to-end:
//
//  1. Manifest has SkipVerify=true and TddMode=false.
//  2. StateFile is in EXECUTING phase with two pending tasks.
//  3. Executor status report for task-1 is submitted via verifierPayloadGuard.
//  4. No error is returned — verifier delegation is never triggered.
//  5. task-1 appears in CompletedTasks; LastVerification is NOT set.
//  6. advanceWithoutVerification path is confirmed.
//  7. The compiled context for the resulting state has VerifierRequired=false
//     and no TDDVerificationContext.
func TestE2E_SkipVerifyTrue_TDDOff_VerifierNeverCalled(t *testing.T) {
	// AC-1: manifest with SkipVerify=true, TddMode=false.
	cfg := skipVerifyManifest(false, true) // TDD=off, skipVerify=true

	tddOff := false
	st := state.StateFile{
		Phase: state.PhaseExecuting,
		OverrideTasks: []state.SpecTask{
			{ID: "task-1", Title: "first task", TDDEnabled: &tddOff},
			{ID: "task-2", Title: "second task", TDDEnabled: &tddOff},
		},
		Execution: state.ExecutionState{
			TDDCycle:  "", // TDD=off means no active cycle
			Iteration: 1,
		},
	}

	// AC-2: executor status report submitted (simulates "spec next --answer=<report>").
	report := executorReport("task-1")

	// AC-6: advanceWithoutVerification path is taken via verifierPayloadGuard.
	newSt, err := verifierPayloadGuard(st, cfg, report)

	// AC-3: no error — no verifier-delegation instruction emitted.
	if err != nil {
		t.Fatalf("expected no error for skipVerify=true+TDD=off executor report, got: %v", err)
	}

	// AC-4: task-1 completed → state advanced to next task.
	if !contains(newSt.Execution.CompletedTasks, "task-1") {
		t.Errorf("expected task-1 in CompletedTasks after advance, got: %v", newSt.Execution.CompletedTasks)
	}
	if newSt.Execution.Iteration != st.Execution.Iteration+1 {
		t.Errorf("iteration not incremented: got %d, want %d", newSt.Execution.Iteration, st.Execution.Iteration+1)
	}

	// AC-5: LastVerification must NOT be set — applyVerifierReport was never called.
	if newSt.Execution.LastVerification != nil {
		t.Error("LastVerification is set but verifier should never have been called (skipVerify=true + TDD=off)")
	}

	// AC-3 & AC-7: compiled context for the next state must have VerifierRequired=false
	// and no TDDVerificationContext.
	compiled := ctxpkg.Compile(ctxmodel.CompileInput{
		State:  newSt,
		Config: cfg,
	})

	if compiled.ExecutionData == nil {
		t.Fatal("expected ExecutionData in compiled context, got nil")
	}

	// AC-5: VerifierRequired=false.
	if compiled.ExecutionData.VerifierRequired {
		t.Error("compiled context has VerifierRequired=true but expected false for skipVerify=true + TDD=off")
	}

	// AC-3: TDDVerificationContext must not be present — verifier delegation never emitted.
	if compiled.ExecutionData.TDDVerificationContext != nil {
		t.Errorf("compiled context has TDDVerificationContext set but expected nil for skipVerify=true + TDD=off: %+v",
			compiled.ExecutionData.TDDVerificationContext)
	}
}

// ---------------------------------------------------------------------------
// E2E: skipVerify=true + TDD=on — GREEN-only verifier scenario
// ---------------------------------------------------------------------------

// TestE2E_SkipVerifyTrue_TDDOn_GreenOnlyVerifier exercises the full
// RED→GREEN→REFACTOR→next-task-RED cycle when skipVerify=true and TddMode=true.
//
// AC-1: setup has SkipVerify=true, TddMode=true.
// AC-2: RED phase — executor (test-writer) report is accepted, no verifier
//       payload expected; state transitions to GREEN automatically.
//       VerifierRequired=false during RED.
// AC-3: GREEN phase — verifier report with refactorNotes is accepted;
//       PendingRefactorNotes populated, TDDCycle → "refactor".
//       VerifierRequired=true during GREEN.
// AC-4: REFACTOR phase — executor report with refactorApplied=true causes
//       task completion + next task RED seeding.
//       VerifierRequired=false during REFACTOR.
// AC-5: After REFACTOR step the next task's RED is seeded (TDDCycle="red").
// AC-6: Full chain returns no errors.
func TestE2E_SkipVerifyTrue_TDDOn_GreenOnlyVerifier(t *testing.T) {
	// AC-1: manifest with SkipVerify=true, TddMode=true.
	cfg := skipVerifyManifest(true, true) // TDD=on, skipVerify=true

	tddEnabled := true
	st := state.StateFile{
		Phase: state.PhaseExecuting,
		OverrideTasks: []state.SpecTask{
			{ID: "task-1", Title: "first task", Completed: false, TDDEnabled: &tddEnabled},
			{ID: "task-2", Title: "second task", Completed: false, TDDEnabled: &tddEnabled},
		},
		Execution: state.ExecutionState{
			TDDCycle:  state.TDDCycleRed,
			Iteration: 1,
		},
	}

	// -----------------------------------------------------------------------
	// AC-2 / Step 1: RED phase — test-writer (executor) submits report.
	// verifierPayloadGuard must route to advanceWithoutVerification, not call
	// applyVerifierReport.  State must transition to GREEN automatically.
	// -----------------------------------------------------------------------
	redReport := map[string]interface{}{
		"completed": []interface{}{},
		"remaining": []interface{}{"task-1"},
	}

	// AC-6: no error.
	stAfterRed, err := verifierPayloadGuard(st, cfg, redReport)
	if err != nil {
		t.Fatalf("[RED] unexpected error: %v", err)
	}

	// AC-2: state is now GREEN.
	if stAfterRed.Execution.TDDCycle != state.TDDCycleGreen {
		t.Errorf("[RED] expected TDDCycle=%q after RED, got %q",
			state.TDDCycleGreen, stAfterRed.Execution.TDDCycle)
	}

	// AC-2: verifier was NOT called — LastVerification must be nil.
	if stAfterRed.Execution.LastVerification != nil {
		t.Error("[RED] LastVerification set but verifier should not have been called in RED phase")
	}

	// AC-2: VerifierRequired=false compiled for RED state.
	compiledRed := ctxpkg.Compile(ctxmodel.CompileInput{
		State:  st, // original RED state
		Config: cfg,
	})
	if compiledRed.ExecutionData == nil {
		t.Fatal("[RED] expected ExecutionData in compiled context, got nil")
	}
	if compiledRed.ExecutionData.VerifierRequired {
		t.Error("[RED] VerifierRequired=true for RED phase but expected false (skipVerify=true + phase=red)")
	}

	// -----------------------------------------------------------------------
	// AC-3 / Step 2: GREEN phase — verifier submits report with refactorNotes.
	// verifierPayloadGuard must call applyVerifierReport, which populates
	// PendingRefactorNotes and transitions to REFACTOR.
	// -----------------------------------------------------------------------
	greenVerifierReport := map[string]interface{}{
		"passed": true,
		"output": "all tests pass",
		"refactorNotes": []interface{}{
			map[string]interface{}{
				"file":       "internal/foo/bar.go",
				"suggestion": "extract helper",
				"rationale":  "DRY principle",
			},
		},
	}

	// AC-6: no error.
	stAfterGreen, err := verifierPayloadGuard(stAfterRed, cfg, greenVerifierReport)
	if err != nil {
		t.Fatalf("[GREEN] unexpected error: %v", err)
	}

	// AC-3: cycle transitions to REFACTOR.
	if stAfterGreen.Execution.TDDCycle != state.TDDCycleRefactor {
		t.Errorf("[GREEN] expected TDDCycle=%q after GREEN verifier, got %q",
			state.TDDCycleRefactor, stAfterGreen.Execution.TDDCycle)
	}

	// AC-3: PendingRefactorNotes populated (SkipVerify=true + TDD=on path in RecordTDDVerificationFull).
	if len(stAfterGreen.Execution.PendingRefactorNotes) == 0 {
		t.Error("[GREEN] PendingRefactorNotes is empty but verifier supplied refactorNotes")
	}

	// AC-3: VerifierRequired=true compiled for GREEN state.
	compiledGreen := ctxpkg.Compile(ctxmodel.CompileInput{
		State:  stAfterRed, // the GREEN state before verifier ran
		Config: cfg,
	})
	if compiledGreen.ExecutionData == nil {
		t.Fatal("[GREEN] expected ExecutionData in compiled context, got nil")
	}
	if !compiledGreen.ExecutionData.VerifierRequired {
		t.Error("[GREEN] VerifierRequired=false for GREEN phase but expected true (skipVerify=true + phase=green)")
	}

	// -----------------------------------------------------------------------
	// AC-4 / Step 3: REFACTOR phase — executor submits report with
	// refactorApplied=true.  verifierPayloadGuard routes to
	// advanceWithoutVerification, task-1 completes, next task RED seeded.
	// -----------------------------------------------------------------------
	refactorReport := map[string]interface{}{
		"completed":       []interface{}{"task-1"},
		"remaining":       []interface{}{"task-2"},
		"refactorApplied": true,
	}

	// AC-6: no error.
	stAfterRefactor, err := verifierPayloadGuard(stAfterGreen, cfg, refactorReport)
	if err != nil {
		t.Fatalf("[REFACTOR] unexpected error: %v", err)
	}

	// AC-4: task-1 completed.
	if !contains(stAfterRefactor.Execution.CompletedTasks, "task-1") {
		t.Errorf("[REFACTOR] expected task-1 in CompletedTasks, got: %v",
			stAfterRefactor.Execution.CompletedTasks)
	}

	// AC-4: verifier NOT called in REFACTOR — LastVerification from REFACTOR
	// step is still the one set during GREEN (not a new one).
	// The GREEN step set LastVerification; REFACTOR must not overwrite it via
	// applyVerifierReport with a fresh timestamp.  We check TDDCycle is RED
	// (next-task seed) rather than asserting nil, because GREEN already set it.

	// AC-5: next task's RED seeded.
	if stAfterRefactor.Execution.TDDCycle != state.TDDCycleRed {
		t.Errorf("[REFACTOR] expected TDDCycle=%q for next task after REFACTOR, got %q",
			state.TDDCycleRed, stAfterRefactor.Execution.TDDCycle)
	}

	// AC-4: VerifierRequired=false compiled for REFACTOR state.
	compiledRefactor := ctxpkg.Compile(ctxmodel.CompileInput{
		State:  stAfterGreen, // the REFACTOR state before executor ran
		Config: cfg,
	})
	if compiledRefactor.ExecutionData == nil {
		t.Fatal("[REFACTOR] expected ExecutionData in compiled context, got nil")
	}
	if compiledRefactor.ExecutionData.VerifierRequired {
		t.Error("[REFACTOR] VerifierRequired=true for REFACTOR phase but expected false (skipVerify=true + phase=refactor)")
	}
}

// ---------------------------------------------------------------------------
// Task-11: Negative — skipVerify=true + wrong phase → verifier payload rejected
// ---------------------------------------------------------------------------

// TestNegative_SkipVerify_VerifierPayload_Rejected is a table-driven test that
// covers six acceptance criteria for how verifierPayloadGuard behaves when
// skipVerify=true and a verifier-shaped payload is submitted.
//
//   AC-1  skipVerify=true + TDD=on  + phase=red      + verifier → error
//   AC-2  Error message contains "verifier report submitted but skipVerify=true"
//   AC-3  skipVerify=true + TDD=off + verifier        → error
//   AC-4  skipVerify=true + TDD=on  + phase=refactor  + verifier → error
//   AC-5  skipVerify=true + TDD=on  + phase=green     + verifier → accept (no guard error)
//   AC-6  skipVerify=false + any phase                + verifier → accept (existing path)
func TestNegative_SkipVerify_VerifierPayload_Rejected(t *testing.T) {
	type tableEntry struct {
		name      string
		tddMode   bool
		skipVerify bool
		phase     string
		expectErr string // empty means no error expected
	}

	tests := []tableEntry{
		{
			// AC-1 + AC-2: TDD=on, skipVerify=true, RED phase — verifier must be rejected.
			name:       "skip+tdd-on+red+verifier → error",
			tddMode:    true,
			skipVerify: true,
			phase:      state.TDDCycleRed,
			expectErr:  "verifier report submitted but skipVerify=true",
		},
		{
			// AC-3: TDD=off, skipVerify=true — verifier must be rejected (no phase).
			name:       "skip+tdd-off+verifier → error",
			tddMode:    false,
			skipVerify: true,
			phase:      "",
			expectErr:  "verifier report submitted but skipVerify=true",
		},
		{
			// AC-4: TDD=on, skipVerify=true, REFACTOR phase — verifier must be rejected.
			name:       "skip+tdd-on+refactor+verifier → error",
			tddMode:    true,
			skipVerify: true,
			phase:      state.TDDCycleRefactor,
			expectErr:  "verifier report submitted but skipVerify=true",
		},
		{
			// AC-5: TDD=on, skipVerify=true, GREEN phase — verifier is accepted; guard
			// must NOT produce the skipVerify error message.
			name:       "skip+tdd-on+green+verifier → accept",
			tddMode:    true,
			skipVerify: true,
			phase:      state.TDDCycleGreen,
			expectErr:  "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cfg := skipVerifyManifest(tc.tddMode, tc.skipVerify)
			st := baseExecutingState(tc.phase)
			report := verifierReport(true)

			_, err := verifierPayloadGuard(st, cfg, report)

			if tc.expectErr != "" {
				// Expect an error containing the guard message.
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.expectErr)
				}
				if !strings.Contains(err.Error(), tc.expectErr) {
					t.Errorf("error message %q does not contain expected substring %q",
						err.Error(), tc.expectErr)
				}
			} else {
				// Accept path: the skipVerify guard error must NOT appear.
				if err != nil && strings.Contains(err.Error(), "verifier report submitted but skipVerify=true") {
					t.Errorf("verifier report was incorrectly rejected by the skipVerify guard: %v", err)
				}
			}
		})
	}
}

// TestNegative_SkipVerifyFalse_IsVerifierSkipped_ReturnsFalse covers AC-6:
// when skipVerify=false the manifest's IsVerifierSkipped() must return false,
// which is the condition checked by handleExecutingAnswer before invoking
// verifierPayloadGuard. If IsVerifierSkipped()=false, the guard is never
// reached and verifier reports follow the normal (non-rejecting) path.
func TestNegative_SkipVerifyFalse_IsVerifierSkipped_ReturnsFalse(t *testing.T) {
	tests := []struct {
		name    string
		tddMode bool
		phase   string
	}{
		{"no-skip+tdd-on+red → guard bypassed", true, state.TDDCycleRed},
		{"no-skip+tdd-on+green → guard bypassed", true, state.TDDCycleGreen},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cfg := skipVerifyManifest(tc.tddMode, false /*skipVerify=false*/)

			// IsVerifierSkipped()=false is the invariant that ensures handleExecutingAnswer
			// does NOT call verifierPayloadGuard, so verifier reports are accepted.
			if cfg.IsVerifierSkipped() {
				t.Errorf("expected IsVerifierSkipped()=false for skipVerify=false manifest in phase %q, got true", tc.phase)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Shared utility
// ---------------------------------------------------------------------------

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
