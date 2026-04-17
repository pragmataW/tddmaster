# Changelog

All notable changes to `tddmaster` are documented in this file. Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- Multi-runner abstraction: new `internal/runner` package with `Runner` interface, `RunRequest`/`RunResult` types, and sentinel errors (`ErrBinaryNotFound`, `ErrInvalidArgs`, `ErrNonZeroExit`, `ErrJSONParse`, `ErrContextCanceled`, `ErrDuplicateRunner`, `ErrRunnerNotFound`).
- `ClaudeRunner`, `CodexRunner`, and `OpenCodeRunner` implementations registered at package init.
- Runner registry with priority-chain `Select(manifest, toolFlag)`: `--tool` flag > `manifest.defaultRunner` > `manifest.tools[0]` > `claude-code` fallback.
- `tddmaster run --tool=<claude-code|codex|opencode>` CLI flag.
- `NosManifest.DefaultRunner` field (optional, `omitempty`).
- `InteractionHints.AskUserStrategy` enum (`ask_user_question` | `tddmaster_block`) on every `ToolAdapter.Capabilities()`.
- Prompt compiler wiring: adapters with `AskUserStrategy == "tddmaster_block"` receive `tddmaster block "..."` guidance instead of `AskUserQuestion` references.
- Golden snapshot tests for the ask-user-strategy behavioral rules.
- OpenCode detection signal in `internal/detect/tools.go` (`.opencode` directory, `opencode.json` file).
- `docs/RUNNERS.md` documenting the invocation matrix and per-runner flag differences.

### Changed
- `internal/bridge/bridge.go` delegates to the runner abstraction instead of hard-coding the `claude` binary. Public API (`CallAgent(prompt, system)`) is preserved.
- `cmd/run.go` Ralph loop now invokes the selected runner via `runner.Invoke(ctx, RunRequest{...})`; SIGINT/SIGTERM trigger a cancelable context and graceful shutdown.
- `CreateInitialManifest` signature reduced from 4 to 3 parameters — the `providers []ToolId` argument is removed.

### Removed
- `NosManifest.Providers` field (legacy API-key provider list — never consumed).
- `internal/detect/providers.go` (dead `DetectProviders` / `GetAvailableProviderNames` scanning of `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` / `GOOGLE_API_KEY` env vars).
- `state.ToolId` type alias (only used by the removed `Providers` slice).

### Migration
- Existing manifests containing `providers: [...]` still parse cleanly — the field is silently ignored by Go's JSON/YAML unmarshal (additive schema change).
- Existing Claude-only users need no action: with `manifest.tools: ["claude-code"]`, flag-less `tddmaster run` resolves to the Claude runner unchanged.
- To switch CLIs, either add `--tool=codex` to `tddmaster run` or set `defaultRunner: codex` in the manifest.
