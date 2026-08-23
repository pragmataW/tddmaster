package loop

import (
	"encoding/json"
	"strings"
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
	task.Exec = &spec.ExecState{TDDCycle: cycleRed}
	pr := spec.Progress{
		Spec:   slug,
		Status: spec.StatusDraft,
		Tasks:  []spec.Task{task},
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
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}, {ID: "ac-2", Then: "it still works"}}, EdgeCases: []string{"empty input"}}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	tr, err := ctx.LoadTraceability()
	if err != nil {
		t.Fatalf("LoadTraceability on missing file: %v", err)
	}
	if len(tr.Entries) != 0 {
		t.Fatalf("expected empty Traceability, got %v", tr)
	}
}

func TestContextSaveTraceability_RoundTrip(t *testing.T) {
	root := t.TempDir()
	slug := "trace-roundtrip"
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}, {ID: "ac-2", Then: "it still works"}}, EdgeCases: []string{"empty input"}}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	tr := spec.Traceability{
		Entries: map[string][]spec.TraceEntry{
			"testfile_test.go": {
				{FunctionName: "TestSomething", TaskID: "task-1", CriterionIDs: []string{"ac1"}, EC: nil},
			},
		},
	}

	if err := ctx.SaveTraceability(tr); err != nil {
		t.Fatalf("SaveTraceability: %v", err)
	}

	loaded, err := ctx.LoadTraceability()
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}
	if len(loaded.Entries) != 1 {
		t.Fatalf("expected 1 key, got %d", len(loaded.Entries))
	}
	entries := loaded.Entries["testfile_test.go"]
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
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}, {ID: "ac-2", Then: "it still works"}}, EdgeCases: []string{"empty input"}}
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
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}, {ID: "ac-2", Then: "it still works"}}, EdgeCases: []string{"empty input"}}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "", FunctionName: "TestFoo", AC: []string{"ac-1"}},
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
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}, {ID: "ac-2", Then: "it still works"}}, EdgeCases: []string{"empty input"}}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "", AC: []string{"ac-1"}},
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
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}, {ID: "ac-2", Then: "it still works"}}, EdgeCases: []string{"empty input"}}
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
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}, {ID: "ac-2", Then: "it still works"}}, EdgeCases: []string{"empty input"}}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", TaskID: "task-1", AC: []string{"ac-1"}},
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
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}, {ID: "ac-2", Then: "it still works"}}, EdgeCases: []string{"empty input"}}
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
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}, {ID: "ac-2", Then: "it still works"}}, EdgeCases: []string{"empty input"}}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", TaskID: "task-1", AC: []string{"ac-1"}},
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
	entries := loaded.Entries["foo_test.go"]
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
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}, {ID: "ac-2", Then: "it still works"}}, EdgeCases: []string{"empty input"}}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "internal/foo/foo_test.go", FunctionName: "TestFoo", TaskID: "task-1", AC: []string{"ac-1"}},
		},
	}

	if err := validateAndPersistTraceability(ctx, task, report); err != nil {
		t.Fatalf("validateAndPersistTraceability: %v", err)
	}

	loaded, err := ctx.LoadTraceability()
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}
	if _, ok := loaded.Entries["internal/foo/foo_test.go"]; !ok {
		t.Fatalf("expected key %q in loaded traceability, keys present: %v", "internal/foo/foo_test.go", loaded)
	}
}

func TestValidateAndPersistTraceability_EmptyTaskID_FilledFromTask(t *testing.T) {
	root := t.TempDir()
	slug := "fill-taskid"
	task := spec.Task{ID: "task-99", Title: "a task", TDDEnabled: true, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}}}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", TaskID: "", AC: []string{"ac-1"}},
		},
	}

	if err := validateAndPersistTraceability(ctx, task, report); err != nil {
		t.Fatalf("validateAndPersistTraceability: %v", err)
	}

	loaded, err := ctx.LoadTraceability()
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}
	entries := loaded.Entries["foo_test.go"]
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
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}, {ID: "ac-2", Then: "it still works"}}, EdgeCases: []string{"empty input"}}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "bar_test.go", FunctionName: "TestBar", TaskID: "task-1", AC: []string{"ac-1"}},
		},
	}

	if err := validateAndPersistTraceability(ctx, task, report); err != nil {
		t.Fatalf("validateAndPersistTraceability on fresh dir: %v", err)
	}

	loaded, err := ctx.LoadTraceability()
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}
	if len(loaded.Entries["bar_test.go"]) != 1 {
		t.Fatalf("expected 1 entry after merge-write, got %d", len(loaded.Entries["bar_test.go"]))
	}
}

func TestValidateAndPersistTraceability_Dedup_SameFileAndFunc_NoDuplicate(t *testing.T) {
	root := t.TempDir()
	slug := "dedup-same"
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}, {ID: "ac-2", Then: "it still works"}}, EdgeCases: []string{"empty input"}}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", TaskID: "task-1", AC: []string{"ac-1"}},
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
	for _, e := range loaded.Entries["foo_test.go"] {
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
	task := spec.Task{ID: "task-1", Title: "a task", TDDEnabled: true, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}, {ID: "ac-2", Then: "it still works"}}, EdgeCases: []string{"empty input"}}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	first := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", TaskID: "task-1", AC: []string{"ac-1"}},
		},
	}
	second := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", TaskID: "task-1", AC: []string{"ac-1", "ac-2"}},
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
	for _, e := range loaded.Entries["foo_test.go"] {
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
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}}},
	}
	execution := &spec.ExecState{TDDCycle: cycleRed}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)
	settings := spec.Settings{TDDEnabled: true}
	if err := spec.SaveSettings(root, slug, settings); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	ctx = seedLoopSpec(t, root, slug, tasks, execution)

	report := StageReport{
		TaskID:       "t1",
		Passed:       true,
		TestsWritten: []string{"TestFoo"},
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
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}}},
	}
	execution := &spec.ExecState{TDDCycle: cycleRed}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)
	settings := spec.Settings{TDDEnabled: true}
	if err := spec.SaveSettings(root, slug, settings); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	ctx = seedLoopSpec(t, root, slug, tasks, execution)

	report := StageReport{
		TaskID:        "t1",
		Passed:        true,
		TestsWritten:  []string{"TestFoo"},
		FilesModified: []string{"foo_test.go"},
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", TaskID: "t1", AC: []string{"ac-1"}},
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
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true, Criteria: []spec.Criterion{{ID: "ac-1", Then: "it works"}}},
	}
	execution := &spec.ExecState{TDDCycle: cycleGreen}
	ctx := seedLoopSpec(t, root, slug, tasks, execution)

	report := StageReport{
		TaskID:       "t1",
		Completed:    []string{"impl"},
		Traceability: []TraceReportEntry{},
	}

	_, submitErr := ctx.Submit(marshalStageReport(t, report))
	if submitErr != nil {
		t.Fatalf("green stage must not enforce traceability; got error: %v", submitErr)
	}
}

func tracedTask(id string) spec.Task {
	return spec.Task{
		ID:         id,
		Title:      "traced task",
		TDDEnabled: true,
		Criteria: []spec.Criterion{
			{ID: "ac-1", Then: "it works"},
			{ID: "ac-2", Then: "it still works"},
		},
		EdgeCases: []string{"empty input", "huge input"},
	}
}

func TestTraceability_UppercaseIDs_NormalizedToLowercase(t *testing.T) {
	root := t.TempDir()
	task := tracedTask("task-1")
	ctx := seedLoopSpecForTrace(t, root, "trace-upper", task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", TaskID: "task-1", AC: []string{"AC-1"}, EC: []string{"EC-2"}},
		},
	}
	if err := validateAndPersistTraceability(ctx, task, report); err != nil {
		t.Fatalf("uppercase ids must be accepted and normalized, got %v", err)
	}

	tr, err := ctx.LoadTraceability()
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}
	entries := tr.Entries["foo_test.go"]
	if len(entries) != 1 {
		t.Fatalf("expected one persisted entry, got %d", len(entries))
	}
	if got := entries[0].CriterionIDs; len(got) != 1 || got[0] != "ac-1" {
		t.Fatalf("criterion id must persist lowercase, got %v", got)
	}
	if got := entries[0].EC; len(got) != 1 || got[0] != "ec-2" {
		t.Fatalf("edge case id must persist lowercase, got %v", got)
	}
}

func TestTraceability_MalformedID_Rejected(t *testing.T) {
	root := t.TempDir()
	task := tracedTask("task-1")
	ctx := seedLoopSpecForTrace(t, root, "trace-malformed", task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", TaskID: "task-1", AC: []string{"ac1"}},
		},
	}
	err := validateAndPersistTraceability(ctx, task, report)
	if err == nil || !strings.Contains(err.Error(), "malformed id") {
		t.Fatalf("expected malformed-id error for %q, got %v", "ac1", err)
	}
}

func TestTraceability_UnknownID_Rejected(t *testing.T) {
	root := t.TempDir()
	task := tracedTask("task-1")
	ctx := seedLoopSpecForTrace(t, root, "trace-unknown", task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", TaskID: "task-1", AC: []string{"ac-9"}},
		},
	}
	err := validateAndPersistTraceability(ctx, task, report)
	if err == nil || !strings.Contains(err.Error(), "unknown id") {
		t.Fatalf("expected unknown-id error for ac-9, got %v", err)
	}
	if err != nil && !strings.Contains(err.Error(), "ac-1, ac-2") {
		t.Fatalf("error must list the ids this task defines, got %q", err.Error())
	}
}

func TestTraceability_CoverageRound_AllowsTestWithNoCriterion(t *testing.T) {
	root := t.TempDir()
	task := tracedTask("task-1")
	ctx := seedLoopSpecForTrace(t, root, "trace-coverage-round", task)

	settings := spec.DefaultSettings()
	settings.MinTestCoverage = 80
	if err := spec.SaveSettings(root, "trace-coverage-round", settings); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	ctx = seedLoopSpecForTrace(t, root, "trace-coverage-round", task)

	// The task is back in RED only because coverage fell short; a test that
	// protects pre-existing code maps to no ac-N or ec-N.
	task.Exec = &spec.ExecState{
		TDDCycle:     cycleRed,
		LastCoverage: map[string]float64{"internal/cart/cart.go": 71},
	}

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestTotal", TaskID: "task-1"},
		},
	}
	if err := validateAndPersistTraceability(ctx, task, report); err != nil {
		t.Fatalf("coverage round must accept an unmapped test, got %v", err)
	}
}

func TestTraceability_NoCoverageGap_StillRequiresAnID(t *testing.T) {
	root := t.TempDir()
	task := tracedTask("task-1")
	ctx := seedLoopSpecForTrace(t, root, "trace-no-gap", task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", TaskID: "task-1"},
		},
	}
	err := validateAndPersistTraceability(ctx, task, report)
	if err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("expected at-least-one-id error outside a coverage round, got %v", err)
	}
}
