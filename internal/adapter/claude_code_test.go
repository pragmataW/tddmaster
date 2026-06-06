package adapter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/manifest"
	"github.com/pragmataW/tddmaster/internal/paths"
)

func newSyncCtx(root string) SyncContext {
	m := manifest.Defaults()
	m.SelectedTools = []manifest.ToolID{manifest.ToolClaudeCode}
	return SyncContext{
		Root:          root,
		Manifest:      &m,
		CommandPrefix: "tddmaster",
	}
}

func TestClaudeCodeAdapter_ID(t *testing.T) {
	a := ClaudeCodeAdapter{}
	if a.ID() != manifest.ToolClaudeCode {
		t.Fatalf("expected %q, got %q", manifest.ToolClaudeCode, a.ID())
	}
}

func TestClaudeCodeAdapter_Sync_CreatesAgentsDir(t *testing.T) {
	tmp := t.TempDir()
	a := ClaudeCodeAdapter{}

	if err := a.Sync(newSyncCtx(tmp)); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	info, err := os.Stat(paths.ClaudeAgents(tmp))
	if err != nil {
		t.Fatalf("agents dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("agents path is not a directory")
	}
}

func TestClaudeCodeAdapter_Sync_CreatesCLAUDEMd_WithMarkers(t *testing.T) {
	tmp := t.TempDir()
	a := ClaudeCodeAdapter{}

	if err := a.Sync(newSyncCtx(tmp)); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	data, err := os.ReadFile(paths.ClaudeMd(tmp))
	if err != nil {
		t.Fatalf("CLAUDE.md not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "tddmasterStart") {
		t.Error("CLAUDE.md missing tddmasterStart marker")
	}
	if !strings.Contains(content, "tddmasterEnd") {
		t.Error("CLAUDE.md missing tddmasterEnd marker")
	}
	if !strings.Contains(content, "tddmaster") {
		t.Error("CLAUDE.md missing rendered command prefix")
	}
}

func TestClaudeCodeAdapter_Sync_WritesAllAgentFiles(t *testing.T) {
	tmp := t.TempDir()
	a := ClaudeCodeAdapter{}

	if err := a.Sync(newSyncCtx(tmp)); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	agentFiles := []struct {
		filename string
		nameTag  string
	}{
		{"tddmaster-executor.md", "tddmaster-executor"},
		{"tddmaster-verifier.md", "tddmaster-verifier"},
		{"tddmaster-planner.md", "tddmaster-planner"},
		{"tddmaster-test-writer.md", "test-writer"},
	}

	for _, af := range agentFiles {
		path := filepath.Join(paths.ClaudeAgents(tmp), af.filename)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("agent file %q not created: %v", af.filename, err)
			continue
		}
		if len(data) == 0 {
			t.Errorf("agent file %q is empty", af.filename)
			continue
		}
		content := string(data)
		if !strings.HasPrefix(content, "---") {
			t.Errorf("agent file %q does not start with --- frontmatter", af.filename)
		}
		if !strings.Contains(content, "name: "+af.nameTag) {
			t.Errorf("agent file %q missing name: %s", af.filename, af.nameTag)
		}
	}
}

func TestClaudeCodeAdapter_Sync_PreExistingCLAUDEMd_NoMarkers_AppendsBlock(t *testing.T) {
	tmp := t.TempDir()
	a := ClaudeCodeAdapter{}

	existing := "# My Project\nuser stuff\n"
	if err := os.WriteFile(paths.ClaudeMd(tmp), []byte(existing), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := a.Sync(newSyncCtx(tmp)); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	data, err := os.ReadFile(paths.ClaudeMd(tmp))
	if err != nil {
		t.Fatalf("CLAUDE.md unreadable: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "user stuff") {
		t.Error("pre-existing user content was removed")
	}
	if !strings.Contains(content, "tddmasterStart") {
		t.Error("marker block not appended: missing tddmasterStart")
	}
	if !strings.Contains(content, "tddmasterEnd") {
		t.Error("marker block not appended: missing tddmasterEnd")
	}
}

func TestClaudeCodeAdapter_Sync_PreExistingCLAUDEMd_WithMarkers_ReplacesOnlyBlock(t *testing.T) {
	tmp := t.TempDir()
	a := ClaudeCodeAdapter{}

	startMarker := "<!-- tddmasterStart -->"
	endMarker := "<!-- tddmasterEnd -->"

	existing := "# Header\nbefore text\n\n" +
		startMarker + "\nOLD CONTENT\n" + endMarker +
		"\n\nafter text\n"

	if err := os.WriteFile(paths.ClaudeMd(tmp), []byte(existing), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := a.Sync(newSyncCtx(tmp)); err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	data, err := os.ReadFile(paths.ClaudeMd(tmp))
	if err != nil {
		t.Fatalf("CLAUDE.md unreadable: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "before text") {
		t.Error("content before markers was removed")
	}
	if !strings.Contains(content, "after text") {
		t.Error("content after markers was removed")
	}
	if strings.Contains(content, "OLD CONTENT") {
		t.Error("old between-marker content was not replaced")
	}
	if !strings.Contains(content, "tddmasterStart") {
		t.Error("tddmasterStart marker missing after replace")
	}
	if !strings.Contains(content, "tddmasterEnd") {
		t.Error("tddmasterEnd marker missing after replace")
	}
}

func TestClaudeCodeAdapter_Sync_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	a := ClaudeCodeAdapter{}
	ctx := newSyncCtx(tmp)

	if err := a.Sync(ctx); err != nil {
		t.Fatalf("first Sync error: %v", err)
	}
	if err := a.Sync(ctx); err != nil {
		t.Fatalf("second Sync error: %v", err)
	}

	data, err := os.ReadFile(paths.ClaudeMd(tmp))
	if err != nil {
		t.Fatalf("CLAUDE.md unreadable: %v", err)
	}

	content := string(data)
	if strings.Count(content, "tddmasterStart") != 1 {
		t.Errorf("expected exactly 1 tddmasterStart, got %d", strings.Count(content, "tddmasterStart"))
	}
	if strings.Count(content, "tddmasterEnd") != 1 {
		t.Errorf("expected exactly 1 tddmasterEnd, got %d", strings.Count(content, "tddmasterEnd"))
	}

	agentFiles := []string{
		"tddmaster-executor.md",
		"tddmaster-verifier.md",
		"tddmaster-planner.md",
		"tddmaster-test-writer.md",
	}
	for _, f := range agentFiles {
		if _, err := os.Stat(filepath.Join(paths.ClaudeAgents(tmp), f)); err != nil {
			t.Errorf("agent file %q missing after second Sync: %v", f, err)
		}
	}
}
