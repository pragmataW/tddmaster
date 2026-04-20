package meta

import (
	"fmt"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
)

// BuildGate returns the GateInfo for phases that require a user-visible gate
// (DISCOVERY_REFINEMENT, SPEC_APPROVED). Returns nil for phases without gates.
func BuildGate(st state.StateFile, parsedSpec *spec.ParsedSpec) *model.GateInfo {
	switch st.Phase {
	case state.PhaseDiscoveryRefinement:
		totalQuestions := len(model.Questions)
		return &model.GateInfo{
			Message: fmt.Sprintf("%d/%d answers collected.", len(st.Discovery.Answers), totalQuestions),
			Action:  "Type APPROVE to generate spec, or REVISE to correct answers.",
			Phase:   "DISCOVERY_REFINEMENT",
		}
	case state.PhaseSpecApproved:
		taskCount := 0
		if parsedSpec != nil {
			taskCount = len(parsedSpec.Tasks)
		}
		return &model.GateInfo{
			Message: fmt.Sprintf("Spec approved. %d tasks ready.", taskCount),
			Action:  "Type START to begin execution.",
			Phase:   "SPEC_APPROVED",
		}
	}
	return nil
}
