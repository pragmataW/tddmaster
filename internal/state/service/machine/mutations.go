package machine

import (
	"fmt"
	"time"

	"github.com/pragmataW/tddmaster/internal/state/model"
	"github.com/pragmataW/tddmaster/internal/state/service/discovery"
	"github.com/pragmataW/tddmaster/internal/state/service/paths"
)

// specFilePath returns the path for a spec file inside a StateFile context —
// mirrors paths.Paths{}.SpecFile but inlined so mutations stays free of the
// pathHelpers receiver pattern used elsewhere.
func specFilePath(specName string) string {
	return paths.TddmasterDir + "/specs/" + specName + "/spec.md"
}

func freshDiscovery() model.DiscoveryState {
	return model.DiscoveryState{
		Answers:         []model.DiscoveryAnswer{},
		Prefills:        []model.DiscoveryPrefillQuestion{},
		Completed:       false,
		CurrentQuestion: 0,
		Audience:        "human",
		Approved:        false,
	}
}

func freshExecution() model.ExecutionState {
	return model.ExecutionState{
		Iteration:            0,
		LastProgress:         nil,
		ModifiedFiles:        []string{},
		LastVerification:     nil,
		AwaitingStatusReport: false,
		Debt:                 nil,
		CompletedTasks:       []string{},
		DebtCounter:          0,
		NaItems:              []string{},
	}
}

// StartSpec transitions to DISCOVERY and initializes spec-related state.
func StartSpec(state model.StateFile, specName, branch string, description *string) (model.StateFile, error) {
	if err := AssertTransition(state.Phase, model.PhaseDiscovery); err != nil {
		return state, err
	}

	state.Phase = model.PhaseDiscovery
	state.Spec = &specName
	state.SpecDescription = description
	state.Branch = &branch
	state.Discovery = freshDiscovery()
	state.SpecState = model.SpecState{Path: nil, Status: "none"}
	state.Execution = freshExecution()
	state.Decisions = []model.Decision{}
	return state, nil
}

// CompleteDiscovery transitions from DISCOVERY to DISCOVERY_REFINEMENT.
// Blocks if there are pending follow-ups (Jidoka I2).
func CompleteDiscovery(state model.StateFile) (model.StateFile, error) {
	if state.Phase != model.PhaseDiscovery {
		return state, fmt.Errorf("cannot complete discovery in phase: %s", state.Phase)
	}

	pending := discovery.GetPendingFollowUps(state)
	if len(pending) > 0 {
		return state, fmt.Errorf("cannot complete discovery: %d pending follow-up(s). Answer or skip them first", len(pending))
	}

	specPath := ""
	if state.Spec != nil {
		specPath = specFilePath(*state.Spec)
	}

	state.Phase = model.PhaseDiscoveryRefinement
	state.Discovery.Completed = true
	state.SpecState = model.SpecState{
		Path:   &specPath,
		Status: "draft",
	}
	return state, nil
}

// ApproveDiscoveryReview transitions to SPEC_PROPOSAL.
func ApproveDiscoveryReview(state model.StateFile) (model.StateFile, error) {
	if err := AssertTransition(state.Phase, model.PhaseSpecProposal); err != nil {
		return state, err
	}
	state.Phase = model.PhaseSpecProposal
	return state, nil
}

// ApproveSpec transitions to SPEC_APPROVED.
func ApproveSpec(state model.StateFile) (model.StateFile, error) {
	if err := AssertTransition(state.Phase, model.PhaseSpecApproved); err != nil {
		return state, err
	}
	state.Phase = model.PhaseSpecApproved
	state.SpecState.Status = "approved"
	return state, nil
}

// StartExecution transitions to EXECUTING.
func StartExecution(state model.StateFile) (model.StateFile, error) {
	if err := AssertTransition(state.Phase, model.PhaseExecuting); err != nil {
		return state, err
	}

	state.Phase = model.PhaseExecuting
	// Preserve discovery answers in state for revisit support.
	state.Discovery.Completed = true
	state.Discovery.Approved = false
	state.Execution = freshExecution()
	return state, nil
}

// BlockExecution transitions to BLOCKED.
func BlockExecution(state model.StateFile, reason string) (model.StateFile, error) {
	if err := AssertTransition(state.Phase, model.PhaseBlocked); err != nil {
		return state, err
	}
	progress := "BLOCKED: " + reason
	state.Phase = model.PhaseBlocked
	state.Execution.LastProgress = &progress
	return state, nil
}

// CompleteSpec transitions to COMPLETED.
func CompleteSpec(state model.StateFile, reason model.CompletionReason, note *string) (model.StateFile, error) {
	if err := AssertTransition(state.Phase, model.PhaseCompleted); err != nil {
		return state, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	state.Phase = model.PhaseCompleted
	state.CompletionReason = &reason
	state.CompletedAt = &now
	state.CompletionNote = note
	return state, nil
}

// ReopenSpec transitions from COMPLETED back to DISCOVERY.
func ReopenSpec(state model.StateFile) (model.StateFile, error) {
	if state.Phase != model.PhaseCompleted {
		return state, fmt.Errorf("cannot reopen in phase: %s", state.Phase)
	}

	var reopenedFrom *string
	if state.CompletionReason != nil {
		s := string(*state.CompletionReason)
		reopenedFrom = &s
	}

	state.Phase = model.PhaseDiscovery
	state.ReopenedFrom = reopenedFrom
	state.CompletionReason = nil
	state.CompletedAt = nil
	state.CompletionNote = nil
	state.Discovery.Completed = false
	state.Discovery.CurrentQuestion = 0
	state.Execution = freshExecution()
	state.Classification = nil
	return state, nil
}

// ResumeCompletedSpec restores the most recent execution phase from COMPLETED
// without wiping execution progress. When no prior execution transition is
// available, it falls back to EXECUTING.
func ResumeCompletedSpec(state model.StateFile) (model.StateFile, error) {
	if state.Phase != model.PhaseCompleted {
		return state, fmt.Errorf("cannot resume in phase: %s", state.Phase)
	}

	restorePhase := model.PhaseExecuting
	for i := len(state.TransitionHistory) - 1; i >= 0; i-- {
		tr := state.TransitionHistory[i]
		if tr.To != model.PhaseCompleted {
			continue
		}
		if tr.From == model.PhaseExecuting || tr.From == model.PhaseBlocked {
			restorePhase = tr.From
		}
		break
	}
	if err := AssertTransition(state.Phase, restorePhase); err != nil {
		return state, err
	}

	var reopenedFrom *string
	if state.CompletionReason != nil {
		s := string(*state.CompletionReason)
		reopenedFrom = &s
	}

	state.Phase = restorePhase
	state.ReopenedFrom = reopenedFrom
	state.CompletionReason = nil
	state.CompletedAt = nil
	state.CompletionNote = nil
	return state, nil
}

// RevisitSpec goes back from EXECUTING/BLOCKED to DISCOVERY while preserving progress.
func RevisitSpec(state model.StateFile, reason string) (model.StateFile, error) {
	if state.Phase != model.PhaseExecuting && state.Phase != model.PhaseBlocked {
		return state, fmt.Errorf("cannot revisit in phase: %s. Only EXECUTING or BLOCKED can revisit", state.Phase)
	}

	completedTasks := make([]string, len(state.Execution.CompletedTasks))
	copy(completedTasks, state.Execution.CompletedTasks)

	entry := model.RevisitEntry{
		From:           state.Phase,
		Reason:         reason,
		CompletedTasks: completedTasks,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	}

	if state.RevisitHistory == nil {
		state.RevisitHistory = []model.RevisitEntry{}
	}

	state.Phase = model.PhaseDiscovery
	state.Discovery.Completed = false
	state.Discovery.CurrentQuestion = 0
	state.Discovery.Approved = false
	state.Execution = freshExecution()
	state.Classification = nil
	state.RevisitHistory = append(state.RevisitHistory, entry)
	return state, nil
}

// ResetToIdle resets state to IDLE. Only allowed from terminal/safe phases.
func ResetToIdle(state model.StateFile) (model.StateFile, error) {
	allowed := map[model.Phase]bool{
		model.PhaseIdle:      true,
		model.PhaseExecuting: true,
		model.PhaseBlocked:   true,
		model.PhaseCompleted: true,
	}
	if !allowed[state.Phase] {
		return state, fmt.Errorf("cannot reset from %s. Use `cancel` or `wontfix` instead", state.Phase)
	}

	state.Phase = model.PhaseIdle
	state.Spec = nil
	state.Branch = nil
	state.Discovery = freshDiscovery()
	state.SpecState = model.SpecState{Path: nil, Status: "none"}
	state.Execution = freshExecution()
	state.Decisions = []model.Decision{}
	state.Classification = nil
	state.CompletionReason = nil
	state.CompletedAt = nil
	state.CompletionNote = nil
	state.ReopenedFrom = nil
	return state, nil
}
