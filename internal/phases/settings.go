package phases

import (
	"encoding/json"
	"fmt"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/spec"
)

const SettingsAnswerKey = "spec_settings"

type settingsDriver struct{}

func SettingsDriver() engine.Driver {
	return &settingsDriver{}
}

func settingsPrompt() engine.Action {
	instr := promptregistry.MustInstruction(promptregistry.KeySettings)
	d := spec.DefaultSettings()
	return engine.Action{
		Action:      engine.ActionAsk,
		Instruction: instr,
		MultiSelect: true,
		InteractiveOptions: []engine.InteractiveOption{
			{Label: "TDD (Red-Green-Refactor)", Description: "Enforce failing-test-first cycles per task. Default: ON."},
			{Label: "Skip verifier", Description: "Skip the independent verifier sub-agent after the green stage. Default: OFF."},
			{Label: "Important task gate", Description: "Pause tasks flagged important for a plan-first review before execution. Default: OFF."},
		},
		ExpectedInput: engine.ExpectedInput{
			Format:  engine.FormatJSON,
			Example: fmt.Sprintf(`{"tddEnabled":%t,"skipVerifierEnabled":%t,"importantTaskGateEnabled":%t}`, d.TDDEnabled, d.SkipVerifierEnabled, d.ImportantTaskGateEnabled),
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
		return engine.Action{}, false, fmt.Errorf("invalid JSON answer")
	}
	settings := spec.DefaultSettings()
	if err := json.Unmarshal(answer, &settings); err != nil {
		return engine.Action{}, false, fmt.Errorf("parse settings: %w", err)
	}
	if err := c.SaveSettings(settings); err != nil {
		return engine.Action{}, false, err
	}
	if err := c.SetAnswer(SettingsAnswerKey, string(answer)); err != nil {
		return engine.Action{}, false, err
	}
	return engine.Action{}, true, nil
}
