// Package phases contains the small per-phase compilers for IDLE, BLOCKED,
// COMPLETED, SPEC_PROPOSAL, and SPEC_APPROVED. DISCOVERY and EXECUTION live in
// their own sub-packages because they are much larger.
package phases

import (
	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/state"
)

// Renderer is the minimal command-builder interface required by this package.
type Renderer interface {
	C(sub string) string
	CS(sub string, specName *string) string
}

func strPtr(s string) *string { return &s }

// CompileIdle renders the IDLE phase payload — welcome banner, existing specs,
// and the available concerns catalogue.
func CompileIdle(
	activeConcerns []state.ConcernDefinition,
	allConcernDefs []state.ConcernDefinition,
	rulesCount int,
	idleContext *model.IdleContext,
) model.IdleOutput {
	availableConcerns := make([]model.ConcernInfo, len(allConcernDefs))
	for i, cc := range allConcernDefs {
		availableConcerns[i] = model.ConcernInfo{ID: cc.ID, Description: cc.Description}
	}

	activeIDs := make([]string, len(activeConcerns))
	for i, cc := range activeConcerns {
		activeIDs[i] = cc.ID
	}

	activeRulesCount := rulesCount
	existingSpecs := []model.SpecSummary{}
	if idleContext != nil {
		existingSpecs = idleContext.ExistingSpecs
		if idleContext.RulesCount != nil {
			activeRulesCount = *idleContext.RulesCount
		}
	}

	behavioralNote := strPtr("These options are fallbacks. If the user already described what they want, act on it directly without presenting these options.")

	var hint *string
	if len(activeConcerns) == 0 {
		hint = strPtr("No concerns active. Consider adding concerns first — they shape discovery questions and specs.")
	}

	return model.IdleOutput{
		Phase:             "IDLE",
		Instruction:       "No active spec. If the user described what they want, run `tddmaster spec new \"description\"` immediately — name is auto-generated. Present ALL available concerns (split across multiple calls if needed).",
		Welcome:           model.Welcome,
		ExistingSpecs:     existingSpecs,
		AvailableConcerns: availableConcerns,
		ActiveConcerns:    activeIDs,
		ActiveRulesCount:  activeRulesCount,
		BehavioralNote:    behavioralNote,
		Hint:              hint,
	}
}
