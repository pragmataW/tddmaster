package phases

import (
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func seedRefinementSpec(t *testing.T, root, slug string, tasks []spec.Task) {
	t.Helper()
	writeDiscoveryManifest(t, root)
	state := spec.State{
		Version: 1,
		Slug:    slug,
		Phase:   "refinement",
		Answers: map[string][]spec.Answer{},
	}
	if err := spec.SaveState(root, slug, state); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	if err := spec.SaveSettings(root, slug, spec.DefaultSettings()); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	pr := spec.Progress{Spec: slug, Status: spec.StatusDraft, Tasks: tasks}
	if err := spec.SaveProgress(root, slug, pr); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
}

func buildRefinementCtx(t *testing.T, root, slug string) *engine.Context {
	t.Helper()
	defs := []engine.PhaseDef{{ID: "refinement", Driver: RefinementDriver()}}
	ctx, err := engine.Build(root, slug, defs)
	if err != nil {
		t.Fatalf("engine.Build: %v", err)
	}
	return ctx
}

func twoSeededTasks() []spec.Task {
	return []spec.Task{
		{ID: "task-1", Title: "Alpha", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac1"}, {ID: "ac-2", Then: "ac2"}}, TDDEnabled: true, Important: true},
		{ID: "task-2", Title: "Beta", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac3"}}, TDDEnabled: false, Important: false},
	}
}

func TestRenderTaskList_TDDAndImportant_ContainsTaskLine(t *testing.T) {
	tasks := []spec.Task{
		{ID: "task-1", Title: "Alpha", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac1"}}, TDDEnabled: true, Important: true},
	}
	out := RenderTaskList(tasks)
	if !strings.Contains(out, "- task-1: Alpha") {
		t.Errorf("output missing task line; got: %s", out)
	}
}

func TestRenderTaskList_TDDAndImportant_ContainsTDDTag(t *testing.T) {
	tasks := []spec.Task{
		{ID: "task-1", Title: "Alpha", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac1"}}, TDDEnabled: true, Important: true},
	}
	out := RenderTaskList(tasks)
	if !strings.Contains(out, "(TDD)") {
		t.Errorf("output missing (TDD) tag; got: %s", out)
	}
}

func TestRenderTaskList_TDDAndImportant_ContainsImportantTag(t *testing.T) {
	tasks := []spec.Task{
		{ID: "task-1", Title: "Alpha", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac1"}}, TDDEnabled: true, Important: true},
	}
	out := RenderTaskList(tasks)
	if !strings.Contains(out, "(important)") {
		t.Errorf("output missing (important) tag; got: %s", out)
	}
}

func TestRenderTaskList_TDDAndImportant_TDDBeforeImportant(t *testing.T) {
	tasks := []spec.Task{
		{ID: "task-1", Title: "Alpha", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac1"}}, TDDEnabled: true, Important: true},
	}
	out := RenderTaskList(tasks)
	tddIdx := strings.Index(out, "(TDD)")
	impIdx := strings.Index(out, "(important)")
	if tddIdx == -1 || impIdx == -1 {
		t.Fatalf("tags missing; got: %s", out)
	}
	if tddIdx >= impIdx {
		t.Errorf("(TDD) should appear before (important); got: %s", out)
	}
}

func TestRenderTaskList_TDDAndImportant_ContainsACSubBullet(t *testing.T) {
	tasks := []spec.Task{
		{ID: "task-1", Title: "Alpha", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac1"}}, TDDEnabled: true, Important: true},
	}
	out := RenderTaskList(tasks)
	if !strings.Contains(out, "  - [ac-1] THEN ac1") {
		t.Errorf("output missing criterion sub-bullet; got: %s", out)
	}
}

func TestRenderTaskList_SecondTask_NoTagsPresent(t *testing.T) {
	tasks := []spec.Task{
		{ID: "task-2", Title: "Beta", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac3"}}, TDDEnabled: false, Important: false},
	}
	out := RenderTaskList(tasks)
	if !strings.Contains(out, "- task-2: Beta") {
		t.Errorf("output missing task-2 line; got: %s", out)
	}
	if strings.Contains(out, "(TDD)") {
		t.Errorf("output should not contain (TDD) for non-TDD task; got: %s", out)
	}
	if strings.Contains(out, "(important)") {
		t.Errorf("output should not contain (important) for non-important task; got: %s", out)
	}
}

func TestRenderTaskList_TDDOnly_ContainsTDDNotImportant(t *testing.T) {
	tasks := []spec.Task{
		{ID: "task-1", Title: "Gamma", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac1"}}, TDDEnabled: true, Important: false},
	}
	out := RenderTaskList(tasks)
	if !strings.Contains(out, "(TDD)") {
		t.Errorf("output missing (TDD); got: %s", out)
	}
	if strings.Contains(out, "(important)") {
		t.Errorf("output should not contain (important); got: %s", out)
	}
}

func TestRenderTaskList_ImportantOnly_ContainsImportantNotTDD(t *testing.T) {
	tasks := []spec.Task{
		{ID: "task-1", Title: "Delta", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac1"}}, TDDEnabled: false, Important: true},
	}
	out := RenderTaskList(tasks)
	if strings.Contains(out, "(TDD)") {
		t.Errorf("output should not contain (TDD); got: %s", out)
	}
	if !strings.Contains(out, "(important)") {
		t.Errorf("output missing (important); got: %s", out)
	}
}

func TestRenderTaskList_BothTagsCombined_ContainsBoth(t *testing.T) {
	tasks := []spec.Task{
		{ID: "task-1", Title: "Both", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac1"}}, TDDEnabled: true, Important: true},
	}
	out := RenderTaskList(tasks)
	if !strings.Contains(out, "(TDD)") {
		t.Errorf("output missing (TDD); got: %s", out)
	}
	if !strings.Contains(out, "(important)") {
		t.Errorf("output missing (important); got: %s", out)
	}
}

func TestRenderTaskList_NeitherTag_ContainsNeither(t *testing.T) {
	tasks := []spec.Task{
		{ID: "task-1", Title: "Neither", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac1"}}, TDDEnabled: false, Important: false},
	}
	out := RenderTaskList(tasks)
	if strings.Contains(out, "(TDD)") {
		t.Errorf("output should not contain (TDD); got: %s", out)
	}
	if strings.Contains(out, "(important)") {
		t.Errorf("output should not contain (important); got: %s", out)
	}
}

func TestRenderTaskList_MultipleAC_AllSubBulletsPresent(t *testing.T) {
	tasks := []spec.Task{
		{ID: "task-1", Title: "Multi", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac1"}, {ID: "ac-2", Then: "ac2"}}, TDDEnabled: false, Important: false},
	}
	out := RenderTaskList(tasks)
	if !strings.Contains(out, "  - [ac-1] THEN ac1") {
		t.Errorf("output missing criterion sub-bullet ac1; got: %s", out)
	}
	if !strings.Contains(out, "  - [ac-2] THEN ac2") {
		t.Errorf("output missing criterion sub-bullet ac2; got: %s", out)
	}
}

func TestRenderTaskList_TwoTasks_BothLinesPresent(t *testing.T) {
	tasks := twoSeededTasks()
	out := RenderTaskList(tasks)
	if !strings.Contains(out, "- task-1: Alpha") {
		t.Errorf("output missing task-1 line; got: %s", out)
	}
	if !strings.Contains(out, "- task-2: Beta") {
		t.Errorf("output missing task-2 line; got: %s", out)
	}
}

func TestRenderTaskList_EmptyTasks_ContainsNoTasksMessage(t *testing.T) {
	out := RenderTaskList([]spec.Task{})
	if len(out) == 0 {
		t.Fatal("output is empty, want non-empty message")
	}
	if !strings.Contains(out, "No tasks") {
		t.Errorf("output missing 'No tasks'; got: %s", out)
	}
}

func TestRefinementDriver_Next_ActionIsAsk(t *testing.T) {
	root := t.TempDir()
	slug := "ref-ask"
	seedRefinementSpec(t, root, slug, twoSeededTasks())
	ctx := buildRefinementCtx(t, root, slug)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next(): %v", err)
	}
	if action.Action != engine.ActionAsk {
		t.Errorf("action.Action = %q, want %q", action.Action, engine.ActionAsk)
	}
}

func TestRefinementDriver_Next_FormatIsFlag(t *testing.T) {
	root := t.TempDir()
	slug := "ref-flag"
	seedRefinementSpec(t, root, slug, twoSeededTasks())
	ctx := buildRefinementCtx(t, root, slug)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next(): %v", err)
	}
	if action.ExpectedInput.Format != engine.FormatFlag {
		t.Errorf("Format = %q, want %q", action.ExpectedInput.Format, engine.FormatFlag)
	}
}

func TestRefinementDriver_Next_InstructionContainsKeyRefinePrompt(t *testing.T) {
	root := t.TempDir()
	slug := "ref-instr"
	seedRefinementSpec(t, root, slug, twoSeededTasks())
	ctx := buildRefinementCtx(t, root, slug)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next(): %v", err)
	}
	want, _ := promptregistry.Instruction(promptregistry.KeyRefinePrompt)
	if !strings.Contains(action.Instruction, want) {
		t.Errorf("Instruction missing KeyRefinePrompt value; got: %s", action.Instruction)
	}
}

func TestRefinementDriver_Next_InstructionContainsTaskTitle(t *testing.T) {
	root := t.TempDir()
	slug := "ref-title"
	seedRefinementSpec(t, root, slug, twoSeededTasks())
	ctx := buildRefinementCtx(t, root, slug)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next(): %v", err)
	}
	if !strings.Contains(action.Instruction, "Alpha") {
		t.Errorf("Instruction missing task title 'Alpha'; got: %s", action.Instruction)
	}
}

func TestRefinementPrompt_ExampleIsApproveFlag(t *testing.T) {
	root := t.TempDir()
	slug := "ref-example-approve"
	seedRefinementSpec(t, root, slug, twoSeededTasks())
	ctx := buildRefinementCtx(t, root, slug)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next(): %v", err)
	}
	if action.ExpectedInput.Format != engine.FormatFlag {
		t.Errorf("Format = %q, want %q", action.ExpectedInput.Format, engine.FormatFlag)
	}
	if action.ExpectedInput.Example != "approve" {
		t.Errorf("Example = %q, want %q", action.ExpectedInput.Example, "approve")
	}
}

func TestRefinementPrompt_InstructionContainsRefineJSONExample(t *testing.T) {
	root := t.TempDir()
	slug := "ref-instr-json"
	seedRefinementSpec(t, root, slug, twoSeededTasks())
	ctx := buildRefinementCtx(t, root, slug)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next(): %v", err)
	}
	instr := action.Instruction
	if !strings.Contains(instr, "update") && !strings.Contains(instr, "add") && !strings.Contains(instr, "remove") {
		t.Errorf("Instruction missing refine JSON example ('update'/'add'/'remove'); got: %s", instr)
	}
	if !strings.Contains(instr, "edgeCases") {
		t.Errorf("Instruction missing 'edgeCases' in refine JSON example; got: %s", instr)
	}
}

func TestRefinementDriver_SubmitApprove_PhaseDone(t *testing.T) {
	root := t.TempDir()
	slug := "ref-approve"
	seedRefinementSpec(t, root, slug, twoSeededTasks())
	ctx := buildRefinementCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	if _, err := ctx.Submit([]byte("approve")); err != nil {
		t.Fatalf("Submit approve: %v", err)
	}

	state, err := spec.LoadState(root, slug)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.Phase == "refinement" {
		t.Error("phase still 'refinement' after approve, want phase advanced")
	}
}

func TestRefinementDriver_SubmitApprove_NextReturnsDone(t *testing.T) {
	root := t.TempDir()
	slug := "ref-approve-next"
	seedRefinementSpec(t, root, slug, twoSeededTasks())
	ctx := buildRefinementCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("First Next(): %v", err)
	}
	if _, err := ctx.Submit([]byte("approve")); err != nil {
		t.Fatalf("Submit approve: %v", err)
	}

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Second Next() after approve: %v", err)
	}
	if action.Action != "" && action.Action != engine.ActionTerminal {
		t.Errorf("after approve, Next action = %q, want empty or terminal (phase done)", action.Action)
	}
}

func TestRefinementDriver_SubmitDone_PhaseDone(t *testing.T) {
	root := t.TempDir()
	slug := "ref-done"
	seedRefinementSpec(t, root, slug, twoSeededTasks())
	ctx := buildRefinementCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	if _, err := ctx.Submit([]byte("done")); err != nil {
		t.Fatalf("Submit done: %v", err)
	}

	state, err := spec.LoadState(root, slug)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.Phase == "refinement" {
		t.Error("phase still 'refinement' after done, want phase advanced")
	}
}

func TestRefinementDriver_SubmitNope_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "ref-nope"
	seedRefinementSpec(t, root, slug, twoSeededTasks())
	ctx := buildRefinementCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	_, err := ctx.Submit([]byte("nope"))
	if err == nil {
		t.Fatal("Submit(nope) = nil, want non-nil error")
	}
}

func TestRefinementDriver_SubmitNope_NotRecorded(t *testing.T) {
	root := t.TempDir()
	slug := "ref-nope-ans"
	seedRefinementSpec(t, root, slug, twoSeededTasks())
	ctx := buildRefinementCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	ctx.Submit([]byte("nope"))

	ctx2 := buildRefinementCtx(t, root, slug)
	if ctx2.HasAnswer("refinement_approved") {
		t.Error("refinement_approved recorded after nope, want not recorded")
	}
}

func TestRefinementDriver_EmptyTasks_InstructionContainsNoTasks(t *testing.T) {
	root := t.TempDir()
	slug := "ref-empty-tasks"
	seedRefinementSpec(t, root, slug, []spec.Task{})
	ctx := buildRefinementCtx(t, root, slug)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next(): %v", err)
	}
	if !strings.Contains(action.Instruction, "No tasks") {
		t.Errorf("Instruction missing 'No tasks' for empty task list; got: %s", action.Instruction)
	}
}

func TestRefinementDriver_EmptyTasks_InstructionContainsKeyRefinePrompt(t *testing.T) {
	root := t.TempDir()
	slug := "ref-empty-instr"
	seedRefinementSpec(t, root, slug, []spec.Task{})
	ctx := buildRefinementCtx(t, root, slug)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next(): %v", err)
	}
	want, _ := promptregistry.Instruction(promptregistry.KeyRefinePrompt)
	if !strings.Contains(action.Instruction, want) {
		t.Errorf("Instruction missing KeyRefinePrompt value for empty task list; got: %s", action.Instruction)
	}
}

func TestRefinementDriver_EmptyTasks_SubmitApproveWorks(t *testing.T) {
	root := t.TempDir()
	slug := "ref-empty-approve"
	seedRefinementSpec(t, root, slug, []spec.Task{})
	ctx := buildRefinementCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	if _, err := ctx.Submit([]byte("approve")); err != nil {
		t.Errorf("Submit approve on empty-task spec: %v", err)
	}

	state, err := spec.LoadState(root, slug)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.Phase == "refinement" {
		t.Error("phase still 'refinement' after approve on empty-task spec, want phase advanced")
	}
}

func TestRenderTaskList_DependsOn_AppendsSuffix(t *testing.T) {
	tasks := []spec.Task{
		{ID: "task-1", Title: "Alpha", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac1"}}},
		{ID: "task-2", Title: "Beta", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac2"}}},
		{ID: "task-3", Title: "Gamma", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac3"}}, DependsOn: []string{"task-1", "task-2"}},
	}
	out := RenderTaskList(tasks)
	if !strings.Contains(out, "- task-3: Gamma (depends on: task-1, task-2)") {
		t.Errorf("output missing depends-on suffix; got: %s", out)
	}
}

func TestRenderTaskList_NoDependsOn_NoSuffix(t *testing.T) {
	tasks := []spec.Task{
		{ID: "task-1", Title: "Alpha", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac1"}}},
	}
	out := RenderTaskList(tasks)
	if strings.Contains(out, "depends on") {
		t.Errorf("output should not contain depends-on suffix for independent task; got: %s", out)
	}
}

func TestRenderTaskList_DependsOn_AfterTags(t *testing.T) {
	tasks := []spec.Task{
		{ID: "task-1", Title: "Alpha", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac1"}}},
		{ID: "task-2", Title: "Beta", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac2"}}, TDDEnabled: true, Important: true, DependsOn: []string{"task-1"}},
	}
	out := RenderTaskList(tasks)
	if !strings.Contains(out, "- task-2: Beta (TDD) (important) (depends on: task-1)") {
		t.Errorf("output missing full tag ordering with depends-on last; got: %s", out)
	}
}

func TestRefinementDriver_SubmitApprove_InvalidDAG_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "ref-approve-bad-dag"
	tasks := []spec.Task{
		{ID: "task-1", Title: "Alpha", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac1"}}, DependsOn: []string{"task-9"}},
	}
	seedRefinementSpec(t, root, slug, tasks)
	ctx := buildRefinementCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	_, err := ctx.Submit([]byte("approve"))
	if err == nil {
		t.Fatal("Submit(approve) = nil error, want DAG validation error for unknown dependency")
	}
	if !strings.Contains(err.Error(), "task-9") {
		t.Errorf("error %q missing invalid dependency id task-9", err.Error())
	}

	state, loadErr := spec.LoadState(root, slug)
	if loadErr != nil {
		t.Fatalf("LoadState: %v", loadErr)
	}
	if state.Phase != "refinement" {
		t.Errorf("phase = %q, want 'refinement' — approve must not advance on invalid DAG", state.Phase)
	}
}

func TestRefinementDriver_SubmitApprove_CycleDAG_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "ref-approve-cycle"
	tasks := []spec.Task{
		{ID: "task-1", Title: "Alpha", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac1"}}, DependsOn: []string{"task-2"}},
		{ID: "task-2", Title: "Beta", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac2"}}, DependsOn: []string{"task-1"}},
	}
	seedRefinementSpec(t, root, slug, tasks)
	ctx := buildRefinementCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	_, err := ctx.Submit([]byte("approve"))
	if err == nil {
		t.Fatal("Submit(approve) = nil error, want DAG validation error for cycle")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("error %q missing 'cycle'", err.Error())
	}
}

func TestRefinementDriver_SubmitApprove_ValidDAG_Advances(t *testing.T) {
	root := t.TempDir()
	slug := "ref-approve-good-dag"
	tasks := []spec.Task{
		{ID: "task-1", Title: "Alpha", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac1"}}},
		{ID: "task-2", Title: "Beta", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac2"}}, DependsOn: []string{"task-1"}},
	}
	seedRefinementSpec(t, root, slug, tasks)
	ctx := buildRefinementCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	if _, err := ctx.Submit([]byte("approve")); err != nil {
		t.Fatalf("Submit approve with valid DAG: %v", err)
	}

	state, loadErr := spec.LoadState(root, slug)
	if loadErr != nil {
		t.Fatalf("LoadState: %v", loadErr)
	}
	if state.Phase == "refinement" {
		t.Error("phase still 'refinement' after approve with valid DAG, want phase advanced")
	}
}
