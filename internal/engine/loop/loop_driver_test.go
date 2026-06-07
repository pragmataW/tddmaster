package loop

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/spec"
)

var _ engine.Driver = (*LoopDriver)(nil)

func seedLoopSpec(t *testing.T, root, slug string, tasks []spec.Task, execution *spec.ExecState) *engine.Context {
	t.Helper()
	st := spec.State{
		Version: 1,
		Slug:    slug,
		Phase:   "executing",
		Answers: map[string][]spec.Answer{},
	}
	if err := spec.SaveState(root, slug, st); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	pr := spec.Progress{
		Spec:      slug,
		Status:    spec.StatusDraft,
		Tasks:     tasks,
		Execution: execution,
	}
	if err := spec.SaveProgress(root, slug, pr); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	settings := spec.DefaultSettings()
	if err := spec.SaveSettings(root, slug, settings); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	defs := []engine.PhaseDef{
		{
			ID:     "executing",
			Driver: NewLoopDriver(),
		},
	}
	ctx, err := engine.Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return ctx
}

func reportWith(t *testing.T, passed bool, refactorNotes bool) []byte {
	t.Helper()
	var notes []RefactorNote
	if refactorNotes {
		notes = []RefactorNote{{Suggestion: "refactor notes"}}
	}
	b, err := json.Marshal(StageReport{Passed: passed, RefactorNotes: notes})
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	return b
}

func greenImpl(t *testing.T) []byte {
	t.Helper()
	b, err := json.Marshal(StageReport{Completed: []string{"impl"}})
	if err != nil {
		t.Fatalf("marshal green impl report: %v", err)
	}
	return b
}

func submitGreenPass(t *testing.T, ctx *engine.Context, refactorNotes bool) {
	t.Helper()
	if _, err := ctx.Submit(greenImpl(t)); err != nil {
		t.Fatalf("Submit green impl: %v", err)
	}
	if _, err := ctx.Submit(reportWith(t, true, refactorNotes)); err != nil {
		t.Fatalf("Submit green verify: %v", err)
	}
}

func TestLoopDriver_Edge2_NilExecution_InitializesOnFirstNext(t *testing.T) {
	root := t.TempDir()
	slug := "edge2"
	tasks := []spec.Task{
		{ID: "t1", Title: "task one", Done: false, TDDEnabled: true},
	}
	ctx := seedLoopSpec(t, root, slug, tasks, nil)

	action, err := ctx.Next()

	if err != nil {
		t.Fatalf("Next returned error: %v", err)
	}
	if action.Action == "" {
		t.Fatal("Next returned empty ActionType; expected a valid Action")
	}
}

func TestLoopDriver_Edge4_EmptyTaskList_NextReturnsPhaseDone(t *testing.T) {
	root := t.TempDir()
	slug := "edge4"
	ctx := seedLoopSpec(t, root, slug, []spec.Task{}, nil)

	action, err := ctx.Next()

	if err != nil {
		t.Fatalf("Next returned error: %v", err)
	}
	if action.Action != engine.ActionTerminal {
		t.Fatalf("expected ActionTerminal for empty tasks, got %q", action.Action)
	}
}

func TestLoopDriver_Edge4_EmptyTaskList_PhaseAdvances(t *testing.T) {
	root := t.TempDir()
	slug := "edge4-phase"
	ctx := seedLoopSpec(t, root, slug, []spec.Task{}, nil)

	_, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}

	if ctx.Phase() != engine.PhaseComplete {
		t.Fatalf("expected PhaseComplete after empty task list, got %q", ctx.Phase())
	}
}

func TestLoopDriver_Edge3_AllTasksDone_NextReturnsPhaseDoneAndStatusCompleted(t *testing.T) {
	root := t.TempDir()
	slug := "edge3"
	tasks := []spec.Task{
		{ID: "t1", Title: "done task", Done: true, TDDEnabled: false},
	}
	execution := &spec.ExecState{Iteration: 1}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if action.Action != engine.ActionTerminal {
		t.Fatalf("expected ActionTerminal when all tasks done, got %q", action.Action)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Status != spec.StatusCompleted {
		t.Fatalf("expected status %q, got %q", spec.StatusCompleted, pr.Status)
	}
}

func TestLoopDriver_Edge14_TDDTask_RedToGreenOnPassingReport(t *testing.T) {
	root := t.TempDir()
	slug := "edge14"
	tasks := []spec.Task{
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleRed}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	_, err := ctx.Submit(reportWith(t, true, false))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Execution == nil {
		t.Fatal("expected non-nil Execution after Submit")
	}
	if pr.Execution.TDDCycle != cycleGreen {
		t.Fatalf("expected TDDCycle %q after red→green, got %q", cycleGreen, pr.Execution.TDDCycle)
	}
}

func TestLoopDriver_Edge10_GreenPassWithoutRefactorNotes_TaskDone(t *testing.T) {
	root := t.TempDir()
	slug := "edge10-done"
	tasks := []spec.Task{
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleGreen}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	submitGreenPass(t, ctx, false)

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if !pr.Tasks[0].Done {
		t.Fatal("expected task Done=true after green pass without refactor notes")
	}
}

func TestLoopDriver_Edge10_GreenPassWithRefactorNotes_CycleAdvancesToRefactor(t *testing.T) {
	root := t.TempDir()
	slug := "edge10-refactor"
	tasks := []spec.Task{
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleGreen}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	submitGreenPass(t, ctx, true)

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Execution == nil {
		t.Fatal("expected non-nil Execution")
	}
	if pr.Execution.TDDCycle != cycleRefactor {
		t.Fatalf("expected TDDCycle %q after green+refactorNotes, got %q", cycleRefactor, pr.Execution.TDDCycle)
	}
	if pr.Tasks[0].Done {
		t.Fatal("expected task NOT done when advancing to refactor")
	}
}

func TestLoopDriver_Edge5_RefactorCap_TaskDoneAfterCapReached(t *testing.T) {
	root := t.TempDir()
	slug := "edge5"
	tasks := []spec.Task{
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleRefactor, RefactorRounds: 0, RefactorApplied: true}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	for i := 0; i < 3*defaultMaxRefactorRounds; i++ {
		_, err := ctx.Submit(reportWith(t, true, true))
		if err != nil {
			t.Fatalf("Submit iteration %d: %v", i, err)
		}

		pr, err := spec.LoadProgress(root, slug)
		if err != nil {
			t.Fatalf("LoadProgress iteration %d: %v", i, err)
		}
		if pr.Tasks[0].Done {
			return
		}
		if pr.Execution != nil && pr.Execution.TDDCycle == cycleRefactor {
			execution = pr.Execution
		}

		ctx = seedLoopSpec(t, root, slug, pr.Tasks, pr.Execution)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress after cap: %v", err)
	}
	if !pr.Tasks[0].Done {
		t.Fatal("expected task Done=true after refactor cap reached")
	}
}

func TestLoopDriver_Edge9_RefactorApplyHead_WaitsForVerifier(t *testing.T) {
	root := t.TempDir()
	slug := "edge9"
	tasks := []spec.Task{
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleRefactor, RefactorApplied: false}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	applyReport, err := json.Marshal(StageReport{Passed: true, RefactorApplied: true, RefactorNotes: []RefactorNote{{Suggestion: "refactor notes"}}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if _, submitErr := ctx.Submit(applyReport); submitErr != nil {
		t.Fatalf("apply head must not error (no bypass): %v", submitErr)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Execution == nil {
		t.Fatal("expected Execution to remain non-nil")
	}
	if !pr.Execution.RefactorApplied {
		t.Fatal("expected RefactorApplied=true after apply head")
	}
	if pr.Execution.TDDCycle != cycleRefactor {
		t.Fatalf("expected cycle still refactor (waiting for verifier), got %q", pr.Execution.TDDCycle)
	}
	if pr.Tasks[0].Done {
		t.Fatal("expected task NOT done after apply head")
	}
}

func TestLoopDriver_Edge12_MalformedJSON_SubmitReturnsErrorStateUnchanged(t *testing.T) {
	root := t.TempDir()
	slug := "edge12"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleRed, Iteration: 2}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	_, err := ctx.Submit([]byte(`not-valid-json`))
	if err == nil {
		t.Fatal("expected error for malformed JSON answer")
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Execution == nil {
		t.Fatal("expected Execution persisted from seed")
	}
	if pr.Execution.Iteration != 2 {
		t.Fatalf("expected Iteration unchanged at 2, got %d", pr.Execution.Iteration)
	}
	if pr.Tasks[0].Done {
		t.Fatal("expected task unchanged (not done) after bad JSON")
	}
}

func TestLoopDriver_Edge12_EmptyAnswer_SubmitReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "edge12-empty"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleRed}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	_, err := ctx.Submit([]byte{})
	if err == nil {
		t.Fatal("expected error for empty answer")
	}
}

func TestLoopDriver_Edge13_MixedTDDEnabled_ReseedCycleSetsEmptyForNonTDD(t *testing.T) {
	root := t.TempDir()
	slug := "edge13"
	tasks := []spec.Task{
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true},
		{ID: "t2", Title: "non-tdd task", Done: false, TDDEnabled: false},
	}
	execution := &spec.ExecState{TDDCycle: cycleGreen}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	submitGreenPass(t, ctx, false)

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if !pr.Tasks[0].Done {
		t.Fatal("expected t1 Done after green pass")
	}
	if pr.Execution == nil {
		t.Fatal("expected non-nil Execution")
	}
	if pr.Execution.TDDCycle != cycleEmpty {
		t.Fatalf("expected empty TDDCycle for non-TDD next task, got %q", pr.Execution.TDDCycle)
	}

	ctx2 := seedLoopSpec(t, root, slug, pr.Tasks, pr.Execution)
	action, err := ctx2.Next()
	if err != nil {
		t.Fatalf("Next for t2: %v", err)
	}
	if action.DelegateAgent == "" {
		t.Fatal("expected executor-stage action for non-TDD task t2")
	}
}

func TestLoopDriver_Edge11_IterationExceedsMax_ReturnsStopAction(t *testing.T) {
	root := t.TempDir()
	slug := "edge11"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: true},
	}
	maxIter := 15
	execution := &spec.ExecState{TDDCycle: cycleRed, Iteration: maxIter}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if action.Action == "" {
		t.Fatal("expected a non-empty ActionType for stop condition")
	}
	if ctx.Phase() == engine.PhaseComplete {
		t.Fatal("phase must NOT advance to complete when iteration limit reached (phaseDone should be false)")
	}
}

func TestLoopDriver_Persistence_SubmitPersistsExecutionAndTaskDone(t *testing.T) {
	root := t.TempDir()
	slug := "persist"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleRed}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	_, err := ctx.Submit(reportWith(t, true, false))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Execution == nil {
		t.Fatal("Execution must be persisted after Submit")
	}
	if pr.Execution.TDDCycle != cycleGreen {
		t.Fatalf("expected persisted TDDCycle %q, got %q", cycleGreen, pr.Execution.TDDCycle)
	}
}

func TestLoopDriver_Persistence_AllTasksDone_StatusPersistedAsCompleted(t *testing.T) {
	root := t.TempDir()
	slug := "persist-done"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleGreen}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	submitGreenPass(t, ctx, false)

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Status != spec.StatusCompleted {
		t.Fatalf("expected Status=%q after all tasks done, got %q", spec.StatusCompleted, pr.Status)
	}
}

func reportWithCoverage(t *testing.T, passed bool, failedACs []string, uncoveredEdgeCases []string) []byte {
	t.Helper()
	b, err := json.Marshal(StageReport{
		Passed:             passed,
		FailedACs:          failedACs,
		UncoveredEdgeCases: uncoveredEdgeCases,
	})
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	return b
}

func TestLoopDriver_Edge3_GreenPassWithFailedACs_TaskNotDone(t *testing.T) {
	root := t.TempDir()
	slug := "cov-guard-failedacs"
	tasks := []spec.Task{
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleGreen}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	_, err := ctx.Submit(reportWithCoverage(t, true, []string{"ac-1"}, nil))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Tasks[0].Done {
		t.Fatal("expected task.Done=false when FailedACs present despite Passed=true")
	}
	if pr.Execution == nil {
		t.Fatal("expected non-nil Execution")
	}
	if pr.Execution.TDDCycle != cycleGreen {
		t.Fatalf("expected TDDCycle=%q (unchanged), got %q", cycleGreen, pr.Execution.TDDCycle)
	}
	if len(pr.Execution.LastFailedACs) == 0 || pr.Execution.LastFailedACs[0] != "ac-1" {
		t.Fatalf("expected LastFailedACs=[\"ac-1\"], got %v", pr.Execution.LastFailedACs)
	}
}

func TestLoopDriver_Edge4_GreenPassWithUncoveredEC_TaskNotDone(t *testing.T) {
	root := t.TempDir()
	slug := "cov-guard-ec"
	tasks := []spec.Task{
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleGreen}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	_, err := ctx.Submit(reportWithCoverage(t, true, nil, []string{"ec-2"}))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Tasks[0].Done {
		t.Fatal("expected task.Done=false when UncoveredEdgeCases present despite Passed=true")
	}
	if pr.Execution == nil {
		t.Fatal("expected non-nil Execution")
	}
	if pr.Execution.TDDCycle != cycleGreen {
		t.Fatalf("expected TDDCycle=%q (unchanged), got %q", cycleGreen, pr.Execution.TDDCycle)
	}
	if len(pr.Execution.LastUncoveredEC) == 0 || pr.Execution.LastUncoveredEC[0] != "ec-2" {
		t.Fatalf("expected LastUncoveredEC=[\"ec-2\"], got %v", pr.Execution.LastUncoveredEC)
	}
}

func TestLoopDriver_CoverageCleared_OnCleanPass(t *testing.T) {
	root := t.TempDir()
	slug := "cov-cleared"
	tasks := []spec.Task{
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{
		TDDCycle:      cycleGreen,
		LastFailedACs: []string{"ac-1"},
	}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	if _, err := ctx.Submit(greenImpl(t)); err != nil {
		t.Fatalf("Submit green impl: %v", err)
	}
	_, err := ctx.Submit(reportWithCoverage(t, true, nil, nil))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if !pr.Tasks[0].Done {
		t.Fatal("expected task.Done=true after clean pass")
	}
	if pr.Execution != nil && len(pr.Execution.LastFailedACs) > 0 {
		t.Fatalf("expected LastFailedACs cleared, got %v", pr.Execution.LastFailedACs)
	}
}

func executorCompletedReport(t *testing.T, taskID string) []byte {
	t.Helper()
	b, err := json.Marshal(StageReport{Completed: []string{taskID}})
	if err != nil {
		t.Fatalf("marshal executor report: %v", err)
	}
	return b
}

func seedLoopSpecSkipVerify(t *testing.T, root, slug string, tasks []spec.Task, execution *spec.ExecState) *engine.Context {
	t.Helper()
	ctx := seedLoopSpec(t, root, slug, tasks, execution)
	settings := spec.Settings{TDDEnabled: true, SkipVerifierEnabled: true}
	if err := spec.SaveSettings(root, slug, settings); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	rebuilt, err := engine.Build(root, slug, []engine.PhaseDef{{ID: "executing", Driver: NewLoopDriver()}})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	_ = ctx
	return rebuilt
}

func TestLoopDriver_NonTDD_SkipOff_ExecutorReportAdvancesToVerifier_NotComplete(t *testing.T) {
	root := t.TempDir()
	slug := "nontdd-exec-advance"
	tasks := []spec.Task{
		{ID: "t1", Title: "non-tdd task", Done: false, TDDEnabled: false},
	}
	execution := &spec.ExecState{TDDCycle: cycleEmpty}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	if _, err := ctx.Submit(executorCompletedReport(t, "t1")); err != nil {
		t.Fatalf("Submit executor: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Tasks[0].Done {
		t.Fatal("expected task.Done=false after executor report (verifier still pending, skipVerify off)")
	}
	if pr.Execution == nil || !pr.Execution.Implemented {
		t.Fatal("expected Implemented=true after executor report so verifier stage runs next")
	}

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if action.DelegateAgent != string(promptregistry.AgentVerifier) {
		t.Fatalf("expected verifier instruction after executor, got delegate %q", action.DelegateAgent)
	}
}

func TestLoopDriver_NonTDD_SkipOff_TwoStage_CompletesOnVerifierPass(t *testing.T) {
	root := t.TempDir()
	slug := "nontdd-twostage"
	tasks := []spec.Task{
		{ID: "t1", Title: "non-tdd task", Done: false, TDDEnabled: false},
	}
	execution := &spec.ExecState{TDDCycle: cycleEmpty}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	if _, err := ctx.Submit(executorCompletedReport(t, "t1")); err != nil {
		t.Fatalf("Submit executor: %v", err)
	}
	if _, err := ctx.Submit(reportWithCoverage(t, true, nil, nil)); err != nil {
		t.Fatalf("Submit verifier: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if !pr.Tasks[0].Done {
		t.Fatal("expected task.Done=true after executor + verifier pass (skipVerify off)")
	}
}

func TestLoopDriver_NonTDD_SkipOff_VerifierFail_NotComplete(t *testing.T) {
	root := t.TempDir()
	slug := "nontdd-verifierfail"
	tasks := []spec.Task{
		{ID: "t1", Title: "non-tdd task", Done: false, TDDEnabled: false},
	}
	execution := &spec.ExecState{TDDCycle: cycleEmpty}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	if _, err := ctx.Submit(executorCompletedReport(t, "t1")); err != nil {
		t.Fatalf("Submit executor: %v", err)
	}
	if _, err := ctx.Submit(reportWithCoverage(t, true, []string{"ac-1"}, nil)); err != nil {
		t.Fatalf("Submit verifier: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Tasks[0].Done {
		t.Fatal("expected task.Done=false when verifier reports FailedACs (skipVerify off)")
	}
	if pr.Execution == nil || pr.Execution.Implemented {
		t.Fatal("expected Implemented reset to false after verifier fail so executor re-runs")
	}
}

func TestLoopDriver_NonTDD_SkipOn_ExecutorReportCompletesInOneSubmit(t *testing.T) {
	root := t.TempDir()
	slug := "nontdd-skipon"
	tasks := []spec.Task{
		{ID: "t1", Title: "non-tdd task", Done: false, TDDEnabled: false},
	}
	execution := &spec.ExecState{TDDCycle: cycleEmpty}
	ctx := seedLoopSpecSkipVerify(t, root, slug, tasks, execution)

	if _, err := ctx.Submit(executorCompletedReport(t, "t1")); err != nil {
		t.Fatalf("Submit executor: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if !pr.Tasks[0].Done {
		t.Fatal("expected task.Done=true after single executor report when skipVerify on")
	}
}

func TestLoopDriver_NonTDD_SkipOn_StaleImplemented_NoDeadlock_Completes(t *testing.T) {
	root := t.TempDir()
	slug := "nontdd-stale-impl"
	tasks := []spec.Task{
		{ID: "t1", Title: "non-tdd task", Done: false, TDDEnabled: false},
	}
	execution := &spec.ExecState{TDDCycle: cycleEmpty, Implemented: true}
	ctx := seedLoopSpecSkipVerify(t, root, slug, tasks, execution)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if action.Action == engine.ActionNotify {
		t.Fatal("expected an executor instruction, got empty notify (deadlock) on stale Implemented + skipVerify on")
	}
	if action.DelegateAgent != string(promptregistry.AgentExecutor) {
		t.Fatalf("expected executor instruction, got delegate %q", action.DelegateAgent)
	}

	if _, err := ctx.Submit(executorCompletedReport(t, "t1")); err != nil {
		t.Fatalf("Submit executor: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if !pr.Tasks[0].Done {
		t.Fatal("expected task.Done=true after executor report despite stale Implemented (skipVerify on)")
	}
}

func TestLoopDriver_NonTDD_SkipOn_BlockedReport_NotComplete(t *testing.T) {
	root := t.TempDir()
	slug := "nontdd-skipon-blocked"
	tasks := []spec.Task{
		{ID: "t1", Title: "non-tdd task", Done: false, TDDEnabled: false},
	}
	execution := &spec.ExecState{TDDCycle: cycleEmpty}
	ctx := seedLoopSpecSkipVerify(t, root, slug, tasks, execution)

	report, err := json.Marshal(StageReport{Blocked: []string{"t1"}})
	if err != nil {
		t.Fatalf("marshal blocked report: %v", err)
	}
	if _, err := ctx.Submit(report); err != nil {
		t.Fatalf("Submit blocked: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Tasks[0].Done {
		t.Fatal("expected task.Done=false when executor reports blocked (skipVerify on)")
	}
}

func TestLoopDriver_TaskDone_RerendersSpecMd(t *testing.T) {
	root := t.TempDir()
	slug := "rerender-done"
	tasks := []spec.Task{
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleGreen}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	submitGreenPass(t, ctx, false)

	data, err := os.ReadFile(paths.SpecMd(root, slug))
	if err != nil {
		t.Fatalf("ReadFile spec.md: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "- [x] t1: tdd task") {
		t.Errorf("expected done task '[x] t1: tdd task' in spec.md after Submit, got:\n%s", content)
	}
}

func TestLoopDriver_AllTasksDone_SpecMdStatusCompleted(t *testing.T) {
	root := t.TempDir()
	slug := "rerender-completed"
	tasks := []spec.Task{
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleGreen}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	submitGreenPass(t, ctx, false)

	data, err := os.ReadFile(paths.SpecMd(root, slug))
	if err != nil {
		t.Fatalf("ReadFile spec.md: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "## Status\n") {
		t.Errorf("expected '## Status' section in spec.md, got:\n%s", content)
	}
	if !strings.Contains(content, spec.StatusCompleted) {
		t.Errorf("expected %q in spec.md Status section, got:\n%s", spec.StatusCompleted, content)
	}
}

func TestLoopDriver_TaskNotDone_SpecMdNotMarked(t *testing.T) {
	root := t.TempDir()
	slug := "rerender-notdone"
	tasks := []spec.Task{
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleGreen}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	_, err := ctx.Submit(reportWithCoverage(t, true, nil, []string{"ec-1"}))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Tasks[0].Done {
		t.Fatal("expected task.Done=false when UncoveredEdgeCases present")
	}

	data, err := os.ReadFile(paths.SpecMd(root, slug))
	if err != nil {
		t.Fatalf("ReadFile spec.md: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "- [ ] t1: tdd task") {
		t.Errorf("expected pending task '[ ] t1: tdd task' in spec.md when task not done, got:\n%s", content)
	}
}

func TestLoopDriver_IterationMax_NextNotifyHasRestartText(t *testing.T) {
	root := t.TempDir()
	slug := "iter-max-notify"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: true},
	}
	maxIter := 15
	execution := &spec.ExecState{TDDCycle: cycleRed, Iteration: maxIter}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if action.Action != engine.ActionNotify {
		t.Fatalf("expected ActionNotify when iteration >= max, got %q", action.Action)
	}
	if !strings.Contains(action.Instruction, promptregistry.RestartRecommendedText) {
		t.Fatalf("expected Instruction to contain RestartRecommendedText, got %q", action.Instruction)
	}
}

func TestLoopDriver_Continue_ResetsIterationToZero(t *testing.T) {
	root := t.TempDir()
	slug := "continue-reset"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: true},
	}
	maxIter := 15
	execution := &spec.ExecState{TDDCycle: cycleRed, Iteration: maxIter}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	action, err := ctx.Submit([]byte("continue"))
	if err != nil {
		t.Fatalf("Submit(continue): %v", err)
	}
	if action.Action == engine.ActionError {
		t.Fatalf("expected no ActionError for continue, got ActionError")
	}

	pr, loadErr := spec.LoadProgress(root, slug)
	if loadErr != nil {
		t.Fatalf("LoadProgress: %v", loadErr)
	}
	if pr.Execution == nil {
		t.Fatal("expected non-nil Execution after continue")
	}
	if pr.Execution.Iteration != 0 {
		t.Fatalf("expected Iteration=0 after continue, got %d", pr.Execution.Iteration)
	}
}

func TestLoopDriver_Continue_NotTreatedAsInvalidJSON(t *testing.T) {
	root := t.TempDir()
	slug := "continue-valid"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: true},
	}
	maxIter := 15
	execution := &spec.ExecState{TDDCycle: cycleRed, Iteration: maxIter}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	_, err := ctx.Submit([]byte("continue"))
	if err != nil {
		t.Fatalf("Submit(continue) returned error; expected nil (continue is not invalid JSON): %v", err)
	}
}

func TestLoopDriver_InvalidJSON_StillErrors(t *testing.T) {
	root := t.TempDir()
	slug := "invalid-json-still-err"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: true},
	}
	maxIter := 15
	execution := &spec.ExecState{TDDCycle: cycleRed, Iteration: maxIter}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	_, err := ctx.Submit([]byte("{bozuk"))
	if err == nil {
		t.Fatal("expected error for malformed JSON (non-continue), got nil")
	}
}

func TestLoopDriver_Edge5_GreenExecutorSelfReport_DoesNotAdvance(t *testing.T) {
	root := t.TempDir()
	slug := "green-executor-self"
	tasks := []spec.Task{
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleGreen}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	selfReport, err := json.Marshal(StageReport{
		Passed:    false,
		Completed: []string{"impl"},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	_, submitErr := ctx.Submit(selfReport)
	if submitErr != nil {
		t.Fatalf("Submit returned unexpected error: %v", submitErr)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Tasks[0].Done {
		t.Fatal("expected task.Done=false for executor self-report (Passed=false)")
	}
	if pr.Execution == nil {
		t.Fatal("expected non-nil Execution")
	}
	if pr.Execution.TDDCycle != cycleGreen {
		t.Fatalf("expected TDDCycle=%q unchanged, got %q", cycleGreen, pr.Execution.TDDCycle)
	}
}
