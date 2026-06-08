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

func TestTraceability_JSONRoundTrip(t *testing.T) {
	original := Traceability{
		"task-1": []TraceEntry{
			{FunctionName: "TestAlpha", TaskID: "task-1", AC: []string{"ac1"}, EC: nil},
			{FunctionName: "TestBeta", TaskID: "task-1", AC: []string{"ac2"}, EC: []string{"EC-1"}},
		},
		"task-2": []TraceEntry{
			{FunctionName: "TestGamma", TaskID: "task-2", AC: []string{"ac1"}, EC: []string{"EC-2"}},
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

	if len(got) != len(original) {
		t.Fatalf("Traceability length: got %d, want %d", len(got), len(original))
	}

	for taskID, entries := range original {
		gotEntries, ok := got[taskID]
		if !ok {
			t.Errorf("missing key %q in round-tripped Traceability", taskID)
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

func TestTraceability_EmptyMap(t *testing.T) {
	tr := Traceability{}

	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var got Traceability
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("expected empty Traceability, got length %d", len(got))
	}
}
