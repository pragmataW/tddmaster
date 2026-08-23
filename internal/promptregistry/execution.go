package promptregistry

const execRedText = "You are in the TDD RED phase. Write FAILING tests ONLY — no implementation code, and do not run the tests. " +
	"Cover every criterion in the " + NameCriteria + " section and every entry in the " + NameEdgeCases + " section. " +
	"Your report MUST include a `traceability` array: one entry per test function, each with `testFilePath`, `functionName`, `taskId`, and the `ac`/`ec` arrays it covers. " +
	"Reference every criterion by its `ac-N` id and every edge case by its `ec-N` id exactly as printed in those sections — lowercase, never `AC-N` or `EC-N`. " +
	"When a " + NameTraceability + " section is present, those tests already exist: do not rewrite them, add only what is missing. " +
	"When a " + NameLastFailure + " section is present, the verifier found ids with no test — cover every one of them this round. " +
	"`traceability` and `testsWritten` are both REQUIRED — a report missing either is rejected. " +
	"List every file you wrote in `filesModified`, which is also required; the GREEN executor is given that list as the tests it must make pass." +
	FilesModifiedPathNote

const execGreenText = "You are in the TDD GREEN phase. Write a clean, working implementation that makes the existing failing tests pass. " +
	"The tests you must satisfy are the ones listed in the " + NameChangedFiles + " section — read them before writing anything; they define the contract. " +
	"Do NOT write new tests and do NOT run tests — an independent verifier runs them after you. " +
	"Do not artificially minimise the solution; ship the clean version. " +
	"List every path you wrote in `filesModified` — the verifier starts from that list, " +
	"and report `completed: [<taskId>]` (the taskId from the " + NameTask + " section, not criterion ids) once the tests are satisfied." +
	FilesModifiedPathNote

const execRefactorText = "You are re-checking a TDD REFACTOR round for regressions. The refactor notes have already been applied. " +
	"Re-run the full test suite and confirm behavior is unchanged; every test must still pass. " +
	"Report `passed` plus any remaining `refactorNotes`."

const execRefactorApplyText = "TDD REFACTOR phase active — apply mode. Apply each note in the " + NameRefactorNotes + " section verbatim. " +
	"Do NOT change test behavior — tests must still pass. When finished, report `refactorApplied: true`; the verifier will re-run tests. " +
	"A note that targets a test file is not yours to apply — the test-writer owns those. " +
	"Apply the notes that touch production code and report normally; do not report the whole round blocked because one note names a test file."

const execRefactorSkipVerifyText = "TDD REFACTOR phase active — apply mode, and no verifier runs after you. Apply each note in the " + NameRefactorNotes + " section verbatim. " +
	"Because nobody re-runs the suite for you, run the project's own test command yourself and confirm every test still passes before reporting. " +
	"Then submit ALL THREE of `passed: true`, `refactorApplied: true` and `completed: [<taskId>]` in the SAME status report — that single submit advances the task. " +
	"If a test fails or a note cannot be applied, report `blocked` with the reason instead."

const execVerifyFailedText = "The previous verification of this task FAILED. Address every item below before anything else."

const execExecutorText = "Implement the clean, working code required to satisfy every criterion in the " + NameCriteria + " section. " +
	"Do NOT run tests, do NOT write new tests, and do NOT self-verify — an independent verifier does that after you. " +
	"List every path you wrote in `filesModified`, and report `completed: [<taskId>]` when the task is done." +
	FilesModifiedPathNote

const execExecutorSkipVerifyText = "Implement the clean, working code required to satisfy every criterion in the " + NameCriteria + " section. " +
	"Do NOT write new tests. Verification is disabled for this spec, so your report is the final word for this task: " +
	"run the project's own test and type-check commands yourself before reporting, list every path you wrote in `filesModified`, " +
	"and report `completed: [<taskId>]` in that same report." +
	FilesModifiedPathNote

const execVerifierText = "Verify this task independently. You did not write the code and must not trust any self-report about it. " +
	"Start from the " + NameChangedFiles + " section when it is present; when it is absent, discover the changed files yourself. " +
	"Run the project's full test suite, then check each criterion by its `ac-N` id and each edge case by its `ec-N` id. " +
	"Report `passed` (bool), `failedACs` (list of `ac-N` ids), `uncoveredEdgeCases` (list of `ec-N` ids), and `refactorNotes`. " +
	"`refactorNotes` is an array of OBJECTS, each `{\"file\":\"...\",\"suggestion\":\"...\",\"rationale\":\"...\"}` — a list of plain strings is rejected. " +
	"If you found something the user must hear that is neither a failed criterion nor a refactor note — a project command that does not run, a criterion whose wording does not match this repository — " +
	"put it in `reason`; on a passing report that is the only field that reaches the user. " +
	"When a " + NameCoverage + " section is present, also report the per-file coverage array it describes; omit that field otherwise."

const execGateText = "This task is gated: it needs an approved plan before any code is written. " +
	"You are read-only — do NOT edit, create, or delete a single file in this phase. " +
	"Every section above is your input scope; read the code it points at, then produce the plan. " +
	"Return `touchedFiles` (every file the work will modify, and nothing speculative), `approach` (a full narrative, not a summary), " +
	"`assumptions`, `designPatterns`, and `bestPractices`. " +
	"`touchedFiles` becomes binding once the user approves it, so a file you omit will stop the executor cold — list what the work actually needs. " +
	"State each assumption explicitly rather than resolving an ambiguity silently; the user reviews these before approving. " +
	"When a " + NamePriorFeedback + " section is present, the user already rejected a plan: address that feedback directly in this revision."

const GateOrchestratorText = "Important task gate: the task entry with stage \"gate\" needs an approved plan before execution. " +
	"Spawn `tddmaster-planner` (read-only) and pass that entry's `instruction` to it VERBATIM. " +
	"FIRST, BEFORE calling any question tool, present the FULL plan the planner returns to the user as a long plain-text message: " +
	"the complete `approach` narrative, every `touchedFiles` entry with the reason it is touched, each `designPattern` with how it is applied, each `bestPractice`, and every `assumption`. " +
	"Do NOT put the plan content inside the question tool — the question tool call must contain ONLY the accept / revise / reject choice. A one-line summary instead of the full presentation is a protocol violation. " +
	"ONLY AFTER the full plan text has been presented, call AskUserQuestion with accept / revise / reject. " +
	"On accept, submit the plan to `next` with the gated `taskId`. The approved plan is binding — if work later requires a file outside `touchedFiles`, the sub-agent must stop and report blocked."

const RestartRecommendedText = "Iteration limit reached. Start a new conversation to continue, or submit `next <slug> --answer=\"continue\"` to reset the iteration counter and resume."

const ReportExampleExecutor = `{"completed":["task-1"],"remaining":[],"blocked":[],"filesModified":["internal/foo/bar.go"],"phase":"green","refactorApplied":false}`

const ReportExampleRefactorApply = `{"completed":["task-1"],"remaining":[],"blocked":[],"filesModified":["internal/foo/bar.go"],"phase":"refactor","refactorApplied":true}`

const ReportExampleRefactorApplySkip = `{"passed":true,"completed":["task-1"],"remaining":[],"blocked":[],"filesModified":["internal/foo/bar.go"],"phase":"refactor","refactorApplied":true}`

const ReportExampleVerifier = `{"passed":true,"phase":"green","failedACs":[],"uncoveredEdgeCases":[],"refactorNotes":[{"file":"internal/foo/bar.go","suggestion":"extract validation","rationale":"reused in two places"}],"fileCoverage":[{"file":"internal/foo/bar.go","coverage":85}]}`

const ReportExamplePlanner = `{"accepted":true,"plan":{"taskId":"","touchedFiles":["internal/foo/bar.go","internal/foo/bar_test.go"],"approach":"Implement X by extending Y with Z pattern","assumptions":["existing tests cover happy path"],"designPatterns":["strategy"],"bestPractices":["single responsibility"]}}`

const ReportExampleTestWriter = `{"testsWritten":["TestFoo_HappyPath","TestFoo_EdgeCase"],"filesModified":["internal/foo/bar_test.go"],"traceability":[{"testFilePath":"internal/foo/bar_test.go","functionName":"TestFoo_HappyPath","taskId":"task-1","ac":["ac-1"],"ec":["ec-1"]}]}`

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
