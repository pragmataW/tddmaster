package runner

import (
	"context"
	"errors"
	"testing"

	"github.com/pragmataW/tddmaster/internal/state"
)

// fakeRunner is an in-package test double. It implements Runner with
// configurable Name and Available behavior; Invoke always succeeds.
type fakeRunner struct {
	name     string
	availErr error
}

func (f *fakeRunner) Name() string     { return f.name }
func (f *fakeRunner) Available() error { return f.availErr }
func (f *fakeRunner) Invoke(_ context.Context, _ RunRequest) (*RunResult, error) {
	return &RunResult{}, nil
}

// mustRegisterFake is a test helper that calls MustRegister with a fakeRunner
// built from the given name and availErr. t.Helper() so failures pin to callers.
func mustRegisterFake(t *testing.T, name string, availErr error) *fakeRunner {
	t.Helper()
	r := &fakeRunner{name: name, availErr: availErr}
	MustRegister(r)
	return r
}

// manifestWithTools builds a NosManifest whose Tools slice is set to the
// provided CodingToolId values. All other fields are zero.
func manifestWithTools(tools ...state.CodingToolId) *state.NosManifest {
	return &state.NosManifest{Tools: tools}
}

// manifestWithDefaultRunner builds a NosManifest with DefaultRunner set to dr
// and Tools set to the provided values. This will fail to compile until
// NosManifest.DefaultRunner is added by the executor (RED guard).
func manifestWithDefaultRunner(dr string, tools ...state.CodingToolId) *state.NosManifest {
	return &state.NosManifest{DefaultRunner: dr, Tools: tools}
}

// ---------------------------------------------------------------------------
// Register
// ---------------------------------------------------------------------------

func TestRegister_Success(t *testing.T) {
	Reset()

	r := &fakeRunner{name: "claude-code"}
	if err := Register(r); err != nil {
		t.Fatalf("Register first time: want nil error, got %v", err)
	}
}

func TestRegister_Duplicate_ReturnsErrDuplicateRunner(t *testing.T) {
	Reset()

	r := &fakeRunner{name: "claude-code"}
	if err := Register(r); err != nil {
		t.Fatalf("first Register: %v", err)
	}

	r2 := &fakeRunner{name: "claude-code"}
	err := Register(r2)
	if !errors.Is(err, ErrDuplicateRunner) {
		t.Fatalf("second Register: want ErrDuplicateRunner, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// MustRegister
// ---------------------------------------------------------------------------

func TestMustRegister_PanicsOnDuplicate(t *testing.T) {
	Reset()

	MustRegister(&fakeRunner{name: "codex"})

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustRegister duplicate: expected panic, got none")
		}
	}()
	MustRegister(&fakeRunner{name: "codex"})
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestGet_Found(t *testing.T) {
	Reset()

	mustRegisterFake(t, "opencode", nil)

	got, err := Get("opencode")
	if err != nil {
		t.Fatalf("Get registered runner: want nil error, got %v", err)
	}
	if got.Name() != "opencode" {
		t.Fatalf("Get: want name %q, got %q", "opencode", got.Name())
	}
}

func TestGet_NotFound_ReturnsErrRunnerNotFound(t *testing.T) {
	Reset()

	_, err := Get("does-not-exist")
	if !errors.Is(err, ErrRunnerNotFound) {
		t.Fatalf("Get unknown: want ErrRunnerNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Select — table-driven
// ---------------------------------------------------------------------------

func TestSelect(t *testing.T) {
	// Each row is independent; the subtest resets the registry before setup.

	tests := []struct {
		name     string
		setup    func(t *testing.T) // register runners into clean registry
		manifest *state.NosManifest
		toolFlag string
		wantName string // non-empty means we expect success with this runner name
		wantErr  error  // non-nil means we expect errors.Is(err, wantErr)
	}{
		// --- toolFlag takes highest priority ---
		{
			name: "toolFlag registered returns that runner",
			setup: func(t *testing.T) {
				t.Helper()
				mustRegisterFake(t, "codex", nil)
				mustRegisterFake(t, "claude-code", nil)
			},
			manifest: manifestWithTools(state.CodingToolClaudeCode),
			toolFlag: "codex",
			wantName: "codex",
		},

		// --- toolFlag set but unknown → ErrRunnerNotFound (no silent fallback) ---
		{
			name: "toolFlag set but unregistered returns ErrRunnerNotFound",
			setup: func(t *testing.T) {
				t.Helper()
				mustRegisterFake(t, "claude-code", nil)
			},
			manifest: manifestWithTools(state.CodingToolClaudeCode),
			toolFlag: "ghost-runner",
			wantErr:  ErrRunnerNotFound,
		},

		// --- step 2: DefaultRunner (task-11 field) wins over Tools[0] ---

		// AC-10: DefaultRunner set and registered is returned instead of Tools[0].
		{
			name: "manifest DefaultRunner picked over Tools[0] when no toolFlag",
			setup: func(t *testing.T) {
				t.Helper()
				mustRegisterFake(t, "codex", nil)
				mustRegisterFake(t, "claude-code", nil)
			},
			manifest: manifestWithDefaultRunner("codex", state.CodingToolClaudeCode),
			toolFlag: "",
			wantName: "codex",
		},

		// AC-11: DefaultRunner set but unregistered falls through to Tools[0].
		{
			name: "manifest DefaultRunner unknown falls to Tools[0]",
			setup: func(t *testing.T) {
				t.Helper()
				mustRegisterFake(t, "claude-code", nil)
			},
			manifest: manifestWithDefaultRunner("ghost-runner", state.CodingToolClaudeCode),
			toolFlag: "",
			wantName: "claude-code",
		},

		// AC-12: explicit toolFlag still wins over DefaultRunner.
		{
			name: "toolFlag wins over DefaultRunner",
			setup: func(t *testing.T) {
				t.Helper()
				mustRegisterFake(t, "codex", nil)
				mustRegisterFake(t, "claude-code", nil)
			},
			manifest: manifestWithDefaultRunner("codex", state.CodingToolClaudeCode),
			toolFlag: "claude-code",
			wantName: "claude-code",
		},

		// --- step 3: manifest.Tools[0] when registered and no toolFlag ---
		{
			name: "manifest Tools[0] registered picked when no toolFlag",
			setup: func(t *testing.T) {
				t.Helper()
				mustRegisterFake(t, "opencode", nil)
				mustRegisterFake(t, "claude-code", nil)
			},
			manifest: manifestWithTools(state.CodingToolOpencode, state.CodingToolClaudeCode),
			toolFlag: "",
			wantName: "opencode",
		},

		// --- step 3 fallthrough: manifest.Tools[0] unregistered → falls to claude-code ---
		{
			name: "manifest Tools[0] unknown falls through to claude-code fallback",
			setup: func(t *testing.T) {
				t.Helper()
				mustRegisterFake(t, "claude-code", nil)
			},
			manifest: manifestWithTools("unknown-tool"),
			toolFlag: "",
			wantName: "claude-code",
		},

		// --- step 4: claude-code fallback ---
		{
			name: "no toolFlag and empty Tools picks claude-code fallback",
			setup: func(t *testing.T) {
				t.Helper()
				mustRegisterFake(t, "claude-code", nil)
			},
			manifest: manifestWithTools(), // empty Tools slice
			toolFlag: "",
			wantName: "claude-code",
		},

		// --- nil manifest + empty flag → claude-code fallback ---
		{
			name: "nil manifest and empty toolFlag picks claude-code fallback when registered",
			setup: func(t *testing.T) {
				t.Helper()
				mustRegisterFake(t, "claude-code", nil)
			},
			manifest: nil,
			toolFlag: "",
			wantName: "claude-code",
		},

		// --- nil manifest + empty flag but claude-code absent → ErrRunnerNotFound ---
		{
			name: "nil manifest and empty toolFlag and no claude-code returns ErrRunnerNotFound",
			setup: func(t *testing.T) {
				t.Helper()
				mustRegisterFake(t, "codex", nil)
			},
			manifest: nil,
			toolFlag: "",
			wantErr:  ErrRunnerNotFound,
		},

		// --- backward-compat zero-config ---
		{
			name: "backward compat zero-config: Tools=[claude-code] no flag returns claude-code",
			setup: func(t *testing.T) {
				t.Helper()
				mustRegisterFake(t, "claude-code", nil)
			},
			manifest: manifestWithTools(state.CodingToolClaudeCode),
			toolFlag: "",
			wantName: "claude-code",
		},

		// --- manifest.Tools empty and no toolFlag and no claude-code registered ---
		{
			name: "empty Tools no toolFlag no fallback runner returns ErrRunnerNotFound",
			setup: func(t *testing.T) {
				t.Helper()
				mustRegisterFake(t, "codex", nil) // codex registered, but not in chain
			},
			manifest: manifestWithTools(),
			toolFlag: "",
			wantErr:  ErrRunnerNotFound,
		},

		// --- EC-1: Available() is never called by Select; runner returned even if unavailable ---
		{
			name: "EC-1 registry ignores Available: runner with ErrBinaryNotFound still returned by Select",
			setup: func(t *testing.T) {
				t.Helper()
				// Register a runner whose Available() would fail.
				mustRegisterFake(t, "claude-code", ErrBinaryNotFound)
			},
			manifest: manifestWithTools(state.CodingToolClaudeCode),
			toolFlag: "",
			wantName: "claude-code", // Select returns it; caller decides whether to preflight
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			Reset()
			if tc.setup != nil {
				tc.setup(t)
			}

			got, err := Select(tc.manifest, tc.toolFlag)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("Select(%q): want error %v, got %v", tc.toolFlag, tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Select(%q): unexpected error %v", tc.toolFlag, err)
			}
			if got.Name() != tc.wantName {
				t.Fatalf("Select(%q): want runner %q, got %q", tc.toolFlag, tc.wantName, got.Name())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Select does not call Available() — explicit isolation test (EC-1)
// ---------------------------------------------------------------------------

// TestSelect_DoesNotCallAvailable verifies that Select is a pure registry
// lookup and never invokes Available() on any registered runner. We detect
// Available() calls via a spy runner that records whether Available was called.
func TestSelect_DoesNotCallAvailable(t *testing.T) {
	Reset()

	spy := &spyRunner{fakeRunner: fakeRunner{name: "claude-code", availErr: ErrBinaryNotFound}}
	MustRegister(spy)

	_, err := Select(manifestWithTools(state.CodingToolClaudeCode), "")
	if err != nil {
		t.Fatalf("Select: unexpected error %v", err)
	}
	if spy.availableCalled {
		t.Error("Select called Available() — it must not; caller is responsible for preflighting")
	}
}

// spyRunner wraps fakeRunner and records whether Available was called.
type spyRunner struct {
	fakeRunner
	availableCalled bool
}

func (s *spyRunner) Name() string { return s.fakeRunner.name }
func (s *spyRunner) Available() error {
	s.availableCalled = true
	return s.fakeRunner.availErr
}
func (s *spyRunner) Invoke(_ context.Context, _ RunRequest) (*RunResult, error) {
	return &RunResult{}, nil
}
