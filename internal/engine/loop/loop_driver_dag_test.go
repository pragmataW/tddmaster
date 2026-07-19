package loop

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func seedDagSpec(t *testing.T, root, slug string, tasks []spec.Task, mutate func(*spec.Settings), iterations int) *engine.Context {
	t.Helper()
	return seedLoopSpecCore(t, root, slug, tasks, mutate, iterations)
}

func diamondTasks() []spec.Task {
	return []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: false},
		{ID: "task-2", Title: "two", TDDEnabled: false},
		{ID: "task-3", Title: "three", TDDEnabled: false, DependsOn: []string{"task-1", "task-2"}},
	}
}

func taskActionIDs(action engine.Action) []string {
	ids := make([]string, 0, len(action.Tasks))
	for _, ta := range action.Tasks {
		ids = append(ids, ta.TaskID)
	}
	return ids
}

func TestDag_AC5_ReadyTasksBatched_DependentExcluded(t *testing.T) {
	root := t.TempDir()
	slug := "dag-ac5"
	ctx := seedDagSpec(t, root, slug, diamondTasks(), nil, 0)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if action.Action != engine.ActionInstruct {
		t.Fatalf("expected ActionInstruct, got %q", action.Action)
	}
	ids := taskActionIDs(action)
	if len(ids) != 2 || ids[0] != "task-1" || ids[1] != "task-2" {
		t.Fatalf("expected batch [task-1 task-2], got %v", ids)
	}
	for _, ta := range action.Tasks {
		if ta.TaskID == "task-3" {
			t.Fatal("task-3 must not be in the batch while dependencies are pending")
		}
	}
	if action.DelegateAgent != "" {
		t.Fatalf("expected empty top-level DelegateAgent for batch instruct, got %q", action.DelegateAgent)
	}
	if action.ExpectedInput.Format != engine.FormatJSON {
		t.Fatalf("expected top-level ExpectedInput.Format json, got %q", action.ExpectedInput.Format)
	}
	if !strings.Contains(action.Instruction, "2 task(s) ready for parallel execution") {
		t.Fatalf("expected orchestration summary in top-level Instruction, got %q", action.Instruction)
	}
	if !strings.Contains(action.Instruction, "taskId") {
		t.Fatalf("expected taskId requirement in top-level Instruction, got %q", action.Instruction)
	}
}

func TestDag_AC6_ReportForOneTask_DoesNotTouchOtherExec(t *testing.T) {
	root := t.TempDir()
	slug := "dag-ac6"
	ctx := seedDagSpec(t, root, slug, diamondTasks(), nil, 0)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next: %v", err)
	}
	before, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress before: %v", err)
	}
	if before.Tasks[1].Exec == nil {
		t.Fatal("expected task-2 Exec seeded after Next")
	}
	beforeJSON, err := json.Marshal(before.Tasks[1].Exec)
	if err != nil {
		t.Fatalf("marshal before: %v", err)
	}

	if _, err := ctx.Submit(executorCompletedReport(t, "task-1")); err != nil {
		t.Fatalf("Submit task-1: %v", err)
	}

	after, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress after: %v", err)
	}
	afterJSON, err := json.Marshal(after.Tasks[1].Exec)
	if err != nil {
		t.Fatalf("marshal after: %v", err)
	}
	if string(beforeJSON) != string(afterJSON) {
		t.Fatalf("task-2 Exec changed after task-1 report:\nbefore: %s\nafter: %s", beforeJSON, afterJSON)
	}
	if !after.Tasks[0].Exec.Implemented {
		t.Fatal("expected task-1 Exec.Implemented=true after its executor report")
	}
}

func TestDag_AC7_DependencyCompletion_MakesDependentReady(t *testing.T) {
	root := t.TempDir()
	slug := "dag-ac7"
	ctx := seedDagSpec(t, root, slug, diamondTasks(), func(s *spec.Settings) {
		s.SkipVerifierEnabled = true
	}, 0)

	if _, err := ctx.Submit(executorCompletedReport(t, "task-1")); err != nil {
		t.Fatalf("Submit task-1: %v", err)
	}
	if _, err := ctx.Submit(executorCompletedReport(t, "task-2")); err != nil {
		t.Fatalf("Submit task-2: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if !pr.Tasks[0].Done || !pr.Tasks[1].Done {
		t.Fatalf("expected task-1 and task-2 done, got %v %v", pr.Tasks[0].Done, pr.Tasks[1].Done)
	}

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	ids := taskActionIDs(action)
	if len(ids) != 1 || ids[0] != "task-3" {
		t.Fatalf("expected batch [task-3] after dependencies complete, got %v", ids)
	}
}

func TestDag_AC9_ReportMissingTaskID_Errors(t *testing.T) {
	root := t.TempDir()
	slug := "dag-ac9-missing"
	ctx := seedDagSpec(t, root, slug, diamondTasks(), nil, 0)

	_, err := ctx.Submit(marshalStageReport(t, StageReport{Completed: []string{"impl"}}))
	if err == nil {
		t.Fatal("expected error for report without taskId")
	}
	if !strings.Contains(err.Error(), "report missing taskId") {
		t.Fatalf("expected 'report missing taskId' in error, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "task-1") || !strings.Contains(err.Error(), "task-2") {
		t.Fatalf("expected ready task ids listed in error, got %q", err.Error())
	}
}

func TestDag_AC9_UnknownTaskID_Errors(t *testing.T) {
	root := t.TempDir()
	slug := "dag-ac9-unknown"
	ctx := seedDagSpec(t, root, slug, diamondTasks(), nil, 0)

	_, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-99", Completed: []string{"impl"}}))
	if err == nil {
		t.Fatal("expected error for unknown taskId")
	}
	if !strings.Contains(err.Error(), `unknown taskId "task-99"`) {
		t.Fatalf("expected unknown-taskId error, got %q", err.Error())
	}
}

func TestDag_AC9_NotReadyTaskID_Errors(t *testing.T) {
	root := t.TempDir()
	slug := "dag-ac9-notready"
	ctx := seedDagSpec(t, root, slug, diamondTasks(), nil, 0)

	_, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-3", Completed: []string{"impl"}}))
	if err == nil {
		t.Fatal("expected error for not-ready taskId")
	}
	if !strings.Contains(err.Error(), `task "task-3" is not ready`) {
		t.Fatalf("expected not-ready error, got %q", err.Error())
	}
}

func TestDag_AC9_DoneTaskID_Errors(t *testing.T) {
	root := t.TempDir()
	slug := "dag-ac9-done"
	tasks := diamondTasks()
	tasks[0].Done = true
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	_, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-1", Completed: []string{"impl"}}))
	if err == nil {
		t.Fatal("expected error for already-done taskId")
	}
	if !strings.Contains(err.Error(), `task "task-1" is already done`) {
		t.Fatalf("expected already-done error, got %q", err.Error())
	}
}

func TestDag_AC9_CorrectTaskID_UpdatesOnlyThatTask(t *testing.T) {
	root := t.TempDir()
	slug := "dag-ac9-route"
	ctx := seedDagSpec(t, root, slug, diamondTasks(), nil, 0)

	if _, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-2", Completed: []string{"impl"}})); err != nil {
		t.Fatalf("Submit task-2: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Tasks[1].Exec == nil || !pr.Tasks[1].Exec.Implemented {
		t.Fatal("expected task-2 Exec.Implemented=true after its report")
	}
	if pr.Tasks[0].Exec != nil && pr.Tasks[0].Exec.Implemented {
		t.Fatal("task-1 Exec must not be updated by a task-2 report")
	}
}

func TestDag_AC10_BlockedTask_OthersContinue(t *testing.T) {
	root := t.TempDir()
	slug := "dag-ac10"
	ctx := seedDagSpec(t, root, slug, diamondTasks(), nil, 0)

	blockedReport := marshalStageReport(t, StageReport{TaskID: "task-1", Blocked: []string{"missing schema", "unclear AC"}})
	if _, err := ctx.Submit(blockedReport); err != nil {
		t.Fatalf("Submit blocked: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if !pr.Tasks[0].Blocked {
		t.Fatal("expected task-1 Blocked=true after blocked report")
	}
	if pr.Tasks[0].BlockedReason != "missing schema; unclear AC" {
		t.Fatalf("expected joined BlockedReason, got %q", pr.Tasks[0].BlockedReason)
	}
	if pr.Tasks[1].Blocked {
		t.Fatal("task-2 must not be blocked by a task-1 blocked report")
	}

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	ids := taskActionIDs(action)
	if len(ids) != 1 || ids[0] != "task-2" {
		t.Fatalf("expected batch [task-2] after task-1 blocked, got %v", ids)
	}
}

func TestDag_AC12_AllBlockedOrWaiting_ReturnsDeadlockError(t *testing.T) {
	root := t.TempDir()
	slug := "dag-ac12"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: false, Blocked: true, BlockedReason: "missing schema"},
		{ID: "task-2", Title: "two", TDDEnabled: false, DependsOn: []string{"task-1"}},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if action.Action != engine.ActionError {
		t.Fatalf("expected ActionError on deadlock, got %q", action.Action)
	}
	if !strings.Contains(action.Instruction, "Deadlock detected: no ready task remains.") {
		t.Fatalf("expected deadlock message, got %q", action.Instruction)
	}
	if !strings.Contains(action.Instruction, "task-1: missing schema") {
		t.Fatalf("expected blocked task with reason listed, got %q", action.Instruction)
	}
	if !strings.Contains(action.Instruction, "task-2: waiting on blocked dependency (task-1)") {
		t.Fatalf("expected waiting task listed, got %q", action.Instruction)
	}
}

func TestDag_SingleReadyTask_StillUsesTasksFormat(t *testing.T) {
	root := t.TempDir()
	slug := "dag-single"
	tasks := []spec.Task{
		{ID: "task-1", Title: "solo", TDDEnabled: false},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if action.Action != engine.ActionInstruct {
		t.Fatalf("expected ActionInstruct, got %q", action.Action)
	}
	if len(action.Tasks) != 1 {
		t.Fatalf("expected Tasks with 1 entry for single ready task, got %d", len(action.Tasks))
	}
	if action.Tasks[0].TaskID != "task-1" {
		t.Fatalf("expected TaskID task-1, got %q", action.Tasks[0].TaskID)
	}
	if action.Tasks[0].Stage != StageIDExecutor {
		t.Fatalf("expected stage executor, got %q", action.Tasks[0].Stage)
	}
}

func TestDag_Iterations_CountTotalSubmits(t *testing.T) {
	root := t.TempDir()
	slug := "dag-iters"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: false},
		{ID: "task-2", Title: "two", TDDEnabled: false},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	if _, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-1", Passed: false})); err != nil {
		t.Fatalf("Submit 1: %v", err)
	}
	if _, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-2", Passed: false})); err != nil {
		t.Fatalf("Submit 2: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Iterations != 2 {
		t.Fatalf("expected Iterations=2 after two submits, got %d", pr.Iterations)
	}
}

func TestDag_Iterations_MaxAfterSubmit_ReturnsRestartNotifyAndResets(t *testing.T) {
	root := t.TempDir()
	slug := "dag-iters-max"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: false},
		{ID: "task-2", Title: "two", TDDEnabled: false},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 14)

	action, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-1", Passed: false}))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if action.Action != engine.ActionNotify {
		t.Fatalf("expected ActionNotify at iteration max, got %q", action.Action)
	}
	if !strings.Contains(action.Instruction, promptregistry.RestartRecommendedText) {
		t.Fatalf("expected RestartRecommendedText, got %q", action.Instruction)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Iterations != 0 {
		t.Fatalf("expected Iterations reset to 0, got %d", pr.Iterations)
	}
}

func TestDag_WorktreeHint_PersistedAndDeterministic(t *testing.T) {
	root := t.TempDir()
	slug := "dag-worktree"
	ctx := seedDagSpec(t, root, slug, diamondTasks(), nil, 0)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	for _, id := range []string{"task-1", "task-2"} {
		var task spec.Task
		for _, tk := range pr.Tasks {
			if tk.ID == id {
				task = tk
			}
		}
		if task.Exec == nil || task.Exec.Worktree == nil {
			t.Fatalf("expected worktree hint persisted for %s", id)
		}
		wantPath := filepath.Join(".tddmaster", "worktrees", slug, id)
		if task.Exec.Worktree.Path != wantPath {
			t.Fatalf("worktree path for %s: got %q, want %q", id, task.Exec.Worktree.Path, wantPath)
		}
		wantBranch := "tddmaster/" + slug + "/" + id
		if task.Exec.Worktree.Branch != wantBranch {
			t.Fatalf("worktree branch for %s: got %q, want %q", id, task.Exec.Worktree.Branch, wantBranch)
		}
	}

	for _, ta := range action.Tasks {
		if ta.Worktree == nil {
			t.Fatalf("expected TaskAction worktree for %s", ta.TaskID)
		}
		if !strings.Contains(ta.Instruction, "=== WORKTREE (binding) ===") {
			t.Fatalf("expected worktree block prefix in instruction for %s", ta.TaskID)
		}
		if !strings.Contains(ta.Instruction, "cwd: "+ta.Worktree.Path) {
			t.Fatalf("expected cwd line in instruction for %s", ta.TaskID)
		}
	}

	action2, err := ctx.Next()
	if err != nil {
		t.Fatalf("second Next: %v", err)
	}
	for i, ta := range action2.Tasks {
		if *ta.Worktree != *action.Tasks[i].Worktree {
			t.Fatalf("worktree hint not deterministic across Next calls: %+v vs %+v", ta.Worktree, action.Tasks[i].Worktree)
		}
	}
}

func TestDag_MixedStages_InSameBatch(t *testing.T) {
	root := t.TempDir()
	slug := "dag-mixed"
	tasks := []spec.Task{
		{ID: "task-1", Title: "tdd", TDDEnabled: true},
		{ID: "task-2", Title: "plain", TDDEnabled: false},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if len(action.Tasks) != 2 {
		t.Fatalf("expected 2 task actions, got %d", len(action.Tasks))
	}

	byID := map[string]engine.TaskAction{}
	for _, ta := range action.Tasks {
		byID[ta.TaskID] = ta
	}
	red := byID["task-1"]
	if red.Stage != StageIDRed {
		t.Fatalf("task-1 stage: got %q, want %q", red.Stage, StageIDRed)
	}
	if red.DelegateAgent != string(promptregistry.AgentTestWriter) {
		t.Fatalf("task-1 delegate: got %q, want test-writer", red.DelegateAgent)
	}
	if !strings.Contains(red.ExpectedInput.Example, `"taskId":"task-1",`) {
		t.Fatalf("task-1 example missing injected taskId: %q", red.ExpectedInput.Example)
	}

	exec := byID["task-2"]
	if exec.Stage != StageIDExecutor {
		t.Fatalf("task-2 stage: got %q, want %q", exec.Stage, StageIDExecutor)
	}
	if exec.DelegateAgent != string(promptregistry.AgentExecutor) {
		t.Fatalf("task-2 delegate: got %q, want executor", exec.DelegateAgent)
	}
	if !strings.Contains(exec.ExpectedInput.Example, `"taskId":"task-2",`) {
		t.Fatalf("task-2 example missing injected taskId: %q", exec.ExpectedInput.Example)
	}
}

func TestDag_GateFirst_ImportantTaskAskPrecedesBatch(t *testing.T) {
	root := t.TempDir()
	slug := "dag-gate"
	tasks := []spec.Task{
		{ID: "task-1", Title: "important", TDDEnabled: false, Important: true},
		{ID: "task-2", Title: "normal", TDDEnabled: false},
	}
	ctx := seedDagSpec(t, root, slug, tasks, func(s *spec.Settings) {
		s.ImportantTaskGateEnabled = true
	}, 0)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if action.Action != engine.ActionAsk {
		t.Fatalf("expected ActionAsk for gate-first, got %q", action.Action)
	}
	if action.DelegateAgent != string(promptregistry.AgentPlanner) {
		t.Fatalf("expected planner delegate at top level, got %q", action.DelegateAgent)
	}
	if len(action.Tasks) != 2 {
		t.Fatalf("expected gate task plus parallel batch in Tasks, got %d", len(action.Tasks))
	}
	if action.Tasks[0].TaskID != "task-1" || action.Tasks[0].Stage != StageIDGate {
		t.Fatalf("expected gate task action for task-1, got %+v", action.Tasks[0])
	}
	if !strings.Contains(action.Tasks[0].ExpectedInput.Example, `"taskId":"task-1",`) {
		t.Fatalf("gate example missing injected taskId: %q", action.Tasks[0].ExpectedInput.Example)
	}
	if action.Tasks[1].TaskID != "task-2" || action.Tasks[1].Stage != StageIDExecutor {
		t.Fatalf("expected parallel executor task action for task-2, got %+v", action.Tasks[1])
	}
	if !strings.Contains(action.Tasks[1].ExpectedInput.Example, `"taskId":"task-2",`) {
		t.Fatalf("task-2 example missing injected taskId: %q", action.Tasks[1].ExpectedInput.Example)
	}
	if action.ExpectedInput.Example != action.Tasks[0].ExpectedInput.Example {
		t.Fatalf("expected top-level ExpectedInput to mirror gate task, got %q", action.ExpectedInput.Example)
	}

	acceptReport := marshalStageReport(t, StageReport{
		TaskID:   "task-1",
		Accepted: true,
		Plan: &spec.TaskPlan{
			TaskID:       "task-1",
			Approach:     "approach",
			TouchedFiles: []string{"a.go"},
		},
	})
	next, err := ctx.Submit(acceptReport)
	if err != nil {
		t.Fatalf("Submit gate accept: %v", err)
	}
	if next.Action != engine.ActionInstruct {
		t.Fatalf("expected batch instruct after gate approval, got %q", next.Action)
	}
	ids := taskActionIDs(next)
	if len(ids) != 2 {
		t.Fatalf("expected 2 task actions after gate approval, got %v", ids)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Tasks[0].Exec == nil || !pr.Tasks[0].Exec.PlanApproved {
		t.Fatal("expected task-1 PlanApproved persisted after gate accept")
	}
	if pr.Tasks[0].Exec.Plan == nil || pr.Tasks[0].Exec.Plan.Approach != "approach" {
		t.Fatalf("expected task-1 Plan persisted, got %+v", pr.Tasks[0].Exec.Plan)
	}
}

func TestDag_BlockedTask_NormalReport_Unblocks(t *testing.T) {
	root := t.TempDir()
	slug := "dag-unblock"
	ctx := seedDagSpec(t, root, slug, diamondTasks(), nil, 0)

	if _, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-1", Blocked: []string{"merge-conflict: a.go"}, Completed: []string{"impl"}})); err != nil {
		t.Fatalf("Submit blocked: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if !pr.Tasks[0].Blocked {
		t.Fatal("expected task-1 Blocked=true after blocked report")
	}
	if pr.Tasks[0].Exec != nil && pr.Tasks[0].Exec.Implemented {
		t.Fatal("blocked report must not advance exec state")
	}

	if _, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-1", Completed: []string{"impl"}})); err != nil {
		t.Fatalf("Submit recovery: %v", err)
	}

	pr, err = spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress after recovery: %v", err)
	}
	if pr.Tasks[0].Blocked {
		t.Fatal("expected task-1 unblocked after normal report")
	}
	if pr.Tasks[0].BlockedReason != "" {
		t.Fatalf("expected BlockedReason cleared, got %q", pr.Tasks[0].BlockedReason)
	}
	if pr.Tasks[0].Exec == nil || !pr.Tasks[0].Exec.Implemented {
		t.Fatal("substantive unblock report must be processed as a stage result")
	}

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	ids := taskActionIDs(action)
	if !slices.Contains(ids, "task-1") {
		t.Fatalf("expected task-1 back in the batch after unblock, got %v", ids)
	}
}

func TestDag_Deadlock_BlockedTaskReport_StillAccepted(t *testing.T) {
	root := t.TempDir()
	slug := "dag-deadlock-rescue"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: false, Blocked: true, BlockedReason: "missing schema"},
		{ID: "task-2", Title: "two", TDDEnabled: false, DependsOn: []string{"task-1"}},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	_, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-99", Completed: []string{"impl"}}))
	if err == nil || !strings.Contains(err.Error(), `unknown taskId "task-99"`) {
		t.Fatalf("expected routing error for invalid report while all blocked, got %v", err)
	}

	if _, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-1", Completed: []string{"impl"}})); err != nil {
		t.Fatalf("Submit rescue: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Tasks[0].Blocked {
		t.Fatal("expected task-1 unblocked by rescue report")
	}
	if pr.Tasks[0].Exec == nil || !pr.Tasks[0].Exec.Implemented {
		t.Fatal("substantive rescue report must be processed as a stage result")
	}

	next, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if next.Action == engine.ActionError {
		t.Fatalf("expected no deadlock after rescue, got %q", next.Instruction)
	}
	ids := taskActionIDs(next)
	if len(ids) != 1 || ids[0] != "task-1" {
		t.Fatalf("expected [task-1] ready after rescue, got %v", ids)
	}
}

func TestDag_VerifierStage_BlockedPassedReport_NotDone(t *testing.T) {
	root := t.TempDir()
	slug := "dag-verifier-blocked"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: false, Exec: &spec.ExecState{Implemented: true}},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	if _, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-1", Passed: true, Blocked: []string{"merge-conflict: a.go"}})); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Tasks[0].Done {
		t.Fatal("blocked verifier report must not complete the task")
	}
	if !pr.Tasks[0].Blocked {
		t.Fatal("expected task Blocked=true after blocked verifier report")
	}
	if pr.Tasks[0].Exec == nil || !pr.Tasks[0].Exec.Implemented {
		t.Fatal("blocked verifier report must not touch exec state")
	}
}

func TestDag_TDDVerifier_BlockedPassedReport_NotDone(t *testing.T) {
	root := t.TempDir()
	slug := "dag-tdd-verifier-blocked"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: true, Exec: &spec.ExecState{TDDCycle: cycleGreen, Implemented: true}},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	if _, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-1", Passed: true, Blocked: []string{"coverage tool unavailable"}})); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Tasks[0].Done {
		t.Fatal("blocked TDD verifier report must not complete the task")
	}
	if !pr.Tasks[0].Blocked {
		t.Fatal("expected task Blocked=true after blocked TDD verifier report")
	}
	if pr.Tasks[0].Exec == nil || pr.Tasks[0].Exec.TDDCycle != cycleGreen || !pr.Tasks[0].Exec.Implemented {
		t.Fatalf("blocked TDD verifier report must not advance the cycle, got %+v", pr.Tasks[0].Exec)
	}
}

func TestDag_GatePlusBatch_TwoGates_OnlyFirstAsked(t *testing.T) {
	root := t.TempDir()
	slug := "dag-two-gates"
	tasks := []spec.Task{
		{ID: "task-1", Title: "imp1", TDDEnabled: false, Important: true},
		{ID: "task-2", Title: "imp2", TDDEnabled: false, Important: true},
		{ID: "task-3", Title: "normal", TDDEnabled: false},
	}
	ctx := seedDagSpec(t, root, slug, tasks, func(s *spec.Settings) {
		s.ImportantTaskGateEnabled = true
	}, 0)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if action.Action != engine.ActionAsk {
		t.Fatalf("expected ActionAsk, got %q", action.Action)
	}
	ids := taskActionIDs(action)
	if len(ids) != 2 || ids[0] != "task-1" || ids[1] != "task-3" {
		t.Fatalf("expected Tasks [task-1(gate) task-3], got %v", ids)
	}
	if action.Tasks[0].Stage != StageIDGate {
		t.Fatalf("expected first entry stage gate, got %q", action.Tasks[0].Stage)
	}
	if action.Tasks[1].Stage != StageIDExecutor {
		t.Fatalf("expected second entry stage executor, got %q", action.Tasks[1].Stage)
	}
	if !strings.Contains(action.Instruction, "While the plan gate is pending") {
		t.Fatalf("expected parallel-dispatch note in Instruction, got %q", action.Instruction)
	}
}

func TestDag_Unblock_RedStage_BareReport_ClearsBlocked(t *testing.T) {
	root := t.TempDir()
	slug := "dag-unblock-red"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: true, Blocked: true, BlockedReason: "missing schema", Exec: &spec.ExecState{TDDCycle: cycleRed}},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	if _, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-1"})); err != nil {
		t.Fatalf("Submit unblock: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Tasks[0].Blocked {
		t.Fatal("expected task-1 unblocked by bare report")
	}
	if pr.Tasks[0].BlockedReason != "" {
		t.Fatalf("expected BlockedReason cleared, got %q", pr.Tasks[0].BlockedReason)
	}
	if pr.Tasks[0].Exec == nil || pr.Tasks[0].Exec.TDDCycle != cycleRed {
		t.Fatalf("expected TDDCycle to stay red, got %+v", pr.Tasks[0].Exec)
	}

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if len(action.Tasks) != 1 || action.Tasks[0].TaskID != "task-1" || action.Tasks[0].Stage != StageIDRed {
		t.Fatalf("expected task-1 re-emitted at red stage, got %+v", action.Tasks)
	}
}

func TestDag_Next_NoApplicableStage_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "dag-no-stage"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: true, Exec: &spec.ExecState{}},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if action.Action != engine.ActionError {
		t.Fatalf("expected ActionError when no stage applies, got %+v", action)
	}
	if !strings.Contains(action.Instruction, "task-1") {
		t.Fatalf("expected stuck task id in instruction, got %q", action.Instruction)
	}
}

func TestDag_Submit_NoApplicableStage_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "dag-no-stage-submit"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: true, Exec: &spec.ExecState{}},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	_, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-1"}))
	if err == nil || !strings.Contains(err.Error(), "no applicable stage for task task-1") {
		t.Fatalf("expected no-applicable-stage error, got %v", err)
	}
}

func TestDag_SkipVerifier_BlockedWithCompletedReport_NotDone(t *testing.T) {
	root := t.TempDir()
	slug := "dag-skip-blocked-completed"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: false},
		{ID: "task-2", Title: "two", TDDEnabled: false, DependsOn: []string{"task-1"}},
	}
	ctx := seedDagSpec(t, root, slug, tasks, func(s *spec.Settings) {
		s.SkipVerifierEnabled = true
	}, 0)

	if _, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-1", Completed: []string{"AC1"}, Blocked: []string{"merge-conflict: foo.go"}})); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Tasks[0].Done {
		t.Fatal("blocked report must not complete the task even with completed entries")
	}
	if !pr.Tasks[0].Blocked {
		t.Fatal("expected task-1 Blocked=true")
	}
	if pr.Tasks[0].BlockedReason != "merge-conflict: foo.go" {
		t.Fatalf("expected BlockedReason preserved, got %q", pr.Tasks[0].BlockedReason)
	}
	if len(spec.ReadyTaskIndices(pr.Tasks)) != 0 {
		t.Fatal("dependent task must not become ready while task-1 is blocked")
	}
}

func TestDag_Unblock_VerifierStage_BareReport_KeepsImplemented(t *testing.T) {
	root := t.TempDir()
	slug := "dag-unblock-verifier"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: false, Blocked: true, BlockedReason: "coverage tool unavailable", Exec: &spec.ExecState{Implemented: true}},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	if _, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-1"})); err != nil {
		t.Fatalf("Submit unblock: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Tasks[0].Blocked {
		t.Fatal("expected task-1 unblocked by bare report")
	}
	if pr.Tasks[0].Exec == nil || !pr.Tasks[0].Exec.Implemented {
		t.Fatalf("unblock report must not reset Implemented, got %+v", pr.Tasks[0].Exec)
	}

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if len(action.Tasks) != 1 || action.Tasks[0].TaskID != "task-1" || action.Tasks[0].Stage != StageIDVerifier {
		t.Fatalf("expected task-1 re-emitted at verifier stage, got %+v", action.Tasks)
	}
}

func TestDag_RedStage_BlockedReportWithTests_NoErrorCycleStaysRed(t *testing.T) {
	root := t.TempDir()
	slug := "dag-red-blocked-tests"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: true, Exec: &spec.ExecState{TDDCycle: cycleRed}},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	if _, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-1", TestsWritten: []string{"a_test.go"}, Blocked: []string{"missing schema"}})); err != nil {
		t.Fatalf("Submit blocked red report: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if !pr.Tasks[0].Blocked {
		t.Fatal("expected task-1 Blocked=true")
	}
	if pr.Tasks[0].Exec == nil || pr.Tasks[0].Exec.TDDCycle != cycleRed {
		t.Fatalf("blocked red report must not advance the cycle, got %+v", pr.Tasks[0].Exec)
	}
}

func TestDag_RedStage_BlockedReportWithTraceability_CycleStaysRed(t *testing.T) {
	root := t.TempDir()
	slug := "dag-red-blocked-trace"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: true, Exec: &spec.ExecState{TDDCycle: cycleRed}},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	report := StageReport{
		TaskID:       "task-1",
		TestsWritten: []string{"a_test.go"},
		Traceability: []TraceReportEntry{
			{TestFilePath: "a_test.go", FunctionName: "TestA", TaskID: "task-1", AC: []string{"ac1"}},
		},
		Blocked: []string{"cannot cover AC-3: missing fixture"},
	}
	if _, err := ctx.Submit(marshalStageReport(t, report)); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if !pr.Tasks[0].Blocked {
		t.Fatal("expected task-1 Blocked=true")
	}
	if pr.Tasks[0].Exec == nil || pr.Tasks[0].Exec.TDDCycle != cycleRed {
		t.Fatalf("partially-blocked red report must not advance the cycle, got %+v", pr.Tasks[0].Exec)
	}
}

func TestDag_RedStage_EmptyReport_ReturnsTraceabilityError(t *testing.T) {
	root := t.TempDir()
	slug := "dag-red-empty"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: true, Exec: &spec.ExecState{TDDCycle: cycleRed}},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	_, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-1", Passed: true}))
	if err == nil || !strings.Contains(err.Error(), "traceability is required") {
		t.Fatalf("expected traceability-required error for empty red report, got %v", err)
	}
}

func TestDag_WorktreePath_UsesForwardSlashes(t *testing.T) {
	root := t.TempDir()
	slug := "dag-worktree-path"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: false},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if len(action.Tasks) != 1 || action.Tasks[0].Worktree == nil {
		t.Fatalf("expected one task with worktree, got %+v", action.Tasks)
	}
	want := ".tddmaster/worktrees/" + slug + "/task-1"
	if action.Tasks[0].Worktree.Path != want {
		t.Fatalf("worktree path: got %q, want %q", action.Tasks[0].Worktree.Path, want)
	}
}

func TestDag_Unblock_DoneReport_CompletesTask(t *testing.T) {
	root := t.TempDir()
	slug := "dag-unblock-done"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: false, Blocked: true, BlockedReason: "merge-conflict: a.go", Exec: &spec.ExecState{Implemented: true}},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	if _, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-1", Passed: true})); err != nil {
		t.Fatalf("Submit done report after unblock: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Tasks[0].Blocked {
		t.Fatal("expected task-1 unblocked by done report")
	}
	if !pr.Tasks[0].Done {
		t.Fatal("done report submitted to unblock must complete the task, not be swallowed")
	}
}

func TestDag_BlockedVerifierReport_PersistsFailureContext(t *testing.T) {
	root := t.TempDir()
	slug := "dag-blocked-failure-ctx"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: false, Exec: &spec.ExecState{Implemented: true}},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	report := StageReport{
		TaskID:    "task-1",
		Passed:    false,
		FailedACs: []string{"ac-1: rejection missing"},
		Blocked:   []string{"merge-conflict: a.go"},
	}
	if _, err := ctx.Submit(marshalStageReport(t, report)); err != nil {
		t.Fatalf("Submit blocked verifier report: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if !pr.Tasks[0].Blocked {
		t.Fatal("expected task-1 Blocked=true")
	}
	if !slices.Contains(pr.Tasks[0].FailedACReasons, "ac-1: rejection missing") {
		t.Fatalf("blocked report must persist FailedACReasons, got %v", pr.Tasks[0].FailedACReasons)
	}
	if pr.Tasks[0].Exec == nil || !slices.Contains(pr.Tasks[0].Exec.LastFailedACs, "ac-1: rejection missing") {
		t.Fatalf("blocked report must persist LastFailedACs for re-dispatch context, got %+v", pr.Tasks[0].Exec)
	}
	if !pr.Tasks[0].Exec.Implemented {
		t.Fatal("blocked report must not advance exec state")
	}
}

func TestDag_BlockedGateAnswer_RecordsPlanApproval(t *testing.T) {
	root := t.TempDir()
	slug := "dag-blocked-gate-answer"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: false, Important: true},
	}
	ctx := seedDagSpec(t, root, slug, tasks, func(s *spec.Settings) { s.ImportantTaskGateEnabled = true }, 0)

	plan := &spec.TaskPlan{TaskID: "task-1", Approach: "do X", TouchedFiles: []string{"a.go"}}
	report := StageReport{TaskID: "task-1", Accepted: true, Plan: plan, Blocked: []string{"merge-conflict: a.go"}}
	if _, err := ctx.Submit(marshalStageReport(t, report)); err != nil {
		t.Fatalf("Submit blocked gate answer: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if !pr.Tasks[0].Blocked {
		t.Fatal("expected task-1 Blocked=true")
	}
	if pr.Tasks[0].Exec == nil || !pr.Tasks[0].Exec.PlanApproved {
		t.Fatalf("blocked gate answer must record plan approval, got %+v", pr.Tasks[0].Exec)
	}
	if pr.Tasks[0].Exec.Plan == nil || pr.Tasks[0].Exec.Plan.Approach != "do X" {
		t.Fatalf("blocked gate answer must persist the approved plan, got %+v", pr.Tasks[0].Exec.Plan)
	}
}

func TestDag_Unblock_VerifierFailureReport_Processed(t *testing.T) {
	root := t.TempDir()
	slug := "dag-unblock-verify-fail"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: false, Blocked: true, BlockedReason: "merge-conflict: a.go", Exec: &spec.ExecState{Implemented: true}},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	if _, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-1", Passed: false, Phase: "verify"})); err != nil {
		t.Fatalf("Submit unblock verifier failure report: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Tasks[0].Blocked {
		t.Fatal("expected task-1 unblocked")
	}
	if pr.Tasks[0].Exec == nil || pr.Tasks[0].Exec.Implemented {
		t.Fatalf("verifier failure report must be processed as the verify result, got %+v", pr.Tasks[0].Exec)
	}
}

func TestDag_BlockedEmptyVerifierReport_PreservesFailureContext(t *testing.T) {
	root := t.TempDir()
	slug := "dag-blocked-keeps-failure-ctx"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: false, Exec: &spec.ExecState{Implemented: true, LastFailedACs: []string{"ac-1: rejection missing"}}},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	if _, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-1", Blocked: []string{"merge-conflict: a.go"}})); err != nil {
		t.Fatalf("Submit blocked report: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if !pr.Tasks[0].Blocked {
		t.Fatal("expected task-1 Blocked=true")
	}
	if pr.Tasks[0].Exec == nil || !slices.Contains(pr.Tasks[0].Exec.LastFailedACs, "ac-1: rejection missing") {
		t.Fatalf("blocked report without failure data must not erase LastFailedACs, got %+v", pr.Tasks[0].Exec)
	}
}

func TestDag_DuplicateGateAnswer_DoesNotAdvanceExecutor(t *testing.T) {
	root := t.TempDir()
	slug := "dag-duplicate-gate-answer"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: false, Important: true, Exec: &spec.ExecState{PlanApproved: true}},
	}
	ctx := seedDagSpec(t, root, slug, tasks, func(s *spec.Settings) { s.ImportantTaskGateEnabled = true }, 0)

	plan := &spec.TaskPlan{TaskID: "task-1", Approach: "do X", TouchedFiles: []string{"a.go"}}
	if _, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "task-1", Accepted: true, Plan: plan})); err != nil {
		t.Fatalf("Submit duplicate gate answer: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Tasks[0].Exec == nil || pr.Tasks[0].Exec.Implemented {
		t.Fatalf("duplicate gate answer must not mark the executor stage implemented, got %+v", pr.Tasks[0].Exec)
	}
	if pr.Tasks[0].Done {
		t.Fatal("duplicate gate answer must not complete the task")
	}
}

func TestDag_Next_StuckTaskWithSiblings_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "dag-stuck-sibling"
	tasks := []spec.Task{
		{ID: "task-1", Title: "one", TDDEnabled: false},
		{ID: "task-2", Title: "two", TDDEnabled: true, Exec: &spec.ExecState{}},
	}
	ctx := seedDagSpec(t, root, slug, tasks, nil, 0)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if action.Action != engine.ActionError {
		t.Fatalf("a stuck ready task must surface an error even when siblings have stages, got %+v", action)
	}
	if !strings.Contains(action.Instruction, "task-2") {
		t.Fatalf("expected stuck task-2 named in instruction, got %q", action.Instruction)
	}
}
