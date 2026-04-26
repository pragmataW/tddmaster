// Package tdd contains the RED → GREEN → REFACTOR cycle logic and task-level
// TDD resolution helpers used by cmd/next.go and the verifier pipeline.
package tdd

import (
	"fmt"
	"slices"
	"time"

	"github.com/pragmataW/tddmaster/internal/state/model"
	"github.com/pragmataW/tddmaster/internal/state/service/machine"
)

// CurrentTaskID returns the ID of the first task in OverrideTasks that is not
// present in Execution.CompletedTasks. Returns "" when all known tasks have
// been completed (or when OverrideTasks is empty).
func CurrentTaskID(st model.StateFile) string {
	if len(st.OverrideTasks) == 0 {
		return ""
	}
	completed := make(map[string]bool, len(st.Execution.CompletedTasks))
	for _, id := range st.Execution.CompletedTasks {
		completed[id] = true
	}
	for _, t := range st.OverrideTasks {
		if !completed[t.ID] {
			return t.ID
		}
	}
	return ""
}

// IsTaskTDDEnabled resolves the effective TDD setting for a single task.
// Resolution order:
//  1. Task-level override (SpecTask.TDDEnabled) when non-nil.
//  2. Spec-level manifest setting (cfg.IsTDDEnabled()).
//
// Returns false when the task is not found.
func IsTaskTDDEnabled(st model.StateFile, taskID string, cfg *model.NosManifest) bool {
	specLevel := cfg != nil && cfg.IsTDDEnabled()
	for _, t := range st.OverrideTasks {
		if t.ID != taskID {
			continue
		}
		if t.TDDEnabled != nil {
			return *t.TDDEnabled
		}
		return specLevel
	}
	return specLevel
}

// ShouldRunTDDForCurrentTask reports whether the next uncompleted task should
// run through the RED/GREEN/REFACTOR cycle.
func ShouldRunTDDForCurrentTask(st model.StateFile, cfg *model.NosManifest) bool {
	id := CurrentTaskID(st)
	if id == "" {
		return cfg != nil && cfg.IsTDDEnabled()
	}
	return IsTaskTDDEnabled(st, id, cfg)
}

// AnyTaskUsesTDD returns true when at least one known task would run under TDD.
func AnyTaskUsesTDD(st model.StateFile, cfg *model.NosManifest) bool {
	specLevel := cfg != nil && cfg.IsTDDEnabled()
	if len(st.OverrideTasks) == 0 {
		return specLevel
	}
	for _, t := range st.OverrideTasks {
		if t.TDDEnabled != nil {
			if *t.TDDEnabled {
				return true
			}
			continue
		}
		if specLevel {
			return true
		}
	}
	return false
}

// RecordTDDVerification records a TDD verification result, re-queues failed ACs,
// and auto-transitions to BLOCKED when MaxVerificationRetries is reached.
// maxRetries=0 disables the auto-block (treated as unlimited).
// This legacy wrapper skips cycle transitions and refactor-note tracking.
func RecordTDDVerification(
	st model.StateFile,
	maxRetries int,
	passed bool,
	output string,
	failedACs []string,
	uncoveredEdgeCases []string,
) (model.StateFile, error) {
	return RecordTDDVerificationFull(st, maxRetries, 0, passed, output, failedACs, uncoveredEdgeCases, nil, nil)
}

// RecordTDDVerificationFull is the full-featured TDD verification recorder. It
// handles RED→GREEN→REFACTOR cycle transitions and verifier→executor refactor
// note round-tripping in addition to the legacy fail-count/requeue/block logic.
//
// Transitions (when passed==true and st.Execution.TDDCycle is set):
//   - red      → green        (failing tests confirmed, ready for implementation)
//   - green    → refactor     (tests pass, invite refactor notes)
//   - refactor → refactor     (new notes stored; executor applies next)
//     or next-task reset (TDDCycle="") when notes are empty or the round
//     cap is reached.
//
// maxRefactorRounds=0 means "unlimited rounds as long as notes keep coming";
// the default manifest value is 3.
func RecordTDDVerificationFull(
	st model.StateFile,
	maxRetries int,
	maxRefactorRounds int,
	passed bool,
	output string,
	failedACs []string,
	uncoveredEdgeCases []string,
	refactorNotes []model.RefactorNote,
	cfg *model.NosManifest,
) (model.StateFile, error) {
	if st.Phase != model.PhaseExecuting {
		return st, fmt.Errorf("cannot record TDD verification in phase: %s", st.Phase)
	}

	prevFailCount := 0
	if st.Execution.LastVerification != nil {
		prevFailCount = st.Execution.LastVerification.VerificationFailCount
	}
	newFailCount := prevFailCount
	if !passed {
		newFailCount++
	}

	phaseSnapshot := st.Execution.TDDCycle
	now := time.Now().UTC().Format(time.RFC3339)
	st.Execution.LastVerification = &model.VerificationResult{
		Passed:                passed,
		Output:                output,
		Timestamp:             now,
		UncoveredEdgeCases:    uncoveredEdgeCases,
		VerificationFailCount: newFailCount,
		RefactorNotes:         refactorNotes,
		Phase:                 phaseSnapshot,
		FailedACs:             failedACs,
	}

	if !passed {
		if maxRetries > 0 && newFailCount >= maxRetries {
			reason := fmt.Sprintf("verifier max retry reached (%d/%d)", newFailCount, maxRetries)
			return machine.BlockExecution(st, reason)
		}

		return st, nil
	}

	switch phaseSnapshot {
	case model.TDDCycleRed:
		st.Execution.TDDCycle = model.TDDCycleGreen
		st.Execution.LastVerification.VerificationFailCount = 0

	case model.TDDCycleGreen:
		st.Execution.LastVerification.VerificationFailCount = 0
		if len(refactorNotes) == 0 {
			resetCycleForNextTask(&st)
		} else {
			st.Execution.TDDCycle = model.TDDCycleRefactor
			st.Execution.RefactorRounds = 0
			st.Execution.RefactorApplied = false
			if shouldPopulatePendingNotes(cfg) {
				st.Execution.PendingRefactorNotes = refactorNotes
			}
		}

	case model.TDDCycleRefactor:
		if !st.Execution.RefactorApplied {
			if len(refactorNotes) == 0 {
				resetCycleForNextTask(&st)
			}
		} else {
			st.Execution.RefactorRounds++
			capReached := maxRefactorRounds > 0 && st.Execution.RefactorRounds >= maxRefactorRounds
			if len(refactorNotes) == 0 || capReached {
				resetCycleForNextTask(&st)
			} else {
				st.Execution.RefactorApplied = false
				if shouldPopulatePendingNotes(cfg) {
					st.Execution.PendingRefactorNotes = refactorNotes
				}
			}
		}
	}

	return st, nil
}

// shouldPopulatePendingNotes reports whether refactor notes should be stored
// for a verifier-skipped TDD-enabled configuration.
func shouldPopulatePendingNotes(cfg *model.NosManifest) bool {
	return cfg != nil && cfg.IsVerifierSkipped() && cfg.IsTDDEnabled()
}

// resetCycleForNextTask marks the current task as completed and clears per-task
// TDD cycle state. Called from RecordTDDVerificationFull when the verifier signals
// that the cycle is finished (GREEN with empty refactor notes, REFACTOR with no
// notes, or refactor-round cap reached). Without the completion step the
// downstream reseed in cmd/next.go would pick the same task and loop it back to
// RED instead of advancing.
func resetCycleForNextTask(st *model.StateFile) {
	if id := CurrentTaskID(*st); id != "" {
		if !taskInCompletedList(*st, id) {
			st.Execution.CompletedTasks = append(st.Execution.CompletedTasks, id)
		}
		for i := range st.OverrideTasks {
			if st.OverrideTasks[i].ID == id {
				st.OverrideTasks[i].Completed = true
				break
			}
		}
	}
	st.Execution.TDDCycle = ""
	st.Execution.RefactorRounds = 0
	st.Execution.RefactorApplied = false
}

func taskInCompletedList(st model.StateFile, id string) bool {
	return slices.Contains(st.Execution.CompletedTasks, id)
}

// StartTDDCycleForTask initializes the TDD cycle at RED for the current task
// and clears any carried-over refactor state.
func StartTDDCycleForTask(st *model.StateFile) {
	st.Execution.TDDCycle = model.TDDCycleRed
	st.Execution.RefactorRounds = 0
	st.Execution.RefactorApplied = false
}

// MarkRefactorApplied flips the RefactorApplied flag; called by cmd/next.go
// when the executor reports that it consumed the verifier's refactor notes.
func MarkRefactorApplied(st *model.StateFile) {
	st.Execution.RefactorApplied = true
}
