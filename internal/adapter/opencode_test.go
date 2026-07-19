package adapter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/manifest"
	"github.com/pragmataW/tddmaster/internal/paths"
)

func newOpenCodeCtx(root string) SyncContext {
	m := manifest.Defaults()
	m.SelectedTools = []manifest.ToolID{manifest.ToolOpenCode}
	return SyncContext{
		Root:          root,
		Manifest:      &m,
		CommandPrefix: "tddmaster",
	}
}

func TestOpenCodeAdapter_ID(t *testing.T) {
	a := OpenCodeAdapter{}
	if a.ID() != manifest.ToolOpenCode {
		t.Fatalf("expected %q, got %q", manifest.ToolOpenCode, a.ID())
	}
}

func TestOpenCodeAdapter_Sync_CreatesAgentsDir(t *testing.T) {
	tmp := t.TempDir()
	if err := (OpenCodeAdapter{}).Sync(newOpenCodeCtx(tmp)); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	info, err := os.Stat(paths.OpenCodeAgents(tmp))
	if err != nil {
		t.Fatalf("opencode agents dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("opencode agents path is not a directory")
	}
}

func TestOpenCodeAdapter_Sync_WritesAgentsMd_WithMarkers(t *testing.T) {
	tmp := t.TempDir()
	if err := (OpenCodeAdapter{}).Sync(newOpenCodeCtx(tmp)); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	data, err := os.ReadFile(paths.AgentsMd(tmp))
	if err != nil {
		t.Fatalf("AGENTS.md not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "tddmasterStart") || !strings.Contains(content, "tddmasterEnd") {
		t.Error("AGENTS.md missing tddmaster markers")
	}
	if !strings.Contains(content, "tddmaster") {
		t.Error("AGENTS.md missing rendered command prefix")
	}
}

func TestOpenCodeAdapter_Sync_ParallelWorktreeProtocol(t *testing.T) {
	tmp := t.TempDir()
	if err := (OpenCodeAdapter{}).Sync(newOpenCodeCtx(tmp)); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	data, err := os.ReadFile(paths.AgentsMd(tmp))
	if err != nil {
		t.Fatalf("AGENTS.md unreadable: %v", err)
	}
	content := string(data)

	checks := []struct {
		name    string
		snippet string
	}{
		{"parallel spawn instruction", "in the SAME message so they run in parallel"},
		{"worktree protocol section", "### Parallel execution (worktree protocol)"},
		{"worktree add command", "git worktree add <worktree.path> -b <worktree.branch>"},
		{"merge-then-submit rule", "MERGE-THEN-SUBMIT (binding)"},
		{"no-git fallback", "git rev-parse --is-inside-work-tree"},
		{"git worktree exception", "NARROW EXCEPTION — worktree lifecycle only"},
		{"tasks array contract", "\"taskId\": \"task-1\""},
	}
	for _, c := range checks {
		if !strings.Contains(content, c.snippet) {
			t.Errorf("AGENTS.md missing %s: %q", c.name, c.snippet)
		}
	}

	if strings.Contains(content, "does not support parallel sub-agent spawning") {
		t.Error("AGENTS.md must not contain the sequential fallback sentence")
	}
}

func TestOpenCodeAdapter_Sync_WritesAllAgentFiles_Frontmatter(t *testing.T) {
	tmp := t.TempDir()
	if err := (OpenCodeAdapter{}).Sync(newOpenCodeCtx(tmp)); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	for _, spec := range AgentSpecs {
		path := filepath.Join(paths.OpenCodeAgents(tmp), spec.File+".md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("opencode agent %q not created: %v", spec.File, err)
			continue
		}
		content := string(data)
		if !strings.HasPrefix(content, "---") {
			t.Errorf("opencode agent %q does not start with --- frontmatter", spec.File)
		}
		if !strings.Contains(content, "description: ") {
			t.Errorf("opencode agent %q missing description", spec.File)
		}
		if !strings.Contains(content, "mode: subagent") {
			t.Errorf("opencode agent %q missing mode: subagent", spec.File)
		}
		if strings.Contains(content, "name:") {
			t.Errorf("opencode agent %q must NOT contain a name field", spec.File)
		}
		if strings.Contains(content, "model:") {
			t.Errorf("opencode agent %q must NOT contain a model field", spec.File)
		}
		if strings.Contains(content, "tools:") {
			t.Errorf("opencode agent %q must NOT contain a tools field", spec.File)
		}
	}
}

func TestOpenCodeAdapter_Sync_PermissionsDerivedFromTools(t *testing.T) {
	tmp := t.TempDir()
	if err := (OpenCodeAdapter{}).Sync(newOpenCodeCtx(tmp)); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	read := func(file string) string {
		data, err := os.ReadFile(filepath.Join(paths.OpenCodeAgents(tmp), file+".md"))
		if err != nil {
			t.Fatalf("opencode agent %q not created: %v", file, err)
		}
		return string(data)
	}

	executor := read("tddmaster-executor")
	if strings.Contains(executor, "permission:") {
		t.Error("executor must not have a permission block")
	}

	planner := read("tddmaster-planner")
	if !strings.Contains(planner, "edit: deny") || !strings.Contains(planner, "bash: deny") {
		t.Error("planner must deny edit and bash")
	}

	verifier := read("tddmaster-verifier")
	if !strings.Contains(verifier, "edit: deny") {
		t.Error("verifier must deny edit")
	}
	if strings.Contains(verifier, "bash: deny") {
		t.Error("verifier must NOT deny bash")
	}
}

func TestOpenCodeAdapter_Sync_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	ctx := newOpenCodeCtx(tmp)
	if err := (OpenCodeAdapter{}).Sync(ctx); err != nil {
		t.Fatalf("first Sync error: %v", err)
	}
	if err := (OpenCodeAdapter{}).Sync(ctx); err != nil {
		t.Fatalf("second Sync error: %v", err)
	}
	data, err := os.ReadFile(paths.AgentsMd(tmp))
	if err != nil {
		t.Fatalf("AGENTS.md unreadable: %v", err)
	}
	content := string(data)
	if strings.Count(content, "tddmasterStart") != 1 {
		t.Errorf("expected exactly 1 tddmasterStart, got %d", strings.Count(content, "tddmasterStart"))
	}
}
