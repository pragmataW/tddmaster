package loop

import (
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/pragmataW/tddmaster/internal/engine"
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

func findFirstPendingTask(tasks []spec.Task) (spec.Task, int, bool) {
	for i, t := range tasks {
		if !t.Done {
			return t, i, true
		}
	}
	return spec.Task{}, -1, false
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

func (d *LoopDriver) initExecution(c *engine.Context, task spec.Task) error {
	pr := c.Progress()
	if pr.Execution == nil {
		st := spec.ExecState{}
		st = reseedCycle(st, c.Settings().TDDEnabled && task.TDDEnabled)
		pr.Execution = &st
		return c.SaveProgress(pr)
	}
	return nil
}

func (d *LoopDriver) buildExecCtx(c *engine.Context, task spec.Task, taskIdx int) ExecCtx {
	return ExecCtx{
		Settings:          c.Settings(),
		Task:              task,
		State:             *c.Progress().Execution,
		TaskIdx:           taskIdx,
		MaxRefactorRounds: defaultMaxRefactorRounds,
		UserContext:       c.AnswerValue("listen_context"),
		Rules:             c.Rules(),
	}
}

func (d *LoopDriver) Next(c *engine.Context, ph *engine.PhaseDef) (engine.Action, bool) {
	pr := c.Progress()

	if canTerminate(pr) {
		pr.Status = spec.StatusCompleted
		if err := c.SaveProgress(pr); err != nil {
			return engine.Action{Action: engine.ActionError, Instruction: err.Error()}, false
		}
		return engine.Action{Action: engine.ActionTerminal}, true
	}

	task, taskIdx, found := findFirstPendingTask(pr.Tasks)
	if !found {
		pr.Status = spec.StatusCompleted
		if err := c.SaveProgress(pr); err != nil {
			return engine.Action{Action: engine.ActionError, Instruction: err.Error()}, false
		}
		return engine.Action{Action: engine.ActionTerminal}, true
	}

	if pr.Execution == nil {
		if err := d.initExecution(c, task); err != nil {
			return engine.Action{Action: engine.ActionError, Instruction: err.Error()}, false
		}
		pr = c.Progress()
	}

	if pr.Execution.Iteration >= c.MaxIteration() {
		pr.Execution.Iteration = 0
		if err := c.SaveProgress(pr); err != nil {
			return engine.Action{Action: engine.ActionError, Instruction: err.Error()}, false
		}
		return engine.Action{Action: engine.ActionNotify, Instruction: promptregistry.RestartRecommendedText}, false
	}

	ctx := d.buildExecCtx(c, task, taskIdx)

	stage, ok := d.ruleset.Next(ctx)
	if !ok {
		return engine.Action{Action: engine.ActionNotify}, false
	}

	return stage.Prompt(ctx), false
}

func (d *LoopDriver) Submit(c *engine.Context, ph *engine.PhaseDef, answer []byte) (engine.Action, bool, error) {
	if strings.TrimSpace(string(answer)) == "continue" {
		pr := c.Progress()
		if pr.Execution != nil {
			pr.Execution.Iteration = 0
			if err := c.SaveProgress(pr); err != nil {
				return engine.Action{}, false, err
			}
		}
		return engine.Action{}, false, nil
	}

	if len(answer) == 0 || !json.Valid(answer) {
		return engine.Action{}, false, errors.New("invalid JSON answer")
	}

	var report StageReport
	if err := json.Unmarshal(answer, &report); err != nil {
		return engine.Action{}, false, err
	}

	pr := c.Progress()

	if canTerminate(pr) {
		return engine.Action{Action: engine.ActionTerminal}, true, nil
	}

	task, taskIdx, found := findFirstPendingTask(pr.Tasks)
	if !found {
		return engine.Action{Action: engine.ActionTerminal}, true, nil
	}

	if pr.Execution == nil {
		if err := d.initExecution(c, task); err != nil {
			return engine.Action{}, false, err
		}
		pr = c.Progress()
	}

	ctx := d.buildExecCtx(c, task, taskIdx)

	stage, ok := d.ruleset.Next(ctx)
	if !ok {
		return engine.Action{Action: engine.ActionNotify}, false, nil
	}

	if stage.ID() == StageIDRed {
		if err := validateAndPersistTraceability(c, task, report); err != nil {
			return engine.Action{}, false, err
		}
	}

	if stage.ID() == StageIDVerifier && tddActive(ctx) && ctx.State.TDDCycle == cycleGreen && len(report.FileCoverage) > 0 {
		if err := persistCoverage(c, report); err != nil {
			return engine.Action{}, false, err
		}
	}

	newCtx, err := stage.OnReport(ctx, report)
	if err != nil {
		return engine.Action{}, false, err
	}

	if reportFromVerifier(stage.ID(), ctx.State) {
		if report.EffectivePassed() {
			newCtx.State.LastFailedACs = nil
			newCtx.State.LastUncoveredEC = nil
		} else {
			newCtx.State.LastFailedACs = report.FailedACs
			newCtx.State.LastUncoveredEC = report.UncoveredEdgeCases
		}
	}

	pr.Execution = &newCtx.State

	taskComplete := isTaskComplete(c.Settings().TDDEnabled && task.TDDEnabled, c.Settings().SkipVerifierEnabled, ctx.State, newCtx.State, report)
	if taskComplete {
		pr.Tasks = completeCurrentTask(pr.Tasks, taskIdx)

		nextTask, _, hasNext := findFirstPendingTask(pr.Tasks)
		if hasNext {
			newSt := reseedCycle(*pr.Execution, c.Settings().TDDEnabled && nextTask.TDDEnabled)
			pr.Execution = &newSt
		}
	}

	pr.Execution.Iteration++

	if err := c.SaveProgress(pr); err != nil {
		return engine.Action{}, false, err
	}

	writeSpecMd(c)

	pr = c.Progress()
	if allTasksDone(pr.Tasks) {
		pr.Status = spec.StatusCompleted
		if err := c.SaveProgress(pr); err != nil {
			return engine.Action{}, false, err
		}
		writeSpecMd(c)
		return engine.Action{Action: engine.ActionTerminal}, true, nil
	}

	if pr.Execution.Iteration >= c.MaxIteration() {
		pr.Execution.Iteration = 0
		if err := c.SaveProgress(pr); err != nil {
			return engine.Action{}, false, err
		}
		return engine.Action{Action: engine.ActionNotify, Instruction: promptregistry.RestartRecommendedText}, false, nil
	}

	return engine.Action{}, false, nil
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
		return len(report.Blocked) == 0 && (report.EffectivePassed() || len(report.Completed) > 0)
	}
	return oldState.Implemented && report.EffectivePassed()
}
