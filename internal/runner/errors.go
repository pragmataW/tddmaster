package runner

import "errors"

// ErrBinaryNotFound is returned by Runner.Available and (wrapped) by
// Runner.Invoke when the CLI binary cannot be resolved on PATH.
// See EC-1. Callers should map this to a tddmaster block transition
// with a user-facing "install <cli>" message.
var ErrBinaryNotFound = errors.New("runner: CLI binary not found on PATH")

// ErrInvalidArgs is returned when the RunRequest contains combinations
// the adapter cannot honor (e.g. empty Prompt).
var ErrInvalidArgs = errors.New("runner: invalid arguments")

// ErrNonZeroExit marks a CLI invocation that spawned successfully but
// returned a non-zero exit code. Callers that treat any failure as
// fatal can check errors.Is; callers that want to inspect the body
// can read RunResult. This is NOT returned by default — Invoke returns
// nil err on non-zero exit. It is provided for adapters / callers that
// explicitly want to wrap it. See EC-3.
var ErrNonZeroExit = errors.New("runner: CLI returned non-zero exit")

// ErrJSONParse signals that the adapter could not decode the CLI's
// terminal output into structured JSON. Callers should fall back to
// RunResult.Stdout raw bytes. See EC-4.
var ErrJSONParse = errors.New("runner: failed to parse CLI JSON output")

// ErrContextCanceled wraps context.Canceled / context.DeadlineExceeded
// so callers can distinguish a user-initiated cancel from a CLI crash.
// See EC-6.
var ErrContextCanceled = errors.New("runner: invocation canceled by context")

// ErrDuplicateRunner is returned by Register when a runner with the same
// Name() is already registered. MustRegister panics with this error.
var ErrDuplicateRunner = errors.New("runner: duplicate registration")

// ErrRunnerNotFound is returned by Get when the requested name is not
// registered, and by Select when the priority chain resolves to nothing.
var ErrRunnerNotFound = errors.New("runner: not found")
