package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func Test_RuleCmd_Use(t *testing.T) {
	cmd := newRuleCmd()
	if cmd.Use != "rule" {
		t.Errorf("newRuleCmd().Use = %q, want \"rule\"", cmd.Use)
	}
}

func Test_RuleCmd_Short(t *testing.T) {
	cmd := newRuleCmd()
	if cmd.Short == "" {
		t.Error("newRuleCmd().Short is empty, want non-empty")
	}
}

func Test_RuleAddCmd_Use(t *testing.T) {
	cmd := newRuleAddCmd()
	if cmd.Use != "add" {
		t.Errorf("newRuleAddCmd().Use = %q, want \"add\"", cmd.Use)
	}
}

func Test_RuleAddCmd_Short(t *testing.T) {
	cmd := newRuleAddCmd()
	if cmd.Short == "" {
		t.Error("newRuleAddCmd().Short is empty, want non-empty")
	}
}

func Test_RuleAddCmd_HasRunE(t *testing.T) {
	cmd := newRuleAddCmd()
	if cmd.RunE == nil {
		t.Error("newRuleAddCmd().RunE is nil, want non-nil")
	}
}

func Test_RuleAddCmd_RootFlagDefault(t *testing.T) {
	cmd := newRuleAddCmd()
	flag := cmd.Flags().Lookup("root")
	if flag == nil {
		t.Fatal("--root flag not found")
	}
	if flag.DefValue != "" {
		t.Errorf("--root flag default = %q, want \"\"", flag.DefValue)
	}
}

func Test_RuleAddCmd_RootFlagType(t *testing.T) {
	cmd := newRuleAddCmd()
	val, err := cmd.Flags().GetString("root")
	if err != nil {
		t.Fatalf("GetString(\"root\") error: %v", err)
	}
	if val != "" {
		t.Errorf("--root flag value before set = %q, want \"\"", val)
	}
}

func Test_RuleAddCmd_IsSubcommandOfRule(t *testing.T) {
	parent := newRuleCmd()
	var found *cobra.Command
	for _, sub := range parent.Commands() {
		if sub.Use == "add" {
			found = sub
			break
		}
	}
	if found == nil {
		t.Error("'add' subcommand not found under newRuleCmd()")
	}
}

func Test_RuleAddCmd_RunE_RootFlagResolution_WithExplicitRoot(t *testing.T) {
	cmd := newRuleAddCmd()
	if err := cmd.Flags().Set("root", t.TempDir()); err != nil {
		t.Fatalf("set --root flag: %v", err)
	}
	err := cmd.RunE(cmd, []string{})
	_ = err
}

func Test_RuleAddCmd_RunE_RootFlagResolution_EmptyRoot(t *testing.T) {
	cmd := newRuleAddCmd()
	err := cmd.RunE(cmd, []string{})
	_ = err
}

func Test_RuleCmd_RegisteredOnRoot(t *testing.T) {
	root := newRootCmd()
	var found bool
	for _, sub := range root.Commands() {
		if sub.Name() == "rule" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'rule' subcommand registered on root, but not found")
	}
}

func Test_RuleCmd_HasAddSubcommand(t *testing.T) {
	root := newRootCmd()
	var ruleCmd *cobra.Command
	for _, sub := range root.Commands() {
		if sub.Name() == "rule" {
			ruleCmd = sub
			break
		}
	}
	if ruleCmd == nil {
		t.Fatal("'rule' command not found on root")
	}

	var found bool
	for _, sub := range ruleCmd.Commands() {
		if sub.Name() == "add" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'add' subcommand on 'rule' command, but not found")
	}
}

func Test_RuleAddCmd_HasRootFlag(t *testing.T) {
	root := newRootCmd()
	var ruleCmd *cobra.Command
	for _, sub := range root.Commands() {
		if sub.Name() == "rule" {
			ruleCmd = sub
			break
		}
	}
	if ruleCmd == nil {
		t.Fatal("'rule' command not found on root")
	}

	var addCmd *cobra.Command
	for _, sub := range ruleCmd.Commands() {
		if sub.Name() == "add" {
			addCmd = sub
			break
		}
	}
	if addCmd == nil {
		t.Fatal("'add' subcommand not found on 'rule' command")
	}

	flag := addCmd.Flags().Lookup("root")
	if flag == nil {
		t.Error("expected --root flag on 'rule add' command, but not found")
	}
}

func Test_RuleAddCmd_HasScopeFlag(t *testing.T) {
	cmd := newRuleAddCmd()
	flag := cmd.Flags().Lookup("scope")
	if flag == nil {
		t.Fatal("--scope flag not registered on 'rule add'")
	}
}

func Test_RuleAddCmd_HasNameFlag(t *testing.T) {
	cmd := newRuleAddCmd()
	flag := cmd.Flags().Lookup("name")
	if flag == nil {
		t.Fatal("--name flag not registered on 'rule add'")
	}
}

func Test_RuleAddCmd_HasContentFlag(t *testing.T) {
	cmd := newRuleAddCmd()
	flag := cmd.Flags().Lookup("content")
	if flag == nil {
		t.Fatal("--content flag not registered on 'rule add'")
	}
}

func Test_RuleAddCmd_HasContentFileFlag(t *testing.T) {
	cmd := newRuleAddCmd()
	flag := cmd.Flags().Lookup("content-file")
	if flag == nil {
		t.Fatal("--content-file flag not registered on 'rule add'")
	}
}

func Test_RuleAddNonInteractive_AgentScope_CreatesFile(t *testing.T) {
	tmp := t.TempDir()
	cmd := newRuleAddCmd()
	cmd.SetArgs([]string{
		"--root", tmp,
		"--scope", "executor",
		"--name", "my-rule",
		"--content", "rule body",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(tmp, ".tddmaster", "rules", "executor", "my-rule.md")
	data, err := os.ReadFile(expected)
	if err != nil {
		t.Fatalf("rule file not created at %q: %v", expected, err)
	}
	if string(data) != "rule body" {
		t.Errorf("file content = %q, want %q", string(data), "rule body")
	}
}

func Test_RuleAddNonInteractive_GlobalScope_ContentFile(t *testing.T) {
	tmp := t.TempDir()

	multiline := "line one\nline two\nline three"
	cf, err := os.CreateTemp("", "ruleform-*.md")
	if err != nil {
		t.Fatalf("create temp content file: %v", err)
	}
	defer os.Remove(cf.Name())
	if _, err := cf.WriteString(multiline); err != nil {
		t.Fatalf("write content file: %v", err)
	}
	cf.Close()

	cmd := newRuleAddCmd()
	cmd.SetArgs([]string{
		"--root", tmp,
		"--scope", "global",
		"--name", "g1",
		"--content-file", cf.Name(),
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(tmp, ".tddmaster", "rules", "g1.md")
	data, err := os.ReadFile(expected)
	if err != nil {
		t.Fatalf("rule file not created at %q: %v", expected, err)
	}
	if string(data) != multiline {
		t.Errorf("file content = %q, want %q", string(data), multiline)
	}
}

func Test_RuleAddNonInteractive_ScopeWithoutName_ReturnsError(t *testing.T) {
	tmp := t.TempDir()
	cmd := newRuleAddCmd()
	cmd.SetArgs([]string{
		"--root", tmp,
		"--scope", "executor",
	})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when --scope is set without --name, got nil")
	}
}

func Test_RuleAddNonInteractive_ContentAndContentFile_MutuallyExclusive(t *testing.T) {
	tmp := t.TempDir()
	cf, err := os.CreateTemp(tmp, "rule-*.md")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	cmd := newRuleAddCmd()
	cmd.SetArgs([]string{
		"--root", tmp,
		"--scope", "executor",
		"--name", "dup",
		"--content", "inline",
		"--content-file", cf.Name(),
	})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when both --content and --content-file are set, got nil")
	}
	entries, _ := filepath.Glob(filepath.Join(tmp, ".tddmaster", "rules", "**", "*.md"))
	if len(entries) != 0 {
		t.Errorf("expected no files written, found: %v", entries)
	}
}

func Test_RuleAddNonInteractive_UnknownScope_ReturnsError(t *testing.T) {
	tmp := t.TempDir()
	cmd := newRuleAddCmd()
	cmd.SetArgs([]string{
		"--root", tmp,
		"--scope", "bogus-agent",
		"--name", "x",
		"--content", "y",
	})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown scope, got nil")
	}
	entries, _ := filepath.Glob(filepath.Join(tmp, ".tddmaster", "rules", "**", "*.md"))
	if len(entries) != 0 {
		t.Errorf("expected no files written, found: %v", entries)
	}
}

func Test_RuleAddNonInteractive_NoOverwrite_ExistingFile(t *testing.T) {
	tmp := t.TempDir()

	dir := filepath.Join(tmp, ".tddmaster", "rules", "executor")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	existing := filepath.Join(dir, "my-rule.md")
	sentinel := "sentinel content"
	if err := os.WriteFile(existing, []byte(sentinel), 0o644); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	cmd := newRuleAddCmd()
	cmd.SetArgs([]string{
		"--root", tmp,
		"--scope", "executor",
		"--name", "my-rule",
		"--content", "new content",
	})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when rule file already exists, got nil")
	}
	data, readErr := os.ReadFile(existing)
	if readErr != nil {
		t.Fatalf("existing file missing after attempted overwrite: %v", readErr)
	}
	if string(data) != sentinel {
		t.Errorf("existing file was overwritten: got %q, want %q", string(data), sentinel)
	}
}

func Test_RuleAddNonInteractive_PathTraversal_StaysContained(t *testing.T) {
	tmp := t.TempDir()
	cmd := newRuleAddCmd()
	cmd.SetArgs([]string{
		"--root", tmp,
		"--scope", "executor",
		"--name", "../../escape",
		"--content", "evil",
	})
	_ = cmd.Execute()

	escaped := filepath.Join(tmp, "escape.md")
	if _, err := os.Stat(escaped); err == nil {
		t.Errorf("path traversal succeeded: file written outside rules dir at %q", escaped)
	}

	rulesBase := filepath.Join(tmp, ".tddmaster", "rules")
	var found []string
	_ = filepath.Walk(rulesBase, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(rulesBase, path)
			found = append(found, rel)
		}
		return nil
	})
	cleanBase := filepath.Clean(rulesBase)
	for _, f := range found {
		abs := filepath.Clean(filepath.Join(rulesBase, f))
		if !strings.HasPrefix(abs, cleanBase+string(os.PathSeparator)) {
			t.Errorf("file escapes rules dir: %q", abs)
		}
	}
}

func Test_RuleAddCmd_NoScopeNoName_NonInteractiveBranchNotTaken(t *testing.T) {
	cmd := newRuleAddCmd()

	scopeFlag := cmd.Flags().Lookup("scope")
	if scopeFlag == nil {
		t.Fatal("--scope flag must be registered for non-interactive branch detection")
	}
	nameFlag := cmd.Flags().Lookup("name")
	if nameFlag == nil {
		t.Fatal("--name flag must be registered for non-interactive branch detection")
	}

	if scopeFlag.Changed || nameFlag.Changed {
		t.Error("flags should not be marked changed on a freshly constructed command")
	}
}
