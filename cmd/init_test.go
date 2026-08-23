package cmd

import (
	"os"
	"testing"
)

func TestIsTTY_DevNullIsNotATerminal(t *testing.T) {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	defer devNull.Close()

	// /dev/null is a character device, so a ModeCharDevice check would wrongly
	// report a terminal and let the interactive form open and then fail.
	if readerIsTTY(devNull) {
		t.Fatal("/dev/null must not be treated as a terminal")
	}
}

func TestInit_NonTTYWithoutFlag_FailsBeforeOpeningTheForm(t *testing.T) {
	root := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	cmd := newInitCmd()
	cmd.SetArgs(nil)
	cmd.SetOut(os.NewFile(0, os.DevNull))
	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected a non-interactive error when stdin is not a terminal")
	}
	if _, statErr := os.Stat(root + "/.tddmaster"); statErr == nil {
		t.Fatal("init must not scaffold anything when it refuses to run")
	}
}
