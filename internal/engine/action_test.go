package engine

import (
	"encoding/json"
	"testing"
)

func TestActionType_AskValue(t *testing.T) {
	if ActionAsk != ActionType("ask") {
		t.Fatalf("ActionAsk = %q, want %q", ActionAsk, "ask")
	}
}

func TestActionType_InstructValue(t *testing.T) {
	if ActionInstruct != ActionType("instruct") {
		t.Fatalf("ActionInstruct = %q, want %q", ActionInstruct, "instruct")
	}
}

func TestActionType_NotifyValue(t *testing.T) {
	if ActionNotify != ActionType("notify") {
		t.Fatalf("ActionNotify = %q, want %q", ActionNotify, "notify")
	}
}

func TestActionType_TerminalValue(t *testing.T) {
	if ActionTerminal != ActionType("terminal") {
		t.Fatalf("ActionTerminal = %q, want %q", ActionTerminal, "terminal")
	}
}

func TestActionType_ErrorValue(t *testing.T) {
	if ActionError != ActionType("error") {
		t.Fatalf("ActionError = %q, want %q", ActionError, "error")
	}
}

func TestInputFormat_JSONValue(t *testing.T) {
	if FormatJSON != InputFormat("json") {
		t.Fatalf("FormatJSON = %q, want %q", FormatJSON, "json")
	}
}

func TestInputFormat_TextValue(t *testing.T) {
	if FormatText != InputFormat("text") {
		t.Fatalf("FormatText = %q, want %q", FormatText, "text")
	}
}

func TestInputFormat_FlagValue(t *testing.T) {
	if FormatFlag != InputFormat("flag") {
		t.Fatalf("FormatFlag = %q, want %q", FormatFlag, "flag")
	}
}

func TestExpectedInput_ZeroValue(t *testing.T) {
	var ei ExpectedInput
	if ei.Format != "" {
		t.Fatalf("zero ExpectedInput.Format should be empty, got %q", ei.Format)
	}
	if ei.SubmitCmd != "" {
		t.Fatalf("zero ExpectedInput.SubmitCmd should be empty, got %q", ei.SubmitCmd)
	}
	if ei.Example != "" {
		t.Fatalf("zero ExpectedInput.Example should be empty, got %q", ei.Example)
	}
}

func TestExpectedInput_Construct(t *testing.T) {
	ei := ExpectedInput{
		Format:    FormatJSON,
		SubmitCmd: "tddmaster spec foo next --answer=<json>",
		Example:   `{"key":"value"}`,
	}
	if ei.Format != FormatJSON {
		t.Fatalf("Format = %q, want %q", ei.Format, FormatJSON)
	}
	if ei.SubmitCmd != "tddmaster spec foo next --answer=<json>" {
		t.Fatalf("SubmitCmd = %q", ei.SubmitCmd)
	}
}

func TestExpectedInput_JSONRoundTrip(t *testing.T) {
	ei := ExpectedInput{
		Format:    FormatText,
		SubmitCmd: "next",
		Example:   "example",
	}
	b, err := json.Marshal(ei)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got ExpectedInput
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Format != ei.Format || got.SubmitCmd != ei.SubmitCmd || got.Example != ei.Example {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", got, ei)
	}
}

func TestInteractiveOption_ZeroValue(t *testing.T) {
	var opt InteractiveOption
	if opt.Label != "" {
		t.Fatalf("zero InteractiveOption.Label should be empty, got %q", opt.Label)
	}
	if opt.Description != "" {
		t.Fatalf("zero InteractiveOption.Description should be empty, got %q", opt.Description)
	}
}

func TestInteractiveOption_Construct(t *testing.T) {
	opt := InteractiveOption{
		Label:       "approve",
		Description: "Approve the plan",
	}
	if opt.Label != "approve" {
		t.Fatalf("Label = %q, want %q", opt.Label, "approve")
	}
	if opt.Description != "Approve the plan" {
		t.Fatalf("Description = %q", opt.Description)
	}
}

func TestInteractiveOption_JSONRoundTrip(t *testing.T) {
	opt := InteractiveOption{Label: "accept", Description: "Accept it"}
	b, err := json.Marshal(opt)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got InteractiveOption
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Label != opt.Label || got.Description != opt.Description {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestAction_ZeroValue(t *testing.T) {
	var a Action
	if a.Action != "" {
		t.Fatalf("zero Action.Action should be empty, got %q", a.Action)
	}
	if a.Instruction != "" {
		t.Fatalf("zero Action.Instruction should be empty")
	}
}

func TestAction_Construct(t *testing.T) {
	a := Action{
		Action:      ActionAsk,
		Instruction: "What is the target?",
		MultiSelect: false,
		ExpectedInput: ExpectedInput{
			Format:    FormatText,
			SubmitCmd: "next --answer=<text>",
		},
		InteractiveOptions: []InteractiveOption{
			{Label: "opt1"},
		},
	}
	if a.Action != ActionAsk {
		t.Fatalf("Action = %q, want %q", a.Action, ActionAsk)
	}
	if len(a.InteractiveOptions) != 1 {
		t.Fatalf("InteractiveOptions len = %d, want 1", len(a.InteractiveOptions))
	}
}

func TestAction_JSONRoundTrip(t *testing.T) {
	a := Action{
		Action:        ActionNotify,
		Instruction:   "Phase complete.",
		DelegateAgent: "tddmaster-executor",
		CommandMap:    map[string]string{"yes": "next --answer=yes"},
		ExpectedInput: ExpectedInput{Format: FormatFlag},
	}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Action
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Action != a.Action {
		t.Fatalf("Action round-trip: got %q, want %q", got.Action, a.Action)
	}
	if got.DelegateAgent != a.DelegateAgent {
		t.Fatalf("DelegateAgent round-trip: got %q, want %q", got.DelegateAgent, a.DelegateAgent)
	}
	if got.CommandMap["yes"] != "next --answer=yes" {
		t.Fatalf("CommandMap round-trip failed: %v", got.CommandMap)
	}
}
