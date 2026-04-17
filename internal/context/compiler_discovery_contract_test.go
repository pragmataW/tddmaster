package context_test

import (
	"strings"
	"testing"

	ctx "github.com/pragmataW/tddmaster/internal/context"
	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompile_IdleBehavioralRules_UseSupportedDiscoveryModesOnly(t *testing.T) {
	st := state.CreateInitialState()
	st.Phase = state.PhaseIdle

	out := ctx.Compile(st, nil, nil, nil, nil, nil, nil, nil, nil, 0)
	combined := strings.Join(out.Behavioral.Rules, "\n")

	assert.Contains(t, combined, "full, validate, technical-depth, ship-fast, or explore")
	assert.NotContains(t, combined, "quick discovery")
	assert.NotContains(t, combined, "skip to spec draft")
}

func TestCompile_DiscoveryModeSelection_ListsSupportedModesOnly(t *testing.T) {
	st := state.CreateInitialState()
	specName := "discovery-contract"
	desc := "Add a more predictable discovery flow"
	userContext := "The user already shared enough context to move directly into discovery mode selection."
	processed := true

	st.Phase = state.PhaseDiscovery
	st.Spec = &specName
	st.SpecDescription = &desc
	st.Discovery.UserContext = &userContext
	st.Discovery.UserContextProcessed = &processed

	out := ctx.Compile(st, nil, nil, nil, nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.DiscoveryData)
	require.NotNil(t, out.DiscoveryData.ModeSelection)

	var ids []string
	for _, option := range out.DiscoveryData.ModeSelection.Options {
		ids = append(ids, option.ID)
	}

	assert.Equal(t, []string{"full", "validate", "technical-depth", "ship-fast", "explore"}, ids)
}

func TestCompile_DiscoveryBehavioralRules_KeepSequentialContract(t *testing.T) {
	st := state.CreateInitialState()
	st.Phase = state.PhaseDiscovery

	out := ctx.Compile(st, nil, nil, nil, nil, nil, nil, nil, nil, 0)
	combined := strings.Join(out.Behavioral.Rules, "\n")

	assert.Contains(t, combined, "Keep discovery sequential")
	assert.NotContains(t, combined, "Submit all answers together")
	assert.NotContains(t, combined, "skip questions but MUST run premise challenge and alternatives")
}

func TestCompile_DiscoveryHumanMode_ReturnsCurrentQuestionOnly(t *testing.T) {
	st := state.CreateInitialState()
	specName := "discovery-contract"
	mode := state.DiscoveryModeFull
	premisesDone := true

	st.Phase = state.PhaseDiscovery
	st.Spec = &specName
	st.Discovery.Mode = &mode
	st.Discovery.PremisesCompleted = &premisesDone
	st.Discovery.Answers = []state.DiscoveryAnswer{
		{QuestionID: "status_quo", Answer: "Users currently compare exported CSV files by hand before every release."},
	}

	out := ctx.Compile(st, nil, nil, nil, nil, nil, nil, nil, nil, 0)
	require.NotNil(t, out.DiscoveryData)
	require.Len(t, out.DiscoveryData.Questions, 1)
	assert.Equal(t, "ambition", out.DiscoveryData.Questions[0].ID)
	require.NotNil(t, out.DiscoveryData.CurrentQuestion)
	assert.Equal(t, 1, *out.DiscoveryData.CurrentQuestion)
	require.NotNil(t, out.DiscoveryData.TotalQuestions)
	assert.Equal(t, 7, *out.DiscoveryData.TotalQuestions)
	assert.Contains(t, out.DiscoveryData.Transition.OnComplete, `--answer="<answer>"`)
	assert.NotContains(t, out.DiscoveryData.Transition.OnComplete, `{"status_quo"`)
}
