package promptregistry

const RulesInjectionHeader = "\nYou MUST read and follow these project rule files before doing anything. They are mandatory project constraints, not suggestions:\n"

const RulesInjectionFooter = "Do not proceed until you have read every file listed above."

const WorktreeBlockFmt = "=== WORKTREE (binding) ===\ncwd: %s\nbranch: %s\n" +
	"All file reads, edits, tests and coverage runs MUST happen inside this cwd. " +
	"Writing outside it is a protocol violation. Do not run git; the orchestrator owns worktree lifecycle. " +
	"NO-GIT FALLBACK: if the project is not a git repository, ignore this block entirely and work in the project root.\n===\n\n"

const ParallelDispatchDirective = "\n\nWhile the plan gate is pending, dispatch the non-gate entries in `tasks` as parallel sub-agents; the entry with stage \"gate\" is the question itself, not a task to spawn."

const GateReappearDirective = "\n\nThis gate re-appears after every sibling report submit until its answer is submitted. If the planner was already spawned and the plan already presented, do NOT spawn the planner or re-ask the user again — just submit the pending gate answer when it is ready."

const CoverageUnreportedText = "\nThe previous verification reported no coverage measurements. You MUST run the coverage tool now and return a non-empty fileCoverage:[{file,coverage}]. An empty report blocks the cycle and will be rejected.\n"

const CoverageRequirementFmt = "\nCoverage requirement: measure test coverage for each touched file using the project's language-appropriate coverage tool. " +
	"Each file must reach %d%% coverage. " +
	"Report results as fileCoverage:[{file,coverage}]. " +
	"For each file below the threshold, propose new tests.\n" +
	"Coverage measurement is performed exclusively by you, the verifier sub-agent. " +
	"The orchestrator must delegate this to you and must not run any coverage tooling itself.\n"

const CoverageLowFeedbackHeader = "\nThe following files have low test coverage and need additional tests:\n"

const CoverageLowFeedbackFooter = "Add tests to bring these files above the coverage threshold.\n"

const AuditorAnalysisHeader = "Perform a cross-artifact analysis of the task list below. Return JSON {\"verdict\":\"clean|issues|block\",\"findings\":[{severity,category,taskId,acId,detail,suggestion,source}]}."

const AuditorSeverityPolicy = "Severity must be one of: block, warn, info. STRICT POLICY: any finding with severity other than info pauses the phase for an explicit user decision. Use info ONLY for purely advisory notes that need no action; if a finding implies any change to the tasks or criteria, use warn or block."

const AnalysisDecisionHeader = "The analysis flagged findings that need your decision (every finding except info severity). Choose how to proceed:"

const ExampleAnalysisVerdict = `{"verdict":"clean","findings":[]}`

const DiscoverySynthesisText = "Review the discovery synthesis above. When ready, submit \"approve\" to continue."

const ExamplePremises = `{"premises":[{"text":"...","agreed":true,"revision":"..."}]}`

const ExampleTaskGen = `{"tasks":[{"title":"...","criteria":[{"then":"..."}],"linkedEdgeCases":["..."]}]}`

type SettingOption struct {
	Label       string
	Description string
}

var SettingsOptions = []SettingOption{
	{Label: "TDD (Red-Green-Refactor)", Description: "Enforce failing-test-first cycles per task. Default: ON."},
	{Label: "Skip verifier", Description: "Skip the independent verifier sub-agent after the green stage. Default: OFF."},
	{Label: "Important task gate", Description: "Pause tasks flagged important for a plan-first review before execution. Default: OFF."},
	{Label: "Min test coverage", Description: "Coverage gate for the TDD green-phase verifier. Ask the user for a percentage (0-100) and submit it as the number minTestCoverage; 0 disables. Default: 80."},
	{Label: "Rule learning", Description: "Derive tddmaster rules from refactor notes and failed ACs after execution. Default: OFF."},
}
