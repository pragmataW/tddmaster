
// Package compat contains compatibility tests for state and manifest round-trips.
package compat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pragmataW/tddmaster/internal/state"
)

// TestStateRoundTrip verifies that a StateFile can be written to JSON and read back
// with all fields preserved.
func TestStateRoundTrip(t *testing.T) {
	dir := t.TempDir()

	specName := "my-round-trip-spec"
	desc := "Test spec description"
	branch := "feature/test"
	progress := "In progress"

	original := state.CreateInitialState()
	original.Phase = state.PhaseExecuting
	original.Spec = &specName
	original.SpecDescription = &desc
	original.Branch = &branch
	original.Discovery.Completed = true
	original.Discovery.Approved = true
	original.Discovery.Answers = []state.DiscoveryAnswer{
		{QuestionID: "q1", Answer: "answer1"},
		{QuestionID: "q2", Answer: "answer2"},
	}
	original.Execution.Iteration = 3
	original.Execution.LastProgress = &progress
	original.Execution.ModifiedFiles = []string{"file1.go", "file2.go"}
	original.Execution.CompletedTasks = []string{"task-1", "task-2"}
	original.Decisions = []state.Decision{
		{ID: "d1", Question: "Q?", Choice: "A", Promoted: false, Timestamp: "2024-01-01T00:00:00Z"},
	}

	// Write
	err := state.WriteState(dir, original)
	require.NoError(t, err)

	// Verify file exists
	stateFilePath := filepath.Join(dir, ".tddmaster", ".state", "state.json")
	_, err = os.Stat(stateFilePath)
	require.NoError(t, err, "state.json should exist")

	// Read back
	loaded, err := state.ReadState(dir)
	require.NoError(t, err)

	// Compare fields
	assert.Equal(t, original.Phase, loaded.Phase)
	assert.Equal(t, original.Version, loaded.Version)
	require.NotNil(t, loaded.Spec)
	assert.Equal(t, specName, *loaded.Spec)
	require.NotNil(t, loaded.SpecDescription)
	assert.Equal(t, desc, *loaded.SpecDescription)
	require.NotNil(t, loaded.Branch)
	assert.Equal(t, branch, *loaded.Branch)
	assert.Equal(t, original.Discovery.Completed, loaded.Discovery.Completed)
	assert.Equal(t, original.Discovery.Approved, loaded.Discovery.Approved)
	assert.Equal(t, len(original.Discovery.Answers), len(loaded.Discovery.Answers))
	assert.Equal(t, "q1", loaded.Discovery.Answers[0].QuestionID)
	assert.Equal(t, "answer1", loaded.Discovery.Answers[0].Answer)
	assert.Equal(t, original.Execution.Iteration, loaded.Execution.Iteration)
	require.NotNil(t, loaded.Execution.LastProgress)
	assert.Equal(t, progress, *loaded.Execution.LastProgress)
	assert.Equal(t, original.Execution.ModifiedFiles, loaded.Execution.ModifiedFiles)
	assert.Equal(t, original.Execution.CompletedTasks, loaded.Execution.CompletedTasks)
	require.Len(t, loaded.Decisions, 1)
	assert.Equal(t, "d1", loaded.Decisions[0].ID)
}

// TestStateJSONFormat verifies the JSON output format matches expected field names.
func TestStateJSONFormat(t *testing.T) {
	original := state.CreateInitialState()
	specName := "format-test"
	original.Spec = &specName
	original.Phase = state.PhaseDiscovery

	data, err := json.MarshalIndent(original, "", "  ")
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))

	// Top-level keys must match TS expectations
	assert.Contains(t, raw, "version")
	assert.Contains(t, raw, "phase")
	assert.Contains(t, raw, "spec")
	assert.Contains(t, raw, "discovery")
	assert.Contains(t, raw, "specState")
	assert.Contains(t, raw, "execution")
	assert.Contains(t, raw, "decisions")

	// phase value
	assert.Equal(t, "DISCOVERY", raw["phase"])
	assert.Equal(t, "0.1.0", raw["version"])
	assert.Equal(t, "format-test", raw["spec"])

	// discovery sub-fields
	disc, ok := raw["discovery"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, disc, "answers")
	assert.Contains(t, disc, "completed")
	assert.Contains(t, disc, "currentQuestion")
	assert.Contains(t, disc, "audience")
	assert.Contains(t, disc, "approved")

	// execution sub-fields
	exec, ok := raw["execution"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, exec, "iteration")
	assert.Contains(t, exec, "modifiedFiles")
	assert.Contains(t, exec, "completedTasks")
	assert.Contains(t, exec, "awaitingStatusReport")
}

// TestSpecStateRoundTrip verifies per-spec state write/read round-trip.
func TestSpecStateRoundTrip(t *testing.T) {
	dir := t.TempDir()

	specName := "compat-spec"
	original := state.CreateInitialState()
	original.Phase = state.PhaseSpecApproved
	original.Spec = &specName

	err := state.WriteSpecState(dir, specName, original)
	require.NoError(t, err)

	// Verify file path
	stateFile := filepath.Join(dir, ".tddmaster", ".state", "specs", specName+".json")
	_, err = os.Stat(stateFile)
	require.NoError(t, err, "per-spec state file should exist")

	loaded, err := state.ReadSpecState(dir, specName)
	require.NoError(t, err)

	assert.Equal(t, state.PhaseSpecApproved, loaded.Phase)
	require.NotNil(t, loaded.Spec)
	assert.Equal(t, specName, *loaded.Spec)
}

// TestStateNullFieldsHandling verifies that nil/null fields are handled gracefully.
func TestStateNullFieldsHandling(t *testing.T) {
	dir := t.TempDir()

	original := state.CreateInitialState()
	// Ensure nullable fields remain nil
	assert.Nil(t, original.Spec)
	assert.Nil(t, original.SpecDescription)
	assert.Nil(t, original.Branch)
	assert.Nil(t, original.CompletionReason)
	assert.Nil(t, original.CompletedAt)
	assert.Nil(t, original.ReopenedFrom)

	err := state.WriteState(dir, original)
	require.NoError(t, err)

	loaded, err := state.ReadState(dir)
	require.NoError(t, err)

	assert.Nil(t, loaded.Spec)
	assert.Nil(t, loaded.SpecDescription)
	assert.Nil(t, loaded.Branch)
	assert.Nil(t, loaded.CompletionReason)
	assert.Nil(t, loaded.CompletedAt)
	assert.Nil(t, loaded.ReopenedFrom)
}

// TestStateCompletionRoundTrip verifies completion-related fields are preserved.
func TestStateCompletionRoundTrip(t *testing.T) {
	dir := t.TempDir()

	specName := "completed-spec"
	completedAt := "2024-06-01T12:00:00Z"
	completionNote := "All AC passed"
	reason := state.CompletionReasonDone

	original := state.CreateInitialState()
	original.Phase = state.PhaseCompleted
	original.Spec = &specName
	original.CompletedAt = &completedAt
	original.CompletionNote = &completionNote
	original.CompletionReason = &reason

	err := state.WriteState(dir, original)
	require.NoError(t, err)

	loaded, err := state.ReadState(dir)
	require.NoError(t, err)

	assert.Equal(t, state.PhaseCompleted, loaded.Phase)
	require.NotNil(t, loaded.CompletedAt)
	assert.Equal(t, completedAt, *loaded.CompletedAt)
	require.NotNil(t, loaded.CompletionNote)
	assert.Equal(t, completionNote, *loaded.CompletionNote)
	require.NotNil(t, loaded.CompletionReason)
	assert.Equal(t, reason, *loaded.CompletionReason)
}

// TestListSpecStatesCompat verifies listing multiple spec states.
func TestListSpecStatesCompat(t *testing.T) {
	dir := t.TempDir()

	// Write three spec states
	for i, name := range []string{"spec-alpha", "spec-beta", "spec-gamma"} {
		s := state.CreateInitialState()
		s.Spec = &name
		phases := []state.Phase{state.PhaseDiscovery, state.PhaseExecuting, state.PhaseCompleted}
		s.Phase = phases[i]
		require.NoError(t, state.WriteSpecState(dir, name, s))
	}

	entries, err := state.ListSpecStates(dir)
	require.NoError(t, err)
	require.Len(t, entries, 3)

	phaseMap := make(map[string]state.Phase)
	for _, e := range entries {
		phaseMap[e.Name] = e.State.Phase
	}
	assert.Equal(t, state.PhaseDiscovery, phaseMap["spec-alpha"])
	assert.Equal(t, state.PhaseExecuting, phaseMap["spec-beta"])
	assert.Equal(t, state.PhaseCompleted, phaseMap["spec-gamma"])
}
