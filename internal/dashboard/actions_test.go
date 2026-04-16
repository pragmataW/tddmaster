
package dashboard

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pragmataW/tddmaster/internal/state"
)

func setupSpec(t *testing.T, dir, specName string, phase state.Phase) {
	t.Helper()
	specDir := filepath.Join(dir, state.TddmasterDir, "specs", specName)
	require.NoError(t, os.MkdirAll(specDir, 0o755))
	s := state.CreateInitialState()
	s.Phase = phase
	require.NoError(t, state.WriteSpecState(dir, specName, s))
}

func TestApprove_Success(t *testing.T) {
	dir := t.TempDir()
	setupSpec(t, dir, "my-spec", state.PhaseSpecProposal)

	user := &User{Name: "alice", Email: "alice@example.com"}
	result := Approve(dir, "my-spec", user)
	assert.True(t, result.OK)
	assert.Empty(t, result.Error)

	// Verify state was written
	s, err := state.ReadSpecState(dir, "my-spec")
	require.NoError(t, err)
	assert.Equal(t, state.PhaseSpecApproved, s.Phase)

	// Verify event was appended
	events, err := ReadEvents(dir, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, events)
	assert.Equal(t, EventTypePhaseChange, events[0].Type)
}

func TestApprove_WrongPhase(t *testing.T) {
	dir := t.TempDir()
	setupSpec(t, dir, "my-spec", state.PhaseDiscovery)

	result := Approve(dir, "my-spec", &User{Name: "alice", Email: ""})
	assert.False(t, result.OK)
	assert.Contains(t, result.Error, "Cannot approve in phase")
}

func TestAddNote_Success(t *testing.T) {
	dir := t.TempDir()
	setupSpec(t, dir, "my-spec", state.PhaseExecuting)

	user := &User{Name: "bob", Email: "bob@example.com"}
	result := AddNote(dir, "my-spec", "this is a note", user)
	assert.True(t, result.OK)

	s, err := state.ReadSpecState(dir, "my-spec")
	require.NoError(t, err)
	assert.Len(t, s.SpecNotes, 1)
	assert.Equal(t, "this is a note", s.SpecNotes[0].Text)

	events, err := ReadEvents(dir, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, events)
	assert.Equal(t, EventTypeNote, events[0].Type)
}

func TestAddQuestion_Success(t *testing.T) {
	dir := t.TempDir()
	setupSpec(t, dir, "my-spec", state.PhaseExecuting)

	user := &User{Name: "carol", Email: "carol@example.com"}
	result := AddQuestion(dir, "my-spec", "what is the plan?", user)
	assert.True(t, result.OK)

	s, err := state.ReadSpecState(dir, "my-spec")
	require.NoError(t, err)
	require.Len(t, s.SpecNotes, 1)
	assert.Equal(t, "[QUESTION] what is the plan?", s.SpecNotes[0].Text)

	events, err := ReadEvents(dir, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, events)
	assert.Equal(t, EventTypeMention, events[0].Type)
}

func TestSignoff_Success(t *testing.T) {
	dir := t.TempDir()

	user := &User{Name: "dave", Email: "dave@example.com"}
	result := Signoff(dir, "my-spec", user)
	assert.True(t, result.OK)

	events, err := ReadEvents(dir, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, events)
	assert.Equal(t, EventTypeSignoff, events[0].Type)
	assert.Equal(t, "signed", events[0].GetExtra("status"))
}

func TestReplyMention_Success(t *testing.T) {
	dir := t.TempDir()
	setupSpec(t, dir, "my-spec", state.PhaseExecuting)

	user := &User{Name: "eve", Email: "eve@example.com"}
	result := ReplyMention(dir, "my-spec", "mention-42", "here is my reply", user)
	assert.True(t, result.OK)

	s, err := state.ReadSpecState(dir, "my-spec")
	require.NoError(t, err)
	require.Len(t, s.SpecNotes, 1)
	assert.Equal(t, "[REPLY:mention-42] here is my reply", s.SpecNotes[0].Text)

	events, err := ReadEvents(dir, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, events)
	assert.Equal(t, EventTypeMentionReply, events[0].Type)
	assert.Equal(t, "mention-42", events[0].GetExtra("mentionId"))
}

func TestComplete_Success(t *testing.T) {
	dir := t.TempDir()
	setupSpec(t, dir, "my-spec", state.PhaseExecuting)

	user := &User{Name: "frank", Email: "frank@example.com"}
	result := Complete(dir, "my-spec", user)
	assert.True(t, result.OK)

	s, err := state.ReadSpecState(dir, "my-spec")
	require.NoError(t, err)
	assert.Equal(t, state.PhaseCompleted, s.Phase)

	events, err := ReadEvents(dir, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, events)
	assert.Equal(t, EventTypePhaseChange, events[0].Type)
	assert.Equal(t, "COMPLETED", events[0].GetExtra("to"))
}

func TestComplete_WrongPhase(t *testing.T) {
	dir := t.TempDir()
	setupSpec(t, dir, "my-spec", state.PhaseSpecApproved)

	result := Complete(dir, "my-spec", &User{Name: "grace", Email: ""})
	assert.False(t, result.OK)
	assert.Contains(t, result.Error, "Cannot complete in phase")
}
