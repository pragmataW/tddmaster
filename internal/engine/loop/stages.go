package loop

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func instructionFor(key promptregistry.InstructionKey) string {
	text, _ := promptregistry.Instruction(key)
	return text
}

func appendACsAndECs(b *strings.Builder, task spec.Task) {
	if len(task.AC) > 0 {
		b.WriteString("\n\nAcceptance Criteria:\n")
		for _, ac := range task.AC {
			b.WriteString("- ")
			b.WriteString(ac)
			b.WriteString("\n")
		}
	}
	if len(task.EdgeCases) > 0 {
		b.WriteString("\nEdge Cases:\n")
		for _, ec := range task.EdgeCases {
			b.WriteString("- ")
			b.WriteString(ec)
			b.WriteString("\n")
		}
	}
}

func appendUserContext(b *strings.Builder, userContext string) {
	if userContext != "" {
		b.WriteString("\nUser Context: ")
		b.WriteString(userContext)
		b.WriteString("\n")
	}
}

func appendApprovedPlan(b *strings.Builder, state spec.ExecState, taskID string) {
	plan, ok := state.TaskPlans[taskID]
	if !ok {
		return
	}
	b.WriteString("\nApproved plan approach: ")
	b.WriteString(plan.Approach)
	b.WriteString("\nTouched files: ")
	b.WriteString(strings.Join(plan.TouchedFiles, ", "))
	b.WriteString("\n")
}

func appendFailedACs(b *strings.Builder, state spec.ExecState) {
	if len(state.LastFailedACs) == 0 && len(state.LastUncoveredEC) == 0 {
		return
	}
	b.WriteString("\n")
	b.WriteString(instructionFor(promptregistry.KeyExecVerifyFailed))
	b.WriteString("\n")
	if len(state.LastFailedACs) > 0 {
		b.WriteString("Failed ACs:\n")
		for _, ac := range state.LastFailedACs {
			b.WriteString("- ")
			b.WriteString(ac)
			b.WriteString("\n")
		}
	}
	if len(state.LastUncoveredEC) > 0 {
		b.WriteString("Uncovered Edge Cases:\n")
		for _, ec := range state.LastUncoveredEC {
			b.WriteString("- ")
			b.WriteString(ec)
			b.WriteString("\n")
		}
	}
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
	return !slices.Contains(ctx.State.ApprovedPlans, ctx.Task.ID)
}

func (gateStageImpl) Prompt(ctx ExecCtx) engine.Action {
	var b strings.Builder
	b.WriteString(instructionFor(promptregistry.KeyExecGate))
	if feedback, ok := ctx.State.PlanFeedback[ctx.Task.ID]; ok && feedback != "" {
		attempts := ctx.State.PlanAttempts[ctx.Task.ID]
		b.WriteString(fmt.Sprintf("\nPrior feedback: %s\nattemptCount: %d\n", feedback, attempts))
	}
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
	if report.Accepted && report.Plan != nil {
		if ctx.State.TaskPlans == nil {
			ctx.State.TaskPlans = map[string]spec.TaskPlan{}
		}
		ctx.State.TaskPlans[ctx.Task.ID] = *report.Plan
		if !slices.Contains(ctx.State.ApprovedPlans, ctx.Task.ID) {
			ctx.State.ApprovedPlans = append(ctx.State.ApprovedPlans, ctx.Task.ID)
		}
		return ctx, nil
	}
	if !report.Accepted && report.PlanFeedback != "" {
		if ctx.State.PlanFeedback == nil {
			ctx.State.PlanFeedback = map[string]string{}
		}
		if ctx.State.PlanAttempts == nil {
			ctx.State.PlanAttempts = map[string]int{}
		}
		ctx.State.PlanFeedback[ctx.Task.ID] = report.PlanFeedback
		ctx.State.PlanAttempts[ctx.Task.ID]++
		return ctx, nil
	}
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
	b.WriteString(instructionFor(promptregistry.KeyExecRed))
	appendACsAndECs(&b, ctx.Task)
	appendUserContext(&b, ctx.UserContext)
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

func (redStageImpl) OnReport(ctx ExecCtx, report StageReport) (ExecCtx, error) {
	if len(report.TestsWritten) == 0 && !report.Passed {
		return ctx, nil
	}
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
	appendFailedACs(&b, ctx.State)
	b.WriteString(instructionFor(promptregistry.KeyExecGreen))
	appendACsAndECs(&b, ctx.Task)
	appendUserContext(&b, ctx.UserContext)
	appendApprovedPlan(&b, ctx.State, ctx.Task.ID)
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
		b.WriteString(instructionFor(promptregistry.KeyExecRefactorApply))
		appendACsAndECs(&b, ctx.Task)
		appendUserContext(&b, ctx.UserContext)
		return engine.Action{
			Action:        engine.ActionInstruct,
			DelegateAgent: string(promptregistry.AgentExecutor),
			ExpectedInput: engine.ExpectedInput{
				Format:  engine.FormatJSON,
				Example: promptregistry.ReportExampleRefactorApply,
			},
			Instruction: b.String(),
		}
	}
	b.WriteString(instructionFor(promptregistry.KeyExecRefactor))
	appendACsAndECs(&b, ctx.Task)
	appendUserContext(&b, ctx.UserContext)
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
	if !ctx.State.RefactorApplied {
		ctx.State.RefactorApplied = true
		if !ctx.Settings.SkipVerifierEnabled {
			return ctx, nil
		}
	}
	newSt, _, err := advanceCycleStrict(ctx.State, report.EffectivePassed(), report.RefactorNotesPresent(), ctx.MaxRefactorRounds)
	if err != nil {
		return ctx, err
	}
	if newSt.TDDCycle == cycleRefactor {
		newSt.RefactorApplied = false
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
	appendFailedACs(&b, ctx.State)
	if ctx.Settings.SkipVerifierEnabled {
		b.WriteString(instructionFor(promptregistry.KeyExecExecutorSkipVerify))
	} else {
		b.WriteString(instructionFor(promptregistry.KeyExecExecutor))
	}
	appendACsAndECs(&b, ctx.Task)
	appendUserContext(&b, ctx.UserContext)
	appendApprovedPlan(&b, ctx.State, ctx.Task.ID)
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
	if len(report.Blocked) > 0 {
		return ctx, nil
	}
	ctx.State.Implemented = true
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
	b.WriteString(instructionFor(promptregistry.KeyExecVerifier))
	appendACsAndECs(&b, ctx.Task)
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
		return ctx, nil
	}
	if !tddActive(ctx) {
		return ctx, nil
	}
	newSt, _ := advanceCycle(ctx.State, true, report.RefactorNotesPresent(), ctx.MaxRefactorRounds)
	newSt.Implemented = false
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
