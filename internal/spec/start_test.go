package spec

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pragmataW/tddmaster/internal/paths"
)

var fixedNow = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func writeManifest(t *testing.T, root string) {
	t.Helper()
	dir := paths.Tddmaster(root)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Manifest(root), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestStart_CreatesFiveFiles(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root)

	result, err := Start(root, "my-feature", fixedNow)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AlreadyExists {
		t.Fatal("expected AlreadyExists false")
	}
	if len(result.FilesWritten) != 5 {
		t.Fatalf("expected 5 files written, got %d", len(result.FilesWritten))
	}

	for _, p := range []string{
		paths.SpecState(root, "my-feature"),
		paths.SpecSettings(root, "my-feature"),
		paths.SpecProgress(root, "my-feature"),
		paths.SpecTraceability(root, "my-feature"),
		paths.SpecAnalysis(root, "my-feature"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected file to exist: %s, got err: %v", p, err)
		}
	}
}

func TestStart_CreatesAnalysisFileEmpty(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root)

	_, err := Start(root, "my-feature", fixedNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(paths.SpecAnalysis(root, "my-feature")); err != nil {
		t.Fatalf("expected analysis.json to exist at start: %v", err)
	}

	a, err := LoadAnalysis(root, "my-feature")
	if err != nil {
		t.Fatalf("LoadAnalysis error: %v", err)
	}
	if a.Verdict != "" {
		t.Errorf("expected empty verdict at start, got %q", a.Verdict)
	}
	if len(a.Findings) != 0 {
		t.Errorf("expected no findings at start, got %d", len(a.Findings))
	}
}

func TestStart_StateContent(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root)

	_, err := Start(root, "my-feature", fixedNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	state, err := LoadState(root, "my-feature")
	if err != nil {
		t.Fatalf("LoadState error: %v", err)
	}

	if state.Phase != PhaseInitial {
		t.Errorf("expected Phase %q, got %q", PhaseInitial, state.Phase)
	}
	if state.Slug != "my-feature" {
		t.Errorf("expected Slug %q, got %q", "my-feature", state.Slug)
	}
	if state.Version != 1 {
		t.Errorf("expected Version 1, got %d", state.Version)
	}
	if state.Answers == nil {
		t.Fatal("expected Answers non-nil")
	}
	if len(state.Answers) != 0 {
		t.Errorf("expected Answers empty, got len %d", len(state.Answers))
	}

	raw, err := os.ReadFile(paths.SpecState(root, "my-feature"))
	if err != nil {
		t.Fatalf("read state.json: %v", err)
	}
	if !strings.Contains(string(raw), `"answers": {}`) {
		t.Errorf("state.json must contain `\"answers\": {}`, got: %s", string(raw))
	}
}

func TestStart_SettingsContent(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root)

	_, err := Start(root, "my-feature", fixedNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	settings, err := LoadSettings(root, "my-feature")
	if err != nil {
		t.Fatalf("LoadSettings error: %v", err)
	}

	want := DefaultSettings()
	if settings != want {
		t.Errorf("expected settings %+v, got %+v", want, settings)
	}
}

func TestStart_ProgressContent(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root)

	_, err := Start(root, "my-feature", fixedNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	progress, err := LoadProgress(root, "my-feature")
	if err != nil {
		t.Fatalf("LoadProgress error: %v", err)
	}

	if progress.Status != "draft" {
		t.Errorf("expected Status %q, got %q", "draft", progress.Status)
	}
	if progress.Spec != "my-feature" {
		t.Errorf("expected Spec %q, got %q", "my-feature", progress.Spec)
	}
	if progress.Tasks == nil {
		t.Fatal("expected Tasks non-nil")
	}
	if len(progress.Tasks) != 0 {
		t.Errorf("expected Tasks empty, got len %d", len(progress.Tasks))
	}

	raw, err := os.ReadFile(paths.SpecProgress(root, "my-feature"))
	if err != nil {
		t.Fatalf("read progress.json: %v", err)
	}
	if !strings.Contains(string(raw), `"tasks": []`) {
		t.Errorf("progress.json must contain `\"tasks\": []`, got: %s", string(raw))
	}
}

func TestStart_NoSpecMd(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root)

	_, err := Start(root, "my-feature", fixedNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(paths.SpecMd(root, "my-feature")); err == nil {
		t.Error("spec.md must NOT be created by Start")
	}
}

func TestStart_MissingManifestErrors(t *testing.T) {
	root := t.TempDir()

	result, err := Start(root, "my-feature", fixedNow)

	if err == nil {
		t.Fatal("expected error when manifest missing, got nil")
	}
	if !strings.Contains(err.Error(), "tddmaster init") {
		t.Errorf("error must contain 'tddmaster init', got: %q", err.Error())
	}
	if result.AlreadyExists {
		t.Error("AlreadyExists must be false on error")
	}

	if _, err := os.Stat(paths.SpecDir(root, "my-feature")); err == nil {
		t.Error("SpecDir must NOT be created when manifest is missing")
	}
}

func TestStart_InvalidSlug(t *testing.T) {
	cases := []string{"Foo", "", "a/b", "-bad", "bad-"}

	for _, slug := range cases {
		t.Run(slug, func(t *testing.T) {
			root := t.TempDir()
			writeManifest(t, root)

			_, err := Start(root, slug, fixedNow)

			if err == nil {
				t.Errorf("expected error for invalid slug %q, got nil", slug)
			}
			if slug != "" {
				if _, statErr := os.Stat(paths.SpecDir(root, slug)); statErr == nil {
					t.Errorf("SpecDir must NOT be created for invalid slug %q", slug)
				}
			}
		})
	}
}

func TestStart_RerunAlreadyExists(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root)

	_, err := Start(root, "my-feature", fixedNow)
	if err != nil {
		t.Fatalf("first Start error: %v", err)
	}

	stateFile := paths.SpecState(root, "my-feature")
	contentBefore, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read state before second Start: %v", err)
	}
	infoBefore, err := os.Stat(stateFile)
	if err != nil {
		t.Fatalf("stat state before second Start: %v", err)
	}

	result, err := Start(root, "my-feature", fixedNow)

	if err != nil {
		t.Fatalf("second Start error: %v", err)
	}
	if !result.AlreadyExists {
		t.Error("expected AlreadyExists true on second Start")
	}

	contentAfter, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("read state after second Start: %v", err)
	}
	infoAfter, err := os.Stat(stateFile)
	if err != nil {
		t.Fatalf("stat state after second Start: %v", err)
	}

	if string(contentBefore) != string(contentAfter) {
		t.Error("state.json content must not change on second Start")
	}
	if infoBefore.ModTime() != infoAfter.ModTime() {
		t.Error("state.json mtime must not change on second Start")
	}
}

func TestStart_MultipleSlugsIsolated(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root)

	_, err := Start(root, "alpha", fixedNow)
	if err != nil {
		t.Fatalf("Start alpha error: %v", err)
	}
	_, err = Start(root, "beta", fixedNow)
	if err != nil {
		t.Fatalf("Start beta error: %v", err)
	}

	for _, slug := range []string{"alpha", "beta"} {
		if _, err := os.Stat(paths.SpecDir(root, slug)); err != nil {
			t.Errorf("SpecDir for %q must exist: %v", slug, err)
		}
		if _, err := os.Stat(paths.SpecState(root, slug)); err != nil {
			t.Errorf("state.json for %q must exist: %v", slug, err)
		}
	}

	alphaState, err := LoadState(root, "alpha")
	if err != nil {
		t.Fatalf("LoadState alpha: %v", err)
	}
	betaState, err := LoadState(root, "beta")
	if err != nil {
		t.Fatalf("LoadState beta: %v", err)
	}

	if alphaState.Slug != "alpha" {
		t.Errorf("alpha state has wrong slug: %q", alphaState.Slug)
	}
	if betaState.Slug != "beta" {
		t.Errorf("beta state has wrong slug: %q", betaState.Slug)
	}
}

func TestStart_CreatesTraceabilityFile(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root)

	_, err := Start(root, "my-feature", fixedNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(paths.SpecTraceability(root, "my-feature")); err != nil {
		t.Errorf("expected traceability.json to exist: %v", err)
	}
}

func TestStart_TraceabilityFileContentIsEmptyMap(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root)

	_, err := Start(root, "my-feature", fixedNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tr, err := LoadTraceability(root, "my-feature")
	if err != nil {
		t.Fatalf("LoadTraceability: %v", err)
	}
	if len(tr.Entries) != 0 {
		t.Errorf("traceability must start empty, got %d entries", len(tr.Entries))
	}
	if len(tr.Coverage) != 0 {
		t.Errorf("traceability must start with no coverage, got %d", len(tr.Coverage))
	}
}

func TestStart_TraceabilityLoadReturnsEmptyMap(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root)

	_, err := Start(root, "my-feature", fixedNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tr, err := LoadTraceability(root, "my-feature")
	if err != nil {
		t.Fatalf("LoadTraceability error: %v", err)
	}
	if len(tr.Entries) != 0 {
		t.Errorf("expected empty Traceability, got %d entries", len(tr.Entries))
	}
}

func TestStart_TraceabilityPathInFilesWritten(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root)

	result, err := Start(root, "my-feature", fixedNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := paths.SpecTraceability(root, "my-feature")
	found := false
	for _, p := range result.FilesWritten {
		if p == want {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("FilesWritten must contain %q, got: %v", want, result.FilesWritten)
	}
}

func TestStart_ArchivedSlugErrors(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root)
	if err := os.MkdirAll(paths.ArchiveSpecDir(root, "old-spec"), 0755); err != nil {
		t.Fatal(err)
	}

	_, err := Start(root, "old-spec", fixedNow)
	if err == nil {
		t.Fatal("expected error starting a spec whose slug exists in the archive, got nil")
	}
	if !strings.Contains(err.Error(), "restore") {
		t.Errorf("expected error to point at the restore command, got: %q", err.Error())
	}
	if Exists(root, "old-spec") {
		t.Errorf("expected no active spec to be created for an archived slug")
	}
}
