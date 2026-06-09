package loop

import (
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/spec"
)

func makeRedCtxWithCoverage(lastCoverage map[string]int, minCoverage int) ExecCtx {
	task := spec.Task{
		ID:         "task-7",
		Title:      "task-7",
		TDDEnabled: true,
	}
	state := spec.ExecState{
		TDDCycle:     cycleRed,
		LastCoverage: lastCoverage,
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

func TestRedStage_Prompt_LowCoverageFile_IsListedWithPercentAndThreshold(t *testing.T) {
	ctx := makeRedCtxWithCoverage(map[string]int{"a.go": 50, "b.go": 90}, 80)
	action := redStage().Prompt(ctx)
	if !strings.Contains(action.Instruction, "a.go") {
		t.Error("red prompt: expected low-coverage file 'a.go' to be listed")
	}
	if !strings.Contains(action.Instruction, "50") {
		t.Error("red prompt: expected actual coverage '50' to appear for low file")
	}
	if !strings.Contains(action.Instruction, "80") {
		t.Error("red prompt: expected threshold '80' to appear in coverage feedback")
	}
}

func TestRedStage_Prompt_AboveThresholdFile_IsNotListed(t *testing.T) {
	ctx := makeRedCtxWithCoverage(map[string]int{"a.go": 50, "b.go": 90}, 80)
	action := redStage().Prompt(ctx)
	if strings.Contains(action.Instruction, "b.go") {
		t.Error("red prompt: 'b.go' at 90% must NOT be listed as low-coverage when threshold is 80")
	}
}

func TestRedStage_Prompt_BoundaryAtThreshold_NotListed_EC4(t *testing.T) {
	ctx := makeRedCtxWithCoverage(map[string]int{"c.go": 80}, 80)
	action := redStage().Prompt(ctx)
	if strings.Contains(action.Instruction, "c.go") {
		t.Error("red prompt (EC-4): 'c.go' at exactly 80 must NOT be listed when threshold is 80 (>= passes)")
	}
}

func TestRedStage_Prompt_OneBelowBoundary_IsListed_EC4(t *testing.T) {
	ctx := makeRedCtxWithCoverage(map[string]int{"d.go": 79}, 80)
	action := redStage().Prompt(ctx)
	if !strings.Contains(action.Instruction, "d.go") {
		t.Error("red prompt (EC-4): 'd.go' at 79 must be listed when threshold is 80 (79 < 80)")
	}
}

func TestRedStage_Prompt_NilLastCoverage_NoFeedback_AC3(t *testing.T) {
	ctx := makeRedCtxWithCoverage(nil, 80)
	action := redStage().Prompt(ctx)
	if strings.Contains(action.Instruction, "coverage") {
		t.Error("red prompt (AC-3): nil LastCoverage must produce no coverage-feedback text")
	}
}

func TestRedStage_Prompt_EmptyLastCoverage_NoFeedback_AC3(t *testing.T) {
	ctx := makeRedCtxWithCoverage(map[string]int{}, 80)
	action := redStage().Prompt(ctx)
	if strings.Contains(action.Instruction, "coverage") {
		t.Error("red prompt (AC-3): empty LastCoverage must produce no coverage-feedback text")
	}
}

func TestRedStage_Prompt_GateDisabled_NoFeedback(t *testing.T) {
	ctx := makeRedCtxWithCoverage(map[string]int{"e.go": 30}, 0)
	action := redStage().Prompt(ctx)
	if strings.Contains(action.Instruction, "e.go") {
		t.Error("red prompt: coverage gate disabled (MinTestCoverage=0) must not produce low-coverage feedback")
	}
}

func TestAppendCoverageFeedback_LowFiles_ListedWithPercentAndThreshold(t *testing.T) {
	var b strings.Builder
	ctx := makeRedCtxWithCoverage(map[string]int{"internal/foo.go": 50, "internal/bar.go": 90}, 80)
	appendCoverageFeedback(&b, ctx)
	result := b.String()
	if !strings.Contains(result, "internal/foo.go") {
		t.Error("appendCoverageFeedback: expected 'internal/foo.go' in output")
	}
	if !strings.Contains(result, "50") {
		t.Error("appendCoverageFeedback: expected '50' in output")
	}
	if !strings.Contains(result, "80") {
		t.Error("appendCoverageFeedback: expected threshold '80' in output")
	}
	if strings.Contains(result, "internal/bar.go") {
		t.Error("appendCoverageFeedback: 'internal/bar.go' at 90 must not appear")
	}
}

func TestAppendCoverageFeedback_BoundaryExact_NotListed(t *testing.T) {
	var b strings.Builder
	ctx := makeRedCtxWithCoverage(map[string]int{"c.go": 80}, 80)
	appendCoverageFeedback(&b, ctx)
	result := b.String()
	if strings.Contains(result, "c.go") {
		t.Error("appendCoverageFeedback (EC-4): 'c.go' at exactly threshold must not appear")
	}
}

func TestAppendCoverageFeedback_NilCoverage_Empty(t *testing.T) {
	var b strings.Builder
	ctx := makeRedCtxWithCoverage(nil, 80)
	appendCoverageFeedback(&b, ctx)
	if b.Len() != 0 {
		t.Errorf("appendCoverageFeedback: nil LastCoverage must write nothing, got %q", b.String())
	}
}

func TestAppendCoverageFeedback_GateDisabled_Empty(t *testing.T) {
	var b strings.Builder
	ctx := makeRedCtxWithCoverage(map[string]int{"e.go": 30}, 0)
	appendCoverageFeedback(&b, ctx)
	if b.Len() != 0 {
		t.Errorf("appendCoverageFeedback: gate disabled (MinTestCoverage=0) must write nothing, got %q", b.String())
	}
}
