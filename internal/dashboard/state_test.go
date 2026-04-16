
package dashboard

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pragmataW/tddmaster/internal/state"
)

func TestBuildRoadmap(t *testing.T) {
	tests := []struct {
		phase    state.Phase
		contains string
	}{
		{state.PhaseIdle, "[ IDLE ]"},
		{state.PhaseDiscovery, "[ DISCOVERY ]"},
		{state.PhaseDiscoveryRefinement, "[ REVIEW ]"},
		{state.PhaseSpecProposal, "[ DRAFT ]"},
		{state.PhaseSpecApproved, "[ APPROVED ]"},
		{state.PhaseExecuting, "[ EXECUTING ]"},
		{state.PhaseCompleted, "[ DONE ]"},
		{state.PhaseBlocked, "[ EXECUTING ]"},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			roadmap := buildRoadmap(tt.phase)
			assert.Contains(t, roadmap, tt.contains)
			assert.Contains(t, roadmap, "→")
		})
	}
}

func TestGetSpecSummary_NoSpec(t *testing.T) {
	dir := t.TempDir()
	// Create spec directory so ResolveState finds it
	specDir := filepath.Join(dir, state.TddmasterDir, "specs", "my-spec")
	require.NoError(t, os.MkdirAll(specDir, 0o755))

	summary, err := GetSpecSummary(dir, "my-spec")
	require.NoError(t, err)

	assert.Equal(t, "my-spec", summary.Name)
	assert.Equal(t, "my-spec", summary.Slug)
	assert.NotEmpty(t, summary.Roadmap)
	assert.Empty(t, summary.Tasks)
	assert.Empty(t, summary.Contributors)
	assert.Empty(t, summary.PendingQuestions)
	assert.Empty(t, summary.PendingSignoffs)
}

func TestGetState_Empty(t *testing.T) {
	dir := t.TempDir()

	ds, err := GetState(dir)
	require.NoError(t, err)

	assert.Empty(t, ds.Specs)
	assert.Nil(t, ds.ActiveSpec)
	assert.Empty(t, ds.PendingMentions)
	assert.Empty(t, ds.PendingSignoffs)
	assert.Empty(t, ds.RecentEvents)
}

func TestGetState_WithSpec(t *testing.T) {
	dir := t.TempDir()

	// Create spec state
	specName := "test-spec"
	specDir := filepath.Join(dir, state.TddmasterDir, "specs", specName)
	require.NoError(t, os.MkdirAll(specDir, 0o755))

	s := state.CreateInitialState()
	s.Phase = state.PhaseExecuting
	desc := "A test spec"
	s.SpecDescription = &desc
	require.NoError(t, state.WriteSpecState(dir, specName, s))

	ds, err := GetState(dir)
	require.NoError(t, err)

	assert.Len(t, ds.Specs, 1)
	assert.NotNil(t, ds.ActiveSpec)
	assert.Equal(t, specName, ds.ActiveSpec.Name)
	assert.Equal(t, state.PhaseExecuting, ds.ActiveSpec.Phase)
}

func TestGetState_PendingMentions(t *testing.T) {
	dir := t.TempDir()

	// Append mention event
	mentionEv := DashboardEvent{
		Ts:   "2024-01-01T10:00:00Z",
		Type: EventTypeMention,
		Spec: "spec",
		User: "alice",
		Extra: map[string]interface{}{
			"id":       "mention-1",
			"from":     "alice",
			"to":       "bob",
			"question": "What should we do?",
		},
	}
	require.NoError(t, AppendEvent(dir, mentionEv))

	ds, err := GetState(dir)
	require.NoError(t, err)

	assert.Len(t, ds.PendingMentions, 1)
	assert.Equal(t, "mention-1", ds.PendingMentions[0].ID)
	assert.Equal(t, "What should we do?", ds.PendingMentions[0].Question)
}

func TestGetState_RepliedMentionsFiltered(t *testing.T) {
	dir := t.TempDir()

	// Append mention + reply
	mentionEv := DashboardEvent{
		Ts:   "2024-01-01T10:00:00Z",
		Type: EventTypeMention,
		Spec: "spec",
		User: "alice",
		Extra: map[string]interface{}{
			"id":       "mention-1",
			"from":     "alice",
			"to":       "bob",
			"question": "What?",
		},
	}
	replyEv := DashboardEvent{
		Ts:   "2024-01-01T11:00:00Z",
		Type: EventTypeMentionReply,
		Spec: "spec",
		User: "bob",
		Extra: map[string]interface{}{
			"mentionId": "mention-1",
			"text":      "This!",
		},
	}
	require.NoError(t, AppendEvent(dir, mentionEv))
	require.NoError(t, AppendEvent(dir, replyEv))

	ds, err := GetState(dir)
	require.NoError(t, err)

	assert.Empty(t, ds.PendingMentions)
}
