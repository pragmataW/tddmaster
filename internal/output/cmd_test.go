
package output_test

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/output"
)

func TestCmdPrefix_DefaultIsTddmaster(t *testing.T) {
	// Reset to default before test
	output.SetCommandPrefix("tddmaster")

	if got := output.CmdPrefix(); got != "tddmaster" {
		t.Errorf("CmdPrefix() = %q, want %q", got, "tddmaster")
	}
}

func TestSetCommandPrefix(t *testing.T) {
	output.SetCommandPrefix("my-cli")
	defer output.SetCommandPrefix("tddmaster") // restore

	if got := output.CmdPrefix(); got != "my-cli" {
		t.Errorf("CmdPrefix() = %q, want %q", got, "my-cli")
	}
}

func TestCmd_BuildsFullCommand(t *testing.T) {
	output.SetCommandPrefix("tddmaster")
	defer output.SetCommandPrefix("tddmaster")

	got := output.Cmd("spec list")
	want := "tddmaster spec list"
	if got != want {
		t.Errorf("Cmd() = %q, want %q", got, want)
	}
}
