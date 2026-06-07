package loop

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/spec"
)

func makeExecState(cycle string) spec.ExecState {
	return spec.ExecState{
		TDDCycle:        cycle,
		RefactorRounds:  0,
		RefactorApplied: false,
	}
}

func makeTask(id string, tddEnabled bool, done bool) spec.Task {
	return spec.Task{
		ID:         id,
		Title:      id,
		TDDEnabled: tddEnabled,
		Done:       done,
	}
}

func TestRefactorCapReached_MaxZero_NeverReached(t *testing.T) {
	cases := []struct {
		rounds int
		max    int
	}{
		{0, 0},
		{99, 0},
		{1000, 0},
	}
	for _, tc := range cases {
		if refactorCapReached(tc.rounds, tc.max) {
			t.Errorf("refactorCapReached(%d, %d): got true, want false (unlimited)", tc.rounds, tc.max)
		}
	}
}

func TestRefactorCapReached_MaxPositive_CapEnforcedAtBoundary(t *testing.T) {
	cases := []struct {
		rounds int
		max    int
		want   bool
	}{
		{2, 3, false},
		{3, 3, true},
		{4, 3, true},
		{0, 1, false},
		{1, 1, true},
	}
	for _, tc := range cases {
		got := refactorCapReached(tc.rounds, tc.max)
		if got != tc.want {
			t.Errorf("refactorCapReached(%d, %d): got %v, want %v", tc.rounds, tc.max, got, tc.want)
		}
	}
}

func TestAdvanceCycle_Red_Passed_TransitionsToGreen(t *testing.T) {
	st := makeExecState("red")

	got, taskComplete := advanceCycle(st, true, false, 3)

	if got.TDDCycle != "green" {
		t.Errorf("TDDCycle: got %q, want %q", got.TDDCycle, "green")
	}
	if taskComplete {
		t.Error("taskComplete: got true, want false (red→green is not task completion)")
	}
}

func TestAdvanceCycle_Red_Failed_StaysRed(t *testing.T) {
	st := makeExecState("red")

	got, taskComplete := advanceCycle(st, false, false, 3)

	if got.TDDCycle != "red" {
		t.Errorf("TDDCycle: got %q, want %q", got.TDDCycle, "red")
	}
	if taskComplete {
		t.Error("taskComplete: got true, want false")
	}
}

func TestAdvanceCycle_Green_Passed_NotesPresent_TransitionsToRefactor(t *testing.T) {
	st := makeExecState("green")

	got, taskComplete := advanceCycle(st, true, true, 3)

	if got.TDDCycle != "refactor" {
		t.Errorf("TDDCycle: got %q, want %q", got.TDDCycle, "refactor")
	}
	if taskComplete {
		t.Error("taskComplete: got true, want false (notes present → enter refactor)")
	}
}

func TestAdvanceCycle_Green_Passed_NotesPresent_ResetsRefactorCounters(t *testing.T) {
	st := makeExecState("green")
	st.RefactorRounds = 5
	st.RefactorApplied = true

	got, _ := advanceCycle(st, true, true, 3)

	if got.RefactorRounds != 0 {
		t.Errorf("RefactorRounds: got %d, want 0 (reset on entering refactor)", got.RefactorRounds)
	}
	if got.RefactorApplied {
		t.Error("RefactorApplied: got true, want false (reset on entering refactor)")
	}
}

func TestAdvanceCycle_Green_Passed_NoNotes_TaskComplete(t *testing.T) {
	st := makeExecState("green")

	got, taskComplete := advanceCycle(st, true, false, 3)

	if !taskComplete {
		t.Error("taskComplete: got false, want true (green+no notes → done)")
	}
	if got.TDDCycle != "" {
		t.Errorf("TDDCycle: got %q, want empty (task done)", got.TDDCycle)
	}
}

func TestAdvanceCycle_Green_Failed_StaysGreen(t *testing.T) {
	st := makeExecState("green")

	got, taskComplete := advanceCycle(st, false, false, 3)

	if got.TDDCycle != "green" {
		t.Errorf("TDDCycle: got %q, want %q", got.TDDCycle, "green")
	}
	if taskComplete {
		t.Error("taskComplete: got true, want false")
	}
}

func TestAdvanceCycle_Refactor_Passed_NotesPresent_UnderCap_StaysRefactor(t *testing.T) {
	st := makeExecState("refactor")
	st.RefactorApplied = true
	st.RefactorRounds = 0

	got, taskComplete := advanceCycle(st, true, true, 3)

	if got.TDDCycle != "refactor" {
		t.Errorf("TDDCycle: got %q, want %q", got.TDDCycle, "refactor")
	}
	if taskComplete {
		t.Error("taskComplete: got true, want false (under cap, notes present → stay refactor)")
	}
}

func TestAdvanceCycle_Refactor_Passed_NotesPresent_UnderCap_IncrementsRounds(t *testing.T) {
	st := makeExecState("refactor")
	st.RefactorApplied = true
	st.RefactorRounds = 1

	got, _ := advanceCycle(st, true, true, 3)

	if got.RefactorRounds != 2 {
		t.Errorf("RefactorRounds: got %d, want 2 (incremented)", got.RefactorRounds)
	}
}

func TestAdvanceCycle_Refactor_Passed_CapReached_TaskComplete(t *testing.T) {
	st := makeExecState("refactor")
	st.RefactorApplied = true
	st.RefactorRounds = 2

	got, taskComplete := advanceCycle(st, true, true, 3)

	if !taskComplete {
		t.Error("taskComplete: got false, want true (cap reached)")
	}
	if got.TDDCycle != "" {
		t.Errorf("TDDCycle: got %q, want empty (cap reached)", got.TDDCycle)
	}
}

func TestAdvanceCycle_Refactor_Passed_CapReached_RoundsIncremented(t *testing.T) {
	st := makeExecState("refactor")
	st.RefactorApplied = true
	st.RefactorRounds = 2

	got, _ := advanceCycle(st, true, true, 3)

	if got.RefactorRounds != 3 {
		t.Errorf("RefactorRounds: got %d, want 3 (incremented before cap check)", got.RefactorRounds)
	}
}

func TestAdvanceCycle_Refactor_Passed_NoNotes_TaskComplete(t *testing.T) {
	st := makeExecState("refactor")
	st.RefactorApplied = true
	st.RefactorRounds = 0

	got, taskComplete := advanceCycle(st, true, false, 3)

	if !taskComplete {
		t.Error("taskComplete: got false, want true (no notes → done)")
	}
	if got.TDDCycle != "" {
		t.Errorf("TDDCycle: got %q, want empty", got.TDDCycle)
	}
}

func TestAdvanceCycle_Refactor_UnlimitedMax_NeverCompletesOnCountAlone(t *testing.T) {
	st := makeExecState("refactor")
	st.RefactorApplied = true
	st.RefactorRounds = 99

	got, taskComplete := advanceCycle(st, true, true, 0)

	if taskComplete {
		t.Error("taskComplete: got true, want false (max=0 means unlimited)")
	}
	if got.TDDCycle != "refactor" {
		t.Errorf("TDDCycle: got %q, want %q (unlimited rounds)", got.TDDCycle, "refactor")
	}
}

func TestAdvanceCycle_Refactor_Passed_NotApplied_WithNotes_RefactorBypassGuard(t *testing.T) {
	st := makeExecState("refactor")
	st.RefactorApplied = false
	st.RefactorRounds = 0

	_, _, err := advanceCycleStrict(st, true, true, 3)

	if err == nil {
		t.Error("expected error: completing refactor with pending notes before applying is forbidden")
	}
}

func TestAdvanceCycle_Refactor_Passed_NotApplied_NoNotes_Allowed(t *testing.T) {
	st := makeExecState("refactor")
	st.RefactorApplied = false
	st.RefactorRounds = 0

	_, taskComplete, err := advanceCycleStrict(st, true, false, 3)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !taskComplete {
		t.Error("taskComplete: got false, want true (no notes, no pending work)")
	}
}

func TestAdvanceCycle_Refactor_Failed_StaysRefactor(t *testing.T) {
	st := makeExecState("refactor")
	st.RefactorApplied = true
	st.RefactorRounds = 0

	got, taskComplete := advanceCycle(st, false, false, 3)

	if got.TDDCycle != "refactor" {
		t.Errorf("TDDCycle: got %q, want %q", got.TDDCycle, "refactor")
	}
	if taskComplete {
		t.Error("taskComplete: got true, want false (failed verification)")
	}
}

func TestAdvanceCycle_EmptyCycle_Passed_NoOp(t *testing.T) {
	st := makeExecState("")

	got, taskComplete := advanceCycle(st, true, false, 3)

	if got.TDDCycle != "" {
		t.Errorf("TDDCycle: got %q, want empty (no-op when cycle not set)", got.TDDCycle)
	}
	if taskComplete {
		t.Error("taskComplete: got true, want false (empty cycle is a no-op)")
	}
}

func TestCompleteCurrentTask_MarksDone(t *testing.T) {
	tasks := []spec.Task{
		makeTask("t-1", true, false),
		makeTask("t-2", true, false),
	}

	result := completeCurrentTask(tasks, 0)

	if !result[0].Done {
		t.Error("tasks[0].Done: got false, want true")
	}
	if result[1].Done {
		t.Error("tasks[1].Done: got true, want false (should not be touched)")
	}
}

func TestCompleteCurrentTask_LastIndex(t *testing.T) {
	tasks := []spec.Task{
		makeTask("t-1", true, false),
		makeTask("t-2", true, false),
	}

	result := completeCurrentTask(tasks, 1)

	if !result[1].Done {
		t.Error("tasks[1].Done: got false, want true")
	}
	if result[0].Done {
		t.Error("tasks[0].Done: got true, want false")
	}
}

func TestCompleteCurrentTask_Idempotent_AlreadyDone(t *testing.T) {
	tasks := []spec.Task{
		makeTask("t-1", true, true),
	}

	result := completeCurrentTask(tasks, 0)

	if !result[0].Done {
		t.Error("tasks[0].Done: got false, want true (already done must stay done)")
	}
}

func TestCompleteCurrentTask_DoesNotModifyOriginalSlice(t *testing.T) {
	tasks := []spec.Task{
		makeTask("t-1", true, false),
	}

	result := completeCurrentTask(tasks, 0)

	if tasks[0].Done {
		t.Error("original slice tasks[0].Done was mutated, want immutable original")
	}
	if !result[0].Done {
		t.Error("result[0].Done: got false, want true")
	}
}

func TestReseedCycle_NextTaskTDDEnabled_SetsRed(t *testing.T) {
	st := makeExecState("")
	st.RefactorRounds = 3
	st.RefactorApplied = true

	got := reseedCycle(st, true)

	if got.TDDCycle != "red" {
		t.Errorf("TDDCycle: got %q, want %q", got.TDDCycle, "red")
	}
}

func TestReseedCycle_NextTaskTDDEnabled_ClearsRefactorCounters(t *testing.T) {
	st := makeExecState("refactor")
	st.RefactorRounds = 3
	st.RefactorApplied = true

	got := reseedCycle(st, true)

	if got.RefactorRounds != 0 {
		t.Errorf("RefactorRounds: got %d, want 0 (cleared for new task)", got.RefactorRounds)
	}
	if got.RefactorApplied {
		t.Error("RefactorApplied: got true, want false (cleared for new task)")
	}
}

func TestReseedCycle_NextTaskTDDDisabled_SetsEmpty(t *testing.T) {
	st := makeExecState("")

	got := reseedCycle(st, false)

	if got.TDDCycle != "" {
		t.Errorf("TDDCycle: got %q, want empty (TDD disabled)", got.TDDCycle)
	}
}

func TestReseedCycle_NextTaskTDDDisabled_ClearsCounters(t *testing.T) {
	st := makeExecState("green")
	st.RefactorRounds = 2
	st.RefactorApplied = true

	got := reseedCycle(st, false)

	if got.RefactorRounds != 0 {
		t.Errorf("RefactorRounds: got %d, want 0", got.RefactorRounds)
	}
	if got.RefactorApplied {
		t.Error("RefactorApplied: got true, want false")
	}
}

func TestReseedCycle_MixedTDDEnabled_PerTaskControl(t *testing.T) {
	stTDD := makeExecState("")
	gotTDD := reseedCycle(stTDD, true)
	if gotTDD.TDDCycle != "red" {
		t.Errorf("TDD task: TDDCycle got %q, want %q", gotTDD.TDDCycle, "red")
	}

	stNoTDD := makeExecState("")
	gotNoTDD := reseedCycle(stNoTDD, false)
	if gotNoTDD.TDDCycle != "" {
		t.Errorf("non-TDD task: TDDCycle got %q, want empty", gotNoTDD.TDDCycle)
	}
}

func TestAdvanceCycle_TableDriven_AllCycles(t *testing.T) {
	type testCase struct {
		name                string
		cycle               string
		passed              bool
		refactorNotes       bool
		refactorApplied     bool
		refactorRounds      int
		maxRefactorRounds   int
		wantCycle           string
		wantTaskComplete    bool
	}

	cases := []testCase{
		{
			name:             "red passed → green",
			cycle:            "red",
			passed:           true,
			wantCycle:        "green",
			wantTaskComplete: false,
		},
		{
			name:             "green passed + notes → refactor",
			cycle:            "green",
			passed:           true,
			refactorNotes:    true,
			wantCycle:        "refactor",
			wantTaskComplete: false,
		},
		{
			name:             "green passed + no notes → task done",
			cycle:            "green",
			passed:           true,
			refactorNotes:    false,
			wantCycle:        "",
			wantTaskComplete: true,
		},
		{
			name:             "refactor applied + notes under cap → stay refactor",
			cycle:            "refactor",
			passed:           true,
			refactorNotes:    true,
			refactorApplied:  true,
			refactorRounds:   1,
			maxRefactorRounds: 3,
			wantCycle:        "refactor",
			wantTaskComplete: false,
		},
		{
			name:             "refactor applied + notes at cap → task done",
			cycle:            "refactor",
			passed:           true,
			refactorNotes:    true,
			refactorApplied:  true,
			refactorRounds:   2,
			maxRefactorRounds: 3,
			wantCycle:        "",
			wantTaskComplete: true,
		},
		{
			name:             "refactor applied + no notes → task done",
			cycle:            "refactor",
			passed:           true,
			refactorNotes:    false,
			refactorApplied:  true,
			refactorRounds:   0,
			maxRefactorRounds: 3,
			wantCycle:        "",
			wantTaskComplete: true,
		},
		{
			name:             "refactor unlimited + notes present → stay refactor",
			cycle:            "refactor",
			passed:           true,
			refactorNotes:    true,
			refactorApplied:  true,
			refactorRounds:   99,
			maxRefactorRounds: 0,
			wantCycle:        "refactor",
			wantTaskComplete: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := makeExecState(tc.cycle)
			st.RefactorApplied = tc.refactorApplied
			st.RefactorRounds = tc.refactorRounds

			got, taskComplete := advanceCycle(st, tc.passed, tc.refactorNotes, tc.maxRefactorRounds)

			if got.TDDCycle != tc.wantCycle {
				t.Errorf("TDDCycle: got %q, want %q", got.TDDCycle, tc.wantCycle)
			}
			if taskComplete != tc.wantTaskComplete {
				t.Errorf("taskComplete: got %v, want %v", taskComplete, tc.wantTaskComplete)
			}
		})
	}
}
