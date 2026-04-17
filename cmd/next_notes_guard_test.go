package cmd

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// When an agent sneaks a pipe-separated task blob into the `notes` field
// of a SPEC_PROPOSAL refinement, the guard must recover it by routing to
// the tasks override instead of persisting the blob to SpecNotes.
func TestHandleSpecProposalAnswer_NotesWithTaskList_RoutesToTasksOverride(t *testing.T) {
	st := state.CreateInitialState()
	st.Phase = state.PhaseSpecProposal
	st.Classification = &state.SpecClassification{}

	answer := `{"notes":"task-1: First thing | task-2: Second thing | task-3: Third thing"}`

	newState, err := handleSpecProposalAnswer("/tmp", st, nil, nil, answer)

	require.NoError(t, err)
	assert.Empty(t, newState.SpecNotes, "task blob must not be persisted as a spec note")
	require.Len(t, newState.OverrideTasks, 3)
	assert.Equal(t, "First thing", newState.OverrideTasks[0].Title)
	assert.Equal(t, "Second thing", newState.OverrideTasks[1].Title)
	assert.Equal(t, "Third thing", newState.OverrideTasks[2].Title)
}

// When notes contain task-N markers mixed with unparseable prose, the
// guard must reject the payload so the agent is forced to retry with
// proper verbs (add/update/tasks).
func TestHandleSpecProposalAnswer_NotesWithTaskMarkerButUnparseable_ReturnsError(t *testing.T) {
	st := state.CreateInitialState()
	st.Phase = state.PhaseSpecProposal
	st.Classification = &state.SpecClassification{}

	// Mixed content: first chunk parses, second chunk is free-form prose
	// so parseTextualTaskList rejects the whole thing.
	answer := `{"notes":"task-1: A proper task | and some free-form reflection that is not a task"}`

	newState, err := handleSpecProposalAnswer("/tmp", st, nil, nil, answer)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "task markers")
	assert.Empty(t, newState.SpecNotes, "state must not be mutated on error")
}

// Plain free-form notes without any task-N markers must still be allowed
// to flow through to SpecNotes (the guard must not break the happy path).
func TestHandleSpecProposalAnswer_FreeFormNotes_PersistsAsSpecNote(t *testing.T) {
	st := state.CreateInitialState()
	st.Phase = state.PhaseSpecProposal
	st.Classification = &state.SpecClassification{}

	answer := `{"notes":"Remember to double-check the docs folder before shipping."}`

	newState, err := handleSpecProposalAnswer("/tmp", st, nil, nil, answer)

	require.NoError(t, err)
	require.Len(t, newState.SpecNotes, 1)
	assert.Equal(t, "Remember to double-check the docs folder before shipping.", newState.SpecNotes[0].Text)
	assert.Empty(t, newState.OverrideTasks)
}
