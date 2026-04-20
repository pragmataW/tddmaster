package tdd

import (
	"testing"

	statemodel "github.com/pragmataW/tddmaster/internal/state/model"
)

// helper: build a NosManifest with explicit skipVerify and tddMode flags.
func manifestWith(tddMode, skipVerify bool) *statemodel.NosManifest {
	return &statemodel.NosManifest{
		Tdd: &statemodel.Manifest{
			TddMode:    tddMode,
			SkipVerify: skipVerify,
		},
	}
}

// TestVerifierRequired_TableDriven covers AC-1 through AC-5 for the
// VerifierRequired(manifest, phase) function.
func TestVerifierRequired_TableDriven(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		manifest *statemodel.NosManifest
		phase    string
		want     bool
	}{
		// AC-1: skipVerify=false always returns true regardless of TDD/phase.
		{
			name:     "skipVerify=false tdd=on phase=red → true",
			manifest: manifestWith(true, false),
			phase:    statemodel.TDDCycleRed,
			want:     true,
		},
		{
			name:     "skipVerify=false tdd=on phase=green → true",
			manifest: manifestWith(true, false),
			phase:    statemodel.TDDCycleGreen,
			want:     true,
		},
		{
			name:     "skipVerify=false tdd=on phase=refactor → true",
			manifest: manifestWith(true, false),
			phase:    statemodel.TDDCycleRefactor,
			want:     true,
		},
		{
			name:     "skipVerify=false tdd=off phase=empty → true",
			manifest: manifestWith(false, false),
			phase:    "",
			want:     true,
		},
		// AC-2: skipVerify=true + TDD=off → false.
		{
			name:     "skipVerify=true tdd=off phase=empty → false",
			manifest: manifestWith(false, true),
			phase:    "",
			want:     false,
		},
		// AC-4: skipVerify=true + TDD=on + phase=red → false.
		{
			name:     "skipVerify=true tdd=on phase=red → false",
			manifest: manifestWith(true, true),
			phase:    statemodel.TDDCycleRed,
			want:     false,
		},
		// AC-3: skipVerify=true + TDD=on + phase=green → true.
		{
			name:     "skipVerify=true tdd=on phase=green → true",
			manifest: manifestWith(true, true),
			phase:    statemodel.TDDCycleGreen,
			want:     true,
		},
		// AC-4: skipVerify=true + TDD=on + phase=refactor → false.
		{
			name:     "skipVerify=true tdd=on phase=refactor → false",
			manifest: manifestWith(true, true),
			phase:    statemodel.TDDCycleRefactor,
			want:     false,
		},
		// AC-5: manifest=nil → nil-safe, no skip flag → true.
		{
			name:     "manifest=nil → true (no skip flag, treat as verifier required)",
			manifest: nil,
			phase:    "",
			want:     true,
		},
		// AC-5: manifest.Tdd=nil → treat as TDD not enabled and no skipVerify → true.
		{
			name:     "manifest.Tdd=nil → true",
			manifest: &statemodel.NosManifest{Tdd: nil},
			phase:    statemodel.TDDCycleRed,
			want:     true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := VerifierRequired(tc.manifest, tc.phase)
			if got != tc.want {
				t.Errorf("VerifierRequired(%+v, %q) = %v; want %v", tc.manifest, tc.phase, got, tc.want)
			}
		})
	}
}
