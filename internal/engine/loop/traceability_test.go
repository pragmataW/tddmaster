package loop

import (
	"encoding/json"
	"testing"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func seedLoopSpecForTrace(t *testing.T, root, slug string, task spec.Task) *engine.Context {
	t.Helper()
	st := spec.State{
		Version: 1,
		Slug:    slug,
		Phase:   "executing",
		Answers: map[string][]spec.Answer{},
	}
	if err := spec.SaveState(root, slug, st); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	pr := spec.Progress{
		Spec:      slug,
		Status:    spec.StatusDraft,
		Tasks:     []spec.Task{task},
		Execution: &spec.ExecState{TDDCycle: cycleRed},
	}
	if err := spec.SaveProgress(root, slug, pr); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	settings := spec.DefaultSettings()
	settings.TDDEnabled = true
	if err := spec.SaveSettings(root, slug, settings); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	defs := []engine.PhaseDef{
		{
			ID:     "executing",
			Driver: NewLoopDriver(),
		},
	}
	ctx, err := engine.Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return ctx
}

func marshalStageReport(t *testing.T, r StageReport) []byte {
	t.Helper()
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal StageReport: %v", err)
	}
	return b
}

func TestContextLoadTraceability_MissingFile_ReturnsEmpty(t *testing.T) {
	root := t.TempDir()
	slug := "trace-missing"
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	tr, err := ctx.LoadTraceability()
	if err != nil {
		t.Fatalf("LoadTraceability on missing file: %v", err)
	}
	if len(tr) != 0 {
		t.Fatalf("expected empty Traceability, got %v", tr)
	}
}

func TestContextSaveTraceability_RoundTrip(t *testing.T) {
	root := t.TempDir()
	slug := "trace-roundtrip"
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	tr := spec.Traceability{
		"testfile_test.go": []spec.TraceEntry{
			{FunctionName: "TestSomething", TaskID: "task-1", AC: []string{"ac1"}, EC: nil},
		},
	}

	if err := ctx.SaveTraceability(tr); err != nil {
		t.Fatalf("SaveTraceability: %v", err)
	}

	loaded, err := ctx.LoadTraceability()
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 key, got %d", len(loaded))
	}
	entries := loaded["testfile_test.go"]
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].FunctionName != "TestSomething" {
		t.Errorf("FunctionName: got %q, want %q", entries[0].FunctionName, "TestSomething")
	}
}

func TestValidateAndPersistTraceability_EmptyTraceability_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "val-empty"
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	report := StageReport{
		Passed:       true,
		Traceability: []TraceReportEntry{},
	}

	err := validateAndPersistTraceability(ctx, task, report)
	if err == nil {
		t.Fatal("expected error for empty Traceability, got nil")
	}
}

func TestValidateAndPersistTraceability_MissingTestFilePath_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "val-no-filepath"
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "", FunctionName: "TestFoo", AC: []string{"ac1"}},
		},
	}

	err := validateAndPersistTraceability(ctx, task, report)
	if err == nil {
		t.Fatal("expected error for missing TestFilePath, got nil")
	}
}

func TestValidateAndPersistTraceability_MissingFunctionName_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "val-no-funcname"
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "", AC: []string{"ac1"}},
		},
	}

	err := validateAndPersistTraceability(ctx, task, report)
	if err == nil {
		t.Fatal("expected error for missing FunctionName, got nil")
	}
}

func TestValidateAndPersistTraceability_BothACandECEmpty_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "val-no-ac-ec"
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", AC: []string{}, EC: []string{}},
		},
	}

	err := validateAndPersistTraceability(ctx, task, report)
	if err == nil {
		t.Fatal("expected error when both AC and EC are empty, got nil")
	}
}

func TestValidateAndPersistTraceability_ValidEntry_WithAC_NoError(t *testing.T) {
	root := t.TempDir()
	slug := "val-valid-ac"
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", TaskID: "task-1", AC: []string{"ac1"}},
		},
	}

	err := validateAndPersistTraceability(ctx, task, report)
	if err != nil {
		t.Fatalf("expected no error for valid entry with AC, got %v", err)
	}
}

func TestValidateAndPersistTraceability_ValidEntry_OnlyEC_NoError(t *testing.T) {
	root := t.TempDir()
	slug := "val-only-ec"
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", TaskID: "task-1", EC: []string{"EC-1"}},
		},
	}

	err := validateAndPersistTraceability(ctx, task, report)
	if err != nil {
		t.Fatalf("expected no error when only EC present, got %v", err)
	}
}

func TestValidateAndPersistTraceability_PersistHappyPath_EntriesWritten(t *testing.T) {
	root := t.TempDir()
	slug := "persist-happy"
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", TaskID: "task-1", AC: []string{"ac1"}},
			{TestFilePath: "foo_test.go", FunctionName: "TestBar", TaskID: "task-1", EC: []string{"EC-1"}},
		},
	}

	if err := validateAndPersistTraceability(ctx, task, report); err != nil {
		t.Fatalf("validateAndPersistTraceability: %v", err)
	}

	loaded, err := ctx.LoadTraceability()
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}
	entries := loaded["foo_test.go"]
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries under foo_test.go, got %d", len(entries))
	}
	found := false
	for _, e := range entries {
		if e.FunctionName == "TestFoo" {
			found = true
		}
	}
	if !found {
		t.Error("expected TestFoo in persisted entries")
	}
}

func TestValidateAndPersistTraceability_KeyIsTestFilePath(t *testing.T) {
	root := t.TempDir()
	slug := "key-is-filepath"
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "internal/foo/foo_test.go", FunctionName: "TestFoo", TaskID: "task-1", AC: []string{"ac1"}},
		},
	}

	if err := validateAndPersistTraceability(ctx, task, report); err != nil {
		t.Fatalf("validateAndPersistTraceability: %v", err)
	}

	loaded, err := ctx.LoadTraceability()
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}
	if _, ok := loaded["internal/foo/foo_test.go"]; !ok {
		t.Fatalf("expected key %q in loaded traceability, keys present: %v", "internal/foo/foo_test.go", loaded)
	}
}

func TestValidateAndPersistTraceability_EmptyTaskID_FilledFromTask(t *testing.T) {
	root := t.TempDir()
	slug := "fill-taskid"
	task := spec.Task{ID: "task-99", Title: "a task", TDDEnabled: true}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", TaskID: "", AC: []string{"ac1"}},
		},
	}

	if err := validateAndPersistTraceability(ctx, task, report); err != nil {
		t.Fatalf("validateAndPersistTraceability: %v", err)
	}

	loaded, err := ctx.LoadTraceability()
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}
	entries := loaded["foo_test.go"]
	if len(entries) == 0 {
		t.Fatal("expected persisted entries, got none")
	}
	if entries[0].TaskID != "task-99" {
		t.Errorf("expected TaskID %q filled from task, got %q", "task-99", entries[0].TaskID)
	}
}

func TestValidateAndPersistTraceability_MissingTraceFile_MergeWorks(t *testing.T) {
	root := t.TempDir()
	slug := "no-trace-file"
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "bar_test.go", FunctionName: "TestBar", TaskID: "task-1", AC: []string{"ac1"}},
		},
	}

	if err := validateAndPersistTraceability(ctx, task, report); err != nil {
		t.Fatalf("validateAndPersistTraceability on fresh dir: %v", err)
	}

	loaded, err := ctx.LoadTraceability()
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}
	if len(loaded["bar_test.go"]) != 1 {
		t.Fatalf("expected 1 entry after merge-write, got %d", len(loaded["bar_test.go"]))
	}
}

func TestValidateAndPersistTraceability_Dedup_SameFileAndFunc_NoDuplicate(t *testing.T) {
	root := t.TempDir()
	slug := "dedup-same"
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", TaskID: "task-1", AC: []string{"ac1"}},
		},
	}

	if err := validateAndPersistTraceability(ctx, task, report); err != nil {
		t.Fatalf("first persist: %v", err)
	}
	if err := validateAndPersistTraceability(ctx, task, report); err != nil {
		t.Fatalf("second persist: %v", err)
	}

	loaded, err := ctx.LoadTraceability()
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}
	count := 0
	for _, e := range loaded["foo_test.go"] {
		if e.FunctionName == "TestFoo" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 TestFoo entry after dedup, got %d", count)
	}
}

func TestValidateAndPersistTraceability_Dedup_LaterReplacesPrior(t *testing.T) {
	root := t.TempDir()
	slug := "dedup-replace"
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	first := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", TaskID: "task-1", AC: []string{"ac1"}},
		},
	}
	second := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", TaskID: "task-1", AC: []string{"ac1", "ac2"}},
		},
	}

	if err := validateAndPersistTraceability(ctx, task, first); err != nil {
		t.Fatalf("first persist: %v", err)
	}
	if err := validateAndPersistTraceability(ctx, task, second); err != nil {
		t.Fatalf("second persist: %v", err)
	}

	loaded, err := ctx.LoadTraceability()
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}
	count := 0
	for _, e := range loaded["foo_test.go"] {
		if e.FunctionName == "TestFoo" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 TestFoo entry (later replaces earlier), got %d", count)
	}
}

func TestSubmit_RedStage_EmptyTraceability_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "submit-red-empty-trace"
	tasks := []spec.Task{
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleRed}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)
	settings := spec.Settings{TDDEnabled: true}
	if err := spec.SaveSettings(root, slug, settings); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	ctx = seedLoopSpec(t, root, slug, tasks, execution)

	report := StageReport{
		Passed:       true,
		Traceability: []TraceReportEntry{},
	}

	_, submitErr := ctx.Submit(marshalStageReport(t, report))
	if submitErr == nil {
		t.Fatal("expected error for red stage with empty Traceability, got nil")
	}
}

func TestSubmit_RedStage_ValidTraceability_NoError(t *testing.T) {
	root := t.TempDir()
	slug := "submit-red-valid-trace"
	tasks := []spec.Task{
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleRed}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)
	settings := spec.Settings{TDDEnabled: true}
	if err := spec.SaveSettings(root, slug, settings); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	ctx = seedLoopSpec(t, root, slug, tasks, execution)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", TaskID: "t1", AC: []string{"ac1"}},
		},
	}

	_, submitErr := ctx.Submit(marshalStageReport(t, report))
	if submitErr != nil {
		t.Fatalf("expected no error for valid traceability in red stage, got: %v", submitErr)
	}
}

func TestSubmit_GreenStage_EmptyTraceability_NoError(t *testing.T) {
	root := t.TempDir()
	slug := "submit-green-empty-trace"
	tasks := []spec.Task{
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleGreen}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	report := StageReport{
		Completed:    []string{"impl"},
		Traceability: []TraceReportEntry{},
	}

	_, submitErr := ctx.Submit(marshalStageReport(t, report))
	if submitErr != nil {
		t.Fatalf("green stage must not enforce traceability; got error: %v", submitErr)
	}
}
