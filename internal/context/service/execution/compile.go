// Package execution compiles the EXECUTING phase output — task selection,
// edge-case filtering, tier1 reminders, status report request, design
// checklist, promote prompt, pre-execution review, debt carry-forward,
// tension gate, and restart recommendation.
package execution

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/context/service/concerns"
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
)

// Renderer is the minimal command-builder interface required by this package.
type Renderer interface {
	C(sub string) string
	CS(sub string, specName *string) string
}

// Compile renders the EXECUTING phase. Dispatches across the
// awaiting-status-report branch and the normal-execution branch.
func Compile(
	r Renderer,
	st state.StateFile,
	activeConcerns []state.ConcernDefinition,
	rules []string,
	maxIterationsBeforeRestart int,
	parsedSpec *spec.ParsedSpec,
) model.ExecutionOutput {
	edgeCases := spec.DeriveEdgeCases(st.Discovery.Answers, st.Discovery.Premises)
	tensions := concerns.DetectTensions(activeConcerns)
	shouldRestart := st.Execution.Iteration >= maxIterationsBeforeRestart
	verifyFailed := st.Execution.LastVerification != nil && !st.Execution.LastVerification.Passed
	verifyOutputStr := ""
	if st.Execution.LastVerification != nil {
		verifyOutputStr = st.Execution.LastVerification.Output
	}

	var specTasks []spec.ParsedTask
	if parsedSpec != nil {
		specTasks = parsedSpec.Tasks
	}
	completedIDs := st.Execution.CompletedTasks
	completedSet := make(map[string]bool, len(completedIDs))
	for _, id := range completedIDs {
		completedSet[id] = true
	}

	var nextTask *spec.ParsedTask
	for i := range specTasks {
		if !completedSet[specTasks[i].ID] {
			nextTask = &specTasks[i]
			break
		}
	}

	var taskBlock *model.TaskBlock
	if nextTask != nil {
		tb := model.TaskBlock{
			ID:             nextTask.ID,
			Title:          nextTask.Title,
			TotalTasks:     len(specTasks),
			CompletedTasks: len(completedIDs),
		}
		if len(nextTask.Files) > 0 {
			tb.Files = nextTask.Files
		}
		taskBlock = &tb
	}

	taskEdgeCases := edgeCases
	if nextTask != nil && len(nextTask.Covers) > 0 {
		coverSet := make(map[string]bool, len(nextTask.Covers))
		for _, c := range nextTask.Covers {
			coverSet[strings.ToUpper(c)] = true
		}
		var filtered []string
		for i, ec := range edgeCases {
			ecID := fmt.Sprintf("EC-%d", i+1)
			if coverSet[ecID] {
				filtered = append(filtered, ec)
			}
		}
		taskEdgeCases = filtered
	}

	tier1Reminders, _ := concerns.SplitRemindersByTier(activeConcerns)

	if st.Execution.AwaitingStatusReport {
		return compileStatusReport(
			r, st, activeConcerns, rules, verifyFailed, verifyOutputStr,
			parsedSpec, taskEdgeCases, tier1Reminders,
		)
	}

	return compileNormal(
		r, st, rules, shouldRestart, verifyFailed, verifyOutputStr,
		taskBlock, edgeCases, taskEdgeCases, tier1Reminders, tensions, activeConcerns,
	)
}

func compileStatusReport(
	r Renderer,
	st state.StateFile,
	activeConcerns []state.ConcernDefinition,
	rules []string,
	verifyFailed bool,
	verifyOutputStr string,
	parsedSpec *spec.ParsedSpec,
	taskEdgeCases []string,
	tier1Reminders []string,
) model.ExecutionOutput {
	var naItems []string
	if st.Execution.NaItems != nil {
		naItems = st.Execution.NaItems
	}
	criteria := buildAcceptanceCriteria(
		activeConcerns, verifyFailed, verifyOutputStr,
		st.Execution.Debt, st.Classification,
		parsedSpec, naItems,
	)

	// Detect batch task claims.
	var batchTaskIDs []string
	if st.Execution.LastProgress != nil {
		var prevAnswer map[string]any
		if err := json.Unmarshal([]byte(*st.Execution.LastProgress), &prevAnswer); err == nil {
			if completed, ok := prevAnswer["completed"].([]any); ok {
				for _, id := range completed {
					if s, ok := id.(string); ok && strings.HasPrefix(s, "task-") {
						batchTaskIDs = append(batchTaskIDs, s)
					}
				}
			}
		}
	}

	batchInstruction := model.ExecutionStatusReportInstruction
	if len(batchTaskIDs) >= 2 {
		batchInstruction = fmt.Sprintf("%d tasks reported complete. Report status against ALL relevant acceptance criteria.", len(batchTaskIDs))
	}

	statusTrue := true
	out := model.ExecutionOutput{
		Phase:       "EXECUTING",
		Instruction: batchInstruction,
		EdgeCases:   taskEdgeCases,
		Context: model.ContextBlock{
			Rules:            rules,
			ConcernReminders: tier1Reminders,
		},
		Transition: model.TransitionExecution{
			OnComplete: r.CS("next --answer='{\"completed\":[...],\"remaining\":[...],\"blocked\":[]}'", st.Spec),
			OnBlocked:  r.CS("block \"reason\"", st.Spec),
			Iteration:  st.Execution.Iteration,
		},
		StatusReportRequired: &statusTrue,
		StatusReport: &model.StatusReportRequest{
			Criteria:     criteria,
			ReportFormat: model.DefaultReportFormat,
		},
	}
	if len(batchTaskIDs) >= 2 {
		out.BatchTasks = batchTaskIDs
	}
	if verifyFailed {
		trueVal := true
		truncated := verifyOutputStr
		if len(truncated) > model.VerificationOutputTruncateFull {
			truncated = truncated[:model.VerificationOutputTruncateFull]
		}
		out.VerificationFailed = &trueVal
		out.VerificationOutput = &truncated
	}
	return out
}

func compileNormal(
	r Renderer,
	st state.StateFile,
	rules []string,
	shouldRestart, verifyFailed bool,
	verifyOutputStr string,
	taskBlock *model.TaskBlock,
	edgeCases, taskEdgeCases, tier1Reminders []string,
	tensions []model.ConcernTension,
	activeConcerns []state.ConcernDefinition,
) model.ExecutionOutput {
	wasRejected := st.Execution.LastProgress != nil && strings.Contains(*st.Execution.LastProgress, "Task not accepted")
	var debtItems []state.DebtItem
	debtUnaddressed := 0
	if st.Execution.Debt != nil {
		debtItems = st.Execution.Debt.Items
		debtUnaddressed = st.Execution.Debt.UnaddressedIterations
	}

	taskInstruction := fmt.Sprintf("All tasks completed. Run `%s` to finish.", r.CS("done", st.Spec))
	if taskBlock != nil {
		taskInstruction = fmt.Sprintf("Execute task %s: %s (%d/%d completed)",
			taskBlock.ID, taskBlock.Title, taskBlock.CompletedTasks, taskBlock.TotalTasks)
	}

	var baseInstruction string
	if verifyFailed {
		baseInstruction = model.ExecutionVerificationFailed
	} else if wasRejected && len(debtItems) > 0 {
		urgencySuffix := ""
		if debtUnaddressed >= model.DebtUrgentThreshold {
			urgencySuffix = fmt.Sprintf(" These items have been outstanding for %d iterations.", debtUnaddressed)
		}
		baseInstruction = fmt.Sprintf("Task not accepted — %d remaining item(s) must be addressed before this task can be completed.%s Address them, then submit a new status report.", len(debtItems), urgencySuffix)
	} else {
		baseInstruction = taskInstruction
	}

	if phaseHint := tddPhaseInstruction(st.Execution.TDDCycle); phaseHint != "" {
		baseInstruction = phaseHint + "\n\n" + baseInstruction
	}

	out := model.ExecutionOutput{
		Phase:       "EXECUTING",
		Instruction: baseInstruction,
		Task:        taskBlock,
		EdgeCases:   taskEdgeCases,
		Context: model.ContextBlock{
			Rules:            rules,
			ConcernReminders: tier1Reminders,
		},
		Transition: model.TransitionExecution{
			OnComplete: r.CS("next --answer=\"...\"", st.Spec),
			OnBlocked:  r.CS("block \"reason\"", st.Spec),
			Iteration:  st.Execution.Iteration,
		},
	}
	if len(edgeCases) > 0 {
		out.Instruction += " Use the listed edge cases to drive test-writer coverage before implementation."
	}

	if wasRejected && len(debtItems) > 0 {
		trueVal := true
		reason := fmt.Sprintf("%d remaining item(s) must be addressed.", len(debtItems))
		remaining := make([]string, len(debtItems))
		for i, d := range debtItems {
			remaining[i] = d.Text
		}
		out.TaskRejected = &trueVal
		out.RejectionReason = &reason
		out.RejectionRemaining = remaining
	}

	if st.Execution.Debt != nil && len(st.Execution.Debt.Items) > 0 {
		unaddressed := st.Execution.Debt.UnaddressedIterations
		debtNote := "These were not completed in a previous iteration. Address them BEFORE starting new work."
		if unaddressed >= model.DebtUrgentThreshold {
			debtNote = fmt.Sprintf("URGENT: These items have been unaddressed for %d iterations. Address them IMMEDIATELY before any new work.", unaddressed)
		}
		out.PreviousIterationDebt = &model.DebtCarryForward{
			FromIteration: st.Execution.Debt.FromIteration,
			Items:         st.Execution.Debt.Items,
			Note:          debtNote,
		}
	}

	if verifyFailed {
		trueVal := true
		truncated := verifyOutputStr
		if len(truncated) > model.VerificationOutputTruncateFull {
			truncated = truncated[:model.VerificationOutputTruncateFull]
		}
		out.VerificationFailed = &trueVal
		out.VerificationOutput = &truncated
	}

	if len(tensions) > 0 {
		var tensionParts []string
		for _, t := range tensions {
			tensionParts = append(tensionParts, strings.Join(t.Between, " vs ")+": "+t.Issue)
		}
		out.ConcernTensions = tensions
		out.Instruction = fmt.Sprintf("TENSION GATE: %d concern tension(s) detected: %s. You MUST present these to the user and get explicit resolution for each before proceeding. Use AskUserQuestion to ask which side to prioritize.", len(tensions), strings.Join(tensionParts, "; "))
	}

	if shouldRestart {
		trueVal := true
		name := ""
		if st.Spec != nil {
			name = *st.Spec
		}
		restartInstr := fmt.Sprintf("Context may be getting large after %d iterations. Consider starting a new conversation and running `%s` to resume - your progress is saved.", st.Execution.Iteration, r.CS("next", &name))
		out.RestartRecommended = &trueVal
		out.RestartInstruction = &restartInstr
	}

	for i := len(st.Decisions) - 1; i >= 0; i-- {
		d := st.Decisions[i]
		if !d.Promoted {
			lastProgressStr := ""
			if st.Execution.LastProgress != nil {
				lastProgressStr = *st.Execution.LastProgress
			}
			if strings.HasPrefix(lastProgressStr, "Resolved:") {
				out.PromotePrompt = &model.PromotePrompt{
					DecisionID: d.ID,
					Question:   d.Question,
					Choice:     d.Choice,
					Prompt: fmt.Sprintf("You just resolved a decision: \"%s\". Ask the user: \"Should this be a permanent rule for future specs too?\" If yes, run: `%s`",
						d.Choice, r.C(fmt.Sprintf("rule add \"%s\"", d.Choice))),
				}
			}
			break
		}
	}

	if st.Execution.Iteration == 0 && !verifyFailed && !wasRejected {
		out.PreExecutionReview = &model.PreExecutionReview{
			Instruction: model.ExecutionPreReviewInstruction,
		}
	}

	for _, cc := range activeConcerns {
		if cc.ID == "beautiful-product" {
			out.DesignChecklist = &model.DesignChecklist{
				Required:    true,
				Instruction: model.DesignChecklistInstruction,
				Dimensions:  model.DesignChecklistDimensions,
			}
			break
		}
	}

	return out
}

// tddPhaseInstruction returns the per-phase reminder for the active TDD cycle,
// or "" when no TDD phase is set. Prepended to the EXECUTING instruction so
// the orchestrator gets a fresh reminder of which sub-agent to spawn.
func tddPhaseInstruction(cycle string) string {
	switch cycle {
	case state.TDDCycleRed:
		return model.TDDPhaseRedInstruction
	case state.TDDCycleGreen:
		return model.TDDPhaseGreenInstruction
	case state.TDDCycleRefactor:
		return model.TDDPhaseRefactorInstruction
	default:
		return ""
	}
}
