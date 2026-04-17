# Bug Review: Spec Creation And Execution Flow

This document captures the highest-signal bugs found while reviewing the spec creation and execution paths in `tddmaster`, with special attention to TDD behavior, non-TDD behavior, and prompt/state-machine consistency.

The goal here is not to catalog every rough edge. It is to document the bugs that plausibly explain why the system feels unstable in real use.

## Current Status

This review is now partly historical.

As of `2026-04-18`, the repo no longer matches several of the failures documented below:

- `run` now honors the `SPEC_APPROVED` gate and the per-task TDD selection gate before execution starts
- discovery prompt text now uses only real mode names
- discovery prompt guidance now teaches a sequential one-question flow instead of advertising batch submission as the primary contract
- listen-first `UserContext` now feeds a persisted discovery prefill pipeline
- GREEN and REFACTOR verifier docs now both describe the real `refactorNotes` contract
- the full test suite is green again

The remaining prompt-level follow-up is mainly about prompt quality and contract deduplication:

- keep the execution prompt phase payload first, with the expected JSON report shape made explicit
- keep verifier contract text sourced from a shared helper so prompt/runtime drift does not reappear
- keep this document aligned with runtime truth so historical findings are not mistaken for open regressions

## Review Scope

- Spec creation and discovery flow
- Spec proposal and spec approval transitions
- Execution loop
- TDD phase handling
- Prompt/compiler output consistency
- Verifier and executor report contracts

## Current Test Status

The current test suite is green.

Reproduced with:

```bash
rtk go test ./...
```

Current result:

- `go test ./...` passes
- latest local run: `962` tests across `19` packages

The edge-case drift described later in this document should now be treated as a historical finding unless it reappears in future changes.

## Critical Findings

### 1. `run` bypasses the `SPEC_APPROVED` gate and TDD task selection

Confidence: 10/10
Basis: read code in `cmd/run.go`, `cmd/next.go`, `internal/context/compiler.go`, `internal/state/persistence.go`

Expected behavior:

- when a spec is in `SPEC_APPROVED`, the system should go through the same gate as `next --answer=...`
- if spec-level TDD is enabled and `TaskTDDSelected` is still unset, the user must select:
  - `tdd-all`
  - `tdd-none`
  - custom `{"tddTasks":[...]}`
- only after that should execution begin

Actual behavior:

- `cmd/run.go` accepts `SPEC_APPROVED`
- it immediately calls `state.StartExecution(initialState)`
- it writes only the global state file with `state.WriteState(root, newState)`
- it does not call the `handleSpecApprovedAnswer(...)` path
- it does not populate `TaskTDDSelected`
- it does not apply per-task TDD overrides
- it does not seed the TDD cycle based on the chosen task configuration

Relevant code:

- `cmd/run.go:72`
- `cmd/run.go:76`
- `cmd/next.go:1093`
- `cmd/next.go:1100`
- `cmd/next.go:1111`
- `cmd/next.go:1118`
- `cmd/next.go:1124`
- `internal/context/compiler.go:2178`

Why this is bad:

- `run` and `next` implement different state machines for the same phase
- users can start execution through `run` without ever making the TDD scope decision that the rest of the system requires
- the global state can become `EXECUTING` while the per-spec state is still `SPEC_APPROVED`

Why the state desync matters:

- `ResolveState(root, &specName)` prefers the per-spec state file
- if the per-spec state was not updated, later `spec <name> ...` commands can resolve to stale phase data
- this creates the exact class of "it feels random / it just broke" failures that are hard to reason about

Relevant code:

- `internal/state/persistence.go:125`
- `internal/state/persistence.go:136`
- `internal/state/persistence.go:146`

Recommended fix:

- remove the custom `SPEC_APPROVED -> EXECUTING` jump from `run`
- route `run` through the same logic as `next --answer`
- or explicitly refuse to run from `SPEC_APPROVED` until `TaskTDDSelected` is resolved
- always write both main state and per-spec state when phase changes

### 2. The execution loop gives the agent almost none of the real execution context

Confidence: 10/10
Basis: read code in `cmd/run.go` and `internal/context/compiler.go`

Expected behavior:

- the execution loop should hand the agent the current task
- the task's files and edge cases should be included
- if TDD is active, the prompt should include:
  - `tddPhase`
  - `tddVerificationContext`
  - `refactorInstructions`
  - failure context when verification failed

Actual behavior:

- `cmd/run.go` calls `ctxpkg.Compile(...)` with `parsedSpec=nil`
- `compileExecution(...)` only knows the current task when `parsedSpec` is present
- without parsed spec, it cannot compute `nextTask`
- then `buildAgentPrompt(...)` throws away most of the compiled output anyway
- the final prompt includes only:
  - `resumeHint`
  - generic rules
  - generic "report progress" command

Relevant code:

- `cmd/run.go:203`
- `internal/context/compiler.go:2466`
- `internal/context/compiler.go:2476`
- `internal/context/compiler.go:2610`
- `cmd/run.go:257`
- `cmd/run.go:260`
- `cmd/run.go:276`

Why this is bad:

- the agent is not operating on the same contract the compiler generates
- the compiler may know the current task, edge cases, TDD phase, and refactor instructions, but the runner never sees them
- in the worst case, execution falls back to the generic message:
  - `All tasks completed. Run 'done' to finish.`
- that is especially dangerous when tasks do exist but were simply not parsed into the runtime prompt

User-facing impact:

- blind execution
- incorrect self-completion
- failure to honor edge cases
- TDD loop drift
- verifier/executor handoff errors

Recommended fix:

- parse the spec before calling `Compile(...)` inside `run`
- make `buildAgentPrompt(...)` include the phase payload, not only `Meta`
- treat `ExecutionData` as the primary execution contract

### 3. The "listen first" user context is collected and then effectively discarded

Confidence: 10/10
Basis: read code in `cmd/next.go`, `internal/state/machine.go`, `internal/context/compiler.go`

Expected behavior:

- first user context should become meaningful discovery input
- long descriptions should pre-fill discovery answers as promised
- spec generation should be better after a rich first message

Actual behavior:

- first answer in discovery is stored via `state.SetUserContext(...)`
- after that, the system uses `hasUserContext` only as a boolean gate
- there is no code that transforms `UserContext` into discovery answers
- `MarkUserContextProcessed(...)` exists but is not wired into the real flow

Relevant code:

- `cmd/next.go:396`
- `cmd/next.go:400`
- `cmd/next.go:402`
- `internal/state/machine.go:561`
- `internal/state/machine.go:569`
- `internal/context/compiler.go:1539`

Why this is bad:

- the system promises a richer discovery flow than it actually performs
- users who provide detailed context early get no actual benefit
- this creates a perception that the tool "heard" the context but ignored it

Prompt/runtime contradiction:

- behavioral rules promise:
  - rich context `>200 chars` -> pre-fill discovery answers as `STATED` / `INFERRED`
- actual runtime does not do that transformation

Relevant prompt code:

- `internal/context/compiler.go:614`
- `internal/context/compiler.go:1816`
- `internal/context/compiler.go:1941`

Recommended fix:

- make `UserContext` feed a real prefill pipeline
- mark extracted items as `STATED` vs `INFERRED`
- persist the resulting suggestions into a reviewable discovery structure
- delete the dead promise if this feature is not going to exist

### 4. Discovery instructions contradict the actual state machine

Confidence: 9/10
Basis: read code in `internal/context/compiler.go` and protocol text in `internal/sync/adapters/shared/agents_md.go`

There are several mismatches here.

#### 4.1 One-question protocol vs batch-submit discovery

Protocol says:

- one `tddmaster` call per interaction
- ask one question
- wait
- do not batch answers

Relevant text:

- `internal/sync/adapters/shared/agents_md.go:72`
- `internal/sync/adapters/shared/agents_md.go:76`

But human discovery output says:

- ask questions one at a time
- then submit all answers together as a JSON object

Relevant code:

- `internal/context/compiler.go:1872`
- `internal/context/compiler.go:1877`

Why this is bad:

- one layer teaches the agent not to batch
- another layer instructs it to batch
- this makes agent behavior nondeterministic depending on which instruction it latches onto

#### 4.2 Behavioral rules advertise modes that do not exist

Behavioral rules say:

- after spec new, ask:
  - full discovery
  - quick discovery
  - skip to spec draft

Relevant code:

- `internal/context/compiler.go:636`

But the actual discovery mode parser accepts:

- `full`
- `validate`
- `technical-depth`
- `ship-fast`
- `explore`

Relevant code:

- `cmd/next.go:408`
- `internal/context/compiler.go:1573`

Why this is bad:

- the prompt suggests unsupported answers
- "quick discovery" is not a real mode
- "skip to spec draft" is not implemented as a real branch in this flow

#### 4.3 Premise challenge is mandatory even when prompts imply skipping

Once a mode is selected, the flow immediately requires premise challenge JSON.

Relevant code:

- `internal/context/compiler.go:1621`
- `internal/context/compiler.go:1639`
- `cmd/next.go:420`

This means the UX promise of "skip" is not a real skip. It is a narrative convenience with no corresponding state transition.

Recommended fix:

- choose one contract and delete the other
- either enforce one-question discovery everywhere or explicitly redesign discovery as a batch phase
- remove unsupported mode language from prompts
- do not advertise "skip to spec draft" unless there is a real state path for it

### 5. Verifier prompt schema contradicts verifier runtime expectations

Confidence: 10/10
Basis: read code in `internal/sync/adapters/shared/verifier_prompt.go` and `cmd/next.go`

Expected behavior:

- the prompt should describe exactly the JSON that backend code expects

Actual behavior:

- GREEN-phase instructions say:
  - if tests pass, produce `refactorNotes`
- report-format section later says:
  - `refactorNotes` is only populated in REFACTOR phase
- backend logic in `applyVerifierReport(...)` expects GREEN-pass refactor notes and actively guards against them being omitted when output prose suggests refactor work

Relevant code:

- `internal/sync/adapters/shared/verifier_prompt.go:71`
- `internal/sync/adapters/shared/verifier_prompt.go:77`
- `internal/sync/adapters/shared/verifier_prompt.go:130`
- `cmd/next.go:1404`
- `cmd/next.go:1408`

Why this is bad:

- the prompt tells the verifier to do one thing
- the report schema tells it something else
- backend code assumes the GREEN-phase note path is real

Likely symptom in practice:

- verifier returns narrative refactor advice in `output`
- structured `refactorNotes` field is missing
- backend rejects the report and asks for explicit notes
- user experiences this as flaky or repetitive TDD behavior

Recommended fix:

- the report block must say `refactorNotes` is valid in both GREEN and REFACTOR
- GREEN pass should explicitly require either:
  - `refactorNotes: [...]`
  - or `refactorNotes: []`
- backend and prompt must share one contract

### 6. Edge-case contract drift is currently breaking the build

Confidence: 10/10
Basis: read code and reproduced test failure

Current implementation of `DeriveEdgeCases(...)` explicitly limits extraction to:

- literal `edge_cases` answer
- disagreed or revised premises

Relevant code:

- `internal/spec/template.go:280`

But current tests still expect verification-answer bullets to appear as edge cases:

- `- Cover timeout recovery`
- `- Happy path smoke test`

Relevant tests:

- `internal/context/compiler_tdd_test.go:143`
- `internal/context/compiler_tdd_test.go:151`
- `internal/context/compiler_tdd_test.go:165`
- `internal/context/compiler_tdd_test.go:180`
- `internal/context/compiler_tdd_test.go:494`
- `internal/context/compiler_tdd_test.go:502`

Why this matters:

- the repo currently has no single source of truth for what counts as an edge case
- compiler output, spec derivation, and tests are drifting apart
- users will see inconsistent edge-case behavior between spec proposal and execution

Recommended fix:

- decide which contract is correct:
  - narrow edge-case derivation
  - or verification-answer-derived edge cases
- then update code and tests together
- do not leave the project in the current split-brain state

## TDD Flow Inconsistencies

These are not all independent bugs, but together they explain the "deli gibi buglu" feeling in TDD mode.

### TDD selection exists in one path but not the other

- `next` enforces TDD selection at `SPEC_APPROVED`
- `run` skips it

Result:

- task-level TDD selection can exist on paper but be absent in live execution

### TDD runtime state may be correct in compiler output but missing from runner prompt

- compiler can build `tddPhase`
- runner prompt does not include it

Result:

- the loop appears to support TDD but the actual worker process may never see the phase instructions

### Refactor note contract is split across prompt and backend

- backend uses GREEN pass as refactor-note source
- prompt report schema suggests otherwise

Result:

- REFACTOR transitions become fragile

## Non-TDD Flow Inconsistencies

### Non-TDD execution still depends on execution payloads that `run` does not surface

Even outside TDD, the execution loop needs:

- current task
- edge cases
- acceptance/debt context
- blocked/completed report shape

But `run` currently builds a very thin prompt. This is not only a TDD bug. It affects normal execution too.

### Global state vs per-spec state can diverge

This is not TDD-specific.

Any path that writes only `state.json` but not the per-spec state risks making:

- `tddmaster next --spec=<name>`
- `tddmaster status`
- `tddmaster spec <name> ...`

behave as if they are looking at different realities.

## Prompt Fix Candidates

These are the highest-value prompt-level changes to make after code-level fixes.

### 1. Execution prompt should be phase payload first, not `resumeHint` first

Current `run` prompt is too generic.

It should include, in this order:

1. current phase
2. current task id and title
3. files touched or suggested files
4. edge cases
5. TDD phase when present
6. verifier/refactor instructions when present
7. exact report JSON shape expected back

### 2. Discovery prompt should stop advertising fake modes

Remove:

- `quick discovery`
- `skip to spec draft`

unless both are implemented as real accepted answers and real transitions.

### 3. Discovery contract should pick one model

Either:

- one question per interaction

or:

- structured batch confirmation

But not both. Right now the system teaches the agent both behaviors.

### 4. Verifier report block should match runtime truth

The report docs should say:

- GREEN pass must return explicit `refactorNotes`
- REFACTOR pass may also return `refactorNotes`
- empty array is valid in both when no improvements remain

## Suggested Fix Order

Updated status:

1. Done: fix `cmd/run.go` so `run` does not bypass `SPEC_APPROVED` / task-TDD selection gating.
2. Done: resolve the edge-case contract drift and get the suite back to green.
3. Done: wire listen-first `UserContext` into real discovery prefills.
4. Done: clean up discovery prompt contradictions and unsupported mode language.
5. Done: align verifier report docs with GREEN/REFACTOR runtime truth.
6. Remaining hardening: keep `run` prompt construction phase-payload first and keep the verifier contract sourced from shared helpers.

## Summary

The main problem is not one isolated bug. It is contract drift between:

- state machine transitions
- compiler output
- runner prompt construction
- verifier JSON schema
- tests

The most damaging failures are:

- `run` not following the same execution gate as `next` when prompt/state changes drift apart
- agents not receiving the real execution contract when the execution prompt gets too generic
- TDD prompt/runtime mismatch around `refactorNotes` if verifier docs and backend expectations drift again
- discovery promising flows that do not exist when prompt text advertises unsupported paths

That combination is enough to make both TDD and non-TDD execution feel unreliable even when individual lower-level helpers are correct.
