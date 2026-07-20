package cmd

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/pragmataW/tddmaster/internal/spec"
)

func executeCancel(t *testing.T, root string, extraArgs ...string) (string, error) {
	t.Helper()
	root_cmd := newRootCmd()
	var buf bytes.Buffer
	root_cmd.SetOut(&buf)
	root_cmd.SetErr(&buf)
	root_cmd.SetIn(strings.NewReader(""))
	args := append([]string{"cancel"}, extraArgs...)
	if root != "" {
		args = append(args, "--root", root)
	}
	root_cmd.SetArgs(args)
	err := root_cmd.Execute()
	return buf.String(), err
}

func Test_CancelCmd_ac1_Use(t *testing.T) {
	cmd := newCancelCmd()
	if cmd.Use == "" || !strings.HasPrefix(cmd.Use, "cancel") {
		t.Errorf("newCancelCmd().Use = %q, want it to start with \"cancel\"", cmd.Use)
	}
}

func Test_CancelCmd_ac1_Args_NoSlug_ReturnsError(t *testing.T) {
	root := t.TempDir()
	_, err := executeCancel(t, root)
	if err == nil {
		t.Fatal("expected error when no slug arg provided, got nil")
	}
}

func Test_CancelCmd_ac1_Args_TooManyArgs_ReturnsError(t *testing.T) {
	root := t.TempDir()
	_, err := executeCancel(t, root, "slug-one", "slug-two")
	if err == nil {
		t.Fatal("expected error when more than one slug arg provided, got nil")
	}
}

func Test_CancelCmd_ac1_HasForceFlag(t *testing.T) {
	cmd := newCancelCmd()
	flag := cmd.Flags().Lookup("force")
	if flag == nil {
		t.Fatal("--force flag not registered on 'cancel' command")
	}
}

func Test_CancelCmd_ac1_RegisteredOnRoot(t *testing.T) {
	root := newRootCmd()
	var found bool
	for _, sub := range root.Commands() {
		if sub.Name() == "cancel" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'cancel' subcommand registered on root, but not found")
	}
}

func Test_CancelCmd_ac1_Force_RemovesSpecDir(t *testing.T) {
	root := t.TempDir()
	slug := "cancel-force-spec"
	scaffoldSpec(t, root, slug)

	if !spec.Exists(root, slug) {
		t.Fatalf("precondition failed: spec %q does not exist after scaffolding", slug)
	}

	_, err := executeCancel(t, root, slug, "--force")
	if err != nil {
		t.Fatalf("unexpected error running cancel --force: %v", err)
	}
	if spec.Exists(root, slug) {
		t.Errorf("expected spec %q to be removed after cancel --force, but it still exists", slug)
	}
}

func Test_CancelCmd_ac1_InvalidSlug_ReturnsError_NothingDeleted(t *testing.T) {
	root := t.TempDir()

	_, err := executeCancel(t, root, "Not_A_Valid_Slug!", "--force")
	if err == nil {
		t.Fatal("expected error for invalid slug, got nil")
	}
}

func Test_CancelCmd_ac1_NonexistentSlug_ReturnsError_NothingDeleted(t *testing.T) {
	root := t.TempDir()
	slug := "does-not-exist"

	_, err := executeCancel(t, root, slug, "--force")
	if err == nil {
		t.Fatal("expected error when cancelling a spec that does not exist, got nil")
	}
	if spec.Exists(root, slug) {
		t.Errorf("expected nonexistent spec %q to remain nonexistent after failed cancel", slug)
	}
}

func Test_CancelCmd_ec1_NonTTYWithoutForce_DoesNotDeleteSpecDir(t *testing.T) {
	root := t.TempDir()
	slug := "cancel-nontty-spec"
	scaffoldSpec(t, root, slug)

	_, _ = executeCancel(t, root, slug)

	if !spec.Exists(root, slug) {
		t.Errorf("expected spec %q to remain intact after cancel without --force in a non-TTY environment (unconfirmed deletion must never happen)", slug)
	}
}

func Test_CancelCmd_ec1_NonTTYWithoutForce_ReturnsErrorAndLeavesDirIntact(t *testing.T) {
	root := t.TempDir()
	slug := "cancel-nontty-error-spec"
	scaffoldSpec(t, root, slug)

	_, err := executeCancel(t, root, slug)
	if err == nil {
		t.Fatal("expected error when cancel is run without --force in a non-TTY environment, got nil")
	}
	if !spec.Exists(root, slug) {
		t.Errorf("expected spec %q to remain intact after non-TTY cancel without --force", slug)
	}
}

func withCancelConfirmOverride(t *testing.T, fn func(slug string, in io.Reader, out io.Writer) (bool, error)) {
	t.Helper()
	original := cancelConfirm
	cancelConfirm = fn
	t.Cleanup(func() {
		cancelConfirm = original
	})
}

func Test_CancelCmd_cov_ac1_ConfirmTrue_RemovesSpecDirAndPrintsSuccess(t *testing.T) {
	root := t.TempDir()
	slug := "cancel-confirm-true-spec"
	scaffoldSpec(t, root, slug)

	withCancelConfirmOverride(t, func(slug string, in io.Reader, out io.Writer) (bool, error) {
		return true, nil
	})

	out, err := executeCancel(t, root, slug)
	if err != nil {
		t.Fatalf("unexpected error when cancelConfirm returns confirmed=true: %v", err)
	}
	if spec.Exists(root, slug) {
		t.Errorf("expected spec %q to be removed when cancelConfirm returns confirmed=true, but it still exists", slug)
	}
	if !strings.Contains(out, slug) {
		t.Errorf("expected success output to mention slug %q, got %q", slug, out)
	}
}

func Test_CancelCmd_cov_ec1_ConfirmFalseNoError_DoesNotDeleteSpecDir(t *testing.T) {
	root := t.TempDir()
	slug := "cancel-confirm-false-spec"
	scaffoldSpec(t, root, slug)

	withCancelConfirmOverride(t, func(slug string, in io.Reader, out io.Writer) (bool, error) {
		return false, nil
	})

	out, err := executeCancel(t, root, slug)
	if err != nil {
		t.Fatalf("expected nil error when cancelConfirm returns confirmed=false, got: %v", err)
	}
	if !spec.Exists(root, slug) {
		t.Errorf("expected spec %q to remain intact when cancelConfirm returns confirmed=false, but it was removed", slug)
	}
	if !strings.Contains(strings.ToLower(out), "abort") {
		t.Errorf("expected an aborted/cancelled message in output, got %q", out)
	}
}

func Test_CancelCmd_cov_ec1_TypedSlugMismatch_DoesNotDeleteSpecDir(t *testing.T) {
	root := t.TempDir()
	slug := "cancel-typed-mismatch-spec"
	scaffoldSpec(t, root, slug)

	withCancelConfirmOverride(t, func(slug string, in io.Reader, out io.Writer) (bool, error) {
		return false, nil
	})

	out, err := executeCancel(t, root, slug)
	if err != nil {
		t.Fatalf("expected nil error when typed slug does not match confirmation, got: %v", err)
	}
	if !spec.Exists(root, slug) {
		t.Errorf("expected spec %q to remain intact when typed slug confirmation does not match, but it was removed", slug)
	}
	if !strings.Contains(strings.ToLower(out), "abort") {
		t.Errorf("expected an aborted/cancelled message when typed slug does not match, got %q", out)
	}
}

func Test_CancelCmd_cov_ec1_UserAbortedError_DoesNotDeleteSpecDir(t *testing.T) {
	root := t.TempDir()
	slug := "cancel-user-aborted-spec"
	scaffoldSpec(t, root, slug)

	withCancelConfirmOverride(t, func(slug string, in io.Reader, out io.Writer) (bool, error) {
		return false, huh.ErrUserAborted
	})

	out, err := executeCancel(t, root, slug)
	if err != nil {
		t.Fatalf("expected nil error when cancelConfirm returns huh.ErrUserAborted, got: %v", err)
	}
	if !spec.Exists(root, slug) {
		t.Errorf("expected spec %q to remain intact when confirmation is user-aborted, but it was removed", slug)
	}
	if !strings.Contains(strings.ToLower(out), "abort") {
		t.Errorf("expected an aborted/cancelled message when confirmation is user-aborted, got %q", out)
	}
}
