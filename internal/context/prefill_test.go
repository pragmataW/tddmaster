package context_test

import (
	"strings"
	"testing"

	ctx "github.com/pragmataW/tddmaster/internal/context"
	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractUserContextPrefills_RichContextCreatesReviewableSuggestions(t *testing.T) {
	userContext := strings.Join([]string{
		"Users currently export CSVs and manually compare records in Excel before every release.",
		"Existing users depend on the current endpoint, so this must stay backward compatible while we roll it out.",
		"This first version is MVP only; bulk editing and admin analytics are out of scope for v1 and can wait for phase 2.",
		"We will verify it with regression tests, an integration test around the endpoint, and updated docs.",
		"We also need to cover timeout handling, duplicate deliveries, and partial retry failures.",
	}, " ")

	prefills := ctx.ExtractUserContextPrefills(userContext)
	require.NotEmpty(t, prefills)

	byQuestion := make(map[string][]state.DiscoveryPrefillItem, len(prefills))
	for _, prefill := range prefills {
		byQuestion[prefill.QuestionID] = prefill.Items
	}

	require.NotEmpty(t, byQuestion["status_quo"])
	assert.Equal(t, "STATED", byQuestion["status_quo"][0].Type)
	require.NotEmpty(t, byQuestion["verification"])
	assert.Equal(t, "STATED", byQuestion["verification"][0].Type)
	require.NotEmpty(t, byQuestion["scope_boundary"])
	assert.Equal(t, "STATED", byQuestion["scope_boundary"][0].Type)
	require.NotEmpty(t, byQuestion["edge_cases"])
	assert.Equal(t, "STATED", byQuestion["edge_cases"][0].Type)
}

func TestExtractUserContextPrefills_UsesInferredItemsForStrongSignals(t *testing.T) {
	userContext := strings.Join([]string{
		"The change touches the database schema behind a persisted settings table and affects external clients reading the API.",
		"It also fixes a long-running regression in the sync flow that shows up when data drifts between systems.",
		"We need a small first slice now and can expand the rest in a later iteration once the core path is stable.",
		"That gives us enough room to plan carefully without rewriting the whole subsystem in one go.",
	}, " ")

	prefills := ctx.ExtractUserContextPrefills(userContext)
	require.NotEmpty(t, prefills)

	byQuestion := make(map[string][]state.DiscoveryPrefillItem, len(prefills))
	for _, prefill := range prefills {
		byQuestion[prefill.QuestionID] = prefill.Items
	}

	require.NotEmpty(t, byQuestion["reversibility"])
	assert.Equal(t, "INFERRED", byQuestion["reversibility"][0].Type)
	require.NotEmpty(t, byQuestion["user_impact"])
	assert.Equal(t, "INFERRED", byQuestion["user_impact"][0].Type)
}

func TestCompileDiscovery_AttachesPersistedPrefillsAndSuppressesRichDescription(t *testing.T) {
	st := state.CreateInitialState()
	specName := "listen-first"
	mode := state.DiscoveryModeFull
	premisesDone := true
	longDescription := strings.Repeat("Detailed discovery context that would normally trigger rich description fallback. ", 12)

	st.Phase = state.PhaseDiscovery
	st.Spec = &specName
	st.SpecDescription = &longDescription
	st.Discovery.Mode = &mode
	st.Discovery.PremisesCompleted = &premisesDone
	st.Discovery.Prefills = []state.DiscoveryPrefillQuestion{
		{
			QuestionID: "status_quo",
			Items: []state.DiscoveryPrefillItem{
				{Type: "STATED", Text: "Users currently do this by hand.", Basis: "Quoted from initial context"},
			},
		},
	}

	out := ctx.Compile(model.CompileInput{State: st})
	require.NotNil(t, out.DiscoveryData)
	assert.Nil(t, out.DiscoveryData.RichDescription)

	var statusQuo *model.DiscoveryQuestion
	for i := range out.DiscoveryData.Questions {
		if out.DiscoveryData.Questions[i].ID == "status_quo" {
			statusQuo = &out.DiscoveryData.Questions[i]
			break
		}
	}
	require.NotNil(t, statusQuo)
	require.Len(t, statusQuo.Prefills, 1)
	assert.Equal(t, "STATED", statusQuo.Prefills[0].Type)
}

func TestCompileDiscovery_HidesPrefillsOnceQuestionHasConfirmedAnswer(t *testing.T) {
	st := state.CreateInitialState()
	specName := "listen-first"
	mode := state.DiscoveryModeFull
	premisesDone := true

	st.Phase = state.PhaseDiscovery
	st.Spec = &specName
	st.Discovery.Mode = &mode
	st.Discovery.PremisesCompleted = &premisesDone
	st.Discovery.Answers = []state.DiscoveryAnswer{
		{QuestionID: "status_quo", Answer: "Users currently export CSVs and compare them manually before every release."},
	}
	st.Discovery.Prefills = []state.DiscoveryPrefillQuestion{
		{
			QuestionID: "status_quo",
			Items: []state.DiscoveryPrefillItem{
				{Type: "STATED", Text: "Users currently do this by hand.", Basis: "Quoted from initial context"},
			},
		},
	}

	out := ctx.Compile(model.CompileInput{State: st})
	require.NotNil(t, out.DiscoveryData)
	for _, question := range out.DiscoveryData.Questions {
		assert.NotEqual(t, "status_quo", question.ID)
	}
}
