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
	Modules  []ModuleDef
	progress *PhaseProgress
}

func (d *StepTableDriver) bootstrap() {
	if d.progress == nil {
		d.progress = &PhaseProgress{}
	}
	for _, m := range d.Modules {
		found := false
		for _, mp := range d.progress.Modules {
			if mp.Module == m.ID {
				found = true
				break
			}
		}
		if !found {
			mp := ModuleProgress{Module: m.ID}
			for _, s := range m.Steps {
				mp.Steps = append(mp.Steps, StepProgress{Step: s.ID, Answered: false})
			}
			d.progress.Modules = append(d.progress.Modules, mp)
		}
	}
}

func (d *StepTableDriver) traverseSteps(fn func(mi, si int, s StepDef) bool) {
	for mi, m := range d.Modules {
		for si, s := range m.Steps {
			if mi < len(d.progress.Modules) && si < len(d.progress.Modules[mi].Steps) {
				if fn(mi, si, s) {
					return
				}
			}
		}
	}
}

func (d *StepTableDriver) findFirstUnanswered() (modIdx int, stepIdx int, stepDef *StepDef, found bool) {
	d.traverseSteps(func(mi, si int, s StepDef) bool {
		if !d.progress.Modules[mi].Steps[si].Answered {
			copy := s
			modIdx = mi
			stepIdx = si
			stepDef = &copy
			found = true
			return true
		}
		return false
	})
	return
}

func (d *StepTableDriver) findFirstUnansweredStep() (*StepDef, bool) {
	_, _, stepDef, found := d.findFirstUnanswered()
	return stepDef, found
}

func (d *StepTableDriver) allAnswered() bool {
	result := true
	d.traverseSteps(func(mi, si int, s StepDef) bool {
		if !d.progress.Modules[mi].Steps[si].Answered {
			result = false
			return true
		}
		return false
	})
	return result
}

func (d *StepTableDriver) Next(c *Context, ph *PhaseDef) (Action, bool) {
	d.bootstrap()
	stepDef, found := d.findFirstUnansweredStep()
	if !found {
		return Action{}, true
	}
	return stepDef.Prompt(c), false
}

func (d *StepTableDriver) Submit(c *Context, ph *PhaseDef, answer []byte) (Action, bool, error) {
	d.bootstrap()
	mi, si, stepDef, found := d.findFirstUnanswered()
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
		if _, err := stepDef.Emit(answer); err != nil {
			return Action{}, false, err
		}
	}

	d.progress.Modules[mi].Steps[si].Answered = true
	d.progress.Modules[mi].Steps[si].Answer = answer

	phaseDone := d.allAnswered()
	return Action{}, phaseDone, nil
}
