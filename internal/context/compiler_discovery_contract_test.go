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

func TestCompile_DiscoveryModeSelection_EmitsInteractiveOptionsAndAskUserQuestionHint(t *testing.T) {
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

	require.Len(t, out.InteractiveOptions, 5, "mode selection must expose 5 options at top level")
	require.NotNil(t, out.ToolHint)
	assert.Equal(t, "AskUserQuestion", *out.ToolHint)
	require.NotNil(t, out.ToolHintInstruction)
	assert.Contains(t, *out.ToolHintInstruction, "AskUserQuestion")

	labels := make([]string, len(out.InteractiveOptions))
	for i, opt := range out.InteractiveOptions {
		labels[i] = opt.Label
	}
	assert.Equal(t, []string{"Full discovery", "Validate my plan", "Technical depth", "Ship fast", "Explore scope"}, labels)

	require.NotNil(t, out.CommandMap)
	for _, label := range labels {
		cmd, ok := out.CommandMap[label]
		require.Truef(t, ok, "commandMap must contain an entry for %q", label)
		assert.Contains(t, cmd, "next --answer=")
	}

	require.NotNil(t, out.DiscoveryData)
	require.NotNil(t, out.DiscoveryData.ModeSelection)
	msLabels := make([]string, len(out.DiscoveryData.ModeSelection.Options))
	for i, o := range out.DiscoveryData.ModeSelection.Options {
		msLabels[i] = o.Label
	}
	assert.Equal(t, labels, msLabels, "modeSelection.options and interactiveOptions must share the same source of truth")

	assert.Contains(t, out.DiscoveryData.Instruction, "AskUserQuestion")
	assert.Contains(t, out.DiscoveryData.ModeSelection.Instruction, "AskUserQuestion")
}

func TestCompile_DiscoveryListenFirst_InstructsAskUserQuestionViaBehavioralRule(t *testing.T) {
	st := state.CreateInitialState()
	specName := "discovery-contract"
	desc := "Add a more predictable discovery flow"

	st.Phase = state.PhaseDiscovery
	st.Spec = &specName
	st.SpecDescription = &desc
	// No UserContext — listen-first sub-step is active.

	out := ctx.Compile(st, nil, nil, nil, nil, nil, nil, nil, nil, 0)

	assert.Empty(t, out.InteractiveOptions, "listen-first is open-ended; no interactiveOptions should be emitted")

	combined := strings.Join(out.Behavioral.Rules, "\n")
	assert.Contains(t, combined, "Listen-first", "behavioral rules must cover the listen-first sub-step explicitly")
	assert.Contains(t, combined, "AskUserQuestion", "listen-first rule must direct the agent to AskUserQuestion")
	assert.Contains(t, combined, "free-form", "listen-first rule must signal open-ended input")
}

func TestCompile_DiscoveryPremiseChallenge_InstructsPerPremiseAskUserQuestion(t *testing.T) {
	st := state.CreateInitialState()
	specName := "discovery-contract"
	desc := "Add a more predictable discovery flow"
	userContext := "Sufficient context provided."
	processed := true
	mode := state.DiscoveryModeFull

	st.Phase = state.PhaseDiscovery
	st.Spec = &specName
	st.SpecDescription = &desc
	st.Discovery.UserContext = &userContext
	st.Discovery.UserContextProcessed = &processed
	st.Discovery.Mode = &mode
	// PremisesCompleted left nil/false → premise challenge sub-step is active.

	out := ctx.Compile(st, nil, nil, nil, nil, nil, nil, nil, nil, 0)

	require.NotNil(t, out.DiscoveryData)
	require.NotNil(t, out.DiscoveryData.PremiseChallenge, "premise challenge sub-step must be surfaced")

	combined := strings.Join(out.Behavioral.Rules, "\n")
	assert.Contains(t, combined, "Premise challenge", "behavioral rules must cover premise challenge")
	assert.Contains(t, combined, "AskUserQuestion per premise", "premise challenge rule must require one AskUserQuestion per premise")
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

func TestCompile_DiscoveryReview_RendersAnswersBeforeApprovalChoices(t *testing.T) {
	st := state.CreateInitialState()
	specName := "discovery-contract"

	st.Phase = state.PhaseDiscoveryRefinement
	st.Spec = &specName
	st.Discovery.Answers = []state.DiscoveryAnswer{
		{QuestionID: "status_quo", Answer: "Users currently compare exported CSV files by hand before every release."},
		{QuestionID: "ambition", Answer: "The 1-star version removes manual comparison; the 10-star version makes drift obvious immediately."},
	}

	out := ctx.Compile(st, nil, nil, nil, nil, nil, nil, nil, nil, 0)

	require.NotNil(t, out.DiscoveryReviewData)
	assert.Contains(t, out.DiscoveryReviewData.Instruction, "FIRST render `discoveryReviewData.reviewSummary`")
	assert.Contains(t, out.DiscoveryReviewData.ReviewSummary, "[status_quo] What does the user do today without this feature?")
	assert.Contains(t, out.DiscoveryReviewData.ReviewSummary, "Answer: Users currently compare exported CSV files by hand before every release.")
	assert.Contains(t, out.DiscoveryReviewData.ReviewSummary, "[ambition] Describe the 1-star and 10-star versions.")

	combined := strings.Join(out.Behavioral.Rules, "\n")
	assert.Contains(t, combined, "discoveryReviewData.reviewSummary")
	assert.Contains(t, combined, "Do NOT jump straight to interactiveOptions")

	require.NotNil(t, out.Gate)
	assert.Equal(t, "2/7 answers collected.", out.Gate.Message)
}
