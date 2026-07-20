package phases

import (
	"encoding/json"
	"fmt"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/spec"
)

const SettingsAnswerKey = "spec_settings"

type settingsDriver struct{}

func SettingsDriver() engine.Driver {
	return &settingsDriver{}
}

func settingsInteractiveOptions() []engine.InteractiveOption {
	opts := make([]engine.InteractiveOption, len(promptregistry.SettingsOptions))
	for i, o := range promptregistry.SettingsOptions {
		opts[i] = engine.InteractiveOption{Label: o.Label, Description: o.Description}
	}
	return opts
}

func settingsPrompt() engine.Action {
	instr := promptregistry.MustInstruction(promptregistry.KeySettings)
	d := spec.DefaultSettings()
	return engine.Action{
		Action:      engine.ActionAsk,
		Instruction: instr,
		MultiSelect:        true,
		InteractiveOptions: settingsInteractiveOptions(),
		ExpectedInput: engine.ExpectedInput{
			Format:  engine.FormatJSON,
			Example: fmt.Sprintf(`{"tddEnabled":%t,"skipVerifierEnabled":%t,"importantTaskGateEnabled":%t,"minTestCoverage":%d,"ruleLearningEnabled":%t}`, d.TDDEnabled, d.SkipVerifierEnabled, d.ImportantTaskGateEnabled, d.MinTestCoverage, d.RuleLearningEnabled),
		},
	}
}

func (d *settingsDriver) Next(c *engine.Context, ph *engine.PhaseDef) (engine.Action, bool) {
	if !c.HasAnswer(SettingsAnswerKey) {
		return settingsPrompt(), false
	}
	return engine.Action{}, true
}

func (d *settingsDriver) Submit(c *engine.Context, ph *engine.PhaseDef, answer []byte) (engine.Action, bool, error) {
	if c.HasAnswer(SettingsAnswerKey) {
		return engine.Action{}, true, nil
	}
	if !json.Valid(answer) {
		return engine.Action{}, false, errs.New(errs.KeyInvalidJSONAnswer)
	}
	settings := spec.DefaultSettings()
	if err := json.Unmarshal(answer, &settings); err != nil {
		return engine.Action{}, false, errs.Wrap(errs.KeyParseSettings, err)
	}
	settings.ClampCoverage()
	if err := c.SaveSettings(settings); err != nil {
		return engine.Action{}, false, err
	}
	if err := c.SetAnswer(SettingsAnswerKey, string(answer)); err != nil {
		return engine.Action{}, false, err
	}
	return engine.Action{}, true, nil
}
