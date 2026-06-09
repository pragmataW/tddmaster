package engine

import (
	"errors"
	"testing"
)

var _ Driver = (*StepTableDriver)(nil)

func newDriverTestCtx(t *testing.T) *Context {
	t.Helper()
	root := t.TempDir()
	seedTempSpec(t, root, "driver-spec", "test")
	return buildCtxNoPhases(t, root, "driver-spec")
}

func TestStepTableDriver_ImplementsDriver(t *testing.T) {
	d := &StepTableDriver{}
	var _ Driver = d
}

func TestStepTableDriver_EmptyModules_ReturnsPhaseDone(t *testing.T) {
	d := &StepTableDriver{Modules: []ModuleDef{}}
	phaseDef := &PhaseDef{ID: "test"}
	action, phaseDone := d.Next(newDriverTestCtx(t), phaseDef)
	if !phaseDone {
		t.Fatalf("empty modules: phaseDone should be true, got false")
	}
	if action.Action == ActionAsk {
		t.Fatalf("empty modules: action should not be ActionAsk, got %q", action.Action)
	}
}

func TestStepTableDriver_SingleModuleSingleStep_ReturnsAskAction(t *testing.T) {
	wantInstruction := "What is the target?"
	step := StepDef{
		ID: StepID("step-1"),
		Prompt: func(c *Context) Action {
			return Action{Action: ActionAsk, Instruction: wantInstruction}
		},
	}
	d := &StepTableDriver{Modules: []ModuleDef{{ID: "mod-1", Steps: []StepDef{step}}}}
	phaseDef := &PhaseDef{ID: "test"}

	action, phaseDone := d.Next(newDriverTestCtx(t), phaseDef)
	if phaseDone {
		t.Fatalf("single unanswered step: phaseDone should be false")
	}
	if action.Action != ActionAsk {
		t.Fatalf("expected ActionAsk, got %q", action.Action)
	}
	if action.Instruction != wantInstruction {
		t.Fatalf("expected instruction %q, got %q", wantInstruction, action.Instruction)
	}
}

func TestStepTableDriver_Submit_AdvancesInOrder(t *testing.T) {
	validateCalled := []string{}
	emitCalled := []string{}

	makeStep := func(id StepID) StepDef {
		return StepDef{
			ID: id,
			Prompt: func(c *Context) Action {
				return Action{Action: ActionAsk, Instruction: string(id)}
			},
			Validate: func(answer []byte) error {
				validateCalled = append(validateCalled, string(id))
				return nil
			},
			Emit: func(answer []byte) error {
				emitCalled = append(emitCalled, string(id))
				return nil
			},
		}
	}

	step1 := makeStep(StepID("step-1"))
	step2 := makeStep(StepID("step-2"))
	d := &StepTableDriver{Modules: []ModuleDef{{ID: "mod-1", Steps: []StepDef{step1, step2}}}}
	phaseDef := &PhaseDef{ID: "test"}
	ctx := newDriverTestCtx(t)

	_, phaseDone, err := d.Submit(ctx, phaseDef, []byte(`"answer1"`))
	if err != nil {
		t.Fatalf("Submit step-1: unexpected error: %v", err)
	}
	if phaseDone {
		t.Fatalf("Submit step-1: phaseDone should be false after first of two steps")
	}

	action2, done2 := d.Next(ctx, phaseDef)
	if done2 {
		t.Fatalf("Next after step-1: phaseDone should be false")
	}
	if action2.Instruction != "step-2" {
		t.Fatalf("Next after step-1: expected step-2 prompt, got %q", action2.Instruction)
	}

	_, phaseDone2, err2 := d.Submit(ctx, phaseDef, []byte(`"answer2"`))
	if err2 != nil {
		t.Fatalf("Submit step-2: unexpected error: %v", err2)
	}
	if !phaseDone2 {
		t.Fatalf("Submit step-2: phaseDone should be true after last step")
	}

	if len(validateCalled) != 2 {
		t.Fatalf("Validate should have been called twice, got %d: %v", len(validateCalled), validateCalled)
	}
	if len(emitCalled) != 2 {
		t.Fatalf("Emit should have been called twice, got %d: %v", len(emitCalled), emitCalled)
	}
}

func TestStepTableDriver_Submit_PersistsAcrossRebuiltContext(t *testing.T) {
	makeStep := func(id StepID) StepDef {
		return StepDef{
			ID: id,
			Prompt: func(c *Context) Action {
				return Action{Action: ActionAsk, Instruction: string(id)}
			},
		}
	}
	d := &StepTableDriver{Modules: []ModuleDef{{ID: "mod-1", Steps: []StepDef{makeStep("step-1"), makeStep("step-2")}}}}
	phaseDef := &PhaseDef{ID: "test"}

	root := t.TempDir()
	seedTempSpec(t, root, "persist-spec", "test")
	ctx := buildCtxNoPhases(t, root, "persist-spec")

	if _, _, err := d.Submit(ctx, phaseDef, []byte(`"answer1"`)); err != nil {
		t.Fatalf("Submit step-1: unexpected error: %v", err)
	}

	rebuilt := buildCtxNoPhases(t, root, "persist-spec")
	action, done := d.Next(rebuilt, phaseDef)
	if done {
		t.Fatalf("Next after rebuild: phaseDone should be false")
	}
	if action.Instruction != "step-2" {
		t.Fatalf("Next after rebuild: expected step-2 prompt, got %q", action.Instruction)
	}
}

func TestStepTableDriver_Submit_ValidateError_StepRemainsUnanswered(t *testing.T) {
	step := StepDef{
		ID: StepID("step-v"),
		Prompt: func(c *Context) Action {
			return Action{Action: ActionAsk, Instruction: "validate-fail step"}
		},
		Validate: func(answer []byte) error {
			return errors.New("validation failed")
		},
		Emit: func(answer []byte) error {
			return nil
		},
	}
	d := &StepTableDriver{Modules: []ModuleDef{{ID: "mod-v", Steps: []StepDef{step}}}}
	phaseDef := &PhaseDef{ID: "test"}
	ctx := newDriverTestCtx(t)

	_, phaseDone, err := d.Submit(ctx, phaseDef, []byte(`"bad"`))
	if err == nil {
		t.Fatalf("Submit with failing Validate: expected non-nil error")
	}
	if phaseDone {
		t.Fatalf("Submit with failing Validate: phaseDone should be false")
	}

	if ctx.HasAnswer(stepAnswerKey(phaseDef, "mod-v", "step-v")) {
		t.Fatalf("step should remain unanswered after Validate failure")
	}
}

func TestStepTableDriver_Submit_InvalidJSON_WhenFormatJSON_ReturnsError(t *testing.T) {
	step := StepDef{
		ID: StepID("step-j"),
		Prompt: func(c *Context) Action {
			return Action{
				Action:      ActionAsk,
				Instruction: "json step",
				ExpectedInput: ExpectedInput{
					Format: FormatJSON,
				},
			}
		},
		Validate: func(answer []byte) error {
			return nil
		},
		Emit: func(answer []byte) error {
			return nil
		},
	}
	d := &StepTableDriver{Modules: []ModuleDef{{ID: "mod-j", Steps: []StepDef{step}}}}
	phaseDef := &PhaseDef{ID: "test"}
	ctx := newDriverTestCtx(t)

	_, phaseDone, err := d.Submit(ctx, phaseDef, []byte(`not-valid-json`))
	if err == nil {
		t.Fatalf("Submit with invalid JSON when FormatJSON: expected non-nil error")
	}
	if phaseDone {
		t.Fatalf("Submit with invalid JSON: phaseDone should be false")
	}
	if ctx.HasAnswer(stepAnswerKey(phaseDef, "mod-j", "step-j")) {
		t.Fatalf("step should remain unanswered after invalid JSON")
	}
}
