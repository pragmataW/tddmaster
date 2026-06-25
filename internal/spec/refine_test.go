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
		{ID: "task-1", Title: "Alpha", Criteria: []Criterion{{ID: "ac-1", Then: "ac1"}}, Done: false, TDDEnabled: true, Important: false},
	}
	result, _, err := ApplyRefinement(tasks, RefinePayload{}, false, 0)
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
	result, _, err := ApplyRefinement(tasks, RefinePayload{Remove: []string{"task-1"}}, false, 0)
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
	_, _, err := ApplyRefinement(tasks, RefinePayload{Remove: []string{"task-99"}}, false, 0)
	if err == nil {
		t.Fatal("got nil error, want non-nil for unknown remove id")
	}
}

func TestApplyRefinement_Update_SetsTitle(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "Old Title"},
	}
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {Title: strPtr("New Title")}},
	}, false, 0)
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
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {Title: nil}},
	}, false, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].Title != "Keep Me" {
		t.Fatalf("got title %q, want Keep Me", result[0].Title)
	}
}

func TestApplyRefinement_Update_TDDEnabled_FalsePtr_FlipsTrue(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "T", TDDEnabled: true},
	}
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {TDDEnabled: boolPtr(false)}},
	}, false, 0)
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
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {TDDEnabled: nil}},
	}, false, 0)
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
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {Important: boolPtr(false)}},
	}, false, 0)
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
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {Important: nil}},
	}, false, 0)
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
	_, _, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-99": {Title: strPtr("X")}},
	}, false, 0)
	if err == nil {
		t.Fatal("got nil error, want non-nil for unknown update id")
	}
}

func TestApplyRefinement_Update_DoneFlag_NeverModified(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "Old", Done: true},
	}
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {Title: strPtr("New")}},
	}, false, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].Done != true {
		t.Fatal("got Done false, want true — Done must not be modified by refine")
	}
}

func TestApplyRefinement_Add_AutoID_StartsAtTaskOne_WhenEmpty(t *testing.T) {
	result, _, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: strPtr("New Task")}},
	}, false, 0)
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
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Add: []RefineOp{{Title: strPtr("C")}},
	}, false, 0)
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
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Add: []RefineOp{{Title: strPtr("C")}},
	}, false, 0)
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
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Add: []RefineOp{{Title: strPtr("New")}},
	}, false, 0)
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
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Add: []RefineOp{
			{Title: strPtr("Second")},
			{Title: strPtr("Third")},
		},
	}, false, 0)
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
	_, _, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: nil}},
	}, false, 0)
	if err == nil {
		t.Fatal("got nil error, want non-nil for nil title in add op")
	}
}

func TestApplyRefinement_Add_EmptyTitle_ReturnsError(t *testing.T) {
	_, _, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: strPtr("")}},
	}, false, 0)
	if err == nil {
		t.Fatal("got nil error, want non-nil for empty title in add op")
	}
}

func TestApplyRefinement_Add_NewTaskDoneIsFalse(t *testing.T) {
	result, _, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: strPtr("T")}},
	}, false, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].Done != false {
		t.Fatal("got Done true, want false for newly added task")
	}
}

func TestApplyRefinement_Add_TDDEnabled_DefaultsFalse(t *testing.T) {
	result, _, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: strPtr("T")}},
	}, false, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].TDDEnabled != false {
		t.Fatal("got TDDEnabled true, want false when not set in add op")
	}
}

func TestApplyRefinement_Add_TDDEnabled_TrueWhenSet(t *testing.T) {
	result, _, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: strPtr("T"), TDDEnabled: boolPtr(true)}},
	}, false, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].TDDEnabled != true {
		t.Fatal("got TDDEnabled false, want true when set in add op")
	}
}

func TestApplyRefinement_Add_Important_DefaultsFalse(t *testing.T) {
	result, _, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: strPtr("T")}},
	}, false, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].Important != false {
		t.Fatal("got Important true, want false when not set in add op")
	}
}

func TestApplyRefinement_Order_RemoveThenAdd_DoesNotReuseRemovedID(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B"},
		{ID: "task-3", Title: "C"},
	}
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Remove: []string{"task-3"},
		Add:    []RefineOp{{Title: strPtr("D")}},
	}, false, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if len(result) != 3 {
		t.Fatalf("got len %d, want 3", len(result))
	}
	if result[2].ID != "task-4" {
		t.Fatalf("got id %q, want task-4 (removed task-3 must not be reused)", result[2].ID)
	}
}

func TestApplyRefinement_Order_RemoveUpdateAdd_Sequence(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B"},
	}
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Remove: []string{"task-1"},
		Update: map[string]RefineOp{"task-2": {Title: strPtr("B-updated")}},
		Add:    []RefineOp{{Title: strPtr("C")}},
	}, false, 0)
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
		{ID: "task-1", Title: "T", Criteria: []Criterion{{ID: "ac-1", Then: "ac1"}}},
	}
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {EdgeCases: []string{"ec-x"}}},
	}, false, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if !reflect.DeepEqual(result[0].EdgeCases, []string{"ec-x"}) {
		t.Fatalf("got EdgeCases %v, want [ec-x]", result[0].EdgeCases)
	}
}

func TestApplyRefinement_Update_NilEdgeCases_Preserves(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "T", Criteria: []Criterion{{ID: "ac-1", Then: "ac1"}}, EdgeCases: []string{"old"}},
	}
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {Title: strPtr("T2")}},
	}, false, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if !reflect.DeepEqual(result[0].EdgeCases, []string{"old"}) {
		t.Fatalf("got EdgeCases %v, want [old] — nil EdgeCases must not erase existing", result[0].EdgeCases)
	}
}

func TestApplyRefinement_Add_WithEdgeCases(t *testing.T) {
	result, _, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: strPtr("New"), Criteria: []Criterion{{Then: "a"}}, EdgeCases: []string{"ec-1"}}},
	}, false, 0)
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
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Add: []RefineOp{{Title: strPtr("C")}},
	}, false, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].ID != "task-1" || result[1].ID != "task-2" || result[2].ID != "task-3" {
		t.Fatalf("got order %v/%v/%v, want task-1/task-2/task-3",
			result[0].ID, result[1].ID, result[2].ID)
	}
}

func TestApplyRefinement_Add_TDDEnabled_DefaultsToTDDDefault(t *testing.T) {
	result, _, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: strPtr("T")}},
	}, true, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].TDDEnabled != true {
		t.Fatal("got TDDEnabled false, want true when tddDefault is true and op leaves it unset")
	}
}

func TestApplyRefinement_Add_TDDEnabled_ExplicitFalse_OverridesDefault(t *testing.T) {
	result, _, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: strPtr("T"), TDDEnabled: boolPtr(false)}},
	}, true, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].TDDEnabled != false {
		t.Fatal("got TDDEnabled true, want false when op explicitly disables it")
	}
}

func TestApplyRefinement_Seq_PreventsIDReuseAcrossCalls(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B"},
		{ID: "task-3", Title: "C"},
	}
	afterRemove, seq, err := ApplyRefinement(tasks, RefinePayload{Remove: []string{"task-3"}}, false, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if seq != 3 {
		t.Fatalf("got seq %d, want 3 — removed IDs must stay reserved", seq)
	}
	result, seq2, err := ApplyRefinement(afterRemove, RefinePayload{
		Add: []RefineOp{{Title: strPtr("D")}},
	}, false, seq)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[len(result)-1].ID != "task-4" {
		t.Fatalf("got id %q, want task-4 — removed task-3 must not be reused across calls", result[len(result)-1].ID)
	}
	if seq2 != 4 {
		t.Fatalf("got seq %d, want 4", seq2)
	}
}

func TestApplyRefinement_Criteria_ReplacesTaskCriteria(t *testing.T) {
	tasks := []Task{
		{
			ID:       "task-1",
			Title:    "T",
			Criteria: []Criterion{{ID: "ac-1", Then: "old outcome"}},
		},
	}
	newCriteria := []Criterion{
		{Given: "new context", When: "new action", Then: "new outcome"},
	}
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {Criteria: newCriteria}},
	}, false, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if len(result[0].Criteria) != 1 {
		t.Fatalf("got Criteria len %d, want 1", len(result[0].Criteria))
	}
	if result[0].Criteria[0].Then != "new outcome" {
		t.Fatalf("got Then %q, want 'new outcome'", result[0].Criteria[0].Then)
	}
	if result[0].Criteria[0].ID == "" {
		t.Error("criterion ID is empty after update; want ac-N assigned by AssignCriterionIDs")
	}
}

func TestApplyRefinement_Criteria_IDsStableAcrossAddRemove(t *testing.T) {
	tasks := []Task{
		{
			ID:    "task-1",
			Title: "T",
			Criteria: []Criterion{
				{ID: "ac-1", Then: "first"},
				{ID: "ac-2", Then: "second"},
			},
		},
	}
	newCriterion := Criterion{Then: "third"}
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{
			"task-1": {
				Criteria: append(
					[]Criterion{{ID: "ac-1", Then: "first"}, {ID: "ac-2", Then: "second"}},
					newCriterion,
				),
			},
		},
	}, false, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if len(result[0].Criteria) != 3 {
		t.Fatalf("got Criteria len %d, want 3", len(result[0].Criteria))
	}
	if result[0].Criteria[0].ID != "ac-1" {
		t.Errorf("Criteria[0].ID = %q, want ac-1 (must remain stable)", result[0].Criteria[0].ID)
	}
	if result[0].Criteria[1].ID != "ac-2" {
		t.Errorf("Criteria[1].ID = %q, want ac-2 (must remain stable)", result[0].Criteria[1].ID)
	}
	if result[0].Criteria[2].ID != "ac-3" {
		t.Errorf("Criteria[2].ID = %q, want ac-3 (new criterion must get next available id)", result[0].Criteria[2].ID)
	}
}

func TestApplyRefinement_Criteria_NilCriteria_PreservesExisting(t *testing.T) {
	tasks := []Task{
		{
			ID:       "task-1",
			Title:    "T",
			Criteria: []Criterion{{ID: "ac-1", Then: "keep me"}},
		},
	}
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {Title: strPtr("T-updated")}},
	}, false, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if len(result[0].Criteria) != 1 || result[0].Criteria[0].ID != "ac-1" {
		t.Fatalf("got Criteria %v, want original ac-1 preserved when Criteria not set in op", result[0].Criteria)
	}
}

func TestApplyRefinement_Seq_TakesMaxOfSeqAndTaskIDs(t *testing.T) {
	tasks := []Task{
		{ID: "task-5", Title: "E"},
	}
	result, seq, err := ApplyRefinement(tasks, RefinePayload{
		Add: []RefineOp{{Title: strPtr("F")}},
	}, false, 2)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[1].ID != "task-6" {
		t.Fatalf("got id %q, want task-6", result[1].ID)
	}
	if seq != 6 {
		t.Fatalf("got seq %d, want 6", seq)
	}
}
