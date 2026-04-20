package cmd

import (
	"testing"

	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/spf13/cobra"
)

// buildCmd creates a fresh init cobra.Command with --skip-verify and --tdd-enabled flags
// registered, so tests can simulate flag parsing without running the full CLI.
func buildInitCmdForTest() *cobra.Command {
	cmd := &cobra.Command{
		Use: "init",
	}
	cmd.Flags().Bool("skip-verify", false, "Skip verifier sub-agent")
	cmd.Flags().Bool("tdd-enabled", true, "Enable TDD workflow")
	return cmd
}

// parseFlags simulates the user supplying flags to the command.
// Only flags listed in args are marked as Changed by cobra.
func parseFlags(t *testing.T, cmd *cobra.Command, args []string) {
	t.Helper()
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}
}

// nosManifestWithTdd constructs a NosManifest carrying a Tdd sub-block,
// which represents what ReadManifest returns for an already-initialised project.
func nosManifestWithTdd(tddMode, skipVerify bool) *state.NosManifest {
	return &state.NosManifest{
		Tdd: &state.Manifest{
			TddMode:    tddMode,
			SkipVerify: skipVerify,
		},
	}
}

// ---------------------------------------------------------------------------
// resolveTddSettings — unit tests for the helper that determines the effective
// (tddMode, skipVerify) pair from flags + existing manifest, according to the
// priority rule: user flag > existing manifest > default.
// ---------------------------------------------------------------------------

func TestResolveTddSettings_NoFlags_NilManifest_ReturnsDefaults(t *testing.T) {
	cmd := buildInitCmdForTest()

	tddMode, skipVerify := resolveTddSettings(cmd, nil)

	if !tddMode {
		t.Errorf("expected tddMode=true (default), got false")
	}
	if skipVerify {
		t.Errorf("expected skipVerify=false (default), got true")
	}
}

func TestResolveTddSettings_NoFlags_ExistingManifest_PreservesExisting(t *testing.T) {
	tests := []struct {
		name            string
		existingTdd     bool
		existingSkip    bool
		wantTddMode     bool
		wantSkipVerify  bool
	}{
		{
			name:           "skip=true tdd=false",
			existingTdd:    false,
			existingSkip:   true,
			wantTddMode:    false,
			wantSkipVerify: true,
		},
		{
			name:           "skip=false tdd=true",
			existingTdd:    true,
			existingSkip:   false,
			wantTddMode:    true,
			wantSkipVerify: false,
		},
		{
			name:           "both true",
			existingTdd:    true,
			existingSkip:   true,
			wantTddMode:    true,
			wantSkipVerify: true,
		},
		{
			name:           "both false",
			existingTdd:    false,
			existingSkip:   false,
			wantTddMode:    false,
			wantSkipVerify: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := buildInitCmdForTest()
			existing := nosManifestWithTdd(tc.existingTdd, tc.existingSkip)

			tddMode, skipVerify := resolveTddSettings(cmd, existing)

			if tddMode != tc.wantTddMode {
				t.Errorf("tddMode: want %v, got %v", tc.wantTddMode, tddMode)
			}
			if skipVerify != tc.wantSkipVerify {
				t.Errorf("skipVerify: want %v, got %v", tc.wantSkipVerify, skipVerify)
			}
		})
	}
}

func TestResolveTddSettings_SkipVerifyFlag_OverridesExistingFalse(t *testing.T) {
	cmd := buildInitCmdForTest()
	// existing manifest has skipVerify=false; flag overrides to true
	existing := nosManifestWithTdd(true, false)
	parseFlags(t, cmd, []string{"--skip-verify=true"})

	_, skipVerify := resolveTddSettings(cmd, existing)

	if !skipVerify {
		t.Errorf("expected skipVerify=true (flag override), got false")
	}
}

func TestResolveTddSettings_TddEnabledFlag_OverridesExistingTrue(t *testing.T) {
	cmd := buildInitCmdForTest()
	// existing manifest has tddMode=true; flag overrides to false
	existing := nosManifestWithTdd(true, false)
	parseFlags(t, cmd, []string{"--tdd-enabled=false"})

	tddMode, _ := resolveTddSettings(cmd, existing)

	if tddMode {
		t.Errorf("expected tddMode=false (flag override), got true")
	}
}

func TestResolveTddSettings_BothFlags_BothOverride(t *testing.T) {
	cmd := buildInitCmdForTest()
	// existing manifest has tddMode=true, skipVerify=false
	existing := nosManifestWithTdd(true, false)
	parseFlags(t, cmd, []string{"--tdd-enabled=false", "--skip-verify=true"})

	tddMode, skipVerify := resolveTddSettings(cmd, existing)

	if tddMode {
		t.Errorf("expected tddMode=false (flag override), got true")
	}
	if !skipVerify {
		t.Errorf("expected skipVerify=true (flag override), got false")
	}
}

func TestResolveTddSettings_FlagsWithNilManifest_FlagWins(t *testing.T) {
	cmd := buildInitCmdForTest()
	parseFlags(t, cmd, []string{"--skip-verify=true", "--tdd-enabled=false"})

	tddMode, skipVerify := resolveTddSettings(cmd, nil)

	if tddMode {
		t.Errorf("expected tddMode=false (flag), got true")
	}
	if !skipVerify {
		t.Errorf("expected skipVerify=true (flag), got false")
	}
}

func TestResolveTddSettings_ExistingManifestNilTddBlock_ReturnsDefaults(t *testing.T) {
	cmd := buildInitCmdForTest()
	// NosManifest exists but Tdd sub-block is nil (older project without TDD config)
	existing := &state.NosManifest{Tdd: nil}

	tddMode, skipVerify := resolveTddSettings(cmd, existing)

	if !tddMode {
		t.Errorf("expected tddMode=true (default when Tdd block absent), got false")
	}
	if skipVerify {
		t.Errorf("expected skipVerify=false (default when Tdd block absent), got true")
	}
}

// ---------------------------------------------------------------------------
// newInitCmd — flag registration tests (AC-3, AC-4)
// ---------------------------------------------------------------------------

func TestNewInitCmd_RegistersSkipVerifyFlag(t *testing.T) {
	cmd := newInitCmd()

	f := cmd.Flags().Lookup("skip-verify")
	if f == nil {
		t.Fatal("expected --skip-verify flag to be registered on init command")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("expected --skip-verify to be bool, got %s", f.Value.Type())
	}
}

func TestNewInitCmd_RegistersTddEnabledFlag(t *testing.T) {
	cmd := newInitCmd()

	f := cmd.Flags().Lookup("tdd-enabled")
	if f == nil {
		t.Fatal("expected --tdd-enabled flag to be registered on init command")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("expected --tdd-enabled to be bool, got %s", f.Value.Type())
	}
}

func TestNewInitCmd_FlagChangedOnlyWhenExplicitlySet(t *testing.T) {
	cmd := newInitCmd()
	// No flags passed — neither should be marked as Changed
	if err := cmd.ParseFlags([]string{}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}

	if cmd.Flags().Changed("skip-verify") {
		t.Error("skip-verify should NOT be marked Changed when not supplied")
	}
	if cmd.Flags().Changed("tdd-enabled") {
		t.Error("tdd-enabled should NOT be marked Changed when not supplied")
	}
}

func TestNewInitCmd_FlagChangedAfterExplicitSet(t *testing.T) {
	cmd := newInitCmd()
	if err := cmd.ParseFlags([]string{"--skip-verify=true"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}

	if !cmd.Flags().Changed("skip-verify") {
		t.Error("skip-verify SHOULD be marked Changed when explicitly supplied")
	}
	if cmd.Flags().Changed("tdd-enabled") {
		t.Error("tdd-enabled should NOT be marked Changed when not supplied")
	}
}
