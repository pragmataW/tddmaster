package phases

import (
	"fmt"
	"strings"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func RenderTaskList(tasks []spec.Task) string {
	if len(tasks) == 0 {
		return "No tasks"
	}
	var lines []string
	for _, task := range tasks {
		line := fmt.Sprintf("- %s: %s", task.ID, task.Title)
		if task.TDDEnabled {
			line += " (TDD)"
		}
		if task.Important {
			line += " (important)"
		}
		lines = append(lines, line)
		for _, c := range task.Criteria {
			lines = append(lines, "  - ["+c.ID+"]"+spec.FormatCriterionInline(c))
		}
	}
	return strings.Join(lines, "\n")
}

type refinementDriver struct{}

func RefinementDriver() engine.Driver {
	return &refinementDriver{}
}

func (d *refinementDriver) Next(c *engine.Context, ph *engine.PhaseDef) (engine.Action, bool) {
	if c.HasAnswer("refinement_approved") {
		return engine.Action{}, true
	}
	instr := promptregistry.MustInstruction(promptregistry.KeyRefinePrompt)
	return engine.Action{
		Action:      engine.ActionAsk,
		Instruction: instr + "\n\n" + RenderTaskList(c.Progress().Tasks),
		ExpectedInput: engine.ExpectedInput{
			Format:  engine.FormatFlag,
			Example: "approve",
		},
	}, false
}

func (d *refinementDriver) Submit(c *engine.Context, ph *engine.PhaseDef, answer []byte) (engine.Action, bool, error) {
	t := strings.TrimSpace(string(answer))
	if t == "approve" || t == "done" {
		if err := c.SetAnswer("refinement_approved", "approve"); err != nil {
			return engine.Action{}, false, err
		}
		return engine.Action{}, true, nil
	}
	return engine.Action{}, false, fmt.Errorf("refinement expects 'approve' or 'done', got %q", t)
}
