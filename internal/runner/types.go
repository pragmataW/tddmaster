package runner

import "io"

// RunRequest carries all inputs common to every CLI adapter.
// Per-CLI extras that do NOT fit the common shape go into ExtraArgs
// (passed verbatim after the adapter's own flags).
type RunRequest struct {
	// Prompt is the user-facing instruction to send to the agent.
	// Required. Adapters map this to the CLI's prompt flag
	// (claude: -p, codex: exec <prompt>, opencode: run <prompt>).
	Prompt string

	// SystemPrompt is an optional system-level instruction. Adapters
	// that don't support a first-class system flag may prepend it
	// to Prompt or ignore it (document in the adapter).
	SystemPrompt string

	// MaxTurns caps agent iterations inside a single CLI invocation.
	// Zero means "adapter default".
	MaxTurns int

	// OutputFormat is the adapter-agnostic hint. Common values:
	// "json" (single blob, Claude), "ndjson" (streaming, Codex/OpenCode),
	// "text". Adapter is free to pick its nearest equivalent.
	OutputFormat string

	// WorkDir sets the spawned process working directory. Empty means inherit.
	WorkDir string

	// Env is an optional override list (KEY=VAL) merged onto the parent env.
	// Nil means inherit parent environment unchanged.
	Env []string

	// Stdout / Stderr are optional writers. When nil, the runner buffers
	// output and returns it in RunResult.Stdout / RunResult.Stderr.
	// When set, output is streamed live to these writers AND still
	// captured into RunResult for callers that need the full buffer.
	Stdout io.Writer
	Stderr io.Writer

	// ExtraArgs are appended verbatim to the CLI argv. Used for per-CLI
	// knobs that do not belong in the common shape (e.g. --ask-for-approval
	// for codex). Adapters may also inject their own flags — ExtraArgs
	// NEVER overrides adapter-owned flags; the adapter is authoritative.
	ExtraArgs []string
}

// RunResult is what every adapter returns on a successful spawn.
// Non-zero ExitCode is NOT an error on its own — Invoke returns nil
// for err when the CLI spawned cleanly, regardless of exit code. The
// caller decides whether a non-zero exit is fatal.
type RunResult struct {
	// ExitCode is the child process exit code (0 on success).
	ExitCode int

	// Stdout is the full captured stdout. Populated whether or not
	// RunRequest.Stdout was supplied.
	Stdout []byte

	// Stderr is the full captured stderr. Populated whether or not
	// RunRequest.Stderr was supplied.
	Stderr []byte

	// ParsedJSON is the adapter's best-effort decode of the final
	// completion event / blob into a map. Nil when the adapter could
	// not parse the output (e.g. plain-text CLI), or when the CLI
	// emitted no parseable JSON. Callers should fall back to Stdout.
	ParsedJSON map[string]any
}
