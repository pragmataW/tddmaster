// Package tdd builds the TDD behavioural rules and per-phase verification
// context consumed by the Red-Green-Refactor pipeline.
package tdd

import (
	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/state"
)

// InjectRules appends TDD behavioural rules to rules and returns the combined
// slice. The original slice is not modified. When TDD mode is inactive the
// caller must skip this function entirely.
func InjectRules(rules []string) []string {
	combined := make([]string, len(rules), len(rules)+len(model.TDDBehavioralRules))
	copy(combined, rules)
	return append(combined, model.TDDBehavioralRules...)
}

// BuildVerificationContext returns phase-specific verification instructions
// for the verifier. Each phase spells out the expected exit-code contract and
// the JSON output shape the verifier must return so RecordTDDVerificationFull
// can make cycle transitions.
func BuildVerificationContext(cycle string) *model.TDDVerificationContext {
	var instruction string
	switch cycle {
	case state.TDDCycleRed:
		instruction = VerifierRedPhaseInstruction()
	case state.TDDCycleGreen:
		instruction = VerifierGreenPhaseInstruction("", "")
	case state.TDDCycleRefactor:
		instruction = VerifierRefactorPhaseInstruction("", "")
	default:
		return nil
	}
	return &model.TDDVerificationContext{Phase: cycle, Instruction: instruction}
}

// BuildRefactorInstructions packages verifier refactor notes into an executor
// directive. Returns nil when there are no notes to apply or when the executor
// has already consumed the current batch.
func BuildRefactorInstructions(st state.StateFile, maxRounds int) *model.RefactorInstructions {
	if st.Execution.TDDCycle != state.TDDCycleRefactor {
		return nil
	}
	if st.Execution.LastVerification == nil {
		return nil
	}
	if st.Execution.RefactorApplied {
		return nil
	}
	notes := st.Execution.LastVerification.RefactorNotes
	if len(notes) == 0 {
		return nil
	}
	return &model.RefactorInstructions{
		Notes:       notes,
		Instruction: model.RefactorInstructionsText,
		Round:       st.Execution.RefactorRounds + 1,
		MaxRounds:   maxRounds,
	}
}
