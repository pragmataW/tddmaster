package promptregistry

const RulesInjectionHeader = "\n" + SectionRules + "\nYou MUST read and follow these project rule files before doing anything. They are mandatory project constraints, not suggestions:\n"

const RulesInjectionFooter = "Do not proceed until you have read every file listed above.\n"

const RulesInlineHeader = "\n" + SectionRules + "\nThese are mandatory project constraints, not suggestions. " +
	"The full text of every rule file is inlined below — you do not need to read the files.\n"

const RulesInlineFooter = "Every line above is binding for this task.\n"

// WorktreeBlockFmt prints cwd as an absolute path and names the project root
// alongside it. A bare relative cwd resolves against whatever directory the
// sub-agent happens to start in, and `.tddmaster/worktrees/...` exists under
// more than one repository on a typical machine.
const WorktreeBlockFmt = SectionWorktree + "\nprojectRoot: %s\ncwd: %s\nbranch: %s\n" +
	"cwd is absolute — use it verbatim, do not re-resolve it against your own working directory. " +
	"All file reads, edits, tests and coverage runs MUST happen inside this cwd. " +
	"Writing outside it is a protocol violation. Do not run git; the orchestrator owns worktree lifecycle.\n\n"

const BatchSummaryFmt = "%d task(s) ready: %s. " +
	"%s Spawn the `delegateAgent` named in that task's entry: never substitute a different agent for it, " +
	"and never run a verifier of your own — the engine dispatches verification as its own stage. " +
	"A note below may tell you to spawn one further agent ALONGSIDE an entry; apart from what such a note authorises, spawn nothing else. " +
	"Pass each entry's `instruction` to its sub-agent VERBATIM, including every section block. " +
	"Sub-agents never call the CLI: submit one report per task yourself, and every report MUST include its `taskId`. " +
	"Only spawn sub-agents for tasks without an in-flight report."

const BatchWorktreeIsolation = "Run them IN PARALLEL, each in its own worktree via a separate sub-agent, using the `worktree` block carried by that entry."

const BatchSingleWorktreeIsolation = "Run it in its own worktree via the `delegateAgent` and `worktree` block carried by that entry."

// BatchTestWriterNote covers the one case the stage machine cannot: with TDD off
// there is no red stage, and the executor is forbidden from writing tests, so a
// task that genuinely needs test files would dead-end on a blocked report.
const BatchTestWriterNote = " Note: TDD is off for this batch, so no red stage runs and the executor may not write tests. " +
	"If a task's work genuinely needs test files, spawn `tddmaster-test-writer` for that task yourself first, then run its executor entry as usual."

// BatchRefactorTestWriterNote covers the second case the stage machine cannot.
// Refactor notes come from the verifier scanning every modified file, so a note
// can land on a test file — but the refactor stage always dispatches the
// executor, which refuses to touch tests. The engine has no stage that clears
// that report, and an unblock submit re-emits the identical round, so the task
// blocks forever unless the orchestrator brings in the test-writer itself.
const BatchRefactorTestWriterNote = " Note: this batch contains a refactor round, and refactor notes are produced by scanning every file the task modified — " +
	"some may therefore target test files, which the executor is forbidden to edit. " +
	"Read the " + NameRefactorNotes + " section of each refactor entry and decide per task: if any note targets a test file, " +
	"spawn `tddmaster-test-writer` for that task first — in the same worktree — and have it apply exactly those notes, then run the executor entry for the remaining ones. " +
	"When every note targets production code, run the executor entry alone."

const BatchNoGitIsolation = "This project is NOT a git repository, so there are no worktrees and no isolation between concurrent writes: " +
	"run these tasks SEQUENTIALLY in the project root, one sub-agent at a time."

const ParallelDispatchDirective = "\n\nWhile the plan gate is pending, dispatch the non-gate entries in `tasks` as parallel sub-agents; the entry with stage \"gate\" is the question itself, not a task to spawn."

// VerifierNoteFmt surfaces a passing verifier's own prose. `reason` and
// `output` are the only channel it has for a finding that is neither a failed
// criterion nor a refactor note — a broken project command, a criterion whose
// wording does not match the repository — and without this the text is stored
// and never seen again.
const VerifierNoteFmt = "\n\nVerifier note on %s (the task still passed — relay this to the user): %s"

const GateReappearDirective = "\n\nThis gate re-appears after every sibling report submit until its answer is submitted. If the planner was already spawned and the plan already presented, do NOT spawn the planner or re-ask the user again — just submit the pending gate answer when it is ready."

const CoverageUnreportedText = "The previous verification reported no coverage measurements. You MUST run the coverage tool now and return a non-empty fileCoverage:[{file,coverage}]. An empty report blocks the cycle and will be rejected.\n"

const CoverageRequirementFmt = "Coverage requirement: measure test coverage for each touched file. " +
	"When the " + NameProjectCmds + " section lists a coverage command, run that one; otherwise use the project's language-appropriate coverage tool. " +
	"Each file must reach %d%% coverage. " +
	"Report results as fileCoverage:[{file,coverage}]. " +
	"For each file below the threshold, propose new tests.\n" +
	"You are the only agent that measures coverage — no one runs it for you, so do not assume a prior measurement exists.\n"

const CoverageLowFeedbackHeader = "\nThe following files have low test coverage and need additional tests:\n"

const CoverageLowFeedbackFooter = "Add tests to bring these files above the coverage threshold.\n"

const AuditorAnalysisHeader = "Perform a cross-artifact analysis of the task list below. Return JSON {\"verdict\":\"clean|issues|block\",\"findings\":[{severity,category,taskId,acId,detail,suggestion,source}]}."

const AuditorSeverityPolicy = "Severity must be one of: block, warn, info. STRICT POLICY: any finding with severity other than info pauses the phase for an explicit user decision. Use info ONLY for purely advisory notes that need no action; if a finding implies any change to the tasks or criteria, use warn or block."

const AnalysisDecisionHeader = "The analysis flagged findings that need your decision (every finding except info severity). Choose how to proceed:"

const AnalysisAdvisoryHeader = "The cross-artifact analysis passed. It left advisory notes — display them to the user, then call `next` again to continue into execution:"

const ExampleAnalysisVerdict = `{"verdict":"clean","findings":[]}`

const ExampleAnalysisDecision = `accept-anyway | {"action":"edit","payload":{"update":{"task-1":{"criteria":[{"then":"..."}]}}}}`

const ExampleRuleProposal = `{"rules":[{"scope":"executor","name":"no-globals","content":"Never use global variables.","rationale":"..."}]}`

const ExampleRuleApproval = `{"accepted":true} | {"accepted":false} | {"feedback":"..."}`

const RuleProposalDirective = "\n\nReturn the proposal as JSON: " + ExampleRuleProposal +
	"\n`scope` is the sub-agent the rule binds (executor, test-writer, verifier, planner, auditor) or `global` for every agent. Do NOT write any rule file yourself in this step."

const RuleReviewHeader = "Review the proposed rules below and decide: accept applies them, revise re-runs the synthesizer with your feedback, reject writes nothing."

const DiscoverySynthesisText = "Discovery is complete. Present the synthesis below to the user in full — it is everything the spec will be built from, " +
	"and it is the user's last chance to correct an answer before tasks are generated. " +
	"When they confirm it reads correctly, submit \"approve\" to continue; to change an answer, roll back with `tddmaster rollback <slug> discovery`."

const DiscoverySynthesisHeader = "\n=== DISCOVERY SYNTHESIS ===\n"

const RuleApplyReportDirective = "\nWhen every rule has been written, report back so the phase can close: " +
	"submit `applied` once all commands above have succeeded, or the error text if one failed."

const ExamplePremises = `{"premises":[{"text":"...","agreed":true,"revision":"..."}]}`

const ExampleTaskGen = `{"tasks":[{"title":"...","criteria":[{"then":"..."}],"linkedEdgeCases":["..."]}]}`

type SettingOption struct {
	Label       string
	Description string
}

var SettingsOptions = []SettingOption{
	{Label: "TDD (Red-Green-Refactor)", Description: "Enforce failing-test-first cycles per task. Default: ON."},
	{Label: "Skip verifier", Description: "Skip non-TDD verification and TDD post-refactor re-verification. TDD green verification always runs. Default: OFF."},
	{Label: "Important task gate", Description: "Pause tasks flagged important for a plan-first review before execution. Default: OFF."},
	{Label: "Min test coverage", Description: "Coverage gate for the TDD green-phase verifier. Ask the user for a percentage (0-100) and submit it as the number minTestCoverage; 0 disables. Default: 80."},
	{Label: "Rule learning", Description: "Derive tddmaster rules from refactor notes and failed ACs after execution. Default: OFF."},
}
