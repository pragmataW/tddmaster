package errs

import (
	"errors"
	"strings"
	"testing"
)

func TestTemplatesNonEmpty(t *testing.T) {
	for key, tpl := range templateMap {
		if strings.TrimSpace(tpl) == "" {
			t.Errorf("template for key %q is empty", key)
		}
	}
}

func TestNew(t *testing.T) {
	err := New(KeyInvalidJSONAnswer)
	if err.Error() != "invalid JSON answer" {
		t.Fatalf("got %q", err.Error())
	}
}

func TestNewf(t *testing.T) {
	err := Newf(KeySpecNotFoundRunStart, "foo", "foo")
	want := "spec \"foo\" not found: run tddmaster start foo first"
	if err.Error() != want {
		t.Fatalf("got %q want %q", err.Error(), want)
	}
}

func TestWrapNoArgs(t *testing.T) {
	inner := errors.New("boom")
	err := Wrap(KeyResolveRoot, inner)
	if err.Error() != "resolve root: boom" {
		t.Fatalf("got %q", err.Error())
	}
	if !errors.Is(err, inner) {
		t.Fatal("Wrap must preserve the wrapped error for errors.Is")
	}
}

func TestWrapWithArgs(t *testing.T) {
	inner := errors.New("boom")
	err := Wrap(KeyAdapterWriteAgent, inner, "codex", "planner.md")
	want := "write codex agent planner.md: boom"
	if err.Error() != want {
		t.Fatalf("got %q want %q", err.Error(), want)
	}
}

func TestSentinelMatchesByKey(t *testing.T) {
	err := New(KeyInvalidJSONAnswer)
	if !errors.Is(err, Sentinel(KeyInvalidJSONAnswer)) {
		t.Fatal("errors.Is must match the sentinel of the same key")
	}
	if errors.Is(err, Sentinel(KeyResolveRoot)) {
		t.Fatal("errors.Is must not match a different key")
	}
}

func TestUnknownKeyPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for unregistered key")
		}
	}()
	_ = New(ErrorKey("errs:does-not-exist"))
}
