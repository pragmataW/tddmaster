// Package tdd builds the TDD behavioural rules and per-phase verification
// context consumed by the Red-Green-Refactor pipeline.
package tdd

import (
	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/state"
	statemodel "github.com/pragmataW/tddmaster/internal/state/model"
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

// VerifierRequired returns true when the verifier sub-agent must run for the
// given manifest + TDD phase combination.
//
// Rules:
//   - manifest==nil or manifest.Tdd==nil → treat as skipVerify=false → true.
//   - skipVerify=false → always true (AC-1).
//   - skipVerify=true + TDD=off → false (AC-2).
//   - skipVerify=true + TDD=on + phase=green → true (AC-3).
//   - skipVerify=true + TDD=on + phase∈{red,refactor} → false (AC-4).
func VerifierRequired(manifest *statemodel.NosManifest, phase string) bool {
	if manifest == nil || manifest.Tdd == nil {
		// No skip flag configured → verifier is required.
		return true
	}
	if !manifest.IsVerifierSkipped() {
		// skipVerify=false → always require verifier.
		return true
	}
	// skipVerify=true from here on.
	if !manifest.IsTDDEnabled() {
		// TDD off → skip.
		return false
	}
	// TDD on → only green phase requires verifier.
	return phase == statemodel.TDDCycleGreen
}

// BuildRefactorInstructions packages verifier refactor notes into an executor
// directive. Returns nil when there are no notes to apply or when the executor
// has already consumed the current batch. When verifierRequired=false (skip-
// verify in REFACTOR phase) the instruction text tells the executor to submit
// `refactorApplied:true` and `completed:[<id>]` together, since no verifier
// will re-run to advance the cycle.
func BuildRefactorInstructions(st state.StateFile, maxRounds int, verifierRequired bool) *model.RefactorInstructions {
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
	instruction := model.RefactorInstructionsText
	if !verifierRequired {
		instruction = model.RefactorInstructionsSkipVerifyText
	}
	return &model.RefactorInstructions{
		Notes:       notes,
		Instruction: instruction,
		Round:       st.Execution.RefactorRounds + 1,
		MaxRounds:   maxRounds,
	}
}
