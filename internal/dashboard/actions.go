
// Dashboard actions — thin wrappers around state machine that add event logging.

package dashboard

import (
	"fmt"
	"log"
	"time"

	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
)

// =============================================================================
// Result type
// =============================================================================

// ActionResult is returned by dashboard actions.
type ActionResult struct {
	OK    bool
	Error string
}

// =============================================================================
// Helpers
// =============================================================================

// resolveUser resolves a User for actions. Falls back to git/config if user is nil.
func resolveUser(root string, user *User) (state.User, error) {
	if user != nil {
		return state.User{Name: user.Name, Email: user.Email}, nil
	}
	return state.ResolveUser(root)
}

// =============================================================================
// Actions
// =============================================================================

// Approve transitions SPEC_PROPOSAL → SPEC_APPROVED.
func Approve(root, specName string, user *User) ActionResult {
	s, err := state.ResolveState(root, &specName)
	if err != nil {
		return ActionResult{OK: false, Error: err.Error()}
	}

	if s.Phase != state.PhaseSpecProposal {
		return ActionResult{OK: false, Error: fmt.Sprintf("Cannot approve in phase: %s", s.Phase)}
	}

	resolved, err := resolveUser(root, user)
	if err != nil {
		return ActionResult{OK: false, Error: err.Error()}
	}

	s, err = state.ApproveSpec(s)
	if err != nil {
		return ActionResult{OK: false, Error: err.Error()}
	}
	ui := &state.UserInfo{Name: resolved.Name, Email: resolved.Email}
	s = state.RecordTransition(s, state.PhaseSpecProposal, state.PhaseSpecApproved, ui, nil)

	if err := state.WriteSpecState(root, specName, s); err != nil {
		return ActionResult{OK: false, Error: err.Error()}
	}

	_ = AppendEvent(root, DashboardEvent{
		Ts:   time.Now().UTC().Format(time.RFC3339),
		Type: EventTypePhaseChange,
		Spec: specName,
		User: resolved.Name,
		Extra: map[string]interface{}{
			"from": string(state.PhaseSpecProposal),
			"to":   string(state.PhaseSpecApproved),
		},
	})

	return ActionResult{OK: true}
}

// AddNote adds a note to a spec.
func AddNote(root, specName, text string, user *User) ActionResult {
	s, err := state.ResolveState(root, &specName)
	if err != nil {
		return ActionResult{OK: false, Error: err.Error()}
	}

	resolved, err := resolveUser(root, user)
	if err != nil {
		return ActionResult{OK: false, Error: err.Error()}
	}

	ui := &state.UserInfo{Name: resolved.Name, Email: resolved.Email}
	s = state.AddSpecNote(s, text, ui)

	if err := state.WriteSpecState(root, specName, s); err != nil {
		return ActionResult{OK: false, Error: err.Error()}
	}

	_ = AppendEvent(root, DashboardEvent{
		Ts:   time.Now().UTC().Format(time.RFC3339),
		Type: EventTypeNote,
		Spec: specName,
		User: resolved.Name,
		Extra: map[string]interface{}{
			"text": text,
		},
	})

	return ActionResult{OK: true}
}

// AddQuestion adds a question to a spec (stored as a [QUESTION] note).
func AddQuestion(root, specName, text string, user *User) ActionResult {
	s, err := state.ResolveState(root, &specName)
	if err != nil {
		return ActionResult{OK: false, Error: err.Error()}
	}

	resolved, err := resolveUser(root, user)
	if err != nil {
		return ActionResult{OK: false, Error: err.Error()}
	}

	ui := &state.UserInfo{Name: resolved.Name, Email: resolved.Email}
	s = state.AddSpecNote(s, "[QUESTION] "+text, ui)

	if err := state.WriteSpecState(root, specName, s); err != nil {
		return ActionResult{OK: false, Error: err.Error()}
	}

	mentionID := fmt.Sprintf("mention-%d", time.Now().UnixMilli())
	_ = AppendEvent(root, DashboardEvent{
		Ts:   time.Now().UTC().Format(time.RFC3339),
		Type: EventTypeMention,
		Spec: specName,
		User: resolved.Name,
		Extra: map[string]interface{}{
			"from":     resolved.Name,
			"to":       "",
			"question": text,
			"id":       mentionID,
		},
	})

	return ActionResult{OK: true}
}

// Signoff signs off on a spec.
func Signoff(root, specName string, user *User) ActionResult {
	resolved, err := resolveUser(root, user)
	if err != nil {
		return ActionResult{OK: false, Error: err.Error()}
	}

	_ = AppendEvent(root, DashboardEvent{
		Ts:   time.Now().UTC().Format(time.RFC3339),
		Type: EventTypeSignoff,
		Spec: specName,
		User: resolved.Name,
		Extra: map[string]interface{}{
			"role":   "reviewer",
			"status": "signed",
		},
	})

	return ActionResult{OK: true}
}

// ReplyMention replies to a mention.
func ReplyMention(root, specName, mentionID, text string, user *User) ActionResult {
	s, err := state.ResolveState(root, &specName)
	if err != nil {
		return ActionResult{OK: false, Error: err.Error()}
	}

	resolved, err := resolveUser(root, user)
	if err != nil {
		return ActionResult{OK: false, Error: err.Error()}
	}

	ui := &state.UserInfo{Name: resolved.Name, Email: resolved.Email}
	s = state.AddSpecNote(s, fmt.Sprintf("[REPLY:%s] %s", mentionID, text), ui)

	if err := state.WriteSpecState(root, specName, s); err != nil {
		return ActionResult{OK: false, Error: err.Error()}
	}

	_ = AppendEvent(root, DashboardEvent{
		Ts:   time.Now().UTC().Format(time.RFC3339),
		Type: EventTypeMentionReply,
		Spec: specName,
		User: resolved.Name,
		Extra: map[string]interface{}{
			"mentionId": mentionID,
			"text":      text,
		},
	})

	return ActionResult{OK: true}
}

// Complete marks a spec as complete.
func Complete(root, specName string, user *User) ActionResult {
	s, err := state.ResolveState(root, &specName)
	if err != nil {
		return ActionResult{OK: false, Error: err.Error()}
	}

	if s.Phase != state.PhaseExecuting {
		return ActionResult{OK: false, Error: fmt.Sprintf("Cannot complete in phase: %s", s.Phase)}
	}

	resolved, err := resolveUser(root, user)
	if err != nil {
		return ActionResult{OK: false, Error: err.Error()}
	}

	reason := state.CompletionReasonDone
	completedState, err := state.CompleteSpec(s, reason, nil)
	if err != nil {
		return ActionResult{OK: false, Error: err.Error()}
	}

	ui := &state.UserInfo{Name: resolved.Name, Email: resolved.Email}
	completedState = state.RecordTransition(completedState, state.PhaseExecuting, state.PhaseCompleted, ui, nil)

	// Per-spec: COMPLETED
	if err := state.WriteSpecState(root, specName, completedState); err != nil {
		return ActionResult{OK: false, Error: err.Error()}
	}

	// Global: return to IDLE (best effort)
	idleState := completedState
	idleState.Phase = state.PhaseIdle
	idleState.Spec = nil
	_ = state.WriteState(root, idleState)

	// Update spec.md / progress.json (best effort — state is already committed).
	if err := spec.UpdateSpecStatus(root, specName, "completed"); err != nil {
		log.Printf("dashboard: warning: spec.md status update failed: %v", err)
	}
	if err := spec.UpdateProgressStatus(root, specName, "completed"); err != nil {
		log.Printf("dashboard: warning: progress.json status update failed: %v", err)
	}

	_ = AppendEvent(root, DashboardEvent{
		Ts:   time.Now().UTC().Format(time.RFC3339),
		Type: EventTypePhaseChange,
		Spec: specName,
		User: resolved.Name,
		Extra: map[string]interface{}{
			"from": string(state.PhaseExecuting),
			"to":   string(state.PhaseCompleted),
		},
	})

	return ActionResult{OK: true}
}
