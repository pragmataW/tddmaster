package tddcontract

import (
	"fmt"
	"strings"
)

func VerifierRedPhaseInstruction() string {
	return strings.Join([]string{
		"RED phase (READ-ONLY): DO NOT run tests or type-checkers. DO NOT execute any shell command.",
		"Read each test file written in this iteration using your Read and Grep tools.",
		"Verify: (1) each test asserts behavior tied to a planned task or edge-case; (2) no placeholder/TODO/empty test bodies; (3) syntax is well-formed (imports resolve, function signatures match framework conventions).",
		`Return JSON: {"passed": true|false, "phase": "red", "readOnly": true, "output": "<summary>", "results": [...]}.`,
		"`passed=true` means \"tests are well-formed and cover the planned tasks.\"",
	}, " ")
}

func VerifierGreenPhaseInstruction(typeCheckCmd, testCmd string) string {
	lines := []string{
		"GREEN phase:",
	}
	if typeCheckCmd != "" {
		lines = append(lines, fmt.Sprintf("run type check `%s` on modified files.", typeCheckCmd))
	}
	if testCmd != "" {
		lines = append(lines, fmt.Sprintf("run the full test suite `%s` for the target ACs.", testCmd))
	} else {
		lines = append(lines, "run the full test suite for the target ACs.")
	}
	lines = append(lines,
		"Exit code MUST be zero and all tests MUST pass.",
		"If any test fails, set `passed=false` and report `reason='expected-pass-but-failed'` plus failing test names in `failedACs`. Set `refactorNotes=[]`.",
		"If tests pass, scan the modified files and produce `refactorNotes` — a JSON array of {file, suggestion, rationale} describing concrete improvements.",
		"Apply a language-agnostic quality rubric: repeated magic literals that should become a named single-source symbol, duplication/DRY, single responsibility, naming clarity, oversized functions and parameter lists, coupling and dependency direction, open-closed dispatch hints, cohesion, error and result shaping, dead code and obsolete guards. Phrase each suggestion using the identifiers of the code under review — do not prescribe language-specific constructs.",
		"An empty array is valid and means the code is already clean.",
		`Return JSON: {"passed": true|false, "phase": "green", "output": "<summary>", "failedACs": [...], "refactorNotes": [...]}.`,
	)
	return strings.Join(lines, " ")
}

func VerifierRefactorPhaseInstruction(typeCheckCmd, testCmd string) string {
	lines := []string{
		"REFACTOR phase:",
	}
	if typeCheckCmd != "" {
		lines = append(lines, fmt.Sprintf("run type check `%s`.", typeCheckCmd))
	}
	if testCmd != "" {
		lines = append(lines, fmt.Sprintf("run the full suite `%s`.", testCmd))
	} else {
		lines = append(lines, "run the full suite.")
	}
	lines = append(lines,
		"If red, reason='behavior-changed' (treated like GREEN regression).",
		"If green, produce refactorNotes — a JSON array of {file, suggestion, rationale} describing concrete improvements.",
		"Apply the same language-agnostic quality rubric as GREEN: repeated magic literals needing a named single-source symbol, duplication/DRY, single responsibility, naming clarity, oversized functions and parameter lists, coupling and dependency direction, open-closed dispatch hints, cohesion, error and result shaping, dead code and obsolete guards. Phrase each suggestion using the identifiers of the code under review — do not prescribe language-specific constructs.",
		"An empty array is valid and means the task is clean.",
		`Return JSON: {"passed": true|false, "phase": "refactor", "output": "<summary>", "refactorNotes": [...]}.`,
	)
	return strings.Join(lines, " ")
}

func VerifierReportSchemaJSON() string {
	return `{"passed": true|false, "phase": "red|green|refactor|", "readOnly": true|false, "output": "<summary>", "failedACs": ["<ac-id>"], "uncoveredEdgeCases": ["<EC-id>"], "refactorNotes": [{"file": "<path>", "suggestion": "<what to change>", "rationale": "<why>"}], "results": [{"id": "ac-1", "status": "PASS", "evidence": "..."}]}`
}

func VerifierReportRules() []string {
	return []string{
		"- `phase`: echo the current TDD phase, or an empty string when TDD is disabled.",
		"- `readOnly`: set to true in RED phase (no commands executed). Omit or set false in GREEN/REFACTOR.",
		"- `passed`: in RED phase, `true` means \"tests are well-formed and cover planned tasks\". In GREEN/REFACTOR, `true` means \"all tests pass\".",
		"- `refactorNotes`: required in GREEN and REFACTOR phases whenever `passed: true`. Provide either a non-empty array of `{file, suggestion, rationale}` entries, or `[]` if the code is already clean. Omit the field only when `passed: false`. Omitting it on a GREEN PASS whose `output` mentions refactor/extract/cleanup/rename hints will be rejected by the orchestrator — include `[]` explicitly to confirm \"nothing to do\".",
		"- `results`: legacy per-AC breakdown; keep it filled in for compatibility.",
	}
}
