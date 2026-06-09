package loop

import (
	"encoding/json"
	"testing"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func seedLoopSpecWithCoverageGate(t *testing.T, root, slug string, tasks []spec.Task, execution *spec.ExecState, minCoverage int) *engine.Context {
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
		Tasks:     tasks,
		Execution: execution,
	}
	if err := spec.SaveProgress(root, slug, pr); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	settings := spec.Settings{
		TDDEnabled:      true,
		MinTestCoverage: minCoverage,
	}
	if err := spec.SaveSettings(root, slug, settings); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	defs := []engine.PhaseDef{
		{ID: "executing", Driver: NewLoopDriver()},
	}
	ctx, err := engine.Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return ctx
}

func marshalVerifierReport(t *testing.T, r StageReport) []byte {
	t.Helper()
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal verifier report: %v", err)
	}
	return b
}

func TestPersistCoverage_MergesFileCoverageIntoTraceability(t *testing.T) {
	root := t.TempDir()
	slug := "persist-cov-merge"

	existing := spec.Traceability{
		Entries: map[string][]spec.TraceEntry{
			"foo_test.go": {{FunctionName: "TestFoo", TaskID: "t1", AC: []string{"AC-1"}}},
		},
		Coverage: map[string]int{
			"existing.go": 70,
		},
	}
	if err := spec.SaveTraceability(root, slug, existing); err != nil {
		t.Fatalf("SaveTraceability: %v", err)
	}

	ctx := seedLoopSpecWithCoverageGate(t, root, slug, []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: true},
	}, &spec.ExecState{TDDCycle: cycleGreen, Implemented: true}, 0)

	report := StageReport{
		Passed: true,
		FileCoverage: []FileCoverageEntry{
			{File: "a.go", Coverage: 85},
			{File: "b.go", Coverage: 40},
		},
	}

	if err := persistCoverage(ctx, report); err != nil {
		t.Fatalf("persistCoverage: %v", err)
	}

	tr, err := spec.LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}

	if tr.Coverage["a.go"] != 85 {
		t.Errorf("Coverage[a.go]: got %d, want 85", tr.Coverage["a.go"])
	}
	if tr.Coverage["b.go"] != 40 {
		t.Errorf("Coverage[b.go]: got %d, want 40", tr.Coverage["b.go"])
	}
	if tr.Coverage["existing.go"] != 70 {
		t.Errorf("Coverage[existing.go]: got %d, want 70 (must preserve prior entry)", tr.Coverage["existing.go"])
	}
	if len(tr.Entries["foo_test.go"]) == 0 {
		t.Error("Entries[foo_test.go]: must be preserved after persistCoverage")
	}
}

func TestPersistCoverage_OverwritesSameFile(t *testing.T) {
	root := t.TempDir()
	slug := "persist-cov-overwrite"

	existing := spec.Traceability{
		Entries:  map[string][]spec.TraceEntry{},
		Coverage: map[string]int{"a.go": 60},
	}
	if err := spec.SaveTraceability(root, slug, existing); err != nil {
		t.Fatalf("SaveTraceability: %v", err)
	}

	ctx := seedLoopSpecWithCoverageGate(t, root, slug, []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: true},
	}, &spec.ExecState{TDDCycle: cycleGreen, Implemented: true}, 0)

	report := StageReport{
		Passed:       true,
		FileCoverage: []FileCoverageEntry{{File: "a.go", Coverage: 90}},
	}

	if err := persistCoverage(ctx, report); err != nil {
		t.Fatalf("persistCoverage: %v", err)
	}

	tr, err := spec.LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}

	if tr.Coverage["a.go"] != 90 {
		t.Errorf("Coverage[a.go]: got %d, want 90 (overwrite of prior entry)", tr.Coverage["a.go"])
	}
}

func TestPersistCoverage_EmptyStore_CreatesMapWithoutPanic(t *testing.T) {
	root := t.TempDir()
	slug := "persist-cov-empty-ec1"

	ctx := seedLoopSpecWithCoverageGate(t, root, slug, []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: true},
	}, &spec.ExecState{TDDCycle: cycleGreen, Implemented: true}, 0)

	report := StageReport{
		Passed: true,
		FileCoverage: []FileCoverageEntry{
			{File: "x.go", Coverage: 75},
		},
	}

	if err := persistCoverage(ctx, report); err != nil {
		t.Fatalf("persistCoverage into empty store: %v", err)
	}

	tr, err := spec.LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}

	if tr.Coverage == nil {
		t.Fatal("Coverage map must not be nil after persistCoverage into empty store (EC-1)")
	}
	if tr.Coverage["x.go"] != 75 {
		t.Errorf("Coverage[x.go]: got %d, want 75", tr.Coverage["x.go"])
	}
}

func TestPersistCoverage_NilTraceabilityFile_NoPanic(t *testing.T) {
	root := t.TempDir()
	slug := "persist-cov-nil-file"

	ctx := seedLoopSpecWithCoverageGate(t, root, slug, []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: true},
	}, &spec.ExecState{TDDCycle: cycleGreen, Implemented: true}, 0)

	report := StageReport{
		Passed:       true,
		FileCoverage: []FileCoverageEntry{{File: "y.go", Coverage: 55}},
	}

	if err := persistCoverage(ctx, report); err != nil {
		t.Fatalf("persistCoverage with nil/missing file panicked or errored: %v", err)
	}

	tr, err := spec.LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}

	if tr.Coverage["y.go"] != 55 {
		t.Errorf("Coverage[y.go]: got %d, want 55", tr.Coverage["y.go"])
	}
}

func TestDriverSubmit_GreenVerifier_WithHighCoverage_UpdatesCoverageAndTaskDone(t *testing.T) {
	root := t.TempDir()
	slug := "driver-high-cov"
	tasks := []spec.Task{
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleGreen}
	ctx := seedLoopSpecWithCoverageGate(t, root, slug, tasks, execution, 80)

	if _, err := ctx.Submit(greenImpl(t)); err != nil {
		t.Fatalf("Submit green impl: %v", err)
	}

	verifierReport := marshalVerifierReport(t, StageReport{
		Passed: true,
		FileCoverage: []FileCoverageEntry{
			{File: "impl.go", Coverage: 90},
			{File: "helper.go", Coverage: 85},
		},
	})
	if _, err := ctx.Submit(verifierReport); err != nil {
		t.Fatalf("Submit verifier high coverage: %v", err)
	}

	tr, err := spec.LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}

	if tr.Coverage["impl.go"] != 90 {
		t.Errorf("Coverage[impl.go]: got %d, want 90", tr.Coverage["impl.go"])
	}
	if tr.Coverage["helper.go"] != 85 {
		t.Errorf("Coverage[helper.go]: got %d, want 85", tr.Coverage["helper.go"])
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if !pr.Tasks[0].Done {
		t.Error("task must be Done after verifier passes with coverage >= threshold")
	}
}

func TestDriverSubmit_GreenVerifier_WithLowCoverage_UpdatesCoverageButTaskNotDone(t *testing.T) {
	root := t.TempDir()
	slug := "driver-low-cov"
	tasks := []spec.Task{
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleGreen}
	ctx := seedLoopSpecWithCoverageGate(t, root, slug, tasks, execution, 80)

	if _, err := ctx.Submit(greenImpl(t)); err != nil {
		t.Fatalf("Submit green impl: %v", err)
	}

	verifierReport := marshalVerifierReport(t, StageReport{
		Passed: true,
		FileCoverage: []FileCoverageEntry{
			{File: "impl.go", Coverage: 50},
		},
	})
	if _, err := ctx.Submit(verifierReport); err != nil {
		t.Fatalf("Submit verifier low coverage: %v", err)
	}

	tr, err := spec.LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}

	if tr.Coverage["impl.go"] != 50 {
		t.Errorf("Coverage[impl.go]: got %d, want 50 (coverage must persist even when gate fires)", tr.Coverage["impl.go"])
	}

	pr, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if pr.Tasks[0].Done {
		t.Error("task must NOT be Done when coverage below threshold")
	}
	if pr.Execution == nil || pr.Execution.TDDCycle != cycleRed {
		cycle := ""
		if pr.Execution != nil {
			cycle = pr.Execution.TDDCycle
		}
		t.Errorf("TDDCycle: got %q, want %q — low coverage must drive cycle back to red", cycle, cycleRed)
	}
}

func TestDriverSubmit_GreenVerifier_NoCoverageInReport_NoTraceabilityWrite(t *testing.T) {
	root := t.TempDir()
	slug := "driver-no-cov-report"
	tasks := []spec.Task{
		{ID: "t1", Title: "tdd task", Done: false, TDDEnabled: true},
	}
	execution := &spec.ExecState{TDDCycle: cycleGreen}
	ctx := seedLoopSpecWithCoverageGate(t, root, slug, tasks, execution, 0)

	if _, err := ctx.Submit(greenImpl(t)); err != nil {
		t.Fatalf("Submit green impl: %v", err)
	}

	verifierReport := marshalVerifierReport(t, StageReport{
		Passed:       true,
		FileCoverage: nil,
	})
	if _, err := ctx.Submit(verifierReport); err != nil {
		t.Fatalf("Submit verifier with no coverage: %v", err)
	}

	tr, err := spec.LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}

	if len(tr.Coverage) != 0 {
		t.Errorf("Coverage: got %v, want empty — persistCoverage must not run when FileCoverage is empty", tr.Coverage)
	}
}

func TestValidateAndPersistTraceability_Regression_EmptyStore_PopulatesEntries(t *testing.T) {
	root := t.TempDir()
	slug := "trace-regression-ac1"

	task := spec.Task{ID: "t1", TDDEnabled: true}
	ctx := seedLoopSpecForTrace(t, root, slug, task)

	report := StageReport{
		Passed: true,
		Traceability: []TraceReportEntry{
			{TestFilePath: "foo_test.go", FunctionName: "TestFoo", TaskID: "t1", AC: []string{"AC-1"}},
		},
	}

	if _, err := ctx.Submit(marshalStageReport(t, report)); err != nil {
		t.Fatalf("Submit red report: %v", err)
	}

	tr, err := spec.LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}

	if tr.Entries == nil {
		t.Fatal("Entries must not be nil after persisting RED traceability into empty store (AC-1 regression)")
	}
	entries := tr.Entries["foo_test.go"]
	if len(entries) == 0 {
		t.Fatal("Entries[foo_test.go]: must be populated after persist (AC-1 regression)")
	}
	if entries[0].FunctionName != "TestFoo" {
		t.Errorf("Entries[foo_test.go][0].FunctionName: got %q, want %q", entries[0].FunctionName, "TestFoo")
	}
}
