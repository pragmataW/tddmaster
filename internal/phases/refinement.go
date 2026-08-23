package phases

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/errs"
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
		if len(task.DependsOn) > 0 {
			line += " (depends on: " + strings.Join(task.DependsOn, ", ") + ")"
		}
		lines = append(lines, line)
		for _, c := range task.Criteria {
			lines = append(lines, "  - ["+c.ID+"]"+spec.FormatCriterionInline(c))
		}
		for i, ec := range task.EdgeCases {
			lines = append(lines, "  - ["+spec.EdgeCaseIDPrefix+strconv.Itoa(i+1)+"] edge case: "+ec)
		}
	}
	return strings.Join(lines, "\n")
}

// refinementApprovedKey marks the refinement phase as finished. The analysis
// phase clears it when the user sends the spec back here.
const refinementApprovedKey = "refinement_approved"

type refinementDriver struct{}

func RefinementDriver() engine.Driver {
	return &refinementDriver{}
}

func (d *refinementDriver) Next(c *engine.Context, ph *engine.PhaseDef) (engine.Action, bool) {
	if c.HasAnswer(refinementApprovedKey) {
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
		if err := spec.ValidateDAG(c.Progress().Tasks); err != nil {
			return engine.Action{}, false, err
		}
		if err := c.SetAnswer(refinementApprovedKey, "approve"); err != nil {
			return engine.Action{}, false, err
		}
		return engine.Action{}, true, nil
	}
	return engine.Action{}, false, errs.Newf(errs.KeyRefinementExpects, t)
}
