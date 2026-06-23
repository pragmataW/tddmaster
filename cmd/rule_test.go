package cmd

import (
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
