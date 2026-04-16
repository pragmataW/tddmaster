package shared

import "strings"

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
func VerifierInstructionsAllPhases(typeCheckCmd, testCmd string) string {
	lines := genericVerifierLines(typeCheckCmd, testCmd)
	lines = append(lines,
		"",
		"## TDD Phase Behavior",
		"If your task context includes `tddPhase`, follow the matching phase block below.",
		"Otherwise ignore them and use the generic steps above.",
		"",
		VerifierRedPhaseBlock(),
		"",
		VerifierGreenPhaseBlock(typeCheckCmd, testCmd),
		"",
		VerifierRefactorPhaseBlock(typeCheckCmd, testCmd),
	)
	lines = append(lines, "", verifierReportBlock())
	return strings.Join(lines, "\n")
}

// VerifierRedPhaseBlock returns the standalone RED-phase instructions.
// Exposed so tests and specialized adapters can reuse the block verbatim.
// RED phase is read-only: the verifier inspects test files without running them.
func VerifierRedPhaseBlock() string {
	return strings.Join([]string{
		"### TDD RED Phase (READ-ONLY)",
		"You are verifying that the test-writer produced well-formed tests BEFORE any implementation exists.",
		"- DO NOT run tests. DO NOT invoke type-checkers. DO NOT execute any shell command.",
		"- READ each test file written in this iteration using your Read and Grep tools.",
		"- VERIFY: (1) each test asserts behavior tied to a planned task or edge-case; (2) no placeholder/TODO/empty test bodies; (3) syntax is well-formed (imports resolve, function signatures match framework conventions).",
		"- Set `passed: true` when all test files are well-formed and cover the planned tasks.",
		"- Set `passed: false` with `reason: \"tests-malformed\"` and describe what is missing or incorrect.",
		`- Report {"passed": true|false, "phase": "red", "readOnly": true, "output": "<summary>", "results": [...]}.`,
		"- Do NOT comment on implementation quality. Do NOT run typeCheckCmd or testCmd.",
	}, "\n")
}

// VerifierGreenPhaseBlock returns the standalone GREEN-phase instructions.
// In the merged flow this phase also acts as the refactor scanner: if tests pass
// the verifier immediately produces refactorNotes so a separate REFACTOR verifier
// call is no longer required.
func VerifierGreenPhaseBlock(typeCheckCmd, testCmd string) string {
	return strings.Join([]string{
		"### TDD GREEN Phase",
		"You are verifying that the executor's minimum implementation makes the target tests pass.",
		"- Run type check: `" + typeCheckCmd + "` on modified files.",
		"- Run the full test suite: `" + testCmd + "` for the target ACs. Exit code MUST be zero.",
		"- If any test fails, set `passed: false`, report `reason: \"expected-pass-but-failed\"` plus failing test names in `failedACs`, and set `refactorNotes: []`.",
		"- If all tests pass, set `passed: true` and scan the modified files for concrete improvements.",
		"  Produce `refactorNotes` — a JSON array of `{file, suggestion, rationale}` (dead code, duplication, naming, structure).",
		"  An empty array is valid and means the code is already clean — the orchestrator will skip refactor.",
		"  Do NOT suggest changes that alter test behavior or public API.",
	}, "\n")
}

// VerifierRefactorPhaseBlock returns the standalone REFACTOR-phase instructions.
// In the merged flow this phase is a regression-check after the executor applied
// the notes produced during the GREEN scan. It may also emit new notes for another round.
func VerifierRefactorPhaseBlock(typeCheckCmd, testCmd string) string {
	return strings.Join([]string{
		"### TDD REFACTOR Phase",
		"You are verifying that the executor's refactor changes did not break behavior.",
		"- Run type check: `" + typeCheckCmd + "`.",
		"- Run the full test suite: `" + testCmd + "`. If red, set `passed: false` and report `reason: \"behavior-changed\"` with failing test names.",
		"- If green, scan the modified files for any remaining improvements and produce `refactorNotes` — a JSON array of `{file, suggestion, rationale}`.",
		"- An empty `refactorNotes` array means the task is clean; the orchestrator advances to the next task.",
		"- Do NOT suggest changes that alter test behavior or public API.",
	}, "\n")
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
	return strings.Join([]string{
		"## Report Format",
		"Return a structured JSON summary:",
		"```json",
		`{"passed": true|false, "phase": "red|green|refactor|", "readOnly": true|false, "output": "<summary>", "failedACs": ["<ac-id>"], "uncoveredEdgeCases": ["<EC-id>"], "refactorNotes": [{"file": "<path>", "suggestion": "<what to change>", "rationale": "<why>"}], "results": [{"id": "ac-1", "status": "PASS", "evidence": "..."}]}`,
		"```",
		"",
		"Rules for the JSON fields:",
		"- `phase`: echo the current TDD phase, or an empty string when TDD is disabled.",
		"- `readOnly`: set to true in RED phase (no commands executed). Omit or set false in GREEN/REFACTOR.",
		"- `passed`: in RED phase, `true` means \"tests are well-formed and cover planned tasks\". In GREEN/REFACTOR, `true` means \"all tests pass\".",
		"- `refactorNotes`: only populated in REFACTOR phase. An empty array means \"no improvements found; task is clean.\"",
		"- `results`: legacy per-AC breakdown; keep it filled in for compatibility.",
	}, "\n")
}
