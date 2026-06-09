package engine

import (
	"encoding/json"
	"errors"
)

type Driver interface {
	Next(c *Context, ph *PhaseDef) (Action, bool)
	Submit(c *Context, ph *PhaseDef, answer []byte) (Action, bool, error)
}

type StepTableDriver struct {
	Modules []ModuleDef
}

func stepAnswerKey(ph *PhaseDef, mod ModuleID, step StepID) string {
	phase := ""
	if ph != nil {
		phase = string(ph.ID)
	}
	return "step:" + phase + ":" + string(mod) + ":" + string(step)
}

func (d *StepTableDriver) findFirstUnanswered(c *Context, ph *PhaseDef) (StepDef, string, bool) {
	for _, m := range d.Modules {
		for _, s := range m.Steps {
			key := stepAnswerKey(ph, m.ID, s.ID)
			if !c.HasAnswer(key) {
				return s, key, true
			}
		}
	}
	return StepDef{}, "", false
}

func (d *StepTableDriver) Next(c *Context, ph *PhaseDef) (Action, bool) {
	stepDef, _, found := d.findFirstUnanswered(c, ph)
	if !found {
		return Action{}, true
	}
	return stepDef.Prompt(c), false
}

func (d *StepTableDriver) Submit(c *Context, ph *PhaseDef, answer []byte) (Action, bool, error) {
	stepDef, key, found := d.findFirstUnanswered(c, ph)
	if !found {
		return Action{}, true, nil
	}

	promptAction := stepDef.Prompt(c)
	if promptAction.ExpectedInput.Format == FormatJSON {
		if !json.Valid(answer) {
			return Action{}, false, errors.New("invalid JSON answer")
		}
	}

	if stepDef.Validate != nil {
		if err := stepDef.Validate(answer); err != nil {
			return Action{}, false, err
		}
	}

	if stepDef.Emit != nil {
		if err := stepDef.Emit(answer); err != nil {
			return Action{}, false, err
		}
	}

	if err := c.SetAnswer(key, string(answer)); err != nil {
		return Action{}, false, err
	}

	_, _, stillUnanswered := d.findFirstUnanswered(c, ph)
	return Action{}, !stillUnanswered, nil
}
