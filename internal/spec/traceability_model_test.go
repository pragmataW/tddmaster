package spec

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestTraceEntry_JSONTags(t *testing.T) {
	entry := TraceEntry{
		FunctionName: "TestFoo_Bar",
		TaskID:       "task-1",
		AC:           []string{"ac1"},
		EC:           []string{"EC-3"},
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	for _, tag := range []string{`"functionName"`, `"taskId"`, `"ac"`, `"ec"`} {
		if !strings.Contains(raw, tag) {
			t.Errorf("JSON missing tag %s in: %s", tag, raw)
		}
	}
}

func TestTraceEntry_JSONRoundTrip(t *testing.T) {
	original := TraceEntry{
		FunctionName: "TestDoSomething",
		TaskID:       "task-2",
		AC:           []string{"must handle nil input"},
		EC:           []string{"EC-1"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got TraceEntry
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if got.FunctionName != original.FunctionName {
		t.Errorf("FunctionName: got %q, want %q", got.FunctionName, original.FunctionName)
	}
	if got.TaskID != original.TaskID {
		t.Errorf("TaskID: got %q, want %q", got.TaskID, original.TaskID)
	}
	if !reflect.DeepEqual(got.AC, original.AC) {
		t.Errorf("AC: got %q, want %q", got.AC, original.AC)
	}
	if !reflect.DeepEqual(got.EC, original.EC) {
		t.Errorf("EC: got %q, want %q", got.EC, original.EC)
	}
}

func TestTraceability_IsStruct(t *testing.T) {
	tr := Traceability{
		Entries: map[string][]TraceEntry{
			"task-1": {
				{FunctionName: "TestAlpha", TaskID: "task-1", AC: []string{"ac1"}, EC: nil},
			},
		},
		Coverage: map[string]int{
			"internal/spec/model.go": 80,
		},
	}

	if tr.Entries == nil {
		t.Fatal("Entries must not be nil after explicit init")
	}
	if tr.Coverage == nil {
		t.Fatal("Coverage must not be nil after explicit init")
	}
}

func TestTraceability_JSONRoundTrip(t *testing.T) {
	original := Traceability{
		Entries: map[string][]TraceEntry{
			"task-1": {
				{FunctionName: "TestAlpha", TaskID: "task-1", AC: []string{"ac1"}, EC: nil},
				{FunctionName: "TestBeta", TaskID: "task-1", AC: []string{"ac2"}, EC: []string{"EC-1"}},
			},
			"task-2": {
				{FunctionName: "TestGamma", TaskID: "task-2", AC: []string{"ac1"}, EC: []string{"EC-2"}},
			},
		},
		Coverage: map[string]int{
			"internal/spec/model.go": 90,
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got Traceability
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(got.Entries) != len(original.Entries) {
		t.Fatalf("Entries length: got %d, want %d", len(got.Entries), len(original.Entries))
	}

	for taskID, entries := range original.Entries {
		gotEntries, ok := got.Entries[taskID]
		if !ok {
			t.Errorf("missing key %q in round-tripped Entries", taskID)
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

	if len(got.Coverage) != len(original.Coverage) {
		t.Fatalf("Coverage length: got %d, want %d", len(got.Coverage), len(original.Coverage))
	}
	for k, v := range original.Coverage {
		gotV, ok := got.Coverage[k]
		if !ok {
			t.Errorf("Coverage missing key %q", k)
			continue
		}
		if gotV != v {
			t.Errorf("Coverage[%q]: got %d, want %d", k, gotV, v)
		}
	}
}

func TestTraceability_JSONTags(t *testing.T) {
	tr := Traceability{
		Entries:  map[string][]TraceEntry{},
		Coverage: map[string]int{"src/main.go": 50},
	}

	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	if !strings.Contains(raw, `"entries"`) {
		t.Errorf("JSON missing tag %q in: %s", "entries", raw)
	}
	if !strings.Contains(raw, `"coverage"`) {
		t.Errorf("JSON missing tag %q in: %s", "coverage", raw)
	}
}

func TestTraceability_CoverageOmittedWhenEmpty(t *testing.T) {
	tr := Traceability{
		Entries:  map[string][]TraceEntry{},
		Coverage: nil,
	}

	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	if strings.Contains(raw, `"coverage"`) {
		t.Errorf("coverage should be omitted when nil, got: %s", raw)
	}
}

func TestTraceability_EmptyEntries(t *testing.T) {
	tr := Traceability{
		Entries:  map[string][]TraceEntry{},
		Coverage: map[string]int{},
	}

	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got Traceability
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(got.Entries) != 0 {
		t.Errorf("expected empty Entries, got length %d", len(got.Entries))
	}
}
