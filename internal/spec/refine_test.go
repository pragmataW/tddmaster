package spec

import (
	"encoding/json"
	"reflect"
	"strings"
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

func depsPtr(deps ...string) *[]string {
	s := append([]string{}, deps...)
	return &s
}

func TestApplyRefinement_Update_SetsDependsOn(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B"},
	}
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-2": {DependsOn: depsPtr("task-1")}},
	}, false, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if !reflect.DeepEqual(result[1].DependsOn, []string{"task-1"}) {
		t.Fatalf("got DependsOn %v, want [task-1]", result[1].DependsOn)
	}
}

func TestApplyRefinement_Update_NilDependsOn_Preserves(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B", DependsOn: []string{"task-1"}},
	}
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-2": {Title: strPtr("B2")}},
	}, false, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if !reflect.DeepEqual(result[1].DependsOn, []string{"task-1"}) {
		t.Fatalf("got DependsOn %v, want [task-1] — nil DependsOn must not erase existing", result[1].DependsOn)
	}
}

func TestApplyRefinement_Update_EmptyDependsOn_Clears(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B", DependsOn: []string{"task-1"}},
	}
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-2": {DependsOn: depsPtr()}},
	}, false, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if len(result[1].DependsOn) != 0 {
		t.Fatalf("got DependsOn %v, want empty — empty slice must clear dependencies", result[1].DependsOn)
	}
}

func TestApplyRefinement_Add_WithDependsOn(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "A"},
	}
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Add: []RefineOp{{Title: strPtr("B"), DependsOn: depsPtr("task-1")}},
	}, false, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if !reflect.DeepEqual(result[1].DependsOn, []string{"task-1"}) {
		t.Fatalf("got DependsOn %v, want [task-1]", result[1].DependsOn)
	}
}

func TestApplyRefinement_Add_NoDependsOn_DefaultsEmptyNonNil(t *testing.T) {
	result, _, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: strPtr("A")}},
	}, false, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if result[0].DependsOn == nil {
		t.Fatal("got nil DependsOn, want non-nil empty slice for new task")
	}
	if len(result[0].DependsOn) != 0 {
		t.Fatalf("got DependsOn %v, want empty", result[0].DependsOn)
	}
}

func TestApplyRefinement_Remove_WithDependents_ReturnsError(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B", DependsOn: []string{"task-1"}},
		{ID: "task-3", Title: "C", DependsOn: []string{"task-1"}},
	}
	_, _, err := ApplyRefinement(tasks, RefinePayload{Remove: []string{"task-1"}}, false, 0)
	if err == nil {
		t.Fatal("got nil error, want non-nil when removing a task with dependents")
	}
	msg := err.Error()
	if !strings.Contains(msg, "cannot remove task-1") {
		t.Errorf("error %q missing 'cannot remove task-1'", msg)
	}
	if !strings.Contains(msg, "task-2") || !strings.Contains(msg, "task-3") {
		t.Errorf("error %q missing dependent task ids task-2/task-3", msg)
	}
	if !strings.Contains(msg, "dependsOn in the same payload") {
		t.Errorf("error %q missing same-payload hint", msg)
	}
}

func TestApplyRefinement_RemoveAndUpdate_SamePayload_Succeeds(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B", DependsOn: []string{"task-1"}},
	}
	result, _, err := ApplyRefinement(tasks, RefinePayload{
		Remove: []string{"task-1"},
		Update: map[string]RefineOp{"task-2": {DependsOn: depsPtr()}},
	}, false, 0)
	if err != nil {
		t.Fatalf("got error %v, want nil — dependents fixed in the same payload", err)
	}
	if len(result) != 1 || result[0].ID != "task-2" {
		t.Fatalf("got %+v, want only task-2", result)
	}
	if len(result[0].DependsOn) != 0 {
		t.Fatalf("got DependsOn %v, want empty", result[0].DependsOn)
	}
}

func TestApplyRefinement_Update_SelfDependency_ReturnsError(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "A"},
	}
	_, _, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {DependsOn: depsPtr("task-1")}},
	}, false, 0)
	if err == nil {
		t.Fatal("got nil error, want non-nil for self-dependency")
	}
	if !strings.Contains(err.Error(), "task-1") {
		t.Errorf("error %q missing offending task id", err.Error())
	}
}

func TestApplyRefinement_Update_UnknownDependency_ReturnsError(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "A"},
	}
	_, _, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-1": {DependsOn: depsPtr("task-99")}},
	}, false, 0)
	if err == nil {
		t.Fatal("got nil error, want non-nil for unknown dependency id")
	}
	if !strings.Contains(err.Error(), "task-99") {
		t.Errorf("error %q missing invalid id task-99", err.Error())
	}
}

func TestApplyRefinement_Update_Cycle_ReturnsError(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "A", DependsOn: []string{"task-2"}},
		{ID: "task-2", Title: "B"},
	}
	_, _, err := ApplyRefinement(tasks, RefinePayload{
		Update: map[string]RefineOp{"task-2": {DependsOn: depsPtr("task-1")}},
	}, false, 0)
	if err == nil {
		t.Fatal("got nil error, want non-nil for dependency cycle")
	}
	msg := err.Error()
	if !strings.Contains(msg, "cycle") {
		t.Errorf("error %q missing 'cycle'", msg)
	}
	if !strings.Contains(msg, "task-1") || !strings.Contains(msg, "task-2") {
		t.Errorf("error %q missing cycle task ids", msg)
	}
}

func TestApplyRefinement_Add_UnknownDependency_ReturnsError(t *testing.T) {
	_, _, err := ApplyRefinement([]Task{}, RefinePayload{
		Add: []RefineOp{{Title: strPtr("A"), DependsOn: depsPtr("task-42")}},
	}, false, 0)
	if err == nil {
		t.Fatal("got nil error, want non-nil for add with unknown dependency")
	}
	if !strings.Contains(err.Error(), "task-42") {
		t.Errorf("error %q missing invalid id task-42", err.Error())
	}
}

func TestRefineOp_DependsOn_JSONNilVsEmpty(t *testing.T) {
	var absent RefineOp
	if err := json.Unmarshal([]byte(`{"title":"T"}`), &absent); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if absent.DependsOn != nil {
		t.Fatalf("got DependsOn %v, want nil pointer when field absent", absent.DependsOn)
	}
	var empty RefineOp
	if err := json.Unmarshal([]byte(`{"dependsOn":[]}`), &empty); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if empty.DependsOn == nil {
		t.Fatal("got nil DependsOn pointer, want non-nil for explicit empty list")
	}
	if len(*empty.DependsOn) != 0 {
		t.Fatalf("got DependsOn %v, want empty slice", *empty.DependsOn)
	}
}

func TestApplyRefinement_RemoveWithDependents_AndUnknownDep_BothReported(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B"},
		{ID: "task-3", Title: "C", DependsOn: []string{"task-2"}},
	}
	payload := RefinePayload{
		Remove: []string{"task-2"},
		Add:    []RefineOp{{Title: strPtr("D"), DependsOn: &[]string{"task-9"}}},
	}
	_, _, err := ApplyRefinement(tasks, payload, false, 0)
	if err == nil {
		t.Fatal("expected error for removal-with-dependents plus unknown dep")
	}
	msg := err.Error()
	if !strings.Contains(msg, "cannot remove task-2") {
		t.Errorf("error %q missing 'cannot remove task-2'", msg)
	}
	if !strings.Contains(msg, "task-9") {
		t.Errorf("error %q missing unknown dep task-9", msg)
	}
}

func TestApplyRefinement_Remove_DuplicateID_ReturnsError(t *testing.T) {
	tasks := []Task{
		{ID: "task-1", Title: "Alpha"},
		{ID: "task-2", Title: "Beta"},
	}
	_, _, err := ApplyRefinement(tasks, RefinePayload{Remove: []string{"task-2", "task-2"}}, false, 0)
	if err == nil {
		t.Fatal("duplicate remove id must error, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate task id in remove: task-2") {
		t.Fatalf("expected duplicate-remove error, got %q", err.Error())
	}
}
