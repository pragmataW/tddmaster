package cmd

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleDiscoveryAnswer_BatchFallback_IgnoresUnknownKeysAndRealignsCurrentQuestion(t *testing.T) {
	st := state.CreateInitialState()
	mode := state.DiscoveryModeFull
	premisesDone := true

	st.Phase = state.PhaseDiscovery
	st.Discovery.Mode = &mode
	st.Discovery.PremisesCompleted = &premisesDone

	answer := `{"status_quo":"Users manually compare exported CSV files before every release.","notes":"this key must be ignored"}` //nolint:lll

	newState, err := handleDiscoveryAnswer("", st, nil, answer, nil)
	require.NoError(t, err)
	require.Len(t, newState.Discovery.Answers, 1)
	assert.Equal(t, "status_quo", newState.Discovery.Answers[0].QuestionID)
	require.NotNil(t, newState.Discovery.BatchSubmitted)
	assert.True(t, *newState.Discovery.BatchSubmitted)
	assert.Equal(t, 1, newState.Discovery.CurrentQuestion)
	assert.Equal(t, state.PhaseDiscovery, newState.Phase)
}

func TestHandleDiscoveryAnswer_BatchFallback_CompletesDiscoveryWithCanonicalAnswers(t *testing.T) {
	st := state.CreateInitialState()
	mode := state.DiscoveryModeFull
	premisesDone := true
	specName := "discovery-contract"

	st.Phase = state.PhaseDiscovery
	st.Spec = &specName
	st.Discovery.Mode = &mode
	st.Discovery.PremisesCompleted = &premisesDone

	answer := `{
		"status_quo":"Users manually compare exported CSV files before every release.",
		"ambition":"The 1-star version stabilizes syncs; the 10-star version makes reconciliation disappear.",
		"reversibility":"This touches persisted state and needs a safe rollback plan if the migration goes badly.",
		"user_impact":"Existing users keep their current workflow while rollout happens behind a compatibility layer.",
		"verification":["Add regression coverage for the sync path.","Add an integration test that proves the new flow works end to end."],
		"scope_boundary":"This first slice must not add admin analytics, bulk editing, or a brand-new UI surface.",
		"edge_cases":"Protect against duplicate deliveries, timeout retries, and partial writes during sync.",
		"notes":"ignore this extra key"
	}`

	newState, err := handleDiscoveryAnswer("", st, nil, answer, nil)
	require.NoError(t, err)
	assert.Equal(t, state.PhaseDiscoveryRefinement, newState.Phase)
	assert.True(t, newState.Discovery.Completed)
	require.NotNil(t, newState.Discovery.BatchSubmitted)
	assert.True(t, *newState.Discovery.BatchSubmitted)
	assert.Len(t, newState.Discovery.Answers, 7)
	assert.Equal(t, 7, newState.Discovery.CurrentQuestion)
}
