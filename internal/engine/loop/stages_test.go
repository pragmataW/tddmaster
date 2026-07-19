package loop

import (
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func makeSettings(tdd, skipVerify, gate bool) spec.Settings {
	return spec.Settings{
		TDDEnabled:               tdd,
		SkipVerifierEnabled:      skipVerify,
		ImportantTaskGateEnabled: gate,
	}
}

func makeImportantTask(id string, tddEnabled bool) spec.Task {
	return spec.Task{
		ID:         id,
		Title:      id,
		TDDEnabled: tddEnabled,
		Important:  true,
	}
}

func makeExecCtx(settings spec.Settings, task spec.Task, st spec.ExecState, taskIdx, maxRefactor int) ExecCtx {
	return ExecCtx{
		Settings:          settings,
		Task:              task,
		State:             st,
		TaskIdx:           taskIdx,
		MaxRefactorRounds: maxRefactor,
	}
}

func isPlanApproved(st spec.ExecState) bool {
	return st.PlanApproved
}

func stateWithApprovedPlan() spec.ExecState {
	return spec.ExecState{
		PlanApproved: true,
	}
}

func TestStageIDs_AllStagesHaveUniqueIDs(t *testing.T) {
	stages := allStages()
	seen := map[string]bool{}
	for _, s := range stages {
		id := s.ID()
		if seen[id] {
			t.Errorf("duplicate stage ID: %q", id)
		}
		seen[id] = true
	}
}

func TestStageIDs_ExpectedValues(t *testing.T) {
	stages := allStages()
	ids := make([]string, len(stages))
	for i, s := range stages {
		ids[i] = s.ID()
	}
	expected := []string{
		StageIDGate,
		StageIDRed,
		StageIDGreen,
		StageIDRefactor,
		StageIDExecutor,
		StageIDVerifier,
	}
	if len(ids) != len(expected) {
		t.Fatalf("stage count: got %d, want %d", len(ids), len(expected))
	}
	for i, want := range expected {
		if ids[i] != want {
			t.Errorf("stage[%d].ID(): got %q, want %q", i, ids[i], want)
		}
	}
}

func TestGateStage_Applies_WhenGateEnabledAndImportantAndNotApproved(t *testing.T) {
	settings := makeSettings(false, false, true)
	task := makeImportantTask("t-1", false)
	st := makeExecState("")
	ctx := makeExecCtx(settings, task, st, 0, 3)

	if !gateStage().Applies(ctx) {
		t.Error("gate Applies: got false, want true (gate enabled + important + unapproved)")
	}
}

func TestGateStage_NotApplies_WhenGateDisabled(t *testing.T) {
	settings := makeSettings(false, false, false)
	task := makeImportantTask("t-1", false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)

	if gateStage().Applies(ctx) {
		t.Error("gate Applies: got true, want false (gate disabled)")
	}
}

func TestGateStage_NotApplies_WhenTaskNotImportant(t *testing.T) {
	settings := makeSettings(false, false, true)
	task := makeTask("t-1", false, false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)

	if gateStage().Applies(ctx) {
		t.Error("gate Applies: got true, want false (task not important)")
	}
}

func TestGateStage_NotApplies_WhenPlanAlreadyApproved(t *testing.T) {
	settings := makeSettings(false, false, true)
	task := makeImportantTask("t-1", false)
	st := stateWithApprovedPlan()
	ctx := makeExecCtx(settings, task, st, 0, 3)

	if gateStage().Applies(ctx) {
		t.Error("gate Applies: got true, want false (plan already approved)")
	}
}

func TestRedStage_Applies_WhenTDDEnabledAndCycleRed(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("red"), 0, 3)

	if !redStage().Applies(ctx) {
		t.Error("red Applies: got false, want true")
	}
}

func TestRedStage_NotApplies_WhenTDDDisabled(t *testing.T) {
	settings := makeSettings(false, false, false)
	task := makeTask("t-1", false, false)
	ctx := makeExecCtx(settings, task, makeExecState("red"), 0, 3)

	if redStage().Applies(ctx) {
		t.Error("red Applies: got true, want false (TDD disabled)")
	}
}

func TestRedStage_NotApplies_WhenCycleNotRed(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("green"), 0, 3)

	if redStage().Applies(ctx) {
		t.Error("red Applies: got true, want false (cycle is green)")
	}
}

func TestGreenStage_Applies_WhenTDDEnabledAndCycleGreen(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("green"), 0, 3)

	if !greenStage().Applies(ctx) {
		t.Error("green Applies: got false, want true")
	}
}

func TestGreenStage_NotApplies_WhenCycleRed(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("red"), 0, 3)

	if greenStage().Applies(ctx) {
		t.Error("green Applies: got true, want false (cycle is red)")
	}
}

func TestRefactorStage_Applies_WhenTDDEnabledAndCycleRefactor(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("refactor"), 0, 3)

	if !refactorStage().Applies(ctx) {
		t.Error("refactor Applies: got false, want true")
	}
}

func TestRefactorStage_NotApplies_WhenCycleGreen(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("green"), 0, 3)

	if refactorStage().Applies(ctx) {
		t.Error("refactor Applies: got true, want false (cycle is green)")
	}
}

func TestExecutorStage_Applies_WhenTDDDisabled(t *testing.T) {
	settings := makeSettings(false, false, false)
	task := makeTask("t-1", false, false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)

	if !executorStage().Applies(ctx) {
		t.Error("executor Applies: got false, want true (TDD off)")
	}
}

func TestExecutorStage_Applies_WhenSkipVerifyOn_EvenIfImplemented(t *testing.T) {
	settings := makeSettings(false, true, false)
	task := makeTask("t-1", false, false)
	st := makeExecState("")
	st.Implemented = true
	ctx := makeExecCtx(settings, task, st, 0, 3)

	if !executorStage().Applies(ctx) {
		t.Error("executor Applies: got false, want true (skipVerify on → executor always runs, no verifier deadlock)")
	}
}

func TestExecutorStage_NotApplies_WhenTDDEnabled(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)

	if executorStage().Applies(ctx) {
		t.Error("executor Applies: got true, want false (TDD on)")
	}
}

func implementedGreen() spec.ExecState {
	st := makeExecState("green")
	st.Implemented = true
	return st
}

func TestVerifierStage_Applies_WhenTDDGreenImplemented_SkipOff(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, implementedGreen(), 0, 3)

	if !verifierStage().Applies(ctx) {
		t.Error("verifier Applies: got false, want true (TDD green + implemented, skipVerify off)")
	}
}

func TestVerifierStage_Applies_WhenTDDGreenImplemented_SkipOn(t *testing.T) {
	settings := makeSettings(true, true, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, implementedGreen(), 0, 3)

	if !verifierStage().Applies(ctx) {
		t.Error("verifier Applies: got false, want true (green ALWAYS verifies, even skipVerify on)")
	}
}

func TestVerifierStage_NotApplies_WhenGreenNotYetImplemented(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("green"), 0, 3)

	if verifierStage().Applies(ctx) {
		t.Error("verifier Applies: got true, want false (green not yet implemented → executor runs first)")
	}
}

func TestVerifierStage_Applies_WhenNonTDDImplemented_SkipOff(t *testing.T) {
	settings := makeSettings(false, false, false)
	task := makeTask("t-1", false, false)
	st := makeExecState("")
	st.Implemented = true
	ctx := makeExecCtx(settings, task, st, 0, 3)

	if !verifierStage().Applies(ctx) {
		t.Error("verifier Applies: got false, want true (non-TDD implemented + skipVerify off → verifier stage runs)")
	}
}

func TestVerifierStage_NotApplies_WhenNonTDDImplemented_SkipOn(t *testing.T) {
	settings := makeSettings(false, true, false)
	task := makeTask("t-1", false, false)
	st := makeExecState("")
	st.Implemented = true
	ctx := makeExecCtx(settings, task, st, 0, 3)

	if verifierStage().Applies(ctx) {
		t.Error("verifier Applies: got true, want false (non-TDD + skipVerify on → no verifier stage)")
	}
}

func TestVerifierStage_NotApplies_WhenNonTDDNotYetImplemented(t *testing.T) {
	settings := makeSettings(false, false, false)
	task := makeTask("t-1", false, false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)

	if verifierStage().Applies(ctx) {
		t.Error("verifier Applies: got true, want false (non-TDD not yet implemented → executor runs first)")
	}
}

func TestVerifierStage_NotApplies_WhenCycleRefactor(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	st := makeExecState("refactor")
	st.Implemented = true
	ctx := makeExecCtx(settings, task, st, 0, 3)

	if verifierStage().Applies(ctx) {
		t.Error("verifier Applies: got true, want false (refactor verify is handled by refactorStage)")
	}
}

func TestVerifierStage_NotApplies_WhenCycleRed(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	st := makeExecState("red")
	st.Implemented = true
	ctx := makeExecCtx(settings, task, st, 0, 3)

	if verifierStage().Applies(ctx) {
		t.Error("verifier Applies: got true, want false (red cycle → test-writer)")
	}
}

func TestRedStage_Prompt_ReturnsInstructAction(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("red"), 0, 3)

	action := redStage().Prompt(ctx)

	if action.Action != engine.ActionInstruct {
		t.Errorf("Prompt ActionType: got %q, want %q", action.Action, engine.ActionInstruct)
	}
	if action.DelegateAgent == "" {
		t.Error("Prompt DelegateAgent: got empty, want test-writer agent")
	}
}

func TestGreenStage_Prompt_ReturnsInstructAction(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("green"), 0, 3)

	action := greenStage().Prompt(ctx)

	if action.Action != engine.ActionInstruct {
		t.Errorf("Prompt ActionType: got %q, want %q", action.Action, engine.ActionInstruct)
	}
	if action.DelegateAgent == "" {
		t.Error("Prompt DelegateAgent: got empty, want executor agent")
	}
}

func TestRefactorStage_Prompt_ReturnsInstructAction(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("refactor"), 0, 3)

	action := refactorStage().Prompt(ctx)

	if action.Action != engine.ActionInstruct {
		t.Errorf("Prompt ActionType: got %q, want %q", action.Action, engine.ActionInstruct)
	}
}

func TestExecutorStage_Prompt_ReturnsInstructAction(t *testing.T) {
	settings := makeSettings(false, false, false)
	task := makeTask("t-1", false, false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)

	action := executorStage().Prompt(ctx)

	if action.Action != engine.ActionInstruct {
		t.Errorf("Prompt ActionType: got %q, want %q", action.Action, engine.ActionInstruct)
	}
	if action.DelegateAgent == "" {
		t.Error("Prompt DelegateAgent: got empty, want executor agent")
	}
}

func TestVerifierStage_Prompt_ReturnsAskAction(t *testing.T) {
	settings := makeSettings(false, false, false)
	task := makeTask("t-1", false, false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)

	action := verifierStage().Prompt(ctx)

	if action.Action != engine.ActionAsk && action.Action != engine.ActionInstruct {
		t.Errorf("Prompt ActionType: got %q, want ask or instruct", action.Action)
	}
	if action.DelegateAgent == "" {
		t.Error("Prompt DelegateAgent: got empty, want verifier agent")
	}
}

func TestGateStage_Prompt_ReturnsDelegateToPlannerAgent(t *testing.T) {
	settings := makeSettings(false, false, true)
	task := makeImportantTask("t-1", false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)

	action := gateStage().Prompt(ctx)

	if action.DelegateAgent == "" {
		t.Error("gate Prompt DelegateAgent: got empty, want tddmaster-planner")
	}
}

func TestRedStage_OnReport_AdvancesToGreen(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("red"), 0, 3)

	report := StageReport{Passed: true, TestsWritten: []string{"t1_test.go"}}
	newCtx, err := redStage().OnReport(ctx, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newCtx.State.TDDCycle != cycleGreen {
		t.Errorf("TDDCycle: got %q, want %q", newCtx.State.TDDCycle, cycleGreen)
	}
}

func TestGreenStage_OnReport_SetsImplemented_DoesNotAdvance(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("green"), 0, 3)

	report := StageReport{Completed: []string{"t-1"}}
	newCtx, err := greenStage().OnReport(ctx, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !newCtx.State.Implemented {
		t.Error("Implemented: got false, want true after executor implements")
	}
	if newCtx.State.TDDCycle != cycleGreen {
		t.Errorf("TDDCycle: got %q, want green (advance deferred to verifier stage)", newCtx.State.TDDCycle)
	}
}

func TestVerifierStage_OnReport_NotesPresent_AdvancesToRefactor(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, implementedGreen(), 0, 3)

	report := StageReport{Passed: true, RefactorNotes: []RefactorNote{{Suggestion: "refactor notes"}}}
	newCtx, err := verifierStage().OnReport(ctx, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newCtx.State.TDDCycle != cycleRefactor {
		t.Errorf("TDDCycle: got %q, want %q", newCtx.State.TDDCycle, cycleRefactor)
	}
	if newCtx.State.Implemented {
		t.Error("Implemented: got true, want false (cleared on advance)")
	}
}

func TestVerifierStage_OnReport_NoNotes_TaskComplete(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, implementedGreen(), 0, 3)

	report := StageReport{Passed: true, RefactorNotes: nil}
	newCtx, err := verifierStage().OnReport(ctx, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newCtx.State.TDDCycle != cycleEmpty {
		t.Errorf("TDDCycle: got %q, want empty (task complete)", newCtx.State.TDDCycle)
	}
}

func TestVerifierStage_OnReport_Failed_StaysGreen_ClearsImplemented(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, implementedGreen(), 0, 3)

	report := StageReport{Passed: false, FailedACs: []string{"ac-1"}}
	newCtx, err := verifierStage().OnReport(ctx, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newCtx.State.TDDCycle != cycleGreen {
		t.Errorf("TDDCycle: got %q, want green (verifier failed → re-implement)", newCtx.State.TDDCycle)
	}
	if newCtx.State.Implemented {
		t.Error("Implemented: got true, want false (re-run executor on failed verify)")
	}
}

func TestRefactorStage_OnReport_IncrementsRounds(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	st := makeExecState("refactor")
	st.RefactorApplied = true
	st.RefactorRounds = 1
	ctx := makeExecCtx(settings, task, st, 0, 5)

	report := StageReport{Passed: true, RefactorNotes: []RefactorNote{{Suggestion: "refactor notes"}}}
	newCtx, err := refactorStage().OnReport(ctx, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newCtx.State.RefactorRounds != 2 {
		t.Errorf("RefactorRounds: got %d, want 2", newCtx.State.RefactorRounds)
	}
}

func TestRefactorStage_OnReport_CapReached_TaskComplete(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	st := makeExecState("refactor")
	st.RefactorApplied = true
	st.RefactorRounds = 2
	ctx := makeExecCtx(settings, task, st, 0, 3)

	report := StageReport{Passed: true, RefactorNotes: []RefactorNote{{Suggestion: "refactor notes"}}}
	newCtx, err := refactorStage().OnReport(ctx, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newCtx.State.TDDCycle != cycleEmpty {
		t.Errorf("TDDCycle: got %q, want empty (cap reached)", newCtx.State.TDDCycle)
	}
}

func TestRefactorStage_OnReport_ApplyHead_SetsAppliedAndWaits(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	st := makeExecState("refactor")
	st.RefactorApplied = false
	st.RefactorRounds = 0
	ctx := makeExecCtx(settings, task, st, 0, 3)

	report := StageReport{Passed: true, RefactorNotes: []RefactorNote{{Suggestion: "refactor notes"}}}
	newCtx, err := refactorStage().OnReport(ctx, report)

	if err != nil {
		t.Fatalf("apply head must not error (no bypass): %v", err)
	}
	if !newCtx.State.RefactorApplied {
		t.Error("RefactorApplied: got false, want true after apply head")
	}
	if newCtx.State.TDDCycle != cycleRefactor {
		t.Errorf("TDDCycle: got %q, want refactor (waiting for verifier)", newCtx.State.TDDCycle)
	}
	if newCtx.State.RefactorRounds != 0 {
		t.Errorf("RefactorRounds: got %d, want 0 (not advanced until verify)", newCtx.State.RefactorRounds)
	}
}

func TestRefactorStage_OnReport_SkipVerifier_ApplyAdvancesDirectly(t *testing.T) {
	settings := makeSettings(true, true, false)
	task := makeTask("t-1", true, false)
	st := makeExecState("refactor")
	st.RefactorApplied = false
	ctx := makeExecCtx(settings, task, st, 0, 3)

	report := StageReport{Passed: true, RefactorNotes: nil}
	newCtx, err := refactorStage().OnReport(ctx, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newCtx.State.TDDCycle != cycleEmpty {
		t.Errorf("TDDCycle: got %q, want empty (skipVerifier → apply completes refactor)", newCtx.State.TDDCycle)
	}
}

func TestRedStage_OnReport_Failed_StaysRed(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("red"), 0, 3)

	report := StageReport{Passed: false}
	newCtx, err := redStage().OnReport(ctx, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newCtx.State.TDDCycle != cycleRed {
		t.Errorf("TDDCycle: got %q, want %q (failed → stay red)", newCtx.State.TDDCycle, cycleRed)
	}
}

func TestGreenStage_OnReport_Failed_StaysGreen(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("green"), 0, 3)

	report := StageReport{Passed: false}
	newCtx, err := greenStage().OnReport(ctx, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newCtx.State.TDDCycle != cycleGreen {
		t.Errorf("TDDCycle: got %q, want green (failed → stay green)", newCtx.State.TDDCycle)
	}
}

func TestRefactorStage_OnReport_NoNotes_TaskComplete(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTask("t-1", true, false)
	st := makeExecState("refactor")
	st.RefactorApplied = true
	ctx := makeExecCtx(settings, task, st, 0, 3)

	report := StageReport{Passed: true, RefactorNotes: nil}
	newCtx, err := refactorStage().OnReport(ctx, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newCtx.State.TDDCycle != cycleEmpty {
		t.Errorf("TDDCycle: got %q, want empty (no notes → done)", newCtx.State.TDDCycle)
	}
}

func TestPerTaskTDDEnabled_TDDTask_UsesRedStage(t *testing.T) {
	settings := makeSettings(true, false, false)
	tddTask := makeTask("t-tdd", true, false)
	ctx := makeExecCtx(settings, tddTask, makeExecState("red"), 0, 3)

	if !redStage().Applies(ctx) {
		t.Error("red stage should apply for TDD-enabled task in red cycle")
	}
	if executorStage().Applies(ctx) {
		t.Error("executor stage should not apply for TDD-enabled task")
	}
}

func TestPerTaskTDDEnabled_NonTDDTask_UsesExecutorStage(t *testing.T) {
	settings := makeSettings(true, false, false)
	noTDDTask := makeTask("t-notdd", false, false)
	ctx := makeExecCtx(settings, noTDDTask, makeExecState(""), 0, 3)

	if !executorStage().Applies(ctx) {
		t.Error("executor stage should apply for TDD-disabled task")
	}
	if redStage().Applies(ctx) {
		t.Error("red stage should not apply for TDD-disabled task")
	}
}

func TestPerTaskTDDEnabled_MixedTasks_IndependentFlows(t *testing.T) {
	settings := makeSettings(true, false, false)

	tddTask := makeTask("t-1", true, false)
	ctxTDD := makeExecCtx(settings, tddTask, makeExecState("red"), 0, 3)
	if !redStage().Applies(ctxTDD) {
		t.Error("TDD task: red stage should apply")
	}

	noTDDTask := makeTask("t-2", false, false)
	ctxNoTDD := makeExecCtx(settings, noTDDTask, makeExecState(""), 0, 3)
	if !executorStage().Applies(ctxNoTDD) {
		t.Error("non-TDD task: executor stage should apply")
	}
}

func TestSpecLevelTDDDisabled_TaskEnabled_NoRedStage(t *testing.T) {
	settings := makeSettings(false, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("red"), 0, 3)

	if redStage().Applies(ctx) {
		t.Error("red stage must not apply when spec-level TDD disabled even if task enabled")
	}
	if !executorStage().Applies(ctx) {
		t.Error("executor stage must apply when spec-level TDD disabled")
	}
}

func TestSpecLevelTDDDisabled_TaskEnabled_NoGreenStage(t *testing.T) {
	settings := makeSettings(false, false, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, makeExecState("green"), 0, 3)

	if greenStage().Applies(ctx) {
		t.Error("green stage must not apply when spec-level TDD disabled")
	}
}

func TestSpecLevelTDDDisabled_TaskEnabled_NoTDDVerifierStage_SkipOn(t *testing.T) {
	settings := makeSettings(false, true, false)
	task := makeTask("t-1", true, false)
	ctx := makeExecCtx(settings, task, implementedGreen(), 0, 3)

	if verifierStage().Applies(ctx) {
		t.Error("verifier stage must not apply when spec-level TDD disabled and skipVerify on")
	}
}

func makeTaskWithACAndEC(id string, tddEnabled bool) spec.Task {
	return spec.Task{
		ID:         id,
		Title:      id,
		TDDEnabled: tddEnabled,
		Criteria:   []spec.Criterion{{ID: "ac-1", Then: "ac-1: foo"}, {ID: "ac-2", Then: "ac-2: bar"}},
		EdgeCases:  []string{"ec-1: x", "ec-2: y"},
	}
}

func instructionText(key promptregistry.InstructionKey) string {
	text, _ := promptregistry.Instruction(key)
	return text
}

func TestRedStage_Prompt_Contract(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTaskWithACAndEC("task-1", true)
	ctx := makeExecCtx(settings, task, makeExecState("red"), 0, 3)

	action := redStage().Prompt(ctx)

	if action.DelegateAgent != string(promptregistry.AgentTestWriter) {
		t.Errorf("DelegateAgent: got %q, want %q", action.DelegateAgent, string(promptregistry.AgentTestWriter))
	}
	if action.ExpectedInput.Format != engine.FormatJSON {
		t.Errorf("ExpectedInput.Format: got %q, want %q", action.ExpectedInput.Format, engine.FormatJSON)
	}
	if action.ExpectedInput.Example != promptregistry.ReportExampleTestWriter {
		t.Errorf("ExpectedInput.Example: got %q, want ReportExampleTestWriter", action.ExpectedInput.Example)
	}
	keyText := instructionText(promptregistry.KeyExecRed)
	if !strings.Contains(action.Instruction, keyText) {
		t.Errorf("Instruction missing KeyExecRed text")
	}
	if !strings.Contains(action.Instruction, "ac-1: foo") {
		t.Error("Instruction missing AC item 'ac-1: foo'")
	}
	if !strings.Contains(action.Instruction, "ac-2: bar") {
		t.Error("Instruction missing AC item 'ac-2: bar'")
	}
	if !strings.Contains(action.Instruction, "ec-1: x") {
		t.Error("Instruction missing EdgeCase item 'ec-1: x'")
	}
	if !strings.Contains(action.Instruction, "ec-2: y") {
		t.Error("Instruction missing EdgeCase item 'ec-2: y'")
	}
}

func TestRedStage_Prompt_WithUserContext(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTaskWithACAndEC("task-1", true)
	ctx := makeExecCtx(settings, task, makeExecState("red"), 0, 3)
	ctx.UserContext = "kullanici baglami"

	action := redStage().Prompt(ctx)

	if !strings.Contains(action.Instruction, "kullanici baglami") {
		t.Error("Instruction missing UserContext 'kullanici baglami'")
	}
}

func TestGreenStage_Prompt_Contract(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTaskWithACAndEC("task-1", true)
	ctx := makeExecCtx(settings, task, makeExecState("green"), 0, 3)

	action := greenStage().Prompt(ctx)

	if action.DelegateAgent != string(promptregistry.AgentExecutor) {
		t.Errorf("DelegateAgent: got %q, want %q", action.DelegateAgent, string(promptregistry.AgentExecutor))
	}
	if action.ExpectedInput.Format != engine.FormatJSON {
		t.Errorf("ExpectedInput.Format: got %q, want %q", action.ExpectedInput.Format, engine.FormatJSON)
	}
	if action.ExpectedInput.Example != promptregistry.ReportExampleExecutor {
		t.Errorf("ExpectedInput.Example: got %q, want ReportExampleExecutor", action.ExpectedInput.Example)
	}
	keyText := instructionText(promptregistry.KeyExecGreen)
	if !strings.Contains(action.Instruction, keyText) {
		t.Errorf("Instruction missing KeyExecGreen text")
	}
	if !strings.Contains(strings.ToLower(action.Instruction), "verifier") {
		t.Error("Instruction must mention 'verifier' (mandatory verifier requirement)")
	}
	if !strings.Contains(action.Instruction, "ac-1: foo") {
		t.Error("Instruction missing AC item 'ac-1: foo'")
	}
	if !strings.Contains(action.Instruction, "ec-1: x") {
		t.Error("Instruction missing EdgeCase item 'ec-1: x'")
	}
}

func TestGreenStage_Prompt_WithUserContext(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTaskWithACAndEC("task-1", true)
	ctx := makeExecCtx(settings, task, makeExecState("green"), 0, 3)
	ctx.UserContext = "kullanici baglami"

	action := greenStage().Prompt(ctx)

	if !strings.Contains(action.Instruction, "kullanici baglami") {
		t.Error("Instruction missing UserContext 'kullanici baglami'")
	}
}

func TestGreenStage_Prompt_WithApprovedPlan(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTaskWithACAndEC("task-1", true)
	st := makeExecState("green")
	st.Plan = &spec.TaskPlan{
		Approach:     "yaklasim X",
		TouchedFiles: []string{"a.go", "b.go"},
	}
	ctx := makeExecCtx(settings, task, st, 0, 3)

	action := greenStage().Prompt(ctx)

	if !strings.Contains(action.Instruction, "yaklasim X") {
		t.Error("Instruction missing approved plan Approach 'yaklasim X'")
	}
	if !strings.Contains(action.Instruction, "a.go") {
		t.Error("Instruction missing TouchedFile 'a.go'")
	}
	if !strings.Contains(action.Instruction, "b.go") {
		t.Error("Instruction missing TouchedFile 'b.go'")
	}
}

func TestGreenStage_Prompt_WithFailedACs(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTaskWithACAndEC("task-1", true)
	st := makeExecState("green")
	st.LastFailedACs = []string{"ac-1: foo"}
	st.LastUncoveredEC = []string{"ec-2: y"}
	ctx := makeExecCtx(settings, task, st, 0, 3)

	action := greenStage().Prompt(ctx)

	failedText := instructionText(promptregistry.KeyExecVerifyFailed)
	if !strings.Contains(action.Instruction, failedText) {
		t.Errorf("Instruction missing KeyExecVerifyFailed text")
	}
	if !strings.Contains(action.Instruction, "ac-1: foo") {
		t.Error("Instruction missing failed AC 'ac-1: foo'")
	}
	if !strings.Contains(action.Instruction, "ec-2: y") {
		t.Error("Instruction missing uncovered EC 'ec-2: y'")
	}
}

func TestRefactorStage_Prompt_BeforeApply(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTaskWithACAndEC("task-1", true)
	st := makeExecState("refactor")
	st.RefactorApplied = false
	ctx := makeExecCtx(settings, task, st, 0, 3)

	action := refactorStage().Prompt(ctx)

	if action.Action != engine.ActionInstruct {
		t.Errorf("Action: got %q, want %q", action.Action, engine.ActionInstruct)
	}
	if action.DelegateAgent != string(promptregistry.AgentExecutor) {
		t.Errorf("DelegateAgent: got %q, want %q", action.DelegateAgent, string(promptregistry.AgentExecutor))
	}
	if action.ExpectedInput.Format != engine.FormatJSON {
		t.Errorf("ExpectedInput.Format: got %q, want %q", action.ExpectedInput.Format, engine.FormatJSON)
	}
	if action.ExpectedInput.Example != promptregistry.ReportExampleRefactorApply {
		t.Errorf("ExpectedInput.Example: got %q, want ReportExampleRefactorApply", action.ExpectedInput.Example)
	}
	keyText := instructionText(promptregistry.KeyExecRefactorApply)
	if !strings.Contains(action.Instruction, keyText) {
		t.Errorf("Instruction missing KeyExecRefactorApply text")
	}
}

func TestRefactorStage_Prompt_AfterApply(t *testing.T) {
	settings := makeSettings(true, false, false)
	task := makeTaskWithACAndEC("task-1", true)
	st := makeExecState("refactor")
	st.RefactorApplied = true
	ctx := makeExecCtx(settings, task, st, 0, 3)

	action := refactorStage().Prompt(ctx)

	if action.DelegateAgent != string(promptregistry.AgentVerifier) {
		t.Errorf("DelegateAgent: got %q, want %q", action.DelegateAgent, string(promptregistry.AgentVerifier))
	}
	if action.ExpectedInput.Format != engine.FormatJSON {
		t.Errorf("ExpectedInput.Format: got %q, want %q", action.ExpectedInput.Format, engine.FormatJSON)
	}
	if action.ExpectedInput.Example != promptregistry.ReportExampleVerifier {
		t.Errorf("ExpectedInput.Example: got %q, want ReportExampleVerifier", action.ExpectedInput.Example)
	}
	keyText := instructionText(promptregistry.KeyExecRefactor)
	if !strings.Contains(action.Instruction, keyText) {
		t.Errorf("Instruction missing KeyExecRefactor text")
	}
}

func TestExecutorStage_Prompt_Contract(t *testing.T) {
	settings := makeSettings(false, false, false)
	task := makeTaskWithACAndEC("task-1", false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)

	action := executorStage().Prompt(ctx)

	if action.DelegateAgent != string(promptregistry.AgentExecutor) {
		t.Errorf("DelegateAgent: got %q, want %q", action.DelegateAgent, string(promptregistry.AgentExecutor))
	}
	if action.ExpectedInput.Format != engine.FormatJSON {
		t.Errorf("ExpectedInput.Format: got %q, want %q", action.ExpectedInput.Format, engine.FormatJSON)
	}
	if action.ExpectedInput.Example != promptregistry.ReportExampleExecutor {
		t.Errorf("ExpectedInput.Example: got %q, want ReportExampleExecutor", action.ExpectedInput.Example)
	}
	keyText := instructionText(promptregistry.KeyExecExecutor)
	if !strings.Contains(action.Instruction, keyText) {
		t.Errorf("Instruction missing KeyExecExecutor text")
	}
	if !strings.Contains(action.Instruction, "ac-1: foo") {
		t.Error("Instruction missing AC item 'ac-1: foo'")
	}
	if !strings.Contains(action.Instruction, "ec-1: x") {
		t.Error("Instruction missing EdgeCase item 'ec-1: x'")
	}
}

func TestExecutorStage_Prompt_WithUserContext(t *testing.T) {
	settings := makeSettings(false, false, false)
	task := makeTaskWithACAndEC("task-1", false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)
	ctx.UserContext = "kullanici baglami"

	action := executorStage().Prompt(ctx)

	if !strings.Contains(action.Instruction, "kullanici baglami") {
		t.Error("Instruction missing UserContext 'kullanici baglami'")
	}
}

func TestExecutorStage_Prompt_WithApprovedPlan(t *testing.T) {
	settings := makeSettings(false, false, false)
	task := makeTaskWithACAndEC("task-1", false)
	st := makeExecState("")
	st.Plan = &spec.TaskPlan{
		Approach:     "yaklasim X",
		TouchedFiles: []string{"a.go", "b.go"},
	}
	ctx := makeExecCtx(settings, task, st, 0, 3)

	action := executorStage().Prompt(ctx)

	if !strings.Contains(action.Instruction, "yaklasim X") {
		t.Error("Instruction missing approved plan Approach 'yaklasim X'")
	}
	if !strings.Contains(action.Instruction, "a.go") {
		t.Error("Instruction missing TouchedFile 'a.go'")
	}
	if !strings.Contains(action.Instruction, "b.go") {
		t.Error("Instruction missing TouchedFile 'b.go'")
	}
}

func TestVerifierStage_Prompt_Contract(t *testing.T) {
	settings := makeSettings(false, false, false)
	task := makeTaskWithACAndEC("task-1", false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)

	action := verifierStage().Prompt(ctx)

	if action.DelegateAgent != string(promptregistry.AgentVerifier) {
		t.Errorf("DelegateAgent: got %q, want %q", action.DelegateAgent, string(promptregistry.AgentVerifier))
	}
	if action.ExpectedInput.Format != engine.FormatJSON {
		t.Errorf("ExpectedInput.Format: got %q, want %q", action.ExpectedInput.Format, engine.FormatJSON)
	}
	if action.ExpectedInput.Example != promptregistry.ReportExampleVerifier {
		t.Errorf("ExpectedInput.Example: got %q, want ReportExampleVerifier", action.ExpectedInput.Example)
	}
	keyText := instructionText(promptregistry.KeyExecVerifier)
	if !strings.Contains(action.Instruction, keyText) {
		t.Errorf("Instruction missing KeyExecVerifier text")
	}
	if !strings.Contains(action.Instruction, "ac-1: foo") {
		t.Error("Instruction missing AC item 'ac-1: foo'")
	}
	if !strings.Contains(action.Instruction, "ec-1: x") {
		t.Error("Instruction missing EdgeCase item 'ec-1: x'")
	}
}

func TestGateStage_Prompt_Contract(t *testing.T) {
	settings := makeSettings(false, false, true)
	task := makeImportantTask("task-1", false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)

	action := gateStage().Prompt(ctx)

	if action.Action != engine.ActionAsk {
		t.Errorf("Action: got %q, want %q", action.Action, engine.ActionAsk)
	}
	if action.DelegateAgent != string(promptregistry.AgentPlanner) {
		t.Errorf("DelegateAgent: got %q, want %q", action.DelegateAgent, string(promptregistry.AgentPlanner))
	}
	if action.ExpectedInput.Example != promptregistry.ReportExamplePlanner {
		t.Errorf("ExpectedInput.Example: got %q, want ReportExamplePlanner", action.ExpectedInput.Example)
	}
	keyText := instructionText(promptregistry.KeyExecGate)
	if !strings.Contains(action.Instruction, keyText) {
		t.Errorf("Instruction missing KeyExecGate text")
	}
}

func TestGateStage_Prompt_WithPriorFeedback(t *testing.T) {
	settings := makeSettings(false, false, true)
	task := makeImportantTask("task-1", false)
	st := makeExecState("")
	st.PlanFeedback = "dosya disina cikma"
	st.PlanAttempts = 2
	ctx := makeExecCtx(settings, task, st, 0, 3)

	action := gateStage().Prompt(ctx)

	if !strings.Contains(action.Instruction, "dosya disina cikma") {
		t.Error("Instruction missing priorFeedback 'dosya disina cikma'")
	}
	if !strings.Contains(action.Instruction, "2") {
		t.Error("Instruction missing attemptCount '2'")
	}
}

func TestGateOnReport_Accept_PersistsPlanAndApproves(t *testing.T) {
	settings := makeSettings(false, false, true)
	task := makeImportantTask("task-1", false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)

	plan := &spec.TaskPlan{
		TaskID:       "task-1",
		Approach:     "x",
		TouchedFiles: []string{"a.go"},
	}
	report := StageReport{Accepted: true, Plan: plan}
	newCtx, err := gateStage().OnReport(ctx, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored := newCtx.State.Plan
	if stored == nil {
		t.Fatal("Plan is nil after accept")
	}
	if stored.Approach != "x" {
		t.Errorf("TaskPlan.Approach: got %q, want %q", stored.Approach, "x")
	}
	found := false
	for _, f := range stored.TouchedFiles {
		if f == "a.go" {
			found = true
		}
	}
	if !found {
		t.Error("TaskPlan.TouchedFiles missing 'a.go'")
	}
	if !isPlanApproved(newCtx.State) {
		t.Error("PlanApproved must be true after accept")
	}
}

func TestGateOnReport_Revise_WritesFeedbackAndIncrementsAttempts(t *testing.T) {
	settings := makeSettings(false, false, true)
	task := makeImportantTask("task-1", false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)

	report := StageReport{Accepted: false, PlanFeedback: "dosya disina cikma"}
	newCtx, err := gateStage().OnReport(ctx, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newCtx.State.PlanFeedback != "dosya disina cikma" {
		t.Errorf("PlanFeedback: got %q, want 'dosya disina cikma'", newCtx.State.PlanFeedback)
	}
	if newCtx.State.PlanAttempts != 1 {
		t.Errorf("PlanAttempts: got %d, want 1", newCtx.State.PlanAttempts)
	}
	if isPlanApproved(newCtx.State) {
		t.Error("PlanApproved must be false after revise")
	}
}

func TestGateOnReport_ReviseTwice_AttemptsIncrementTo2(t *testing.T) {
	settings := makeSettings(false, false, true)
	task := makeImportantTask("task-1", false)
	ctx := makeExecCtx(settings, task, makeExecState(""), 0, 3)

	first := StageReport{Accepted: false, PlanFeedback: "ilk feedback"}
	ctx, err := gateStage().OnReport(ctx, first)
	if err != nil {
		t.Fatalf("first revise error: %v", err)
	}

	second := StageReport{Accepted: false, PlanFeedback: "ikinci feedback"}
	ctx, err = gateStage().OnReport(ctx, second)
	if err != nil {
		t.Fatalf("second revise error: %v", err)
	}

	if ctx.State.PlanAttempts != 2 {
		t.Errorf("PlanAttempts: got %d, want 2", ctx.State.PlanAttempts)
	}
	if ctx.State.PlanFeedback != "ikinci feedback" {
		t.Errorf("PlanFeedback: got %q, want 'ikinci feedback'", ctx.State.PlanFeedback)
	}
}

func TestGateOnReport_Accept_Idempotent_StaysApproved(t *testing.T) {
	settings := makeSettings(false, false, true)
	task := makeImportantTask("task-1", false)
	st := stateWithApprovedPlan()
	ctx := makeExecCtx(settings, task, st, 0, 3)

	plan := &spec.TaskPlan{TaskID: "task-1", Approach: "y"}
	report := StageReport{Accepted: true, Plan: plan}
	newCtx, err := gateStage().OnReport(ctx, report)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !newCtx.State.PlanApproved {
		t.Error("PlanApproved must stay true after re-accept")
	}
	if newCtx.State.Plan == nil || newCtx.State.Plan.Approach != "y" {
		t.Errorf("Plan must be replaced by the latest accepted plan, got %+v", newCtx.State.Plan)
	}
}

func TestGateOnReport_ZeroState_HandledSafely(t *testing.T) {
	settings := makeSettings(false, false, true)
	task := makeImportantTask("task-1", false)

	zeroSt := spec.ExecState{}
	ctxAccept := makeExecCtx(settings, task, zeroSt, 0, 3)
	plan := &spec.TaskPlan{TaskID: "task-1", Approach: "z"}
	reportAccept := StageReport{Accepted: true, Plan: plan}
	newCtx, err := gateStage().OnReport(ctxAccept, reportAccept)
	if err != nil {
		t.Fatalf("accept on zero state: unexpected error: %v", err)
	}
	if newCtx.State.Plan == nil {
		t.Error("Plan must be set after accept")
	}
	if !isPlanApproved(newCtx.State) {
		t.Error("PlanApproved must be true after accept on zero state")
	}

	zeroSt2 := spec.ExecState{}
	ctxRevise := makeExecCtx(settings, task, zeroSt2, 0, 3)
	reportRevise := StageReport{Accepted: false, PlanFeedback: "feedback"}
	newCtx2, err := gateStage().OnReport(ctxRevise, reportRevise)
	if err != nil {
		t.Fatalf("revise on zero state: unexpected error: %v", err)
	}
	if newCtx2.State.PlanFeedback != "feedback" {
		t.Errorf("PlanFeedback: got %q, want 'feedback'", newCtx2.State.PlanFeedback)
	}
	if newCtx2.State.PlanAttempts != 1 {
		t.Errorf("PlanAttempts: got %d, want 1", newCtx2.State.PlanAttempts)
	}
}

func makeTaskWithCriteria(id string, tddEnabled bool, criteria []spec.Criterion) spec.Task {
	return spec.Task{
		ID:         id,
		Title:      id,
		TDDEnabled: tddEnabled,
		Criteria:   criteria,
	}
}

func TestAppendACsAndECs_FullTriple_RendersGivenWhenThen(t *testing.T) {
	task := makeTaskWithCriteria("t-1", true, []spec.Criterion{
		{ID: "ac-1", Given: "a user", When: "they submit", Then: "it succeeds"},
	})
	var b strings.Builder
	appendACsAndECs(&b, task)
	got := b.String()
	want := "- [ac-1] GIVEN a user WHEN they submit THEN it succeeds"
	if !strings.Contains(got, want) {
		t.Errorf("appendACsAndECs full triple: want %q in output, got:\n%s", want, got)
	}
}

func TestAppendACsAndECs_OmitsEmptyGivenAndWhen(t *testing.T) {
	task := makeTaskWithCriteria("t-1", true, []spec.Criterion{
		{ID: "ac-1", Given: "", When: "", Then: "it succeeds"},
	})
	var b strings.Builder
	appendACsAndECs(&b, task)
	got := b.String()
	wantLine := "- [ac-1] THEN it succeeds"
	if !strings.Contains(got, wantLine) {
		t.Errorf("appendACsAndECs empty given+when: want %q, got:\n%s", wantLine, got)
	}
	if strings.Contains(got, "GIVEN") || strings.Contains(got, "WHEN") {
		t.Errorf("appendACsAndECs: GIVEN/WHEN must be omitted when empty, got:\n%s", got)
	}
}

func TestAppendACsAndECs_GivenEmptyWhenPresent(t *testing.T) {
	task := makeTaskWithCriteria("t-1", true, []spec.Criterion{
		{ID: "ac-1", Given: "", When: "they submit", Then: "it succeeds"},
	})
	var b strings.Builder
	appendACsAndECs(&b, task)
	got := b.String()
	wantLine := "- [ac-1] WHEN they submit THEN it succeeds"
	if !strings.Contains(got, wantLine) {
		t.Errorf("appendACsAndECs empty given, when+then present: want %q, got:\n%s", wantLine, got)
	}
	if strings.Contains(got, "GIVEN") {
		t.Errorf("appendACsAndECs: GIVEN must be omitted when empty, got:\n%s", got)
	}
}

func TestAppendACsAndECs_RawFallback(t *testing.T) {
	task := makeTaskWithCriteria("t-1", true, []spec.Criterion{
		{ID: "ac-1", Given: "ignored", When: "ignored", Then: "ignored", Raw: "custom raw text"},
	})
	var b strings.Builder
	appendACsAndECs(&b, task)
	got := b.String()
	wantLine := "- [ac-1] custom raw text"
	if !strings.Contains(got, wantLine) {
		t.Errorf("appendACsAndECs raw fallback: want %q, got:\n%s", wantLine, got)
	}
	if strings.Contains(got, "GIVEN") || strings.Contains(got, "WHEN") || strings.Contains(got, "THEN") {
		t.Errorf("appendACsAndECs raw fallback: GIVEN/WHEN/THEN must not appear when Raw is used, got:\n%s", got)
	}
}

func TestAppendACsAndECs_EmptyThenAndRaw_NoPanic(t *testing.T) {
	task := makeTaskWithCriteria("t-1", true, []spec.Criterion{
		{ID: "ac-1", Given: "", When: "", Then: "", Raw: ""},
	})
	var b strings.Builder
	appendACsAndECs(&b, task)
	got := b.String()
	if !strings.Contains(got, "[ac-1]") {
		t.Errorf("appendACsAndECs empty then+raw: criterion id [ac-1] must still appear, got:\n%s", got)
	}
	if strings.Contains(got, "THEN") {
		t.Errorf("appendACsAndECs empty then+raw: THEN must not appear when Then is empty, got:\n%s", got)
	}
}

func TestAppendACsAndECs_MultipleCriteria_EachOnOwnLineWithId(t *testing.T) {
	task := makeTaskWithCriteria("t-1", true, []spec.Criterion{
		{ID: "ac-1", Given: "g1", When: "w1", Then: "t1"},
		{ID: "ac-2", Given: "g2", When: "w2", Then: "t2"},
	})
	var b strings.Builder
	appendACsAndECs(&b, task)
	got := b.String()
	if !strings.Contains(got, "- [ac-1] GIVEN g1 WHEN w1 THEN t1") {
		t.Errorf("appendACsAndECs multiple criteria: missing ac-1 line, got:\n%s", got)
	}
	if !strings.Contains(got, "- [ac-2] GIVEN g2 WHEN w2 THEN t2") {
		t.Errorf("appendACsAndECs multiple criteria: missing ac-2 line, got:\n%s", got)
	}
	ac1Idx := strings.Index(got, "[ac-1]")
	ac2Idx := strings.Index(got, "[ac-2]")
	if ac1Idx == -1 || ac2Idx == -1 || ac1Idx >= ac2Idx {
		t.Errorf("appendACsAndECs multiple criteria: ac-1 must appear before ac-2, got:\n%s", got)
	}
}

func TestGateApplies_AfterRevise_StillFires(t *testing.T) {
	settings := makeSettings(false, false, true)
	task := makeImportantTask("task-1", false)

	stRevised := spec.ExecState{
		PlanFeedback: "feedback",
		PlanAttempts: 1,
	}
	ctxRevised := makeExecCtx(settings, task, stRevised, 0, 3)
	if !gateStage().Applies(ctxRevised) {
		t.Error("gate Applies: got false after revise, want true (plan not approved)")
	}

	stApproved := stateWithApprovedPlan()
	ctxApproved := makeExecCtx(settings, task, stApproved, 0, 3)
	if gateStage().Applies(ctxApproved) {
		t.Error("gate Applies: got true after accept, want false (plan approved)")
	}
}

func TestGateOnReport_AcceptWithoutPlan_ReturnsError(t *testing.T) {
	settings := makeSettings(false, false, true)
	task := makeImportantTask("task-1", false)
	ctx := makeExecCtx(settings, task, spec.ExecState{}, 0, 3)

	_, err := gateStage().OnReport(ctx, StageReport{Accepted: true})
	if err == nil {
		t.Fatal("expected error for accepted gate answer without plan")
	}
	if !strings.Contains(err.Error(), "plan") {
		t.Fatalf("expected plan-missing error, got %q", err.Error())
	}
}

func TestGateOnReport_RejectWithoutFeedback_ReturnsError(t *testing.T) {
	settings := makeSettings(false, false, true)
	task := makeImportantTask("task-1", false)
	ctx := makeExecCtx(settings, task, spec.ExecState{}, 0, 3)

	_, err := gateStage().OnReport(ctx, StageReport{Accepted: false})
	if err == nil {
		t.Fatal("expected error for rejected gate answer without feedback")
	}
	if !strings.Contains(err.Error(), "planFeedback") {
		t.Fatalf("expected empty-feedback error, got %q", err.Error())
	}
}

func TestGateOnReport_PlanWithoutAcceptedFlag_Approves(t *testing.T) {
	settings := makeSettings(false, false, true)
	task := makeImportantTask("task-1", false)
	ctx := makeExecCtx(settings, task, spec.ExecState{}, 0, 3)

	plan := &spec.TaskPlan{TaskID: "task-1", Approach: "do X", TouchedFiles: []string{"a.go"}}
	newCtx, err := gateStage().OnReport(ctx, StageReport{Plan: plan})
	if err != nil {
		t.Fatalf("OnReport: %v", err)
	}
	if !newCtx.State.PlanApproved {
		t.Fatal("a gate answer carrying a plan without feedback must approve the plan")
	}
	if newCtx.State.Plan == nil || newCtx.State.Plan.Approach != "do X" {
		t.Fatalf("expected plan stored, got %+v", newCtx.State.Plan)
	}
}
