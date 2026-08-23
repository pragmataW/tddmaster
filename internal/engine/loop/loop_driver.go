package loop

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/projectcmds"
	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func writeSpecMd(c *engine.Context) {
	var trace []spec.Traceability
	if tr, err := c.LoadTraceability(); err == nil {
		trace = append(trace, tr)
	} else {
		log.Printf("tddmaster: failed to load traceability for spec.md %q: %v", c.Slug(), err)
	}
	if err := c.WriteSpecMd(spec.RenderSpecMd(c.Slug(), c.State(), c.Progress(), trace...)); err != nil {
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

func taskOwnedFiles(t spec.Task) []string {
	if t.Exec == nil {
		return nil
	}
	if t.Exec.Plan != nil && len(t.Exec.Plan.TouchedFiles) > 0 {
		return t.Exec.Plan.TouchedFiles
	}
	return t.Exec.LastModifiedFiles
}

func siblingTasks(pr spec.Progress, ready []int, selfIdx int) []SiblingTask {
	siblings := make([]SiblingTask, 0, len(ready))
	for _, i := range ready {
		if i == selfIdx {
			continue
		}
		siblings = append(siblings, SiblingTask{
			ID:    pr.Tasks[i].ID,
			Title: pr.Tasks[i].Title,
			Files: taskOwnedFiles(pr.Tasks[i]),
		})
	}
	return siblings
}

func taskOverviewStatus(t spec.Task, ready map[string]bool) string {
	switch {
	case t.Done:
		return "done"
	case t.Blocked:
		return "blocked"
	case ready[t.ID]:
		return "ready"
	default:
		return "waiting"
	}
}

func specOverview(pr spec.Progress, selfID string) []TaskOverview {
	ready := make(map[string]bool)
	for _, i := range spec.ReadyTaskIndices(pr.Tasks) {
		ready[pr.Tasks[i].ID] = true
	}
	overview := make([]TaskOverview, 0, len(pr.Tasks))
	for _, t := range pr.Tasks {
		overview = append(overview, TaskOverview{
			ID:     t.ID,
			Title:  t.Title,
			Status: taskOverviewStatus(t, ready),
			Self:   t.ID == selfID,
		})
	}
	return overview
}

func taskTraceRefs(c *engine.Context, taskID string) []TraceRef {
	tr, err := c.LoadTraceability()
	if err != nil {
		log.Printf("tddmaster: failed to load traceability for %q: %v", c.Slug(), err)
		return nil
	}
	paths := make([]string, 0, len(tr.Entries))
	for path := range tr.Entries {
		paths = append(paths, path)
	}
	slices.Sort(paths)

	var refs []TraceRef
	for _, path := range paths {
		for _, e := range tr.Entries[path] {
			if e.TaskID != taskID {
				continue
			}
			refs = append(refs, TraceRef{
				TestFilePath: path,
				FunctionName: e.FunctionName,
				AC:           e.CriterionIDs,
				EC:           e.EC,
			})
		}
	}
	return refs
}

func (d *LoopDriver) buildExecCtx(c *engine.Context, task spec.Task, taskIdx int, siblings []SiblingTask) ExecCtx {
	return ExecCtx{
		Settings:          c.Settings(),
		Task:              task,
		State:             *task.Exec,
		TaskIdx:           taskIdx,
		MaxRefactorRounds: defaultMaxRefactorRounds,
		Slug:              c.Slug(),
		Mode:              c.AnswerValue("mode"),
		Worktrees:         worktreesAvailable(c.Root()),
		UserContext:       c.AnswerValue("listen_context"),
		ScopeBoundary:     c.AnswerValue("scope_boundary"),
		VerificationHint:  c.AnswerValue("verification"),
		ProjectCommands:   projectcmds.Detect(c.Root()),
		Intent: SpecIntent{
			StatusQuo:          c.AnswerValue("status_quo"),
			Ambition:           c.AnswerValue("ambition"),
			Reversibility:      c.AnswerValue("reversibility"),
			UserImpact:         c.AnswerValue("user_impact"),
			ChallengedPremises: spec.ParseChallengedPremises(c.AnswerValue("premises")),
		},
		Overview: specOverview(c.Progress(), task.ID),
		Trace:    taskTraceRefs(c, task.ID),
		Siblings: siblings,
		Rules:    c.Rules(),
	}
}

// worktreeInstructionBlock resolves the worktree cwd against the project root
// before handing it to a sub-agent. The stored path is relative because the
// orchestrator's git commands run from the root, but a sub-agent has no such
// anchor — and `.tddmaster/worktrees/<slug>/<task>` is a path that exists under
// more than one repository on a machine that develops tddmaster itself.
func worktreeInstructionBlock(root string, w *spec.WorktreeRef) string {
	if w == nil {
		return ""
	}
	absRoot := root
	if abs, err := filepath.Abs(root); err == nil {
		absRoot = abs
	}
	checkoutRoot := w.Path
	if !filepath.IsAbs(checkoutRoot) {
		checkoutRoot = filepath.Join(absRoot, w.Path)
	}
	cwd := checkoutRoot
	if gitRoot, ok := gitWorkingTreeRoot(absRoot); ok {
		// git worktree checks out the whole repository at checkoutRoot. When
		// tddmaster was initialized in a nested project (for example repo/cmd),
		// restore that same repository-relative directory inside the checkout.
		if projectRel, err := filepath.Rel(gitRoot, absRoot); err == nil && projectRel != "." {
			cwd = filepath.Join(checkoutRoot, projectRel)
		}
	}
	return fmt.Sprintf(promptregistry.WorktreeBlockFmt, absRoot, cwd, w.Branch)
}

func firstPath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

// reportShapeError turns a Go decoding failure into something the orchestrator
// can act on. The raw error names the struct field and the Go type behind it,
// which tells a sub-agent nothing about the JSON it should have sent.
func reportShapeError(err error) error {
	var typeErr *json.UnmarshalTypeError
	if !errors.As(err, &typeErr) || typeErr.Field == "" {
		return errs.Newf(errs.KeyReportShape, "report", err.Error())
	}
	want := "a value of a different type"
	if hint, ok := reportFieldShapes[typeErr.Field]; ok {
		want = hint
	}
	return errs.Newf(errs.KeyReportShape, typeErr.Field, "got a JSON "+typeErr.Value+", expected "+want)
}

var reportFieldShapes = map[string]string{
	"refactorNotes": `an array of objects, each {"file":"...","suggestion":"...","rationale":"..."}`,
	"traceability":  `an array of objects, each {"testFilePath":"...","functionName":"...","taskId":"...","ac":["ac-1"],"ec":["ec-1"]}`,
	"fileCoverage":  `an array of objects, each {"file":"...","coverage":85}`,
	"plan":          "an object, not a string",
	"filesModified": "an array of path strings",
	"testsWritten":  "an array of test function name strings",
	"failedACs":     "an array of `ac-N` id strings",
	"blocked":       "an array of reason strings",
}

// normalizeReportedPaths strips the worktree prefix a sub-agent may have glued
// onto its reported paths. Left alone, one stage reporting `calc/calc.go` and
// the next reporting `.tddmaster/worktrees/s/task-1/calc/calc.go` puts the same
// file in the CHANGED FILES section twice under two names.
func normalizeReportedPaths(paths []string, w *spec.WorktreeRef) []string {
	if len(paths) == 0 {
		return paths
	}
	prefixes := []string{}
	if w != nil && w.Path != "" {
		prefixes = append(prefixes, filepath.Clean(w.Path))
	}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		clean := filepath.Clean(strings.TrimSpace(p))
		for _, prefix := range prefixes {
			if rel, err := filepath.Rel(prefix, clean); err == nil && !strings.HasPrefix(rel, "..") {
				clean = rel
				break
			}
		}
		out = append(out, clean)
	}
	return out
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

func batchSummary(entries []string, worktrees, needsTestWriterNote, needsRefactorNote bool) string {
	isolation := promptregistry.BatchWorktreeIsolation
	if !worktrees {
		isolation = promptregistry.BatchNoGitIsolation
	} else if len(entries) == 1 {
		isolation = promptregistry.BatchSingleWorktreeIsolation
	}
	summary := fmt.Sprintf(promptregistry.BatchSummaryFmt, len(entries), strings.Join(entries, ", "), isolation)
	if needsTestWriterNote {
		summary += promptregistry.BatchTestWriterNote
	}
	if needsRefactorNote {
		summary += promptregistry.BatchRefactorTestWriterNote
	}
	return summary
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
	var nonTDDBatch bool
	var refactorApplyBatch bool

	for _, i := range ready {
		ctx := d.buildExecCtx(c, pr.Tasks[i], i, siblingTasks(pr, ready, i))
		stage, ok := d.ruleset.Next(ctx)
		if !ok {
			stuck = append(stuck, fmt.Sprintf("%s (exec: %+v)", pr.Tasks[i].ID, *pr.Tasks[i].Exec))
			continue
		}
		stageAction := stage.Prompt(ctx)
		if stage.ID() == StageIDGate {
			if gateAction == nil {
				gateTask = taskActionFor(c.Root(), ctx, stage.ID(), stageAction, false)
				gateAction = &stageAction
			}
			continue
		}
		taskActions = append(taskActions, taskActionFor(c.Root(), ctx, stage.ID(), stageAction, ctx.Worktrees))
		summary = append(summary, pr.Tasks[i].ID+" ("+stage.ID()+")")
		if stage.ID() == StageIDExecutor {
			nonTDDBatch = true
		}
		// Only the apply round carries notes the executor must act on; the
		// follow-up verify round has none, so the note would be pure noise.
		if stage.ID() == StageIDRefactor && !ctx.State.RefactorApplied && len(ctx.State.RefactorNotes) > 0 {
			refactorApplyBatch = true
		}
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
		// The task entry carries the planner-facing brief verbatim; the top-level
		// instruction is the orchestrator's own script and must not leak into it.
		action.Instruction = promptregistry.GateOrchestratorText
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
		Instruction:   batchSummary(summary, worktreesAvailable(c.Root()), nonTDDBatch, refactorApplyBatch),
		ExpectedInput: engine.ExpectedInput{Format: engine.FormatJSON},
		Tasks:         taskActions,
	}, false
}

func taskActionFor(root string, ctx ExecCtx, stageID string, stageAction engine.Action, withWorktree bool) engine.TaskAction {
	ta := engine.TaskAction{
		TaskID:              ctx.Task.ID,
		Stage:               stageID,
		CompletionCondition: taskCompletionCondition(ctx, stageID),
		Instruction:         stageAction.Instruction,
		DelegateAgent:       stageAction.DelegateAgent,
		ExpectedInput:       injectTaskIDIntoExample(stageAction.ExpectedInput, ctx.Task.ID),
	}
	if withWorktree {
		ta.Instruction = worktreeInstructionBlock(root, ctx.Task.Exec.Worktree) + ta.Instruction
		ta.Worktree = ctx.Task.Exec.Worktree
	}
	return ta
}

func taskCompletionCondition(ctx ExecCtx, stageID string) engine.TaskCompletionCondition {
	if !tddActive(ctx) {
		if stageID == StageIDVerifier || (stageID == StageIDExecutor && ctx.Settings.SkipVerifierEnabled) {
			return engine.TaskCompletionOnSuccess
		}
		return engine.TaskCompletionNever
	}

	if stageID == StageIDVerifier && ctx.State.TDDCycle == cycleGreen {
		return engine.TaskCompletionOnSuccessWithoutRefactorNotes
	}
	if stageID != StageIDRefactor {
		return engine.TaskCompletionNever
	}

	terminalRound := refactorCapReached(ctx.State.RefactorRounds+1, ctx.MaxRefactorRounds)
	if ctx.State.RefactorApplied {
		if terminalRound {
			return engine.TaskCompletionOnSuccess
		}
		return engine.TaskCompletionOnSuccessWithoutRefactorNotes
	}
	if ctx.Settings.SkipVerifierEnabled {
		if terminalRound {
			return engine.TaskCompletionOnSuccess
		}
		return engine.TaskCompletionOnSuccessWithoutRefactorNotes
	}
	return engine.TaskCompletionNever
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
		return engine.Action{}, false, reportShapeError(err)
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
	report.FilesModified = normalizeReportedPaths(report.FilesModified, pr.Tasks[taskIdx].Exec.Worktree)
	for i := range report.Traceability {
		report.Traceability[i].TestFilePath = firstPath(normalizeReportedPaths(
			[]string{report.Traceability[i].TestFilePath}, pr.Tasks[taskIdx].Exec.Worktree))
	}

	blocked := len(report.Blocked) > 0
	if pr.Tasks[taskIdx].Blocked && !blocked {
		pr.Tasks[taskIdx].Blocked = false
		pr.Tasks[taskIdx].BlockedReason = ""
		if !report.HasStageResult() {
			return d.finishSubmit(c, pr, "")
		}
	}
	if blocked {
		pr.Tasks[taskIdx].Blocked = true
		pr.Tasks[taskIdx].BlockedReason = strings.Join(report.Blocked, "; ")
	}
	task := pr.Tasks[taskIdx]

	ctx := d.buildExecCtx(c, task, taskIdx, siblingTasks(pr, ready, taskIdx))

	stage, ok := d.ruleset.Next(ctx)
	if !ok {
		return engine.Action{}, false, errs.Newf(errs.KeyNoApplicableStage, task.ID, ctx.State)
	}

	if report.HasGateAnswer() && stage.ID() != StageIDGate {
		return d.finishSubmit(c, pr, "")
	}

	if stage.ID() == StageIDRed && (!blocked || len(report.Traceability) > 0) {
		if err := validateAndPersistTraceability(c, task, report); err != nil {
			return engine.Action{}, false, err
		}
	}

	// With TDD off there is no red stage, so the tests such a task needs are
	// written outside the cycle and their traceability has nowhere to land.
	// Accept it here when offered — optional, since most executor tasks write
	// no tests at all — so traceability.json is not silently incomplete.
	if stage.ID() == StageIDExecutor && len(report.Traceability) > 0 {
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
	if reportFromVerifier(stage.ID(), ctx.State) && !report.EffectivePassed() {
		pr.Tasks[taskIdx].FailedACReasons = appendUniqueStrings(
			pr.Tasks[taskIdx].FailedACReasons,
			failureLearningEntries(report),
		)
	}

	if taskComplete {
		pr.Tasks = completeCurrentTask(pr.Tasks, taskIdx)
	}

	return d.finishSubmit(c, pr, verifierNote(stage.ID(), ctx.State, task.ID, report))
}

func failureLearningEntries(report StageReport) []string {
	ids := make([]string, 0, len(report.FailedACs)+len(report.UncoveredEdgeCases))
	ids = append(ids, report.FailedACs...)
	ids = append(ids, report.UncoveredEdgeCases...)

	details := make([]string, 0, 2)
	for _, detail := range []string{report.Reason, report.Output} {
		detail = strings.TrimSpace(detail)
		if detail != "" && !slices.Contains(details, detail) {
			details = append(details, detail)
		}
	}
	detail := strings.Join(details, "; ")
	if len(ids) == 0 {
		if detail == "" {
			return nil
		}
		return []string{detail}
	}

	entries := make([]string, 0, len(ids))
	for _, id := range ids {
		if detail == "" {
			entries = append(entries, id)
			continue
		}
		entries = append(entries, id+": "+detail)
	}
	return entries
}

// verifierNote extracts the one piece of a passing verification that has no
// other route to the user. A failed criterion drives the cycle and a refactor
// note drives the refactor round, but `reason` on a pass is pure prose: it was
// stored and never surfaced, so a verifier reporting a broken project command
// was talking to nobody.
func verifierNote(stageID string, st spec.ExecState, taskID string, report StageReport) string {
	if !reportFromVerifier(stageID, st) {
		return ""
	}
	reason := strings.TrimSpace(report.Reason)
	if reason == "" || !report.EffectivePassed() {
		return ""
	}
	return fmt.Sprintf(promptregistry.VerifierNoteFmt, taskID, reason)
}

func (d *LoopDriver) finishSubmit(c *engine.Context, pr spec.Progress, note string) (engine.Action, bool, error) {
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

	if note != "" {
		instruction := strings.TrimSpace(note)
		if limitHit {
			instruction = promptregistry.RestartRecommendedText + note
		}
		return engine.Action{Action: engine.ActionNotify, Instruction: instruction}, false, nil
	}

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
