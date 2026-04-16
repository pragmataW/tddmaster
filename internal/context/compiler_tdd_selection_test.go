package context_test

import (
	"testing"

	ctx "github.com/pragmataW/tddmaster/internal/context"
	specpkg "github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func specApprovedState(tasks ...state.SpecTask) state.StateFile {
	st := state.CreateInitialState()
	st.Phase = state.PhaseSpecApproved
	name := "demo"
	st.Spec = &name
	path := ".tddmaster/specs/demo/spec.md"
	st.SpecState.Path = &path
	st.SpecState.Status = "approved"
	if len(tasks) > 0 {
		st.OverrideTasks = tasks
	}
	return st
}

func tddManifest(mode bool) *state.NosManifest {
	return &state.NosManifest{Tdd: &state.Manifest{TddMode: mode}}
}

func parsedTasks(ids ...string) *specpkg.ParsedSpec {
	tasks := make([]specpkg.ParsedTask, 0, len(ids))
	for _, id := range ids {
		tasks = append(tasks, specpkg.ParsedTask{ID: id, Title: "Task " + id})
	}
	return &specpkg.ParsedSpec{Name: "demo", Tasks: tasks}
}

func TestCompile_SpecApproved_TDDSelection_PresentWhenGateNil(t *testing.T) {
	st := specApprovedState(state.SpecTask{ID: "task-1", Title: "Implement login"})
	out := ctx.Compile(st, nil, nil, tddManifest(true), parsedTasks("task-1"), nil, nil, nil, nil, 0)

	require.NotNil(t, out.SpecApprovedData)
	require.NotNil(t, out.SpecApprovedData.TaskTDDSelection)
	assert.True(t, out.SpecApprovedData.TaskTDDSelection.Required)
	assert.NotEmpty(t, out.SpecApprovedData.TaskTDDSelection.Tasks)
	assert.Equal(t, "tdd-all", out.SpecApprovedData.TaskTDDSelection.Answers.All)
	assert.Equal(t, "tdd-none", out.SpecApprovedData.TaskTDDSelection.Answers.None)
}

func TestCompile_SpecApproved_TDDSelection_HiddenWhenTDDDisabled(t *testing.T) {
	st := specApprovedState(state.SpecTask{ID: "task-1", Title: "Implement login"})
	out := ctx.Compile(st, nil, nil, tddManifest(false), parsedTasks("task-1"), nil, nil, nil, nil, 0)

	require.NotNil(t, out.SpecApprovedData)
	assert.Nil(t, out.SpecApprovedData.TaskTDDSelection, "no sub-step when spec-level TDD is off")
}

func TestCompile_SpecApproved_TDDSelection_HiddenAfterGateSet(t *testing.T) {
	st := specApprovedState(state.SpecTask{ID: "task-1", Title: "Implement login"})
	tr := true
	st.TaskTDDSelected = &tr
	out := ctx.Compile(st, nil, nil, tddManifest(true), parsedTasks("task-1"), nil, nil, nil, nil, 0)

	require.NotNil(t, out.SpecApprovedData)
	assert.Nil(t, out.SpecApprovedData.TaskTDDSelection, "no sub-step once selection has been recorded")
}

func TestCompile_SpecApproved_TDDSelection_ShowsThreeInteractiveOptions(t *testing.T) {
	st := specApprovedState(state.SpecTask{ID: "task-1", Title: "Implement login"})
	out := ctx.Compile(st, nil, nil, tddManifest(true), parsedTasks("task-1"), nil, nil, nil, nil, 0)

	require.NotEmpty(t, out.InteractiveOptions)
	labels := make([]string, 0, len(out.InteractiveOptions))
	for _, opt := range out.InteractiveOptions {
		labels = append(labels, opt.Label)
	}
	assert.Contains(t, labels, "TDD for all tasks")
	assert.Contains(t, labels, "No TDD")
	assert.Contains(t, labels, "Pick per task")
}

func TestCompile_Executing_SuppressesTDDPhaseForNonTDDTask(t *testing.T) {
	st := state.CreateInitialState()
	st.Phase = state.PhaseExecuting
	st.Execution.TDDCycle = "red" // stale/leftover — must be ignored for non-TDD task
	name := "demo"
	st.Spec = &name
	falsity := false
	st.OverrideTasks = []state.SpecTask{
		{ID: "task-1", Title: "Download modules", TDDEnabled: &falsity},
	}

	out := ctx.Compile(st, nil, nil, tddManifest(true), nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.ExecutionData)
	assert.Nil(t, out.ExecutionData.TDDPhase, "TDDPhase must be nil for non-TDD task even if cycle is set")
	assert.Nil(t, out.ExecutionData.TDDVerificationContext)
	assert.Nil(t, out.ExecutionData.RefactorInstructions)
}

func TestCompile_Executing_EnablesTDDPhaseForTDDTask(t *testing.T) {
	st := state.CreateInitialState()
	st.Phase = state.PhaseExecuting
	st.Execution.TDDCycle = "red"
	name := "demo"
	st.Spec = &name
	truth := true
	st.OverrideTasks = []state.SpecTask{
		{ID: "task-1", Title: "Implement login", TDDEnabled: &truth},
	}

	out := ctx.Compile(st, nil, nil, tddManifest(true), nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.ExecutionData)
	require.NotNil(t, out.ExecutionData.TDDPhase)
	assert.Equal(t, "red", *out.ExecutionData.TDDPhase)
}
