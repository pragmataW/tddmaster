package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/phasecatalog"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func runRootCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func seedE2ESpec(t *testing.T, root, slug string) {
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
	state.Phase = string(phasecatalog.PhaseExecution)
	if err := spec.SaveState(root, slug, state); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	if err := spec.SaveSpecMd(root, slug, "# "+slug+"\n\nSpec proposal content."); err != nil {
		t.Fatalf("SaveSpecMd: %v", err)
	}

	prog := spec.Progress{
		Spec:   slug,
		Status: spec.StatusExecuting,
		Tasks: []spec.Task{
			{ID: "task-1", Title: "First task", Done: true},
			{ID: "task-2", Title: "Second task", Done: false},
		},
		TaskSeq:    2,
		UpdatedAt:  time.Now().UTC(),
		Iterations: 3,
	}
	if err := spec.SaveProgress(root, slug, prog); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}

	analysis := spec.Analysis{
		Verdict:  "pass",
		Findings: []spec.Finding{{Severity: spec.Severity("low"), Category: "coverage", Detail: "note"}},
	}
	if err := spec.SaveAnalysis(root, slug, analysis); err != nil {
		t.Fatalf("SaveAnalysis: %v", err)
	}

	trace := spec.Traceability{
		Entries: map[string][]spec.TraceEntry{
			"task-1": {{FunctionName: "TestSomething", TaskID: "task-1", CriterionIDs: []string{"ac-1"}, EC: []string{}}},
		},
	}
	if err := spec.SaveTraceability(root, slug, trace); err != nil {
		t.Fatalf("SaveTraceability: %v", err)
	}
}

func Test_LifecycleE2E_ac1_RootRegistersAllFiveLifecycleCommands(t *testing.T) {
	root := newRootCmd()
	want := []string{"list", "rollback", "archive", "restore", "cancel"}
	got := map[string]bool{}
	for _, sub := range root.Commands() {
		got[sub.Name()] = true
	}
	for _, name := range want {
		if !got[name] {
			t.Errorf("expected root command to register subcommand %q, but it was not found among: %v", name, got)
		}
	}
}

func Test_LifecycleE2E_ac1_FullStagedFlow_RollbackArchiveRestoreCancel(t *testing.T) {
	root := t.TempDir()
	slug := "e2e-lifecycle-spec"
	seedE2ESpec(t, root, slug)

	rollbackOut, err := runRootCmd(t, "rollback", slug, string(phasecatalog.PhaseSpecProposal), "--root", root)
	if err != nil {
		t.Fatalf("root rollback command returned error: %v\noutput: %q", err, rollbackOut)
	}

	if _, statErr := os.Stat(paths.SpecMd(root, slug)); !os.IsNotExist(statErr) {
		t.Errorf("expected spec.md to be removed after rollback to spec-proposal, stat err: %v", statErr)
	}

	progAfterRollback, err := spec.LoadProgress(root, slug)
	if err != nil {
		t.Fatalf("LoadProgress after rollback: %v", err)
	}
	if len(progAfterRollback.Tasks) != 0 {
		t.Errorf("expected Progress.Tasks to be emptied after rollback to spec-proposal, got: %v", progAfterRollback.Tasks)
	}

	listOut, err := runRootCmd(t, "list", "--root", root)
	if err != nil {
		t.Fatalf("root list command returned error: %v\noutput: %q", err, listOut)
	}
	if !strings.Contains(listOut, slug) {
		t.Errorf("expected list output to contain slug %q, got: %q", slug, listOut)
	}
	if !strings.Contains(listOut, string(phasecatalog.PhaseSpecProposal)) {
		t.Errorf("expected list output to show phase %q after rollback, got: %q", phasecatalog.PhaseSpecProposal, listOut)
	}

	archiveOut, err := runRootCmd(t, "archive", slug, "--root", root)
	if err != nil {
		t.Fatalf("root archive command returned error: %v\noutput: %q", err, archiveOut)
	}
	if spec.Exists(root, slug) {
		t.Errorf("expected spec %q to no longer be active after archive", slug)
	}

	archivedListOut, err := runRootCmd(t, "list", "--archived", "--root", root)
	if err != nil {
		t.Fatalf("root list --archived command returned error: %v\noutput: %q", err, archivedListOut)
	}
	if !strings.Contains(archivedListOut, slug) {
		t.Errorf("expected list --archived output to contain slug %q, got: %q", slug, archivedListOut)
	}

	restoreOut, err := runRootCmd(t, "restore", slug, "--root", root)
	if err != nil {
		t.Fatalf("root restore command returned error: %v\noutput: %q", err, restoreOut)
	}
	if !spec.Exists(root, slug) {
		t.Errorf("expected spec %q to be active again after restore", slug)
	}

	cancelOut, err := runRootCmd(t, "cancel", slug, "--force", "--root", root)
	if err != nil {
		t.Fatalf("root cancel command returned error: %v\noutput: %q", err, cancelOut)
	}
	if spec.Exists(root, slug) {
		t.Errorf("expected spec %q to be removed after cancel --force", slug)
	}
}

func Test_LifecycleE2E_ac1_RollbackAdvancesForwardFromEachPhase(t *testing.T) {
	root := t.TempDir()
	slug := "e2e-forward-check"
	seedE2ESpec(t, root, slug)

	if _, err := runRootCmd(t, "rollback", slug, string(phasecatalog.PhaseRefinement), "--root", root); err != nil {
		t.Fatalf("root rollback to refinement returned error: %v", err)
	}

	nextOut, err := runRootCmd(t, "next", slug, "--root", root)
	if err != nil {
		t.Fatalf("root next command after rollback returned error: %v\noutput: %q", err, nextOut)
	}
	state, err := spec.LoadState(root, slug)
	if err != nil {
		t.Fatalf("LoadState after next: %v", err)
	}
	if state.Phase == string(phasecatalog.PhaseExecution) {
		t.Errorf("expected next to advance forward from refinement without jumping back to execution, phase: %q", state.Phase)
	}
}

func Test_LifecycleE2E_ec1_RollbackInvalidTargetPhase_ReturnsErrorListingValidPhases(t *testing.T) {
	root := t.TempDir()
	slug := "e2e-invalid-phase"
	seedE2ESpec(t, root, slug)

	_, err := runRootCmd(t, "rollback", slug, "not-a-real-phase", "--root", root)
	if err == nil {
		t.Fatal("expected error for invalid target phase via root command, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, string(phasecatalog.PhaseSettings)) || !strings.Contains(msg, string(phasecatalog.PhaseDiscovery)) {
		t.Errorf("expected error to list valid phases (e.g. %q, %q), got: %q", phasecatalog.PhaseSettings, phasecatalog.PhaseDiscovery, msg)
	}
}

func Test_LifecycleE2E_ec1_RollbackUnknownSlug_ReturnsError(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(paths.Tddmaster(root), 0o755); err != nil {
		t.Fatalf("mkdir tddmaster dir: %v", err)
	}
	if err := os.WriteFile(paths.Manifest(root), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, err := runRootCmd(t, "rollback", "no-such-slug", string(phasecatalog.PhaseDiscovery), "--root", root)
	if err == nil {
		t.Fatal("expected error rolling back an unknown slug via root command, got nil")
	}
}

func Test_LifecycleE2E_ec2_ListOnCorruptStateJSON_DoesNotCrash(t *testing.T) {
	root := t.TempDir()
	corruptSlug := "corrupt-e2e-spec"
	corruptDir := paths.SpecDir(root, corruptSlug)
	if err := os.MkdirAll(corruptDir, 0o755); err != nil {
		t.Fatalf("MkdirAll corrupt spec dir: %v", err)
	}
	if err := os.WriteFile(paths.SpecState(root, corruptSlug), []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("WriteFile corrupt state.json: %v", err)
	}
	if err := os.WriteFile(paths.SpecProgress(root, corruptSlug), []byte("also not json"), 0o644); err != nil {
		t.Fatalf("WriteFile corrupt progress.json: %v", err)
	}

	out, err := runRootCmd(t, "list", "--root", root)
	if err != nil {
		t.Fatalf("expected root list to not crash on corrupt state.json/progress.json, got error: %v", err)
	}
	if !strings.Contains(out, corruptSlug) {
		t.Errorf("expected list output to still mention corrupt spec %q, got: %q", corruptSlug, out)
	}
}

func Test_LifecycleE2E_ec2_ListOnEmptyRoot_DoesNotError(t *testing.T) {
	root := t.TempDir()

	out, err := runRootCmd(t, "list", "--root", root)
	if err != nil {
		t.Fatalf("expected root list on empty/uninitialized root to not error, got: %v\noutput: %q", err, out)
	}
}
