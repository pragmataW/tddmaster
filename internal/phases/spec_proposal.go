package phases

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/spec"
)

type TaskGenItem struct {
	Title           string   `json:"title"`
	AC              []string `json:"ac"`
	LinkedEdgeCases []string `json:"linkedEdgeCases,omitempty"`
}

type TaskGenPayload struct {
	Tasks []TaskGenItem `json:"tasks"`
}

func BuildTasksFromGen(p TaskGenPayload, tddDefault bool, fallbackEdgeCases []string) ([]spec.Task, error) {
	if len(p.Tasks) == 0 {
		return nil, fmt.Errorf("at least one task required")
	}
	tasks := make([]spec.Task, len(p.Tasks))
	for i, item := range p.Tasks {
		if item.Title == "" {
			return nil, fmt.Errorf("task %d: title is required", i+1)
		}
		if len(item.AC) == 0 {
			return nil, fmt.Errorf("task %d: at least one acceptance criterion required", i+1)
		}
		ec := item.LinkedEdgeCases
		if len(ec) == 0 {
			ec = fallbackEdgeCases
		}
		tasks[i] = spec.Task{
			ID:         fmt.Sprintf("task-%d", i+1),
			Title:      item.Title,
			AC:         item.AC,
			Done:       false,
			Important:  false,
			TDDEnabled: tddDefault,
			EdgeCases:  ec,
		}
	}
	return tasks, nil
}

type specProposalDriver struct{}

func SpecProposalDriver() engine.Driver {
	return &specProposalDriver{}
}

func taskGenPrompt() engine.Action {
	instr := promptregistry.MustInstruction(promptregistry.KeySpecTaskGen)
	return engine.Action{
		Action:      engine.ActionAsk,
		Instruction: instr,
		ExpectedInput: engine.ExpectedInput{
			Format:  engine.FormatJSON,
			Example: `{"tasks":[{"title":"...","ac":["..."],"linkedEdgeCases":["..."]}]}`,
		},
	}
}

func selfReviewPrompt() engine.Action {
	instr := promptregistry.MustInstruction(promptregistry.KeySelfReview)
	return engine.Action{
		Action:      engine.ActionAsk,
		Instruction: instr,
		ExpectedInput: engine.ExpectedInput{
			Format:  engine.FormatFlag,
			Example: "approve",
		},
	}
}

func (d *specProposalDriver) Next(c *engine.Context, ph *engine.PhaseDef) (engine.Action, bool) {
	if !c.HasAnswer("tasks_generated") {
		return taskGenPrompt(), false
	}
	if !c.HasAnswer("self_review") {
		return selfReviewPrompt(), false
	}
	return engine.Action{}, true
}

func (d *specProposalDriver) Submit(c *engine.Context, ph *engine.PhaseDef, answer []byte) (engine.Action, bool, error) {
	if !c.HasAnswer("tasks_generated") {
		if !json.Valid(answer) {
			return engine.Action{}, false, fmt.Errorf("invalid JSON answer")
		}
		var payload TaskGenPayload
		if err := json.Unmarshal(answer, &payload); err != nil {
			return engine.Action{}, false, err
		}
		fallback := spec.ParseEdgeCases(c.AnswerValue("edge_cases"))
		tasks, err := BuildTasksFromGen(payload, c.Settings().TDDEnabled, fallback)
		if err != nil {
			return engine.Action{}, false, err
		}
		progress := c.Progress()
		progress.Tasks = tasks
		progress.Status = spec.StatusDraft
		if err := c.SaveProgress(progress); err != nil {
			return engine.Action{}, false, err
		}
		content := spec.RenderSpecMd(c.Slug(), c.State(), progress)
		if err := c.WriteSpecMd(content); err != nil {
			return engine.Action{}, false, err
		}
		if err := c.SetAnswer("tasks_generated", string(answer)); err != nil {
			return engine.Action{}, false, err
		}
		return engine.Action{}, false, nil
	}

	if !c.HasAnswer("self_review") {
		if strings.TrimSpace(string(answer)) != "approve" {
			return engine.Action{}, false, fmt.Errorf("self-review requires approve")
		}
		if err := c.SetAnswer("self_review", "approve"); err != nil {
			return engine.Action{}, false, err
		}
		return engine.Action{}, true, nil
	}

	return engine.Action{}, true, nil
}
