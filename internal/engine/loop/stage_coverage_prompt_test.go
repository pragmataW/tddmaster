package loop

import (
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/spec"
)

func makeGreenVerifierCtx(minCoverage int, touchedFiles []string, filesModified []string) ExecCtx {
	taskID := "task-5"
	task := spec.Task{
		ID:         taskID,
		Title:      taskID,
		TDDEnabled: true,
	}
	state := spec.ExecState{
		TDDCycle:          cycleGreen,
		Implemented:       true,
		LastModifiedFiles: filesModified,
		TaskPlans: map[string]spec.TaskPlan{
			taskID: {
				TaskID:       taskID,
				TouchedFiles: touchedFiles,
			},
		},
	}
	settings := spec.Settings{
		TDDEnabled:      true,
		MinTestCoverage: minCoverage,
	}
	return ExecCtx{
		Settings: settings,
		Task:     task,
		State:    state,
	}
}

func TestAppendCoverageRequirement_GreenTDDWithCoverage_ContainsCoverageText(t *testing.T) {
	var b strings.Builder
	ctx := makeGreenVerifierCtx(80, []string{"internal/foo/bar.go"}, nil)
	appendCoverageRequirement(&b, ctx)
	result := b.String()
	if !strings.Contains(result, "coverage") {
		t.Error("appendCoverageRequirement: expected 'coverage' in output when MinTestCoverage=80")
	}
}

func TestAppendCoverageRequirement_GreenTDDWithCoverage_ContainsThresholdNumber(t *testing.T) {
	var b strings.Builder
	ctx := makeGreenVerifierCtx(80, []string{"internal/foo/bar.go"}, nil)
	appendCoverageRequirement(&b, ctx)
	result := b.String()
	if !strings.Contains(result, "80") {
		t.Error("appendCoverageRequirement: expected MinTestCoverage value '80' in output")
	}
}

func TestAppendCoverageRequirement_GreenTDDWithCoverage_ContainsFileCoverageKey(t *testing.T) {
	var b strings.Builder
	ctx := makeGreenVerifierCtx(80, []string{"internal/foo/bar.go"}, nil)
	appendCoverageRequirement(&b, ctx)
	result := b.String()
	if !strings.Contains(result, "fileCoverage") {
		t.Error("appendCoverageRequirement: expected 'fileCoverage' in output")
	}
}

func TestAppendCoverageRequirement_GreenTDDWithCoverage_DoesNotHardcodeGoTest(t *testing.T) {
	var b strings.Builder
	ctx := makeGreenVerifierCtx(80, []string{"internal/foo/bar.go"}, nil)
	appendCoverageRequirement(&b, ctx)
	result := b.String()
	if strings.Contains(result, "go test") {
		t.Error("appendCoverageRequirement: must NOT hardcode 'go test'; prompt must be language-agnostic")
	}
}

func TestAppendCoverageRequirement_GreenTDDWithCoverage_ContainsTouchedFileName(t *testing.T) {
	var b strings.Builder
	touchedFile := "internal/engine/loop/stages.go"
	ctx := makeGreenVerifierCtx(80, []string{touchedFile}, nil)
	appendCoverageRequirement(&b, ctx)
	result := b.String()
	if !strings.Contains(result, touchedFile) {
		t.Errorf("appendCoverageRequirement: expected touched file %q in output", touchedFile)
	}
}

func TestAppendCoverageRequirement_GateDisabled_AppendsNothing(t *testing.T) {
	var b strings.Builder
	ctx := makeGreenVerifierCtx(0, []string{"internal/foo/bar.go"}, nil)
	appendCoverageRequirement(&b, ctx)
	result := b.String()
	if result != "" {
		t.Errorf("appendCoverageRequirement: expected empty output when MinTestCoverage=0, got %q", result)
	}
}

func TestVerifierStagePrompt_GreenTDD_ContainsCoverageText(t *testing.T) {
	ctx := makeGreenVerifierCtx(80, []string{"internal/foo/bar.go"}, nil)
	action := verifierStage().Prompt(ctx)
	if !strings.Contains(action.Instruction, "coverage") {
		t.Error("verifierStage.Prompt: expected 'coverage' in instruction for green TDD with MinTestCoverage=80")
	}
}

func TestVerifierStagePrompt_GreenTDD_ContainsThresholdAndFileCoverage(t *testing.T) {
	ctx := makeGreenVerifierCtx(80, []string{"internal/foo/bar.go"}, nil)
	action := verifierStage().Prompt(ctx)
	if !strings.Contains(action.Instruction, "80") {
		t.Error("verifierStage.Prompt: expected '80' in instruction for green TDD with MinTestCoverage=80")
	}
	if !strings.Contains(action.Instruction, "fileCoverage") {
		t.Error("verifierStage.Prompt: expected 'fileCoverage' in instruction for green TDD")
	}
}

func TestVerifierStagePrompt_GreenTDD_DoesNotHardcodeGoTest(t *testing.T) {
	ctx := makeGreenVerifierCtx(80, []string{"internal/foo/bar.go"}, nil)
	action := verifierStage().Prompt(ctx)
	if strings.Contains(action.Instruction, "go test") {
		t.Error("verifierStage.Prompt: must NOT hardcode 'go test'; prompt must be language-agnostic")
	}
}

func TestVerifierStagePrompt_GreenTDD_ContainsTouchedFiles(t *testing.T) {
	touchedFile := "internal/engine/loop/stages.go"
	ctx := makeGreenVerifierCtx(80, []string{touchedFile}, nil)
	action := verifierStage().Prompt(ctx)
	if !strings.Contains(action.Instruction, touchedFile) {
		t.Errorf("verifierStage.Prompt: expected touched file %q in instruction for green TDD", touchedFile)
	}
}

func TestVerifierStagePrompt_GateDisabled_NoCoverageText(t *testing.T) {
	ctx := makeGreenVerifierCtx(0, []string{"internal/foo/bar.go"}, nil)
	action := verifierStage().Prompt(ctx)
	if strings.Contains(action.Instruction, "fileCoverage") {
		t.Error("verifierStage.Prompt: must NOT contain 'fileCoverage' when MinTestCoverage=0")
	}
}

func TestVerifierStagePrompt_NonTDD_NoCoverageText(t *testing.T) {
	task := spec.Task{
		ID:         "task-5",
		Title:      "task-5",
		TDDEnabled: false,
	}
	state := spec.ExecState{
		Implemented: true,
	}
	settings := spec.Settings{
		TDDEnabled:      true,
		MinTestCoverage: 80,
	}
	ctx := ExecCtx{
		Settings: settings,
		Task:     task,
		State:    state,
	}
	action := verifierStage().Prompt(ctx)
	if strings.Contains(action.Instruction, "fileCoverage") {
		t.Error("verifierStage.Prompt: must NOT contain coverage text when task TDDEnabled=false")
	}
}

func TestVerifierStagePrompt_TDDDisabledGlobally_NoCoverageText(t *testing.T) {
	task := spec.Task{
		ID:         "task-5",
		Title:      "task-5",
		TDDEnabled: true,
	}
	state := spec.ExecState{
		Implemented: true,
		TDDCycle:    cycleGreen,
	}
	settings := spec.Settings{
		TDDEnabled:      false,
		MinTestCoverage: 80,
	}
	ctx := ExecCtx{
		Settings: settings,
		Task:     task,
		State:    state,
	}
	action := verifierStage().Prompt(ctx)
	if strings.Contains(action.Instruction, "fileCoverage") {
		t.Error("verifierStage.Prompt: must NOT contain coverage text when global TDDEnabled=false")
	}
}

func TestRefactorStagePrompt_VerifierBranch_NoCoverageText(t *testing.T) {
	taskID := "task-5"
	task := spec.Task{
		ID:         taskID,
		Title:      taskID,
		TDDEnabled: true,
	}
	state := spec.ExecState{
		TDDCycle:        cycleRefactor,
		Implemented:     true,
		RefactorApplied: true,
		TaskPlans: map[string]spec.TaskPlan{
			taskID: {
				TaskID:       taskID,
				TouchedFiles: []string{"internal/foo/bar.go"},
			},
		},
	}
	settings := spec.Settings{
		TDDEnabled:      true,
		MinTestCoverage: 80,
	}
	ctx := ExecCtx{
		Settings: settings,
		Task:     task,
		State:    state,
	}
	action := refactorStage().Prompt(ctx)
	if strings.Contains(action.Instruction, "fileCoverage") {
		t.Error("refactorStage.Prompt (verifier branch): must NOT contain coverage text")
	}
}

func TestVerifierStagePrompt_GreenTDD_DemandsMeasurementForFileCoverage_EC1(t *testing.T) {
	ctx := makeGreenVerifierCtx(80, []string{"internal/foo/bar.go"}, nil)
	action := verifierStage().Prompt(ctx)
	if !strings.Contains(action.Instruction, "fileCoverage") {
		t.Error("verifierStage.Prompt (EC-1): green TDD prompt must demand fileCoverage reporting; empty fileCoverage means coverageMet=false")
	}
}

func TestAppendCoverageRequirement_Unreported_AddsExplicitDemand(t *testing.T) {
	var b strings.Builder
	ctx := makeGreenVerifierCtx(80, []string{"internal/foo/bar.go"}, nil)
	st := ctx.State
	st.CoverageUnreported = true
	ctx.State = st
	appendCoverageRequirement(&b, ctx)
	result := strings.ToLower(b.String())
	if !strings.Contains(result, "no coverage measurements") {
		t.Errorf("appendCoverageRequirement: CoverageUnreported must add an explicit demand about the missing report; got %q", b.String())
	}
}

func TestAppendCoverageRequirement_NotUnreported_NoExtraDemand(t *testing.T) {
	var b strings.Builder
	ctx := makeGreenVerifierCtx(80, []string{"internal/foo/bar.go"}, nil)
	appendCoverageRequirement(&b, ctx)
	if strings.Contains(strings.ToLower(b.String()), "no coverage measurements") {
		t.Error("appendCoverageRequirement: must not add the unreported demand when CoverageUnreported is false")
	}
}

func TestAppendCoverageRequirement_NoPlan_FallsBackToLastModifiedFiles(t *testing.T) {
	var b strings.Builder
	modified := "internal/engine/loop/stage_coverage.go"
	ctx := makeGreenVerifierCtx(80, nil, []string{modified})
	appendCoverageRequirement(&b, ctx)
	result := b.String()
	if !strings.Contains(result, modified) {
		t.Errorf("appendCoverageRequirement: expected fallback to LastModifiedFiles %q when plan has no touched files; got %q", modified, result)
	}
}

func TestVerifierStagePrompt_GreenTDD_NoPlan_ContainsModifiedFiles(t *testing.T) {
	modified := "internal/engine/loop/stage_coverage.go"
	ctx := makeGreenVerifierCtx(80, nil, []string{modified})
	action := verifierStage().Prompt(ctx)
	if !strings.Contains(action.Instruction, modified) {
		t.Errorf("verifierStage.Prompt: gate-off path must list LastModifiedFiles %q; got %q", modified, action.Instruction)
	}
}

func TestAppendCoverageRequirement_PlanTakesPrecedenceOverModified(t *testing.T) {
	var b strings.Builder
	planned := "internal/planned.go"
	modified := "internal/modified.go"
	ctx := makeGreenVerifierCtx(80, []string{planned}, []string{modified})
	appendCoverageRequirement(&b, ctx)
	result := b.String()
	if !strings.Contains(result, planned) {
		t.Errorf("appendCoverageRequirement: expected plan file %q when plan present; got %q", planned, result)
	}
	if strings.Contains(result, modified) {
		t.Errorf("appendCoverageRequirement: must prefer plan over LastModifiedFiles; unexpected %q in %q", modified, result)
	}
}

func TestAppendCoverageRequirement_StatesVerifierIsSoleMeasurer(t *testing.T) {
	var b strings.Builder
	ctx := makeGreenVerifierCtx(80, []string{"internal/foo/bar.go"}, nil)
	appendCoverageRequirement(&b, ctx)
	result := strings.ToLower(b.String())
	if !strings.Contains(result, "verifier") {
		t.Errorf("appendCoverageRequirement: must state measurement belongs to the verifier sub-agent; got %q", b.String())
	}
	if !strings.Contains(result, "orchestrator") {
		t.Errorf("appendCoverageRequirement: must instruct the orchestrator not to measure itself; got %q", b.String())
	}
}
