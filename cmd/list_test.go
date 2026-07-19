package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pragmataW/tddmaster/internal/lifecycle"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/phasecatalog"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func writeManifestForList(t *testing.T, root string) {
	t.Helper()
	dir := paths.Tddmaster(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll manifest dir: %v", err)
	}
	if _, err := os.Stat(paths.Manifest(root)); err == nil {
		return
	}
	if err := os.WriteFile(paths.Manifest(root), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("WriteFile manifest: %v", err)
	}
}

func seedListSpec(t *testing.T, root, slug, phase, status string) {
	t.Helper()
	writeManifestForList(t, root)
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
	prog, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	prog.Status = status
	if err := spec.SaveProgress(root, slug, prog); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
}

func executeList(t *testing.T, root string, extraArgs ...string) (string, error) {
	t.Helper()
	rootCmd := newRootCmd()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	args := append([]string{"list"}, extraArgs...)
	if root != "" {
		args = append(args, "--root", root)
	}
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return buf.String(), err
}

func TestListCmd_ac1_RendersActiveSpecsWithHeaderAndAlignedColumns(t *testing.T) {
	root := t.TempDir()
	seedListSpec(t, root, "alpha-spec", string(phasecatalog.PhaseExecution), spec.StatusExecuting)
	seedListSpec(t, root, "beta-spec", string(phasecatalog.PhaseAnalysis), spec.StatusDraft)

	out, err := executeList(t, root)
	if err != nil {
		t.Fatalf("list command returned error: %v", err)
	}

	if !strings.Contains(out, "SLUG") {
		t.Errorf("expected output to contain header %q, got: %q", "SLUG", out)
	}
	if !strings.Contains(out, "PHASE") {
		t.Errorf("expected output to contain header %q, got: %q", "PHASE", out)
	}
	if !strings.Contains(out, "STATUS") {
		t.Errorf("expected output to contain header %q, got: %q", "STATUS", out)
	}

	if !strings.Contains(out, "alpha-spec") {
		t.Errorf("expected output to contain slug %q, got: %q", "alpha-spec", out)
	}
	if !strings.Contains(out, string(phasecatalog.PhaseExecution)) {
		t.Errorf("expected output to contain phase %q, got: %q", phasecatalog.PhaseExecution, out)
	}
	if !strings.Contains(out, spec.StatusExecuting) {
		t.Errorf("expected output to contain status %q, got: %q", spec.StatusExecuting, out)
	}

	if !strings.Contains(out, "beta-spec") {
		t.Errorf("expected output to contain slug %q, got: %q", "beta-spec", out)
	}
	if !strings.Contains(out, string(phasecatalog.PhaseAnalysis)) {
		t.Errorf("expected output to contain phase %q, got: %q", phasecatalog.PhaseAnalysis, out)
	}
	if !strings.Contains(out, spec.StatusDraft) {
		t.Errorf("expected output to contain status %q, got: %q", spec.StatusDraft, out)
	}
}

func TestListCmd_ac1_ArchivedFlagShowsOnlyArchivedSpecs(t *testing.T) {
	root := t.TempDir()
	seedListSpec(t, root, "active-only-spec", string(phasecatalog.PhaseExecution), spec.StatusExecuting)
	seedListSpec(t, root, "archived-spec", string(phasecatalog.PhaseAnalysis), spec.StatusDraft)
	if err := lifecycle.Archive(root, "archived-spec", time.Now().UTC()); err != nil {
		t.Fatalf("lifecycle.Archive: %v", err)
	}

	out, err := executeList(t, root, "--archived")
	if err != nil {
		t.Fatalf("list --archived returned error: %v", err)
	}

	if !strings.Contains(out, "archived-spec") {
		t.Errorf("expected --archived output to contain %q, got: %q", "archived-spec", out)
	}
	if strings.Contains(out, "active-only-spec") {
		t.Errorf("expected --archived output to NOT contain active spec %q, got: %q", "active-only-spec", out)
	}
}

func TestListCmd_ac1_PlainListShowsActiveSpecsNotArchived(t *testing.T) {
	root := t.TempDir()
	seedListSpec(t, root, "still-active-spec", string(phasecatalog.PhaseExecution), spec.StatusExecuting)
	seedListSpec(t, root, "now-archived-spec", string(phasecatalog.PhaseAnalysis), spec.StatusDraft)
	if err := lifecycle.Archive(root, "now-archived-spec", time.Now().UTC()); err != nil {
		t.Fatalf("lifecycle.Archive: %v", err)
	}

	out, err := executeList(t, root)
	if err != nil {
		t.Fatalf("list returned error: %v", err)
	}

	if !strings.Contains(out, "still-active-spec") {
		t.Errorf("expected plain list output to contain active spec %q, got: %q", "still-active-spec", out)
	}
	if strings.Contains(out, "now-archived-spec") {
		t.Errorf("expected plain list output to NOT contain archived spec %q, got: %q", "now-archived-spec", out)
	}
}

func TestListCmd_ac1_ec1_CorruptStateJSONDoesNotCrashAndOtherSpecsStillPrint(t *testing.T) {
	root := t.TempDir()
	seedListSpec(t, root, "healthy-spec", string(phasecatalog.PhaseExecution), spec.StatusExecuting)

	corruptSlug := "corrupt-spec"
	corruptDir := paths.SpecDir(root, corruptSlug)
	if err := os.MkdirAll(corruptDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(paths.SpecState(root, corruptSlug), []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("WriteFile corrupt state.json: %v", err)
	}

	out, err := executeList(t, root)
	if err != nil {
		t.Fatalf("expected list to not crash on corrupt state.json, got error: %v", err)
	}

	if !strings.Contains(out, "healthy-spec") {
		t.Errorf("expected output to still contain healthy spec %q, got: %q", "healthy-spec", out)
	}
	if !strings.Contains(out, corruptSlug) {
		t.Errorf("expected output to still list corrupt spec %q, got: %q", corruptSlug, out)
	}
}

func TestListCmd_ac1_ec1_EmptySpecsDirPrintsEmptyTableWithoutError(t *testing.T) {
	root := t.TempDir()

	out, err := executeList(t, root)
	if err != nil {
		t.Fatalf("expected list on empty specs dir to not error, got: %v", err)
	}

	if !strings.Contains(out, "SLUG") {
		t.Errorf("expected header %q even with no specs, got: %q", "SLUG", out)
	}

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) > 1 {
		t.Errorf("expected no data rows for empty specs dir, got extra lines: %q", out)
	}
}

func TestListCmd_ac1_RegisteredOnRoot(t *testing.T) {
	root := newRootCmd()
	var found bool
	for _, sub := range root.Commands() {
		if sub.Name() == "list" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'list' subcommand registered on root, but not found")
	}
}
