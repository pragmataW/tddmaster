package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pragmataW/tddmaster/internal/spec"
	specmodel "github.com/pragmataW/tddmaster/internal/spec/model"
	specservice "github.com/pragmataW/tddmaster/internal/spec/service"
	"github.com/pragmataW/tddmaster/internal/state"
)

// tryHandleImportantPlanAnswer intercepts answers shaped as gate responses
// (`{"plan": {...}, "accepted": true}` or `{"planFeedback": "<reason>"}`) and
// applies them to state. Returns handled=true when the answer matched a gate
// shape; the caller then short-circuits the normal status-report routing.
//
// Gate is opt-in. When the manifest has IsImportantTaskGateEnabled()==false
// this is a no-op.
func tryHandleImportantPlanAnswer(
	root string,
	st state.StateFile,
	config *state.NosManifest,
	answer string,
) (state.StateFile, bool, error) {
	if config == nil || !config.IsImportantTaskGateEnabled() {
		return st, false, nil
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(answer), &raw); err != nil {
		return st, false, nil
	}

	_, hasPlan := raw["plan"]
	feedback, hasFeedback := raw["planFeedback"].(string)
	if !hasPlan && !hasFeedback {
		return st, false, nil
	}

	taskID, err := resolveGateTaskID(root, st)
	if err != nil {
		return st, true, err
	}
	if taskID == "" {
		return st, true, fmt.Errorf("important plan submitted but no important task is awaiting a plan")
	}

	if hasFeedback {
		newState := applyPlanFeedback(st, taskID, feedback)
		return newState, true, nil
	}

	planRaw, ok := raw["plan"].(map[string]any)
	if !ok {
		return st, true, fmt.Errorf("plan payload must be an object with assumptions/touchedFiles/designPatterns/bestPractices/approach")
	}
	plan, err := decodePlan(planRaw)
	if err != nil {
		return st, true, err
	}
	plan.TaskID = taskID

	accepted, _ := raw["accepted"].(bool)
	if !accepted {
		// Plan supplied without explicit accept signal — treat as proposal only,
		// do not persist. Orchestrator must re-submit with accepted:true after
		// the user accepts via AskUserQuestion.
		return st, true, fmt.Errorf("plan submitted without accepted:true; resubmit with `accepted: true` after user approval, or submit planFeedback to revise")
	}

	plan.AttemptCount = currentAttemptCount(st, taskID) + 1
	plan.ApprovedAt = time.Now().UTC().Format(time.RFC3339)
	if u, _ := state.ResolveUser(root); u.Name != "" {
		plan.ApprovedBy = state.FormatUser(state.User{Name: u.Name, Email: u.Email})
	}

	if st.Spec != nil {
		if err := specservice.AppendTaskPlan(root, *st.Spec, plan); err != nil {
			return st, true, fmt.Errorf("persisting task plan: %w", err)
		}
	}

	newState := applyPlanApproval(st, taskID)
	return newState, true, nil
}

// resolveGateTaskID finds the first non-completed task in the active spec that
// is flagged Important and has no approved plan yet. Returns "" when no such
// task exists (e.g. user submitted a plan but the gate is no longer relevant).
func resolveGateTaskID(root string, st state.StateFile) (string, error) {
	if st.Spec == nil {
		return "", nil
	}
	parsed, err := spec.ParseSpec(root, *st.Spec)
	if err != nil || parsed == nil {
		return "", nil
	}
	completed := make(map[string]bool, len(st.Execution.CompletedTasks))
	for _, id := range st.Execution.CompletedTasks {
		completed[id] = true
	}
	importants := make(map[string]bool, len(st.OverrideTasks))
	for _, t := range st.OverrideTasks {
		if t.Important != nil && *t.Important {
			importants[t.ID] = true
		}
	}
	approved := make(map[string]bool, len(st.Execution.ApprovedImportantPlans))
	for _, id := range st.Execution.ApprovedImportantPlans {
		approved[id] = true
	}
	for _, t := range parsed.Tasks {
		if completed[t.ID] {
			continue
		}
		if importants[t.ID] && !approved[t.ID] {
			return t.ID, nil
		}
		// First non-completed task that is not gating: gate is not active.
		return "", nil
	}
	return "", nil
}

func decodePlan(raw map[string]any) (specmodel.ProgressTaskPlan, error) {
	var plan specmodel.ProgressTaskPlan
	plan.Assumptions = extractStringSlice(raw["assumptions"])
	plan.TouchedFiles = extractStringSlice(raw["touchedFiles"])
	plan.DesignPatterns = extractStringSlice(raw["designPatterns"])
	plan.BestPractices = extractStringSlice(raw["bestPractices"])
	if a, ok := raw["approach"].(string); ok {
		plan.Approach = a
	}
	if plan.Approach == "" {
		return plan, fmt.Errorf("plan.approach is required and must be a non-empty string")
	}
	if len(plan.TouchedFiles) == 0 {
		return plan, fmt.Errorf("plan.touchedFiles is required and must list at least one path")
	}
	return plan, nil
}

func extractStringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

func currentAttemptCount(st state.StateFile, taskID string) int {
	if st.Execution.PendingPlanAttempts == nil {
		return 0
	}
	return st.Execution.PendingPlanAttempts[taskID]
}

func applyPlanFeedback(st state.StateFile, taskID, feedback string) state.StateFile {
	newState := st
	if newState.Execution.PendingPlanAttempts == nil {
		newState.Execution.PendingPlanAttempts = map[string]int{}
	}
	newState.Execution.PendingPlanAttempts[taskID]++
	if newState.Execution.LastPlanFeedback == nil {
		newState.Execution.LastPlanFeedback = map[string]string{}
	}
	newState.Execution.LastPlanFeedback[taskID] = feedback
	return newState
}

func applyPlanApproval(st state.StateFile, taskID string) state.StateFile {
	newState := st
	// Add to approved set if not already there.
	already := false
	for _, id := range newState.Execution.ApprovedImportantPlans {
		if id == taskID {
			already = true
			break
		}
	}
	if !already {
		newState.Execution.ApprovedImportantPlans = append(newState.Execution.ApprovedImportantPlans, taskID)
	}
	// Clear pending bookkeeping for this task.
	if newState.Execution.PendingPlanAttempts != nil {
		delete(newState.Execution.PendingPlanAttempts, taskID)
	}
	if newState.Execution.LastPlanFeedback != nil {
		delete(newState.Execution.LastPlanFeedback, taskID)
	}
	return newState
}
