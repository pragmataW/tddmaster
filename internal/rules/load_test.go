package rules_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pragmataW/tddmaster/internal/rules"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func rulesDir(root string) string {
	return filepath.Join(root, ".tddmaster", "rules")
}

func TestLoad_MissingRulesDir_ReturnsEmptySetNilError(t *testing.T) {
	root := t.TempDir()
	s, err := rules.Load(root)
	if err != nil {
		t.Fatalf("Load returned error for missing rules dir: %v", err)
	}
	if got := s.For("executor"); len(got) != 0 {
		t.Errorf("For(executor) = %v; want empty", got)
	}
}

func TestLoad_EmptyRulesDir_ReturnsEmptySetNilError(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(rulesDir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := rules.Load(root)
	if err != nil {
		t.Fatalf("Load returned error for empty rules dir: %v", err)
	}
	for _, agent := range []string{"test-writer", "executor", "verifier", "planner"} {
		if got := s.For(agent); len(got) != 0 {
			t.Errorf("For(%q) = %v; want empty", agent, got)
		}
	}
}

func TestLoad_RootMdFiles_CollectedAsGlobal(t *testing.T) {
	root := t.TempDir()
	rd := rulesDir(root)
	writeFile(t, filepath.Join(rd, "naming.md"), "")
	writeFile(t, filepath.Join(rd, "style.md"), "")

	s, err := rules.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := s.For("executor")
	want := []string{
		filepath.Join(".tddmaster", "rules", "naming.md"),
		filepath.Join(".tddmaster", "rules", "style.md"),
	}
	if len(got) != len(want) {
		t.Fatalf("For(executor) = %v; want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("For(executor)[%d] = %q; want %q", i, got[i], want[i])
		}
	}
}

func TestLoad_AgentMdFiles_CollectedPerAgent(t *testing.T) {
	root := t.TempDir()
	rd := rulesDir(root)
	writeFile(t, filepath.Join(rd, "executor", "db.md"), "")

	s, err := rules.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := s.For("executor")
	want := []string{filepath.Join(".tddmaster", "rules", "executor", "db.md")}
	if len(got) != len(want) {
		t.Fatalf("For(executor) = %v; want %v", got, want)
	}
	if got[0] != want[0] {
		t.Errorf("For(executor)[0] = %q; want %q", got[0], want[0])
	}
}

func TestLoad_RootFirstThenAgentLexicallySorted(t *testing.T) {
	root := t.TempDir()
	rd := rulesDir(root)
	writeFile(t, filepath.Join(rd, "z-global.md"), "")
	writeFile(t, filepath.Join(rd, "a-global.md"), "")
	writeFile(t, filepath.Join(rd, "executor", "z-agent.md"), "")
	writeFile(t, filepath.Join(rd, "executor", "a-agent.md"), "")

	s, err := rules.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := s.For("executor")
	want := []string{
		filepath.Join(".tddmaster", "rules", "a-global.md"),
		filepath.Join(".tddmaster", "rules", "z-global.md"),
		filepath.Join(".tddmaster", "rules", "executor", "a-agent.md"),
		filepath.Join(".tddmaster", "rules", "executor", "z-agent.md"),
	}
	if len(got) != len(want) {
		t.Fatalf("For(executor) = %v; want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("For(executor)[%d] = %q; want %q", i, got[i], want[i])
		}
	}
}

func TestLoad_NonMdFilesIgnored(t *testing.T) {
	root := t.TempDir()
	rd := rulesDir(root)
	writeFile(t, filepath.Join(rd, "notes.txt"), "")
	writeFile(t, filepath.Join(rd, "noext"), "")
	writeFile(t, filepath.Join(rd, ".hidden"), "")
	writeFile(t, filepath.Join(rd, "valid.md"), "")
	writeFile(t, filepath.Join(rd, "executor", "skip.json"), "")
	writeFile(t, filepath.Join(rd, "executor", "keep.md"), "")

	s, err := rules.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := s.For("executor")
	want := []string{
		filepath.Join(".tddmaster", "rules", "valid.md"),
		filepath.Join(".tddmaster", "rules", "executor", "keep.md"),
	}
	if len(got) != len(want) {
		t.Fatalf("For(executor) = %v; want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("For(executor)[%d] = %q; want %q", i, got[i], want[i])
		}
	}
}

func TestLoad_UnknownSubdirIgnored(t *testing.T) {
	root := t.TempDir()
	rd := rulesDir(root)
	writeFile(t, filepath.Join(rd, "random-dir", "rule.md"), "")
	writeFile(t, filepath.Join(rd, "root.md"), "")

	s, err := rules.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := s.For("executor")
	want := []string{filepath.Join(".tddmaster", "rules", "root.md")}
	if len(got) != len(want) {
		t.Fatalf("For(executor) = %v; want %v", got, want)
	}
	if got[0] != want[0] {
		t.Errorf("For(executor)[0] = %q; want %q", got[0], want[0])
	}
}

func TestLoad_AgentWithNoRules_ForReturnsEmpty(t *testing.T) {
	root := t.TempDir()
	rd := rulesDir(root)
	writeFile(t, filepath.Join(rd, "executor", "only.md"), "")

	s, err := rules.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := s.For("planner")
	if len(got) != 0 {
		t.Errorf("For(planner) = %v; want empty (no planner rules)", got)
	}
}

func TestLoad_AllFourAgentSubdirsCollected(t *testing.T) {
	root := t.TempDir()
	rd := rulesDir(root)
	agents := []string{"test-writer", "executor", "verifier", "planner"}
	for _, a := range agents {
		writeFile(t, filepath.Join(rd, a, "rule.md"), "")
	}

	s, err := rules.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, a := range agents {
		got := s.For(a)
		want := filepath.Join(".tddmaster", "rules", a, "rule.md")
		if len(got) != 1 || got[0] != want {
			t.Errorf("For(%q) = %v; want [%q]", a, got, want)
		}
	}
}

func TestLoad_PathsAreRelativeToRoot(t *testing.T) {
	root := t.TempDir()
	rd := rulesDir(root)
	writeFile(t, filepath.Join(rd, "naming.md"), "")
	writeFile(t, filepath.Join(rd, "executor", "db.md"), "")

	s, err := rules.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, p := range s.For("executor") {
		if filepath.IsAbs(p) {
			t.Errorf("path %q is absolute; want relative to root", p)
		}
	}
}

func TestLoad_RootRulesAppearsForEveryAgent(t *testing.T) {
	root := t.TempDir()
	rd := rulesDir(root)
	writeFile(t, filepath.Join(rd, "global.md"), "")

	s, err := rules.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	globalPath := filepath.Join(".tddmaster", "rules", "global.md")
	for _, agent := range []string{"test-writer", "executor", "verifier", "planner"} {
		got := s.For(agent)
		if len(got) == 0 || got[0] != globalPath {
			t.Errorf("For(%q)[0] = %v; want %q as first entry", agent, got, globalPath)
		}
	}
}

func TestLoad_NoError_WhenRulesDirMissing(t *testing.T) {
	root := t.TempDir()
	_, err := rules.Load(root)
	if err != nil {
		t.Errorf("Load returned non-nil error for missing rules dir: %v", err)
	}
}

func TestFor_UnknownAgent_ReturnsOnlyGlobalRules(t *testing.T) {
	root := t.TempDir()
	rd := rulesDir(root)
	writeFile(t, filepath.Join(rd, "global.md"), "")

	s, err := rules.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := s.For("unknown-agent")
	want := []string{filepath.Join(".tddmaster", "rules", "global.md")}
	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("For(unknown-agent) = %v; want %v", got, want)
	}
}
