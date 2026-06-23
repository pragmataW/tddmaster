package ruleform_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/ui/ruleform"
)

func TestTargets_ContainsFiveKnownTargets(t *testing.T) {
	got := ruleform.Targets()
	want := []string{"global", "test-writer", "executor", "verifier", "planner"}
	if len(got) != len(want) {
		t.Fatalf("Targets() len = %d, want %d; got %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("Targets()[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestTargets_GlobalIsFirst(t *testing.T) {
	got := ruleform.Targets()
	if len(got) == 0 || got[0] != "global" {
		t.Errorf("Targets()[0] = %q, want \"global\"", func() string {
			if len(got) == 0 {
				return "<empty>"
			}
			return got[0]
		}())
	}
}

func TestSlugify(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"My Rule", "my-rule"},
		{"  spaced  name ", "spaced-name"},
		{"../../etc/passwd", "etc-passwd"},
		{"/abs/name", "abs-name"},
		{"weird@@@chars!!", "weird-chars"},
		{"已", ""},
		{"!!!", ""},
		{"", ""},
		{"hello-world", "hello-world"},
		{"UPPER", "upper"},
		{"a--b", "a-b"},
		{"-leading", "leading"},
		{"trailing-", "trailing"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			got := ruleform.Slugify(tc.input)
			if got != tc.want {
				t.Errorf("Slugify(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestEnsureMd_AppendsExtension(t *testing.T) {
	got := ruleform.EnsureMd("my-rule")
	if got != "my-rule.md" {
		t.Errorf("EnsureMd(\"my-rule\") = %q, want \"my-rule.md\"", got)
	}
}

func TestEnsureMd_NoDoubleExtension(t *testing.T) {
	got := ruleform.EnsureMd("my-rule.md")
	if got != "my-rule.md" {
		t.Errorf("EnsureMd(\"my-rule.md\") = %q, want \"my-rule.md\" (no doubling)", got)
	}
}

func TestEnsureMd_EmptyReturnsEmpty(t *testing.T) {
	got := ruleform.EnsureMd("")
	if got != "" {
		t.Errorf("EnsureMd(\"\") = %q, want \"\"", got)
	}
}

func TestTargetDir_Global(t *testing.T) {
	root := t.TempDir()
	got, err := ruleform.TargetDir(root, "global")
	if err != nil {
		t.Fatalf("TargetDir(root, \"global\") error: %v", err)
	}
	want := paths.Rules(root)
	if got != want {
		t.Errorf("TargetDir(root, \"global\") = %q, want %q", got, want)
	}
}

func TestTargetDir_KnownAgents(t *testing.T) {
	root := t.TempDir()
	agents := []string{"test-writer", "executor", "verifier", "planner"}
	for _, agent := range agents {
		agent := agent
		t.Run(agent, func(t *testing.T) {
			got, err := ruleform.TargetDir(root, agent)
			if err != nil {
				t.Fatalf("TargetDir(root, %q) error: %v", agent, err)
			}
			want := paths.RulesAgentDir(root, agent)
			if got != want {
				t.Errorf("TargetDir(root, %q) = %q, want %q", agent, got, want)
			}
		})
	}
}

func TestTargetDir_UnknownTarget_ReturnsError(t *testing.T) {
	root := t.TempDir()
	_, err := ruleform.TargetDir(root, "unknown-agent")
	if err == nil {
		t.Error("TargetDir with unknown target should return error, got nil")
	}
}

func TestWriteRule_HappyPath_GlobalTarget(t *testing.T) {
	root := t.TempDir()
	content := "# My Rule\nsome content"
	written, err := ruleform.WriteRule(root, "global", "My Rule", content)
	if err != nil {
		t.Fatalf("WriteRule error: %v", err)
	}

	targetDir := paths.Rules(root)
	if !strings.HasPrefix(written, targetDir) {
		t.Errorf("written path %q is not under targetDir %q", written, targetDir)
	}

	if filepath.Base(written) != "my-rule.md" {
		t.Errorf("written file basename = %q, want \"my-rule.md\"", filepath.Base(written))
	}

	data, err := os.ReadFile(written)
	if err != nil {
		t.Fatalf("ReadFile(%q) error: %v", written, err)
	}
	if string(data) != content {
		t.Errorf("file content = %q, want %q", string(data), content)
	}
}

func TestWriteRule_HappyPath_AgentTarget(t *testing.T) {
	root := t.TempDir()
	content := "agent rule body"
	written, err := ruleform.WriteRule(root, "executor", "Agent Rule", content)
	if err != nil {
		t.Fatalf("WriteRule error: %v", err)
	}

	targetDir := paths.RulesAgentDir(root, "executor")
	if !strings.HasPrefix(written, targetDir) {
		t.Errorf("written path %q is not under targetDir %q", written, targetDir)
	}

	if filepath.Base(written) != "agent-rule.md" {
		t.Errorf("written file basename = %q, want \"agent-rule.md\"", filepath.Base(written))
	}

	data, err := os.ReadFile(written)
	if err != nil {
		t.Fatalf("ReadFile(%q) error: %v", written, err)
	}
	if string(data) != content {
		t.Errorf("file content = %q, want %q", string(data), content)
	}
}

func TestWriteRule_CreatesDirIfNotExist(t *testing.T) {
	root := t.TempDir()
	targetDir := paths.RulesAgentDir(root, "verifier")

	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		t.Skip("dir already exists before test")
	}

	_, err := ruleform.WriteRule(root, "verifier", "new-rule", "body")
	if err != nil {
		t.Fatalf("WriteRule error: %v", err)
	}

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		t.Errorf("expected dir %q to be created, but it does not exist", targetDir)
	}
}

func TestWriteRule_TraversalSafety(t *testing.T) {
	root := t.TempDir()
	written, err := ruleform.WriteRule(root, "global", "../../escape", "content")
	if err != nil {
		t.Fatalf("WriteRule traversal input should succeed after sanitization, got error: %v", err)
	}

	targetDir := paths.Rules(root)
	if !strings.HasPrefix(written, targetDir) {
		t.Errorf("traversal: written path %q escaped targetDir %q", written, targetDir)
	}

	base := filepath.Base(written)
	if strings.Contains(base, "..") || strings.Contains(base, "/") {
		t.Errorf("traversal: basename %q contains unsafe characters", base)
	}

	if base != "escape.md" {
		t.Errorf("traversal: basename = %q, want \"escape.md\"", base)
	}
}

func TestWriteRule_EmptyAfterSlug_ReturnsError(t *testing.T) {
	root := t.TempDir()
	_, err := ruleform.WriteRule(root, "global", "!!!", "content")
	if err == nil {
		t.Error("WriteRule with name that slugifies to empty should return error, got nil")
	}
}

func TestWriteRule_UnknownTarget_ReturnsError(t *testing.T) {
	root := t.TempDir()
	_, err := ruleform.WriteRule(root, "unknown", "my-rule", "content")
	if err == nil {
		t.Error("WriteRule with unknown target should return error, got nil")
	}
}

func TestWriteRule_OverwriteWins(t *testing.T) {
	root := t.TempDir()
	_, err := ruleform.WriteRule(root, "global", "overwrite-me", "first content")
	if err != nil {
		t.Fatalf("first WriteRule error: %v", err)
	}

	written, err := ruleform.WriteRule(root, "global", "overwrite-me", "second content")
	if err != nil {
		t.Fatalf("second WriteRule error: %v", err)
	}

	data, err := os.ReadFile(written)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(data) != "second content" {
		t.Errorf("overwrite-wins: content = %q, want \"second content\"", string(data))
	}
}
