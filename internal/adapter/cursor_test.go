package adapter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/manifest"
	"github.com/pragmataW/tddmaster/internal/paths"
)

func newCursorCtx(root string) SyncContext {
	m := manifest.Defaults()
	m.SelectedTools = []manifest.ToolID{manifest.ToolCursor}
	return SyncContext{
		Root:          root,
		Manifest:      &m,
		CommandPrefix: "tddmaster",
	}
}

func TestCursorAdapter_ID(t *testing.T) {
	a := CursorAdapter{}
	if a.ID() != manifest.ToolCursor {
		t.Fatalf("expected %q, got %q", manifest.ToolCursor, a.ID())
	}
}

func TestCursorAdapter_Sync_CreatesAgentsDir(t *testing.T) {
	tmp := t.TempDir()
	if err := (CursorAdapter{}).Sync(newCursorCtx(tmp)); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	info, err := os.Stat(paths.CursorAgents(tmp))
	if err != nil {
		t.Fatalf("cursor agents dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("cursor agents path is not a directory")
	}
}

func TestCursorAdapter_Sync_WritesAgentsMd_WithMarkers(t *testing.T) {
	tmp := t.TempDir()
	if err := (CursorAdapter{}).Sync(newCursorCtx(tmp)); err != nil {
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

func TestCursorAdapter_Sync_WritesAllAgentFiles_MinimalFrontmatter(t *testing.T) {
	tmp := t.TempDir()
	if err := (CursorAdapter{}).Sync(newCursorCtx(tmp)); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}
	for _, spec := range AgentSpecs {
		path := filepath.Join(paths.CursorAgents(tmp), spec.File+".md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("cursor agent %q not created: %v", spec.File, err)
			continue
		}
		content := string(data)
		if !strings.HasPrefix(content, "---") {
			t.Errorf("cursor agent %q does not start with --- frontmatter", spec.File)
		}
		if !strings.Contains(content, "name: "+spec.Name) {
			t.Errorf("cursor agent %q missing name: %s", spec.File, spec.Name)
		}
		if !strings.Contains(content, "description: ") {
			t.Errorf("cursor agent %q missing description", spec.File)
		}
		if strings.Contains(content, "model:") {
			t.Errorf("cursor agent %q must NOT contain a model field", spec.File)
		}
		if strings.Contains(content, "tools:") {
			t.Errorf("cursor agent %q must NOT contain a tools field", spec.File)
		}
	}
}

func TestCursorAdapter_Sync_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	ctx := newCursorCtx(tmp)
	if err := (CursorAdapter{}).Sync(ctx); err != nil {
		t.Fatalf("first Sync error: %v", err)
	}
	if err := (CursorAdapter{}).Sync(ctx); err != nil {
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
