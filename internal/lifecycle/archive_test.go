package lifecycle

import (
	"os"
	"testing"

	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/phasecatalog"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func TestArchiveRestore_ac2_RoundTripToActive(t *testing.T) {
	root := t.TempDir()
	slug := "roundtrip-spec"
	setupRollbackFixture(t, root, slug, string(phasecatalog.PhaseExecution))

	if err := Archive(root, slug, rollbackFixedNow); err != nil {
		t.Fatalf("Archive returned error: %v", err)
	}
	if spec.Exists(root, slug) {
		t.Errorf("expected spec to no longer be active after Archive")
	}
	if _, err := os.Stat(paths.ArchiveSpecDir(root, slug)); err != nil {
		t.Errorf("expected archived spec dir to exist: %v", err)
	}

	if err := Restore(root, slug, rollbackFixedNow); err != nil {
		t.Fatalf("Restore returned error: %v", err)
	}
	if !spec.Exists(root, slug) {
		t.Errorf("expected spec to be active again after Restore")
	}
	if _, err := os.Stat(paths.ArchiveSpecDir(root, slug)); !os.IsNotExist(err) {
		t.Errorf("expected archived spec dir removed after Restore, stat err = %v", err)
	}
}

func TestRestore_ac2_ActiveSlugCollisionReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "collide-spec"
	setupRollbackFixture(t, root, slug, string(phasecatalog.PhaseExecution))

	if err := Archive(root, slug, rollbackFixedNow); err != nil {
		t.Fatalf("Archive returned error: %v", err)
	}

	if _, err := spec.Start(root, slug, rollbackFixedNow); err != nil {
		t.Fatalf("spec.Start (new active spec with colliding slug): %v", err)
	}

	if err := Restore(root, slug, rollbackFixedNow); err == nil {
		t.Fatalf("expected error restoring into an existing active slug, got nil")
	}
	if _, err := os.Stat(paths.ArchiveSpecDir(root, slug)); err != nil {
		t.Errorf("expected archived spec dir to remain after rejected restore: %v", err)
	}
}

func TestCancel_ac2_RemovesDirEntirely(t *testing.T) {
	root := t.TempDir()
	slug := "cancel-spec"
	setupRollbackFixture(t, root, slug, string(phasecatalog.PhaseExecution))

	if err := Cancel(root, slug); err != nil {
		t.Fatalf("Cancel returned error: %v", err)
	}

	if spec.Exists(root, slug) {
		t.Errorf("expected spec to no longer exist after Cancel")
	}
	if _, err := os.Stat(paths.SpecDir(root, slug)); !os.IsNotExist(err) {
		t.Errorf("expected spec directory removed entirely after Cancel, stat err = %v", err)
	}
}

func TestList_ac2_ReturnsActiveAndArchivedSpecsWithArchivedFlag(t *testing.T) {
	root := t.TempDir()
	activeSlug := "active-spec"
	archivedSlug := "archived-spec"
	setupRollbackFixture(t, root, activeSlug, string(phasecatalog.PhaseExecution))
	setupRollbackFixture(t, root, archivedSlug, string(phasecatalog.PhaseAnalysis))
	if err := Archive(root, archivedSlug, rollbackFixedNow); err != nil {
		t.Fatalf("Archive returned error: %v", err)
	}

	infos, err := List(root)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	bySlug := map[string]SpecInfo{}
	for _, info := range infos {
		bySlug[info.Slug] = info
	}

	active, ok := bySlug[activeSlug]
	if !ok {
		t.Fatalf("expected active spec %q in list", activeSlug)
	}
	if active.Archived {
		t.Errorf("expected active spec Archived=false, got true")
	}
	if active.Phase != string(phasecatalog.PhaseExecution) {
		t.Errorf("expected active spec Phase %q, got %q", phasecatalog.PhaseExecution, active.Phase)
	}
	if active.Status != spec.StatusExecuting {
		t.Errorf("expected active spec Status %q, got %q", spec.StatusExecuting, active.Status)
	}

	archived, ok := bySlug[archivedSlug]
	if !ok {
		t.Fatalf("expected archived spec %q in list", archivedSlug)
	}
	if !archived.Archived {
		t.Errorf("expected archived spec Archived=true, got false")
	}
	if archived.Phase != string(phasecatalog.PhaseAnalysis) {
		t.Errorf("expected archived spec Phase %q, got %q", phasecatalog.PhaseAnalysis, archived.Phase)
	}
}

func TestArchive_ec2_ArchivingAlreadyArchivedReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "double-archive-spec"
	setupRollbackFixture(t, root, slug, string(phasecatalog.PhaseExecution))

	if err := Archive(root, slug, rollbackFixedNow); err != nil {
		t.Fatalf("first Archive returned error: %v", err)
	}

	if err := Archive(root, slug, rollbackFixedNow); err == nil {
		t.Fatalf("expected error archiving an already-archived spec, got nil")
	}
	if _, err := os.Stat(paths.ArchiveSpecDir(root, slug)); err != nil {
		t.Errorf("expected archived spec dir to remain intact: %v", err)
	}
}

func TestList_ec3_TolerantOfCorruptStateJSON(t *testing.T) {
	root := t.TempDir()
	slug := "corrupt-spec"
	dir := paths.SpecDir(root, slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(paths.SpecState(root, slug), []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("WriteFile corrupt state.json: %v", err)
	}

	infos, err := List(root)
	if err != nil {
		t.Fatalf("expected List to tolerate corrupt state.json without erroring, got: %v", err)
	}

	var found *SpecInfo
	for i := range infos {
		if infos[i].Slug == slug {
			found = &infos[i]
		}
	}
	if found == nil {
		t.Fatalf("expected corrupt spec %q to still appear in list", slug)
	}
	if found.Status != "unknown" {
		t.Errorf("expected corrupt spec Status marked as sentinel %q, got %q", "unknown", found.Status)
	}
}

func TestList_ec3_TolerantOfMissingProgressJSON(t *testing.T) {
	root := t.TempDir()
	slug := "missing-progress-spec"
	dir := paths.SpecDir(root, slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	state := spec.State{
		Version:   1,
		Slug:      slug,
		Phase:     string(phasecatalog.PhaseDiscovery),
		Answers:   map[string][]spec.Answer{},
		CreatedAt: rollbackFixedNow,
		UpdatedAt: rollbackFixedNow,
	}
	if err := spec.SaveState(root, slug, state); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	infos, err := List(root)
	if err != nil {
		t.Fatalf("expected List to tolerate missing progress.json without erroring, got: %v", err)
	}

	var found *SpecInfo
	for i := range infos {
		if infos[i].Slug == slug {
			found = &infos[i]
		}
	}
	if found == nil {
		t.Fatalf("expected spec with missing progress.json to still appear in list")
	}
	if found.Status != "unknown" {
		t.Errorf("expected missing-progress spec Status marked as sentinel %q, got %q", "unknown", found.Status)
	}
	if found.Phase != string(phasecatalog.PhaseDiscovery) {
		t.Errorf("expected Phase read from valid state.json, got %q", found.Phase)
	}
}

func TestList_ec3_EmptySpecsDirReturnsEmptySlice(t *testing.T) {
	root := t.TempDir()

	infos, err := List(root)
	if err != nil {
		t.Fatalf("List returned error on empty root: %v", err)
	}
	if len(infos) != 0 {
		t.Errorf("expected empty slice for empty specs dir, got %d entries", len(infos))
	}
}
