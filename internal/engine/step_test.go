package engine

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestRunResult_ZeroValue(t *testing.T) {
	var rr RunResult
	if rr.Done {
		t.Fatalf("zero RunResult.Done should be false")
	}
	if rr.Contribution != nil {
		t.Fatalf("zero RunResult.Contribution should be nil")
	}
}

func TestRunResult_Construct(t *testing.T) {
	raw := json.RawMessage(`{"key":"val"}`)
	rr := RunResult{
		Done:         true,
		Contribution: raw,
		Action: Action{
			Action:      ActionNotify,
			Instruction: "done",
		},
	}
	if !rr.Done {
		t.Fatalf("RunResult.Done should be true")
	}
	if string(rr.Contribution) != `{"key":"val"}` {
		t.Fatalf("RunResult.Contribution = %s", rr.Contribution)
	}
	if rr.Action.Action != ActionNotify {
		t.Fatalf("RunResult.Action.Action = %q", rr.Action.Action)
	}
}

func TestRunResult_NilContributionAllowed(t *testing.T) {
	rr := RunResult{
		Done:         false,
		Contribution: nil,
		Action:       Action{Action: ActionAsk, Instruction: "?"},
	}
	if rr.Contribution != nil {
		t.Fatalf("Contribution should be nil")
	}
}

func TestStepID_IsStringType(t *testing.T) {
	var id StepID = "step-1"
	if string(id) != "step-1" {
		t.Fatalf("StepID string conversion: got %q", string(id))
	}
}

func TestStepDef_ZeroValue(t *testing.T) {
	var sd StepDef
	if sd.ID != "" {
		t.Fatalf("zero StepDef.ID should be empty")
	}
	if sd.Prompt != nil {
		t.Fatalf("zero StepDef.Prompt should be nil")
	}
	if sd.Validate != nil {
		t.Fatalf("zero StepDef.Validate should be nil")
	}
	if sd.Emit != nil {
		t.Fatalf("zero StepDef.Emit should be nil")
	}
	if sd.Run != nil {
		t.Fatalf("zero StepDef.Run should be nil")
	}
}

func TestStepDef_Construct(t *testing.T) {
	sd := StepDef{
		ID: StepID("collect-target"),
		Prompt: func(c *Context) Action {
			return Action{Action: ActionAsk, Instruction: "Enter target"}
		},
		Validate: func(answer []byte) error {
			if len(answer) == 0 {
				return errors.New("empty answer")
			}
			return nil
		},
		Emit: func(answer []byte) error {
			return nil
		},
		Run: func(c *Context, decision []byte) (RunResult, error) {
			return RunResult{Done: true}, nil
		},
	}
	if sd.ID != StepID("collect-target") {
		t.Fatalf("StepDef.ID = %q", sd.ID)
	}
	if sd.Prompt == nil {
		t.Fatalf("StepDef.Prompt should not be nil")
	}
	if sd.Validate == nil {
		t.Fatalf("StepDef.Validate should not be nil")
	}
	if sd.Emit == nil {
		t.Fatalf("StepDef.Emit should not be nil")
	}
	if sd.Run == nil {
		t.Fatalf("StepDef.Run should not be nil")
	}
}

func TestStepDef_PromptReturnsAction(t *testing.T) {
	sd := StepDef{
		ID: StepID("step-a"),
		Prompt: func(c *Context) Action {
			return Action{Action: ActionInstruct, Instruction: "do this"}
		},
	}
	got := sd.Prompt(nil)
	if got.Action != ActionInstruct {
		t.Fatalf("Prompt returned action %q, want %q", got.Action, ActionInstruct)
	}
}

func TestStepDef_ValidateCalled(t *testing.T) {
	called := false
	sd := StepDef{
		ID: StepID("step-b"),
		Validate: func(answer []byte) error {
			called = true
			return nil
		},
	}
	_ = sd.Validate([]byte("answer"))
	if !called {
		t.Fatalf("Validate was not called")
	}
}

func TestStepProgress_ZeroValue(t *testing.T) {
	var sp StepProgress
	if sp.Step != "" {
		t.Fatalf("zero StepProgress.Step should be empty")
	}
	if sp.Answered {
		t.Fatalf("zero StepProgress.Answered should be false")
	}
	if sp.Answer != nil {
		t.Fatalf("zero StepProgress.Answer should be nil")
	}
}

func TestStepProgress_Construct(t *testing.T) {
	raw := json.RawMessage(`"yes"`)
	sp := StepProgress{
		Step:     StepID("collect-target"),
		Answered: true,
		Answer:   raw,
	}
	if sp.Step != StepID("collect-target") {
		t.Fatalf("StepProgress.Step = %q", sp.Step)
	}
	if !sp.Answered {
		t.Fatalf("StepProgress.Answered should be true")
	}
	if string(sp.Answer) != `"yes"` {
		t.Fatalf("StepProgress.Answer = %s", sp.Answer)
	}
}

func TestStepProgress_JSONRoundTrip(t *testing.T) {
	raw := json.RawMessage(`{"target":"example.com"}`)
	sp := StepProgress{
		Step:     StepID("step-1"),
		Answered: true,
		Answer:   raw,
	}
	b, err := json.Marshal(sp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got StepProgress
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Step != sp.Step {
		t.Fatalf("Step round-trip: got %q, want %q", got.Step, sp.Step)
	}
	if got.Answered != sp.Answered {
		t.Fatalf("Answered round-trip: got %v, want %v", got.Answered, sp.Answered)
	}
}

func TestStepProgress_UnansweredOmitsAnswer(t *testing.T) {
	sp := StepProgress{
		Step:     StepID("step-2"),
		Answered: false,
	}
	b, err := json.Marshal(sp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal map: %v", err)
	}
	if _, ok := m["answer"]; ok {
		t.Fatalf("answer key should be omitted when nil, but was present")
	}
}
