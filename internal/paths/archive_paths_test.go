package paths

import (
	"path/filepath"
	"testing"
)

func TestArchive_ac1(t *testing.T) {
	root := "/tmp/proj"
	want := filepath.Join(root, ".tddmaster", DirArchive)
	got := Archive(root)
	if got != want {
		t.Errorf("Archive(%q) = %q; want %q", root, got, want)
	}
}

func TestArchiveSpecDir_ac1(t *testing.T) {
	root := "/tmp/proj"
	slug := "my-spec"
	want := filepath.Join(Archive(root), slug)
	got := ArchiveSpecDir(root, slug)
	if got != want {
		t.Errorf("ArchiveSpecDir(%q, %q) = %q; want %q", root, slug, got, want)
	}
}
