package cmd

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTDDSelectionAnswer_AllVariants(t *testing.T) {
	cases := []struct {
		in   string
		mode string
	}{
		{"tdd-all", "all"},
		{"TDD-ALL", "all"},
		{"all", "all"},
		{"tdd-none", "none"},
		{"NONE", "none"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			choice, err := parseTDDSelectionAnswer(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.mode, choice.Mode)
			assert.Empty(t, choice.TDDTasks)
		})
	}
}

func TestParseTDDSelectionAnswer_Custom(t *testing.T) {
	choice, err := parseTDDSelectionAnswer(`{"tddTasks":["task-1","task-3"]}`)
	require.NoError(t, err)
	assert.Equal(t, "custom", choice.Mode)
	assert.Equal(t, []string{"task-1", "task-3"}, choice.TDDTasks)
}

func TestParseTDDSelectionAnswer_CustomDropsInvalidIDs(t *testing.T) {
	// "foo" is not a task ID shape; it should be dropped silently.
	choice, err := parseTDDSelectionAnswer(`{"tddTasks":["task-1","foo","task-2"]}`)
	require.NoError(t, err)
	assert.Equal(t, []string{"task-1", "task-2"}, choice.TDDTasks)
}

func TestParseTDDSelectionAnswer_RejectsBadJSON(t *testing.T) {
	_, err := parseTDDSelectionAnswer(`not-a-known-keyword`)
	require.Error(t, err)
}

func TestParseTDDSelectionAnswer_RejectsMissingTDDTasksKey(t *testing.T) {
	_, err := parseTDDSelectionAnswer(`{"other":1}`)
	require.Error(t, err)
}

func TestApplyTDDSelection_AllMarksEveryTaskTrue(t *testing.T) {
	st := state.CreateInitialState()
	tasks := []state.SpecTask{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B"},
	}
	newSt := applyTDDSelectionToOverrides(st, tasks, tddSelectionChoice{Mode: "all"})
	require.Len(t, newSt.OverrideTasks, 2)
	for i, ot := range newSt.OverrideTasks {
		require.NotNil(t, ot.TDDEnabled, "task %d missing TDDEnabled", i)
		assert.True(t, *ot.TDDEnabled)
	}
}

func TestApplyTDDSelection_NoneMarksEveryTaskFalse(t *testing.T) {
	st := state.CreateInitialState()
	tasks := []state.SpecTask{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B"},
	}
	newSt := applyTDDSelectionToOverrides(st, tasks, tddSelectionChoice{Mode: "none"})
	require.Len(t, newSt.OverrideTasks, 2)
	for _, ot := range newSt.OverrideTasks {
		require.NotNil(t, ot.TDDEnabled)
		assert.False(t, *ot.TDDEnabled)
	}
}

func TestApplyTDDSelection_CustomOnlyMarksListedTasksTrue(t *testing.T) {
	st := state.CreateInitialState()
	tasks := []state.SpecTask{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B"},
		{ID: "task-3", Title: "C"},
	}
	newSt := applyTDDSelectionToOverrides(st, tasks, tddSelectionChoice{
		Mode:     "custom",
		TDDTasks: []string{"task-2"},
	})
	require.Len(t, newSt.OverrideTasks, 3)

	byID := make(map[string]*state.SpecTask, 3)
	for i := range newSt.OverrideTasks {
		byID[newSt.OverrideTasks[i].ID] = &newSt.OverrideTasks[i]
	}
	require.NotNil(t, byID["task-1"].TDDEnabled)
	require.NotNil(t, byID["task-2"].TDDEnabled)
	require.NotNil(t, byID["task-3"].TDDEnabled)
	assert.False(t, *byID["task-1"].TDDEnabled)
	assert.True(t, *byID["task-2"].TDDEnabled)
	assert.False(t, *byID["task-3"].TDDEnabled)
}

func TestApplyTDDSelection_AddsMissingTasksAndPreservesExisting(t *testing.T) {
	st := state.CreateInitialState()
	// Existing override row with Covers metadata should be preserved, but have
	// its TDDEnabled flag set by the selection.
	st.OverrideTasks = []state.SpecTask{
		{ID: "task-1", Title: "A", Covers: []string{"EC-1"}},
	}
	tasks := []state.SpecTask{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B"},
	}
	newSt := applyTDDSelectionToOverrides(st, tasks, tddSelectionChoice{Mode: "all"})
	require.Len(t, newSt.OverrideTasks, 2)
	assert.Equal(t, []string{"EC-1"}, newSt.OverrideTasks[0].Covers, "Covers preserved on existing row")
	require.NotNil(t, newSt.OverrideTasks[0].TDDEnabled)
	require.NotNil(t, newSt.OverrideTasks[1].TDDEnabled)
}

func TestReseedTDDCycleIfNeeded_StartsRedForTDDTask(t *testing.T) {
	truth := true
	st := state.CreateInitialState()
	st.OverrideTasks = []state.SpecTask{
		{ID: "task-1", Title: "A", TDDEnabled: &truth},
	}
	st.Execution.TDDCycle = "" // just reset by RecordTDDVerificationFull
	cfg := &state.NosManifest{Tdd: &state.Manifest{TddMode: true}}

	reseedTDDCycleIfNeeded(&st, cfg)
	assert.Equal(t, "red", st.Execution.TDDCycle)
}

func TestReseedTDDCycleIfNeeded_LeavesEmptyForNonTDDTask(t *testing.T) {
	falsity := false
	st := state.CreateInitialState()
	st.OverrideTasks = []state.SpecTask{
		{ID: "task-1", Title: "A", TDDEnabled: &falsity},
	}
	st.Execution.TDDCycle = ""
	cfg := &state.NosManifest{Tdd: &state.Manifest{TddMode: true}}

	reseedTDDCycleIfNeeded(&st, cfg)
	assert.Equal(t, "", st.Execution.TDDCycle, "non-TDD task must not trigger a RED seed")
}

func TestReseedTDDCycleIfNeeded_ClearsStaleCycleWhenTaskSwitchesToNonTDD(t *testing.T) {
	falsity := false
	st := state.CreateInitialState()
	st.OverrideTasks = []state.SpecTask{
		{ID: "task-1", Title: "A", TDDEnabled: &falsity},
	}
	// Leftover cycle from a previous TDD task — must be scrubbed.
	st.Execution.TDDCycle = "red"
	st.Execution.RefactorRounds = 2
	st.Execution.RefactorApplied = true
	cfg := &state.NosManifest{Tdd: &state.Manifest{TddMode: true}}

	reseedTDDCycleIfNeeded(&st, cfg)
	assert.Equal(t, "", st.Execution.TDDCycle)
	assert.Equal(t, 0, st.Execution.RefactorRounds)
	assert.False(t, st.Execution.RefactorApplied)
}
