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
		Tasks:     []spec.Task{{ID: "task-1", Title: initialTitle, AC: []string{}}},
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
	initialTasks := []spec.Task{{ID: "task-1", Title: "Old", AC: []string{}}}
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
	initialTasks := []spec.Task{{ID: "task-1", Title: "Existing", AC: []string{}}}
	scaffoldRefinementState(t, root, slug, initialTasks)

	answer := `{"add":[{"title":"Added","ac":["a1"]}]}`
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
		{ID: "task-1", Title: "Keep", AC: []string{}},
		{ID: "task-2", Title: "Remove", AC: []string{}},
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
	initialTasks := []spec.Task{{ID: "task-1", Title: "Existing", AC: []string{}}}
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
	scaffoldRefinementState(t, root, slug, []spec.Task{{ID: "task-1", Title: "T", AC: []string{}}})

	_, err := executeRefine(t, root, slug, "--answer", "{bad")
	if err == nil {
		t.Fatal("expected error for invalid JSON answer, got nil")
	}
}

func Test_RefineCmd_MissingAnswerFlag_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	scaffoldRefinementState(t, root, slug, []spec.Task{{ID: "task-1", Title: "T", AC: []string{}}})

	_, err := executeRefine(t, root, slug)
	if err == nil {
		t.Fatal("expected error when --answer flag is missing/empty, got nil")
	}
}

func Test_RefineCmd_SuccessOutput_ContainsTaskIDsAndNextHint(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	initialTasks := []spec.Task{{ID: "task-1", Title: "Old", AC: []string{}}}
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
	initialTasks := []spec.Task{{ID: "task-1", Title: "Existing", AC: []string{}}}
	scaffoldRefinementState(t, root, slug, initialTasks)

	answer := `{"add":[{"title":"WithEC","ac":["a1"],"edgeCases":["ec-alpha","ec-beta"]}]}`
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
