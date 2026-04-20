// Package discovery applies phase-internal updates to the DiscoveryState,
// FollowUp, and Delegation structures that back tddmaster's discovery phase.
package discovery

import (
	"fmt"
	"strings"

	"github.com/pragmataW/tddmaster/internal/state/model"
)

// SetDiscoveryMode sets the discovery mode. Only valid in DISCOVERY phase.
func SetDiscoveryMode(state model.StateFile, mode model.DiscoveryMode) (model.StateFile, error) {
	if state.Phase != model.PhaseDiscovery {
		return state, fmt.Errorf("cannot set discovery mode in phase: %s", state.Phase)
	}
	state.Discovery.Mode = &mode
	return state, nil
}

// CompletePremises records premises and marks them completed. Only valid in DISCOVERY phase.
func CompletePremises(state model.StateFile, premises []model.Premise) (model.StateFile, error) {
	if state.Phase != model.PhaseDiscovery {
		return state, fmt.Errorf("cannot complete premises in phase: %s", state.Phase)
	}
	t := true
	state.Discovery.Premises = premises
	state.Discovery.PremisesCompleted = &t
	return state, nil
}

// SelectApproach records the selected approach. Only valid in DISCOVERY_REFINEMENT phase.
func SelectApproach(state model.StateFile, approach model.SelectedApproach) (model.StateFile, error) {
	if state.Phase != model.PhaseDiscoveryRefinement {
		return state, fmt.Errorf("cannot select approach in phase: %s", state.Phase)
	}
	t := true
	state.Discovery.SelectedApproach = &approach
	state.Discovery.AlternativesPresented = &t
	return state, nil
}

// SkipAlternatives marks alternatives as presented without selecting one.
func SkipAlternatives(state model.StateFile) (model.StateFile, error) {
	if state.Phase != model.PhaseDiscoveryRefinement {
		return state, fmt.Errorf("cannot skip alternatives in phase: %s", state.Phase)
	}
	t := true
	state.Discovery.AlternativesPresented = &t
	return state, nil
}

// AddDiscoveryAnswer adds or replaces a discovery answer for a question.
// Validates that answer is at least 20 chars (Jidoka).
func AddDiscoveryAnswer(state model.StateFile, questionID, answer string) (model.StateFile, error) {
	if state.Phase != model.PhaseDiscovery && state.Phase != model.PhaseDiscoveryRefinement {
		return state, fmt.Errorf("cannot add discovery answer in phase: %s", state.Phase)
	}

	if len(strings.TrimSpace(answer)) < 20 {
		return state, fmt.Errorf("answer too short. Discovery answers must be meaningful (minimum 20 characters)")
	}

	existingAnswers := make([]model.DiscoveryAnswer, 0, len(state.Discovery.Answers))
	for _, a := range state.Discovery.Answers {
		if a.QuestionID != questionID {
			existingAnswers = append(existingAnswers, a)
		}
	}

	newAnswer := model.DiscoveryAnswer{
		QuestionID: questionID,
		Answer:     answer,
	}

	state.Discovery.Answers = append(existingAnswers, newAnswer)
	return state, nil
}

// AdvanceDiscoveryQuestion increments the current question index.
func AdvanceDiscoveryQuestion(state model.StateFile) (model.StateFile, error) {
	if state.Phase != model.PhaseDiscovery {
		return state, fmt.Errorf("cannot advance discovery question in phase: %s", state.Phase)
	}
	state.Discovery.CurrentQuestion++
	return state, nil
}

// ApproveDiscoveryAnswers approves discovery answers without transitioning phase.
// Used when a split proposal is detected — stays in DISCOVERY_REFINEMENT.
func ApproveDiscoveryAnswers(state model.StateFile) (model.StateFile, error) {
	if state.Phase != model.PhaseDiscoveryRefinement {
		return state, fmt.Errorf("cannot approve discovery answers in phase: %s", state.Phase)
	}
	state.Discovery.Approved = true
	return state, nil
}

// SetUserContext stores user context shared before discovery starts.
func SetUserContext(state model.StateFile, context string) model.StateFile {
	f := false
	state.Discovery.UserContext = &context
	state.Discovery.Prefills = []model.DiscoveryPrefillQuestion{}
	state.Discovery.UserContextProcessed = &f
	return state
}

// SetDiscoveryPrefills stores persisted discovery suggestions derived from user context.
func SetDiscoveryPrefills(state model.StateFile, prefills []model.DiscoveryPrefillQuestion) model.StateFile {
	if prefills == nil {
		state.Discovery.Prefills = []model.DiscoveryPrefillQuestion{}
		return state
	}
	copied := make([]model.DiscoveryPrefillQuestion, len(prefills))
	for i, prefill := range prefills {
		items := make([]model.DiscoveryPrefillItem, len(prefill.Items))
		copy(items, prefill.Items)
		copied[i] = model.DiscoveryPrefillQuestion{
			QuestionID: prefill.QuestionID,
			Items:      items,
		}
	}
	state.Discovery.Prefills = copied
	return state
}

// MarkUserContextProcessed marks user context as processed (pre-fill done).
func MarkUserContextProcessed(state model.StateFile) model.StateFile {
	t := true
	state.Discovery.UserContextProcessed = &t
	return state
}

// SetContributors sets contributors for a spec.
func SetContributors(state model.StateFile, contributors []string) model.StateFile {
	state.Discovery.Contributors = contributors
	return state
}
