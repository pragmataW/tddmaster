package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pragmataW/tddmaster/internal/state"
)

// seedUndoState writes a minimal .tddmaster scaffold, then creates a spec dir
// and populates state.json with an EXECUTING spec that has overrideTasks and
// completedTasks. Returns the temp root and spec name.
func seedUndoState(t *testing.T, tasks []state.SpecTask, completedIDs []string) (root string, specName string) {
	t.Helper()

	root = t.TempDir()
	specName = "my-spec"

	require.NoError(t, state.ScaffoldDir(root))

	// Create spec directory so ResolveState can find it.
	specDir := filepath.Join(root, state.TddmasterDir, "specs", specName)
	require.NoError(t, os.MkdirAll(specDir, 0o755))

	st := state.CreateInitialState()
	st.Phase = state.PhaseExecuting
	spec := specName
	st.Spec = &spec
	st.OverrideTasks = tasks
	st.Execution.CompletedTasks = completedIDs

	require.NoError(t, state.WriteState(root, st))
	require.NoError(t, state.WriteSpecState(root, specName, st))

	manifest := state.CreateInitialManifest([]string{}, []state.CodingToolId{}, state.ProjectTraits{})
	require.NoError(t, state.WriteManifest(root, manifest))

	return root, specName
}

// TestUndoCmd_UndoLast removes the most recently completed task when no ID given.
func TestUndoCmd_UndoLast(t *testing.T) {
	tasks := []state.SpecTask{
		{ID: "task-1", Title: "First", Completed: true},
		{ID: "task-2", Title: "Second", Completed: true},
	}
	root, specName := seedUndoState(t, tasks, []string{"task-1", "task-2"})
	setProjectRoot(t, root)

	err := runUndoWithArgs([]string{"--spec=" + specName})
	require.NoError(t, err)

	st, err := state.ResolveState(root, &specName)
	require.NoError(t, err)

	// task-2 was last completed; it must now be removed from CompletedTasks.
	assert.NotContains(t, st.Execution.CompletedTasks, "task-2")
	assert.Contains(t, st.Execution.CompletedTasks, "task-1")
}

// TestUndoCmd_UndoSpecificID undoes a specific task by ID.
func TestUndoCmd_UndoSpecificID(t *testing.T) {
	tasks := []state.SpecTask{
		{ID: "task-1", Title: "First", Completed: true},
		{ID: "task-2", Title: "Second", Completed: true},
		{ID: "task-3", Title: "Third", Completed: true},
	}
	root, specName := seedUndoState(t, tasks, []string{"task-1", "task-2", "task-3"})
	setProjectRoot(t, root)

	err := runUndoWithArgs([]string{"--spec=" + specName, "task-2"})
	require.NoError(t, err)

	st, err := state.ResolveState(root, &specName)
	require.NoError(t, err)

	assert.NotContains(t, st.Execution.CompletedTasks, "task-2")
	assert.Contains(t, st.Execution.CompletedTasks, "task-1")
	assert.Contains(t, st.Execution.CompletedTasks, "task-3")
}

// TestUndoCmd_UndoUnknownID returns an error when the task ID is not completed.
func TestUndoCmd_UndoUnknownID(t *testing.T) {
	root, specName := seedUndoState(t, nil, []string{"task-1"})
	setProjectRoot(t, root)

	err := runUndoWithArgs([]string{"--spec=" + specName, "task-99"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task-99")
}

// TestUndoCmd_UndoOnCompletedSpec returns an error pointing the user at reopen.
func TestUndoCmd_UndoOnCompletedSpec(t *testing.T) {
	root := t.TempDir()
	specName := "done-spec"

	require.NoError(t, state.ScaffoldDir(root))
	specDir := filepath.Join(root, state.TddmasterDir, "specs", specName)
	require.NoError(t, os.MkdirAll(specDir, 0o755))

	st := state.CreateInitialState()
	st.Phase = state.PhaseCompleted
	sp := specName
	st.Spec = &sp
	st.Execution.CompletedTasks = []string{"task-1"}

	require.NoError(t, state.WriteState(root, st))
	require.NoError(t, state.WriteSpecState(root, specName, st))

	manifest := state.CreateInitialManifest([]string{}, []state.CodingToolId{}, state.ProjectTraits{})
	require.NoError(t, state.WriteManifest(root, manifest))

	setProjectRoot(t, root)

	err := runUndoWithArgs([]string{"--spec=" + specName})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reopen")
	assert.Contains(t, err.Error(), "--resume-execution")
}
