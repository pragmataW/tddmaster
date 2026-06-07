package loop

import (
	"errors"

	"github.com/pragmataW/tddmaster/internal/spec"
)

const (
	cycleRed      = "red"
	cycleGreen    = "green"
	cycleRefactor = "refactor"
	cycleEmpty    = ""
)

func refactorCapReached(rounds, max int) bool {
	return max > 0 && rounds >= max
}

func isRefactorBypass(st spec.ExecState, passed bool, refactorNotesPresent bool) bool {
	return st.TDDCycle == cycleRefactor && passed && refactorNotesPresent && !st.RefactorApplied
}

func resetRefactorCounters(st spec.ExecState) spec.ExecState {
	st.RefactorRounds = 0
	st.RefactorApplied = false
	return st
}

func advanceCycle(st spec.ExecState, passed bool, refactorNotesPresent bool, maxRefactorRounds int) (spec.ExecState, bool) {
	if !passed {
		return st, false
	}

	switch st.TDDCycle {
	case cycleRed:
		st.TDDCycle = cycleGreen
		return st, false

	case cycleGreen:
		if refactorNotesPresent {
			st.TDDCycle = cycleRefactor
			st = resetRefactorCounters(st)
			return st, false
		}
		st.TDDCycle = cycleEmpty
		return st, true

	case cycleRefactor:
		st.RefactorRounds++
		if !refactorNotesPresent || refactorCapReached(st.RefactorRounds, maxRefactorRounds) {
			st.TDDCycle = cycleEmpty
			return st, true
		}
		return st, false

	default:
		return st, false
	}
}

func advanceCycleStrict(st spec.ExecState, passed bool, refactorNotesPresent bool, maxRefactorRounds int) (spec.ExecState, bool, error) {
	if isRefactorBypass(st, passed, refactorNotesPresent) {
		return st, false, errors.New("refactor bypass: cannot complete refactor phase with pending notes before applying refactor")
	}
	newSt, taskComplete := advanceCycle(st, passed, refactorNotesPresent, maxRefactorRounds)
	return newSt, taskComplete, nil
}

func completeCurrentTask(tasks []spec.Task, idx int) []spec.Task {
	result := make([]spec.Task, len(tasks))
	copy(result, tasks)
	result[idx].Done = true
	return result
}

func reseedCycle(st spec.ExecState, taskTDDEnabled bool) spec.ExecState {
	st = resetRefactorCounters(st)
	st.Implemented = false
	if taskTDDEnabled {
		st.TDDCycle = cycleRed
	} else {
		st.TDDCycle = cycleEmpty
	}
	return st
}
