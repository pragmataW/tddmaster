package cmd

import (
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleDiscoveryAnswer_FirstListenResponse_PersistsPrefills(t *testing.T) {
	st := state.CreateInitialState()
	st.Phase = state.PhaseDiscovery
	desc := "Add a safer sync workflow"
	st.SpecDescription = &desc

	answer := strings.Join([]string{
		"Users currently export CSVs and compare the data by hand before every release.",
		"Existing users rely on the current sync endpoint, so this needs to remain backward compatible.",
		"This first version is MVP only; admin dashboards are out of scope for v1.",
		"We will verify it with regression tests, an integration test, and updated docs.",
		"We also need timeout and duplicate-delivery coverage.",
	}, " ")

	newState, err := handleDiscoveryAnswer("", st, nil, answer, nil)
	require.NoError(t, err)
	require.NotNil(t, newState.Discovery.UserContext)
	assert.Equal(t, answer, *newState.Discovery.UserContext)
	require.NotNil(t, newState.Discovery.UserContextProcessed)
	assert.True(t, *newState.Discovery.UserContextProcessed)
	assert.Empty(t, newState.Discovery.Answers, "prefills must not become confirmed discovery answers")
	assert.NotEmpty(t, newState.Discovery.Prefills)
}

func TestHandleDiscoveryAnswer_FirstListenResponse_ShortContextMarksProcessedWithoutPrefills(t *testing.T) {
	st := state.CreateInitialState()
	st.Phase = state.PhaseDiscovery
	desc := "Add upload support"
	st.SpecDescription = &desc

	newState, err := handleDiscoveryAnswer("", st, nil, "Need photo uploads in settings.", nil)
	require.NoError(t, err)
	require.NotNil(t, newState.Discovery.UserContextProcessed)
	assert.True(t, *newState.Discovery.UserContextProcessed)
	assert.Empty(t, newState.Discovery.Prefills)
	assert.Empty(t, newState.Discovery.Answers)
}

func TestHandleDiscoveryAnswer_ProcessesStoredContextBeforeModeSelection(t *testing.T) {
	st := state.CreateInitialState()
	st.Phase = state.PhaseDiscovery
	desc := "Add a safer sync workflow"
	st.SpecDescription = &desc
	st = state.SetUserContext(st, strings.Join([]string{
		"Users currently reconcile this manually and external clients read the current API response.",
		"This touches the database schema behind persisted sync metadata and fixes a regression customers keep hitting.",
		"We can ship the first slice now and leave broader cleanup for later.",
	}, " "))

	newState, err := handleDiscoveryAnswer("", st, nil, "validate", nil)
	require.NoError(t, err)
	require.NotNil(t, newState.Discovery.Mode)
	assert.Equal(t, state.DiscoveryModeValidate, *newState.Discovery.Mode)
	require.NotNil(t, newState.Discovery.UserContextProcessed)
	assert.True(t, *newState.Discovery.UserContextProcessed)
	assert.NotEmpty(t, newState.Discovery.Prefills)
}
