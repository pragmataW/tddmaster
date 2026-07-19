package lifecycle

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/phasecatalog"
	"github.com/pragmataW/tddmaster/internal/spec"
)

var rollbackFixedNow = time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

func writeManifestForLifecycle(t *testing.T, root string) {
	t.Helper()
	dir := paths.Tddmaster(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll manifest dir: %v", err)
	}
	if err := os.WriteFile(paths.Manifest(root), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("WriteFile manifest: %v", err)
	}
}

func setupRollbackFixture(t *testing.T, root, slug, phase string) {
	t.Helper()
	writeManifestForLifecycle(t, root)
	if _, err := spec.Start(root, slug, rollbackFixedNow); err != nil {
		t.Fatalf("spec.Start: %v", err)
	}
	state := buildFullFixtureState()
	state.Slug = slug
	state.Phase = phase
	if err := spec.SaveState(root, slug, state); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	prog := buildFullFixtureProgress()
	prog.Spec = slug
	if err := spec.SaveProgress(root, slug, prog); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
	writeFixtureFiles(t, root, slug)
}

func writeUserSourceFile(t *testing.T, root string) string {
	t.Helper()
	p := filepath.Join(root, "main.go")
	if err := os.WriteFile(p, []byte("package app\n"), 0o644); err != nil {
		t.Fatalf("WriteFile user source: %v", err)
	}
	return p
}

func TestRollback_ac1_ResetsDownstreamArtifactsAndPreservesEarlierFieldsAndUserFiles(t *testing.T) {
	root := t.TempDir()
	setupRollbackFixture(t, root, fixtureSlug, string(phasecatalog.PhaseExecution))
	userFile := writeUserSourceFile(t, root)

	if _, err := Rollback(root, fixtureSlug, string(phasecatalog.PhaseSpecProposal), rollbackFixedNow); err != nil {
		t.Fatalf("Rollback returned error: %v", err)
	}

	assertSpecMdDeleted(t, root, fixtureSlug)

	prog, err := spec.LoadProgress(root, fixtureSlug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if len(prog.Tasks) != 0 {
		t.Errorf("expected Tasks emptied, got %d", len(prog.Tasks))
	}
	if prog.Status != spec.StatusDraft {
		t.Errorf("expected Status draft, got %q", prog.Status)
	}

	state, err := spec.LoadState(root, fixtureSlug)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.Phase != string(phasecatalog.PhaseSpecProposal) {
		t.Errorf("expected Phase updated to target %q, got %q", phasecatalog.PhaseSpecProposal, state.Phase)
	}
	assertAnswerKeysPresent(t, state,
		"spec_settings",
		"listen_context", "mode", "premises", "status_quo", "ambition",
		"reversibility", "user_impact", "verification", "scope_boundary",
		"edge_cases", "synthesis",
	)
	assertAnswerKeysAbsent(t, state,
		"tasks_generated", "self_review", "refinement_approved",
		"analysis_complete", "analysis_audited", "analysis_findings", "analysis_attempts",
		"rule_proposal", "rule_approved", "rule_applied", "rule_feedback", "rule_attempt",
	)

	data, err := os.ReadFile(userFile)
	if err != nil {
		t.Fatalf("expected user source file to survive rollback: %v", err)
	}
	if string(data) != "package app\n" {
		t.Errorf("user source file content changed unexpectedly: %q", string(data))
	}
}

func TestRollback_ac1_ForwardRollbackRejected(t *testing.T) {
	root := t.TempDir()
	setupRollbackFixture(t, root, fixtureSlug, string(phasecatalog.PhaseSpecProposal))

	if _, err := Rollback(root, fixtureSlug, string(phasecatalog.PhaseExecution), rollbackFixedNow); err == nil {
		t.Fatalf("expected error for forward rollback, got nil")
	}

	state, err := spec.LoadState(root, fixtureSlug)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.Phase != string(phasecatalog.PhaseSpecProposal) {
		t.Errorf("expected Phase unchanged after rejected rollback, got %q", state.Phase)
	}
	assertSpecMdPreserved(t, root, fixtureSlug)
}

func TestRollback_ac1_EqualRollbackRejected(t *testing.T) {
	root := t.TempDir()
	setupRollbackFixture(t, root, fixtureSlug, string(phasecatalog.PhaseAnalysis))

	if _, err := Rollback(root, fixtureSlug, string(phasecatalog.PhaseAnalysis), rollbackFixedNow); err == nil {
		t.Fatalf("expected error for rollback to the same phase, got nil")
	}

	state, err := spec.LoadState(root, fixtureSlug)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.Phase != string(phasecatalog.PhaseAnalysis) {
		t.Errorf("expected Phase unchanged after rejected rollback, got %q", state.Phase)
	}
}

func TestRollback_ec1_UnknownTargetPhaseReturnsError(t *testing.T) {
	root := t.TempDir()
	setupRollbackFixture(t, root, fixtureSlug, string(phasecatalog.PhaseExecution))

	if _, err := Rollback(root, fixtureSlug, "not-a-real-phase", rollbackFixedNow); err == nil {
		t.Fatalf("expected error for unknown target phase, got nil")
	}
}

func TestRollback_ec1_NonexistentSlugReturnsError(t *testing.T) {
	root := t.TempDir()
	writeManifestForLifecycle(t, root)

	if _, err := Rollback(root, "does-not-exist", string(phasecatalog.PhaseSpecProposal), rollbackFixedNow); err == nil {
		t.Fatalf("expected error for nonexistent slug, got nil")
	}
}

func TestRollback_ec1_InvalidSlugFormatReturnsError(t *testing.T) {
	root := t.TempDir()
	writeManifestForLifecycle(t, root)

	if _, err := Rollback(root, "Invalid_Slug!", string(phasecatalog.PhaseSpecProposal), rollbackFixedNow); err == nil {
		t.Fatalf("expected error for invalid slug format, got nil")
	}
}

func TestRollback_ec2_CompletedSpecPhaseNotInOrder_RollbackAllowed(t *testing.T) {
	root := t.TempDir()
	setupRollbackFixture(t, root, fixtureSlug, string(engine.PhaseComplete))

	prog, err := spec.LoadProgress(root, fixtureSlug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	prog.Status = spec.StatusCompleted
	if err := spec.SaveProgress(root, fixtureSlug, prog); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}

	if _, err := Rollback(root, fixtureSlug, string(phasecatalog.PhaseAnalysis), rollbackFixedNow); err != nil {
		t.Fatalf("expected rollback allowed on a completed spec whose Phase is outside the enabled order, got error: %v", err)
	}

	if a := mustLoadAnalysis(t, root, fixtureSlug); a.Verdict != "" || len(a.Findings) != 0 {
		t.Errorf("expected analysis reset, got %+v", a)
	}

	loadedProg, err := spec.LoadProgress(root, fixtureSlug)
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if loadedProg.Status != spec.StatusDraft {
		t.Errorf("expected Status draft after rollback, got %q", loadedProg.Status)
	}

	state, err := spec.LoadState(root, fixtureSlug)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if state.Phase != string(phasecatalog.PhaseAnalysis) {
		t.Errorf("expected Phase updated to target %q, got %q", phasecatalog.PhaseAnalysis, state.Phase)
	}
}
