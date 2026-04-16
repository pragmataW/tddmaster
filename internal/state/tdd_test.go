package state

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func boolPtr(v bool) *bool { return &v }

func tddManifest(mode bool) *NosManifest {
	return &NosManifest{Tdd: &Manifest{TddMode: mode}}
}

func TestCurrentTaskID_ReturnsFirstUncompleted(t *testing.T) {
	st := CreateInitialState()
	st.OverrideTasks = []SpecTask{
		{ID: "task-1", Title: "A", Completed: false},
		{ID: "task-2", Title: "B", Completed: false},
		{ID: "task-3", Title: "C", Completed: false},
	}
	st.Execution.CompletedTasks = []string{"task-1"}
	assert.Equal(t, "task-2", CurrentTaskID(st))
}

func TestCurrentTaskID_ReturnsEmptyWhenAllCompleted(t *testing.T) {
	st := CreateInitialState()
	st.OverrideTasks = []SpecTask{
		{ID: "task-1", Title: "A"},
		{ID: "task-2", Title: "B"},
	}
	st.Execution.CompletedTasks = []string{"task-1", "task-2"}
	assert.Equal(t, "", CurrentTaskID(st))
}

func TestCurrentTaskID_ReturnsEmptyWhenNoTasks(t *testing.T) {
	st := CreateInitialState()
	assert.Equal(t, "", CurrentTaskID(st))
}

func TestIsTaskTDDEnabled_TaskOverrideBeatsSpecLevel(t *testing.T) {
	st := CreateInitialState()
	st.OverrideTasks = []SpecTask{
		{ID: "task-1", TDDEnabled: boolPtr(true)},
		{ID: "task-2", TDDEnabled: boolPtr(false)},
	}
	// spec-level off, task-1 says true → true
	assert.True(t, IsTaskTDDEnabled(st, "task-1", tddManifest(false)))
	// spec-level on, task-2 says false → false
	assert.False(t, IsTaskTDDEnabled(st, "task-2", tddManifest(true)))
}

func TestIsTaskTDDEnabled_NilFallsBackToSpec(t *testing.T) {
	st := CreateInitialState()
	st.OverrideTasks = []SpecTask{
		{ID: "task-1", TDDEnabled: nil},
	}
	assert.True(t, IsTaskTDDEnabled(st, "task-1", tddManifest(true)))
	assert.False(t, IsTaskTDDEnabled(st, "task-1", tddManifest(false)))
}

func TestIsTaskTDDEnabled_UnknownTaskFallsBackToSpec(t *testing.T) {
	st := CreateInitialState()
	assert.True(t, IsTaskTDDEnabled(st, "task-99", tddManifest(true)))
	assert.False(t, IsTaskTDDEnabled(st, "task-99", tddManifest(false)))
}

func TestShouldRunTDDForCurrentTask_MixedOverrides(t *testing.T) {
	st := CreateInitialState()
	st.OverrideTasks = []SpecTask{
		{ID: "task-1", TDDEnabled: boolPtr(false)}, // non-TDD plumbing
		{ID: "task-2", TDDEnabled: boolPtr(true)},  // TDD behavior task
	}
	cfg := tddManifest(true)

	// No completion yet → current task is task-1 (non-TDD)
	assert.False(t, ShouldRunTDDForCurrentTask(st, cfg))

	// After task-1 completes → current task is task-2 (TDD)
	st.Execution.CompletedTasks = []string{"task-1"}
	assert.True(t, ShouldRunTDDForCurrentTask(st, cfg))
}

func TestShouldRunTDDForCurrentTask_NoOverrideTasksFallsBackToSpec(t *testing.T) {
	st := CreateInitialState()
	// No OverrideTasks → should fall back to spec-level flag so older state
	// files and bootstrap flows keep their pre-feature behavior.
	assert.True(t, ShouldRunTDDForCurrentTask(st, tddManifest(true)))
	assert.False(t, ShouldRunTDDForCurrentTask(st, tddManifest(false)))
}

func TestShouldRunTDDForCurrentTask_AllCompletedFallsBackToSpec(t *testing.T) {
	st := CreateInitialState()
	st.OverrideTasks = []SpecTask{{ID: "task-1", TDDEnabled: boolPtr(false)}}
	st.Execution.CompletedTasks = []string{"task-1"}
	// No current task → spec-level decides (avoids dragging the last task's
	// flag into post-completion compile output).
	assert.True(t, ShouldRunTDDForCurrentTask(st, tddManifest(true)))
}

func TestAnyTaskUsesTDD_MixedOverrides(t *testing.T) {
	st := CreateInitialState()
	st.OverrideTasks = []SpecTask{
		{ID: "task-1", TDDEnabled: boolPtr(false)},
		{ID: "task-2", TDDEnabled: boolPtr(true)},
	}
	assert.True(t, AnyTaskUsesTDD(st, tddManifest(false)))
}

func TestAnyTaskUsesTDD_AllDisabledReturnsFalse(t *testing.T) {
	st := CreateInitialState()
	st.OverrideTasks = []SpecTask{
		{ID: "task-1", TDDEnabled: boolPtr(false)},
		{ID: "task-2", TDDEnabled: boolPtr(false)},
	}
	assert.False(t, AnyTaskUsesTDD(st, tddManifest(true)))
}

func TestAnyTaskUsesTDD_NilFollowsSpecLevel(t *testing.T) {
	st := CreateInitialState()
	st.OverrideTasks = []SpecTask{
		{ID: "task-1", TDDEnabled: nil},
	}
	assert.True(t, AnyTaskUsesTDD(st, tddManifest(true)))
	assert.False(t, AnyTaskUsesTDD(st, tddManifest(false)))
}

func TestSpecTask_TDDEnabled_JSONRoundTrip(t *testing.T) {
	st := CreateInitialState()
	truth := true
	falsity := false
	st.OverrideTasks = []SpecTask{
		{ID: "task-1", Title: "yes", TDDEnabled: &truth},
		{ID: "task-2", Title: "no", TDDEnabled: &falsity},
		{ID: "task-3", Title: "default"},
	}
	data, err := json.Marshal(st)
	if err != nil {
		assert.Fail(t, "marshal failed", err.Error())
		return
	}
	var round StateFile
	assert.NoError(t, json.Unmarshal(data, &round))
	if assert.Len(t, round.OverrideTasks, 3) {
		assert.NotNil(t, round.OverrideTasks[0].TDDEnabled)
		assert.True(t, *round.OverrideTasks[0].TDDEnabled)
		assert.NotNil(t, round.OverrideTasks[1].TDDEnabled)
		assert.False(t, *round.OverrideTasks[1].TDDEnabled)
		assert.Nil(t, round.OverrideTasks[2].TDDEnabled)
	}
}
