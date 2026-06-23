package promptregistry

const execRedText = "TDD RED phase active. Spawn the `tddmaster-test-writer` sub-agent. " +
	"It writes FAILING tests only — no implementation, no test execution. " +
	"Pass `edgeCases` from this `next` output verbatim. " +
	"The test-writer MUST include a `traceability` field in its report: one entry per test function, each with `testFilePath`, `functionName`, `taskId`, and the `ac`/`ec` arrays it covers. Reference each acceptance criterion and edge case by its canonical id `AC-<n>` / `EC-<n>`, where <n> is the 1-based position of that item in the task's acceptance-criteria / edge-case list. This field is REQUIRED on RED submit — reports missing `traceability` are invalid. " +
	"After the test-writer reports, run `tddmaster next <slug>` again."

const execGreenText = "TDD GREEN phase active. Spawn the `tddmaster-executor` sub-agent. " +
	"It writes a clean, working implementation that makes the existing failing tests pass. " +
	"It does NOT write new tests and does NOT run tests. " +
	"Submit the executor's status report to `next`. " +
	"Do NOT spawn the verifier yourself — the orchestrator dispatches `tddmaster-verifier` as a separate stage on the next `next` call to run the tests and produce `refactorNotes`."

const execRefactorText = "TDD REFACTOR phase active. " +
	"If `refactorInstructions` is present, spawn `tddmaster-executor` to apply each note verbatim and report `refactorApplied: true`. " +
	"If absent, spawn `tddmaster-verifier` for a regression re-check; tests must still pass."

const execRefactorApplyText = "Apply each refactor note verbatim. Do NOT change test behavior — tests must still pass. When finished, report `refactorApplied: true` in your JSON output; the verifier will re-run tests."

const execRefactorSkipVerifyText = "Apply each refactor note verbatim. Tests must still pass. Submit BOTH `refactorApplied: true` AND `completed: [<task-id>]` in the SAME status report — verifier is disabled in this mode, so this single submit advances the task."

const execVerifyFailedText = "Verification FAILED. Fix the failing tests before continuing."

const execExecutorText = "Executor flow: the tddmaster-executor sub-agent receives the approved plan. " +
	"It implements the minimum clean, working code required to satisfy every acceptance criterion. " +
	"It does NOT run tests, does NOT write new tests, and does NOT self-verify. " +
	"Submit the executor's status report with `completed: [<task-id>]`. " +
	"Do NOT spawn the verifier yourself — the engine dispatches `tddmaster-verifier` as a separate stage on the next `next` call."

const execExecutorSkipVerifyText = "Executor flow (verifier DISABLED): the tddmaster-executor sub-agent receives the approved plan. " +
	"It implements the minimum clean, working code required to satisfy every acceptance criterion. " +
	"It does NOT write new tests. " +
	"Do NOT spawn a verifier — verification is disabled for this spec. " +
	"Submit the executor's status report with `completed: [<task-id>]` in the SAME report; this single submit advances the task."

const execVerifierText = "Independent verification: the tddmaster-verifier sub-agent runs the full test suite and performs an independent scan. " +
	"It produces `passed` (bool), `failedACs` (list), `refactorNotes` (list), and `uncoveredEdgeCases` (list). " +
	"Its structured report is the authoritative result submitted to `next` — not the executor's self-report. " +
	"When coverage measurement is required (green phase only), also include the per-file coverage array described in any appended coverage requirement; this field is omitted in all other phases."

const execGateText = "Important task gate: this task requires a plan before execution. " +
	"Spawn `tddmaster-planner` (read-only) to produce an approved plan with `touchedFiles`, `approach`, `assumptions`, and `designPatterns`. " +
	"FIRST, BEFORE calling any question tool, present the FULL plan to the user as a long plain-text message: the complete `approach` narrative, every `touchedFiles` entry with the reason it is touched, each `designPattern` with how it is applied, each `bestPractice`, and every `assumption`. " +
	"Do NOT put the plan content inside the question tool — the question tool call must contain ONLY the accept / revise / reject choice. A one-line summary instead of the full presentation is a protocol violation. " +
	"ONLY AFTER the full plan text has been presented, call AskUserQuestion with accept / revise / reject. " +
	"On accept, submit the plan to `next`. The approved plan is binding — if work requires a file outside `touchedFiles`, stop and report."

const RestartRecommendedText = "Iteration limit reached. Start a new conversation to continue, or submit `next <slug> --answer=\"continue\"` to reset the iteration counter and resume."

const ReportExampleExecutor = `{"completed":["task-1"],"remaining":[],"blocked":[],"filesModified":["internal/foo/bar.go"],"phase":"green","refactorApplied":false}`

const ReportExampleRefactorApply = `{"completed":["task-1"],"remaining":[],"blocked":[],"filesModified":["internal/foo/bar.go"],"phase":"refactor","refactorApplied":true}`

const ReportExampleRefactorApplySkip = `{"passed":true,"completed":["task-1"],"remaining":[],"blocked":[],"filesModified":["internal/foo/bar.go"],"phase":"refactor","refactorApplied":true}`

const ReportExampleVerifier = `{"passed":true,"phase":"green","failedACs":[],"uncoveredEdgeCases":[],"refactorNotes":[{"file":"internal/foo/bar.go","suggestion":"extract validation","rationale":"reused in two places"}],"fileCoverage":[{"file":"internal/foo/bar.go","coverage":85}]}`

const ReportExamplePlanner = `{"plan":{"taskId":"","touchedFiles":["internal/foo/bar.go","internal/foo/bar_test.go"],"approach":"Implement X by extending Y with Z pattern","assumptions":["existing tests cover happy path"],"designPatterns":["strategy"],"bestPractices":["single responsibility"]}}`

const ReportExampleTestWriter = `{"testsWritten":["TestFoo_HappyPath","TestFoo_EdgeCase"],"filesModified":["internal/foo/bar_test.go"],"traceability":[{"testFilePath":"internal/foo/bar_test.go","functionName":"TestFoo_HappyPath","taskId":"task-1","ac":["AC-1"],"ec":["EC-1"]}]}`

const ReportExampleRuleSynthesizer = `{"rules":[{"scope":"executor","name":"prefer-table-tests","content":"Use table-driven tests for all functions with multiple input cases.","rationale":"Reduces duplication and makes edge cases explicit."}]}`

const ruleLearnProposeText = "Synthesize rules from the accumulated learnings gathered during execution. " +
	"Analyze the refactor note suggestions and failed AC reasons provided. " +
	"For each rule, decide its SCOPE: use 'global' to apply to all agents, or one of 'executor', 'test-writer', 'verifier', 'planner' for a specific agent. " +
	"Return ONLY a JSON proposal without writing any files: {\"rules\":[{\"scope\":\"<scope>\",\"name\":\"<name>\",\"content\":\"<rule text>\",\"rationale\":\"<why>\"}]}."

const ruleLearnApplyText = "Apply the approved rules by running `tddmaster rule add` for each rule. " +
	"For each rule: write its content to a temporary file, then run `tddmaster rule add --scope <scope> --name <name> --content-file <path>`. " +
	"Never edit rule files directly. Never overwrite existing rules. Surface any errors immediately."

func init() {
	instructionMap[KeyExecRed] = execRedText
	instructionMap[KeyExecGreen] = execGreenText
	instructionMap[KeyExecRefactor] = execRefactorText
	instructionMap[KeyExecRefactorApply] = execRefactorApplyText
	instructionMap[KeyExecExecutor] = execExecutorText
	instructionMap[KeyExecExecutorSkipVerify] = execExecutorSkipVerifyText
	instructionMap[KeyExecVerifier] = execVerifierText
	instructionMap[KeyExecGate] = execGateText
	instructionMap[KeyExecVerifyFailed] = execVerifyFailedText
	instructionMap[KeyExecRefactorSkipVerify] = execRefactorSkipVerifyText
	instructionMap[KeyRuleLearnPropose] = ruleLearnProposeText
	instructionMap[KeyRuleLearnApply] = ruleLearnApplyText
}
