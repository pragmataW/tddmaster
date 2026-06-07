package promptregistry

const execRedText = "TDD RED phase active. Spawn the `test-writer` sub-agent. " +
	"It writes FAILING tests only — no implementation, no test execution. " +
	"Pass `edgeCases` from this `next` output verbatim. After the test-writer reports, run `tddmaster spec <name> next` again."

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
	"Its structured report is the authoritative result submitted to `next` — not the executor's self-report."

const execGateText = "Important task gate: this task requires a plan before execution. " +
	"Spawn `tddmaster-planner` (read-only) to produce an approved plan with `touchedFiles`, `approach`, `assumptions`, and `designPatterns`. " +
	"Present the plan to the user via AskUserQuestion (accept / revise / reject). " +
	"On accept, submit the plan to `next`. The approved plan is binding — if work requires a file outside `touchedFiles`, stop and report."

const RestartRecommendedText = "Iteration limit reached. Start a new conversation to continue, or submit `next <slug> --answer=\"continue\"` to reset the iteration counter and resume."

const ReportExampleExecutor = `{"completed":["task-1"],"remaining":[],"blocked":[],"filesModified":["internal/foo/bar.go"],"phase":"green","refactorApplied":false}`

const ReportExampleVerifier = `{"passed":true,"phase":"green","failedACs":[],"uncoveredEdgeCases":[],"refactorNotes":[{"file":"internal/foo/bar.go","suggestion":"extract validation","rationale":"reused in two places"}]}`

const ReportExamplePlanner = `{"plan":{"taskId":"","touchedFiles":["internal/foo/bar.go","internal/foo/bar_test.go"],"approach":"Implement X by extending Y with Z pattern","assumptions":["existing tests cover happy path"],"designPatterns":["strategy"],"bestPractices":["single responsibility"]}}`

const ReportExampleTestWriter = `{"testsWritten":["TestFoo_HappyPath","TestFoo_EdgeCase"],"filesModified":["internal/foo/bar_test.go"]}`

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
	_ = execRefactorSkipVerifyText
}
