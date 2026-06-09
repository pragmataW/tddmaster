package spec

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pragmataW/tddmaster/internal/paths"
)

func TestSaveLoadState_RoundTrip(t *testing.T) {
	root := t.TempDir()
	slug := "test-spec"
	now := time.Now().UTC().Truncate(time.Second)
	original := State{
		Version: 1,
		Slug:    slug,
		Phase:   PhaseInitial,
		Answers: map[string][]Answer{
			"k": {{Key: "key1", Value: "val1"}},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := SaveState(root, slug, original)
	if err != nil {
		t.Fatalf("SaveState returned error: %v", err)
	}

	loaded, err := LoadState(root, slug)
	if err != nil {
		t.Fatalf("LoadState returned error: %v", err)
	}

	if loaded.Version != original.Version {
		t.Errorf("Version mismatch: got %d, want %d", loaded.Version, original.Version)
	}
	if loaded.Slug != original.Slug {
		t.Errorf("Slug mismatch: got %q, want %q", loaded.Slug, original.Slug)
	}
	if loaded.Phase != original.Phase {
		t.Errorf("Phase mismatch: got %q, want %q", loaded.Phase, original.Phase)
	}
	if !reflect.DeepEqual(loaded.Answers, original.Answers) {
		t.Errorf("Answers mismatch: got %v, want %v", loaded.Answers, original.Answers)
	}
	if !loaded.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", loaded.CreatedAt, original.CreatedAt)
	}
	if loaded.UpdatedAt.Before(original.UpdatedAt) {
		t.Errorf("UpdatedAt should be refreshed on save: got %v, want >= %v", loaded.UpdatedAt, original.UpdatedAt)
	}
}

func TestSaveLoadSettings_RoundTrip(t *testing.T) {
	root := t.TempDir()
	slug := "test-spec"
	original := DefaultSettings()

	err := SaveSettings(root, slug, original)
	if err != nil {
		t.Fatalf("SaveSettings returned error: %v", err)
	}

	loaded, err := LoadSettings(root, slug)
	if err != nil {
		t.Fatalf("LoadSettings returned error: %v", err)
	}

	if loaded.TDDEnabled != original.TDDEnabled {
		t.Errorf("TDDEnabled mismatch: got %v, want %v", loaded.TDDEnabled, original.TDDEnabled)
	}
	if loaded.SkipVerifierEnabled != original.SkipVerifierEnabled {
		t.Errorf("SkipVerifierEnabled mismatch: got %v, want %v", loaded.SkipVerifierEnabled, original.SkipVerifierEnabled)
	}
	if loaded.ImportantTaskGateEnabled != original.ImportantTaskGateEnabled {
		t.Errorf("ImportantTaskGateEnabled mismatch: got %v, want %v", loaded.ImportantTaskGateEnabled, original.ImportantTaskGateEnabled)
	}
}

func TestSaveLoadProgress_RoundTrip(t *testing.T) {
	root := t.TempDir()
	slug := "test-spec"
	original := Progress{
		Spec:   slug,
		Status: "draft",
		Tasks: []Task{
			{
				ID:         "t1",
				Title:      "First Task",
				AC:         []string{"ac1"},
				Done:       false,
				TDDEnabled: true,
				Important:  false,
			},
		},
		UpdatedAt: time.Now().UTC().Truncate(time.Second),
	}

	err := SaveProgress(root, slug, original)
	if err != nil {
		t.Fatalf("SaveProgress returned error: %v", err)
	}

	loaded, err := LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress returned error: %v", err)
	}

	if loaded.Spec != original.Spec {
		t.Errorf("Spec mismatch: got %q, want %q", loaded.Spec, original.Spec)
	}
	if loaded.Status != original.Status {
		t.Errorf("Status mismatch: got %q, want %q", loaded.Status, original.Status)
	}
	if !reflect.DeepEqual(loaded.Tasks, original.Tasks) {
		t.Errorf("Tasks mismatch: got %v, want %v", loaded.Tasks, original.Tasks)
	}
	if loaded.UpdatedAt.Before(original.UpdatedAt) {
		t.Errorf("UpdatedAt should be refreshed on save: got %v, want >= %v", loaded.UpdatedAt, original.UpdatedAt)
	}
}

func TestSaveState_Indented(t *testing.T) {
	root := t.TempDir()
	slug := "indent-spec"
	s := State{
		Version: 1,
		Slug:    slug,
		Phase:   PhaseInitial,
		Answers: map[string][]Answer{},
	}

	err := SaveState(root, slug, s)
	if err != nil {
		t.Fatalf("SaveState returned error: %v", err)
	}

	data, err := os.ReadFile(paths.SpecState(root, slug))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	if !strings.Contains(string(data), "\n  ") {
		t.Errorf("expected 2-space indented JSON, got:\n%s", string(data))
	}
}

func TestSaveState_EmptyAnswersMapPersistsAsObject(t *testing.T) {
	root := t.TempDir()
	slug := "empty-answers-spec"
	s := State{
		Version: 1,
		Slug:    slug,
		Phase:   PhaseInitial,
		Answers: map[string][]Answer{},
	}

	err := SaveState(root, slug, s)
	if err != nil {
		t.Fatalf("SaveState returned error: %v", err)
	}

	data, err := os.ReadFile(paths.SpecState(root, slug))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, `"answers": {}`) {
		t.Errorf("expected empty answers to be {} (object), got:\n%s", content)
	}
}

func TestSaveProgress_EmptyTasksPersistAsArray(t *testing.T) {
	root := t.TempDir()
	slug := "empty-tasks-spec"
	p := Progress{
		Spec:   slug,
		Status: "draft",
		Tasks:  []Task{},
	}

	err := SaveProgress(root, slug, p)
	if err != nil {
		t.Fatalf("SaveProgress returned error: %v", err)
	}

	data, err := os.ReadFile(paths.SpecProgress(root, slug))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, `"tasks": []`) {
		t.Errorf("expected empty tasks to be [] (array), got:\n%s", content)
	}
}

func TestExists(t *testing.T) {
	root := t.TempDir()

	if Exists(root, "foo") {
		t.Error("expected Exists to be false for non-existent spec")
	}

	s := State{Version: 1, Slug: "foo", Phase: PhaseInitial, Answers: map[string][]Answer{}}
	err := SaveState(root, "foo", s)
	if err != nil {
		t.Fatalf("SaveState returned error: %v", err)
	}

	if !Exists(root, "foo") {
		t.Error("expected Exists to be true after SaveState")
	}

	err = os.MkdirAll(paths.SpecDir(root, "bar"), 0o755)
	if err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	if Exists(root, "bar") {
		t.Error("expected Exists to be false when dir exists but state.json does not")
	}
}

func TestLoadState_MissingReturnsError(t *testing.T) {
	root := t.TempDir()

	_, err := LoadState(root, "nonexistent")
	if err == nil {
		t.Error("expected error when loading state for non-existent spec, got nil")
	}
}

func TestSaveLoadProgress_WithExecState_RoundTrip(t *testing.T) {
	root := t.TempDir()
	slug := "exec-state-spec"
	exec := &ExecState{
		Iteration:       3,
		TDDCycle:        "green",
		RefactorRounds:  2,
		RefactorApplied: true,
		ApprovedPlans:   []string{"plan-a", "plan-b"},
		PlanAttempts:    map[string]int{"task-1": 2, "task-2": 1},
		PlanFeedback:    map[string]string{"task-1": "needs more detail"},
	}
	original := Progress{
		Spec:      slug,
		Status:    "executing",
		Tasks:     []Task{},
		UpdatedAt: time.Now().UTC().Truncate(time.Second),
		Execution: exec,
	}

	err := SaveProgress(root, slug, original)
	if err != nil {
		t.Fatalf("SaveProgress returned error: %v", err)
	}

	loaded, err := LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress returned error: %v", err)
	}

	if loaded.Execution == nil {
		t.Fatal("expected Execution to be non-nil after round-trip")
	}
	if !reflect.DeepEqual(loaded.Execution, original.Execution) {
		t.Errorf("Execution mismatch: got %+v, want %+v", loaded.Execution, original.Execution)
	}
}

func TestLoadProgress_OldFormatWithoutExecutionKey_ReturnsNilExecution(t *testing.T) {
	root := t.TempDir()
	slug := "old-format-spec"

	oldJSON := []byte(`{
  "spec": "old-format-spec",
  "status": "draft",
  "tasks": [],
  "updatedAt": "2024-01-01T00:00:00Z"
}`)

	dir := paths.SpecDir(root, slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(paths.SpecProgress(root, slug), oldJSON, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	loaded, err := LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress returned error: %v", err)
	}
	if loaded.Execution != nil {
		t.Errorf("expected Execution to be nil for old-format file, got %+v", loaded.Execution)
	}
}
