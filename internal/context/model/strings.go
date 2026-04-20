package model

// Welcome is the IDLE welcome banner.
const Welcome = "tddmaster is a state-machine orchestrator that acts as a scrum master for both you and your agent — keeping work focused, decisions in your hands, and tokens efficient."

// GitReadonlyRule is the behavioural rule forbidding git write commands.
const GitReadonlyRule = "NEVER run git write commands (commit, add, push, checkout, stash, reset, merge, rebase, cherry-pick). Git is read-only for agents. The user controls git. You may read: git log, git diff, git status, git show, git blame."

// Mandatory behavioural rules — preamble shared across every phase.
const (
	MandatoryRuleProgress       = "Report progress honestly. Not done = 'not done'. Partial = 'partial: [works]/[doesn't]'. Untested = 'untested'. 4 of 6 = '4 of 6 done, 2 remaining'."
	MandatoryRuleNeverSkipFmt   = "Never skip steps or infer decisions. %s Recommend first, then ask. One tddmaster call per interaction — never batch-submit or backfill."
	MandatoryRuleRoadmapFirst   = "Display `roadmap` before other content. Display `gate` prominently."
	MandatoryRuleNoBypass       = "NEVER suggest bypassing, skipping, or 'breaking out of' tddmaster. Discovery helps the user — it is not an obstacle. If scope changes: revise spec, reset and create new, or split."
	MandatoryRuleNoPermission   = "NEVER ask permission to run the next tddmaster command. After spec new → run next immediately. After answering questions → run next. After approve → run next. After task completion → run next. The workflow is sequential — each step has one next step. Just run it."
	MandatoryRuleListenFirst    = "Listen first: after spec creation, ask 'Tell me about this — share as much context as you have.' Wait for their response before mode selection. Rich context (>200 chars) → pre-fill discovery answers as STATED/INFERRED. Brief response → proceed normally."
	MandatoryRuleAdaptiveQ      = "Discovery questions are adaptive. After each answer, generate 1-3 follow-up questions if the answer reveals ambiguity, risk, dependencies, or missing detail. Submit follow-ups via `tddmaster spec <name> followup <questionId> \"question\"`. Max 3 per question. Do NOT rush through discovery."
	MandatoryRuleConfidence     = "Confidence scoring: every technical finding needs a confidence score (1-10). 9-10: verified (read code, ran test). 7-8: strong evidence. 5-6: reasonable inference. 3-4: guess. 1-2: speculation. State basis ('read X', 'inferred from Y'). If confidence < 5, prefix with '\u26A0 Unverified:'."
)

// AskUserStrategy-specific ask method templates for mandatoryRules second entry.
const (
	AskMethodBlock    = "Do not ask the user inline. Instead, use `tddmaster block \"question\"` at every decision point — the orchestrator will pause for human input."
	AskMethodProse    = "Present options as a numbered list at every decision point."
	AskMethodAskUser  = "Use AskUserQuestion for all decision points."
)

// Discovery mode rule sets.
var ModeRulesFull = []string{
	"Ask each discovery question as written. Push for specific, concrete answers.",
	"If the answer is vague, ask follow-up questions before accepting.",
}

var ModeRulesValidate = []string{
	"The user has a plan. Your job is to challenge it, not explore it.",
	"For each question, identify assumptions and ask: 'What would prove this wrong?'",
	"If the description already answers a question, present your understanding and ask to confirm.",
	"When pre-filling answers from a rich description, plan, or prior discussion, DISTINGUISH between what the user EXPLICITLY STATED and what you INFERRED. Format each pre-filled item as: '[STATED] GPU skinning in all 3 renderers — you said this during technical discussion' or '[INFERRED] tangent space is 10-star scope — I assumed this based on complexity'. The user confirms stated items and corrects inferred items.",
	"Present pre-filled answers ONE ITEM AT A TIME for confirmation, not as a completed block. The user's job is to correct your inferences, not rubber-stamp your summary. If you pre-fill 5 items and 2 are wrong, the user must be able to catch them individually.",
}

var ModeRulesTechnicalDepth = []string{
	"Focus on architecture, data flow, performance, and integration points.",
	"Before each question, scan the codebase for related implementations.",
	"Ask: 'How does this interact with [existing system]?' for each integration point.",
}

var ModeRulesShipFast = []string{
	"Focus on minimum viable scope.",
	"For each question, also ask: 'What can we defer to a follow-up?'",
	"Push for the smallest version that delivers value.",
}

var ModeRulesExplore = []string{
	"Think bigger. What's the 10x version?",
	"For each question, ask about adjacent opportunities.",
	"Suggest possibilities the user might not have considered.",
}

// TDDBehavioralRules is the canonical set of TDD behavioural rules injected
// when TDD mode is active. Keeping them in a package-level slice lets tests
// verify the exact rule texts without coupling to buildBehavioral internals.
var TDDBehavioralRules = []string{
	"TDD REQUIRED: Write tests before implementation. No production code without a failing test.",
	"TDD REQUIRED: Follow red-green-refactor. (1) Write test — it MUST fail. (2) Write minimum code to make it pass. (3) Refactor without breaking tests.",
	"TDD REQUIRED: Test task comes first. When a task has a corresponding test task, complete the test task before any implementation task.",
	"TDD REQUIRED: In RED phase, test-writer writes tests only — it does NOT run them. tddmaster-verifier in RED phase performs read-only inspection (no test execution) to confirm tests are well-formed.",
	"TDD REQUIRED: Sub-agent selection for each task follows the cycle. Do NOT conflate roles — test-writer writes tests only, tddmaster-executor writes implementation or applies refactor notes only, tddmaster-verifier is read-only and evaluates each step. Delegation table (drive it by `tddPhase`, `lastVerification.phase`, and `refactorInstructions`):\n" +
		"  - tddPhase='red' && no verifier result yet OR lastVerification.phase!='red' → spawn `test-writer`. Pass the spec's edge cases explicitly.\n" +
		"  - tddPhase='red' && lastVerification.phase='red' && lastVerification.passed=false → spawn `tddmaster-verifier` with phase='red'. It reads test files without running them and confirms they are well-formed (readOnly: true).\n" +
		"  - tddPhase='green' && (no verifier result OR lastVerification.phase!='green') → spawn `tddmaster-executor` with the task; executor writes minimum implementation (does NOT run tests).\n" +
		"  - tddPhase='green' && lastVerification.phase='green' && lastVerification.passed=false → spawn `tddmaster-verifier` with phase='green' to run and re-check tests.\n" +
		"  - tddPhase='green' && lastVerification.phase='green' && lastVerification.passed=true → tddmaster already advanced to the next phase; no action needed here — run `next`.\n" +
		"  - tddPhase='refactor' && refactorInstructions is present → spawn `tddmaster-executor` to apply the notes (from the GREEN scan or a prior REFACTOR re-check) verbatim and report `refactorApplied: true` (does NOT run tests).\n" +
		"  - tddPhase='refactor' && refactorInstructions is absent → spawn `tddmaster-verifier` with phase='refactor'. This is a regression-check: it runs tests to confirm the executor's changes did not break behavior, and optionally produces new refactorNotes for another round. An empty array means the task is clean.\n" +
		"Pass the full tddmaster `next` output to whichever sub-agent you spawn. If `edgeCases` is non-empty, pass them explicitly to test-writer.",
}

// Discovery / prefill instructions.
const (
	PreDiscoveryResearchInstruction = "Before asking discovery questions, research the current state of all platforms, runtimes, libraries, and APIs mentioned in the spec description. Use web search and Context7 MCP if available. Report findings as a pre-discovery brief to the user. Do NOT assume your training data is current — versions change, APIs get added, features get deprecated."

	PlanContextInstruction = "A plan document was provided. Read it carefully, extract relevant information for each discovery question, and present pre-filled answers for user review. Do NOT skip any question — present your extraction and let the user confirm, correct, or expand. IMPORTANT: When extracting answers from the plan, mark each extraction as [STATED] (directly written in the plan) or [INFERRED] (your interpretation). Present extractions individually for confirmation."

	PlanContextOversizedInstruction = "A plan document was provided but it exceeds the 50KB embedding limit. Read it directly from the filesystem (path is available in state.PlanPath) and apply the same extraction protocol: for each discovery question, extract relevant information and mark it [STATED] or [INFERRED]. Do NOT skip any question."

	RichDescriptionInstructionAgent = "The user provided a detailed description. For each question, extract relevant info and present as a pre-filled suggestion. IMPORTANT: When extracting answers from the description, mark each extraction as [STATED] (directly written by the user) or [INFERRED] (your interpretation). Present extractions individually for confirmation."

	RichDescriptionInstructionUser = "The user provided a detailed description. For each question, extract relevant info and present as a pre-filled suggestion."

	DiscoveryListenFirstInstruction = "The user just created this spec. Before starting discovery, ask them to share whatever context they have — requirements, notes, tasks, or just a brief description. Say: 'Tell me about this — share as much context as you have.' Listen first, then proceed."

	DiscoveryModeSelectionInstruction = "Before starting discovery, select the discovery mode via AskUserQuestion. Use the options provided in interactiveOptions — do NOT present them as prose or a numbered list."

	ModeSelectionOutputInstruction = "Select the discovery mode via AskUserQuestion. Present the five options exactly as listed in interactiveOptions — do not paraphrase them into prose."

	DiscoveryPremiseInstruction = "Before asking discovery questions, challenge the premises of this spec."

	DiscoveryRevisitedInstruction = "This spec was revisited from EXECUTING. Previous discovery answers are preserved — review and revise as needed, or approve to regenerate tasks."

	DiscoveryNormalInstruction = "Conduct a thorough discovery conversation. FIRST: perform a pre-discovery codebase scan (README, CLAUDE.md, recent git log, TODOs, directory structure) and present a brief audit summary. THEN: challenge the user's spec description against your findings. THEN: ask the current discovery question, wait for the answer, and submit it immediately with `tddmaster next --answer=\"<answer>\"`. Continue one question at a time, offering concrete options based on codebase knowledge. AFTER questions: present a dream state table (current → this spec → future), scored expansion proposals, architectural decisions, and an error/rescue map. FINALLY: present a complete discovery synthesis for user confirmation before approve."

	PremiseChallengeInstructionFmt = "Read the spec description%s. Identify 2-4 premises the spec assumes. Present each premise and ask the user to agree or disagree. Submit as JSON: {\"premises\":[{\"text\":\"...\",\"agreed\":true/false,\"revision\":\"...\"}]}"
)

// Discovery review / split / alternatives / checklist instructions.
const (
	DiscoveryReviewDefaultInstruction = "FIRST render `discoveryReviewData.reviewSummary` to the user. The user must confirm or correct each answer before the spec can be generated. Only AFTER the review should you ask whether to approve or revise."

	DiscoveryReviewSplitInstruction = "FIRST render `discoveryReviewData.reviewSummary` to the user. ALSO present the split proposal — tddmaster detected multiple independent areas. Only after the answer list is visible should you ask the user what to do next."

	DiscoveryReviewSplitApprovedInstruction = "FIRST render `discoveryReviewData.reviewSummary` to the user. THEN present the split proposal and let them decide whether to keep this as one spec or split it into separate specs."

	DiscoveryReviewAlternativesInstruction = "FIRST render `discoveryReviewData.reviewSummary` to the user. THEN propose 2-3 distinct implementation approaches with name, summary, effort (S/M/L/XL), risk (Low/Med/High), pros, and cons. Ask the user to choose one, or skip."

	AlternativesInstruction = "Generate 2-3 approaches from discovery answers and codebase. Present via AskUserQuestion."

	BatchWarning = " IMPORTANT: These answers were BATCH-SUBMITTED (not confirmed one-by-one). You MUST present EVERY answer individually and get explicit user confirmation for each. Do NOT auto-approve."

	ReviewChecklistInstruction = "Before approving, review the plan against each dimension below. For dimensions marked evidenceRequired, cite specific files or code. Present findings to the user for each dimension via AskUserQuestion — one dimension at a time."

	ReviewChecklistRegistryInstruction = "Registry dimensions (isRegistry=true) require a structured table with every row filled. These tables will be included in the generated spec."
)

// Spec draft / approved instructions.
const (
	SpecClassifyInstruction = "Before generating the spec, classify what this spec involves. Ask the user to select all that apply."

	SpecSelfReviewInstructionFmt = "Review draft against these checks. If issues are found, send a refinement — DO NOT put a task list in `notes`. " +
		"Full task replacement: `next --answer='{\"refinement\":\"task-1: Title | task-2: Title | task-3: Title\"}'`. " +
		"Verb patch: `next --answer='{\"refinement\":{\"update\":{\"task-1\":\"New title\"},\"add\":[\"New task\"],\"remove\":[\"task-2\"]}}'`. " +
		"`notes` is reserved for free-form context only. Fix inline; do not ask the user to fix."

	SpecDraftReadyInstruction = "Spec draft is ready. Self-review before presenting to user."

	SpecApprovedWaitingInstruction = "Spec is approved and ready. When the user is ready to start, begin execution."

	SpecApprovedTDDInstruction = "Select TDD scope for this spec before starting execution. Some tasks (infrastructure setup, module downloads) often do not benefit from red/green/refactor."

	TaskTDDSelectionInstruction = "Choose which tasks run with TDD. 'All' keeps current behavior; 'None' skips red/green/refactor for every task; 'Custom' lets you pick task-by-task."
)

// Spec draft self-review checks.
var SpecSelfReviewChecks = []string{
	"Placeholder scan: no TBD, TODO, vague requirements",
	"Consistency: tasks match discovery, ACs match tasks",
	"Scope: single spec, not multiple independent subsystems",
	"Ambiguity: every AC has one interpretation",
	"Edge cases: discovery answers and revised premises are captured for test-writer coverage",
}

// Classification options.
var ClassificationOptions = []ClassificationOption{
	{"involvesWebUI", "Web/Mobile UI — layouts, responsive design, visual components"},
	{"involvesCLI", "CLI/Terminal UI — spinners, progress bars, interactive prompts"},
	{"involvesPublicAPI", "Public API changes"},
	{"involvesMigration", "Data migration or schema changes"},
	{"involvesDataHandling", "Data handling or privacy"},
}

// Design checklist for beautiful-product concern.
var DesignChecklistDimensions = []DesignChecklistDimension{
	{ID: "hierarchy", Label: "Information hierarchy — what does the user see first, second, third?"},
	{ID: "states", Label: "Interaction states — loading, empty, error, success all specified?"},
	{ID: "edge-cases", Label: "Edge cases — long text, zero results, slow connection handled?"},
	{ID: "intentionality", Label: "Overall intentionality — does this feel designed or generated?"},
}

const DesignChecklistInstruction = "Before completing any UI task, rate your implementation 0-10 on these dimensions and include the ratings in your AC report:"

// Execution instructions.
const (
	ExecutionVerificationFailed = "Verification FAILED. Fix the failing tests before continuing."

	ExecutionPreReviewInstruction = "Re-read spec before starting. Flag: missing info that will block mid-execution, wrong task order, unclear ACs. Better to catch now than mid-execution."

	ExecutionStatusReportInstruction = "Before this task is accepted, report your completion status against these acceptance criteria."

	RefactorInstructionsText = "Apply each refactor note verbatim. Do NOT change test behavior — tests must still pass. When finished, report `refactorApplied: true` in your JSON output; the verifier will re-run tests."

	BlockedInstruction = "A decision is needed. Ask the user."
)

// Status report format hints rendered under executionData.statusReport.reportFormat.
var DefaultReportFormat = ReportFormat{
	Completed: "list item IDs you finished (e.g., ['debt-1', 'ac-3']) with evidence",
	Remaining: "list item IDs not yet done",
	Blocked:   "list item IDs that need a decision from the user",
	NA:        "(optional) list item IDs that are not applicable to this task — they will be removed from future criteria",
	NewIssues: "(optional) list NEW issues discovered during implementation — free text, will be assigned debt IDs automatically",
}

// Premise challenge prompts.
var PremiseChallengePrompts = []string{
	"Is this the right problem to solve? Could a different framing yield a simpler solution?",
	"What happens if we do nothing? Is this a real pain point or a hypothetical one?",
	"What existing code already partially solves this? Can we build on it instead?",
}

// Alternatives output fields.
var AlternativesFields = []string{"id", "name", "summary", "effort", "risk", "pros", "cons"}

// Phase-specific behavioural tones.
const (
	ToneIdle                 = "Welcoming. Present choices, then wait."
	ToneDiscovery            = "Curious interviewer with a stake in the answers. Comes PREPARED — read the codebase first. Challenge assumptions, think about architecture and failure modes."
	ToneDiscoveryRefinement  = "Careful reviewer. The user must confirm every answer."
	ToneSpecProposal         = "Thoughtful reviewer preparing to hand off to an implementer."
	ToneSpecApproved         = "Patient. Wait for the go signal."
	ToneExecuting            = "Direct. Orchestrate immediately — spawn sub-agents."
	ToneBlocked              = "Brief. The user is making a decision, not having a discussion."
	ToneCompleted            = "Concise. Celebrate briefly, then stop."
	ToneDefault              = "Neutral. Waiting for direction."
)

// Behavioural mode overrides.
const (
	ModeOverrideDiscovery = "plan mode. DO NOT create, edit, or write any files. DO NOT run state-modifying commands. MAY read files and run read-only commands (cat, ls, grep, git log, git diff)."

	ModeOverrideDiscoveryRefinement = "You are in plan mode. Do not create, edit, or write any files. Present the discovery answers to the user for review and confirmation."

	ModeOverrideSpecProposal = "plan mode. DO NOT create, edit, or write any files. DO NOT run state-modifying commands. MAY read files and run read-only commands."
)
