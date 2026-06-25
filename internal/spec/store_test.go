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
				Criteria:   []Criterion{{ID: "ac-1", Then: "ac1"}},
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


func TestSaveLoadTraceability_RoundTrip(t *testing.T) {
	root := t.TempDir()
	slug := "trace-spec"

	original := Traceability{
		Entries: map[string][]TraceEntry{
			"task-1": {
				{FunctionName: "TestFoo", TaskID: "task-1", CriterionIDs: []string{"ac-1"}, EC: nil},
				{FunctionName: "TestBar", TaskID: "task-1", CriterionIDs: []string{"ac-2"}, EC: []string{"EC-1"}},
			},
			"task-2": {
				{FunctionName: "TestBaz", TaskID: "task-2", CriterionIDs: []string{"ac-1"}, EC: []string{"EC-2"}},
			},
		},
		Coverage: map[string]float64{
			"internal/spec/store.go": 75,
		},
	}

	if err := SaveTraceability(root, slug, original); err != nil {
		t.Fatalf("SaveTraceability returned error: %v", err)
	}

	loaded, err := LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability returned error: %v", err)
	}

	if len(loaded.Entries) != len(original.Entries) {
		t.Fatalf("Entries length: got %d, want %d", len(loaded.Entries), len(original.Entries))
	}

	for taskID, entries := range original.Entries {
		gotEntries, ok := loaded.Entries[taskID]
		if !ok {
			t.Errorf("missing key %q in loaded Entries", taskID)
			continue
		}
		if len(gotEntries) != len(entries) {
			t.Errorf("task %q: entry count got %d, want %d", taskID, len(gotEntries), len(entries))
			continue
		}
		for i, e := range entries {
			if gotEntries[i].FunctionName != e.FunctionName {
				t.Errorf("task %q[%d].FunctionName: got %q, want %q", taskID, i, gotEntries[i].FunctionName, e.FunctionName)
			}
			if gotEntries[i].TaskID != e.TaskID {
				t.Errorf("task %q[%d].TaskID: got %q, want %q", taskID, i, gotEntries[i].TaskID, e.TaskID)
			}
			if !reflect.DeepEqual(gotEntries[i].CriterionIDs, e.CriterionIDs) {
				t.Errorf("task %q[%d].CriterionIDs: got %q, want %q", taskID, i, gotEntries[i].CriterionIDs, e.CriterionIDs)
			}
			if !reflect.DeepEqual(gotEntries[i].EC, e.EC) {
				t.Errorf("task %q[%d].EC: got %q, want %q", taskID, i, gotEntries[i].EC, e.EC)
			}
		}
	}

	if loaded.Coverage["internal/spec/store.go"] != 75 {
		t.Errorf("Coverage round-trip: got %v, want 75", loaded.Coverage["internal/spec/store.go"])
	}
}

func TestSaveTraceability_WritesToCorrectPath(t *testing.T) {
	root := t.TempDir()
	slug := "trace-path-spec"

	tr := Traceability{
		Entries: map[string][]TraceEntry{
			"task-1": {
				{FunctionName: "TestX", TaskID: "task-1", CriterionIDs: []string{"ac-1"}, EC: nil},
			},
		},
		Coverage: map[string]float64{},
	}

	if err := SaveTraceability(root, slug, tr); err != nil {
		t.Fatalf("SaveTraceability returned error: %v", err)
	}

	expectedPath := paths.SpecTraceability(root, slug)
	loaded, err := loadJSON[Traceability](expectedPath)
	if err != nil {
		t.Fatalf("loadJSON from expected path returned error: %v", err)
	}

	if len(loaded.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(loaded.Entries))
	}
}

func TestLoadTraceability_MissingFile_ReturnsNonNilMaps(t *testing.T) {
	root := t.TempDir()
	slug := "nonexistent-spec"

	got, err := LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability on missing file returned error: %v", err)
	}

	if got.Entries == nil {
		t.Fatal("Entries must be non-nil on missing file load")
	}
	if got.Coverage == nil {
		t.Fatal("Coverage must be non-nil on missing file load")
	}
}

func TestLoadTraceability_MissingFile_IndexDoesNotPanic(t *testing.T) {
	root := t.TempDir()
	slug := "nonexistent-spec-2"

	got, err := LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability on missing file returned error: %v", err)
	}

	_ = got.Entries["task-1"]
	_ = got.Coverage["internal/spec/model.go"]
}

func TestLoadTraceability_MissingFile_EmptySlug_ReturnsNonNilMaps(t *testing.T) {
	root := t.TempDir()
	slug := ""

	got, err := LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability with empty slug on missing file returned error: %v", err)
	}

	if got.Entries == nil {
		t.Fatal("Entries must be non-nil on empty-slug missing file load")
	}
	if got.Coverage == nil {
		t.Fatal("Coverage must be non-nil on empty-slug missing file load")
	}
}

func TestLoadTraceability_EmptyFile_DoesNotPanic(t *testing.T) {
	root := t.TempDir()
	slug := "empty-file-spec"

	p := paths.SpecTraceability(root, slug)
	if err := os.MkdirAll(paths.SpecDir(root, slug), dirPerm); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(p, []byte(`{}`), filePerm); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability on empty JSON object returned error: %v", err)
	}

	if got.Entries == nil {
		t.Fatal("Entries must be non-nil after loading empty JSON object")
	}
	if got.Coverage == nil {
		t.Fatal("Coverage must be non-nil after loading empty JSON object")
	}

	_ = got.Entries["task-1"]
	_ = got.Coverage["src/main.go"]
}

func TestSaveLoadTraceability_CoverageRoundTrip(t *testing.T) {
	root := t.TempDir()
	slug := "coverage-spec"

	original := Traceability{
		Entries:  map[string][]TraceEntry{},
		Coverage: map[string]float64{"internal/spec/model.go": 100, "cmd/root.go": 42},
	}

	if err := SaveTraceability(root, slug, original); err != nil {
		t.Fatalf("SaveTraceability returned error: %v", err)
	}

	loaded, err := LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability returned error: %v", err)
	}

	if !reflect.DeepEqual(loaded.Coverage, original.Coverage) {
		t.Errorf("Coverage round-trip: got %v, want %v", loaded.Coverage, original.Coverage)
	}
}

func TestSaveTraceability_EmptyEntries(t *testing.T) {
	root := t.TempDir()
	slug := "empty-trace-spec"

	if err := SaveTraceability(root, slug, Traceability{Entries: map[string][]TraceEntry{}, Coverage: map[string]float64{}}); err != nil {
		t.Fatalf("SaveTraceability with empty maps returned error: %v", err)
	}

	loaded, err := LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability returned error: %v", err)
	}

	if len(loaded.Entries) != 0 {
		t.Errorf("expected empty Entries after round-trip, got: %v", loaded.Entries)
	}
}

func TestSaveTraceability_Idempotent(t *testing.T) {
	root := t.TempDir()
	slug := "idempotent-spec"

	tr := Traceability{
		Entries: map[string][]TraceEntry{
			"task-1": {
				{FunctionName: "TestIdempotent", TaskID: "task-1", CriterionIDs: []string{"ac-1"}, EC: nil},
			},
		},
		Coverage: map[string]float64{"src/main.go": 60},
	}

	if err := SaveTraceability(root, slug, tr); err != nil {
		t.Fatalf("first SaveTraceability returned error: %v", err)
	}

	if err := SaveTraceability(root, slug, tr); err != nil {
		t.Fatalf("second SaveTraceability returned error: %v", err)
	}

	loaded, err := LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability returned error: %v", err)
	}

	if len(loaded.Entries["task-1"]) != 1 {
		t.Errorf("expected 1 entry after idempotent save, got %d", len(loaded.Entries["task-1"]))
	}
	if loaded.Coverage["src/main.go"] != 60 {
		t.Errorf("Coverage after idempotent save: got %v, want 60", loaded.Coverage["src/main.go"])
	}
}

func TestSaveLoadAnalysis_RoundTrip(t *testing.T) {
	root := t.TempDir()
	slug := "analysis-spec"

	original := Analysis{
		Verdict: "issues",
		Findings: []Finding{
			{
				Severity:   "block",
				Category:   "untestable",
				TaskID:     "task-1",
				AcID:       "ac-1",
				Detail:     "Then clause is empty",
				Suggestion: "Provide observable outcome",
				Source:     "criterion_lint",
			},
			{
				Severity: "warn",
				Category: "weak-criterion",
				TaskID:   "task-2",
				AcID:     "ac-2",
				Detail:   "When clause is missing",
				Source:   "criterion_lint",
			},
		},
	}

	if err := SaveAnalysis(root, slug, original); err != nil {
		t.Fatalf("SaveAnalysis returned error: %v", err)
	}

	loaded, err := LoadAnalysis(root, slug)
	if err != nil {
		t.Fatalf("LoadAnalysis returned error: %v", err)
	}

	if loaded.Verdict != original.Verdict {
		t.Errorf("Verdict: got %q, want %q", loaded.Verdict, original.Verdict)
	}
	if len(loaded.Findings) != len(original.Findings) {
		t.Fatalf("Findings length: got %d, want %d", len(loaded.Findings), len(original.Findings))
	}
	for i, f := range original.Findings {
		got := loaded.Findings[i]
		if got.Severity != f.Severity {
			t.Errorf("Findings[%d].Severity: got %q, want %q", i, got.Severity, f.Severity)
		}
		if got.Category != f.Category {
			t.Errorf("Findings[%d].Category: got %q, want %q", i, got.Category, f.Category)
		}
		if got.TaskID != f.TaskID {
			t.Errorf("Findings[%d].TaskID: got %q, want %q", i, got.TaskID, f.TaskID)
		}
		if got.AcID != f.AcID {
			t.Errorf("Findings[%d].AcID: got %q, want %q", i, got.AcID, f.AcID)
		}
		if got.Detail != f.Detail {
			t.Errorf("Findings[%d].Detail: got %q, want %q", i, got.Detail, f.Detail)
		}
		if got.Source != f.Source {
			t.Errorf("Findings[%d].Source: got %q, want %q", i, got.Source, f.Source)
		}
	}
}

func TestSaveAnalysis_WritesToCorrectPath(t *testing.T) {
	root := t.TempDir()
	slug := "analysis-path-spec"

	a := Analysis{
		Verdict: "clean",
		Findings: []Finding{
			{Severity: "warn", Category: "weak-criterion", Detail: "missing When", Source: "criterion_lint"},
		},
	}

	if err := SaveAnalysis(root, slug, a); err != nil {
		t.Fatalf("SaveAnalysis returned error: %v", err)
	}

	expectedPath := paths.SpecAnalysis(root, slug)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected file at %q but it does not exist", expectedPath)
	}
}

func TestLoadAnalysis_MissingFile_NoError_OrEmpty(t *testing.T) {
	root := t.TempDir()
	slug := "nonexistent-analysis-spec"

	got, err := LoadAnalysis(root, slug)
	if err != nil {
		t.Fatalf("LoadAnalysis on missing file returned error: %v", err)
	}

	_ = got.Verdict
	_ = got.Findings
}

func TestSaveLoadAnalysis_ZeroFindings_RoundTrip(t *testing.T) {
	root := t.TempDir()
	slug := "zero-findings-spec"

	original := Analysis{
		Verdict:  "clean",
		Findings: []Finding{},
	}

	if err := SaveAnalysis(root, slug, original); err != nil {
		t.Fatalf("SaveAnalysis returned error: %v", err)
	}

	loaded, err := LoadAnalysis(root, slug)
	if err != nil {
		t.Fatalf("LoadAnalysis returned error: %v", err)
	}

	if loaded.Verdict != "clean" {
		t.Errorf("Verdict: got %q, want %q", loaded.Verdict, "clean")
	}

	_ = loaded.Findings
}

func TestSaveLoadTraceability_CriterionIDs_RoundTrip(t *testing.T) {
	root := t.TempDir()
	slug := "criterion-ids-spec"

	original := Traceability{
		Entries: map[string][]TraceEntry{
			"task-1": {
				{
					FunctionName: "TestFirst",
					TaskID:       "task-1",
					CriterionIDs: []string{"ac-1", "ac-2"},
					EC:           []string{"EC-1"},
				},
				{
					FunctionName: "TestSecond",
					TaskID:       "task-1",
					CriterionIDs: []string{"ac-3"},
					EC:           nil,
				},
			},
		},
		Coverage: map[string]float64{"internal/spec/model.go": 85},
	}

	if err := SaveTraceability(root, slug, original); err != nil {
		t.Fatalf("SaveTraceability returned error: %v", err)
	}

	loaded, err := LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability returned error: %v", err)
	}

	entries, ok := loaded.Entries["task-1"]
	if !ok {
		t.Fatal("task-1 entries missing after round-trip")
	}
	if len(entries) != 2 {
		t.Fatalf("task-1 entry count: got %d, want 2", len(entries))
	}
	if !reflect.DeepEqual(entries[0].CriterionIDs, original.Entries["task-1"][0].CriterionIDs) {
		t.Errorf("entries[0].CriterionIDs: got %v, want %v", entries[0].CriterionIDs, original.Entries["task-1"][0].CriterionIDs)
	}
	if !reflect.DeepEqual(entries[1].CriterionIDs, original.Entries["task-1"][1].CriterionIDs) {
		t.Errorf("entries[1].CriterionIDs: got %v, want %v", entries[1].CriterionIDs, original.Entries["task-1"][1].CriterionIDs)
	}
}
