// Package machine owns tddmaster's phase state machine: the transition guards
// (CanTransition/AssertTransition/Transition) and the phase-changing mutations
// (StartSpec, CompleteDiscovery, ApproveSpec, StartExecution, BlockExecution,
// CompleteSpec, ReopenSpec, ResumeCompletedSpec, RevisitSpec, ResetToIdle).
package machine

import (
	"fmt"
	"strings"

	"github.com/pragmataW/tddmaster/internal/state/model"
)

// CanTransition returns true if the transition from -> to is valid.
func CanTransition(from, to model.Phase) bool {
	allowed, ok := model.ValidTransitions[from]
	if !ok {
		return false
	}
	for _, p := range allowed {
		if p == to {
			return true
		}
	}
	return false
}

// AssertTransition returns an error if the transition is not valid.
func AssertTransition(from, to model.Phase) error {
	if !CanTransition(from, to) {
		allowed := model.ValidTransitions[from]
		parts := make([]string, len(allowed))
		for i, p := range allowed {
			parts[i] = string(p)
		}
		return fmt.Errorf("invalid phase transition: %s → %s. Allowed: %s",
			from, to, strings.Join(parts, ", "))
	}
	return nil
}

// Transition moves state to the given phase (validates transition first).
func Transition(state model.StateFile, to model.Phase) (model.StateFile, error) {
	if err := AssertTransition(state.Phase, to); err != nil {
		return state, err
	}
	state.Phase = to
	return state, nil
}
