package shared

import (
	"strings"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/context/service/tdd"
)

// VerifierInstructions returns the generic (phase-agnostic) verifier body.
//
// Use this for non-TDD projects (manifest.TddMode == false).
// It runs type-check + tests and produces a pass/fail report — no RED/GREEN/REFACTOR blocks.
// IMPORTANT: Do NOT call VerifierInstructionsAllPhases for non-TDD projects; it would
// inject TDD phase instructions that the verifier cannot meaningfully apply.
func VerifierInstructions(typeCheckCmd, testCmd string) string {
	return strings.Join(genericVerifierLines(typeCheckCmd, testCmd), "\n")
}

// VerifierInstructionsAllPhases returns the full TDD verifier prompt: the
// generic body plus the three phase-specific blocks (RED, GREEN, REFACTOR).
//
// Use this ONLY for TDD projects (manifest.TddMode == true).
// The adapter templates are static; the orchestrator injects `tddPhase`
// into the sub-agent task context at runtime. The verifier follows whichever
// phase block matches its current context.
//
// Selection rule in adapters:
//
//	if manifest != nil && manifest.TddMode {
//	    VerifierInstructionsAllPhases(...)
//	} else {
//	    VerifierInstructions(...)   // no TDD blocks for non-TDD projects
//	}
func VerifierInstructionsAllPhases(typeCheckCmd, testCmd string, skipVerify bool) string {
	lines := genericVerifierLines(typeCheckCmd, testCmd)
	lines = append(lines,
		"",
		"## TDD Phase Behavior",
		"If your task context includes `tddPhase`, follow the matching phase block below.",
		"Otherwise ignore them and use the generic steps above.",
		"",
		verifierRedPhaseBlockWithSkip(skipVerify),
		"",
		verifierGreenPhaseBlockWithSkip(typeCheckCmd, testCmd, skipVerify),
		"",
		verifierRefactorPhaseBlockWithSkip(typeCheckCmd, testCmd, skipVerify),
	)
	lines = append(lines, "", verifierReportBlock())
	return strings.Join(lines, "\n")
}

// VerifierRedPhaseBlock returns the standalone RED-phase instructions (non-skipVerify).
// Exposed so tests and specialized adapters can reuse the block verbatim.
// RED phase is read-only: the verifier inspects test files without running them.
func VerifierRedPhaseBlock() string {
	return verifierRedPhaseBlockWithSkip(false)
}

func verifierRedPhaseBlockWithSkip(skipVerify bool) string {
	var executionLine, qualityLine string
	if skipVerify {
		executionLine = "- Test execution is optional in skip-verify mode. If needed, you may inspect test files."
		qualityLine = "- Do NOT comment on implementation quality."
	} else {
		executionLine = "- DO NOT run tests. DO NOT invoke type-checkers. DO NOT execute any shell command."
		qualityLine = "- Do NOT comment on implementation quality. Do NOT run typeCheckCmd or testCmd."
	}
	lines := []string{
		"### TDD RED Phase (READ-ONLY)",
		"You are verifying that the test-writer produced well-formed tests BEFORE any implementation exists.",
		executionLine,
		"- READ each test file written in this iteration using your Read and Grep tools.",
		"- VERIFY: (1) each test asserts behavior tied to a planned task or edge-case; (2) no placeholder/TODO/empty test bodies; (3) syntax is well-formed (imports resolve, function signatures match framework conventions).",
		"- Set `passed: true` when all test files are well-formed and cover the planned tasks.",
		"- Set `passed: false` with `reason: \"tests-malformed\"` and describe what is missing or incorrect.",
		"- " + tdd.VerifierRedPhaseInstruction(),
		qualityLine,
	}
	return strings.Join(lines, "\n")
}

// VerifierGreenPhaseBlock returns the standalone GREEN-phase instructions (non-skipVerify).
// In the merged flow this phase also acts as the refactor scanner: if tests pass
// the verifier immediately produces refactorNotes so a separate REFACTOR verifier
// call is no longer required.
func VerifierGreenPhaseBlock(typeCheckCmd, testCmd string) string {
	return verifierGreenPhaseBlockWithSkip(typeCheckCmd, testCmd, false)
}

func verifierGreenPhaseBlockWithSkip(typeCheckCmd, testCmd string, skipVerify bool) string {
	lines := []string{
		"### TDD GREEN Phase",
		"You are verifying that the executor's implementation makes the target tests pass.",
		"- Run type check: `" + typeCheckCmd + "` on modified files.",
		"- Run the full test suite: `" + testCmd + "` for the target ACs. Exit code MUST be zero.",
		"- If any test fails, set `passed: false`, report `reason: \"expected-pass-but-failed\"` plus failing test names in `failedACs`, and set `refactorNotes: []`.",
		"- If all tests pass, set `passed: true` and scan the modified files against the Quality rubric below.",
		"  Produce `refactorNotes` — a JSON array of `{file, suggestion, rationale}` describing every concrete improvement you can spot.",
		"  An empty array is valid and means the code is already clean — the orchestrator will skip refactor.",
		"  `refactorNotes` is REQUIRED in every GREEN PASS report — provide `[{...}]` with concrete entries, or `[]` to assert the code is clean. Never omit the field on a pass.",
		"  Contract: " + tdd.VerifierGreenPhaseInstruction(typeCheckCmd, testCmd),
		"  Do NOT suggest changes that alter test behavior or public API.",
	}
	if skipVerify {
		lines = append(lines, "  "+model.GreenRefactorNotesMandatory)
	}
	lines = append(lines, refactorQualityRubric()...)
	return strings.Join(lines, "\n")
}

// VerifierRefactorPhaseBlock returns the standalone REFACTOR-phase instructions (non-skipVerify).
// In the merged flow this phase is a regression-check after the executor applied
// the notes produced during the GREEN scan. It may also emit new notes for another round.
func VerifierRefactorPhaseBlock(typeCheckCmd, testCmd string) string {
	return verifierRefactorPhaseBlockWithSkip(typeCheckCmd, testCmd, false)
}

func verifierRefactorPhaseBlockWithSkip(typeCheckCmd, testCmd string, skipVerify bool) string {
	var testRunLine string
	if skipVerify {
		testRunLine = "- Running the full test suite is optional in skip-verify mode; if needed run `" + testCmd + "`. If red, set `passed: false` and report `reason: \"behavior-changed\"` with failing test names."
	} else {
		testRunLine = "- Run the full test suite: `" + testCmd + "`. If red, set `passed: false` and report `reason: \"behavior-changed\"` with failing test names."
	}
	lines := []string{
		"### TDD REFACTOR Phase",
		"You are verifying that the executor's refactor changes did not break behavior.",
		"- Run type check: `" + typeCheckCmd + "`.",
		testRunLine,
		"- If green, scan the modified files against the Quality rubric below and produce `refactorNotes` — a JSON array of `{file, suggestion, rationale}`.",
		"- An empty `refactorNotes` array means the task is clean; the orchestrator advances to the next task.",
		"- Contract: " + tdd.VerifierRefactorPhaseInstruction(typeCheckCmd, testCmd),
		"- Do NOT suggest changes that alter test behavior or public API.",
	}
	lines = append(lines, refactorQualityRubric()...)
	return strings.Join(lines, "\n")
}

// refactorQualityRubric returns the language-agnostic quality checklist that
// GREEN and REFACTOR phases append to their refactorNotes scan instructions.
// The list is intentionally written in framework-neutral terms ("named
// constant", "module-level symbol") so the same prompt applies to Go,
// TypeScript, Python, or any other target. Do NOT introduce language-specific
// keywords here (e.g. const, enum, final, var) — those would bias non-matching
// languages and drift against the "same standard everywhere" goal.
func refactorQualityRubric() []string {
	return []string{
		"",
		"#### Quality rubric (language-agnostic)",
		"Scan the diff against every item below. For each hit, emit a `refactorNotes` entry whose `suggestion` speaks in the vocabulary of the code under review (reuse its identifiers and filenames) rather than prescribing a specific language construct. The executor is expected to ship a clean, working implementation — your job is to surface ONLY concrete improvements that are clearly worth a refactor round. If the code is already good, return an empty array; do NOT invent issues to justify a refactor. Equally, do not be overly permissive when a real improvement exists.",
		"- Magic values: any literal string, number, or boolean that carries meaning and appears more than once, or that a future change would have to update in multiple places. Suggest extracting a named, single-source-of-truth symbol.",
		"- Duplication (DRY): near-identical blocks or branches that could be parameterised into a shared helper.",
		"- Single responsibility: a function, method, or type that mixes unrelated concerns — split along the seams.",
		"- Naming clarity: vague, abbreviated, single-letter, or misleading identifiers; suggest names that describe intent.",
		"- Function and parameter size: overly long bodies or long parameter lists; suggest decomposition or grouping into a structured argument.",
		"- Coupling and dependency direction: concrete dependencies hard-wired where an interface or injection point would isolate change.",
		"- Open–closed hints: repeated type-switch or long if/else chains keyed on a discriminator; suggest polymorphism or a dispatch table.",
		"- Cohesion: types that bundle fields serving unrelated lifecycles or callers; suggest splitting.",
		"- Error and result shaping: silently swallowed errors, ignored return values, opaque fallbacks that hide failure modes.",
		"- Dead code and obsolete guards: branches, flags, or helpers introduced only to satisfy tests that no production path exercises.",
		"Never suggest changes that alter observable behavior, public API, or existing test expectations — those belong to a new RED cycle, not REFACTOR.",
	}
}

func genericVerifierLines(typeCheckCmd, testCmd string) []string {
	return []string{
		"You are verifying another agent's work. You have NO context about how it was done.",
		"Read the changed files. Check each acceptance criterion independently.",
		"",
		"For each acceptance criterion:",
		"- PASS: with evidence — show the grep result, the test output, or the file content that proves it.",
		"- FAIL: with specific reason — what's missing, what's wrong, what doesn't match.",
		"",
		"Be skeptical. Don't assume anything works — verify it yourself.",
		"You CANNOT edit files. Read-only access only.",
		"",
		"## Verification Steps",
		"1. Read each modified file and verify the changes are correct.",
		"2. Check each acceptance criterion against actual file contents.",
		"3. When TDD phase context is provided (see phase blocks below), follow the matching phase block.",
		"4. When TDD is disabled, run type check: `" + typeCheckCmd + "` and tests: `" + testCmd + "`.",
	}
}

func verifierReportBlock() string {
	lines := []string{
		"## Report Format",
		"Return a structured JSON summary:",
		"```json",
		tdd.VerifierReportSchemaJSON(),
		"```",
		"",
		"Rules for the JSON fields:",
	}
	lines = append(lines, tdd.VerifierReportRules()...)
	return strings.Join(lines, "\n")
}
