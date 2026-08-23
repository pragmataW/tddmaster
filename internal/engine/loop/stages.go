package loop

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/rules"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func instructionFor(key promptregistry.InstructionKey) string {
	text, _ := promptregistry.Instruction(key)
	return text
}

func section(b *strings.Builder, header string) {
	if b.Len() > 0 {
		b.WriteString("\n")
	}
	b.WriteString(header)
	b.WriteString("\n")
}

func writeList(b *strings.Builder, items []string) {
	for _, item := range items {
		b.WriteString("- ")
		b.WriteString(item)
		b.WriteString("\n")
	}
}

func edgeCaseID(idx int) string {
	return spec.EdgeCaseIDPrefix + strconv.Itoa(idx+1)
}

func appendTaskHeader(b *strings.Builder, ctx ExecCtx, stageID string) {
	section(b, promptregistry.SectionTask)
	if ctx.Slug != "" {
		b.WriteString("spec: ")
		b.WriteString(ctx.Slug)
		b.WriteString("\n")
	}
	b.WriteString("taskId: ")
	b.WriteString(ctx.Task.ID)
	b.WriteString("\ntitle: ")
	b.WriteString(ctx.Task.Title)
	b.WriteString("\nstage: ")
	b.WriteString(stageID)
	b.WriteString("\n")
	if ctx.Mode != "" {
		b.WriteString("discoveryMode: ")
		b.WriteString(ctx.Mode)
		if desc := promptregistry.ModeDescription(ctx.Mode); desc != "" {
			b.WriteString(" — ")
			b.WriteString(desc)
		}
		b.WriteString("\n")
	}
	if tddActive(ctx) && ctx.State.TDDCycle != "" {
		b.WriteString("tddPhase: ")
		b.WriteString(ctx.State.TDDCycle)
		b.WriteString("\n")
	}
	if len(ctx.Task.DependsOn) > 0 {
		b.WriteString("dependsOn: ")
		b.WriteString(strings.Join(ctx.Task.DependsOn, ", "))
		b.WriteString("\n")
	}
}

func appendACsAndECs(b *strings.Builder, task spec.Task) {
	if len(task.Criteria) > 0 {
		section(b, promptregistry.SectionCriteria)
		b.WriteString(promptregistry.CriterionIDNote)
		for _, c := range task.Criteria {
			b.WriteString("- [")
			b.WriteString(c.ID)
			b.WriteString("]")
			b.WriteString(spec.FormatCriterionInline(c))
			b.WriteString("\n")
		}
	}
	if len(task.EdgeCases) > 0 {
		section(b, promptregistry.SectionEdgeCases)
		for i, ec := range task.EdgeCases {
			b.WriteString("- [")
			b.WriteString(edgeCaseID(i))
			b.WriteString("] ")
			b.WriteString(ec)
			b.WriteString("\n")
		}
	}
}

func appendUserContext(b *strings.Builder, userContext string) {
	if userContext == "" {
		return
	}
	section(b, promptregistry.SectionSpecContext)
	b.WriteString(promptregistry.SpecContextNote)
	b.WriteString(userContext)
	b.WriteString("\n")
}

func appendSpecOverview(b *strings.Builder, ctx ExecCtx) {
	if len(ctx.Overview) == 0 {
		return
	}
	section(b, promptregistry.SectionSpecOverview)
	b.WriteString(promptregistry.SpecOverviewNote)
	for _, t := range ctx.Overview {
		b.WriteString("- [")
		b.WriteString(t.Status)
		b.WriteString("] ")
		b.WriteString(t.ID)
		b.WriteString(": ")
		b.WriteString(t.Title)
		if t.Self {
			b.WriteString("  <- YOU")
		}
		b.WriteString("\n")
	}
}

func appendSpecIntent(b *strings.Builder, ctx ExecCtx) {
	if ctx.Intent.Empty() {
		return
	}
	section(b, promptregistry.SectionSpecIntent)
	b.WriteString(promptregistry.SpecIntentNote)
	for _, row := range []struct{ label, value string }{
		{"today, without this feature", ctx.Intent.StatusQuo},
		{"ambition (1-star vs 10-star)", ctx.Intent.Ambition},
		{"reversibility", ctx.Intent.Reversibility},
		{"impact on existing users", ctx.Intent.UserImpact},
	} {
		if row.value == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(row.label)
		b.WriteString(": ")
		b.WriteString(row.value)
		b.WriteString("\n")
	}
	if len(ctx.Intent.ChallengedPremises) == 0 {
		return
	}
	b.WriteString(promptregistry.ChallengedPremisesLabel)
	b.WriteString("\n")
	writeList(b, ctx.Intent.ChallengedPremises)
}

func mergeFiles(lists ...[]string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, list := range lists {
		for _, f := range list {
			if f == "" || seen[f] {
				continue
			}
			seen[f] = true
			out = append(out, f)
		}
	}
	return out
}

func appendProjectCommands(b *strings.Builder, ctx ExecCtx) {
	if len(ctx.ProjectCommands) == 0 {
		return
	}
	section(b, promptregistry.SectionProjectCmds)
	b.WriteString(promptregistry.ProjectCommandsNote)
	for _, cmd := range ctx.ProjectCommands {
		b.WriteString("- ")
		b.WriteString(cmd.Label)
		b.WriteString(": ")
		b.WriteString(cmd.Value)
		b.WriteString("\n")
	}
}

func appendTraceability(b *strings.Builder, ctx ExecCtx) {
	if len(ctx.Trace) == 0 {
		return
	}
	section(b, promptregistry.SectionTraceability)
	b.WriteString(promptregistry.TraceabilityNote)
	for _, t := range ctx.Trace {
		b.WriteString("- ")
		b.WriteString(t.TestFilePath)
		b.WriteString("::")
		b.WriteString(t.FunctionName)
		covered := append(append([]string{}, t.AC...), t.EC...)
		if len(covered) > 0 {
			b.WriteString(" covers ")
			b.WriteString(strings.Join(covered, ", "))
		}
		b.WriteString("\n")
	}
}

func appendOutOfScope(b *strings.Builder, ctx ExecCtx) {
	if ctx.ScopeBoundary == "" {
		return
	}
	section(b, promptregistry.SectionOutOfScope)
	b.WriteString(promptregistry.OutOfScopeNote)
	b.WriteString(ctx.ScopeBoundary)
	b.WriteString("\n")
}

func appendVerificationMethod(b *strings.Builder, ctx ExecCtx) {
	if ctx.VerificationHint == "" {
		return
	}
	section(b, promptregistry.SectionVerification)
	b.WriteString(promptregistry.VerificationNote)
	b.WriteString(ctx.VerificationHint)
	b.WriteString("\n")
}

func appendSiblings(b *strings.Builder, ctx ExecCtx) {
	if len(ctx.Siblings) == 0 {
		return
	}
	section(b, promptregistry.SectionParallelTasks)
	if ctx.Worktrees {
		b.WriteString(promptregistry.ParallelTasksNote)
	} else {
		b.WriteString(promptregistry.ParallelTasksNoGitNote)
	}
	anyUnknown := false
	for _, s := range ctx.Siblings {
		b.WriteString("- ")
		b.WriteString(s.ID)
		if s.Title != "" {
			b.WriteString(": ")
			b.WriteString(s.Title)
		}
		if len(s.Files) > 0 {
			b.WriteString(" (owns: ")
			b.WriteString(strings.Join(s.Files, ", "))
			b.WriteString(")")
		} else {
			anyUnknown = true
		}
		b.WriteString("\n")
	}
	if anyUnknown {
		b.WriteString(promptregistry.ParallelTasksUnknownFilesNote)
	}
}

// implementationFiles lists every production file this task has written so far.
// It accumulates across rounds so a later stage still sees work an earlier round
// produced, even though LastModifiedFiles only ever holds the newest stage.
func implementationFiles(st spec.ExecState) []string {
	return mergeFiles(st.ImplFiles, st.LastModifiedFiles)
}

func appendChangedFilesWithNote(b *strings.Builder, ctx ExecCtx, note string) {
	files := mergeFiles(implementationFiles(ctx.State), ctx.State.TestFiles)
	if len(files) == 0 {
		return
	}
	section(b, promptregistry.SectionChangedFiles)
	b.WriteString(note)
	writeList(b, files)
}

func appendChangedFiles(b *strings.Builder, ctx ExecCtx) {
	appendChangedFilesWithNote(b, ctx, promptregistry.ChangedFilesNote)
}

func appendGreenChangedFiles(b *strings.Builder, ctx ExecCtx) {
	files := mergeFiles(ctx.State.TestFiles, implementationFiles(ctx.State))
	if len(files) == 0 {
		return
	}
	section(b, promptregistry.SectionChangedFiles)
	b.WriteString(promptregistry.GreenChangedFilesNote)
	writeList(b, files)
}

func appendRefactorNotes(b *strings.Builder, notes []spec.RefactorNote) {
	if len(notes) == 0 {
		return
	}
	section(b, promptregistry.SectionRefactorNotes)
	b.WriteString(promptregistry.RefactorNotesNote)
	for _, n := range notes {
		b.WriteString("- [")
		b.WriteString(n.File)
		b.WriteString("] ")
		b.WriteString(n.Suggestion)
		if n.Rationale != "" {
			b.WriteString(" (")
			b.WriteString(n.Rationale)
			b.WriteString(")")
		}
		b.WriteString("\n")
	}
}

func appendApprovedPlan(b *strings.Builder, state spec.ExecState) {
	if state.Plan == nil {
		return
	}
	section(b, promptregistry.SectionApprovedPlan)
	b.WriteString("approach: ")
	b.WriteString(state.Plan.Approach)
	b.WriteString("\ntouchedFiles (BINDING — a file outside this list means STOP and report blocked):\n")
	writeList(b, state.Plan.TouchedFiles)
	if len(state.Plan.DesignPatterns) > 0 {
		b.WriteString("designPatterns:\n")
		writeList(b, state.Plan.DesignPatterns)
	}
	if len(state.Plan.BestPractices) > 0 {
		b.WriteString("bestPractices:\n")
		writeList(b, state.Plan.BestPractices)
	}
	if len(state.Plan.Assumptions) > 0 {
		b.WriteString("assumptions (if one no longer holds, STOP and report blocked):\n")
		writeList(b, state.Plan.Assumptions)
	}
}

func appendFailedACs(b *strings.Builder, state spec.ExecState) {
	appendFailedACsWithNote(b, state, instructionFor(promptregistry.KeyExecVerifyFailed))
}

func appendFailedACsWithNote(b *strings.Builder, state spec.ExecState, note string) {
	if len(state.LastFailedACs) == 0 && len(state.LastUncoveredEC) == 0 {
		return
	}
	section(b, promptregistry.SectionLastFailure)
	b.WriteString(note)
	b.WriteString("\n")
	if len(state.LastFailedACs) > 0 {
		b.WriteString("failed criteria:\n")
		writeList(b, state.LastFailedACs)
	}
	if len(state.LastUncoveredEC) > 0 {
		b.WriteString("uncovered edge cases:\n")
		writeList(b, state.LastUncoveredEC)
	}
}

const inlineRulesBudget = 8000

func rulesFitInline(docs []rules.Doc) bool {
	total := 0
	for _, d := range docs {
		total += len(d.Body)
	}
	return total > 0 && total <= inlineRulesBudget
}

func appendRules(b *strings.Builder, ctx ExecCtx, agent string) {
	paths := ctx.Rules.For(agent)
	if len(paths) == 0 {
		return
	}
	docs := ctx.Rules.Docs(agent)
	if len(docs) == len(paths) && rulesFitInline(docs) {
		b.WriteString(promptregistry.RulesInlineHeader)
		for _, d := range docs {
			b.WriteString("\n--- ")
			b.WriteString(d.Path)
			b.WriteString(" ---\n")
			b.WriteString(strings.TrimSpace(d.Body))
			b.WriteString("\n")
		}
		b.WriteString(promptregistry.RulesInlineFooter)
		return
	}
	b.WriteString(promptregistry.RulesInjectionHeader)
	if len(docs) == len(paths) {
		b.WriteString(promptregistry.RulesTruncatedNote)
	}
	writeList(b, paths)
	b.WriteString(promptregistry.RulesInjectionFooter)
}

func appendInstruction(b *strings.Builder, key promptregistry.InstructionKey) {
	section(b, promptregistry.SectionInstruction)
	b.WriteString(instructionFor(key))
	b.WriteString("\n")
}

func appendScopeBlock(b *strings.Builder, ctx ExecCtx, stageID string) {
	appendTaskHeader(b, ctx, stageID)
	appendSpecOverview(b, ctx)
	appendUserContext(b, ctx.UserContext)
	appendSpecIntent(b, ctx)
	appendACsAndECs(b, ctx.Task)
	appendOutOfScope(b, ctx)
}

func tddActive(ctx ExecCtx) bool {
	return ctx.Settings.TDDEnabled && ctx.Task.TDDEnabled
}

func tddCycleApplies(ctx ExecCtx, cycle string) bool {
	return tddActive(ctx) && ctx.State.TDDCycle == cycle
}

type gateStageImpl struct{}

func gateStage() Stage { return gateStageImpl{} }

func (gateStageImpl) ID() string { return StageIDGate }

func (gateStageImpl) Applies(ctx ExecCtx) bool {
	if !ctx.Settings.ImportantTaskGateEnabled {
		return false
	}
	if !ctx.Task.Important {
		return false
	}
	return !ctx.State.PlanApproved
}

func (gateStageImpl) Prompt(ctx ExecCtx) engine.Action {
	var b strings.Builder
	appendScopeBlock(&b, ctx, StageIDGate)
	appendVerificationMethod(&b, ctx)
	appendSiblings(&b, ctx)
	if ctx.State.PlanFeedback != "" {
		section(&b, promptregistry.SectionPriorFeedback)
		fmt.Fprintf(&b, "attemptCount: %d\n%s\n", ctx.State.PlanAttempts, ctx.State.PlanFeedback)
	}
	appendRules(&b, ctx, "planner")
	appendInstruction(&b, promptregistry.KeyExecGate)
	return engine.Action{
		Action:        engine.ActionAsk,
		DelegateAgent: string(promptregistry.AgentPlanner),
		ExpectedInput: engine.ExpectedInput{
			Format:  engine.FormatJSON,
			Example: promptregistry.ReportExamplePlanner,
		},
		Instruction: b.String(),
	}
}

func (gateStageImpl) OnReport(ctx ExecCtx, report StageReport) (ExecCtx, error) {
	if report.PlanFeedback != "" {
		ctx.State.PlanFeedback = report.PlanFeedback
		ctx.State.PlanAttempts++
		return ctx, nil
	}
	if report.Plan == nil {
		return ctx, errs.New(errs.KeyGateAnswerInvalid)
	}
	plan := *report.Plan
	ctx.State.Plan = &plan
	ctx.State.PlanApproved = true
	return ctx, nil
}

type redStageImpl struct{}

func redStage() Stage { return redStageImpl{} }

func (redStageImpl) ID() string { return StageIDRed }

func (redStageImpl) Applies(ctx ExecCtx) bool {
	return tddCycleApplies(ctx, cycleRed)
}

func (redStageImpl) Prompt(ctx ExecCtx) engine.Action {
	var b strings.Builder
	appendScopeBlock(&b, ctx, StageIDRed)
	appendVerificationMethod(&b, ctx)
	appendApprovedPlan(&b, ctx.State)
	appendTraceability(&b, ctx)
	appendSiblings(&b, ctx)
	appendFailedACsWithNote(&b, ctx.State, promptregistry.RedVerifyFailedNote)
	appendCoverageFeedback(&b, ctx)
	appendRules(&b, ctx, "test-writer")
	appendInstruction(&b, promptregistry.KeyExecRed)
	return engine.Action{
		Action:        engine.ActionInstruct,
		DelegateAgent: string(promptregistry.AgentTestWriter),
		ExpectedInput: engine.ExpectedInput{
			Format:  engine.FormatJSON,
			Example: promptregistry.ReportExampleTestWriter,
		},
		Instruction: b.String(),
	}
}

// OnReport rejects an incomplete RED report instead of absorbing it. An empty
// TestsWritten used to leave the cycle in RED with no error at all, so `next`
// re-emitted the identical round until the iteration limit; an empty
// FilesModified silently robbed the GREEN executor of the failing-test list the
// prompt promises it. Both are contract violations and must say so.
func (redStageImpl) OnReport(ctx ExecCtx, report StageReport) (ExecCtx, error) {
	if len(report.TestsWritten) == 0 {
		return ctx, errs.New(errs.KeyRedTestsWrittenEmpty)
	}
	if len(report.FilesModified) == 0 {
		return ctx, errs.New(errs.KeyRedFilesModifiedEmpty)
	}
	ctx.State.LastModifiedFiles = report.FilesModified
	ctx.State.TestFiles = mergeFiles(ctx.State.TestFiles, report.FilesModified)
	ctx.State.TDDCycle = cycleGreen
	return ctx, nil
}

type greenStageImpl struct{}

func greenStage() Stage { return greenStageImpl{} }

func (greenStageImpl) ID() string { return StageIDGreen }

func (greenStageImpl) Applies(ctx ExecCtx) bool {
	return tddCycleApplies(ctx, cycleGreen) && !ctx.State.Implemented
}

func (greenStageImpl) Prompt(ctx ExecCtx) engine.Action {
	var b strings.Builder
	appendScopeBlock(&b, ctx, StageIDGreen)
	appendVerificationMethod(&b, ctx)
	appendApprovedPlan(&b, ctx.State)
	appendGreenChangedFiles(&b, ctx)
	appendTraceability(&b, ctx)
	appendSiblings(&b, ctx)
	appendFailedACs(&b, ctx.State)
	appendRules(&b, ctx, "executor")
	appendInstruction(&b, promptregistry.KeyExecGreen)
	return engine.Action{
		Action:        engine.ActionInstruct,
		DelegateAgent: string(promptregistry.AgentExecutor),
		ExpectedInput: engine.ExpectedInput{
			Format:  engine.FormatJSON,
			Example: promptregistry.ReportExampleExecutor,
		},
		Instruction: b.String(),
	}
}

func (greenStageImpl) OnReport(ctx ExecCtx, report StageReport) (ExecCtx, error) {
	ctx.State.Implemented = true
	if len(report.FilesModified) > 0 {
		ctx.State.LastModifiedFiles = report.FilesModified
		ctx.State.ImplFiles = mergeFiles(ctx.State.ImplFiles, report.FilesModified)
	}
	return ctx, nil
}

type refactorStageImpl struct{}

func refactorStage() Stage { return refactorStageImpl{} }

func (refactorStageImpl) ID() string { return StageIDRefactor }

func (refactorStageImpl) Applies(ctx ExecCtx) bool {
	return tddCycleApplies(ctx, cycleRefactor)
}

func (refactorStageImpl) Prompt(ctx ExecCtx) engine.Action {
	var b strings.Builder
	if !ctx.State.RefactorApplied {
		applyKey := promptregistry.KeyExecRefactorApply
		applyExample := promptregistry.ReportExampleRefactorApply
		if ctx.Settings.SkipVerifierEnabled {
			applyKey = promptregistry.KeyExecRefactorSkipVerify
			applyExample = promptregistry.ReportExampleRefactorApplySkip
		}
		appendScopeBlock(&b, ctx, StageIDRefactor)
		appendVerificationMethod(&b, ctx)
		appendApprovedPlan(&b, ctx.State)
		appendChangedFiles(&b, ctx)
		appendTraceability(&b, ctx)
		appendRefactorNotes(&b, ctx.State.RefactorNotes)
		appendSiblings(&b, ctx)
		appendFailedACs(&b, ctx.State)
		if ctx.Settings.SkipVerifierEnabled {
			appendProjectCommands(&b, ctx)
		}
		appendRules(&b, ctx, "executor")
		appendInstruction(&b, applyKey)
		return engine.Action{
			Action:        engine.ActionInstruct,
			DelegateAgent: string(promptregistry.AgentExecutor),
			ExpectedInput: engine.ExpectedInput{
				Format:  engine.FormatJSON,
				Example: applyExample,
			},
			Instruction: b.String(),
		}
	}
	appendScopeBlock(&b, ctx, StageIDRefactor)
	appendVerificationMethod(&b, ctx)
	appendApprovedPlan(&b, ctx.State)
	appendChangedFiles(&b, ctx)
	appendTraceability(&b, ctx)
	appendSiblings(&b, ctx)
	appendProjectCommands(&b, ctx)
	appendRules(&b, ctx, "verifier")
	appendInstruction(&b, promptregistry.KeyExecRefactor)
	return engine.Action{
		Action:        engine.ActionInstruct,
		DelegateAgent: string(promptregistry.AgentVerifier),
		ExpectedInput: engine.ExpectedInput{
			Format:  engine.FormatJSON,
			Example: promptregistry.ReportExampleVerifier,
		},
		Instruction: b.String(),
	}
}

func (refactorStageImpl) OnReport(ctx ExecCtx, report StageReport) (ExecCtx, error) {
	if len(report.FilesModified) > 0 {
		ctx.State.LastModifiedFiles = report.FilesModified
		ctx.State.ImplFiles = mergeFiles(ctx.State.ImplFiles, report.FilesModified)
	}
	applyRound := !ctx.State.RefactorApplied
	if applyRound {
		ctx.State.RefactorApplied = true
		if !ctx.Settings.SkipVerifierEnabled {
			return ctx, nil
		}
	}
	passed := report.EffectivePassed()
	if applyRound {
		passed = report.RefactorRoundPassed()
	}
	newSt, _, err := advanceCycleStrict(ctx.State, passed, report.RefactorNotesPresent(), ctx.MaxRefactorRounds)
	if err != nil {
		return ctx, err
	}
	if newSt.TDDCycle == cycleRefactor {
		newSt.RefactorApplied = false
		if passed {
			newSt.RefactorNotes = report.RefactorNotes
		}
	} else {
		newSt.RefactorNotes = nil
	}
	ctx.State = newSt
	return ctx, nil
}

type executorStageImpl struct{}

func executorStage() Stage { return executorStageImpl{} }

func (executorStageImpl) ID() string { return StageIDExecutor }

func (executorStageImpl) Applies(ctx ExecCtx) bool {
	if tddActive(ctx) {
		return false
	}
	if ctx.Settings.SkipVerifierEnabled {
		return true
	}
	return !ctx.State.Implemented
}

func (executorStageImpl) Prompt(ctx ExecCtx) engine.Action {
	var b strings.Builder
	key := promptregistry.KeyExecExecutor
	if ctx.Settings.SkipVerifierEnabled {
		key = promptregistry.KeyExecExecutorSkipVerify
	}
	appendScopeBlock(&b, ctx, StageIDExecutor)
	appendVerificationMethod(&b, ctx)
	appendApprovedPlan(&b, ctx.State)
	appendSiblings(&b, ctx)
	appendFailedACs(&b, ctx.State)
	if ctx.Settings.SkipVerifierEnabled {
		appendProjectCommands(&b, ctx)
	}
	appendRules(&b, ctx, "executor")
	appendInstruction(&b, key)
	return engine.Action{
		Action:        engine.ActionInstruct,
		DelegateAgent: string(promptregistry.AgentExecutor),
		ExpectedInput: engine.ExpectedInput{
			Format:  engine.FormatJSON,
			Example: promptregistry.ReportExampleExecutor,
		},
		Instruction: b.String(),
	}
}

func (executorStageImpl) OnReport(ctx ExecCtx, report StageReport) (ExecCtx, error) {
	ctx.State.Implemented = true
	if len(report.FilesModified) > 0 {
		ctx.State.LastModifiedFiles = report.FilesModified
		ctx.State.ImplFiles = mergeFiles(ctx.State.ImplFiles, report.FilesModified)
	}
	return ctx, nil
}

type verifierStageImpl struct{}

func verifierStage() Stage { return verifierStageImpl{} }

func (verifierStageImpl) ID() string { return StageIDVerifier }

func (verifierStageImpl) Applies(ctx ExecCtx) bool {
	if !ctx.State.Implemented {
		return false
	}
	if tddActive(ctx) {
		return ctx.State.TDDCycle == cycleGreen
	}
	return !ctx.Settings.SkipVerifierEnabled
}

func (verifierStageImpl) Prompt(ctx ExecCtx) engine.Action {
	var b strings.Builder
	appendScopeBlock(&b, ctx, StageIDVerifier)
	appendVerificationMethod(&b, ctx)
	appendApprovedPlan(&b, ctx.State)
	appendChangedFiles(&b, ctx)
	appendTraceability(&b, ctx)
	appendSiblings(&b, ctx)
	if tddActive(ctx) && ctx.State.TDDCycle == cycleGreen {
		appendCoverageRequirement(&b, ctx)
	}
	appendProjectCommands(&b, ctx)
	appendRules(&b, ctx, "verifier")
	appendInstruction(&b, promptregistry.KeyExecVerifier)
	return engine.Action{
		Action:        engine.ActionInstruct,
		DelegateAgent: string(promptregistry.AgentVerifier),
		ExpectedInput: engine.ExpectedInput{
			Format:  engine.FormatJSON,
			Example: promptregistry.ReportExampleVerifier,
		},
		Instruction: b.String(),
	}
}

func (verifierStageImpl) OnReport(ctx ExecCtx, report StageReport) (ExecCtx, error) {
	if !report.EffectivePassed() {
		ctx.State.Implemented = false
		if tddActive(ctx) && ctx.State.TDDCycle == cycleGreen && len(report.UncoveredEdgeCases) > 0 {
			ctx.State.TDDCycle = cycleRed
		}
		return ctx, nil
	}
	if !tddActive(ctx) {
		return ctx, nil
	}
	if ctx.State.TDDCycle == cycleGreen && coverageEnforced(ctx.Settings) {
		if len(report.FileCoverage) == 0 {
			ctx.State.CoverageUnreported = true
			return ctx, nil
		}
		ctx.State.CoverageUnreported = false
		if !coverageMet(report, ctx.Settings) {
			ctx.State.LastCoverage = coverageMap(report)
			ctx.State.Implemented = false
			ctx.State.TDDCycle = cycleRed
			return ctx, nil
		}
	}
	if ctx.State.TDDCycle == cycleGreen {
		ctx.State.LastCoverage = coverageMap(report)
	}
	newSt, _ := advanceCycle(ctx.State, true, report.RefactorNotesPresent(), ctx.MaxRefactorRounds)
	newSt.Implemented = false
	if newSt.TDDCycle == cycleRefactor {
		newSt.RefactorNotes = report.RefactorNotes
	} else {
		newSt.RefactorNotes = nil
	}
	ctx.State = newSt
	return ctx, nil
}

func allStages() []Stage {
	return []Stage{
		gateStage(),
		redStage(),
		greenStage(),
		refactorStage(),
		executorStage(),
		verifierStage(),
	}
}
