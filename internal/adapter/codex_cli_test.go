package adapter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/manifest"
	"github.com/pragmataW/tddmaster/internal/paths"
)

func newCodexCtx(root string) SyncContext {
	m := manifest.Defaults()
	m.SelectedTools = []manifest.ToolID{manifest.ToolCodexCLI}
	return SyncContext{
		Root:          root,
		Manifest:      &m,
		CommandPrefix: "tddmaster",
	}
}

func TestCodexCLIAdapter_ID(t *testing.T) {
	a := CodexCLIAdapter{}
	if a.ID() != manifest.ToolCodexCLI {
		t.Fatalf("expected %q, got %q", manifest.ToolCodexCLI, a.ID())
	}
}

func TestCodexCLIAdapter_Sync_CreatesAgentsDir(t *testing.T) {
	tmp := t.TempDir()
	if err := (CodexCLIAdapter{}).Sync(newCodexCtx(tmp)); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	info, err := os.Stat(paths.CodexAgents(tmp))
	if err != nil {
		t.Fatalf("codex agents dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("codex agents path is not a directory")
	}
}

func TestCodexCLIAdapter_Sync_WritesAgentsMd_WithMarkers(t *testing.T) {
	tmp := t.TempDir()
	if err := (CodexCLIAdapter{}).Sync(newCodexCtx(tmp)); err != nil {
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
}

func TestCodexCLIAdapter_Sync_WritesAllAgentFiles_TomlMinimal(t *testing.T) {
	tmp := t.TempDir()
	if err := (CodexCLIAdapter{}).Sync(newCodexCtx(tmp)); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	for _, spec := range AgentSpecs {
		path := filepath.Join(paths.CodexAgents(tmp), spec.File+".toml")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("codex agent %q not created: %v", spec.File, err)
			continue
		}
		content := string(data)
		if !strings.Contains(content, "name = \""+spec.Name+"\"") {
			t.Errorf("codex agent %q missing name field", spec.File)
		}
		if !strings.Contains(content, "description = \"") {
			t.Errorf("codex agent %q missing description field", spec.File)
		}
		if !strings.Contains(content, "developer_instructions = \"\"\"") {
			t.Errorf("codex agent %q missing developer_instructions field", spec.File)
		}
		if !strings.HasSuffix(content, "\"\"\"\n") {
			t.Errorf("codex agent %q developer_instructions not properly terminated", spec.File)
		}
		if strings.Contains(content, "model = ") {
			t.Errorf("codex agent %q must NOT contain a model field", spec.File)
		}
		if strings.Contains(content, "model_reasoning_effort") {
			t.Errorf("codex agent %q must NOT contain an effort field", spec.File)
		}
	}
}

func TestCodexCLIAdapter_Sync_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	ctx := newCodexCtx(tmp)
	if err := (CodexCLIAdapter{}).Sync(ctx); err != nil {
		t.Fatalf("first Sync error: %v", err)
	}
	if err := (CodexCLIAdapter{}).Sync(ctx); err != nil {
		t.Fatalf("second Sync error: %v", err)
	}
	data, err := os.ReadFile(paths.AgentsMd(tmp))
	if err != nil {
		t.Fatalf("AGENTS.md unreadable: %v", err)
	}
	if strings.Count(string(data), "tddmasterStart") != 1 {
		t.Errorf("expected exactly 1 tddmasterStart, got %d", strings.Count(string(data), "tddmasterStart"))
	}
}
