package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/phasecatalog"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func executeRefine(t *testing.T, root string, extraArgs ...string) (string, error) {
	t.Helper()
	root_cmd := newRootCmd()
	var buf bytes.Buffer
	root_cmd.SetOut(&buf)
	root_cmd.SetErr(&buf)
	args := append([]string{"refine"}, extraArgs...)
	if root != "" {
		args = append(args, "--root", root)
	}
	root_cmd.SetArgs(args)
	err := root_cmd.Execute()
	return buf.String(), err
}

func scaffoldRefinementState(t *testing.T, root, slug string, tasks []spec.Task) {
	t.Helper()
	scaffoldSpec(t, root, slug)
	st, err := spec.LoadState(root, slug)
	if err != nil {
		t.Fatalf("scaffoldRefinementState LoadState: %v", err)
	}
	st.Phase = string(phasecatalog.PhaseRefinement)
	st.UpdatedAt = time.Now().UTC()
	if err := spec.SaveState(root, slug, st); err != nil {
		t.Fatalf("scaffoldRefinementState SaveState: %v", err)
	}
	p := spec.Progress{
		Spec:      slug,
		Status:    spec.StatusDraft,
		Tasks:     tasks,
		UpdatedAt: time.Now().UTC(),
	}
	if err := spec.SaveProgress(root, slug, p); err != nil {
		t.Fatalf("scaffoldRefinementState SaveProgress: %v", err)
	}
}

func Test_RefineCmd_NoSlugArg_ReturnsError(t *testing.T) {
	root := t.TempDir()
	_, err := executeRefine(t, root)
	if err == nil {
		t.Fatal("expected error when no slug arg provided, got nil")
	}
}

func Test_RefineCmd_SpecNotExist_ReturnsError(t *testing.T) {
	root := t.TempDir()
	_, err := executeRefine(t, root, "ghost-slug", "--answer", `{"update":{}}`)
	if err == nil {
		t.Fatal("expected error when spec does not exist, got nil")
	}
}

func Test_RefineCmd_WrongPhase_ReturnsRefinementError(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	scaffoldSpec(t, root, slug)

	initialTitle := "Original"
	p := spec.Progress{
		Spec:      slug,
		Status:    spec.StatusDraft,
		Tasks:     []spec.Task{{ID: "task-1", Title: initialTitle}},
		UpdatedAt: time.Now().UTC(),
	}
	if err := spec.SaveProgress(root, slug, p); err != nil {
		t.Fatalf("setup SaveProgress: %v", err)
	}

	_, err := executeRefine(t, root, slug, "--answer", `{"update":{"task-1":{"title":"New"}}}`)
	if err == nil {
		t.Fatal("expected error when phase is not refinement, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "refinement") {
		t.Errorf("expected error to mention 'refinement', got: %q", err.Error())
	}

	loaded, loadErr := spec.LoadProgress(root, slug)
	if loadErr != nil {
		t.Fatalf("reload progress: %v", loadErr)
	}
	if len(loaded.Tasks) != 1 || loaded.Tasks[0].Title != initialTitle {
		t.Errorf("expected tasks unchanged after wrong-phase error, got: %+v", loaded.Tasks)
	}
}

func Test_RefineCmd_Update_HappyPath(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	tddTrue := true
	initialTasks := []spec.Task{{ID: "task-1", Title: "Old"}}
	scaffoldRefinementState(t, root, slug, initialTasks)

	answer := `{"update":{"task-1":{"title":"New","tddEnabled":true}}}`
	_, err := executeRefine(t, root, slug, "--answer", answer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, loadErr := spec.LoadProgress(root, slug)
	if loadErr != nil {
		t.Fatalf("reload progress: %v", loadErr)
	}
	if len(loaded.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(loaded.Tasks))
	}
	if loaded.Tasks[0].Title != "New" {
		t.Errorf("expected task title 'New', got %q", loaded.Tasks[0].Title)
	}
	if loaded.Tasks[0].TDDEnabled != tddTrue {
		t.Errorf("expected TDDEnabled=true, got %v", loaded.Tasks[0].TDDEnabled)
	}

	mdPath := paths.SpecMd(root, slug)
	mdBytes, mdErr := os.ReadFile(mdPath)
	if mdErr != nil {
		t.Fatalf("expected spec.md to exist at %s, got error: %v", mdPath, mdErr)
	}
	if !strings.Contains(string(mdBytes), "New") {
		t.Errorf("expected spec.md to contain 'New', got: %q", string(mdBytes))
	}
}

func Test_RefineCmd_Add_HappyPath(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	initialTasks := []spec.Task{{ID: "task-1", Title: "Existing"}}
	scaffoldRefinementState(t, root, slug, initialTasks)

	answer := `{"add":[{"title":"Added","criteria":[{"then":"a1"}]}]}`
	_, err := executeRefine(t, root, slug, "--answer", answer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, loadErr := spec.LoadProgress(root, slug)
	if loadErr != nil {
		t.Fatalf("reload progress: %v", loadErr)
	}
	if len(loaded.Tasks) != 2 {
		t.Fatalf("expected 2 tasks after add, got %d", len(loaded.Tasks))
	}
	var found bool
	for _, tk := range loaded.Tasks {
		if tk.Title == "Added" && tk.ID == "task-2" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected task with id 'task-2' and title 'Added', tasks: %+v", loaded.Tasks)
	}
}

func Test_RefineCmd_Remove_HappyPath(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	initialTasks := []spec.Task{
		{ID: "task-1", Title: "Keep"},
		{ID: "task-2", Title: "Remove"},
	}
	scaffoldRefinementState(t, root, slug, initialTasks)

	answer := `{"remove":["task-2"]}`
	_, err := executeRefine(t, root, slug, "--answer", answer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, loadErr := spec.LoadProgress(root, slug)
	if loadErr != nil {
		t.Fatalf("reload progress: %v", loadErr)
	}
	if len(loaded.Tasks) != 1 {
		t.Fatalf("expected 1 task after remove, got %d", len(loaded.Tasks))
	}
	if loaded.Tasks[0].ID != "task-1" {
		t.Errorf("expected remaining task to be task-1, got %q", loaded.Tasks[0].ID)
	}
}

func Test_RefineCmd_UnknownIDUpdate_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	initialTasks := []spec.Task{{ID: "task-1", Title: "Existing"}}
	scaffoldRefinementState(t, root, slug, initialTasks)

	answer := `{"update":{"task-9":{"title":"X"}}}`
	_, err := executeRefine(t, root, slug, "--answer", answer)
	if err == nil {
		t.Fatal("expected error for unknown task id, got nil")
	}

	loaded, loadErr := spec.LoadProgress(root, slug)
	if loadErr != nil {
		t.Fatalf("reload progress after error: %v", loadErr)
	}
	if len(loaded.Tasks) != 1 || loaded.Tasks[0].ID != "task-1" {
		t.Errorf("expected tasks unchanged after unknown id error, got: %+v", loaded.Tasks)
	}
}

func Test_RefineCmd_InvalidJSONAnswer_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	scaffoldRefinementState(t, root, slug, []spec.Task{{ID: "task-1", Title: "T"}})

	_, err := executeRefine(t, root, slug, "--answer", "{bad")
	if err == nil {
		t.Fatal("expected error for invalid JSON answer, got nil")
	}
}

func Test_RefineCmd_MissingAnswerFlag_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	scaffoldRefinementState(t, root, slug, []spec.Task{{ID: "task-1", Title: "T"}})

	_, err := executeRefine(t, root, slug)
	if err == nil {
		t.Fatal("expected error when --answer flag is missing/empty, got nil")
	}
}

func Test_RefineCmd_SuccessOutput_ContainsTaskIDsAndNextHint(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	initialTasks := []spec.Task{{ID: "task-1", Title: "Old"}}
	scaffoldRefinementState(t, root, slug, initialTasks)

	answer := `{"update":{"task-1":{"title":"Updated"}}}`
	out, err := executeRefine(t, root, slug, "--answer", answer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &result); jsonErr != nil {
		t.Fatalf("stdout is not valid JSON: %v\noutput: %q", jsonErr, out)
	}

	outStr := strings.ToLower(out)
	if !strings.Contains(outStr, "task-1") {
		t.Errorf("expected output to contain 'task-1', got: %q", out)
	}
	if !strings.Contains(outStr, "next") {
		t.Errorf("expected output to reference 'next', got: %q", out)
	}
}

func Test_RefineCmd_OutputIncludesEdgeCases(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	initialTasks := []spec.Task{{ID: "task-1", Title: "Existing"}}
	scaffoldRefinementState(t, root, slug, initialTasks)

	answer := `{"add":[{"title":"WithEC","criteria":[{"then":"a1"}],"edgeCases":["ec-alpha","ec-beta"]}]}`
	out, err := executeRefine(t, root, slug, "--answer", answer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "edgeCases") {
		t.Errorf("expected output to contain 'edgeCases', got: %q", out)
	}
	if !strings.Contains(out, "ec-alpha") {
		t.Errorf("expected output to contain 'ec-alpha', got: %q", out)
	}
	if !strings.Contains(out, "ec-beta") {
		t.Errorf("expected output to contain 'ec-beta', got: %q", out)
	}
}

func Test_RefineCmd_DecodesCriteriaPayload(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	initialTasks := []spec.Task{{ID: "task-1", Title: "T", Criteria: []spec.Criterion{{ID: "ac-1", Then: "a1"}}}}
	scaffoldRefinementState(t, root, slug, initialTasks)

	answer := `{"update":{"task-1":{"criteria":[{"given":"a user","when":"they act","then":"it works"}]}}}`
	_, err := executeRefine(t, root, slug, "--answer", answer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, loadErr := spec.LoadProgress(root, slug)
	if loadErr != nil {
		t.Fatalf("reload progress: %v", loadErr)
	}
	if len(loaded.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(loaded.Tasks))
	}
	if len(loaded.Tasks[0].Criteria) != 1 {
		t.Fatalf("expected 1 criterion on task-1, got %d", len(loaded.Tasks[0].Criteria))
	}
	c := loaded.Tasks[0].Criteria[0]
	if c.Given != "a user" {
		t.Errorf("Criterion.Given = %q, want 'a user'", c.Given)
	}
	if c.When != "they act" {
		t.Errorf("Criterion.When = %q, want 'they act'", c.When)
	}
	if c.Then != "it works" {
		t.Errorf("Criterion.Then = %q, want 'it works'", c.Then)
	}
	if c.ID == "" {
		t.Error("Criterion.ID is empty; want ac-N assigned by AssignCriterionIDs")
	}
}

func Test_RefineCmd_RegisteredOnRoot(t *testing.T) {
	root := newRootCmd()
	var found bool
	for _, sub := range root.Commands() {
		if sub.Name() == "refine" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'refine' subcommand registered on root, but not found")
	}
}

func Test_RefineCmd_DependsOn_AddPersists(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	initialTasks := []spec.Task{{ID: "task-1", Title: "Existing"}}
	scaffoldRefinementState(t, root, slug, initialTasks)

	answer := `{"add":[{"title":"Dependent","criteria":[{"then":"a1"}],"dependsOn":["task-1"]}]}`
	_, err := executeRefine(t, root, slug, "--answer", answer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, loadErr := spec.LoadProgress(root, slug)
	if loadErr != nil {
		t.Fatalf("reload progress: %v", loadErr)
	}
	if len(loaded.Tasks) != 2 {
		t.Fatalf("expected 2 tasks after add, got %d", len(loaded.Tasks))
	}
	if len(loaded.Tasks[1].DependsOn) != 1 || loaded.Tasks[1].DependsOn[0] != "task-1" {
		t.Errorf("expected task-2 dependsOn [task-1], got %v", loaded.Tasks[1].DependsOn)
	}
}

func Test_RefineCmd_DependsOn_UpdateAndClear(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	initialTasks := []spec.Task{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B"},
	}
	scaffoldRefinementState(t, root, slug, initialTasks)

	if _, err := executeRefine(t, root, slug, "--answer", `{"update":{"task-2":{"dependsOn":["task-1"]}}}`); err != nil {
		t.Fatalf("unexpected error on update: %v", err)
	}
	loaded, loadErr := spec.LoadProgress(root, slug)
	if loadErr != nil {
		t.Fatalf("reload progress: %v", loadErr)
	}
	if len(loaded.Tasks[1].DependsOn) != 1 || loaded.Tasks[1].DependsOn[0] != "task-1" {
		t.Fatalf("expected task-2 dependsOn [task-1], got %v", loaded.Tasks[1].DependsOn)
	}

	if _, err := executeRefine(t, root, slug, "--answer", `{"update":{"task-2":{"dependsOn":[]}}}`); err != nil {
		t.Fatalf("unexpected error on clear: %v", err)
	}
	loaded, loadErr = spec.LoadProgress(root, slug)
	if loadErr != nil {
		t.Fatalf("reload progress after clear: %v", loadErr)
	}
	if len(loaded.Tasks[1].DependsOn) != 0 {
		t.Errorf("expected task-2 dependsOn cleared, got %v", loaded.Tasks[1].DependsOn)
	}
}

func Test_RefineCmd_DependsOn_Cycle_ReturnsError_TasksUnchanged(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	initialTasks := []spec.Task{
		{ID: "task-1", Title: "A", DependsOn: []string{"task-2"}},
		{ID: "task-2", Title: "B"},
	}
	scaffoldRefinementState(t, root, slug, initialTasks)

	_, err := executeRefine(t, root, slug, "--answer", `{"update":{"task-2":{"dependsOn":["task-1"]}}}`)
	if err == nil {
		t.Fatal("expected error for dependency cycle, got nil")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("expected error to mention 'cycle', got: %q", err.Error())
	}

	loaded, loadErr := spec.LoadProgress(root, slug)
	if loadErr != nil {
		t.Fatalf("reload progress after error: %v", loadErr)
	}
	if len(loaded.Tasks[1].DependsOn) != 0 {
		t.Errorf("expected task-2 unchanged after cycle error, got dependsOn %v", loaded.Tasks[1].DependsOn)
	}
}

func Test_RefineCmd_DependsOn_SelfDependency_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	initialTasks := []spec.Task{{ID: "task-1", Title: "A"}}
	scaffoldRefinementState(t, root, slug, initialTasks)

	_, err := executeRefine(t, root, slug, "--answer", `{"update":{"task-1":{"dependsOn":["task-1"]}}}`)
	if err == nil {
		t.Fatal("expected error for self-dependency, got nil")
	}
}

func Test_RefineCmd_DependsOn_UnknownID_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	initialTasks := []spec.Task{{ID: "task-1", Title: "A"}}
	scaffoldRefinementState(t, root, slug, initialTasks)

	_, err := executeRefine(t, root, slug, "--answer", `{"update":{"task-1":{"dependsOn":["task-77"]}}}`)
	if err == nil {
		t.Fatal("expected error for unknown dependency id, got nil")
	}
	if !strings.Contains(err.Error(), "task-77") {
		t.Errorf("expected error to name invalid id task-77, got: %q", err.Error())
	}
}

func Test_RefineCmd_RemoveWithDependents_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	initialTasks := []spec.Task{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B", DependsOn: []string{"task-1"}},
	}
	scaffoldRefinementState(t, root, slug, initialTasks)

	_, err := executeRefine(t, root, slug, "--answer", `{"remove":["task-1"]}`)
	if err == nil {
		t.Fatal("expected error when removing a task with dependents, got nil")
	}
	if !strings.Contains(err.Error(), "cannot remove task-1") {
		t.Errorf("expected error to contain 'cannot remove task-1', got: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "task-2") {
		t.Errorf("expected error to list dependent task-2, got: %q", err.Error())
	}

	loaded, loadErr := spec.LoadProgress(root, slug)
	if loadErr != nil {
		t.Fatalf("reload progress after error: %v", loadErr)
	}
	if len(loaded.Tasks) != 2 {
		t.Errorf("expected tasks unchanged after remove error, got: %+v", loaded.Tasks)
	}
}

func Test_RefineCmd_RemoveAndUpdateSamePayload_Succeeds(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	initialTasks := []spec.Task{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B", DependsOn: []string{"task-1"}},
	}
	scaffoldRefinementState(t, root, slug, initialTasks)

	answer := `{"remove":["task-1"],"update":{"task-2":{"dependsOn":[]}}}`
	_, err := executeRefine(t, root, slug, "--answer", answer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, loadErr := spec.LoadProgress(root, slug)
	if loadErr != nil {
		t.Fatalf("reload progress: %v", loadErr)
	}
	if len(loaded.Tasks) != 1 || loaded.Tasks[0].ID != "task-2" {
		t.Fatalf("expected only task-2 remaining, got: %+v", loaded.Tasks)
	}
	if len(loaded.Tasks[0].DependsOn) != 0 {
		t.Errorf("expected task-2 dependsOn cleared, got %v", loaded.Tasks[0].DependsOn)
	}
}

func Test_RefineCmd_SpecMdContainsDependsOn(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	initialTasks := []spec.Task{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B"},
	}
	scaffoldRefinementState(t, root, slug, initialTasks)

	if _, err := executeRefine(t, root, slug, "--answer", `{"update":{"task-2":{"dependsOn":["task-1"]}}}`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mdBytes, mdErr := os.ReadFile(paths.SpecMd(root, slug))
	if mdErr != nil {
		t.Fatalf("read spec.md: %v", mdErr)
	}
	if !strings.Contains(string(mdBytes), "Depends on: task-1") {
		t.Errorf("expected spec.md to contain 'Depends on: task-1', got: %q", string(mdBytes))
	}
}
