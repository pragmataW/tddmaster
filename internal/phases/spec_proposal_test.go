package phases

import (
	"os"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func seedSpecProposalSpec(t *testing.T, root, slug string) {
	t.Helper()
	writeDiscoveryManifest(t, root)
	state := spec.State{
		Version: 1,
		Slug:    slug,
		Phase:   "spec-proposal",
		Answers: map[string][]spec.Answer{},
	}
	if err := spec.SaveState(root, slug, state); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	if err := spec.SaveSettings(root, slug, spec.DefaultSettings()); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	pr := spec.Progress{Spec: slug, Status: spec.StatusDraft, Tasks: []spec.Task{}}
	if err := spec.SaveProgress(root, slug, pr); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
}

func buildSpecProposalCtx(t *testing.T, root, slug string) *engine.Context {
	t.Helper()
	defs := []engine.PhaseDef{{ID: "spec-proposal", Driver: SpecProposalDriver()}}
	ctx, err := engine.Build(root, slug, defs)
	if err != nil {
		t.Fatalf("engine.Build: %v", err)
	}
	return ctx
}

func TestBuildTasksFromGen_ValidPayload_ReturnsTasks(t *testing.T) {
	p := TaskGenPayload{
		Tasks: []TaskGenItem{
			{Title: "T1", Criteria: []spec.Criterion{{Then: "a1"}}, LinkedEdgeCases: []string{"ec1"}},
			{Title: "T2", Criteria: []spec.Criterion{{Then: "a2"}, {Then: "a3"}}},
		},
	}
	tasks, err := BuildTasksFromGen(p, true, nil)
	if err != nil {
		t.Fatalf("BuildTasksFromGen returned unexpected error: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestBuildTasksFromGen_IDsAreOneBased(t *testing.T) {
	p := TaskGenPayload{
		Tasks: []TaskGenItem{
			{Title: "T1", Criteria: []spec.Criterion{{Then: "a1"}}},
			{Title: "T2", Criteria: []spec.Criterion{{Then: "a2"}}},
			{Title: "T3", Criteria: []spec.Criterion{{Then: "a3"}}},
		},
	}
	tasks, err := BuildTasksFromGen(p, false, nil)
	if err != nil {
		t.Fatalf("BuildTasksFromGen: %v", err)
	}
	wantIDs := []string{"task-1", "task-2", "task-3"}
	for i, task := range tasks {
		if task.ID != wantIDs[i] {
			t.Errorf("tasks[%d].ID = %q, want %q", i, task.ID, wantIDs[i])
		}
	}
}

func TestBuildTasksFromGen_TitleCopied(t *testing.T) {
	p := TaskGenPayload{
		Tasks: []TaskGenItem{
			{Title: "My Title", Criteria: []spec.Criterion{{Then: "ac1"}}},
		},
	}
	tasks, err := BuildTasksFromGen(p, false, nil)
	if err != nil {
		t.Fatalf("BuildTasksFromGen: %v", err)
	}
	if tasks[0].Title != "My Title" {
		t.Errorf("Title = %q, want %q", tasks[0].Title, "My Title")
	}
}

func TestBuildTasksFromGen_ACCopied(t *testing.T) {
	p := TaskGenPayload{
		Tasks: []TaskGenItem{
			{Title: "T1", Criteria: []spec.Criterion{{Then: "ac1"}, {Then: "ac2"}}},
		},
	}
	tasks, err := BuildTasksFromGen(p, false, nil)
	if err != nil {
		t.Fatalf("BuildTasksFromGen: %v", err)
	}
	if len(tasks[0].Criteria) != 2 || tasks[0].Criteria[0].Then != "ac1" || tasks[0].Criteria[1].Then != "ac2" {
		t.Errorf("Criteria = %v, want [ac1 ac2]", tasks[0].Criteria)
	}
}

func TestBuildTasksFromGen_DoneIsFalse(t *testing.T) {
	p := TaskGenPayload{
		Tasks: []TaskGenItem{{Title: "T1", Criteria: []spec.Criterion{{Then: "a"}}}},
	}
	tasks, _ := BuildTasksFromGen(p, false, nil)
	if tasks[0].Done {
		t.Error("Done = true, want false")
	}
}

func TestBuildTasksFromGen_ImportantIsFalse(t *testing.T) {
	p := TaskGenPayload{
		Tasks: []TaskGenItem{{Title: "T1", Criteria: []spec.Criterion{{Then: "a"}}}},
	}
	tasks, _ := BuildTasksFromGen(p, false, nil)
	if tasks[0].Important {
		t.Error("Important = true, want false")
	}
}

func TestBuildTasksFromGen_TDDEnabled_MatchesDefault_True(t *testing.T) {
	p := TaskGenPayload{
		Tasks: []TaskGenItem{{Title: "T1", Criteria: []spec.Criterion{{Then: "a"}}}},
	}
	tasks, _ := BuildTasksFromGen(p, true, nil)
	if !tasks[0].TDDEnabled {
		t.Error("TDDEnabled = false, want true")
	}
}

func TestBuildTasksFromGen_TDDEnabled_MatchesDefault_False(t *testing.T) {
	p := TaskGenPayload{
		Tasks: []TaskGenItem{{Title: "T1", Criteria: []spec.Criterion{{Then: "a"}}}},
	}
	tasks, _ := BuildTasksFromGen(p, false, nil)
	if tasks[0].TDDEnabled {
		t.Error("TDDEnabled = true, want false")
	}
}

func TestBuildTasksFromGen_ZeroTasks_ReturnsError(t *testing.T) {
	p := TaskGenPayload{Tasks: []TaskGenItem{}}
	_, err := BuildTasksFromGen(p, false, nil)
	if err == nil {
		t.Fatal("expected error for zero tasks, got nil")
	}
}

func TestBuildTasksFromGen_EmptyTitle_ReturnsError(t *testing.T) {
	p := TaskGenPayload{
		Tasks: []TaskGenItem{
			{Title: "", Criteria: []spec.Criterion{{Then: "a1"}}},
		},
	}
	_, err := BuildTasksFromGen(p, false, nil)
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
}

func TestBuildTasksFromGen_EmptyAC_ReturnsError(t *testing.T) {
	p := TaskGenPayload{
		Tasks: []TaskGenItem{
			{Title: "T1", Criteria: []spec.Criterion{}},
		},
	}
	_, err := BuildTasksFromGen(p, false, nil)
	if err == nil {
		t.Fatal("expected error for empty AC, got nil")
	}
}

func TestSpecProposalDriver_FreshSpec_NextReturnsTaskGenPrompt(t *testing.T) {
	root := t.TempDir()
	slug := "sp-fresh"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	want, _ := promptregistry.Instruction(promptregistry.KeySpecTaskGen)
	if action.Instruction != want {
		t.Errorf("Instruction = %q, want KeySpecTaskGen value", action.Instruction)
	}
}

func TestSpecProposalDriver_FreshSpec_NextFormatIsJSON(t *testing.T) {
	root := t.TempDir()
	slug := "sp-format"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next(): %v", err)
	}
	if action.ExpectedInput.Format != engine.FormatJSON {
		t.Errorf("Format = %q, want %q", action.ExpectedInput.Format, engine.FormatJSON)
	}
}

func TestSpecProposalDriver_FreshSpec_NextExampleContainsTasks(t *testing.T) {
	root := t.TempDir()
	slug := "sp-example"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next(): %v", err)
	}
	if !strings.Contains(action.ExpectedInput.Example, "tasks") {
		t.Errorf("Example = %q, want it to contain 'tasks'", action.ExpectedInput.Example)
	}
}

func TestSpecProposalDriver_SubmitValidTaskGen_UpdatesProgressTasks(t *testing.T) {
	root := t.TempDir()
	slug := "sp-tasks"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}

	payload := `{"tasks":[{"title":"T1","criteria":[{"then":"a1"}],"linkedEdgeCases":["ec1"]},{"title":"T2","criteria":[{"then":"a2"},{"then":"a3"}]}]}`
	if _, err := ctx.Submit([]byte(payload)); err != nil {
		t.Fatalf("Submit valid task-gen JSON: %v", err)
	}

	ctx2 := buildSpecProposalCtx(t, root, slug)
	tasks := ctx2.Progress().Tasks
	if len(tasks) != 2 {
		t.Fatalf("Progress().Tasks len = %d, want 2", len(tasks))
	}
}

func TestSpecProposalDriver_SubmitValidTaskGen_FirstTaskID(t *testing.T) {
	root := t.TempDir()
	slug := "sp-id1"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	payload := `{"tasks":[{"title":"T1","criteria":[{"then":"a1"}],"linkedEdgeCases":["ec1"]},{"title":"T2","criteria":[{"then":"a2"},{"then":"a3"}]}]}`
	if _, err := ctx.Submit([]byte(payload)); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	ctx2 := buildSpecProposalCtx(t, root, slug)
	if ctx2.Progress().Tasks[0].ID != "task-1" {
		t.Errorf("Tasks[0].ID = %q, want task-1", ctx2.Progress().Tasks[0].ID)
	}
}

func TestSpecProposalDriver_SubmitValidTaskGen_FirstTaskTitle(t *testing.T) {
	root := t.TempDir()
	slug := "sp-title"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	payload := `{"tasks":[{"title":"T1","criteria":[{"then":"a1"}],"linkedEdgeCases":["ec1"]},{"title":"T2","criteria":[{"then":"a2"},{"then":"a3"}]}]}`
	if _, err := ctx.Submit([]byte(payload)); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	ctx2 := buildSpecProposalCtx(t, root, slug)
	if ctx2.Progress().Tasks[0].Title != "T1" {
		t.Errorf("Tasks[0].Title = %q, want T1", ctx2.Progress().Tasks[0].Title)
	}
}

func TestSpecProposalDriver_SubmitValidTaskGen_FirstTaskAC(t *testing.T) {
	root := t.TempDir()
	slug := "sp-ac"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	payload := `{"tasks":[{"title":"T1","criteria":[{"then":"a1"}],"linkedEdgeCases":["ec1"]},{"title":"T2","criteria":[{"then":"a2"},{"then":"a3"}]}]}`
	if _, err := ctx.Submit([]byte(payload)); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	ctx2 := buildSpecProposalCtx(t, root, slug)
	crit := ctx2.Progress().Tasks[0].Criteria
	if len(crit) != 1 || crit[0].Then != "a1" {
		t.Errorf("Tasks[0].Criteria = %v, want one criterion with Then a1", crit)
	}
}

func TestSpecProposalDriver_SubmitValidTaskGen_SecondTaskID(t *testing.T) {
	root := t.TempDir()
	slug := "sp-id2"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	payload := `{"tasks":[{"title":"T1","criteria":[{"then":"a1"}],"linkedEdgeCases":["ec1"]},{"title":"T2","criteria":[{"then":"a2"},{"then":"a3"}]}]}`
	if _, err := ctx.Submit([]byte(payload)); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	ctx2 := buildSpecProposalCtx(t, root, slug)
	if ctx2.Progress().Tasks[1].ID != "task-2" {
		t.Errorf("Tasks[1].ID = %q, want task-2", ctx2.Progress().Tasks[1].ID)
	}
}

func TestSpecProposalDriver_SubmitValidTaskGen_TDDEnabledMatchesSettings(t *testing.T) {
	root := t.TempDir()
	slug := "sp-tdd"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	defaultSettings := spec.DefaultSettings()

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	payload := `{"tasks":[{"title":"T1","criteria":[{"then":"a1"}],"linkedEdgeCases":["ec1"]},{"title":"T2","criteria":[{"then":"a2"},{"then":"a3"}]}]}`
	if _, err := ctx.Submit([]byte(payload)); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	ctx2 := buildSpecProposalCtx(t, root, slug)
	for i, task := range ctx2.Progress().Tasks {
		if task.TDDEnabled != defaultSettings.TDDEnabled {
			t.Errorf("Tasks[%d].TDDEnabled = %v, want %v", i, task.TDDEnabled, defaultSettings.TDDEnabled)
		}
	}
}

func TestSpecProposalDriver_SubmitValidTaskGen_ProgressStatusIsDraft(t *testing.T) {
	root := t.TempDir()
	slug := "sp-status"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	payload := `{"tasks":[{"title":"T1","criteria":[{"then":"a1"}],"linkedEdgeCases":["ec1"]},{"title":"T2","criteria":[{"then":"a2"},{"then":"a3"}]}]}`
	if _, err := ctx.Submit([]byte(payload)); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	ctx2 := buildSpecProposalCtx(t, root, slug)
	if ctx2.Progress().Status != spec.StatusDraft {
		t.Errorf("Progress().Status = %q, want %q", ctx2.Progress().Status, spec.StatusDraft)
	}
}

func TestSpecProposalDriver_SubmitValidTaskGen_SpecMdWritten(t *testing.T) {
	root := t.TempDir()
	slug := "sp-md"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	payload := `{"tasks":[{"title":"T1","criteria":[{"then":"a1"}],"linkedEdgeCases":["ec1"]},{"title":"T2","criteria":[{"then":"a2"},{"then":"a3"}]}]}`
	if _, err := ctx.Submit([]byte(payload)); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	data, err := os.ReadFile(paths.SpecMd(root, slug))
	if err != nil {
		t.Fatalf("ReadFile spec.md: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("spec.md is empty after valid task-gen submit")
	}
}

func TestSpecProposalDriver_SubmitValidTaskGen_SpecMdContainsT1(t *testing.T) {
	root := t.TempDir()
	slug := "sp-md-t1"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	payload := `{"tasks":[{"title":"T1","criteria":[{"then":"a1"}],"linkedEdgeCases":["ec1"]},{"title":"T2","criteria":[{"then":"a2"},{"then":"a3"}]}]}`
	if _, err := ctx.Submit([]byte(payload)); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	data, _ := os.ReadFile(paths.SpecMd(root, slug))
	if !strings.Contains(string(data), "T1") {
		t.Errorf("spec.md does not contain 'T1': %s", string(data))
	}
}

func TestSpecProposalDriver_SubmitValidTaskGen_SpecMdContainsTask1(t *testing.T) {
	root := t.TempDir()
	slug := "sp-md-task1"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	payload := `{"tasks":[{"title":"T1","criteria":[{"then":"a1"}],"linkedEdgeCases":["ec1"]},{"title":"T2","criteria":[{"then":"a2"},{"then":"a3"}]}]}`
	if _, err := ctx.Submit([]byte(payload)); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	data, _ := os.ReadFile(paths.SpecMd(root, slug))
	if !strings.Contains(string(data), "task-1") {
		t.Errorf("spec.md does not contain 'task-1': %s", string(data))
	}
}

func TestSpecProposalDriver_SubmitValidTaskGen_TasksGeneratedAnswered(t *testing.T) {
	root := t.TempDir()
	slug := "sp-answered"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	payload := `{"tasks":[{"title":"T1","criteria":[{"then":"a1"}],"linkedEdgeCases":["ec1"]},{"title":"T2","criteria":[{"then":"a2"},{"then":"a3"}]}]}`
	if _, err := ctx.Submit([]byte(payload)); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	ctx2 := buildSpecProposalCtx(t, root, slug)
	if !ctx2.HasAnswer("tasks_generated") {
		t.Error("tasks_generated not recorded after valid submit")
	}
}

func TestSpecProposalDriver_SubmitMalformedJSON_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "sp-malformed"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	_, err := ctx.Submit([]byte("{bad"))
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestSpecProposalDriver_SubmitMalformedJSON_TasksUnchanged(t *testing.T) {
	root := t.TempDir()
	slug := "sp-malformed-tasks"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	ctx.Submit([]byte("{bad"))

	ctx2 := buildSpecProposalCtx(t, root, slug)
	if len(ctx2.Progress().Tasks) != 0 {
		t.Errorf("Progress().Tasks = %v, want empty after failed submit", ctx2.Progress().Tasks)
	}
}

func TestSpecProposalDriver_SubmitMalformedJSON_TasksGeneratedNotAnswered(t *testing.T) {
	root := t.TempDir()
	slug := "sp-malformed-ans"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	ctx.Submit([]byte("{bad"))

	ctx2 := buildSpecProposalCtx(t, root, slug)
	if ctx2.HasAnswer("tasks_generated") {
		t.Error("tasks_generated recorded after malformed JSON submit, want not recorded")
	}
}

func TestSpecProposalDriver_SubmitZeroTasks_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "sp-zerotasks"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	_, err := ctx.Submit([]byte(`{"tasks":[]}`))
	if err == nil {
		t.Fatal("expected error for zero tasks, got nil")
	}
}

func TestSpecProposalDriver_SubmitZeroTasks_TasksUnchanged(t *testing.T) {
	root := t.TempDir()
	slug := "sp-zerotasks-tasks"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	ctx.Submit([]byte(`{"tasks":[]}`))

	ctx2 := buildSpecProposalCtx(t, root, slug)
	if len(ctx2.Progress().Tasks) != 0 {
		t.Errorf("Progress().Tasks len = %d, want 0 after zero-tasks submit", len(ctx2.Progress().Tasks))
	}
}

func TestSpecProposalDriver_SubmitZeroTasks_NotAnswered(t *testing.T) {
	root := t.TempDir()
	slug := "sp-zerotasks-ans"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	ctx.Submit([]byte(`{"tasks":[]}`))

	ctx2 := buildSpecProposalCtx(t, root, slug)
	if ctx2.HasAnswer("tasks_generated") {
		t.Error("tasks_generated recorded after zero-tasks submit, want not recorded")
	}
}

func TestSpecProposalDriver_SubmitEmptyTitle_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "sp-emptytitle"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	_, err := ctx.Submit([]byte(`{"tasks":[{"title":"","criteria":[{"then":"x"}]}]}`))
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
}

func TestSpecProposalDriver_SubmitEmptyTitle_TasksUnchanged(t *testing.T) {
	root := t.TempDir()
	slug := "sp-emptytitle-tasks"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	ctx.Submit([]byte(`{"tasks":[{"title":"","criteria":[{"then":"x"}]}]}`))

	ctx2 := buildSpecProposalCtx(t, root, slug)
	if len(ctx2.Progress().Tasks) != 0 {
		t.Errorf("Progress().Tasks len = %d, want 0 after empty-title submit", len(ctx2.Progress().Tasks))
	}
}

func TestSpecProposalDriver_SubmitEmptyTitle_NotAnswered(t *testing.T) {
	root := t.TempDir()
	slug := "sp-emptytitle-ans"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	ctx.Submit([]byte(`{"tasks":[{"title":"","criteria":[{"then":"x"}]}]}`))

	ctx2 := buildSpecProposalCtx(t, root, slug)
	if ctx2.HasAnswer("tasks_generated") {
		t.Error("tasks_generated recorded after empty-title submit, want not recorded")
	}
}

func TestSpecProposalDriver_SubmitEmptyAC_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "sp-emptyac"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	_, err := ctx.Submit([]byte(`{"tasks":[{"title":"X","criteria":[]}]}`))
	if err == nil {
		t.Fatal("expected error for empty AC, got nil")
	}
}

func TestSpecProposalDriver_SubmitEmptyAC_TasksUnchanged(t *testing.T) {
	root := t.TempDir()
	slug := "sp-emptyac-tasks"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	ctx.Submit([]byte(`{"tasks":[{"title":"X","criteria":[]}]}`))

	ctx2 := buildSpecProposalCtx(t, root, slug)
	if len(ctx2.Progress().Tasks) != 0 {
		t.Errorf("Progress().Tasks len = %d, want 0 after empty-AC submit", len(ctx2.Progress().Tasks))
	}
}

func TestSpecProposalDriver_SubmitEmptyAC_NotAnswered(t *testing.T) {
	root := t.TempDir()
	slug := "sp-emptyac-ans"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Next(): %v", err)
	}
	ctx.Submit([]byte(`{"tasks":[{"title":"X","criteria":[]}]}`))

	ctx2 := buildSpecProposalCtx(t, root, slug)
	if ctx2.HasAnswer("tasks_generated") {
		t.Error("tasks_generated recorded after empty-AC submit, want not recorded")
	}
}

func TestSpecProposalDriver_AfterTaskGen_NextReturnsSelfReviewPrompt(t *testing.T) {
	root := t.TempDir()
	slug := "sp-selfrev"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("First Next(): %v", err)
	}
	payload := `{"tasks":[{"title":"T1","criteria":[{"then":"a1"}],"linkedEdgeCases":["ec1"]},{"title":"T2","criteria":[{"then":"a2"},{"then":"a3"}]}]}`
	if _, err := ctx.Submit([]byte(payload)); err != nil {
		t.Fatalf("Submit task-gen: %v", err)
	}

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Second Next(): %v", err)
	}
	want, _ := promptregistry.Instruction(promptregistry.KeySelfReview)
	if action.Instruction != want {
		t.Errorf("Instruction = %q, want KeySelfReview value", action.Instruction)
	}
}

func TestSpecProposalDriver_AfterTaskGen_NextFormatIsFlag(t *testing.T) {
	root := t.TempDir()
	slug := "sp-flag"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("First Next(): %v", err)
	}
	payload := `{"tasks":[{"title":"T1","criteria":[{"then":"a1"}],"linkedEdgeCases":["ec1"]},{"title":"T2","criteria":[{"then":"a2"},{"then":"a3"}]}]}`
	if _, err := ctx.Submit([]byte(payload)); err != nil {
		t.Fatalf("Submit task-gen: %v", err)
	}

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Second Next(): %v", err)
	}
	if action.ExpectedInput.Format != engine.FormatFlag {
		t.Errorf("Format = %q, want %q", action.ExpectedInput.Format, engine.FormatFlag)
	}
}

func TestSpecProposalDriver_SelfReviewApprove_PhaseDone(t *testing.T) {
	root := t.TempDir()
	slug := "sp-approve"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("First Next(): %v", err)
	}
	payload := `{"tasks":[{"title":"T1","criteria":[{"then":"a1"}],"linkedEdgeCases":["ec1"]},{"title":"T2","criteria":[{"then":"a2"},{"then":"a3"}]}]}`
	if _, err := ctx.Submit([]byte(payload)); err != nil {
		t.Fatalf("Submit task-gen: %v", err)
	}

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Second Next(): %v", err)
	}
	if _, err := ctx.Submit([]byte("approve")); err != nil {
		t.Fatalf("Submit approve: %v", err)
	}

	state, err := spec.LoadState(root, slug)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.Phase == "spec-proposal" {
		t.Error("phase still spec-proposal after approve, want phase advanced")
	}
}

func TestSpecProposalDriver_SelfReviewNope_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "sp-nope"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("First Next(): %v", err)
	}
	payload := `{"tasks":[{"title":"T1","criteria":[{"then":"a1"}],"linkedEdgeCases":["ec1"]},{"title":"T2","criteria":[{"then":"a2"},{"then":"a3"}]}]}`
	if _, err := ctx.Submit([]byte(payload)); err != nil {
		t.Fatalf("Submit task-gen: %v", err)
	}

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Second Next(): %v", err)
	}
	_, err := ctx.Submit([]byte("nope"))
	if err == nil {
		t.Fatal("Submit(nope) self-review = nil, want non-nil error")
	}
}

func TestSpecProposalDriver_SelfReviewNope_PhaseNotAdvanced(t *testing.T) {
	root := t.TempDir()
	slug := "sp-nope-phase"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("First Next(): %v", err)
	}
	payload := `{"tasks":[{"title":"T1","criteria":[{"then":"a1"}],"linkedEdgeCases":["ec1"]},{"title":"T2","criteria":[{"then":"a2"},{"then":"a3"}]}]}`
	if _, err := ctx.Submit([]byte(payload)); err != nil {
		t.Fatalf("Submit task-gen: %v", err)
	}

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Second Next(): %v", err)
	}
	ctx.Submit([]byte("nope"))

	state, err := spec.LoadState(root, slug)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.Phase != "spec-proposal" {
		t.Errorf("phase after nope = %q, want spec-proposal", state.Phase)
	}
}

func TestSpecProposalDriver_SelfReviewNope_NotRecorded(t *testing.T) {
	root := t.TempDir()
	slug := "sp-nope-ans"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("First Next(): %v", err)
	}
	payload := `{"tasks":[{"title":"T1","criteria":[{"then":"a1"}],"linkedEdgeCases":["ec1"]},{"title":"T2","criteria":[{"then":"a2"},{"then":"a3"}]}]}`
	if _, err := ctx.Submit([]byte(payload)); err != nil {
		t.Fatalf("Submit task-gen: %v", err)
	}

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Second Next(): %v", err)
	}
	ctx.Submit([]byte("nope"))

	ctx2 := buildSpecProposalCtx(t, root, slug)
	if ctx2.HasAnswer("self_review") {
		t.Error("self_review recorded after nope, want not recorded")
	}
}

func TestBuildTasksFromGen_LinkedEdgeCasesPersist(t *testing.T) {
	p := TaskGenPayload{
		Tasks: []TaskGenItem{
			{Title: "T1", Criteria: []spec.Criterion{{Then: "a1"}}, LinkedEdgeCases: []string{"ec-1", "ec-2"}},
		},
	}
	tasks, err := BuildTasksFromGen(p, false, nil)
	if err != nil {
		t.Fatalf("BuildTasksFromGen: %v", err)
	}
	got := tasks[0].EdgeCases
	want := []string{"ec-1", "ec-2"}
	if len(got) != len(want) {
		t.Fatalf("EdgeCases len = %d, want %d; got %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("EdgeCases[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBuildTasksFromGen_FallbackWhenEmpty(t *testing.T) {
	fallback := []string{"ec-a", "ec-b"}
	p := TaskGenPayload{
		Tasks: []TaskGenItem{
			{Title: "T1", Criteria: []spec.Criterion{{Then: "a1"}}, LinkedEdgeCases: []string{}},
		},
	}
	tasks, err := BuildTasksFromGen(p, false, fallback)
	if err != nil {
		t.Fatalf("BuildTasksFromGen: %v", err)
	}
	got := tasks[0].EdgeCases
	if len(got) != len(fallback) {
		t.Fatalf("EdgeCases len = %d, want %d; got %v", len(got), len(fallback), got)
	}
	for i := range fallback {
		if got[i] != fallback[i] {
			t.Errorf("EdgeCases[%d] = %q, want %q", i, got[i], fallback[i])
		}
	}
}

func TestBuildTasksFromGen_NoFallbackNoLinked(t *testing.T) {
	p := TaskGenPayload{
		Tasks: []TaskGenItem{
			{Title: "T1", Criteria: []spec.Criterion{{Then: "a1"}}, LinkedEdgeCases: []string{}},
		},
	}
	tasks, err := BuildTasksFromGen(p, false, nil)
	if err != nil {
		t.Fatalf("BuildTasksFromGen: %v", err)
	}
	got := tasks[0].EdgeCases
	if len(got) != 0 {
		t.Errorf("EdgeCases = %v, want empty/nil", got)
	}
}

func TestBuildTasksFromGen_PopulatesCriteria(t *testing.T) {
	p := TaskGenPayload{
		Tasks: []TaskGenItem{
			{
				Title: "T1",
				Criteria: []spec.Criterion{
					{Given: "a user", When: "they act", Then: "something happens"},
				},
			},
		},
	}
	tasks, err := BuildTasksFromGen(p, false, nil)
	if err != nil {
		t.Fatalf("BuildTasksFromGen: %v", err)
	}
	if len(tasks[0].Criteria) != 1 {
		t.Fatalf("Criteria len = %d, want 1", len(tasks[0].Criteria))
	}
	c := tasks[0].Criteria[0]
	if c.Given != "a user" {
		t.Errorf("Given = %q, want %q", c.Given, "a user")
	}
	if c.When != "they act" {
		t.Errorf("When = %q, want %q", c.When, "they act")
	}
	if c.Then != "something happens" {
		t.Errorf("Then = %q, want %q", c.Then, "something happens")
	}
	if c.ID == "" {
		t.Error("criterion ID is empty; want ac-N assigned by AssignCriterionIDs")
	}
}

func TestBuildTasksFromGen_AssignsStableCriterionIDs(t *testing.T) {
	p := TaskGenPayload{
		Tasks: []TaskGenItem{
			{
				Title: "T1",
				Criteria: []spec.Criterion{
					{Then: "first outcome"},
					{Then: "second outcome"},
				},
			},
		},
	}
	tasks, err := BuildTasksFromGen(p, false, nil)
	if err != nil {
		t.Fatalf("BuildTasksFromGen: %v", err)
	}
	if len(tasks[0].Criteria) != 2 {
		t.Fatalf("Criteria len = %d, want 2", len(tasks[0].Criteria))
	}
	if tasks[0].Criteria[0].ID != "ac-1" {
		t.Errorf("Criteria[0].ID = %q, want ac-1", tasks[0].Criteria[0].ID)
	}
	if tasks[0].Criteria[1].ID != "ac-2" {
		t.Errorf("Criteria[1].ID = %q, want ac-2", tasks[0].Criteria[1].ID)
	}
}

func TestBuildTasksFromGen_MixedTasks(t *testing.T) {
	fallback := []string{"ec-x", "ec-y"}
	p := TaskGenPayload{
		Tasks: []TaskGenItem{
			{Title: "T1", Criteria: []spec.Criterion{{Then: "a1"}}, LinkedEdgeCases: []string{"linked-1"}},
			{Title: "T2", Criteria: []spec.Criterion{{Then: "a2"}}, LinkedEdgeCases: []string{}},
		},
	}
	tasks, err := BuildTasksFromGen(p, false, fallback)
	if err != nil {
		t.Fatalf("BuildTasksFromGen: %v", err)
	}
	if len(tasks[0].EdgeCases) != 1 || tasks[0].EdgeCases[0] != "linked-1" {
		t.Errorf("tasks[0].EdgeCases = %v, want [linked-1]", tasks[0].EdgeCases)
	}
	if len(tasks[1].EdgeCases) != 2 || tasks[1].EdgeCases[0] != "ec-x" || tasks[1].EdgeCases[1] != "ec-y" {
		t.Errorf("tasks[1].EdgeCases = %v, want [ec-x ec-y]", tasks[1].EdgeCases)
	}
}

func TestSpecProposalDriver_BothAnswered_NextReturnsDone(t *testing.T) {
	root := t.TempDir()
	slug := "sp-done"
	seedSpecProposalSpec(t, root, slug)
	ctx := buildSpecProposalCtx(t, root, slug)

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("First Next(): %v", err)
	}
	payload := `{"tasks":[{"title":"T1","criteria":[{"then":"a1"}],"linkedEdgeCases":["ec1"]},{"title":"T2","criteria":[{"then":"a2"},{"then":"a3"}]}]}`
	if _, err := ctx.Submit([]byte(payload)); err != nil {
		t.Fatalf("Submit task-gen: %v", err)
	}

	if _, err := ctx.Next(); err != nil {
		t.Fatalf("Second Next(): %v", err)
	}
	if _, err := ctx.Submit([]byte("approve")); err != nil {
		t.Fatalf("Submit approve: %v", err)
	}

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Third Next() after both answered: %v", err)
	}
	if action.Action != "" && action.Action != engine.ActionTerminal {
		t.Errorf("after both answered, Next action = %q, want empty or terminal (phase done)", action.Action)
	}
}
