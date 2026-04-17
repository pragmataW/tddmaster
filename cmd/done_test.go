package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pragmataW/tddmaster/internal/state"
)

func TestDoneCmd_RejectsUnexpectedPositionalArg(t *testing.T) {
	tasks := []state.SpecTask{{ID: "task-1", Title: "First", Completed: true}}
	root, specName := seedUndoState(t, tasks, []string{"task-1"})
	setProjectRoot(t, root)

	err := runDoneWithArgs([]string{"--spec=" + specName, "task-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected positional arguments")
	assert.Contains(t, err.Error(), "undo")

	st, readErr := state.ResolveState(root, &specName)
	require.NoError(t, readErr)
	assert.Equal(t, state.PhaseExecuting, st.Phase)
}

func TestDoneCmd_RejectsPendingTasks(t *testing.T) {
	tasks := []state.SpecTask{
		{ID: "task-1", Title: "First", Completed: true},
		{ID: "task-2", Title: "Second"},
	}
	root, specName := seedUndoState(t, tasks, []string{"task-1"})
	setProjectRoot(t, root)

	err := runDoneWithArgs([]string{"--spec=" + specName})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pending tasks remain")
	assert.Contains(t, err.Error(), "task-2")
	assert.Contains(t, err.Error(), "next --answer")

	st, readErr := state.ResolveState(root, &specName)
	require.NoError(t, readErr)
	assert.Equal(t, state.PhaseExecuting, st.Phase)
}

func TestDoneCmd_AllTasksCompleted_CompletesSpec(t *testing.T) {
	tasks := []state.SpecTask{
		{ID: "task-1", Title: "First", Completed: true},
		{ID: "task-2", Title: "Second", Completed: true},
	}
	root, specName := seedUndoState(t, tasks, []string{"task-1", "task-2"})
	setProjectRoot(t, root)

	err := runDoneWithArgs([]string{"--spec=" + specName})
	require.NoError(t, err)

	activeState, readErr := state.ReadState(root)
	require.NoError(t, readErr)
	assert.Equal(t, state.PhaseIdle, activeState.Phase)

	specState, readErr := state.ResolveState(root, &specName)
	require.NoError(t, readErr)
	assert.Equal(t, state.PhaseCompleted, specState.Phase)
	require.NotNil(t, specState.CompletionReason)
	assert.Equal(t, state.CompletionReasonDone, *specState.CompletionReason)
}
