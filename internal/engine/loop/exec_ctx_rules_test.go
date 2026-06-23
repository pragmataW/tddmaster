package loop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func seedLoopSpecWithRules(t *testing.T, root, slug string, tasks []spec.Task, execution *spec.ExecState) *engine.Context {
	t.Helper()
	return seedLoopSpec(t, root, slug, tasks, execution)
}

func writeLoopRuleFile(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll %q: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("# rule"), 0o644); err != nil {
		t.Fatalf("WriteFile %q: %v", name, err)
	}
}

func buildContextWithRules(t *testing.T, root, slug string, tasks []spec.Task, execution *spec.ExecState) *engine.Context {
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
	settings := spec.DefaultSettings()
	settings.MinTestCoverage = 0
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

func TestBuildExecCtx_Rules_CopiedFromContext(t *testing.T) {
	root := t.TempDir()
	slug := "execctx-rules"

	rulesDir := paths.Rules(root)
	writeLoopRuleFile(t, filepath.Join(rulesDir, "executor"), "exec-rule.md")

	tasks := []spec.Task{
		{ID: "t1", Title: "task one", Done: false, TDDEnabled: false},
	}
	execution := &spec.ExecState{TDDCycle: cycleEmpty}
	ctx := buildContextWithRules(t, root, slug, tasks, execution)

	task := tasks[0]
	d := NewLoopDriver()
	execCtx := d.buildExecCtx(ctx, task, 0)

	ctxRules := ctx.Rules().For("executor")
	execCtxRules := execCtx.Rules.For("executor")

	if len(ctxRules) == 0 {
		t.Fatal("expected context to have executor rules loaded")
	}
	if len(execCtxRules) != len(ctxRules) {
		t.Fatalf("ExecCtx.Rules.For(executor) len=%d, want %d", len(execCtxRules), len(ctxRules))
	}
	for i, r := range ctxRules {
		if execCtxRules[i] != r {
			t.Errorf("ExecCtx.Rules.For(executor)[%d] = %q, want %q", i, execCtxRules[i], r)
		}
	}
}

func TestBuildExecCtx_Rules_EmptyWhenNoRulesDir(t *testing.T) {
	root := t.TempDir()
	slug := "execctx-no-rules"

	tasks := []spec.Task{
		{ID: "t1", Title: "task one", Done: false, TDDEnabled: false},
	}
	execution := &spec.ExecState{TDDCycle: cycleEmpty}
	ctx := buildContextWithRules(t, root, slug, tasks, execution)

	task := tasks[0]
	d := NewLoopDriver()
	execCtx := d.buildExecCtx(ctx, task, 0)

	got := execCtx.Rules.For("executor")
	if len(got) != 0 {
		t.Fatalf("expected empty ExecCtx.Rules when no rules dir, got %v", got)
	}
}

func TestBuildExecCtx_Rules_GlobalRuleVisibleForAllAgents(t *testing.T) {
	root := t.TempDir()
	slug := "execctx-global-rules"

	rulesDir := paths.Rules(root)
	writeLoopRuleFile(t, rulesDir, "all-agents.md")

	tasks := []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: false},
	}
	execution := &spec.ExecState{TDDCycle: cycleEmpty}
	ctx := buildContextWithRules(t, root, slug, tasks, execution)

	task := tasks[0]
	d := NewLoopDriver()
	execCtx := d.buildExecCtx(ctx, task, 0)

	for _, agent := range []string{"executor", "test-writer", "verifier", "planner"} {
		got := execCtx.Rules.For(agent)
		if len(got) == 0 {
			t.Errorf("ExecCtx.Rules.For(%q): expected global rule, got none", agent)
		}
	}
}

func TestBuildExecCtx_Rules_MatchesContextRulesExactly(t *testing.T) {
	root := t.TempDir()
	slug := "execctx-rules-match"

	rulesDir := paths.Rules(root)
	writeLoopRuleFile(t, rulesDir, "root-rule.md")
	writeLoopRuleFile(t, filepath.Join(rulesDir, "verifier"), "verifier-rule.md")

	tasks := []spec.Task{
		{ID: "t1", Title: "task", Done: false, TDDEnabled: false},
	}
	execution := &spec.ExecState{TDDCycle: cycleEmpty}
	ctx := buildContextWithRules(t, root, slug, tasks, execution)

	task := tasks[0]
	d := NewLoopDriver()
	execCtx := d.buildExecCtx(ctx, task, 0)

	for _, agent := range []string{"executor", "verifier", "test-writer", "planner"} {
		ctxRules := ctx.Rules().For(agent)
		execCtxRules := execCtx.Rules.For(agent)
		if len(ctxRules) != len(execCtxRules) {
			t.Errorf("agent %q: ctx has %d rules, ExecCtx has %d", agent, len(ctxRules), len(execCtxRules))
			continue
		}
		for i, r := range ctxRules {
			if execCtxRules[i] != r {
				t.Errorf("agent %q rule[%d]: ctx=%q execCtx=%q", agent, i, r, execCtxRules[i])
			}
		}
	}
}
