package spec

import (
	"os"
	"reflect"
	"testing"

	"github.com/pragmataW/tddmaster/internal/paths"
)

func TestSaveLoadTraceability_RoundTrip(t *testing.T) {
	root := t.TempDir()
	slug := "trace-spec"

	original := Traceability{
		Entries: map[string][]TraceEntry{
			"task-1": {
				{FunctionName: "TestFoo", TaskID: "task-1", AC: []string{"ac1"}, EC: nil},
				{FunctionName: "TestBar", TaskID: "task-1", AC: []string{"ac2"}, EC: []string{"EC-1"}},
			},
			"task-2": {
				{FunctionName: "TestBaz", TaskID: "task-2", AC: []string{"ac1"}, EC: []string{"EC-2"}},
			},
		},
		Coverage: map[string]int{
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
			if !reflect.DeepEqual(gotEntries[i].AC, e.AC) {
				t.Errorf("task %q[%d].AC: got %q, want %q", taskID, i, gotEntries[i].AC, e.AC)
			}
			if !reflect.DeepEqual(gotEntries[i].EC, e.EC) {
				t.Errorf("task %q[%d].EC: got %q, want %q", taskID, i, gotEntries[i].EC, e.EC)
			}
		}
	}

	if loaded.Coverage["internal/spec/store.go"] != 75 {
		t.Errorf("Coverage round-trip: got %d, want 75", loaded.Coverage["internal/spec/store.go"])
	}
}

func TestSaveTraceability_WritesToCorrectPath(t *testing.T) {
	root := t.TempDir()
	slug := "trace-path-spec"

	tr := Traceability{
		Entries: map[string][]TraceEntry{
			"task-1": {
				{FunctionName: "TestX", TaskID: "task-1", AC: []string{"ac1"}, EC: nil},
			},
		},
		Coverage: map[string]int{},
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
		Coverage: map[string]int{"internal/spec/model.go": 100, "cmd/root.go": 42},
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

	if err := SaveTraceability(root, slug, Traceability{Entries: map[string][]TraceEntry{}, Coverage: map[string]int{}}); err != nil {
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
				{FunctionName: "TestIdempotent", TaskID: "task-1", AC: []string{"ac1"}, EC: nil},
			},
		},
		Coverage: map[string]int{"src/main.go": 60},
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
		t.Errorf("Coverage after idempotent save: got %d, want 60", loaded.Coverage["src/main.go"])
	}
}
