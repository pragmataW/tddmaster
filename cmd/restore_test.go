package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pragmataW/tddmaster/internal/spec"
)

func executeRestore(t *testing.T, root string, extraArgs ...string) (string, error) {
	t.Helper()
	cmd := newRestoreCmd()
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

func Test_RestoreCmd_ac1_RoundTripThenCollisionConflictError(t *testing.T) {
	root := t.TempDir()
	slug := "roundtrip-cmd-spec"
	scaffoldSpec(t, root, slug)

	if _, err := executeArchive(t, root, slug); err != nil {
		t.Fatalf("unexpected error archiving spec: %v", err)
	}
	if spec.Exists(root, slug) {
		t.Fatalf("expected spec %q to be inactive after archive", slug)
	}

	out, err := executeRestore(t, root, slug)
	if err != nil {
		t.Fatalf("unexpected error restoring spec: %v", err)
	}
	if !spec.Exists(root, slug) {
		t.Errorf("expected spec %q to be active again after restore", slug)
	}
	if !strings.Contains(out, slug) {
		t.Errorf("expected restore success output to mention slug %q, got: %q", slug, out)
	}

	if _, err := executeArchive(t, root, slug); err != nil {
		t.Fatalf("unexpected error re-archiving spec: %v", err)
	}
	scaffoldSpec(t, root, slug)

	_, err = executeRestore(t, root, slug)
	if err == nil {
		t.Fatal("expected error restoring into an existing active slug, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "conflict") {
		t.Errorf("expected restore collision error to mention 'conflict', got: %q", err.Error())
	}
}

func Test_RestoreCmd_ac1_MissingSlugArg_ReturnsError(t *testing.T) {
	root := t.TempDir()

	_, err := executeRestore(t, root)
	if err == nil {
		t.Fatal("expected error when no slug arg provided, got nil")
	}
}

func Test_RestoreCmd_ac1_InvalidSlug_ReturnsError(t *testing.T) {
	root := t.TempDir()

	_, err := executeRestore(t, root, "Not_A_Valid_Slug!")
	if err == nil {
		t.Fatal("expected error when slug is invalid, got nil")
	}
}
