package engine

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/spec"
)

func buildContextForAccessors(t *testing.T) (*Context, string, string) {
	t.Helper()
	root := t.TempDir()
	slug := "accessor-spec"
	seedSpec(t, root, slug, spec.PhaseInitial)
	defs := []PhaseDef{makeOneStepPhase(PhaseID(spec.PhaseInitial), "q")}
	ctx, err := Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return ctx, root, slug
}

func TestContext_Progress_ReturnsSeedValue(t *testing.T) {
	ctx, _, _ := buildContextForAccessors(t)

	p := ctx.Progress()
	if p.Spec != "accessor-spec" {
		t.Fatalf("Progress().Spec = %q, want %q", p.Spec, "accessor-spec")
	}
	if p.Status != spec.StatusDraft {
		t.Fatalf("Progress().Status = %q, want %q", p.Status, spec.StatusDraft)
	}
}

func TestContext_SaveProgress_RoundTrips(t *testing.T) {
	ctx, root, slug := buildContextForAccessors(t)

	updated := spec.Progress{
		Spec:   slug,
		Status: spec.StatusExecuting,
		Tasks: []spec.Task{
			{ID: "t-1", Title: "first task", Criteria: []spec.Criterion{{ID: "ac-1", Then: "ac1"}}, Done: false},
		},
	}

	if err := ctx.SaveProgress(updated); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}

	got := ctx.Progress()
	if got.Status != spec.StatusExecuting {
		t.Fatalf("in-memory Progress().Status = %q, want %q", got.Status, spec.StatusExecuting)
	}
	if len(got.Tasks) != 1 {
		t.Fatalf("in-memory Progress().Tasks len = %d, want 1", len(got.Tasks))
	}

	persisted, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress after SaveProgress: %v", err)
	}
	if persisted.Status != spec.StatusExecuting {
		t.Fatalf("persisted Progress.Status = %q, want %q", persisted.Status, spec.StatusExecuting)
	}
	if len(persisted.Tasks) != 1 || persisted.Tasks[0].ID != "t-1" {
		t.Fatalf("persisted Progress.Tasks unexpected: %+v", persisted.Tasks)
	}
}

func TestContext_Settings_ReturnsSeedValue(t *testing.T) {
	ctx, _, _ := buildContextForAccessors(t)

	s := ctx.Settings()
	want := spec.DefaultSettings()
	if s != want {
		t.Fatalf("Settings() = %+v, want %+v", s, want)
	}
}

func TestContext_SaveSettings_RoundTrips(t *testing.T) {
	ctx, root, slug := buildContextForAccessors(t)

	updated := spec.Settings{
		TDDEnabled:               false,
		SkipVerifierEnabled:      true,
		ImportantTaskGateEnabled: true,
		MinTestCoverage:          90,
	}

	if err := ctx.SaveSettings(updated); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	got := ctx.Settings()
	if got != updated {
		t.Fatalf("in-memory Settings() = %+v, want %+v", got, updated)
	}

	persisted, err := spec.LoadSettings(root, slug)
	if err != nil {
		t.Fatalf("LoadSettings after SaveSettings: %v", err)
	}
	if persisted != updated {
		t.Fatalf("persisted Settings = %+v, want %+v", persisted, updated)
	}
}

func TestContext_MaxIteration_ReturnsPositiveValue(t *testing.T) {
	ctx, _, _ := buildContextForAccessors(t)

	max := ctx.MaxIteration()
	if max <= 0 {
		t.Fatalf("MaxIteration() = %d, want > 0", max)
	}
}

func TestContext_LoadTraceability_EmptyWhenNotSaved(t *testing.T) {
	ctx, _, _ := buildContextForAccessors(t)

	tr, err := ctx.LoadTraceability()
	if err != nil {
		t.Fatalf("LoadTraceability on fresh spec: %v", err)
	}
	if len(tr.Entries) != 0 {
		t.Fatalf("expected empty Entries on fresh spec, got %v", tr.Entries)
	}
}

func TestContext_SaveTraceability_RoundTrips(t *testing.T) {
	ctx, root, slug := buildContextForAccessors(t)

	tr := spec.Traceability{
		Entries: map[string][]spec.TraceEntry{
			"context_accessors_test.go": {
				{
					FunctionName: "TestContext_SaveTraceability_RoundTrips",
					TaskID:       "task-2",
					CriterionIDs: []string{"AC-3"},
					EC:           []string{},
				},
			},
		},
		Coverage: map[string]map[string]float64{
			"task-2": {"internal/engine/context.go": 82.5},
		},
	}

	if err := ctx.SaveTraceability(tr); err != nil {
		t.Fatalf("SaveTraceability: %v", err)
	}

	persisted, err := spec.LoadTraceability(root, slug)
	if err != nil {
		t.Fatalf("LoadTraceability after SaveTraceability: %v", err)
	}

	entries, ok := persisted.Entries["context_accessors_test.go"]
	if !ok || len(entries) == 0 {
		t.Fatalf("persisted Traceability missing expected file entry")
	}
	if entries[0].FunctionName != "TestContext_SaveTraceability_RoundTrips" {
		t.Fatalf("persisted FunctionName = %q, want %q", entries[0].FunctionName, "TestContext_SaveTraceability_RoundTrips")
	}
	if entries[0].TaskID != "task-2" {
		t.Fatalf("persisted TaskID = %q, want %q", entries[0].TaskID, "task-2")
	}
	if cov, exists := persisted.Coverage["task-2"]["internal/engine/context.go"]; !exists || cov != 82.5 {
		t.Fatalf("persisted Coverage unexpected: %v", persisted.Coverage)
	}

	reloaded, err := ctx.LoadTraceability()
	if err != nil {
		t.Fatalf("ctx.LoadTraceability after SaveTraceability: %v", err)
	}
	if _, ok := reloaded.Entries["context_accessors_test.go"]; !ok {
		t.Fatalf("ctx.LoadTraceability did not return saved entries")
	}
}
