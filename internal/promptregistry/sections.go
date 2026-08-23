package promptregistry

// A section name is the bare label; the header wraps it in `=== ... ===`.
// Prose that needs to point at a section MUST reference the Name form: an
// orchestrator splits the prompt on header lines, and reproducing the
// delimiters mid-sentence turns a cross-reference into a phantom boundary.
const (
	NameWorktree      = "WORKTREE (binding)"
	NameTask          = "TASK"
	NameSpecOverview  = "SPEC OVERVIEW"
	NameSpecContext   = "SPEC CONTEXT"
	NameSpecIntent    = "SPEC INTENT"
	NameCriteria      = "ACCEPTANCE CRITERIA"
	NameEdgeCases     = "EDGE CASES"
	NameOutOfScope    = "OUT OF SCOPE"
	NameVerification  = "VERIFICATION METHOD"
	NameApprovedPlan  = "APPROVED PLAN"
	NameChangedFiles  = "CHANGED FILES"
	NameTraceability  = "TEST TRACEABILITY"
	NameRefactorNotes = "REFACTOR NOTES"
	NameLastFailure   = "LAST VERIFICATION FAILURE"
	NameCoverage      = "COVERAGE"
	NameParallelTasks = "PARALLEL TASKS"
	NameRules         = "RULES"
	NameInstruction   = "INSTRUCTION"
	NamePriorFeedback = "PRIOR PLAN FEEDBACK"
	NameProjectCmds   = "PROJECT COMMANDS"
)

const (
	SectionWorktree      = "=== " + NameWorktree + " ==="
	SectionTask          = "=== " + NameTask + " ==="
	SectionSpecOverview  = "=== " + NameSpecOverview + " ==="
	SectionSpecContext   = "=== " + NameSpecContext + " ==="
	SectionSpecIntent    = "=== " + NameSpecIntent + " ==="
	SectionCriteria      = "=== " + NameCriteria + " ==="
	SectionEdgeCases     = "=== " + NameEdgeCases + " ==="
	SectionOutOfScope    = "=== " + NameOutOfScope + " ==="
	SectionVerification  = "=== " + NameVerification + " ==="
	SectionApprovedPlan  = "=== " + NameApprovedPlan + " ==="
	SectionChangedFiles  = "=== " + NameChangedFiles + " ==="
	SectionTraceability  = "=== " + NameTraceability + " ==="
	SectionRefactorNotes = "=== " + NameRefactorNotes + " ==="
	SectionLastFailure   = "=== " + NameLastFailure + " ==="
	SectionCoverage      = "=== " + NameCoverage + " ==="
	SectionParallelTasks = "=== " + NameParallelTasks + " ==="
	SectionRules         = "=== " + NameRules + " ==="
	SectionInstruction   = "=== " + NameInstruction + " ==="
	SectionPriorFeedback = "=== " + NamePriorFeedback + " ==="
	SectionProjectCmds   = "=== " + NameProjectCmds + " ==="
)

var AllSections = []string{
	SectionWorktree,
	SectionTask,
	SectionSpecOverview,
	SectionSpecContext,
	SectionSpecIntent,
	SectionCriteria,
	SectionEdgeCases,
	SectionOutOfScope,
	SectionVerification,
	SectionApprovedPlan,
	SectionChangedFiles,
	SectionTraceability,
	SectionRefactorNotes,
	SectionLastFailure,
	SectionCoverage,
	SectionParallelTasks,
	SectionRules,
	SectionInstruction,
	SectionPriorFeedback,
	SectionProjectCmds,
}

const SectionContractNote = "Every sub-agent prompt is assembled from section blocks, each opened by a line of the form `=== NAME ===`. " +
	"Only a line that is nothing but such a header starts a section; a section name mentioned inside a sentence is a cross-reference, not a boundary. " +
	"A section is present only when it carries data; never assume a section exists. " +
	"There are no JSON fields in the prompt body — read the section blocks."

const SpecContextNote = "Verbatim context the user gave when this spec was created. " +
	"It is the primary reference for intent; the acceptance criteria refine it, they do not replace it.\n"

const OutOfScopeNote = "Declared out of scope for this spec. Do NOT implement, refactor, or test anything listed here. " +
	"If the task appears to require it, STOP and report blocked with reason `out-of-scope`.\n"

const VerificationNote = "The user declared how this work is verified. Follow this method; do not substitute your own. " +
	"It outranks the " + NameProjectCmds + " section: where a detected command contradicts this method, this method wins.\n"

const ChangedFilesNote = "Files reported as modified by the implementation stage of this task. Start here.\n"

const GreenChangedFilesNote = "Files written by the preceding stages of this task. The list always LEADS with every test file the RED stage wrote — " +
	"on a retry round your own earlier implementation files follow them. The failing tests you must make pass live in those leading test files. Read them first.\n"

const SpecOverviewNote = "Every task in this spec and its current state. Only the task marked `<- YOU` is yours to implement. " +
	"The others are listed so you neither duplicate nor pre-empt them: a behavior owned by another task is out of scope for you.\n"

const SpecIntentNote = "The user's discovery answers about WHY this spec exists. " +
	"Use them to resolve ambiguity in the criteria and to weigh risk. They never widen the scope.\n"

const TraceabilityNote = "Tests already written for this task, mapped to the criteria and edge cases they cover. " +
	"Do not duplicate an existing test; add only what is missing. A criterion or edge case absent from this list has no test yet.\n"

const RedVerifyFailedNote = "The previous verification failed. Write or repair tests so every item below is covered by a real, failing-first test. " +
	"Do NOT write production code — the executor fixes the implementation in the next GREEN round.\n"

const ParallelTasksNote = "These tasks run concurrently in separate worktrees. Do NOT touch files owned by them. " +
	"If your work requires one, STOP and report blocked with reason `cross-task-file-conflict`.\n"

const ParallelTasksNoGitNote = "These tasks are part of the same batch and share this working directory — there are no worktrees here. " +
	"Do NOT touch files owned by them. If your work requires one, STOP and report blocked with reason `cross-task-file-conflict`.\n"

// ParallelTasksUnknownFilesNote covers the first round of a batch, when no
// sibling has written anything yet and no plan pins its files. An `owns:` list
// is the only concrete guard in this section, so its absence has to be stated.
const ParallelTasksUnknownFilesNote = "An entry without an `owns:` list has not declared its files yet. " +
	"Derive them from that task's title and stay away from anything it plainly owns; when in doubt, report blocked rather than guessing.\n"

const RefactorNotesNote = "Apply every note below verbatim. Behavior must not change.\n"

const CriterionIDNote = "Reference every criterion by its `ac-N` id and every edge case by its `ec-N` id, exactly as printed below.\n"

const ChallengedPremisesLabel = "premises the user did NOT accept (a rejection outranks any assumption you would otherwise make):"

// ProjectCommandsNote states its own precedence: detection reads config files,
// so it can surface a command the repository cannot actually run. The user's
// declared verification method is the authority whenever the two disagree.
const ProjectCommandsNote = "Build, type-check, and test commands detected from the project root. " +
	"Use these instead of deriving your own, with one exception: the " + NameVerification + " section outranks this one. " +
	"Where a command here contradicts the declared verification method, follow that method and report the mismatch. " +
	"If one of these commands turns out to be wrong for this repository, report that instead of silently substituting another command.\n"

// FilesModifiedPathNote pins the path form of every reported file. Without it a
// sub-agent working inside a worktree may report its cwd-prefixed path while an
// earlier stage reported the repo-relative one, and the same file then appears
// twice in the CHANGED FILES section under two names.
const FilesModifiedPathNote = " Report every path in `filesModified` relative to the directory you were told to work in — " +
	"the worktree cwd when a " + NameWorktree + " section is present, the project root otherwise. " +
	"Never prefix a path with the worktree location."

const RulesTruncatedNote = "The rule files are too large to inline. Read every path below before you start — " +
	"their contents are binding even though they are not reproduced here.\n"
