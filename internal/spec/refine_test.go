package spec

import (
	"encoding/json"
	"reflect"
	"testing"
)

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func TestApplyRefinement_EmptyPayload_ReturnsUnchanged(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "Alpha", AC: []string{"ac1"}, Done: false, TDDEnabled: true, Important: false},
	}
	result, err := ApplyRefinement(tasks, RefinePayload{})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if !reflect.DeepEqual(result, tasks) {
		t.Fatalf("got %+v, want %+v", result, tasks)
	}
}

func TestApplyRefinement_Remove_DropsMatchedTask(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "Alpha"},
		{ID: "task-2", Title: "Beta"},
	}
	result, err := ApplyRefinement(tasks, RefinePayload{Remove: []string{"task-1"}})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if len(result) != 1 {
		t.Fatalf("got len %d, want 1", len(result))
	}
	if result[0].ID != "task-2" {
		t.Fatalf("got id %q, want task-2", result[0].ID)
	}
}

func TestApplyRefinement_Remove_UnknownID_ReturnsError(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "Alpha"},
	}
	_, err := ApplyRefinement(tasks, RefinePayload{Remove: []string{"task-99"}})
	if err == nil {
		t.Fatal("got nil error, want non-nil for unknown remove id")
	}
}

func TestApplyRefinement_Update_SetsTitle(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "Old Title"},
	}
	result, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {Title: strPtr("New Title")}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].Title != "New Title" {
		t.Fatalf("got title %q, want New Title", result[0].Title)
	}
}

func TestApplyRefinement_Update_NilTitle_KeepsExisting(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "Keep Me"},
	}
	result, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {Title: nil}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].Title != "Keep Me" {
		t.Fatalf("got title %q, want Keep Me", result[0].Title)
	}
}

func TestApplyRefinement_Update_ReplacesACList(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "T", AC: []string{"old-ac"}},
	}
	result, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {AC: []string{"new-ac1", "new-ac2"}}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if !reflect.DeepEqual(result[0].AC, []string{"new-ac1", "new-ac2"}) {
		t.Fatalf("got AC %v, want [new-ac1 new-ac2]", result[0].AC)
	}
}

func TestApplyRefinement_Update_NilAC_KeepsExisting(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "T", AC: []string{"keep-this"}},
	}
	result, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if !reflect.DeepEqual(result[0].AC, []string{"keep-this"}) {
		t.Fatalf("got AC %v, want [keep-this]", result[0].AC)
	}
}

func TestApplyRefinement_Update_TDDEnabled_FalsePtr_FlipsTrue(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "T", TDDEnabled: true},
	}
	result, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {TDDEnabled: boolPtr(false)}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].TDDEnabled != false {
		t.Fatal("got TDDEnabled true, want false")
	}
}

func TestApplyRefinement_Update_TDDEnabled_NilPtr_KeepsExisting(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "T", TDDEnabled: true},
	}
	result, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {TDDEnabled: nil}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].TDDEnabled != true {
		t.Fatal("got TDDEnabled false, want true")
	}
}

func TestApplyRefinement_Update_Important_FalsePtr_FlipsTrue(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "T", Important: true},
	}
	result, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {Important: boolPtr(false)}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].Important != false {
		t.Fatal("got Important true, want false")
	}
}

func TestApplyRefinement_Update_Important_NilPtr_KeepsExisting(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "T", Important: true},
	}
	result, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {Important: nil}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].Important != true {
		t.Fatal("got Important false, want true")
	}
}

func TestApplyRefinement_Update_UnknownID_ReturnsError(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "T"},
	}
	_, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-99": {Title: strPtr("X")}},
	})
	if err == nil {
		t.Fatal("got nil error, want non-nil for unknown update id")
	}
}

func TestApplyRefinement_Update_DoneFlag_NeverModified(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "Old", Done: true},
	}
	result, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {Title: strPtr("New")}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].Done != true {
		t.Fatal("got Done false, want true — Done must not be modified by refine")
	}
}

func TestApplyRefinement_Add_AutoID_StartsAtTaskOne_WhenEmpty(t *testing.T) {
	result, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: strPtr("New Task")}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if len(result) != 1 {
		t.Fatalf("got len %d, want 1", len(result))
	}
	if result[0].ID != "task-1" {
		t.Fatalf("got id %q, want task-1", result[0].ID)
	}
}

func TestApplyRefinement_Add_AutoID_MaxPlusOne(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B"},
	}
	result, err := ApplyRefinement(tasks, RefinePayload{
		Add: []RefineOp{{Title: strPtr("C")}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[2].ID != "task-3" {
		t.Fatalf("got id %q, want task-3", result[2].ID)
	}
}

func TestApplyRefinement_Add_AutoID_IgnoresNonNumericIDs(t *testing.T) {
	tasks := []Task{
		{ID: "garbage-id", Title: "A"},
		{ID: "task-2", Title: "B"},
	}
	result, err := ApplyRefinement(tasks, RefinePayload{
		Add: []RefineOp{{Title: strPtr("C")}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[2].ID != "task-3" {
		t.Fatalf("got id %q, want task-3 (garbage ids must be ignored)", result[2].ID)
	}
}

func TestApplyRefinement_Add_AutoID_OnlyNonNumericIDs_FallsBackToTaskOne(t *testing.T) {
	tasks := []Task{
		{ID: "garbage-id", Title: "A"},
	}
	result, err := ApplyRefinement(tasks, RefinePayload{
		Add: []RefineOp{{Title: strPtr("New")}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[1].ID != "task-1" {
		t.Fatalf("got id %q, want task-1 when no numeric task ids exist", result[1].ID)
	}
}

func TestApplyRefinement_Add_MultipleAdds_IncrementSequentially(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "Existing"},
	}
	result, err := ApplyRefinement(tasks, RefinePayload{
		Add: []RefineOp{
			{Title: strPtr("Second")},
			{Title: strPtr("Third")},
		},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[1].ID != "task-2" {
		t.Fatalf("got id %q, want task-2", result[1].ID)
	}
	if result[2].ID != "task-3" {
		t.Fatalf("got id %q, want task-3", result[2].ID)
	}
}

func TestApplyRefinement_Add_NilTitle_ReturnsError(t *testing.T) {
	_, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: nil}},
	})
	if err == nil {
		t.Fatal("got nil error, want non-nil for nil title in add op")
	}
}

func TestApplyRefinement_Add_EmptyTitle_ReturnsError(t *testing.T) {
	_, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: strPtr("")}},
	})
	if err == nil {
		t.Fatal("got nil error, want non-nil for empty title in add op")
	}
}

func TestApplyRefinement_Add_NewTaskDoneIsFalse(t *testing.T) {
	result, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: strPtr("T")}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].Done != false {
		t.Fatal("got Done true, want false for newly added task")
	}
}

func TestApplyRefinement_Add_TDDEnabled_DefaultsFalse(t *testing.T) {
	result, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: strPtr("T")}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].TDDEnabled != false {
		t.Fatal("got TDDEnabled true, want false when not set in add op")
	}
}

func TestApplyRefinement_Add_TDDEnabled_TrueWhenSet(t *testing.T) {
	result, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: strPtr("T"), TDDEnabled: boolPtr(true)}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].TDDEnabled != true {
		t.Fatal("got TDDEnabled false, want true when set in add op")
	}
}

func TestApplyRefinement_Add_Important_DefaultsFalse(t *testing.T) {
	result, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: strPtr("T")}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].Important != false {
		t.Fatal("got Important true, want false when not set in add op")
	}
}

func TestApplyRefinement_Add_AC_EmptyWhenNotSet(t *testing.T) {
	result, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: strPtr("T")}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if len(result[0].AC) != 0 {
		t.Fatalf("got AC %v, want empty slice", result[0].AC)
	}
}

func TestApplyRefinement_Order_RemoveThenAdd_IDReflectsPostRemoveMax(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B"},
		{ID: "task-3", Title: "C"},
	}
	result, err := ApplyRefinement(tasks, RefinePayload{
		Remove: []string{"task-3"},
		Add:    []RefineOp{{Title: strPtr("D")}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if len(result) != 3 {
		t.Fatalf("got len %d, want 3", len(result))
	}
	if result[2].ID != "task-3" {
		t.Fatalf("got id %q, want task-3 (max after remove is task-2, so new = task-3)", result[2].ID)
	}
}

func TestApplyRefinement_Order_RemoveUpdateAdd_Sequence(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B"},
	}
	result, err := ApplyRefinement(tasks, RefinePayload{
		Remove: []string{"task-1"},
		Update: map[string]RefineOp{"task-2": {Title: strPtr("B-updated")}},
		Add:    []RefineOp{{Title: strPtr("C")}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].Title != "B-updated" {
		t.Fatalf("got title %q, want B-updated", result[0].Title)
	}
	if result[1].ID != "task-3" {
		t.Fatalf("got id %q, want task-3", result[1].ID)
	}
}

func TestRefineOp_EdgeCases_JSONTag(t *testing.T) {
	var op RefineOp
	if err := json.Unmarshal([]byte(`{"edgeCases":["ec-1","ec-2"]}`), &op); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(op.EdgeCases) != 2 {
		t.Fatalf("got EdgeCases len %d, want 2", len(op.EdgeCases))
	}
	if op.EdgeCases[0] != "ec-1" || op.EdgeCases[1] != "ec-2" {
		t.Fatalf("got EdgeCases %v, want [ec-1 ec-2]", op.EdgeCases)
	}
}

func TestApplyRefinement_Update_SetsEdgeCases(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "T", AC: []string{"ac1"}},
	}
	result, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {EdgeCases: []string{"ec-x"}}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if !reflect.DeepEqual(result[0].EdgeCases, []string{"ec-x"}) {
		t.Fatalf("got EdgeCases %v, want [ec-x]", result[0].EdgeCases)
	}
}

func TestApplyRefinement_Update_NilEdgeCases_Preserves(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "T", AC: []string{"ac1"}, EdgeCases: []string{"old"}},
	}
	result, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {Title: strPtr("T2")}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if !reflect.DeepEqual(result[0].EdgeCases, []string{"old"}) {
		t.Fatalf("got EdgeCases %v, want [old] — nil EdgeCases must not erase existing", result[0].EdgeCases)
	}
}

func TestApplyRefinement_Add_WithEdgeCases(t *testing.T) {
	result, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: strPtr("New"), AC: []string{"a"}, EdgeCases: []string{"ec-1"}}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if len(result) != 1 {
		t.Fatalf("got len %d, want 1", len(result))
	}
	if !reflect.DeepEqual(result[0].EdgeCases, []string{"ec-1"}) {
		t.Fatalf("got EdgeCases %v, want [ec-1]", result[0].EdgeCases)
	}
}

func TestApplyRefinement_ReturnedSlice_OrderIsOriginalThenAdded(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B"},
	}
	result, err := ApplyRefinement(tasks, RefinePayload{
		Add: []RefineOp{{Title: strPtr("C")}},
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].ID != "task-1" || result[1].ID != "task-2" || result[2].ID != "task-3" {
		t.Fatalf("got order %v/%v/%v, want task-1/task-2/task-3",
			result[0].ID, result[1].ID, result[2].ID)
	}
}
