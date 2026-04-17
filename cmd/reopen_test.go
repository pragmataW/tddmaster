package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pragmataW/tddmaster/internal/state"
)

func TestReopenCmd_ResumeExecution_PreservesProgressAndUpdatesArtifacts(t *testing.T) {
	tasks := []state.SpecTask{
		{ID: "task-1", Title: "First", Completed: true},
		{ID: "task-2", Title: "Second"},
	}
	root, specName := seedUndoState(t, tasks, []string{"task-1"})
	setProjectRoot(t, root)

	progress := "executor status"
	completionReason := state.CompletionReasonDone
	completedAt := "2026-04-17T12:00:00Z"

	st, err := state.ResolveState(root, &specName)
	require.NoError(t, err)
	st.Phase = state.PhaseCompleted
	st.CompletionReason = &completionReason
	st.CompletedAt = &completedAt
	st.Execution.Iteration = 3
	st.Execution.LastProgress = &progress
	st.TransitionHistory = []state.PhaseTransition{{From: state.PhaseExecuting, To: state.PhaseCompleted}}

	require.NoError(t, state.WriteState(root, state.CreateInitialState()))
	require.NoError(t, state.WriteSpecState(root, specName, st))

	writeCmdSpecMd(t, root, specName, "# Spec: my-spec\n\n## Status: completed\n")
	writeCmdProgressJSON(t, root, specName, map[string]interface{}{
		"spec":   specName,
		"status": "completed",
		"tasks": []interface{}{
			map[string]interface{}{"id": "task-1", "title": "First", "status": "done"},
			map[string]interface{}{"id": "task-2", "title": "Second", "status": "pending"},
		},
	})

	err = runReopenWithArgs([]string{"--spec=" + specName, "--resume-execution"})
	require.NoError(t, err)

	activeState, err := state.ReadState(root)
	require.NoError(t, err)
	assert.Equal(t, state.PhaseExecuting, activeState.Phase)
	assert.Equal(t, []string{"task-1"}, activeState.Execution.CompletedTasks)
	require.NotNil(t, activeState.Execution.LastProgress)
	assert.Equal(t, progress, *activeState.Execution.LastProgress)
	assert.Nil(t, activeState.CompletionReason)

	specState, err := state.ResolveState(root, &specName)
	require.NoError(t, err)
	assert.Equal(t, state.PhaseExecuting, specState.Phase)
	assert.Equal(t, 3, specState.Execution.Iteration)

	specMd := readCmdSpecMd(t, root, specName)
	assert.Contains(t, specMd, "## Status: executing")

	progressJSON := readCmdProgressJSON(t, root, specName)
	assert.Equal(t, "executing", progressJSON["status"])
}

func TestReopenCmd_RejectsUnexpectedPositionalArg(t *testing.T) {
	root := t.TempDir()
	specName := "done-spec"

	require.NoError(t, state.ScaffoldDir(root))
	require.NoError(t, state.WriteManifest(root, state.CreateInitialManifest(nil, nil, state.ProjectTraits{})))
	setProjectRoot(t, root)

	sp := specName
	st := state.CreateInitialState()
	st.Phase = state.PhaseCompleted
	st.Spec = &sp
	require.NoError(t, os.MkdirAll(filepath.Join(root, state.TddmasterDir, "specs", specName), 0o755))
	require.NoError(t, state.WriteState(root, st))
	require.NoError(t, state.WriteSpecState(root, specName, st))

	err := runReopenWithArgs([]string{"--spec=" + specName, "task-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected positional arguments")
}

func writeCmdSpecMd(t *testing.T, root, specName, content string) {
	t.Helper()
	specDir := filepath.Join(root, ".tddmaster", "specs", specName)
	require.NoError(t, os.MkdirAll(specDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "spec.md"), []byte(content), 0o644))
}

func readCmdSpecMd(t *testing.T, root, specName string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".tddmaster", "specs", specName, "spec.md"))
	require.NoError(t, err)
	return string(data)
}

func writeCmdProgressJSON(t *testing.T, root, specName string, data map[string]interface{}) {
	t.Helper()
	specDir := filepath.Join(root, ".tddmaster", "specs", specName)
	require.NoError(t, os.MkdirAll(specDir, 0o755))
	b, err := json.MarshalIndent(data, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(specDir, "progress.json"), append(b, '\n'), 0o644))
}

func readCmdProgressJSON(t *testing.T, root, specName string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".tddmaster", "specs", specName, "progress.json"))
	require.NoError(t, err)
	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &out))
	return out
}
