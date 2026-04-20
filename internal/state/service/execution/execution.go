// Package execution applies phase-internal updates to the ExecutionState and
// related task/confidence/decision bookkeeping.
package execution

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/pragmataW/tddmaster/internal/state/model"
)

// AdvanceExecution increments the iteration counter and sets last progress.
func AdvanceExecution(state model.StateFile, progress string) (model.StateFile, error) {
	if state.Phase != model.PhaseExecuting {
		return state, fmt.Errorf("cannot advance execution in phase: %s", state.Phase)
	}
	state.Execution.Iteration++
	state.Execution.LastProgress = &progress
	return state, nil
}

// AddDecision appends a decision to the state.
func AddDecision(state model.StateFile, decision model.Decision) model.StateFile {
	state.Decisions = append(state.Decisions, decision)
	return state
}

// AddCustomAC adds a custom acceptance criterion.
func AddCustomAC(state model.StateFile, text string, user *model.UserInfo) model.StateFile {
	if state.CustomACs == nil {
		state.CustomACs = []model.CustomAC{}
	}
	id := fmt.Sprintf("custom-ac-%d", len(state.CustomACs)+1)

	userName := "Unknown User"
	userEmail := ""
	if user != nil {
		userName = user.Name
		userEmail = user.Email
	}

	ac := model.CustomAC{
		ID:           id,
		Text:         text,
		User:         userName,
		Email:        userEmail,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		AddedInPhase: state.Phase,
	}
	state.CustomACs = append(state.CustomACs, ac)
	return state
}

// AddSpecNote adds a note to the spec.
func AddSpecNote(state model.StateFile, text string, user *model.UserInfo) model.StateFile {
	if state.SpecNotes == nil {
		state.SpecNotes = []model.SpecNote{}
	}
	id := fmt.Sprintf("note-%d", len(state.SpecNotes)+1)

	userName := "Unknown User"
	userEmail := ""
	if user != nil {
		userName = user.Name
		userEmail = user.Email
	}

	note := model.SpecNote{
		ID:        id,
		Text:      text,
		User:      userName,
		Email:     userEmail,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Phase:     state.Phase,
	}
	state.SpecNotes = append(state.SpecNotes, note)
	return state
}

// ClampConfidence clamps confidence to 1-10 range.
func ClampConfidence(value float64) int {
	rounded := math.Round(value)
	if rounded < 1 {
		return 1
	}
	if rounded > 10 {
		return 10
	}
	return int(rounded)
}

// AddConfidenceFinding adds a confidence-scored finding to execution state.
func AddConfidenceFinding(state model.StateFile, finding string, confidence float64, basis string) (model.StateFile, error) {
	clamped := ClampConfidence(confidence)

	// Jidoka: high confidence requires evidence
	if clamped >= 7 && len(strings.TrimSpace(basis)) < 10 {
		return state, fmt.Errorf("high confidence (>=7) requires a basis explaining why (minimum 10 characters)")
	}

	if state.Execution.ConfidenceFindings == nil {
		state.Execution.ConfidenceFindings = []model.ConfidenceFinding{}
	}

	entry := model.ConfidenceFinding{
		Finding:    finding,
		Confidence: clamped,
		Basis:      basis,
	}
	state.Execution.ConfidenceFindings = append(state.Execution.ConfidenceFindings, entry)
	return state, nil
}

// GetLowConfidenceFindings returns findings with confidence below threshold.
func GetLowConfidenceFindings(state model.StateFile, threshold int) []model.ConfidenceFinding {
	result := make([]model.ConfidenceFinding, 0)
	for _, f := range state.Execution.ConfidenceFindings {
		if f.Confidence < threshold {
			result = append(result, f)
		}
	}
	return result
}

// GetAverageConfidence calculates the average confidence across all findings.
// Returns nil if there are no findings.
func GetAverageConfidence(state model.StateFile) *float64 {
	findings := state.Execution.ConfidenceFindings
	if len(findings) == 0 {
		return nil
	}
	sum := 0
	for _, f := range findings {
		sum += f.Confidence
	}
	avg := math.Round(float64(sum)/float64(len(findings))*10) / 10
	return &avg
}

// RecordTransition records a phase transition in the history.
func RecordTransition(state model.StateFile, from, to model.Phase, user *model.UserInfo, reason *string) model.StateFile {
	userName := "Unknown User"
	userEmail := ""
	if user != nil {
		userName = user.Name
		userEmail = user.Email
	}

	entry := model.PhaseTransition{
		From:      from,
		To:        to,
		User:      userName,
		Email:     userEmail,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Reason:    reason,
	}

	if state.TransitionHistory == nil {
		state.TransitionHistory = []model.PhaseTransition{}
	}
	state.TransitionHistory = append(state.TransitionHistory, entry)
	return state
}

// ErrTaskNotCompleted is returned by UncompleteTask when the given task ID is
// not found in CompletedTasks.
var ErrTaskNotCompleted = fmt.Errorf("task not found in completed tasks")

// UncompleteTask reverses the completion of a single task identified by taskID.
// It removes the ID from CompletedTasks and sets Completed=false on the matching
// OverrideTasks entry (if present). Phase, Iteration, and TDDCycle are not
// touched — this is a task-flag flip only.
func UncompleteTask(state model.StateFile, taskID string) (model.StateFile, error) {
	found := false
	kept := make([]string, 0, len(state.Execution.CompletedTasks))
	for _, id := range state.Execution.CompletedTasks {
		if id == taskID {
			found = true
			continue
		}
		kept = append(kept, id)
	}
	if !found {
		return state, fmt.Errorf("%w: %s", ErrTaskNotCompleted, taskID)
	}
	state.Execution.CompletedTasks = kept

	for i := range state.OverrideTasks {
		if state.OverrideTasks[i].ID == taskID {
			state.OverrideTasks[i].Completed = false
			break
		}
	}

	return state, nil
}
