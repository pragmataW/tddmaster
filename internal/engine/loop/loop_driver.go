package loop

import (
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func writeSpecMd(c *engine.Context) {
	if err := c.WriteSpecMd(spec.RenderSpecMd(c.Slug(), c.State(), c.Progress())); err != nil {
		log.Printf("tddmaster: failed to write spec.md for %q: %v", c.Slug(), err)
	}
}

const defaultMaxRefactorRounds = 3

type LoopDriver struct {
	ruleset *RuleSet
}

func NewLoopDriver() *LoopDriver {
	return &LoopDriver{ruleset: newRuleSet()}
}

func allTasksDone(tasks []spec.Task) bool {
	for _, t := range tasks {
		if !t.Done {
			return false
		}
	}
	return true
}

func canTerminate(pr spec.Progress) bool {
	return len(pr.Tasks) == 0 || allTasksDone(pr.Tasks)
}

func worktreeHint(slug, taskID string) spec.WorktreeRef {
	return spec.WorktreeRef{
		Path:   ".tddmaster/worktrees/" + slug + "/" + taskID,
		Branch: "tddmaster/" + slug + "/" + taskID,
	}
}

func seedTaskExec(c *engine.Context, task *spec.Task) bool {
	if task.Exec != nil {
		return false
	}
	st := reseedCycle(spec.ExecState{}, c.Settings().TDDEnabled && task.TDDEnabled)
	wt := worktreeHint(c.Slug(), task.ID)
	st.Worktree = &wt
	task.Exec = &st
	return true
}

func (d *LoopDriver) buildExecCtx(c *engine.Context, task spec.Task, taskIdx int) ExecCtx {
	return ExecCtx{
		Settings:          c.Settings(),
		Task:              task,
		State:             *task.Exec,
		TaskIdx:           taskIdx,
		MaxRefactorRounds: defaultMaxRefactorRounds,
		UserContext:       c.AnswerValue("listen_context"),
		Rules:             c.Rules(),
	}
}

func worktreeInstructionBlock(w *spec.WorktreeRef) string {
	if w == nil {
		return ""
	}
	return fmt.Sprintf(promptregistry.WorktreeBlockFmt, w.Path, w.Branch)
}

func injectTaskIDIntoExample(in engine.ExpectedInput, taskID string) engine.ExpectedInput {
	if idx := strings.Index(in.Example, "{"); idx >= 0 {
		in.Example = in.Example[:idx+1] + `"taskId":"` + taskID + `",` + in.Example[idx+1:]
	}
	return in
}

func deadlockAction(tasks []spec.Task) engine.Action {
	blocked := spec.BlockedSet(tasks)
	done := make(map[string]bool, len(tasks))
	for _, t := range tasks {
		if t.Done {
			done[t.ID] = true
		}
	}
	var b strings.Builder
	b.WriteString("Deadlock detected: no ready task remains.")
	var blockedLines, waitingLines []string
	for _, t := range tasks {
		if t.Done {
			continue
		}
		pending := make([]string, 0, len(t.DependsOn))
		for _, dep := range t.DependsOn {
			if !done[dep] {
				pending = append(pending, dep)
			}
		}
		switch {
		case t.Blocked:
			reason := t.BlockedReason
			if reason == "" {
				reason = "blocked"
			}
			blockedLines = append(blockedLines, t.ID+": "+reason)
		case blocked[t.ID]:
			waitingLines = append(waitingLines, t.ID+": waiting on blocked dependency ("+strings.Join(pending, ", ")+")")
		default:
			waitingLines = append(waitingLines, t.ID+": waiting on dependencies ("+strings.Join(pending, ", ")+")")
		}
	}
	if len(blockedLines) > 0 {
		b.WriteString("\nBlocked tasks:\n")
		for _, line := range blockedLines {
			b.WriteString("- ")
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	if len(waitingLines) > 0 {
		b.WriteString("\nWaiting tasks:\n")
		for _, line := range waitingLines {
			b.WriteString("- ")
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return engine.Action{Action: engine.ActionError, Instruction: b.String()}
}

func batchSummary(entries []string) string {
	return fmt.Sprintf(
		"%d task(s) ready for parallel execution: %s. Run each in its own worktree via a separate sub-agent. "+
			"Submit one report per task; every report MUST include taskId. "+
			"Only spawn sub-agents for tasks without an in-flight report.",
		len(entries), strings.Join(entries, ", "))
}

func (d *LoopDriver) Next(c *engine.Context, ph *engine.PhaseDef) (engine.Action, bool) {
	pr := c.Progress()

	if canTerminate(pr) {
		pr.Status = spec.StatusCompleted
		if err := c.SaveProgress(pr); err != nil {
			return engine.Action{Action: engine.ActionError, Instruction: err.Error()}, false
		}
		return engine.Action{}, true
	}

	if iterationLimitReached(c, &pr) {
		if err := c.SaveProgress(pr); err != nil {
			return engine.Action{Action: engine.ActionError, Instruction: err.Error()}, false
		}
		return engine.Action{Action: engine.ActionNotify, Instruction: promptregistry.RestartRecommendedText}, false
	}

	ready := spec.ReadyTaskIndices(pr.Tasks)
	if len(ready) == 0 {
		return deadlockAction(pr.Tasks), false
	}

	seeded := false
	for _, i := range ready {
		if seedTaskExec(c, &pr.Tasks[i]) {
			seeded = true
		}
	}
	if seeded {
		if err := c.SaveProgress(pr); err != nil {
			return engine.Action{Action: engine.ActionError, Instruction: err.Error()}, false
		}
	}

	var gateAction *engine.Action
	var gateTask engine.TaskAction
	var taskActions []engine.TaskAction
	var summary []string
	var stuck []string

	for _, i := range ready {
		ctx := d.buildExecCtx(c, pr.Tasks[i], i)
		stage, ok := d.ruleset.Next(ctx)
		if !ok {
			stuck = append(stuck, fmt.Sprintf("%s (exec: %+v)", pr.Tasks[i].ID, *pr.Tasks[i].Exec))
			continue
		}
		stageAction := stage.Prompt(ctx)
		if stage.ID() == StageIDGate {
			if gateAction == nil {
				gateTask = taskActionFor(pr.Tasks[i], stage.ID(), stageAction, false)
				gateAction = &stageAction
			}
			continue
		}
		taskActions = append(taskActions, taskActionFor(pr.Tasks[i], stage.ID(), stageAction, true))
		summary = append(summary, pr.Tasks[i].ID+" ("+stage.ID()+")")
	}

	if len(stuck) > 0 {
		return engine.Action{
			Action:      engine.ActionError,
			Instruction: "no applicable stage for ready tasks: " + strings.Join(stuck, "; "),
		}, false
	}

	if gateAction != nil {
		action := *gateAction
		action.Tasks = append([]engine.TaskAction{gateTask}, taskActions...)
		action.ExpectedInput = gateTask.ExpectedInput
		if len(taskActions) > 0 {
			action.Instruction += promptregistry.ParallelDispatchDirective
		}
		action.Instruction += promptregistry.GateReappearDirective
		return action, false
	}

	if len(taskActions) == 0 {
		return engine.Action{
			Action:      engine.ActionError,
			Instruction: "no applicable stage for ready tasks",
		}, false
	}

	return engine.Action{
		Action:        engine.ActionInstruct,
		Instruction:   batchSummary(summary),
		ExpectedInput: engine.ExpectedInput{Format: engine.FormatJSON},
		Tasks:         taskActions,
	}, false
}

func taskActionFor(task spec.Task, stageID string, stageAction engine.Action, withWorktree bool) engine.TaskAction {
	ta := engine.TaskAction{
		TaskID:        task.ID,
		Stage:         stageID,
		Instruction:   stageAction.Instruction,
		DelegateAgent: stageAction.DelegateAgent,
		ExpectedInput: injectTaskIDIntoExample(stageAction.ExpectedInput, task.ID),
	}
	if withWorktree {
		ta.Instruction = worktreeInstructionBlock(task.Exec.Worktree) + ta.Instruction
		ta.Worktree = task.Exec.Worktree
	}
	return ta
}

func routeReportTask(pr spec.Progress, taskID string, ready []int) (int, error) {
	readyIDs := make([]string, 0, len(ready))
	for _, i := range ready {
		readyIDs = append(readyIDs, pr.Tasks[i].ID)
	}
	readyList := strings.Join(readyIDs, ", ")

	if taskID == "" {
		return -1, errs.Newf(errs.KeyReportMissingTaskID, readyList)
	}
	for _, i := range ready {
		if pr.Tasks[i].ID == taskID {
			return i, nil
		}
	}
	for i, t := range pr.Tasks {
		if t.ID != taskID {
			continue
		}
		if t.Done {
			return -1, errs.Newf(errs.KeyTaskAlreadyDone, taskID, readyList)
		}
		if t.Blocked {
			return i, nil
		}
		return -1, errs.Newf(errs.KeyTaskNotReady, taskID, readyList)
	}
	return -1, errs.Newf(errs.KeyUnknownTaskIDReady, taskID, readyList)
}

func (d *LoopDriver) Submit(c *engine.Context, ph *engine.PhaseDef, answer []byte) (engine.Action, bool, error) {
	if strings.TrimSpace(string(answer)) == "continue" {
		pr := c.Progress()
		pr.Iterations = 0
		if err := c.SaveProgress(pr); err != nil {
			return engine.Action{}, false, err
		}
		return engine.Action{}, false, nil
	}

	if len(answer) == 0 || !json.Valid(answer) {
		return engine.Action{}, false, errs.New(errs.KeyInvalidJSONAnswer)
	}

	var report StageReport
	if err := json.Unmarshal(answer, &report); err != nil {
		return engine.Action{}, false, err
	}

	pr := c.Progress()

	if canTerminate(pr) {
		return engine.Action{}, true, nil
	}

	ready := spec.ReadyTaskIndices(pr.Tasks)

	taskIdx, err := routeReportTask(pr, report.TaskID, ready)
	if err != nil {
		return engine.Action{}, false, err
	}

	seedTaskExec(c, &pr.Tasks[taskIdx])

	blocked := len(report.Blocked) > 0
	if pr.Tasks[taskIdx].Blocked && !blocked {
		pr.Tasks[taskIdx].Blocked = false
		pr.Tasks[taskIdx].BlockedReason = ""
		if !report.HasStageResult() {
			return d.finishSubmit(c, pr)
		}
	}
	if blocked {
		pr.Tasks[taskIdx].Blocked = true
		pr.Tasks[taskIdx].BlockedReason = strings.Join(report.Blocked, "; ")
	}
	task := pr.Tasks[taskIdx]

	ctx := d.buildExecCtx(c, task, taskIdx)

	stage, ok := d.ruleset.Next(ctx)
	if !ok {
		return engine.Action{}, false, errs.Newf(errs.KeyNoApplicableStage, task.ID, ctx.State)
	}

	if report.HasGateAnswer() && stage.ID() != StageIDGate {
		return d.finishSubmit(c, pr)
	}

	if stage.ID() == StageIDRed && (!blocked || len(report.Traceability) > 0) {
		if err := validateAndPersistTraceability(c, task, report); err != nil {
			return engine.Action{}, false, err
		}
	}

	if stage.ID() == StageIDVerifier && tddActive(ctx) && ctx.State.TDDCycle == cycleGreen && len(report.FileCoverage) > 0 {
		if err := persistCoverage(c, task.ID, report); err != nil {
			return engine.Action{}, false, err
		}
	}

	newCtx := ctx
	if !blocked || (stage.ID() == StageIDGate && report.HasGateAnswer()) {
		var err error
		newCtx, err = stage.OnReport(ctx, report)
		if err != nil {
			return engine.Action{}, false, err
		}
	}

	if reportFromVerifier(stage.ID(), ctx.State) && (!blocked || len(report.FailedACs) > 0 || len(report.UncoveredEdgeCases) > 0) {
		if report.EffectivePassed() {
			newCtx.State.LastFailedACs = nil
			newCtx.State.LastUncoveredEC = nil
		} else {
			newCtx.State.LastFailedACs = report.FailedACs
			newCtx.State.LastUncoveredEC = report.UncoveredEdgeCases
		}
	}

	newState := newCtx.State
	pr.Tasks[taskIdx].Exec = &newState

	taskComplete := !blocked && isTaskComplete(c.Settings().TDDEnabled && task.TDDEnabled, c.Settings().SkipVerifierEnabled, ctx.State, newState, report)

	pr.Tasks[taskIdx].RefactorNotes = appendUniqueNotes(pr.Tasks[taskIdx].RefactorNotes, report.RefactorNotes)
	combined := make([]string, 0, len(report.FailedACs)+len(report.UncoveredEdgeCases))
	combined = append(combined, report.FailedACs...)
	combined = append(combined, report.UncoveredEdgeCases...)
	pr.Tasks[taskIdx].FailedACReasons = appendUniqueStrings(pr.Tasks[taskIdx].FailedACReasons, combined)

	if taskComplete {
		pr.Tasks = completeCurrentTask(pr.Tasks, taskIdx)
	}

	return d.finishSubmit(c, pr)
}

func (d *LoopDriver) finishSubmit(c *engine.Context, pr spec.Progress) (engine.Action, bool, error) {
	pr.Iterations++

	done := allTasksDone(pr.Tasks)
	if done {
		pr.Status = spec.StatusCompleted
	}
	limitHit := !done && iterationLimitReached(c, &pr)

	if err := c.SaveProgress(pr); err != nil {
		return engine.Action{}, false, err
	}

	writeSpecMd(c)

	if done {
		return engine.Action{}, true, nil
	}

	if limitHit {
		return engine.Action{Action: engine.ActionNotify, Instruction: promptregistry.RestartRecommendedText}, false, nil
	}

	return engine.Action{}, false, nil
}

func iterationLimitReached(c *engine.Context, pr *spec.Progress) bool {
	if pr.Iterations < c.MaxIteration() {
		return false
	}
	pr.Iterations = 0
	return true
}

func appendUnique[T any](dst, src []T, eq func(a, b T) bool) []T {
	for _, item := range src {
		if !slices.ContainsFunc(dst, func(existing T) bool { return eq(existing, item) }) {
			dst = append(dst, item)
		}
	}
	return dst
}

func appendUniqueNotes(dst []RefactorNote, src []RefactorNote) []RefactorNote {
	return appendUnique(dst, src, func(a, b RefactorNote) bool {
		return a.File == b.File && a.Suggestion == b.Suggestion && a.Rationale == b.Rationale
	})
}

func appendUniqueStrings(dst []string, src []string) []string {
	for _, s := range src {
		if !slices.Contains(dst, s) {
			dst = append(dst, s)
		}
	}
	return dst
}

func reportFromVerifier(stageID string, st spec.ExecState) bool {
	if stageID == StageIDVerifier {
		return true
	}
	return stageID == StageIDRefactor && st.RefactorApplied
}

func isTaskComplete(tddActive, skipVerifier bool, oldState, newState spec.ExecState, report StageReport) bool {
	if tddActive {
		return oldState.TDDCycle != cycleEmpty && newState.TDDCycle == cycleEmpty
	}
	if skipVerifier {
		return report.EffectivePassed() || len(report.Completed) > 0
	}
	return oldState.Implemented && report.EffectivePassed()
}
