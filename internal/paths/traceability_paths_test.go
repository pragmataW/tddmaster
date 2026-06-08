package paths

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestFileTraceability_Constant(t *testing.T) {
	if FileTraceability != "traceability.json" {
		t.Errorf("FileTraceability = %q; want %q", FileTraceability, "traceability.json")
	}
}

func TestSpecTraceability(t *testing.T) {
	root := "/tmp/proj"
	slug := "my-spec"
	want := filepath.Join(SpecDir(root, slug), FileTraceability)
	got := SpecTraceability(root, slug)
	if got != want {
		t.Errorf("SpecTraceability(%q, %q) = %q; want %q", root, slug, got, want)
	}
}

func TestSpecTraceability_MatchesProgressPattern(t *testing.T) {
	root := "/tmp/proj"
	slug := "my-spec"

	progressPath := SpecProgress(root, slug)
	traceabilityPath := SpecTraceability(root, slug)

	progressDir := filepath.Dir(progressPath)
	traceabilityDir := filepath.Dir(traceabilityPath)

	if progressDir != traceabilityDir {
		t.Errorf("SpecTraceability dir = %q; want same dir as SpecProgress = %q", traceabilityDir, progressDir)
	}
}

func TestSpecTraceability_IsUnderSpecDir(t *testing.T) {
	root := "/tmp/proj"
	slug := "my-spec"
	specDir := SpecDir(root, slug)
	got := SpecTraceability(root, slug)
	prefix := specDir + string(filepath.Separator)
	if !strings.HasPrefix(got, prefix) {
		t.Errorf("SpecTraceability(%q, %q) = %q is not under SpecDir = %q", root, slug, got, specDir)
	}
}

func TestSpecTraceability_SlugIsolation(t *testing.T) {
	root := "/tmp/proj"
	pathAlpha := SpecTraceability(root, "alpha")
	pathBeta := SpecTraceability(root, "beta")
	if pathAlpha == pathBeta {
		t.Errorf("SpecTraceability with different slugs returned the same path: %q", pathAlpha)
	}
}
