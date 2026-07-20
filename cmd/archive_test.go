package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/paths"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func executeArchive(t *testing.T, root string, extraArgs ...string) (string, error) {
	t.Helper()
	cmd := newArchiveCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	args := append([]string{}, extraArgs...)
	if root != "" {
		args = append(args, "--root", root)
	}
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func Test_ArchiveCmd_ac1_ArchivesActiveSpec_Success(t *testing.T) {
	root := t.TempDir()
	slug := "archive-me"
	scaffoldSpec(t, root, slug)

	out, err := executeArchive(t, root, slug)
	if err != nil {
		t.Fatalf("unexpected error archiving spec: %v", err)
	}

	if spec.Exists(root, slug) {
		t.Errorf("expected spec %q to no longer be active after archive", slug)
	}
	if _, statErr := os.Stat(paths.ArchiveSpecDir(root, slug)); statErr != nil {
		t.Errorf("expected archived spec dir to exist: %v", statErr)
	}
	if !strings.Contains(out, slug) {
		t.Errorf("expected archive success output to mention slug %q, got: %q", slug, out)
	}
}

func Test_ArchiveCmd_ac1_MissingSlugArg_ReturnsError(t *testing.T) {
	root := t.TempDir()

	_, err := executeArchive(t, root)
	if err == nil {
		t.Fatal("expected error when no slug arg provided, got nil")
	}
}

func Test_ArchiveCmd_ac1_InvalidSlug_ReturnsError(t *testing.T) {
	root := t.TempDir()

	_, err := executeArchive(t, root, "Not_A_Valid_Slug!")
	if err == nil {
		t.Fatal("expected error when slug is invalid, got nil")
	}
}

func Test_ArchiveCmd_ec1_ArchivingAlreadyArchivedReturnsError(t *testing.T) {
	root := t.TempDir()
	slug := "double-archive"
	scaffoldSpec(t, root, slug)

	if _, err := executeArchive(t, root, slug); err != nil {
		t.Fatalf("unexpected error on first archive: %v", err)
	}

	_, err := executeArchive(t, root, slug)
	if err == nil {
		t.Fatal("expected error archiving an already-archived spec, got nil")
	}
	if _, statErr := os.Stat(paths.ArchiveSpecDir(root, slug)); statErr != nil {
		t.Errorf("expected archived spec dir to remain intact after rejected re-archive: %v", statErr)
	}
}
