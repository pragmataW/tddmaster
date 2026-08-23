package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func seedSpecFiles(t *testing.T, root, slug, phase, status string) {
	t.Helper()
	dir := filepath.Join(paths.Specs(root), slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := spec.SaveState(root, slug, spec.State{Version: 1, Slug: slug, Phase: phase}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	if err := spec.SaveProgress(root, slug, spec.Progress{Spec: slug, Status: status}); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
}

func statusFor(t *testing.T, infos []SpecInfo, slug string) string {
	t.Helper()
	for _, info := range infos {
		if info.Slug == slug {
			return info.Status
		}
	}
	t.Fatalf("spec %q not listed", slug)
	return ""
}

func TestList_TasksDoneButPhasePending_IsNotReportedCompleted(t *testing.T) {
	root := t.TempDir()
	seedSpecFiles(t, root, "still-running", "rule-learning", spec.StatusCompleted)

	infos, err := List(root)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := statusFor(t, infos, "still-running"); got == spec.StatusCompleted {
		t.Fatal("a spec whose phase is still rule-learning must not read as completed")
	}
}

func TestList_TerminalPhase_IsReportedCompleted(t *testing.T) {
	root := t.TempDir()
	seedSpecFiles(t, root, "finished", "completed", spec.StatusCompleted)

	infos, err := List(root)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := statusFor(t, infos, "finished"); got != spec.StatusCompleted {
		t.Fatalf("terminal phase must report completed, got %q", got)
	}
}
