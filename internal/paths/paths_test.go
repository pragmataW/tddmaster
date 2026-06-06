package paths

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestTddmaster(t *testing.T) {
	root := "/tmp/x"
	want := filepath.Join(root, ".tddmaster")
	got := Tddmaster(root)
	if got != want {
		t.Errorf("Tddmaster(%q) = %q; want %q", root, got, want)
	}
}

func TestManifest(t *testing.T) {
	root := "/tmp/x"
	want := filepath.Join(root, ".tddmaster", "manifest.json")
	got := Manifest(root)
	if got != want {
		t.Errorf("Manifest(%q) = %q; want %q", root, got, want)
	}
}

func TestClaudeAgents(t *testing.T) {
	root := "/tmp/x"
	want := filepath.Join(root, ".claude", "agents")
	got := ClaudeAgents(root)
	if got != want {
		t.Errorf("ClaudeAgents(%q) = %q; want %q", root, got, want)
	}
}

func TestClaudeMd(t *testing.T) {
	root := "/tmp/x"
	want := filepath.Join(root, "CLAUDE.md")
	got := ClaudeMd(root)
	if got != want {
		t.Errorf("ClaudeMd(%q) = %q; want %q", root, got, want)
	}
}

func TestManifest_IsUnderTddmaster(t *testing.T) {
	root := "/tmp/x"
	tddmasterDir := Tddmaster(root)
	manifestPath := Manifest(root)
	prefix := tddmasterDir + string(filepath.Separator)
	if !strings.HasPrefix(manifestPath, prefix) {
		t.Errorf("Manifest(%q) = %q is not under Tddmaster(%q) = %q", root, manifestPath, root, tddmasterDir)
	}
}
