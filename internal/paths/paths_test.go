package paths

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestTddmaster(t *testing.T) {
	root := "/tmp/x"
	want := filepath.Join(root, ".tddmaster")
	got := Tddmaster(root)
	if got != want {
		t.Errorf("Tddmaster(%q) = %q; want %q", root, got, want)
	}
}

func TestManifest(t *testing.T) {
	root := "/tmp/x"
	want := filepath.Join(root, ".tddmaster", "manifest.json")
	got := Manifest(root)
	if got != want {
		t.Errorf("Manifest(%q) = %q; want %q", root, got, want)
	}
}

func TestClaudeAgents(t *testing.T) {
	root := "/tmp/x"
	want := filepath.Join(root, ".claude", "agents")
	got := ClaudeAgents(root)
	if got != want {
		t.Errorf("ClaudeAgents(%q) = %q; want %q", root, got, want)
	}
}

func TestClaudeMd(t *testing.T) {
	root := "/tmp/x"
	want := filepath.Join(root, "CLAUDE.md")
	got := ClaudeMd(root)
	if got != want {
		t.Errorf("ClaudeMd(%q) = %q; want %q", root, got, want)
	}
}

func TestManifest_IsUnderTddmaster(t *testing.T) {
	root := "/tmp/x"
	tddmasterDir := Tddmaster(root)
	manifestPath := Manifest(root)
	prefix := tddmasterDir + string(filepath.Separator)
	if !strings.HasPrefix(manifestPath, prefix) {
		t.Errorf("Manifest(%q) = %q is not under Tddmaster(%q) = %q", root, manifestPath, root, tddmasterDir)
	}
}

func TestSpecs(t *testing.T) {
	root := "/tmp/proj"
	want := filepath.Join(root, ".tddmaster", DirSpecs)
	got := Specs(root)
	if got != want {
		t.Errorf("Specs(%q) = %q; want %q", root, got, want)
	}
}

func TestSpecs_IsUnderTddmaster(t *testing.T) {
	root := "/tmp/proj"
	want := filepath.Join(Tddmaster(root), DirSpecs)
	got := Specs(root)
	if got != want {
		t.Errorf("Specs(%q) = %q; want %q (child of Tddmaster)", root, got, want)
	}
}

func TestSpecDir(t *testing.T) {
	root := "/tmp/proj"
	slug := "my-spec"
	want := filepath.Join(Specs(root), slug)
	got := SpecDir(root, slug)
	if got != want {
		t.Errorf("SpecDir(%q, %q) = %q; want %q", root, slug, got, want)
	}
}

func TestSpecState(t *testing.T) {
	root := "/tmp/proj"
	slug := "my-spec"
	want := filepath.Join(SpecDir(root, slug), FileState)
	got := SpecState(root, slug)
	if got != want {
		t.Errorf("SpecState(%q, %q) = %q; want %q", root, slug, got, want)
	}
}

func TestSpecSettings(t *testing.T) {
	root := "/tmp/proj"
	slug := "my-spec"
	want := filepath.Join(SpecDir(root, slug), FileSettings)
	got := SpecSettings(root, slug)
	if got != want {
		t.Errorf("SpecSettings(%q, %q) = %q; want %q", root, slug, got, want)
	}
}

func TestSpecProgress(t *testing.T) {
	root := "/tmp/proj"
	slug := "my-spec"
	want := filepath.Join(SpecDir(root, slug), FileProgress)
	got := SpecProgress(root, slug)
	if got != want {
		t.Errorf("SpecProgress(%q, %q) = %q; want %q", root, slug, got, want)
	}
}

func TestSpecMd(t *testing.T) {
	root := "/tmp/proj"
	slug := "my-spec"
	want := filepath.Join(SpecDir(root, slug), "spec.md")
	got := SpecMd(root, slug)
	if got != want {
		t.Errorf("SpecMd(%q, %q) = %q; want %q", root, slug, got, want)
	}
}

func TestSpecFileHelpers_TableDriven(t *testing.T) {
	root := "/tmp/proj"
	slug := "example"
	specDir := SpecDir(root, slug)

	cases := []struct {
		name string
		got  string
		want string
	}{
		{"SpecState", SpecState(root, slug), filepath.Join(specDir, FileState)},
		{"SpecSettings", SpecSettings(root, slug), filepath.Join(specDir, FileSettings)},
		{"SpecProgress", SpecProgress(root, slug), filepath.Join(specDir, FileProgress)},
		{"SpecMd", SpecMd(root, slug), filepath.Join(specDir, "spec.md")},
	}

	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s = %q; want %q", tc.name, tc.got, tc.want)
		}
	}
}

func TestSpecAnalysis_Path(t *testing.T) {
	root := "/tmp/proj"
	slug := "my-spec"
	want := filepath.Join(SpecDir(root, slug), "analysis.json")
	got := SpecAnalysis(root, slug)
	if got != want {
		t.Errorf("SpecAnalysis(%q, %q) = %q; want %q", root, slug, got, want)
	}
}

func TestSpecDir_SlugIsolation(t *testing.T) {
	root := "/tmp/proj"
	dirAlpha := SpecDir(root, "alpha")
	dirBeta := SpecDir(root, "beta")
	specsDir := Specs(root)

	if dirAlpha == dirBeta {
		t.Errorf("SpecDir(root, alpha) == SpecDir(root, beta); slugs must produce distinct directories")
	}

	prefixAlpha := dirAlpha + string(filepath.Separator)
	prefixBeta := dirBeta + string(filepath.Separator)

	if strings.HasPrefix(dirBeta, prefixAlpha) {
		t.Errorf("SpecDir(root, beta) is nested under SpecDir(root, alpha); slugs must be non-overlapping")
	}
	if strings.HasPrefix(dirAlpha, prefixBeta) {
		t.Errorf("SpecDir(root, alpha) is nested under SpecDir(root, beta); slugs must be non-overlapping")
	}

	specsPrefix := specsDir + string(filepath.Separator)
	if !strings.HasPrefix(dirAlpha, specsPrefix) {
		t.Errorf("SpecDir(root, alpha) = %q is not under Specs(root) = %q", dirAlpha, specsDir)
	}
	if !strings.HasPrefix(dirBeta, specsPrefix) {
		t.Errorf("SpecDir(root, beta) = %q is not under Specs(root) = %q", dirBeta, specsDir)
	}
}
