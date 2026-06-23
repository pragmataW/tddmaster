package paths

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDirRules_Constant(t *testing.T) {
	if DirRules != "rules" {
		t.Errorf("DirRules = %q; want %q", DirRules, "rules")
	}
}

func TestRules(t *testing.T) {
	root := "/tmp/proj"
	want := filepath.Join(root, ".tddmaster", "rules")
	got := Rules(root)
	if got != want {
		t.Errorf("Rules(%q) = %q; want %q", root, got, want)
	}
}

func TestRules_IsUnderTddmaster(t *testing.T) {
	root := "/tmp/proj"
	tddmasterDir := Tddmaster(root)
	rulesPath := Rules(root)
	prefix := tddmasterDir + string(filepath.Separator)
	if !strings.HasPrefix(rulesPath, prefix) {
		t.Errorf("Rules(%q) = %q is not under Tddmaster(%q) = %q", root, rulesPath, root, tddmasterDir)
	}
}

func TestRulesAgentDir(t *testing.T) {
	root := "/tmp/proj"
	cases := []struct {
		agent string
	}{
		{"test-writer"},
		{"executor"},
		{"verifier"},
		{"planner"},
	}
	for _, tc := range cases {
		want := filepath.Join(Rules(root), tc.agent)
		got := RulesAgentDir(root, tc.agent)
		if got != want {
			t.Errorf("RulesAgentDir(%q, %q) = %q; want %q", root, tc.agent, got, want)
		}
	}
}

func TestRulesAgentDir_IsUnderRules(t *testing.T) {
	root := "/tmp/proj"
	rulesDir := Rules(root)
	agentDir := RulesAgentDir(root, "executor")
	prefix := rulesDir + string(filepath.Separator)
	if !strings.HasPrefix(agentDir, prefix) {
		t.Errorf("RulesAgentDir(%q, executor) = %q is not under Rules(%q) = %q", root, agentDir, root, rulesDir)
	}
}

func TestRulesAgentDir_SlugIsolation(t *testing.T) {
	root := "/tmp/proj"
	agents := []string{"test-writer", "executor", "verifier", "planner"}
	dirs := make(map[string]string, len(agents))
	for _, a := range agents {
		dirs[a] = RulesAgentDir(root, a)
	}
	for i, a := range agents {
		for j, b := range agents {
			if i == j {
				continue
			}
			if dirs[a] == dirs[b] {
				t.Errorf("RulesAgentDir(root, %q) == RulesAgentDir(root, %q); agent dirs must be distinct", a, b)
			}
		}
	}
}
