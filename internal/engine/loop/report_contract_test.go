package loop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/promptregistry"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func TestNormalizeReportedPaths_StripsWorktreePrefix(t *testing.T) {
	wt := &spec.WorktreeRef{Path: ".tddmaster/worktrees/demo/task-1", Branch: "tddmaster/demo/task-1"}

	got := normalizeReportedPaths([]string{
		".tddmaster/worktrees/demo/task-1/calc/calc.go",
		"calc/calc_test.go",
		"./calc/calc.go",
	}, wt)

	want := []string{"calc/calc.go", "calc/calc_test.go", "calc/calc.go"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("path %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNormalizeReportedPaths_LeavesUnrelatedPathsAlone(t *testing.T) {
	wt := &spec.WorktreeRef{Path: ".tddmaster/worktrees/demo/task-1"}

	got := normalizeReportedPaths([]string{"internal/foo/bar.go"}, wt)

	if len(got) != 1 || got[0] != "internal/foo/bar.go" {
		t.Fatalf("got %v, want [internal/foo/bar.go]", got)
	}
	if out := normalizeReportedPaths([]string{"internal/foo/bar.go"}, nil); out[0] != "internal/foo/bar.go" {
		t.Fatalf("no worktree: got %v", out)
	}
}

// A worktree-prefixed path used to reach ExecState untouched, so the same file
// appeared twice in the CHANGED FILES section under two different names.
func TestSubmit_GreenReport_WorktreePrefixedPath_DoesNotDuplicateChangedFiles(t *testing.T) {
	root := t.TempDir()
	slug := "prefix-dedup"
	tasks := []spec.Task{
		{ID: "t1", Title: "tdd task", TDDEnabled: true, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}}},
	}
	ctx := seedLoopSpec(t, root, slug, tasks, &spec.ExecState{
		TDDCycle:  cycleRed,
		TestFiles: []string{"calc/calc_test.go"},
		Worktree:  &spec.WorktreeRef{Path: ".tddmaster/worktrees/" + slug + "/t1", Branch: "b"},
	})

	if _, err := ctx.Submit(marshalStageReport(t, StageReport{
		TaskID:        "t1",
		TestsWritten:  []string{"TestCalc"},
		FilesModified: []string{".tddmaster/worktrees/" + slug + "/t1/calc/calc_test.go"},
		Traceability: []TraceReportEntry{
			{TestFilePath: ".tddmaster/worktrees/" + slug + "/t1/calc/calc_test.go", FunctionName: "TestCalc", TaskID: "t1", AC: []string{"ac-1"}},
		},
	})); err != nil {
		t.Fatalf("Submit red: %v", err)
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	for _, f := range pr.Tasks[0].Exec.TestFiles {
		if strings.Contains(f, "worktrees") {
			t.Fatalf("worktree-prefixed path survived into TestFiles: %v", pr.Tasks[0].Exec.TestFiles)
		}
	}
	if len(pr.Tasks[0].Exec.TestFiles) != 1 {
		t.Fatalf("same file recorded twice: %v", pr.Tasks[0].Exec.TestFiles)
	}

	tr, err := spec.LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}
	if _, ok := tr.Entries["calc/calc_test.go"]; !ok {
		t.Fatalf("traceability key must be normalized, got keys %v", tr.Entries)
	}
}

// The raw decoder error named a Go struct field and a Go type, which tells a
// sub-agent nothing about the JSON it should have sent instead.
func TestReportShapeError_NamesTheFieldAndTheExpectedShape(t *testing.T) {
	var report StageReport
	err := json.Unmarshal([]byte(`{"refactorNotes":["extract validation"]}`), &report)
	if err == nil {
		t.Fatal("expected a decode error for a string refactorNotes entry")
	}

	msg := reportShapeError(err).Error()
	for _, want := range []string{"refactorNotes", "suggestion", "rationale"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message must mention %q, got %q", want, msg)
		}
	}
	if strings.Contains(msg, "spec.RefactorNote") {
		t.Fatalf("message must not leak the Go type, got %q", msg)
	}
}

func TestReportShapeError_UnknownFieldStillProducesAMessage(t *testing.T) {
	var report StageReport
	err := json.Unmarshal([]byte(`"not an object"`), &report)
	if err == nil {
		t.Fatal("expected a decode error")
	}
	if msg := reportShapeError(err).Error(); msg == "" {
		t.Fatal("expected a non-empty message")
	}
}

// `reason` on a passing verification is the only channel for a finding that is
// neither a failed criterion nor a refactor note. It used to be stored and
// never shown to anyone.
func TestSubmit_PassingVerifierWithReason_SurfacesItAsNotify(t *testing.T) {
	root := t.TempDir()
	slug := "verifier-note"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", TDDEnabled: false, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}}},
	}
	ctx := seedLoopSpec(t, root, slug, tasks, &spec.ExecState{Implemented: true})

	action, err := ctx.Submit(marshalStageReport(t, StageReport{
		TaskID: "t1",
		Passed: true,
		Reason: "project-commands-wrong-for-repo: npm run test fails with vitest: command not found",
	}))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if action.Action != engine.ActionNotify {
		t.Fatalf("expected a notify carrying the verifier note, got %q", action.Action)
	}
	if !strings.Contains(action.Instruction, "vitest: command not found") {
		t.Fatalf("notify must carry the reason verbatim, got %q", action.Instruction)
	}
	if !strings.Contains(action.Instruction, "t1") {
		t.Fatalf("notify must name the task, got %q", action.Instruction)
	}
}

func TestSubmit_PassingVerifierWithoutReason_DoesNotNotify(t *testing.T) {
	root := t.TempDir()
	slug := "verifier-quiet"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", TDDEnabled: false, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}}},
		{ID: "t2", Title: "other", TDDEnabled: false, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}}},
	}
	ctx := seedLoopSpec(t, root, slug, tasks, &spec.ExecState{Implemented: true})

	action, err := ctx.Submit(marshalStageReport(t, StageReport{TaskID: "t1", Passed: true}))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if action.Action == engine.ActionNotify {
		t.Fatalf("a silent pass must not interrupt the loop, got %q", action.Instruction)
	}
}

func TestSubmit_FailingVerifierWithReason_DoesNotNotify(t *testing.T) {
	root := t.TempDir()
	slug := "verifier-fail"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", TDDEnabled: false, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}}},
	}
	ctx := seedLoopSpec(t, root, slug, tasks, &spec.ExecState{Implemented: true})

	action, err := ctx.Submit(marshalStageReport(t, StageReport{
		TaskID:    "t1",
		Passed:    false,
		Reason:    "expected-pass-but-failed",
		FailedACs: []string{"ac-1"},
	}))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if action.Action == engine.ActionNotify {
		t.Fatalf("a failure already drives the cycle; no note expected, got %q", action.Instruction)
	}
}

// With TDD off there is no red stage, so a task that needs tests has nowhere to
// record their traceability. Accept it on the executor report when offered.
func TestSubmit_ExecutorStage_AcceptsOptionalTraceability(t *testing.T) {
	root := t.TempDir()
	slug := "executor-trace"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", TDDEnabled: false, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}}},
	}
	ctx := seedLoopSpec(t, root, slug, tasks, nil)

	if _, err := ctx.Submit(marshalStageReport(t, StageReport{
		TaskID:        "t1",
		Completed:     []string{"t1"},
		FilesModified: []string{"calc/add_test.go"},
		Traceability: []TraceReportEntry{
			{TestFilePath: "calc/add_test.go", FunctionName: "TestAdd", TaskID: "t1", AC: []string{"ac-1"}},
		},
	})); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	tr, err := spec.LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}
	entries := tr.Entries["calc/add_test.go"]
	if len(entries) != 1 || entries[0].FunctionName != "TestAdd" {
		t.Fatalf("executor traceability not persisted, got %v", tr.Entries)
	}
}

func TestSubmit_ExecutorStage_TraceabilityStaysOptional(t *testing.T) {
	root := t.TempDir()
	slug := "executor-no-trace"
	tasks := []spec.Task{
		{ID: "t1", Title: "task", TDDEnabled: false, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}}},
	}
	ctx := seedLoopSpec(t, root, slug, tasks, nil)

	if _, err := ctx.Submit(marshalStageReport(t, StageReport{
		TaskID:        "t1",
		Completed:     []string{"t1"},
		FilesModified: []string{"calc/add.go"},
	})); err != nil {
		t.Fatalf("an executor report with no tests must not require traceability: %v", err)
	}
}

func TestWorktreeInstructionBlock_CwdIsAbsoluteAndRootIsNamed(t *testing.T) {
	root := t.TempDir()
	wt := &spec.WorktreeRef{Path: ".tddmaster/worktrees/demo/task-1", Branch: "tddmaster/demo/task-1"}

	block := worktreeInstructionBlock(root, wt)

	if !strings.Contains(block, "projectRoot: "+root) {
		t.Fatalf("block must name the project root, got:\n%s", block)
	}
	if !strings.Contains(block, "cwd: "+root+"/.tddmaster/worktrees/demo/task-1") {
		t.Fatalf("cwd must be absolute, got:\n%s", block)
	}
	// A line that is nothing but `===` is not a header and confuses any
	// orchestrator that splits the prompt on header lines.
	for _, line := range strings.Split(block, "\n") {
		if strings.TrimSpace(line) == "===" {
			t.Fatalf("stray bare === terminator in worktree block:\n%s", block)
		}
	}
	if worktreeInstructionBlock(root, nil) != "" {
		t.Fatal("no worktree means no block")
	}
}

func TestWorktreeInstructionBlock_NestedProjectKeepsProjectRelativeCwd(t *testing.T) {
	gitRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(gitRoot, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	projectRoot := filepath.Join(gitRoot, "cmd")
	if err := os.Mkdir(projectRoot, 0o755); err != nil {
		t.Fatalf("mkdir nested project: %v", err)
	}
	wt := &spec.WorktreeRef{Path: ".tddmaster/worktrees/demo/task-1", Branch: "tddmaster/demo/task-1"}

	block := worktreeInstructionBlock(projectRoot, wt)

	wantCwd := filepath.Join(projectRoot, wt.Path, "cmd")
	if !strings.Contains(block, "projectRoot: "+projectRoot) {
		t.Fatalf("block must preserve the configured project root, got:\n%s", block)
	}
	if !strings.Contains(block, "cwd: "+wantCwd) {
		t.Fatalf("nested project cwd must point at the matching directory inside the checkout; want %q, got:\n%s", wantCwd, block)
	}
}

// Prose may name a section, but only a line that is exactly `=== NAME ===`
// starts one. A reference reproduced with its delimiters becomes a phantom
// boundary for anyone parsing the prompt.
func TestExecutionInstructions_DoNotReproduceSectionDelimitersInProse(t *testing.T) {
	for _, key := range []promptregistry.InstructionKey{
		promptregistry.KeyExecRed, promptregistry.KeyExecGreen, promptregistry.KeyExecRefactor,
		promptregistry.KeyExecRefactorApply, promptregistry.KeyExecRefactorSkipVerify,
		promptregistry.KeyExecExecutor, promptregistry.KeyExecExecutorSkipVerify,
		promptregistry.KeyExecVerifier, promptregistry.KeyExecGate,
	} {
		text := promptregistry.MustInstruction(key)
		for _, header := range promptregistry.AllSections {
			if strings.Contains(text, header) {
				t.Fatalf("%s reproduces the header %q inside prose; reference it by name instead", key, header)
			}
		}
	}
}
