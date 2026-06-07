package engine

import (
	"os"
	"testing"

	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func seedSpecProposalCtxSpec(t *testing.T, root, slug string) {
	t.Helper()
	dir := paths.Tddmaster(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll tddmaster dir: %v", err)
	}
	payload := `{"selectedTools":["claude-code"],"maxIterationBeforeStart":15,"command":"tddmaster"}`
	if err := os.WriteFile(paths.Manifest(root), []byte(payload), 0o644); err != nil {
		t.Fatalf("WriteFile manifest: %v", err)
	}
	st := spec.State{
		Version: 1,
		Slug:    slug,
		Phase:   "spec-proposal",
		Answers: map[string][]spec.Answer{},
	}
	if err := spec.SaveState(root, slug, st); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	if err := spec.SaveSettings(root, slug, spec.DefaultSettings()); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	pr := spec.Progress{Spec: slug, Status: spec.StatusDraft, Tasks: []spec.Task{}}
	if err := spec.SaveProgress(root, slug, pr); err != nil {
		t.Fatalf("SaveProgress: %v", err)
	}
}

func TestContext_Slug_ReturnsSlug(t *testing.T) {
	root := t.TempDir()
	slug := "my-slug"
	seedSpecProposalCtxSpec(t, root, slug)

	defs := []PhaseDef{}
	ctx, err := Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if got := ctx.Slug(); got != slug {
		t.Errorf("Slug() = %q, want %q", got, slug)
	}
}

func TestContext_State_PhaseMatchesSaved(t *testing.T) {
	root := t.TempDir()
	slug := "phase-slug"
	seedSpecProposalCtxSpec(t, root, slug)

	defs := []PhaseDef{}
	ctx, err := Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if got := ctx.State().Phase; got != "spec-proposal" {
		t.Errorf("State().Phase = %q, want %q", got, "spec-proposal")
	}
}

func TestContext_State_SlugMatchesSaved(t *testing.T) {
	root := t.TempDir()
	slug := "state-slug"
	seedSpecProposalCtxSpec(t, root, slug)

	defs := []PhaseDef{}
	ctx, err := Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if got := ctx.State().Slug; got != slug {
		t.Errorf("State().Slug = %q, want %q", got, slug)
	}
}

func TestContext_WriteSpecMd_WritesContentToSpecMdPath(t *testing.T) {
	root := t.TempDir()
	slug := "write-slug"
	seedSpecProposalCtxSpec(t, root, slug)

	defs := []PhaseDef{}
	ctx, err := Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	content := "# Spec: write-slug\n\nsome content"
	if err := ctx.WriteSpecMd(content); err != nil {
		t.Fatalf("WriteSpecMd: %v", err)
	}

	data, err := os.ReadFile(paths.SpecMd(root, slug))
	if err != nil {
		t.Fatalf("ReadFile spec.md: %v", err)
	}
	if got := string(data); got != content {
		t.Errorf("spec.md content = %q, want %q", got, content)
	}
}

func TestContext_WriteSpecMd_ExactContentRoundtrip(t *testing.T) {
	root := t.TempDir()
	slug := "roundtrip-slug"
	seedSpecProposalCtxSpec(t, root, slug)

	defs := []PhaseDef{}
	ctx, err := Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	want := "# Spec: roundtrip-slug\n\n## Tasks\n- [ ] task-1: Do something\n"
	if err := ctx.WriteSpecMd(want); err != nil {
		t.Fatalf("WriteSpecMd: %v", err)
	}

	data, err := os.ReadFile(paths.SpecMd(root, slug))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != want {
		t.Errorf("roundtrip failed: got %q, want %q", string(data), want)
	}
}

func TestContext_WriteSpecMd_EmptyContent_WritesEmptyFile(t *testing.T) {
	root := t.TempDir()
	slug := "empty-slug"
	seedSpecProposalCtxSpec(t, root, slug)

	defs := []PhaseDef{}
	ctx, err := Build(root, slug, defs)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if err := ctx.WriteSpecMd(""); err != nil {
		t.Fatalf("WriteSpecMd empty: %v", err)
	}

	data, err := os.ReadFile(paths.SpecMd(root, slug))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty file, got %q", string(data))
	}
}
