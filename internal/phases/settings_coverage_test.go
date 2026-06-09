package phases

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func TestSettingsPrompt_Example_ContainsMinTestCoverage(t *testing.T) {
	action := settingsPrompt()
	example := action.ExpectedInput.Example
	if !strings.Contains(example, `"minTestCoverage"`) {
		t.Fatalf("Example field does not contain minTestCoverage key; got: %s", example)
	}
}

func TestSettingsPrompt_Example_ContainsDefaultCoverageValue(t *testing.T) {
	action := settingsPrompt()
	example := action.ExpectedInput.Example
	if !strings.Contains(example, `"minTestCoverage":80`) {
		t.Fatalf("Example field does not contain minTestCoverage:80; got: %s", example)
	}
}

func TestSettingsPrompt_Example_IsValidJSON(t *testing.T) {
	action := settingsPrompt()
	example := action.ExpectedInput.Example
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(example), &m); err != nil {
		t.Fatalf("Example is not valid JSON: %v; got: %s", err, example)
	}
	if _, ok := m["minTestCoverage"]; !ok {
		t.Fatal("Example JSON does not have minTestCoverage key")
	}
}

func TestKeySettings_Instruction_MentionsCoverage(t *testing.T) {
	instr := promptregistry.MustInstruction(promptregistry.KeySettings)
	lower := strings.ToLower(instr)
	if !strings.Contains(lower, "coverage") {
		t.Fatalf("KeySettings instruction does not mention coverage; got: %s", instr)
	}
}

func TestKeySettings_Instruction_MentionsMinimumPercentage(t *testing.T) {
	instr := promptregistry.MustInstruction(promptregistry.KeySettings)
	lower := strings.ToLower(instr)
	if !strings.Contains(lower, "percentage") && !strings.Contains(lower, "percent") && !strings.Contains(lower, "%") {
		t.Fatalf("KeySettings instruction does not mention a percentage concept; got: %s", instr)
	}
}

func TestKeySettings_Instruction_MentionsZeroDisabled(t *testing.T) {
	instr := promptregistry.MustInstruction(promptregistry.KeySettings)
	if !strings.Contains(instr, "0") {
		t.Fatalf("KeySettings instruction does not mention 0 (disabled); got: %s", instr)
	}
	lower := strings.ToLower(instr)
	if !strings.Contains(lower, "disabled") && !strings.Contains(lower, "disable") {
		t.Fatalf("KeySettings instruction does not mention disabled; got: %s", instr)
	}
}

func TestKeySettings_Instruction_MentionsDefault80(t *testing.T) {
	instr := promptregistry.MustInstruction(promptregistry.KeySettings)
	if !strings.Contains(instr, "80") {
		t.Fatalf("KeySettings instruction does not mention default value 80; got: %s", instr)
	}
}

func TestSettingsDriver_Submit_OmittedMinTestCoverage_KeepsDefault80(t *testing.T) {
	root := t.TempDir()
	seedSettingsSpec(t, root, "s")
	ctx := buildSettingsCtx(t, root, "s")

	payload := []byte(`{"tddEnabled":true,"skipVerifierEnabled":false,"importantTaskGateEnabled":false}`)
	_, done, err := (&settingsDriver{}).Submit(ctx, nil, payload)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if !done {
		t.Fatal("Submit did not mark phase done")
	}

	got, err := spec.LoadSettings(root, "s")
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if got.MinTestCoverage != 80 {
		t.Fatalf("MinTestCoverage = %d, want 80 (default)", got.MinTestCoverage)
	}
}

func TestSettingsDriver_Submit_OverHundredMinTestCoverage_ClampedTo100(t *testing.T) {
	root := t.TempDir()
	seedSettingsSpec(t, root, "s")
	ctx := buildSettingsCtx(t, root, "s")

	payload := []byte(`{"tddEnabled":true,"skipVerifierEnabled":false,"importantTaskGateEnabled":false,"minTestCoverage":150}`)
	_, done, err := (&settingsDriver{}).Submit(ctx, nil, payload)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if !done {
		t.Fatal("Submit did not mark phase done")
	}

	got, err := spec.LoadSettings(root, "s")
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if got.MinTestCoverage != 100 {
		t.Fatalf("MinTestCoverage = %d, want 100 (clamped from 150)", got.MinTestCoverage)
	}
}

func TestSettingsDriver_Submit_NegativeMinTestCoverage_ClampedTo0(t *testing.T) {
	root := t.TempDir()
	seedSettingsSpec(t, root, "s")
	ctx := buildSettingsCtx(t, root, "s")

	payload := []byte(`{"tddEnabled":true,"skipVerifierEnabled":false,"importantTaskGateEnabled":false,"minTestCoverage":-5}`)
	_, done, err := (&settingsDriver{}).Submit(ctx, nil, payload)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if !done {
		t.Fatal("Submit did not mark phase done")
	}

	got, err := spec.LoadSettings(root, "s")
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if got.MinTestCoverage != 0 {
		t.Fatalf("MinTestCoverage = %d, want 0 (clamped from -5)", got.MinTestCoverage)
	}
}

func TestSettingsDriver_Submit_ExplicitZeroMinTestCoverage_RoundTripsToZero(t *testing.T) {
	root := t.TempDir()
	seedSettingsSpec(t, root, "s")
	ctx := buildSettingsCtx(t, root, "s")

	payload := []byte(`{"tddEnabled":true,"skipVerifierEnabled":false,"importantTaskGateEnabled":false,"minTestCoverage":0}`)
	_, done, err := (&settingsDriver{}).Submit(ctx, nil, payload)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if !done {
		t.Fatal("Submit did not mark phase done")
	}

	got, err := spec.LoadSettings(root, "s")
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if got.MinTestCoverage != 0 {
		t.Fatalf("MinTestCoverage = %d, want 0 (explicit zero disables gate)", got.MinTestCoverage)
	}
}
