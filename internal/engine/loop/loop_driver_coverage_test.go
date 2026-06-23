package loop

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func TestLoopDriver_Next_MaxIteration_ReturnsNotifyWithRestartText(t *testing.T) {
	root := t.TempDir()
	slug := "cov-next-maxiter"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleRed, Iteration: 15}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next returned error: %v", err)
	}
	if action.Action != engine.ActionNotify {
		t.Fatalf("expected ActionNotify when Iteration >= MaxIteration, got %q", action.Action)
	}
	if !strings.Contains(action.Instruction, promptregistry.RestartRecommendedText) {
		t.Fatalf("expected RestartRecommendedText in Instruction, got %q", action.Instruction)
	}
}

func TestLoopDriver_Next_MaxIteration_ResetsIterationToZero(t *testing.T) {
	root := t.TempDir()
	slug := "cov-next-maxiter-reset"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleRed, Iteration: 15}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	_, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next returned error: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Execution == nil {
		t.Fatal("expected non-nil Execution after MaxIteration notify")
	}
	if pr.Execution.Iteration != 0 {
		t.Fatalf("expected Iteration reset to 0, got %d", pr.Execution.Iteration)
	}
}

func TestLoopDriver_Submit_Continue_ResetsIterationToZero(t *testing.T) {
	root := t.TempDir()
	slug := "cov-submit-continue"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleRed, Iteration: 12}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	_, err := ctx.Submit([]byte("continue"))
	if err != nil {
		t.Fatalf("Submit(continue) returned error: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Execution == nil {
		t.Fatal("expected non-nil Execution after continue")
	}
	if pr.Execution.Iteration != 0 {
		t.Fatalf("expected Iteration=0 after continue, got %d", pr.Execution.Iteration)
	}
}

func TestLoopDriver_Submit_Continue_NilExecution_NoError(t *testing.T) {
	root := t.TempDir()
	slug := "cov-submit-continue-nil"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: false},
	}
	ctx := seedLoopSpec(t, root, slug, tasks, nil)

	_, err := ctx.Submit([]byte("continue"))
	if err != nil {
		t.Fatalf("Submit(continue) with nil Execution returned error: %v", err)
	}
}

func TestLoopDriver_Submit_EmptyAnswer_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "cov-submit-empty"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleRed}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	_, err := ctx.Submit([]byte{})
	if err == nil {
		t.Fatal("expected error for empty answer, got nil")
	}
}

func TestLoopDriver_Submit_InvalidJSON_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "cov-submit-invalid-json"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleRed}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	_, err := ctx.Submit([]byte("not-valid-json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestLoopDriver_Submit_MaxIteration_AfterProcessing_ReturnsRestartNotify(t *testing.T) {
	root := t.TempDir()
	slug := "cov-submit-maxiter-after"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleRed, Iteration: 14}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	report, err := json.Marshal(StageReport{
		Passed:       true,
		TestsWritten: []string{"t1_test.go"},
		Traceability: []TraceReportEntry{
			{TestFilePath: "t1_test.go", FunctionName: "TestT1", TaskID: "t1", AC: []string{"ac1"}},
		},
	})
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	action, submitErr := ctx.Submit(report)
	if submitErr != nil {
		t.Fatalf("Submit returned error: %v", submitErr)
	}
	if action.Action != engine.ActionNotify {
		t.Fatalf("expected ActionNotify when iteration reaches max after submit, got %q", action.Action)
	}
	if !strings.Contains(action.Instruction, promptregistry.RestartRecommendedText) {
		t.Fatalf("expected RestartRecommendedText in Instruction, got %q", action.Instruction)
	}
}

func TestLoopDriver_Submit_MaxIteration_ResetsIterationToZeroAfterProcessing(t *testing.T) {
	root := t.TempDir()
	slug := "cov-submit-maxiter-reset"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleRed, Iteration: 14}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	report, err := json.Marshal(StageReport{
		Passed:       true,
		TestsWritten: []string{"t1_test.go"},
		Traceability: []TraceReportEntry{
			{TestFilePath: "t1_test.go", FunctionName: "TestT1", TaskID: "t1", AC: []string{"ac1"}},
		},
	})
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	if _, submitErr := ctx.Submit(report); submitErr != nil {
		t.Fatalf("Submit returned error: %v", submitErr)
	}

	pr, loadErr := spec.LoadProgress(root, slug)
	if loadErr != nil {
		t.Fatalf("LoadProgress: %v", loadErr)
	}
	if pr.Execution == nil {
		t.Fatal("expected non-nil Execution after submit with max iteration")
	}
	if pr.Execution.Iteration != 0 {
		t.Fatalf("expected Iteration reset to 0 after max iteration notify, got %d", pr.Execution.Iteration)
	}
}

func TestLoopDriver_Next_AllTasksDone_ReturnsTerminalAndStatusCompleted(t *testing.T) {
	root := t.TempDir()
	slug := "cov-next-alltasksdone"
	tasks := []spec.Task{
		{ID: "t1", Title: "done task", Done: true},
		{ID: "t2", Title: "also done", Done: true},
	}
	execution := &spec.ExecState{Iteration: 3}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	action, err := ctx.Next()
	if err != nil {
		t.Fatalf("Next returned error: %v", err)
	}
	if action.Action != engine.ActionTerminal {
		t.Fatalf("expected ActionTerminal when all tasks done, got %q", action.Action)
	}

	pr, loadErr := spec.LoadProgress(root, slug)
	if loadErr != nil {
		t.Fatalf("LoadProgress: %v", loadErr)
	}
	if pr.Status != spec.StatusCompleted {
		t.Fatalf("expected Status=%q, got %q", spec.StatusCompleted, pr.Status)
	}
}

func TestLoopDriver_Submit_AllTasksDoneAtSubmitTime_ReturnsTerminal(t *testing.T) {
	root := t.TempDir()
	slug := "cov-submit-alltasksdone"
	tasks := []spec.Task{
		{ID: "t1", Title: "done task", Done: true},
	}
	execution := &spec.ExecState{Iteration: 1}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	report, err := json.Marshal(StageReport{Passed: true})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	action, submitErr := ctx.Submit(report)
	if submitErr != nil {
		t.Fatalf("Submit returned error: %v", submitErr)
	}
	if action.Action != engine.ActionTerminal {
		t.Fatalf("expected ActionTerminal when all tasks done at submit time, got %q", action.Action)
	}
}
