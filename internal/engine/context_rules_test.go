package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func seedSpecForRules(t *testing.T, root, slug string) {
	t.Helper()
	seedSpec(t, root, slug, spec.PhaseInitial)
}

func writeRuleFile(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll %q: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("# rule"), 0o644); err != nil {
		t.Fatalf("WriteFile %q: %v", name, err)
	}
}

func TestContext_Rules_ReturnsRulesLoadedFromRoot(t *testing.T) {
	root := t.TempDir()
	slug := "rules-wired"
	seedSpecForRules(t, root, slug)

	rulesDir := paths.Rules(root)
	writeRuleFile(t, rulesDir, "global.md")
	writeRuleFile(t, filepath.Join(rulesDir, "executor"), "exec-rule.md")

	defs := []PhaseDef{makeOneStepPhase(PhaseID(spec.PhaseInitial), "q")}
	ctx, err := Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	got := ctx.Rules().For("executor")
	if len(got) == 0 {
		t.Fatal("expected Rules().For(executor) to return at least one path, got none")
	}
}

func TestContext_Rules_GlobalRuleAppearsForAllAgents(t *testing.T) {
	root := t.TempDir()
	slug := "rules-global"
	seedSpecForRules(t, root, slug)

	rulesDir := paths.Rules(root)
	writeRuleFile(t, rulesDir, "all-agents.md")

	defs := []PhaseDef{makeOneStepPhase(PhaseID(spec.PhaseInitial), "q")}
	ctx, err := Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	for _, agent := range []string{"executor", "test-writer", "verifier", "planner"} {
		got := ctx.Rules().For(agent)
		if len(got) == 0 {
			t.Errorf("agent %q: expected global rule to appear, got none", agent)
		}
	}
}

func TestContext_Rules_AgentSpecificRuleExcludedFromOtherAgents(t *testing.T) {
	root := t.TempDir()
	slug := "rules-agent-specific"
	seedSpecForRules(t, root, slug)

	rulesDir := paths.Rules(root)
	writeRuleFile(t, filepath.Join(rulesDir, "executor"), "exec-only.md")

	defs := []PhaseDef{makeOneStepPhase(PhaseID(spec.PhaseInitial), "q")}
	ctx, err := Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	executorRules := ctx.Rules().For("executor")
	if len(executorRules) == 0 {
		t.Fatal("expected executor-specific rule to appear for executor agent")
	}

	verifierRules := ctx.Rules().For("verifier")
	if len(verifierRules) != 0 {
		t.Fatalf("expected no rules for verifier agent when only executor rule exists, got %v", verifierRules)
	}
}

func TestContext_Rules_NoRulesDir_ReturnsEmptySetAndBuildSucceeds(t *testing.T) {
	root := t.TempDir()
	slug := "rules-missing-dir"
	seedSpecForRules(t, root, slug)

	defs := []PhaseDef{makeOneStepPhase(PhaseID(spec.PhaseInitial), "q")}
	ctx, err := Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build returned error when rules dir absent: %v", err)
	}
	if ctx == nil {
		t.Fatal("Build returned nil context when rules dir absent")
	}

	got := ctx.Rules().For("executor")
	if len(got) != 0 {
		t.Fatalf("expected empty slice from Rules().For when no rules dir, got %v", got)
	}
}

func TestContext_Rules_EmptyRulesDir_ReturnsEmptySetAndBuildSucceeds(t *testing.T) {
	root := t.TempDir()
	slug := "rules-empty-dir"
	seedSpecForRules(t, root, slug)

	rulesDir := paths.Rules(root)
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	defs := []PhaseDef{makeOneStepPhase(PhaseID(spec.PhaseInitial), "q")}
	ctx, err := Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build returned error for empty rules dir: %v", err)
	}
	if ctx == nil {
		t.Fatal("Build returned nil context for empty rules dir")
	}

	got := ctx.Rules().For("executor")
	if len(got) != 0 {
		t.Fatalf("expected empty slice from Rules().For for empty rules dir, got %v", got)
	}
}

func TestContext_Rules_UnknownAgentSubdir_ReturnsNilForThatAgent(t *testing.T) {
	root := t.TempDir()
	slug := "rules-unknown-subdir"
	seedSpecForRules(t, root, slug)

	rulesDir := paths.Rules(root)
	writeRuleFile(t, filepath.Join(rulesDir, "unknown-agent"), "rule.md")

	defs := []PhaseDef{makeOneStepPhase(PhaseID(spec.PhaseInitial), "q")}
	ctx, err := Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	got := ctx.Rules().For("unknown-agent")
	if len(got) != 0 {
		t.Fatalf("expected empty slice for unknown-agent subdir, got %v", got)
	}
}
