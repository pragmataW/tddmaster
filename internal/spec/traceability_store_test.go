package spec

import (
	"reflect"
	"testing"

	"github.com/pragmataW/tddmaster/internal/paths"
)

func TestSaveLoadTraceability_RoundTrip(t *testing.T) {
	root := t.TempDir()
	slug := "trace-spec"

	original := Traceability{
		"task-1": []TraceEntry{
			{FunctionName: "TestFoo", TaskID: "task-1", AC: []string{"ac1"}, EC: nil},
			{FunctionName: "TestBar", TaskID: "task-1", AC: []string{"ac2"}, EC: []string{"EC-1"}},
		},
		"task-2": []TraceEntry{
			{FunctionName: "TestBaz", TaskID: "task-2", AC: []string{"ac1"}, EC: []string{"EC-2"}},
		},
	}

	if err := SaveTraceability(root, slug, original); err != nil {
		t.Fatalf("SaveTraceability returned error: %v", err)
	}

	loaded, err := LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability returned error: %v", err)
	}

	if len(loaded) != len(original) {
		t.Fatalf("Traceability length: got %d, want %d", len(loaded), len(original))
	}

	for taskID, entries := range original {
		gotEntries, ok := loaded[taskID]
		if !ok {
			t.Errorf("missing key %q in loaded Traceability", taskID)
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
}

func TestSaveTraceability_WritesToCorrectPath(t *testing.T) {
	root := t.TempDir()
	slug := "trace-path-spec"

	tr := Traceability{
		"task-1": []TraceEntry{
			{FunctionName: "TestX", TaskID: "task-1", AC: []string{"ac1"}, EC: nil},
		},
	}

	if err := SaveTraceability(root, slug, tr); err != nil {
		t.Fatalf("SaveTraceability returned error: %v", err)
	}

	expectedPath := paths.SpecTraceability(root, slug)
	loaded, err := loadJSON[Traceability](expectedPath)
	if err != nil {
		t.Fatalf("loadJSON from expected path returned error: %v", err)
	}

	if len(loaded) != 1 {
		t.Errorf("expected 1 entry, got %d", len(loaded))
	}
}

func TestLoadTraceability_MissingFile_ReturnsEmptyWithoutError(t *testing.T) {
	root := t.TempDir()
	slug := "nonexistent-spec"

	got, err := LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability on missing file returned error: %v", err)
	}

	if got == nil {
		t.Fatal("LoadTraceability on missing file returned nil map, want empty Traceability{}")
	}

	if len(got) != 0 {
		t.Errorf("LoadTraceability on missing file returned non-empty map: %v", got)
	}
}

func TestLoadTraceability_MissingFile_EmptyTaskID_ReturnsEmptyWithoutError(t *testing.T) {
	root := t.TempDir()
	slug := ""

	got, err := LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability with empty slug on missing file returned error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("LoadTraceability with empty slug on missing file returned non-empty map: %v", got)
	}
}

func TestSaveTraceability_EmptyMap(t *testing.T) {
	root := t.TempDir()
	slug := "empty-trace-spec"

	if err := SaveTraceability(root, slug, Traceability{}); err != nil {
		t.Fatalf("SaveTraceability with empty map returned error: %v", err)
	}

	loaded, err := LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability returned error: %v", err)
	}

	if len(loaded) != 0 {
		t.Errorf("expected empty Traceability after round-trip, got: %v", loaded)
	}
}

func TestSaveTraceability_Idempotent(t *testing.T) {
	root := t.TempDir()
	slug := "idempotent-spec"

	tr := Traceability{
		"task-1": []TraceEntry{
			{FunctionName: "TestIdempotent", TaskID: "task-1", AC: []string{"ac1"}, EC: nil},
		},
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

	if len(loaded["task-1"]) != 1 {
		t.Errorf("expected 1 entry after idempotent save, got %d", len(loaded["task-1"]))
	}
}
