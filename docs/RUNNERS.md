# Runners

## Overview

`tddmaster` supports three coding CLIs — Claude Code, Codex CLI, and OpenCode — through the `internal/runner` abstraction. Each CLI is registered as a named adapter that implements the `Runner` interface; the orchestrator dispatches every execution request to whichever adapter is currently selected, without hard-coding any binary name in the outer loop.

Runner selection follows a four-step priority chain evaluated at the start of each `tddmaster run` invocation: the `--tool` flag overrides everything; if that flag is absent, `manifest.defaultRunner` is tried; if that field is empty or unknown, the first entry in `manifest.tools` is used; and if `manifest.tools` is also empty or unresolvable, the system falls back to `claude-code`. This means an existing project with no runner configuration continues to use Claude Code with no migration required.

---

## Invocation Matrix

| Runner | Binary | Non-interactive flag | Output format | Max turns | System prompt | Streaming shape |
|---|---|---|---|---|---|---|
| `claude-code` | `claude` | `-p <prompt>` | `--output-format json` | `--max-turns N` | `--system-prompt` | Single terminal JSON object |
| `codex` | `codex` | `exec <prompt>` | `--json` (NDJSON stream) | `job_max_runtime_seconds` in `~/.codex/config.toml` (no CLI flag) | No system-prompt CLI flag | Newline-delimited event objects |
| `opencode` | `opencode` | `run <prompt>` | `--format json` | No max-turns flag | No system-prompt CLI flag | One JSON object per line (events) |

---

## Flag Differences

**Max-turns is Claude-only.** Claude is the only CLI that accepts a first-class `--max-turns N` flag. Codex exposes `job_max_runtime_seconds` in `config.toml` (a time-based limit, not a turn count); it can be overridden at invocation via `-c agents.job_max_runtime_seconds=N` but there is no equivalent CLI flag on `codex exec`. OpenCode has no max-turns or runtime-limit mechanism at the CLI level — the agent loop runs until the session reaches an idle state, so process-level timeouts (context cancellation) are the only available guard.

**Codex approval behavior is controlled at startup.** Codex `exec` approval is set by the `--ask-for-approval` flag (values: `untrusted`, `on-failure`, `on-request`, `never`) or by `config.toml`, not by an in-band ask-user tool the agent can call mid-run. For unattended pipelines the Codex adapter defaults to `--ask-for-approval never`.

**OpenCode agent selection affects permission defaults.** OpenCode uses `--agent plan` to engage a read-only agent that asks permission before running bash commands; the default agent is `build`, which runs commands without pausing. Use `--agent build` (or omit the flag) for non-interactive pipelines where pausing is undesirable.

**Streaming output vs. single object.** All three CLIs support streamed output. Only Claude returns a single terminal JSON object when invoked with `--output-format json`. Codex and OpenCode both produce NDJSON streams (one event object per line) that must be consumed by a streaming line reader rather than a one-shot JSON decode.

---

## Selecting a Runner

There are three ways to pick the runner (in priority order):

1. **CLI flag** (highest priority):

   ```bash
   tddmaster run --spec=my-spec --tool=codex
   ```

   Errors with `runner not found` if the tool name is not registered.

2. **Manifest `defaultRunner`** (per-project persistent choice):

   ```yaml
   # .tddmaster/manifest.yml
   defaultRunner: codex
   tools:
     - codex
     - claude-code
   ```

   Unknown `defaultRunner` values silently fall through to `tools[0]`.

3. **First entry in `tools`** (fallback):

   When neither `--tool` nor `defaultRunner` resolves, the first registered tool in `manifest.tools` wins.

4. **Hard fallback**: `claude-code` if nothing above resolves.

---

## AskUserQuestion vs tddmaster block

The two interaction strategies map to the ask-user capability of the underlying CLI.

Claude Code has a native `AskUserQuestion` MCP tool exposed to the agent in `-p` mode. Prompts compiled for Claude include the instruction `Use AskUserQuestion for all decision points.`, which causes the agent to pause and surface a structured question that the orchestrator can route to the operator.

Codex and OpenCode do **not** have an inline ask-user tool. In `codex exec` mode, the `ThreadItemDetails` event schema contains no ask-user or question event type — the only approval mechanism is the startup-time `--ask-for-approval` flag. In `opencode run` mode, there is similarly no structured ask-user JSON event in the `--format json` stream. Prompts compiled for Codex and OpenCode use `tddmaster block "question"` instead, so the orchestrator pauses at the shell level for human input rather than expecting the CLI to break mid-task with a question.

The `AskUserStrategy` enum (`ask_user_question` | `tddmaster_block`) on each adapter's `ToolCapabilities.InteractionHints` drives this routing at compile time; the prompt compiler reads the strategy from the selected adapter's capabilities and emits the correct guidance in the prompt text.

---

## Edge Cases

The following edge cases apply to runner invocation and are handled by the runner abstraction:

- **EC-1 — binary not on PATH**: When `os/exec.LookPath` cannot find the runner binary, the adapter returns `ErrBinaryNotFound`. The orchestrator transitions the spec to a `BLOCKED` state and surfaces an install hint to the operator.

- **EC-2 — CLI flag mismatch**: Each runner constructs its own `argv` slice. Flags are never shared across adapters; the Codex adapter never receives `--output-format`, and the Claude adapter never receives `--json`.

- **EC-3 — non-zero exit**: A non-zero exit code from the CLI process is logged and wrapped as `ErrNonZeroExit`. This is treated as non-fatal at the runner layer; the state file continues to drive progress so a retry is possible.

- **EC-4 — JSON parse failure**: If the output cannot be decoded as valid JSON, the adapter returns `ErrJSONParse` and surfaces the raw text output as a fallback so no output is silently discarded.

- **EC-5 — ask-user routing**: The ask-user strategy (`AskUserQuestion` vs. `tddmaster block`) is enforced via the adapter's `ToolCapabilities.InteractionHints.AskUserStrategy` field. Correct routing is compile-checked by golden snapshot tests.

- **EC-6 — SIGINT/context cancel**: When the parent context is canceled (SIGINT or SIGTERM), the runner sends `os.Interrupt` to the child process, waits up to two seconds for a clean exit, and then calls `Kill` if the process has not exited within that grace period.
