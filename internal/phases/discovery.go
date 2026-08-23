package phases

import (
	"encoding/json"
	"strings"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/errs"
	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/spec"
)

type DiscoveryStep struct {
	ID       engine.StepID
	Key      string
	Prompt   func(mode string) engine.Action
	Validate func(answer []byte) error
}

func DiscoverySteps() []DiscoveryStep {
	steps := []DiscoveryStep{
		{
			ID:  "step-listen-first",
			Key: "listen_context",
			Prompt: func(mode string) engine.Action {
				instr := promptregistry.MustInstruction(promptregistry.KeyListenFirst)
				return engine.Action{
					Action:      engine.ActionAsk,
					Instruction: instr,
					ExpectedInput: engine.ExpectedInput{
						Format: engine.FormatText,
					},
				}
			},
		},
		{
			ID:  "step-mode-selection",
			Key: "mode",
			Prompt: func(mode string) engine.Action {
				instr := promptregistry.MustInstruction(promptregistry.KeyModeSelection)
				opts := make([]engine.InteractiveOption, len(promptregistry.ModeOptions))
				cmdMap := make(map[string]string, len(promptregistry.ModeOptions))
				for i, o := range promptregistry.ModeOptions {
					opts[i] = engine.InteractiveOption{Label: o.Label, Description: o.Description}
					cmdMap[o.Label] = o.ID
				}
				return engine.Action{
					Action:             engine.ActionAsk,
					Instruction:        instr,
					InteractiveOptions: opts,
					CommandMap:         cmdMap,
					ExpectedInput: engine.ExpectedInput{
						Format: engine.FormatText,
					},
				}
			},
		},
		{
			ID:  "step-premise-challenge",
			Key: "premises",
			Prompt: func(mode string) engine.Action {
				instr := promptregistry.MustInstruction(promptregistry.KeyPremiseChallenge)
				return engine.Action{
					Action:      engine.ActionAsk,
					Instruction: instr,
					ExpectedInput: engine.ExpectedInput{
						Format:  engine.FormatJSON,
						Example: promptregistry.ExamplePremises,
					},
				}
			},
		},
	}

	questionKeys := []string{
		"status_quo", "ambition", "reversibility",
		"user_impact", "verification", "scope_boundary", "edge_cases",
	}
	for _, key := range questionKeys {
		k := key
		steps = append(steps, DiscoveryStep{
			ID:  engine.StepID("step-q-" + k),
			Key: k,
			Prompt: func(mode string) engine.Action {
				instr := promptregistry.MustInstruction(promptregistry.KeyDiscoveryQuestion(k))
				if mode != "" {
					rules := promptregistry.ModeRules(mode)
					if len(rules) > 0 {
						instr += "\n\n" + strings.Join(rules, "\n")
					}
				}
				if k == "verification" {
					instr += "\n\n" + strings.Join(promptregistry.BuiltInExtras, "\n")
				}
				instr += "\n\n" + promptregistry.AskWithSuggestionsDirective
				return engine.Action{
					Action:      engine.ActionAsk,
					Instruction: instr,
					ExpectedInput: engine.ExpectedInput{
						Format: engine.FormatText,
					},
				}
			},
		})
	}

	steps = append(steps, DiscoveryStep{
		ID:  "step-synthesis",
		Key: "synthesis",
		Prompt: func(mode string) engine.Action {
			return engine.Action{
				Action:      engine.ActionAsk,
				Instruction: promptregistry.DiscoverySynthesisText,
				ExpectedInput: engine.ExpectedInput{
					Format: engine.FormatFlag,
				},
			}
		},
		Validate: func(answer []byte) error {
			if strings.TrimSpace(string(answer)) == "approve" {
				return nil
			}
			return errs.Newf(errs.KeyExpectedApprove, strings.TrimSpace(string(answer)))
		},
	})

	return steps
}

// RenderDiscoverySynthesis reproduces every answer the discovery phase collected
// so the approval step has something concrete to review. Without it the step asks
// the user to confirm a synthesis that was never shown.
func RenderDiscoverySynthesis(c *engine.Context) string {
	var b strings.Builder
	b.WriteString(promptregistry.DiscoverySynthesisHeader)

	if v := strings.TrimSpace(c.AnswerValue("listen_context")); v != "" {
		b.WriteString("\ncontext (verbatim):\n")
		b.WriteString(v)
		b.WriteString("\n")
	}
	if mode := strings.TrimSpace(c.AnswerValue("mode")); mode != "" {
		b.WriteString("\nmode: ")
		b.WriteString(mode)
		if desc := promptregistry.ModeDescription(mode); desc != "" {
			b.WriteString(" — ")
			b.WriteString(desc)
		}
		b.WriteString("\n")
	}

	b.WriteString("\nanswers:\n")
	for _, q := range promptregistry.Questions {
		answer := strings.TrimSpace(c.AnswerValue(q.ID))
		if answer == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(q.Text)
		b.WriteString("\n  ")
		b.WriteString(answer)
		b.WriteString("\n")
	}

	if premises := spec.ParseChallengedPremises(c.AnswerValue("premises")); len(premises) > 0 {
		b.WriteString("\npremises the user did NOT accept (these outrank any later assumption):\n")
		for _, p := range premises {
			b.WriteString("- ")
			b.WriteString(p)
			b.WriteString("\n")
		}
	}
	return b.String()
}

type discoveryDriver struct{}

func DiscoveryDriver() engine.Driver {
	return &discoveryDriver{}
}

func (d *discoveryDriver) Next(c *engine.Context, ph *engine.PhaseDef) (engine.Action, bool) {
	mode := c.AnswerValue("mode")
	for _, step := range DiscoverySteps() {
		if !c.HasAnswer(step.Key) {
			action := step.Prompt(mode)
			if step.Key == "synthesis" {
				action.Instruction += "\n" + RenderDiscoverySynthesis(c)
			}
			return action, false
		}
	}
	return engine.Action{}, true
}

func (d *discoveryDriver) Submit(c *engine.Context, ph *engine.PhaseDef, answer []byte) (engine.Action, bool, error) {
	mode := c.AnswerValue("mode")
	steps := DiscoverySteps()

	var current *DiscoveryStep
	for i := range steps {
		if !c.HasAnswer(steps[i].Key) {
			current = &steps[i]
			break
		}
	}
	if current == nil {
		return engine.Action{}, true, nil
	}

	promptAction := current.Prompt(mode)
	if promptAction.ExpectedInput.Format == engine.FormatJSON && !json.Valid(answer) {
		return engine.Action{}, false, errs.New(errs.KeyInvalidJSONAnswer)
	}

	if current.Validate != nil {
		if err := current.Validate(answer); err != nil {
			return engine.Action{}, false, err
		}
	}

	value := string(answer)
	if current.Key == "synthesis" {
		value = strings.TrimSpace(value)
	}

	if err := c.SetAnswer(current.Key, value); err != nil {
		return engine.Action{}, false, err
	}

	phaseDone := true
	for _, step := range steps {
		if !c.HasAnswer(step.Key) {
			phaseDone = false
			break
		}
	}

	return engine.Action{}, phaseDone, nil
}
