package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/phasecatalog"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func seedRollbackSpec(t *testing.T, root, slug, phase string, ruleLearningEnabled bool) {
	t.Helper()
	if err := os.MkdirAll(paths.Tddmaster(root), 0o755); err != nil {
		t.Fatalf("mkdir tddmaster dir: %v", err)
	}
	if err := os.WriteFile(paths.Manifest(root), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if _, err := spec.Start(root, slug, time.Now().UTC()); err != nil {
		t.Fatalf("spec.Start: %v", err)
	}

	state, err := spec.LoadState(root, slug)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	state.Phase = phase
	if err := spec.SaveState(root, slug, state); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	settings := spec.DefaultSettings()
	settings.RuleLearningEnabled = ruleLearningEnabled
	if err := spec.SaveSettings(root, slug, settings); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
}

func writeGlobalRuleFile(t *testing.T, root, name, content string) string {
	t.Helper()
	rulesDir := paths.Rules(root)
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("mkdir rules dir: %v", err)
	}
	p := filepath.Join(rulesDir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write global rule file: %v", err)
	}
	return p
}

func executeRollback(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := newRollbackCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func Test_RollbackCmd_ac1_ValidEarlierTarget_RollsBackAndPrintsSuccess(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	seedRollbackSpec(t, root, slug, string(phasecatalog.PhaseExecution), false)

	out, err := executeRollback(t, slug, string(phasecatalog.PhaseRefinement), "--root", root)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %q", err, out)
	}

	lower := strings.ToLower(out)
	if !strings.Contains(lower, slug) {
		t.Errorf("expected success output to mention slug %q, got: %q", slug, out)
	}
	if !(strings.Contains(out, "✓") || strings.Contains(lower, "success") || strings.Contains(lower, "rolled back")) {
		t.Errorf("expected success-style output (checkmark/success/rolled back), got: %q", out)
	}

	state, err := spec.LoadState(root, slug)
	if err != nil {
		t.Fatalf("LoadState after rollback: %v", err)
	}
	if state.Phase != string(phasecatalog.PhaseRefinement) {
		t.Errorf("expected phase %q after rollback, got %q", phasecatalog.PhaseRefinement, state.Phase)
	}
}

func Test_RollbackCmd_ac1_InvalidTargetPhase_ReturnsErrorListingValidPhases(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	seedRollbackSpec(t, root, slug, string(phasecatalog.PhaseExecution), false)

	_, err := executeRollback(t, slug, "not-a-real-phase", "--root", root)
	if err == nil {
		t.Fatal("expected error for invalid target phase, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, string(phasecatalog.PhaseSettings)) || !strings.Contains(msg, string(phasecatalog.PhaseDiscovery)) {
		t.Errorf("expected error to list valid phases (e.g. %q, %q), got: %q", phasecatalog.PhaseSettings, phasecatalog.PhaseDiscovery, msg)
	}
}

func Test_RollbackCmd_ec1_UnknownSlug_ReturnsError(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(paths.Tddmaster(root), 0o755); err != nil {
		t.Fatalf("mkdir tddmaster dir: %v", err)
	}
	if err := os.WriteFile(paths.Manifest(root), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, err := executeRollback(t, "nonexistent-spec", string(phasecatalog.PhaseDiscovery), "--root", root)
	if err == nil {
		t.Fatal("expected error for unknown slug, got nil")
	}
}

func Test_RollbackCmd_ec1_InvalidSlugFormat_ReturnsError(t *testing.T) {
	root := t.TempDir()

	_, err := executeRollback(t, "Not Valid Slug!!", string(phasecatalog.PhaseDiscovery), "--root", root)
	if err == nil {
		t.Fatal("expected error for invalid slug format, got nil")
	}
}

func Test_RollbackCmd_ec1_ForwardRollback_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	seedRollbackSpec(t, root, slug, string(phasecatalog.PhaseDiscovery), false)

	_, err := executeRollback(t, slug, string(phasecatalog.PhaseExecution), "--root", root)
	if err == nil {
		t.Fatal("expected error rolling forward to a later phase, got nil")
	}
}

func Test_RollbackCmd_ec1_EqualPhaseRollback_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	seedRollbackSpec(t, root, slug, string(phasecatalog.PhaseDiscovery), false)

	_, err := executeRollback(t, slug, string(phasecatalog.PhaseDiscovery), "--root", root)
	if err == nil {
		t.Fatal("expected error rolling back to the same (current) phase, got nil")
	}
}

func Test_RollbackCmd_ec1_WrongArgCount_ReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	seedRollbackSpec(t, root, slug, string(phasecatalog.PhaseExecution), false)

	if _, err := executeRollback(t, slug, "--root", root); err == nil {
		t.Fatal("expected error when target phase argument is missing, got nil")
	}
	if _, err := executeRollback(t, slug, string(phasecatalog.PhaseRefinement), "extra-arg", "--root", root); err == nil {
		t.Fatal("expected error when too many positional arguments are given, got nil")
	}
}

func Test_RollbackCmd_ac2_RuleLearningRange_PreservesGlobalRuleFilesAndWarns(t *testing.T) {
	root := t.TempDir()
	slug := "my-spec"
	seedRollbackSpec(t, root, slug, string(phasecatalog.PhaseRuleLearning), true)

	ruleFile := writeGlobalRuleFile(t, root, "shared-rule.md", "always write tests first")

	out, err := executeRollback(t, slug, string(phasecatalog.PhaseExecution), "--root", root)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %q", err, out)
	}

	lower := strings.ToLower(out)
	if !strings.Contains(lower, "rule") {
		t.Errorf("expected output to warn about global rule files, got: %q", out)
	}
	if !(strings.Contains(lower, "preserved") || strings.Contains(lower, "untouched") || strings.Contains(lower, "intact") || strings.Contains(lower, "unchanged")) {
		t.Errorf("expected output to state global rule files were preserved/untouched, got: %q", out)
	}

	if _, statErr := os.Stat(ruleFile); statErr != nil {
		t.Errorf("expected global rule file %q to still exist after rollback, got stat error: %v", ruleFile, statErr)
	}
	data, readErr := os.ReadFile(ruleFile)
	if readErr != nil {
		t.Fatalf("failed to read preserved global rule file: %v", readErr)
	}
	if string(data) != "always write tests first" {
		t.Errorf("global rule file content changed, got: %q", string(data))
	}
}
