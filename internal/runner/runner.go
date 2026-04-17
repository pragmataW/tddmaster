// Package runner abstracts single-shot invocations of AI coding CLIs
// (claude, codex, opencode, ...). Each implementation wraps one CLI binary
// and translates a [RunRequest] into that CLI's argv/env.
//
// Research documenting the field-level differences between the three
// supported CLIs (NDJSON stream vs single JSON blob, approval flags,
// subcommand names) lives at:
//
//	docs/research/RESEARCH-CLIS.md
package runner

import "context"

// Runner abstracts a single-shot invocation of an AI coding CLI
// (claude, codex, opencode, ...). Each implementation wraps one
// CLI binary and translates RunRequest into that CLI's argv/env.
type Runner interface {
	// Name returns the canonical runner identifier used in manifest
	// (matches state.CodingToolId values: "claude-code", "codex", "opencode").
	Name() string

	// Available performs a cheap preflight check — typically exec.LookPath
	// on the binary. Returns nil if the CLI is usable, a wrapped
	// ErrBinaryNotFound otherwise. Does NOT spawn the CLI.
	Available() error

	// Invoke spawns the CLI, streams output, and returns the structured
	// result. The caller is responsible for any retry policy.
	// Context cancellation MUST propagate to the child process
	// (SIGINT with grace period, then Kill). See EC-6 in spec.
	Invoke(ctx context.Context, req RunRequest) (*RunResult, error)
}
