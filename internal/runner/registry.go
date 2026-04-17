package runner

import (
	"fmt"
	"sync"

	"github.com/pragmataW/tddmaster/internal/state"
)

// registry is the process-global runner map. Tests use Reset() to
// isolate between cases; production init() functions call MustRegister
// exactly once at startup.
var (
	regMu    sync.RWMutex
	registry = make(map[string]Runner)
)

// Register associates r with its Name(). Returns ErrDuplicateRunner if
// a runner with the same name is already present.
func Register(r Runner) error {
	regMu.Lock()
	defer regMu.Unlock()
	name := r.Name()
	if _, exists := registry[name]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateRunner, name)
	}
	registry[name] = r
	return nil
}

// MustRegister panics on duplicate. Intended for package-init calls
// where duplicates indicate a programmer error.
func MustRegister(r Runner) {
	if err := Register(r); err != nil {
		panic(err)
	}
}

// Get returns the runner registered under name, or ErrRunnerNotFound.
func Get(name string) (Runner, error) {
	regMu.RLock()
	defer regMu.RUnlock()
	r, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrRunnerNotFound, name)
	}
	return r, nil
}

// Select picks a runner using the documented priority chain:
//
//  1. toolFlag, if non-empty — MUST resolve or ErrRunnerNotFound
//     (user explicitly asked for this runner; no silent fallback).
//  2. manifest.DefaultRunner, if non-empty and registered.
//     Unknown DefaultRunner falls through to step 3.
//  3. manifest.Tools[0], if the slice is non-empty and that name
//     is registered. Unknown entries fall through to step 4.
//  4. "claude-code" fallback, if registered.
//
// Select NEVER calls Available() — it is a cheap lookup. The caller
// is responsible for preflight.
func Select(manifest *state.NosManifest, toolFlag string) (Runner, error) {
	// Step 1: explicit flag wins and errors if unregistered.
	if toolFlag != "" {
		return Get(toolFlag)
	}

	// Step 2: DefaultRunner from manifest (if set and registered).
	if manifest != nil && manifest.DefaultRunner != "" {
		if r, err := Get(manifest.DefaultRunner); err == nil {
			return r, nil
		}
		// Unknown DefaultRunner falls through to Tools[0].
	}

	// Step 3: manifest.Tools[0] if registered.
	if manifest != nil && len(manifest.Tools) > 0 {
		if r, err := Get(string(manifest.Tools[0])); err == nil {
			return r, nil
		}
		// Fall through when Tools[0] is unknown — we continue to fallback.
	}

	// Step 4: claude-code fallback.
	return Get("claude-code")
}

// Reset clears the registry. Test-only — do not call from production code.
func Reset() {
	regMu.Lock()
	defer regMu.Unlock()
	registry = make(map[string]Runner)
}
