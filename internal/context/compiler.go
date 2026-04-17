package context

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/pragmataW/tddmaster/internal/defaults"
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/pragmataW/tddmaster/internal/tddcontract"
)

// =============================================================================
// Interaction Hints
// =============================================================================

// InteractionHints describes how the active coding tool presents options and delegates.
type InteractionHints struct {
	HasAskUserTool        bool   `json:"hasAskUserTool"`
	OptionPresentation    string `json:"optionPresentation"` // "tool" | "prose"
	HasSubAgentDelegation bool   `json:"hasSubAgentDelegation"`
	SubAgentMethod        string `json:"subAgentMethod"`  // "task" | "delegation" | "spawn" | "fleet" | "none"
	AskUserStrategy       string `json:"askUserStrategy"` // "ask_user_question" | "tddmaster_block"
}

// DefaultHints are the default interaction hints (Claude Code behavior).
var DefaultHints = InteractionHints{
	HasAskUserTool:        true,
	OptionPresentation:    "tool",
	HasSubAgentDelegation: true,
	SubAgentMethod:        "task",
	AskUserStrategy:       "ask_user_question",
}

// =============================================================================
// Output Types (JSON contract for `tddmaster next`)
// =============================================================================

// PhaseOutput is the union of all possible phase outputs.
type PhaseOutput interface {
	phaseOutput()
}

// ClearContextAction signals that context should be cleared.
type ClearContextAction struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

// GateInfo describes the gate at the current phase transition.
type GateInfo struct {
	Message string `json:"message"`
	Action  string `json:"action"`
	Phase   string `json:"phase"`
}

// InteractiveOption is a single option presented to the user.
type InteractiveOption struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// internalOption adds a command to an InteractiveOption for internal routing.
type internalOption struct {
	Label       string
	Description string
	Command     string
}

// ProtocolGuide is shown on first call or after a stale session.
type ProtocolGuide struct {
	What         string `json:"what"`
	How          string `json:"how"`
	CurrentPhase string `json:"currentPhase"`
}

// EnforcementInfo describes the enforcement level for the active tool.
type EnforcementInfo struct {
	Level        string   `json:"level"` // "enforced" | "behavioral"
	Capabilities []string `json:"capabilities"`
	Gaps         []string `json:"gaps,omitempty"`
}

// MetaBlock is the self-documenting resume context included in every output.
type MetaBlock struct {
	Protocol       string           `json:"protocol"`
	Spec           *string          `json:"spec"`
	Branch         *string          `json:"branch"`
	Iteration      int              `json:"iteration"`
	LastProgress   *string          `json:"lastProgress"`
	ActiveConcerns []string         `json:"activeConcerns"`
	ResumeHint     string           `json:"resumeHint"`
	Enforcement    *EnforcementInfo `json:"enforcement,omitempty"`
}

// BehavioralBlock contains phase-aware guardrails for agent behavior.
type BehavioralBlock struct {
	ModeOverride *string  `json:"modeOverride,omitempty"`
	Rules        []string `json:"rules"`
	Tone         string   `json:"tone"`
	Urgency      *string  `json:"urgency,omitempty"`
	OutOfScope   []string `json:"outOfScope,omitempty"`
	Tier2Summary *string  `json:"tier2Summary,omitempty"`
}

// ContextBlock is the rules+reminders context injected into certain phase outputs.
type ContextBlock struct {
	Rules            []string `json:"rules"`
	ConcernReminders []string `json:"concernReminders"`
}

// NextOutput is the top-level JSON output for `tddmaster next`.
type NextOutput struct {
	// Phase-specific fields are embedded via map for flexibility
	Phase string `json:"phase"`

	// Common fields present in all outputs
	Meta                MetaBlock           `json:"meta"`
	Behavioral          BehavioralBlock     `json:"behavioral"`
	Roadmap             string              `json:"roadmap"`
	Gate                *GateInfo           `json:"gate,omitempty"`
	ProtocolGuide       *ProtocolGuide      `json:"protocolGuide,omitempty"`
	ClearContext        *ClearContextAction `json:"clearContext,omitempty"`
	InteractiveOptions  []InteractiveOption `json:"interactiveOptions,omitempty"`
	CommandMap          map[string]string   `json:"commandMap,omitempty"`
	ToolHint            *string             `json:"toolHint,omitempty"`
	ToolHintInstruction *string             `json:"toolHintInstruction,omitempty"`

	// Phase-specific data — only one is populated
	DiscoveryData       *DiscoveryOutput       `json:"discoveryData,omitempty"`
	DiscoveryReviewData *DiscoveryReviewOutput `json:"discoveryReviewData,omitempty"`
	SpecDraftData       *SpecDraftOutput       `json:"specDraftData,omitempty"`
	SpecApprovedData    *SpecApprovedOutput    `json:"specApprovedData,omitempty"`
	ExecutionData       *ExecutionOutput       `json:"executionData,omitempty"`
	BlockedData         *BlockedOutput         `json:"blockedData,omitempty"`
	CompletedData       *CompletedOutput       `json:"completedData,omitempty"`
	IdleData            *IdleOutput            `json:"idleData,omitempty"`
}

// DiscoveryQuestion is a discovery question as presented in output.
type DiscoveryQuestion struct {
	ID       string                       `json:"id"`
	Text     string                       `json:"text"`
	Concerns []string                     `json:"concerns"`
	Extras   []string                     `json:"extras"`
	Prefills []state.DiscoveryPrefillItem `json:"prefills,omitempty"`
}

// PreDiscoveryResearch is injected when tech version terms are detected in description.
type PreDiscoveryResearch struct {
	Required       bool     `json:"required"`
	Instruction    string   `json:"instruction"`
	ExtractedTerms []string `json:"extractedTerms"`
}

// PlanContext provides plan document content for discovery.
type PlanContext struct {
	Provided    bool   `json:"provided"`
	Content     string `json:"content"`
	Instruction string `json:"instruction"`
}

// PreviousProgress summarizes prior work for revisit states.
type PreviousProgress struct {
	CompletedTasks []string `json:"completedTasks"`
	TotalTasks     int      `json:"totalTasks"`
}

// DiscoveryContributor summarizes a contributor's discovery input.
type DiscoveryContributor struct {
	Name          string `json:"name"`
	Contributions string `json:"contributions"`
}

// ModeSelectionOutput provides mode selection options for discovery.
type ModeSelectionOutput struct {
	Required    bool   `json:"required"`
	Instruction string `json:"instruction"`
	Options     []struct {
		ID          string `json:"id"`
		Label       string `json:"label"`
		Description string `json:"description"`
	} `json:"options"`
}

// PremiseChallengeOutput asks the agent to challenge spec premises.
type PremiseChallengeOutput struct {
	Required    bool     `json:"required"`
	Instruction string   `json:"instruction"`
	Prompts     []string `json:"prompts"`
}

// RichDescriptionOutput signals a rich user description is available.
type RichDescriptionOutput struct {
	Provided    bool   `json:"provided"`
	Length      int    `json:"length"`
	Content     string `json:"content"`
	Instruction string `json:"instruction"`
}

// AlternativesOutput asks the agent to propose implementation alternatives.
type AlternativesOutput struct {
	Required    bool   `json:"required"`
	Instruction string `json:"instruction"`
	Format      struct {
		Fields []string `json:"fields"`
	} `json:"format"`
}

// DiscoveryOutput is the output for the DISCOVERY phase.
type DiscoveryOutput struct {
	Phase           string              `json:"phase"`
	Instruction     string              `json:"instruction"`
	Questions       []DiscoveryQuestion `json:"questions"`
	AnsweredCount   int                 `json:"answeredCount"`
	CurrentQuestion *int                `json:"currentQuestion,omitempty"`
	TotalQuestions  *int                `json:"totalQuestions,omitempty"`
	Context         ContextBlock        `json:"context"`
	Transition      struct {
		OnComplete string `json:"onComplete"`
	} `json:"transition"`
	Revisited            *bool                 `json:"revisited,omitempty"`
	RevisitReason        *string               `json:"revisitReason,omitempty"`
	PreviousProgress     *PreviousProgress     `json:"previousProgress,omitempty"`
	PreDiscoveryResearch *PreDiscoveryResearch `json:"preDiscoveryResearch,omitempty"`
	PlanContext          *PlanContext          `json:"planContext,omitempty"`
	CurrentUser          *struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"currentUser,omitempty"`
	PreviousContributors []DiscoveryContributor `json:"previousContributors,omitempty"`
	Notes                []struct {
		Text string `json:"text"`
		User string `json:"user"`
	} `json:"notes,omitempty"`
	ModeSelection    *ModeSelectionOutput    `json:"modeSelection,omitempty"`
	PremiseChallenge *PremiseChallengeOutput `json:"premiseChallenge,omitempty"`
	RichDescription  *RichDescriptionOutput  `json:"richDescription,omitempty"`
	AgreedPremises   []string                `json:"agreedPremises,omitempty"`
	RevisedPremises  []struct {
		Original string `json:"original"`
		Revision string `json:"revision"`
	} `json:"revisedPremises,omitempty"`
	FollowUpHints     []string         `json:"followUpHints,omitempty"`
	PendingFollowUps  []state.FollowUp `json:"pendingFollowUps,omitempty"`
	PreviousLearnings []string         `json:"previousLearnings,omitempty"`
}

// DiscoveryReviewAnswer is a single answer in the discovery review.
type DiscoveryReviewAnswer struct {
	QuestionID string `json:"questionId"`
	Question   string `json:"question"`
	Answer     string `json:"answer"`
}

// ReviewChecklistDimension is a single dimension in the review checklist.
type ReviewChecklistDimension struct {
	ID               string `json:"id"`
	Label            string `json:"label"`
	Prompt           string `json:"prompt"`
	EvidenceRequired bool   `json:"evidenceRequired"`
	IsRegistry       bool   `json:"isRegistry"`
	ConcernID        string `json:"concernId"`
}

// ReviewChecklist is the full review checklist for discovery refinement.
type ReviewChecklist struct {
	Dimensions          []ReviewChecklistDimension `json:"dimensions"`
	Instruction         string                     `json:"instruction"`
	RegistryInstruction *string                    `json:"registryInstruction,omitempty"`
}

// DiscoveryReviewOutput is the output for the DISCOVERY_REFINEMENT phase.
type DiscoveryReviewOutput struct {
	Phase       string                  `json:"phase"`
	Instruction string                  `json:"instruction"`
	Answers     []DiscoveryReviewAnswer `json:"answers"`
	Transition  struct {
		OnApprove string `json:"onApprove"`
		OnRevise  string `json:"onRevise"`
	} `json:"transition"`
	SplitProposal   *SplitProposal      `json:"splitProposal,omitempty"`
	SubPhase        *string             `json:"subPhase,omitempty"`
	Alternatives    *AlternativesOutput `json:"alternatives,omitempty"`
	ReviewChecklist *ReviewChecklist    `json:"reviewChecklist,omitempty"`
}

// ClassificationPrompt asks the user to classify the spec.
type ClassificationPrompt struct {
	Options []struct {
		ID    string `json:"id"`
		Label string `json:"label"`
	} `json:"options"`
	Instruction string `json:"instruction"`
}

// SelfReview is a self-review checklist for the spec draft.
type SelfReview struct {
	Required    bool     `json:"required"`
	Checks      []string `json:"checks"`
	Instruction string   `json:"instruction"`
}

// SpecDraftOutput is the output for the SPEC_PROPOSAL phase.
type SpecDraftOutput struct {
	Phase       string   `json:"phase"`
	Instruction string   `json:"instruction"`
	SpecPath    string   `json:"specPath"`
	EdgeCases   []string `json:"edgeCases,omitempty"`
	Transition  struct {
		OnApprove string `json:"onApprove"`
	} `json:"transition"`
	ClassificationRequired *bool                 `json:"classificationRequired,omitempty"`
	ClassificationPrompt   *ClassificationPrompt `json:"classificationPrompt,omitempty"`
	SelfReview             *SelfReview           `json:"selfReview,omitempty"`
	Saved                  *bool                 `json:"saved,omitempty"`
}

// SpecApprovedOutput is the output for the SPEC_APPROVED phase.
type SpecApprovedOutput struct {
	Phase       string `json:"phase"`
	Instruction string `json:"instruction"`
	SpecPath    string `json:"specPath"`
	Transition  struct {
		OnStart string `json:"onStart"`
	} `json:"transition"`
	Saved            *bool                   `json:"saved,omitempty"`
	TaskTDDSelection *TaskTDDSelectionOutput `json:"taskTDDSelection,omitempty"`
}

// TaskTDDSelectionOutput describes the per-task TDD selection sub-step shown
// after a spec is approved when spec-level TDD is enabled and the selection
// has not yet been made.
type TaskTDDSelectionOutput struct {
	Required    bool                    `json:"required"`
	Instruction string                  `json:"instruction"`
	Tasks       []TaskTDDSelectionEntry `json:"tasks"`
	Answers     TaskTDDSelectionAnswers `json:"answers"`
}

// TaskTDDSelectionEntry describes a single task shown in the selection UI.
// SuggestedTDD is a heuristic hint; the user's answer is authoritative.
type TaskTDDSelectionEntry struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	SuggestedTDD bool   `json:"suggestedTdd"`
}

// TaskTDDSelectionAnswers documents the exact strings the caller may submit
// via --answer to resolve the sub-step.
type TaskTDDSelectionAnswers struct {
	All    string `json:"all"`    // "tdd-all"
	None   string `json:"none"`   // "tdd-none"
	Custom string `json:"custom"` // example JSON, e.g. {"tddTasks":["task-1","task-3"]}
}

// AcceptanceCriterion is a single AC in the status report.
type AcceptanceCriterion struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// StatusReportRequest asks the agent to report AC status.
type StatusReportRequest struct {
	Criteria     []AcceptanceCriterion `json:"criteria"`
	ReportFormat struct {
		Completed string `json:"completed"`
		Remaining string `json:"remaining"`
		Blocked   string `json:"blocked"`
		NA        string `json:"na"`
		NewIssues string `json:"newIssues"`
	} `json:"reportFormat"`
}

// DebtCarryForward carries debt items from previous iterations.
type DebtCarryForward struct {
	FromIteration int              `json:"fromIteration"`
	Items         []state.DebtItem `json:"items"`
	Note          string           `json:"note"`
}

// PromotePrompt asks the user whether to promote a decision to a permanent rule.
type PromotePrompt struct {
	DecisionID string `json:"decisionId"`
	Question   string `json:"question"`
	Choice     string `json:"choice"`
	Prompt     string `json:"prompt"`
}

// TaskBlock describes the current task in execution.
type TaskBlock struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	TotalTasks     int      `json:"totalTasks"`
	CompletedTasks int      `json:"completedTasks"`
	Files          []string `json:"files,omitempty"`
}

// DesignChecklistDimension is a single dimension in the design checklist.
type DesignChecklistDimension struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// DesignChecklist is the design checklist for beautiful-product concern.
type DesignChecklist struct {
	Required    bool                       `json:"required"`
	Instruction string                     `json:"instruction"`
	Dimensions  []DesignChecklistDimension `json:"dimensions"`
}

// PreExecutionReview is shown before the first iteration.
type PreExecutionReview struct {
	Instruction string `json:"instruction"`
}

// TDDVerificationContext provides phase-specific verification instructions for the verifier.
type TDDVerificationContext struct {
	Phase       string `json:"phase"`
	Instruction string `json:"instruction"`
}

// TDDFailureReport provides structured failure details when TDD verification fails.
type TDDFailureReport struct {
	Reason             string   `json:"reason"`
	FailedACs          []string `json:"failedACs,omitempty"`
	UncoveredEdgeCases []string `json:"uncoveredEdgeCases,omitempty"`
	RetryCount         int      `json:"retryCount"`
	MaxRetries         int      `json:"maxRetries"`
	WillBlock          bool     `json:"willBlock"`
}

// RefactorInstructions carries verifier-produced refactor notes to the executor
// for the REFACTOR sub-phase of the TDD cycle.
type RefactorInstructions struct {
	Notes       []state.RefactorNote `json:"notes"`
	Instruction string               `json:"instruction"`
	Round       int                  `json:"round"`
	MaxRounds   int                  `json:"maxRounds"`
}

// ExecutionOutput is the output for the EXECUTING phase.
type ExecutionOutput struct {
	Phase       string       `json:"phase"`
	Instruction string       `json:"instruction"`
	Task        *TaskBlock   `json:"task,omitempty"`
	BatchTasks  []string     `json:"batchTasks,omitempty"`
	EdgeCases   []string     `json:"edgeCases,omitempty"`
	Context     ContextBlock `json:"context"`
	Transition  struct {
		OnComplete string `json:"onComplete"`
		OnBlocked  string `json:"onBlocked"`
		Iteration  int    `json:"iteration"`
	} `json:"transition"`
	ConcernTensions        []ConcernTension        `json:"concernTensions,omitempty"`
	RestartRecommended     *bool                   `json:"restartRecommended,omitempty"`
	RestartInstruction     *string                 `json:"restartInstruction,omitempty"`
	VerificationFailed     *bool                   `json:"verificationFailed,omitempty"`
	VerificationOutput     *string                 `json:"verificationOutput,omitempty"`
	StatusReportRequired   *bool                   `json:"statusReportRequired,omitempty"`
	StatusReport           *StatusReportRequest    `json:"statusReport,omitempty"`
	PreviousIterationDebt  *DebtCarryForward       `json:"previousIterationDebt,omitempty"`
	PromotePrompt          *PromotePrompt          `json:"promotePrompt,omitempty"`
	TaskRejected           *bool                   `json:"taskRejected,omitempty"`
	RejectionReason        *string                 `json:"rejectionReason,omitempty"`
	RejectionRemaining     []string                `json:"rejectionRemaining,omitempty"`
	DesignChecklist        *DesignChecklist        `json:"designChecklist,omitempty"`
	PreExecutionReview     *PreExecutionReview     `json:"preExecutionReview,omitempty"`
	TDDPhase               *string                 `json:"tddPhase,omitempty"`
	TDDVerificationContext *TDDVerificationContext `json:"tddVerificationContext,omitempty"`
	TDDFailureReport       *TDDFailureReport       `json:"tddFailureReport,omitempty"`
	RefactorInstructions   *RefactorInstructions   `json:"refactorInstructions,omitempty"`
}

// BlockedOutput is the output for the BLOCKED phase.
type BlockedOutput struct {
	Phase       string `json:"phase"`
	Instruction string `json:"instruction"`
	Reason      string `json:"reason"`
	Transition  struct {
		OnResolved string `json:"onResolved"`
	} `json:"transition"`
}

// ConcernInfo describes an available concern.
type ConcernInfo struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

// SpecSummary describes an existing spec.
type SpecSummary struct {
	Name      string  `json:"name"`
	Phase     string  `json:"phase"`
	Iteration int     `json:"iteration"`
	Detail    *string `json:"detail,omitempty"`
}

// IdleOutput is the output for the IDLE phase.
type IdleOutput struct {
	Phase             string        `json:"phase"`
	Instruction       string        `json:"instruction"`
	Welcome           string        `json:"welcome"`
	ExistingSpecs     []SpecSummary `json:"existingSpecs"`
	AvailableConcerns []ConcernInfo `json:"availableConcerns"`
	ActiveConcerns    []string      `json:"activeConcerns"`
	ActiveRulesCount  int           `json:"activeRulesCount"`
	BehavioralNote    *string       `json:"behavioralNote,omitempty"`
	Hint              *string       `json:"hint,omitempty"`
}

// CompletedOutput is the output for the COMPLETED phase.
type CompletedOutput struct {
	Phase   string `json:"phase"`
	Summary struct {
		Spec             *string                 `json:"spec"`
		Iterations       int                     `json:"iterations"`
		DecisionsCount   int                     `json:"decisionsCount"`
		CompletionReason *state.CompletionReason `json:"completionReason"`
		CompletionNote   *string                 `json:"completionNote"`
	} `json:"summary"`
	LearningPrompt *struct {
		Instruction string   `json:"instruction"`
		Examples    []string `json:"examples"`
	} `json:"learningPrompt,omitempty"`
	LearningsPending *bool `json:"learningsPending,omitempty"`
	StaleDiagrams    []struct {
		File   string `json:"file"`
		Line   int    `json:"line"`
		Reason string `json:"reason"`
	} `json:"staleDiagrams,omitempty"`
	StaleDiagramsBlocking *bool `json:"staleDiagramsBlocking,omitempty"`
}

// IdleContext provides extra context for the IDLE phase.
type IdleContext struct {
	ExistingSpecs []SpecSummary
	RulesCount    *int
}

// =============================================================================
// Command helpers
// =============================================================================

const defaultCmd = "tddmaster"

var commandPrefix = defaultCmd

// SetCommandPrefix sets the command prefix for output generation.
func SetCommandPrefix(prefix string) {
	commandPrefix = prefix
}

// c builds a full command: prefix + subcommand.
func c(sub string) string {
	return commandPrefix + " " + sub
}

// cs builds a spec-scoped command: prefix + "spec <name> <sub>".
func cs(sub string, specName *string) string {
	if specName == nil {
		return c(sub)
	}
	return c("spec " + *specName + " " + sub)
}

// =============================================================================
// Compiler constants
// =============================================================================

const staleSesssionMS = 5 * 60 * 1000 // 5 minutes in milliseconds

const gitReadonlyRule = "NEVER run git write commands (commit, add, push, checkout, stash, reset, merge, rebase, cherry-pick). Git is read-only for agents. The user controls git. You may read: git log, git diff, git status, git show, git blame."

// =============================================================================
// Behavioral Block Builder
// =============================================================================

func buildBehavioral(
	st state.StateFile,
	maxIterationsBeforeRestart int,
	allowGit bool,
	activeConcerns []state.ConcernDefinition,
	parsedSpec *spec.ParsedSpec,
	hints InteractionHints,
) BehavioralBlock {
	stale := st.Execution.Iteration >= maxIterationsBeforeRestart

	var askMethod string
	switch {
	case hints.AskUserStrategy == "tddmaster_block":
		askMethod = "Do not ask the user inline. Instead, use `tddmaster block \"question\"` at every decision point — the orchestrator will pause for human input."
	case !hints.HasAskUserTool:
		askMethod = "Present options as a numbered list at every decision point."
	default:
		askMethod = "Use AskUserQuestion for all decision points."
	}

	var mandatoryRules []string
	if !allowGit {
		mandatoryRules = append(mandatoryRules, gitReadonlyRule)
	}

	specName := st.Spec

	mandatoryRules = append(mandatoryRules,
		"Report progress honestly. Not done = 'not done'. Partial = 'partial: [works]/[doesn't]'. Untested = 'untested'. 4 of 6 = '4 of 6 done, 2 remaining'.",
		fmt.Sprintf("Never skip steps or infer decisions. %s Recommend first, then ask. One tddmaster call per interaction — never batch-submit or backfill.", askMethod),
		"Display `roadmap` before other content. Display `gate` prominently.",
		"NEVER suggest bypassing, skipping, or 'breaking out of' tddmaster. Discovery helps the user — it is not an obstacle. If scope changes: revise spec, reset and create new, or split.",
		"NEVER ask permission to run the next tddmaster command. After spec new → run next immediately. After answering questions → run next. After approve → run next. After task completion → run next. The workflow is sequential — each step has one next step. Just run it.",
		"Listen first: after spec creation, ask 'Tell me about this — share as much context as you have.' Wait for their response before mode selection. Rich context (>200 chars) → pre-fill discovery answers as STATED/INFERRED. Brief response → proceed normally.",
		"Discovery questions are adaptive. After each answer, generate 1-3 follow-up questions if the answer reveals ambiguity, risk, dependencies, or missing detail. Submit follow-ups via `tddmaster spec <name> followup <questionId> \"question\"`. Max 3 per question. Do NOT rush through discovery.",
		"Confidence scoring: every technical finding needs a confidence score (1-10). 9-10: verified (read code, ran test). 7-8: strong evidence. 5-6: reasonable inference. 3-4: guess. 1-2: speculation. State basis ('read X', 'inferred from Y'). If confidence < 5, prefix with '\u26A0 Unverified:'.",
	)

	var scopeItems []string
	if parsedSpec != nil {
		scopeItems = parsedSpec.OutOfScope
	}

	switch st.Phase {
	case state.PhaseIdle:
		optionRule := "Pass interactiveOptions DIRECTLY to AskUserQuestion options array (header max 12 chars). Use commandMap to resolve selections. For availableConcerns: AskUserQuestion with multiSelect:true, max 4 per question — split across questions if needed. Present ALL concerns."
		if hints.OptionPresentation != "tool" {
			optionRule = "Present interactiveOptions as numbered list. Use commandMap to resolve selections. Present ALL availableConcerns as numbered list for multiselect."
		}
		return BehavioralBlock{
			Rules: append([]string{
				"If the user described a feature/bug/task, create a spec immediately: `tddmaster spec new \"description\"` — name is auto-generated. Do NOT present menus or ask 'What would you like to do?' unless the conversation has no prior context.",
			}, append(mandatoryRules,
				optionRule,
				"Encourage full context: 'Tell me what you want to build — one-liner, detailed requirements, meeting notes, anything.' Slug is auto-generated. Pass full text to `tddmaster spec new \"...\"`.",
				"After spec new, listen first, then ask the user to choose a discovery mode: full, validate, technical-depth, ship-fast, or explore.",
				"Every task gets a spec. No exceptions. A one-liner fix, a config change, a 'simple' refactor — all get specs. The spec can be short but it must exist. 'Too simple for a spec' is the anti-pattern.",
			)...),
			Tone: "Welcoming. Present choices, then wait.",
		}

	case state.PhaseDiscovery:
		var questionMethod string
		switch {
		case hints.AskUserStrategy == "tddmaster_block":
			questionMethod = "Ask one question at a time via `tddmaster block \"question\"`. One question per interaction."
		case !hints.HasAskUserTool:
			questionMethod = "Ask one question at a time as text."
		default:
			questionMethod = "Ask each question via AskUserQuestion. One question per call."
		}

		dreamPrompts := GetDreamStatePrompts(activeConcerns)
		var dreamBase string
		if len(dreamPrompts) > 0 {
			dreamBase = "After answers, " + strings.Join(dreamPrompts, " Also: ")
		} else {
			dreamBase = "After answers, synthesize CURRENT STATE → THIS SPEC → 6-MONTH IDEAL vision."
		}

		return BehavioralBlock{
			ModeOverride: strPtr("plan mode. DO NOT create, edit, or write any files. DO NOT run state-modifying commands. MAY read files and run read-only commands (cat, ls, grep, git log, git diff)."),
			Rules: append(mandatoryRules,
				fmt.Sprintf("%s Never answer questions yourself. Never submit answers without user confirmation. Pre-fill suggested answers from detailed descriptions — user must confirm each. With a fully formed plan, keep discovery brief by confirming pre-filled answers one at a time, but MUST still run premise challenge and alternatives.", questionMethod),
				"DO NOT create, edit, or write any files.",
				"DO NOT run shell commands that modify state.",
				"You MAY read files and run read-only commands (cat, ls, grep, git log, git diff).",
				"Pre-discovery: (1) pre-discovery codebase scan — read README, CLAUDE.md, design docs, last 20 commits, TODOs, existing specs, directory structure. Present a brief audit summary. (2) If `preDiscoveryResearch.required`, web-search every `extractedTerms` entry — report versions, API changes, deprecations. (3) Ask discovery mode using the real options: A) Full discovery B) Validate my plan C) Technical depth D) Ship fast E) Explore scope. Adapt emphasis accordingly.",
				"Before starting discovery questions, challenge the user's initial spec description against codebase findings. Flag: hidden complexity, conflicts with existing code, scope mismatch, overlapping modules. Ask clarifying follow-ups.",
				"When asking questions, offer concrete options from codebase knowledge alongside the open-ended question (e.g., 'I see three scenarios: A)... B)... C)... D) Something else'). Push back on vague answers. Follow up on short answer with 'Can you be more specific?'",
				fmt.Sprintf("%s Then: (1) expansion opportunities as numbered proposals with effort (S/M/L/XL), risk, completeness delta — options: Add/Defer/Skip. (2) Architectural decisions that BLOCK implementation — present with options, RECOMMENDATION, completeness scores. Unresolved = risk flag. (3) Error/rescue map: codepath | failure mode | handling. Flag CRITICAL GAPS as decisions.", dreamBase),
				"Present DISCOVERY SUMMARY for confirmation: intent, scope, dream state, expansions, architectural decisions, error map. Ask for confirmation before generating spec. Keep discovery sequential: submit each confirmed answer as its own `tddmaster next --answer` call.",
			),
			Tone: "Curious interviewer with a stake in the answers. Comes PREPARED — read the codebase first. Challenge assumptions, think about architecture and failure modes.",
		}

	case state.PhaseDiscoveryRefinement:
		var confirmQ string
		switch {
		case hints.AskUserStrategy == "tddmaster_block":
			confirmQ = "Use `tddmaster block \"Are these answers correct, or would you like to revise any?\"` to pause for user review."
		case !hints.HasAskUserTool:
			confirmQ = "Ask the user: 'Are these answers correct, or would you like to revise any?' Present approval and revision as numbered options."
		default:
			confirmQ = "Use AskUserQuestion to ask: 'Are these answers correct, or would you like to revise any?'"
		}
		return BehavioralBlock{
			ModeOverride: strPtr("You are in plan mode. Do not create, edit, or write any files. Present the discovery answers to the user for review and confirmation."),
			Rules: append(mandatoryRules,
				"DO NOT create, edit, or write any files.",
				"Present ALL discovery answers to the user clearly, one by one.",
				confirmQ,
				"If the user approves, run the approve command.",
				"If the user wants to revise, collect their corrections and submit them.",
				"You MUST NOT approve on behalf of the user. The user must explicitly confirm.",
				"If tddmaster output contains a splitProposal, present it to the user with the exact options shown. Do NOT split or merge specs on your own. Do NOT recommend one option over the other unless the user asks for your opinion. The user decides.",
			),
			Tone: "Careful reviewer. The user must confirm every answer.",
		}

	case state.PhaseSpecProposal:
		delegations := st.Discovery.Delegations
		var delegationRules []string
		if len(delegations) > 0 {
			var lines []string
			pendingCount := 0
			answeredCount := 0
			for _, d := range delegations {
				status := "PENDING"
				if d.Status == "answered" {
					status = "ANSWERED ✓"
					answeredCount++
				} else {
					pendingCount++
				}
				lines = append(lines, fmt.Sprintf("- %s: delegated to %s — %s", d.QuestionID, d.DelegatedTo, status))
			}
			suffix := fmt.Sprintf("\n%d delegation(s) answered. Approve is allowed.", answeredCount)
			if pendingCount > 0 {
				suffix = fmt.Sprintf("\nApprove BLOCKED — %d pending delegation(s).", pendingCount)
			}
			delegationRules = append(delegationRules, "DELEGATION STATUS:\n"+strings.Join(lines, "\n")+suffix)
		}

		var classifyQ string
		switch {
		case hints.AskUserStrategy == "tddmaster_block":
			classifyQ = "When presenting classification options, present them as a numbered list. Use `tddmaster block` if you need the user to pick. Do NOT infer classification yourself."
		case !hints.HasAskUserTool:
			classifyQ = "When presenting classification options, present them as a numbered list with multiselect (user picks multiple numbers). Do NOT infer classification yourself."
		default:
			classifyQ = "When presenting classification options, use AskUserQuestion with multiSelect:true. Do NOT infer classification yourself."
		}

		return BehavioralBlock{
			ModeOverride: strPtr("plan mode. DO NOT create, edit, or write any files. DO NOT run state-modifying commands. MAY read files and run read-only commands."),
			Rules: append(append(mandatoryRules, delegationRules...),
				"DO NOT create, edit, or write any files.",
				"Read the spec and present a summary to the user.",
				"Flag any tasks that are too vague to execute.",
				"Flag any missing acceptance criteria.",
				"No placeholders in specs. If a task has 'TBD', 'TODO', 'to be determined', 'details to follow', or 'implement appropriate X' — fill in the detail or remove the task and add it as an open question.",
				"Ask the user if they want to refine before approving.",
				classifyQ,
				"When generating or refining tasks, include a 'Files:' hint listing likely files to create/modify. Format: 'Files: `path/to/file.ts`, `path/to/other.ts`'. Hints, not constraints — helps sub-agents load right context.",
				"If you identify issues in the spec (vague tasks, irrelevant sections, missing acceptance criteria), submit a refinement via: `"+
					cs("next --answer='{\"refinement\":\"task-1: Add upload endpoint, task-2: Add validation middleware, task-3: Write integration tests\"}'", specName)+
					"`. The spec will be updated and you can review again.",
			),
			Tone: "Thoughtful reviewer preparing to hand off to an implementer.",
		}

	case state.PhaseSpecApproved:
		var confirmQ string
		switch {
		case hints.AskUserStrategy == "tddmaster_block":
			confirmQ = "Before starting execution, show the spec summary and use `tddmaster block \"Confirm execution?\"` to pause for user go-ahead."
		case !hints.HasAskUserTool:
			confirmQ = "Before starting execution, show the spec summary to the user and ask for final confirmation. Present 'Start execution' and 'Not yet' as numbered options."
		default:
			confirmQ = "Before starting execution, show the spec summary to the user and ask for final confirmation via AskUserQuestion."
		}
		return BehavioralBlock{
			Rules: append(mandatoryRules,
				"The spec is approved but execution has not started.",
				"Do not start coding until the user triggers execution.",
				"If the user wants changes, they must reset and re-spec.",
				confirmQ,
			),
			Tone: "Patient. Wait for the go signal.",
		}

	case state.PhaseExecuting:
		reportCmd := cs("next --answer='{\"completed\":[...],\"remaining\":[...],\"blocked\":[]}'", specName)
		verifierScope := "Verifier scope: (1) AC verification with evidence. (2) Plan alignment — does implementation match task description? Flag deviations. (3) Code quality — follows existing patterns? Flag style breaks, missing error handling. Categorize findings: CRITICAL (blocks completion), IMPORTANT (should fix), SUGGESTION (nice to have)."

		var spawnInstruction, verifyInstruction string
		switch hints.SubAgentMethod {
		case "task":
			spawnInstruction = fmt.Sprintf("Spawn tddmaster-executor via Agent tool. Pass: task title, description, ACs, rules, scope constraints, concern reminders, file paths. Report via `%s`.", reportCmd)
			verifyInstruction = fmt.Sprintf("After executor completes, spawn tddmaster-verifier with changed files + ACs + test commands. Never trust executor self-report alone. %s", verifierScope)
		case "spawn":
			spawnInstruction = fmt.Sprintf("Use spawn_agent for tddmaster-executor. Pass: task, ACs, rules, scope, file paths. Use wait_agent to collect. Report via `%s`.", reportCmd)
			verifyInstruction = fmt.Sprintf("After executor completes, spawn tddmaster-verifier with changed files + ACs + test commands. %s", verifierScope)
		case "fleet":
			spawnInstruction = fmt.Sprintf("Use /fleet for parallel executors. Pass each: task, ACs, rules, scope, file paths. Report via `%s`.", reportCmd)
			verifyInstruction = fmt.Sprintf("After fleet completes, run verification pass. %s", verifierScope)
		case "delegation":
			spawnInstruction = fmt.Sprintf("Use agent delegation for tddmaster-executor. Pass: task, ACs, rules, scope, file paths. Report via `%s`.", reportCmd)
			verifyInstruction = fmt.Sprintf("After executor completes, delegate to tddmaster-verifier with changed files + ACs + test commands. %s", verifierScope)
		default:
			spawnInstruction = fmt.Sprintf("Execute tasks sequentially yourself. Verify (type-check + tests) after each. Report via `%s`.", reportCmd)
			verifyInstruction = ""
		}

		hasSubAgents := hints.SubAgentMethod != "none"
		var orchestratorRule string
		if hasSubAgents {
			orchestratorRule = fmt.Sprintf("You are the orchestrator. NEVER edit files directly — delegate ALL edits to tddmaster-executor. %s On sub-agent failure, fall back to direct execution and note it in status.", spawnInstruction)
		} else {
			orchestratorRule = spawnInstruction
		}

		base := append(mandatoryRules,
			orchestratorRule,
		)
		if verifyInstruction != "" {
			base = append(base, verifyInstruction)
		}
		base = append(base,
			"Do not explore beyond current task. Do not refactor outside scope. Do not add features, tests, or docs not in the spec. timebox context reads — the deliverable is working code.",
		)
		if hasSubAgents {
			base = append(base, "Show a dispatch table: | Step | Agent | Files | Tasks | Est. |. Separate executor for implementation vs tests. Batch tightly-coupled files; parallelize independent modules.")
		}
		base = append(base,
			"Edit discipline: (1) Re-read file before editing. (2) Re-read after to confirm. (3) Files >500 LOC: read in chunks. (4) Run type-check + lint after edits — never mark AC passed if type-check fails. (5) If search returns few results, re-run narrower — assume truncation.",
			fmt.Sprintf("On recurring patterns or corrections, ask: 'Permanent rule or just this task?' If permanent: `%s`. Never write to `.tddmaster/rules/` directly.", c("rule add \"<description>\"")),
			"Before structural refactors on files >300 LOC, remove dead code first. Do NOT suggest pausing or stopping mid-spec — execute to completion. The user decides when to stop.",
			"RATIONALIZATION ALERT: Never use 'should work now', 'looks correct', 'I'm confident', 'seems right', 'probably passes'. Run the command, read the output, report what happened. Evidence, not belief.",
			"TDD: (1) Write test. (2) Run it — MUST fail. If it passes before implementation, the test is wrong. (3) Implement. (4) Run test — must pass. Skipping step 2 means the test is unverified.",
			"If execution output includes `edgeCases`, pass them explicitly to the test-writer. Those cases are mandatory red-phase coverage before implementation.",
			"Parallel vs serial sub-agents: PARALLEL when tasks touch different files with no shared state. SERIAL when tasks modify same files or depend on each other. When unsure, default to serial.",
		)
		if hasSubAgents {
			base = append(base, "VERIFICATION REQUIRED: After EVERY task completion, you MUST spawn tddmaster-verifier before reporting done. If you skip verification and self-report, the status report will flag it. No exceptions — 'it looks correct' is not verification.")
		} else {
			base = append(base, "VERIFICATION REQUIRED: After EVERY task completion, run type-check + tests before reporting done. Evidence of passing tests must be included in the status report.")
		}

		if st.Execution.LastVerification != nil && !st.Execution.LastVerification.Passed {
			base = append(base, "Tests are failing. Fix ONLY the failing tests. Do not refactor passing code.")
		}

		behavioral := BehavioralBlock{
			Rules:      base,
			Tone:       "Direct. Orchestrate immediately — spawn sub-agents.",
			OutOfScope: scopeItems,
		}
		if len(scopeItems) == 0 {
			behavioral.OutOfScope = nil
		}

		if stale {
			urgency := fmt.Sprintf("%d+ iterations — context degrading. Finish current task, recommend fresh session.", st.Execution.Iteration)
			behavioral.Urgency = &urgency
		}

		return behavioral

	case state.PhaseBlocked:
		return BehavioralBlock{
			Rules: append(mandatoryRules,
				"Present the decision to the user exactly as described.",
				"Do not suggest a preferred option unless the user asks for your opinion.",
				"After the user decides, relay the answer immediately. Do not elaborate.",
			),
			Tone: "Brief. The user is making a decision, not having a discussion.",
		}

	case state.PhaseCompleted:
		return BehavioralBlock{
			Rules: append(mandatoryRules,
				"Report the completion summary. Do not start new work.",
				"If the user wants to continue, they start a new spec.",
			),
			Tone: "Concise. Celebrate briefly, then stop.",
		}

	default:
		return BehavioralBlock{
			Rules: append(mandatoryRules,
				fmt.Sprintf("Run `%s` to get your instructions.", cs("next", specName)),
				"Do not take action without tddmaster guidance.",
			),
			Tone: "Neutral. Waiting for direction.",
		}
	}
}

// =============================================================================
// Meta Block Builder
// =============================================================================

func buildEnforcement(hints InteractionHints) *EnforcementInfo {
	if hints.HasSubAgentDelegation {
		return &EnforcementInfo{
			Level: "enforced",
			Capabilities: []string{
				"PreToolUse file edit gate",
				"Git write guard",
				"Stop iteration tracking",
				"PostToolUse file logging",
				"Sub-agent delegation",
			},
		}
	}
	return &EnforcementInfo{
		Level:        "behavioral",
		Capabilities: []string{"Behavioral rules only"},
		Gaps: []string{
			"File edits not blocked in non-execution phases",
			"Git write commands not blocked",
			"No iteration tracking",
			"No file change logging",
			"No sub-agent delegation available",
		},
	}
}

func buildMeta(st state.StateFile, activeConcerns []state.ConcernDefinition, hints InteractionHints) MetaBlock {
	specName := st.Spec
	var resumeHint string

	switch st.Phase {
	case state.PhaseIdle:
		resumeHint = fmt.Sprintf("No active spec. Start one with: `%s`", c("spec new --name=<slug> \"description\""))
	case state.PhaseDiscovery:
		name := ""
		if st.Spec != nil {
			name = *st.Spec
		}
		resumeHint = fmt.Sprintf("Discovery in progress for \"%s\". %d questions answered so far.", name, len(st.Discovery.Answers))
	case state.PhaseDiscoveryRefinement:
		resumeHint = fmt.Sprintf("Discovery answers ready for review. %d answers collected. Waiting for user confirmation.", len(st.Discovery.Answers))
	case state.PhaseSpecProposal:
		path := ""
		if st.SpecState.Path != nil {
			path = *st.SpecState.Path
		}
		resumeHint = fmt.Sprintf("Spec draft ready for review at %s. Waiting for approval.", path)
	case state.PhaseSpecApproved:
		name := ""
		if st.Spec != nil {
			name = *st.Spec
		}
		resumeHint = fmt.Sprintf("Spec \"%s\" is approved. Waiting to start execution.", name)
	case state.PhaseExecuting:
		name := ""
		if st.Spec != nil {
			name = *st.Spec
		}
		if st.Execution.LastProgress != nil {
			resumeHint = fmt.Sprintf("Executing \"%s\", iteration %d. Last progress: %s. Continue with the current task.", name, st.Execution.Iteration, *st.Execution.LastProgress)
		} else {
			resumeHint = fmt.Sprintf("Executing \"%s\", iteration %d. Start the first task.", name, st.Execution.Iteration)
		}
	case state.PhaseBlocked:
		progress := "Unknown"
		if st.Execution.LastProgress != nil {
			progress = *st.Execution.LastProgress
		}
		resumeHint = fmt.Sprintf("Execution blocked: %s. Ask the user to resolve.", progress)
	case state.PhaseCompleted:
		name := ""
		if st.Spec != nil {
			name = *st.Spec
		}
		resumeHint = fmt.Sprintf("Spec \"%s\" completed in %d iterations.", name, st.Execution.Iteration)
	default:
		resumeHint = fmt.Sprintf("Run `%s` to get started.", cs("next", specName))
	}

	concernIDs := make([]string, len(activeConcerns))
	for i, c := range activeConcerns {
		concernIDs[i] = c.ID
	}

	enforcement := buildEnforcement(hints)

	return MetaBlock{
		Protocol:       fmt.Sprintf("Run `%s` to submit results and advance", cs("next --answer=\"...\"", specName)),
		Spec:           st.Spec,
		Branch:         st.Branch,
		Iteration:      st.Execution.Iteration,
		LastProgress:   st.Execution.LastProgress,
		ActiveConcerns: concernIDs,
		ResumeHint:     resumeHint,
		Enforcement:    enforcement,
	}
}

// =============================================================================
// Roadmap Builder
// =============================================================================

type roadmapPhase struct {
	key   string
	label string
}

var roadmapPhases = []roadmapPhase{
	{"IDLE", "IDLE"},
	{"DISCOVERY", "DISCOVERY"},
	{"DISCOVERY_REFINEMENT", "REFINEMENT"},
	{"SPEC_PROPOSAL", "PROPOSAL"},
	{"SPEC_APPROVED", "APPROVED"},
	{"EXECUTING", "EXECUTING"},
	{"COMPLETED", "COMPLETED"},
	{"IDLE_END", "IDLE"},
}

func buildRoadmap(phase state.Phase) string {
	if phase == state.PhaseBlocked {
		parts := make([]string, len(roadmapPhases))
		for i, p := range roadmapPhases {
			if p.key == "EXECUTING" {
				parts[i] = "[ EXECUTING (BLOCKED) ]"
			} else {
				parts[i] = p.label
			}
		}
		return strings.Join(parts, " → ")
	}

	parts := make([]string, len(roadmapPhases))
	for i, p := range roadmapPhases {
		if p.key == "IDLE" && phase == state.PhaseIdle {
			parts[i] = "[ IDLE ]"
		} else if state.Phase(p.key) == phase {
			parts[i] = "[ " + p.label + " ]"
		} else {
			parts[i] = p.label
		}
	}
	return strings.Join(parts, " → ")
}

// =============================================================================
// Gate Builder
// =============================================================================

func buildGate(st state.StateFile, parsedSpec *spec.ParsedSpec) *GateInfo {
	switch st.Phase {
	case state.PhaseDiscoveryRefinement:
		return &GateInfo{
			Message: fmt.Sprintf("%d/6 answers collected.", len(st.Discovery.Answers)),
			Action:  "Type APPROVE to generate spec, or REVISE to correct answers.",
			Phase:   "DISCOVERY_REFINEMENT",
		}
	case state.PhaseSpecApproved:
		taskCount := 0
		if parsedSpec != nil {
			taskCount = len(parsedSpec.Tasks)
		}
		return &GateInfo{
			Message: fmt.Sprintf("Spec approved. %d tasks ready.", taskCount),
			Action:  "Type START to begin execution.",
			Phase:   "SPEC_APPROVED",
		}
	}
	return nil
}

// =============================================================================
// Protocol Guide Builder
// =============================================================================

func buildProtocolGuide(st state.StateFile) *ProtocolGuide {
	specName := st.Spec

	if st.LastCalledAt == nil {
		return &ProtocolGuide{
			What:         "tddmaster orchestrates your work: IDLE → DISCOVERY → DISCOVERY_REFINEMENT → SPEC_PROPOSAL → SPEC_APPROVED → EXECUTING → DONE → IDLE",
			How:          fmt.Sprintf("Run `%s` for instructions. Submit results with `%s`. Never make architectural decisions without asking.", cs("next", specName), cs("next --answer=\"...\"", specName)),
			CurrentPhase: string(st.Phase),
		}
	}

	// Parse last called timestamp
	lastCalledStr := *st.LastCalledAt
	lastCalled, err := time.Parse(time.RFC3339, lastCalledStr)
	if err != nil {
		// Try other formats
		lastCalled, err = time.Parse("2006-01-02T15:04:05.000Z", lastCalledStr)
	}

	if err == nil {
		elapsed := time.Since(lastCalled).Milliseconds()
		if elapsed > staleSesssionMS {
			return &ProtocolGuide{
				What:         "tddmaster orchestrates your work: IDLE → DISCOVERY → DISCOVERY_REFINEMENT → SPEC_PROPOSAL → SPEC_APPROVED → EXECUTING → DONE → IDLE",
				How:          fmt.Sprintf("Run `%s` for instructions. Submit results with `%s`. Never make architectural decisions without asking.", cs("next", specName), cs("next --answer=\"...\"", specName)),
				CurrentPhase: string(st.Phase),
			}
		}
	}

	return nil
}

// =============================================================================
// Interactive Options Builder
// =============================================================================

func buildInteractiveOptions(st state.StateFile, activeConcerns []state.ConcernDefinition, idleContext *IdleContext, config *state.NosManifest) []internalOption {
	specName := st.Spec

	switch st.Phase {
	case state.PhaseIdle:
		var opts []internalOption
		var specs []SpecSummary
		if idleContext != nil {
			specs = idleContext.ExistingSpecs
		}

		// Continuable specs (not COMPLETED)
		var continuable []SpecSummary
		for _, s := range specs {
			if s.Phase != "COMPLETED" {
				continuable = append(continuable, s)
			}
		}

		if len(activeConcerns) == 0 {
			opts = append(opts, internalOption{
				Label:       "Add concerns (Recommended)",
				Description: "Shape how discovery and specs work by adding project concerns",
				Command:     c("concern add <id> [<id2> ...]"),
			})
		}

		opts = append(opts, internalOption{
			Label:       "Start a new spec",
			Description: "Tell me what you want to build — a one-liner, detailed requirements, meeting notes, anything",
			Command:     c("spec new \"description\""),
		})

		// Add continuable specs (max 2)
		for i, sp := range continuable {
			if i >= 2 {
				break
			}
			detail := fmt.Sprintf("Iteration %d", sp.Iteration)
			if sp.Detail != nil {
				detail = *sp.Detail
			}
			opts = append(opts, internalOption{
				Label:       fmt.Sprintf("Continue: %s (%s)", sp.Name, sp.Phase),
				Description: detail,
				Command:     cs("next", &sp.Name),
			})
		}

		if len(activeConcerns) > 0 {
			ids := make([]string, len(activeConcerns))
			for i, cc := range activeConcerns {
				ids[i] = cc.ID
			}
			opts = append(opts, internalOption{
				Label:       "Edit concerns",
				Description: "Currently: " + strings.Join(ids, ", "),
				Command:     c("concern list"),
			})
		}

		if len(opts) > 4 {
			opts = opts[:4]
		}
		return opts

	case state.PhaseDiscoveryRefinement:
		if st.Discovery.Approved {
			return []internalOption{
				{
					Label:       "Keep as one spec",
					Description: "All work in a single spec",
					Command:     cs("next --answer=\"keep\"", specName),
				},
				{
					Label:       "Split into separate specs",
					Description: "Create one spec per independent area",
					Command:     cs("next --answer=\"split\"", specName),
				},
			}
		}
		return []internalOption{
			{
				Label:       "Approve all answers",
				Description: "Answers look correct — generate the spec",
				Command:     cs("next --answer=\"approve\"", specName),
			},
			{
				Label:       "Revise answers",
				Description: "Correct one or more discovery answers",
				Command:     cs("next --answer='{\"revise\":{...}}'", specName),
			},
		}

	case state.PhaseSpecProposal:
		return []internalOption{
			{
				Label:       "Approve spec",
				Description: "Review looks good — approve and move to execution",
				Command:     cs("approve", specName),
			},
			{
				Label:       "Refine spec",
				Description: "Submit refinements to improve tasks or sections",
				Command:     cs("next --answer='{\"refinement\":\"...\"}'", specName),
			},
			{
				Label:       "Save for later",
				Description: "Keep the draft as-is. Others can review, add ACs, notes, or tasks. Come back anytime to approve.",
				Command:     cs("next --answer=\"save\"", specName),
			},
			{
				Label:       "Start over",
				Description: "Reset the spec and start fresh",
				Command:     cs("reset", specName),
			},
		}

	case state.PhaseSpecApproved:
		needsTDDSelection := config != nil && config.IsTDDEnabled() &&
			(st.TaskTDDSelected == nil || !*st.TaskTDDSelected)
		if needsTDDSelection {
			return []internalOption{
				{
					Label:       "TDD for all tasks",
					Description: "Every task follows red → green → refactor (current behavior)",
					Command:     cs("next --answer=\"tdd-all\"", specName),
				},
				{
					Label:       "No TDD",
					Description: "Skip red/green/refactor for every task — run executor → verifier only",
					Command:     cs("next --answer=\"tdd-none\"", specName),
				},
				{
					Label:       "Pick per task",
					Description: "Use AskUserQuestion with multiSelect over specApprovedData.taskTDDSelection.tasks, then submit {\"tddTasks\":[...IDs...]}",
					Command:     cs("next --answer='{\"tddTasks\":[\"task-1\",\"task-3\"]}'", specName),
				},
				{
					Label:       "Save for later",
					Description: "Spec is approved but don't start execution yet. Others can still add ACs or notes.",
					Command:     cs("next --answer=\"save\"", specName),
				},
			}
		}
		return []internalOption{
			{
				Label:       "Start execution",
				Description: "Begin implementing the tasks",
				Command:     cs("next --answer=\"start\"", specName),
			},
			{
				Label:       "Save for later",
				Description: "Spec is approved but don't start execution yet. Others can still add ACs or notes.",
				Command:     cs("next --answer=\"save\"", specName),
			},
		}

	case state.PhaseExecuting:
		return nil // Agent should be working

	case state.PhaseBlocked:
		return []internalOption{
			{
				Label:       "Resolve block",
				Description: "Provide a resolution to unblock execution",
				Command:     cs("next --answer=\"resolution\"", specName),
			},
			{
				Label:       "Reset spec",
				Description: "Abandon this spec and start over",
				Command:     cs("reset", specName),
			},
		}

	case state.PhaseCompleted:
		return []internalOption{
			{
				Label:       "New spec",
				Description: "Start a new feature spec",
				Command:     c("spec new --name=<slug> \"description\""),
			},
			{
				Label:       "Reopen spec",
				Description: "Reopen this spec for revision",
				Command:     cs("reopen", specName),
			},
			{
				Label:       "Check status",
				Description: "Review completed spec summary",
				Command:     c("status"),
			},
		}
	}

	return nil
}

// =============================================================================
// Phase Compilers
// =============================================================================

const welcome = "tddmaster is a state-machine orchestrator that acts as a scrum master for both you and your agent — keeping work focused, decisions in your hands, and tokens efficient."

func compileIdle(
	activeConcerns []state.ConcernDefinition,
	allConcernDefs []state.ConcernDefinition,
	rulesCount int,
	idleContext *IdleContext,
) IdleOutput {
	availableConcerns := make([]ConcernInfo, len(allConcernDefs))
	for i, cc := range allConcernDefs {
		availableConcerns[i] = ConcernInfo{ID: cc.ID, Description: cc.Description}
	}

	activeIDs := make([]string, len(activeConcerns))
	for i, cc := range activeConcerns {
		activeIDs[i] = cc.ID
	}

	activeRulesCount := rulesCount
	existingSpecs := []SpecSummary{}
	if idleContext != nil {
		existingSpecs = idleContext.ExistingSpecs
		if idleContext.RulesCount != nil {
			activeRulesCount = *idleContext.RulesCount
		}
	}

	behavioralNote := strPtr("These options are fallbacks. If the user already described what they want, act on it directly without presenting these options.")

	var hint *string
	if len(activeConcerns) == 0 {
		hint = strPtr("No concerns active. Consider adding concerns first — they shape discovery questions and specs.")
	}

	return IdleOutput{
		Phase:             "IDLE",
		Instruction:       "No active spec. If the user described what they want, run `tddmaster spec new \"description\"` immediately — name is auto-generated. Present ALL available concerns (split across multiple calls if needed).",
		Welcome:           welcome,
		ExistingSpecs:     existingSpecs,
		AvailableConcerns: availableConcerns,
		ActiveConcerns:    activeIDs,
		ActiveRulesCount:  activeRulesCount,
		BehavioralNote:    behavioralNote,
		Hint:              hint,
	}
}

// =============================================================================
// Pre-discovery Research
// =============================================================================

var versionTermPattern = regexp.MustCompile(`(?i)\b(Node\.?js|Deno|Bun|Go|Rust|Python|Ruby|Java|Kotlin|Swift|PHP|React|Vue|Angular|Svelte|Next\.?js|Nuxt|Remix|Astro|SolidJS|Qwik|TypeScript|Webpack|Vite|esbuild|Rollup|Terraform|Docker|Kubernetes|PostgreSQL|MySQL|Redis|MongoDB|SQLite|Prisma|Drizzle|gRPC|GraphQL|tRPC)\s+v?(\d+(?:\.\d+)?(?:\.\d+)?\+?)\b`)

func extractVersionTerms(description string) []string {
	if description == "" {
		return nil
	}

	matches := versionTermPattern.FindAllStringSubmatch(description, -1)
	var terms []string
	for _, m := range matches {
		if len(m) >= 3 {
			terms = append(terms, m[1]+" "+m[2])
		}
	}
	return terms
}

func buildPreDiscoveryResearch(description string) *PreDiscoveryResearch {
	terms := extractVersionTerms(description)
	if len(terms) == 0 {
		return nil
	}

	return &PreDiscoveryResearch{
		Required:       true,
		Instruction:    "Before asking discovery questions, research the current state of all platforms, runtimes, libraries, and APIs mentioned in the spec description. Use web search and Context7 MCP if available. Report findings as a pre-discovery brief to the user. Do NOT assume your training data is current — versions change, APIs get added, features get deprecated.",
		ExtractedTerms: terms,
	}
}

const maxPlanSize = 50 * 1024

func buildPlanContext(planPath string) *PlanContext {
	if planPath == "" {
		return nil
	}

	data, err := os.ReadFile(planPath)
	if err != nil {
		return nil
	}

	if len(data) > maxPlanSize {
		return nil
	}

	content := string(data)
	return &PlanContext{
		Provided:    true,
		Content:     content,
		Instruction: "A plan document was provided. Read it carefully, extract relevant information for each discovery question, and present pre-filled answers for user review. Do NOT skip any question — present your extraction and let the user confirm, correct, or expand. IMPORTANT: When extracting answers from the plan, mark each extraction as [STATED] (directly written in the plan) or [INFERRED] (your interpretation). Present extractions individually for confirmation.",
	}
}

func generateFollowUpHints(answer string) []string {
	var hints []string
	lower := strings.ToLower(answer)

	techPatterns := []string{
		"websocket", "graphql", "grpc", "redis", "postgres", "mongodb",
		"kafka", "rabbitmq", "docker", "kubernetes", "lambda", "s3",
	}
	for _, tech := range techPatterns {
		if strings.Contains(lower, tech) {
			hints = append(hints, fmt.Sprintf("Answer mentions %s — consider: error handling, versioning, fallback strategy", tech))
		}
	}

	if strings.Contains(lower, "should work") || strings.Contains(lower, "standard approach") ||
		strings.Contains(lower, "probably") || strings.Contains(lower, "i think") ||
		strings.Contains(lower, "not sure") {
		hints = append(hints, "Answer is vague — ask for specifics")
	}

	if strings.Contains(lower, "and also") || strings.Contains(lower, "we might") ||
		strings.Contains(lower, "could also") || strings.Contains(lower, "maybe we should") {
		hints = append(hints, "Scope expansion signal — clarify if in scope or deferred")
	}

	if strings.Contains(lower, "tricky") || strings.Contains(lower, "complicated") ||
		strings.Contains(lower, "risky") || strings.Contains(lower, "not sure about") {
		hints = append(hints, "Risk signal — dig deeper into what makes it risky")
	}

	if strings.Contains(lower, "depends on") || strings.Contains(lower, "after") ||
		strings.Contains(lower, "blocked by") || strings.Contains(lower, "waiting for") {
		hints = append(hints, "Dependency detected — clarify what happens if dependency isn't ready")
	}

	if strings.Contains(lower, "real-time") || strings.Contains(lower, "scalab") ||
		strings.Contains(lower, "performance") || strings.Contains(lower, "latency") ||
		strings.Contains(lower, "concurrent") {
		hints = append(hints, "Performance/scale mention — ask about limits, degradation, monitoring")
	}

	return hints
}

func getModeRules(mode state.DiscoveryMode) []string {
	switch mode {
	case state.DiscoveryModeFull:
		return []string{
			"Ask each discovery question as written. Push for specific, concrete answers.",
			"If the answer is vague, ask follow-up questions before accepting.",
		}
	case state.DiscoveryModeValidate:
		return []string{
			"The user has a plan. Your job is to challenge it, not explore it.",
			"For each question, identify assumptions and ask: 'What would prove this wrong?'",
			"If the description already answers a question, present your understanding and ask to confirm.",
			"When pre-filling answers from a rich description, plan, or prior discussion, DISTINGUISH between what the user EXPLICITLY STATED and what you INFERRED. Format each pre-filled item as: '[STATED] GPU skinning in all 3 renderers — you said this during technical discussion' or '[INFERRED] tangent space is 10-star scope — I assumed this based on complexity'. The user confirms stated items and corrects inferred items.",
			"Present pre-filled answers ONE ITEM AT A TIME for confirmation, not as a completed block. The user's job is to correct your inferences, not rubber-stamp your summary. If you pre-fill 5 items and 2 are wrong, the user must be able to catch them individually.",
		}
	case state.DiscoveryModeTechnicalDepth:
		return []string{
			"Focus on architecture, data flow, performance, and integration points.",
			"Before each question, scan the codebase for related implementations.",
			"Ask: 'How does this interact with [existing system]?' for each integration point.",
		}
	case state.DiscoveryModeShipFast:
		return []string{
			"Focus on minimum viable scope.",
			"For each question, also ask: 'What can we defer to a follow-up?'",
			"Push for the smallest version that delivers value.",
		}
	case state.DiscoveryModeExplore:
		return []string{
			"Think bigger. What's the 10x version?",
			"For each question, ask about adjacent opportunities.",
			"Suggest possibilities the user might not have considered.",
		}
	}
	return nil
}

func computeContributors(
	answers []state.DiscoveryAnswer,
	currentUser *struct{ Name, Email string },
) []DiscoveryContributor {
	userMap := make(map[string]int)
	for _, a := range answers {
		_ = a // DiscoveryAnswer doesn't have user field — use "Unknown User"
		name := "Unknown User"
		if currentUser != nil && name == currentUser.Name {
			continue
		}
		userMap[name]++
	}

	var contributors []DiscoveryContributor
	for name, count := range userMap {
		suffix := "answer"
		if count > 1 {
			suffix = "answers"
		}
		contributors = append(contributors, DiscoveryContributor{
			Name:          name,
			Contributions: fmt.Sprintf("%d %s", count, suffix),
		})
	}
	return contributors
}

func buildDiscoveryQuestion(question QuestionWithExtras, prefills []state.DiscoveryPrefillQuestion) DiscoveryQuestion {
	extras := make([]string, len(question.Extras))
	for i, e := range question.Extras {
		extras[i] = e.Text
	}

	out := DiscoveryQuestion{
		ID:       question.ID,
		Text:     question.Text,
		Concerns: question.Concerns,
		Extras:   extras,
	}
	if items := state.GetPrefillsForQuestion(prefills, question.ID); len(items) > 0 {
		out.Prefills = items
	}
	return out
}

func selectCurrentDiscoveryQuestion(
	questions []QuestionWithExtras,
	answers []state.DiscoveryAnswer,
	currentIdx int,
) (*QuestionWithExtras, int) {
	answeredIDs := make(map[string]bool, len(answers))
	for _, answer := range answers {
		answeredIDs[answer.QuestionID] = true
	}

	if currentIdx >= 0 && currentIdx < len(questions) {
		candidate := questions[currentIdx]
		if !answeredIDs[candidate.ID] {
			return &candidate, currentIdx
		}
	}

	for i := range questions {
		if !answeredIDs[questions[i].ID] {
			return &questions[i], i
		}
	}

	return nil, len(questions)
}

func compileDiscovery(
	st state.StateFile,
	activeConcerns []state.ConcernDefinition,
	rules []string,
	currentUser *struct{ Name, Email string },
) DiscoveryOutput {
	specName := st.Spec
	allQuestions := GetQuestionsWithExtras(activeConcerns)
	answeredCount := len(st.Discovery.Answers)
	allAnswered := IsDiscoveryComplete(st.Discovery.Answers)
	isAgent := st.Discovery.Audience == "agent"

	// Build optional multi-user context
	_ = computeContributors(st.Discovery.Answers, currentUser) // contributors computed but not used until rich attribution
	var specNotes []struct {
		Text string `json:"text"`
		User string `json:"user"`
	}
	for _, n := range st.SpecNotes {
		if !strings.HasPrefix(n.Text, "[TASK] ") {
			specNotes = append(specNotes, struct {
				Text string `json:"text"`
				User string `json:"user"`
			}{Text: n.Text, User: n.User})
		}
	}

	hasUserContext := st.Discovery.UserContext != nil && len(*st.Discovery.UserContext) > 0
	hasDescription := st.SpecDescription != nil && len(*st.SpecDescription) > 0
	hasPlan := st.Discovery.PlanPath != nil
	mode := st.Discovery.Mode

	reminders := GetReminders(activeConcerns, nil)

	// Listen-first step
	if mode == nil && !hasUserContext && answeredCount == 0 && !hasPlan && hasDescription {
		out := DiscoveryOutput{
			Phase:         "DISCOVERY",
			Instruction:   "The user just created this spec. Before starting discovery, ask them to share whatever context they have — requirements, notes, tasks, or just a brief description. Say: 'Tell me about this — share as much context as you have.' Listen first, then proceed.",
			Questions:     []DiscoveryQuestion{},
			AnsweredCount: 0,
			Context: ContextBlock{
				Rules:            rules,
				ConcernReminders: reminders,
			},
			Transition: struct {
				OnComplete string `json:"onComplete"`
			}{OnComplete: cs("next --answer=\"<user context or just start>\"", specName)},
		}
		if currentUser != nil {
			out.CurrentUser = &struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			}{Name: currentUser.Name, Email: currentUser.Email}
		}
		if len(specNotes) > 0 {
			out.Notes = specNotes
		}
		return out
	}

	// Mode selection step
	if mode == nil && hasDescription && answeredCount == 0 && !hasPlan {
		type modeOption struct {
			ID          string `json:"id"`
			Label       string `json:"label"`
			Description string `json:"description"`
		}
		ms := &ModeSelectionOutput{
			Required:    true,
			Instruction: "Select the discovery mode.",
		}
		// Manually assign options
		type opt = struct {
			ID          string `json:"id"`
			Label       string `json:"label"`
			Description string `json:"description"`
		}
		ms.Options = []opt{
			{"full", "Full discovery", "Standard 6 questions with all concern extras. Default for new features."},
			{"validate", "Validate my plan", "I already know what I want — challenge my assumptions, find gaps."},
			{"technical-depth", "Technical depth", "Focus on architecture, data flow, performance, integration points."},
			{"ship-fast", "Ship fast", "Minimum viable scope. What can we defer? What's the MVP?"},
			{"explore", "Explore scope", "Think bigger. 10x version? Adjacent opportunities? What are we missing?"},
		}

		out := DiscoveryOutput{
			Phase:         "DISCOVERY",
			Instruction:   "Before starting discovery, select the discovery mode that best fits this spec.",
			Questions:     []DiscoveryQuestion{},
			AnsweredCount: 0,
			Context: ContextBlock{
				Rules:            rules,
				ConcernReminders: reminders,
			},
			Transition: struct {
				OnComplete string `json:"onComplete"`
			}{OnComplete: cs("next --answer=\"<mode>\"", specName)},
			ModeSelection: ms,
		}
		if currentUser != nil {
			out.CurrentUser = &struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			}{Name: currentUser.Name, Email: currentUser.Email}
		}
		return out
	}

	// Premise challenge step
	premisesCompleted := st.Discovery.PremisesCompleted != nil && *st.Discovery.PremisesCompleted
	if mode != nil && !premisesCompleted && !allAnswered {
		planNote := ""
		if st.Discovery.PlanPath != nil {
			planNote = " and the plan document"
		}
		out := DiscoveryOutput{
			Phase:         "DISCOVERY",
			Instruction:   "Before asking discovery questions, challenge the premises of this spec.",
			Questions:     []DiscoveryQuestion{},
			AnsweredCount: 0,
			Context: ContextBlock{
				Rules:            rules,
				ConcernReminders: reminders,
			},
			Transition: struct {
				OnComplete string `json:"onComplete"`
			}{OnComplete: cs("next --answer='{\"premises\":[]}'", specName)},
			PremiseChallenge: &PremiseChallengeOutput{
				Required: true,
				Instruction: "Read the spec description" + planNote +
					". Identify 2-4 premises the spec assumes. Present each premise and ask the user to agree or disagree. Submit as JSON: {\"premises\":[{\"text\":\"...\",\"agreed\":true/false,\"revision\":\"...\"}]}",
				Prompts: []string{
					"Is this the right problem to solve? Could a different framing yield a simpler solution?",
					"What happens if we do nothing? Is this a real pain point or a hypothetical one?",
					"What existing code already partially solves this? Can we build on it instead?",
				},
			},
		}
		if currentUser != nil {
			out.CurrentUser = &struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			}{Name: currentUser.Name, Email: currentUser.Email}
		}
		return out
	}

	// Mode-specific rules
	var modeRules []string
	if mode != nil {
		modeRules = getModeRules(*mode)
	}
	rulesWithMode := append(rules, modeRules...)

	// Rich description
	specDescription := ""
	if st.SpecDescription != nil {
		specDescription = *st.SpecDescription
	}
	isRichDescription := len(specDescription) > 500
	hasPersistedPrefills := len(st.Discovery.Prefills) > 0

	// Premise context
	var agreedPremises []string
	var revisedPremises []struct {
		Original string `json:"original"`
		Revision string `json:"revision"`
	}
	for _, p := range st.Discovery.Premises {
		if p.Agreed {
			agreedPremises = append(agreedPremises, p.Text)
		} else if p.Revision != nil {
			revisedPremises = append(revisedPremises, struct {
				Original string `json:"original"`
				Revision string `json:"revision"`
			}{Original: p.Text, Revision: *p.Revision})
		}
	}

	if allAnswered {
		history := st.RevisitHistory
		var lastRevisit *state.RevisitEntry
		if len(history) > 0 {
			lastRevisit = &history[len(history)-1]
		}

		instrBase := fmt.Sprintf("All discovery questions answered. Run: `%s`", cs("approve", specName))
		if lastRevisit != nil {
			instrBase = "This spec was revisited from EXECUTING. All previous answers are preserved. Review and approve, or revise answers before regenerating the spec."
		}

		base := DiscoveryOutput{
			Phase:         "DISCOVERY",
			Instruction:   instrBase,
			Questions:     []DiscoveryQuestion{},
			AnsweredCount: answeredCount,
			Context:       ContextBlock{Rules: rules, ConcernReminders: []string{}},
			Transition: struct {
				OnComplete string `json:"onComplete"`
			}{OnComplete: cs("approve", specName)},
		}
		if currentUser != nil {
			base.CurrentUser = &struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			}{Name: currentUser.Name, Email: currentUser.Email}
		}
		if len(specNotes) > 0 {
			base.Notes = specNotes
		}

		if lastRevisit != nil {
			trueVal := true
			reason := lastRevisit.Reason
			base.Revisited = &trueVal
			base.RevisitReason = &reason
			base.PreviousProgress = &PreviousProgress{
				CompletedTasks: lastRevisit.CompletedTasks,
				TotalTasks:     len(lastRevisit.CompletedTasks),
			}
		}
		return base
	}

	currentQ, currentIdx := selectCurrentDiscoveryQuestion(allQuestions, st.Discovery.Answers, st.Discovery.CurrentQuestion)

	// Agent mode: return only current question
	if isAgent {
		if currentQ == nil {
			return DiscoveryOutput{
				Phase:         "DISCOVERY",
				Instruction:   fmt.Sprintf("All discovery questions answered. Run: `%s`", cs("approve", specName)),
				Questions:     []DiscoveryQuestion{},
				AnsweredCount: answeredCount,
				Context:       ContextBlock{Rules: rules, ConcernReminders: []string{}},
				Transition: struct {
					OnComplete string `json:"onComplete"`
				}{OnComplete: cs("approve", specName)},
			}
		}

		question := buildDiscoveryQuestion(*currentQ, st.Discovery.Prefills)

		total := len(allQuestions)
		agentOut := DiscoveryOutput{
			Phase: "DISCOVERY",
			Instruction: fmt.Sprintf("Ask this question to the user using AskUserQuestion. Submit the answer with: `%s`",
				cs("next --agent --answer=\"<answer>\"", specName)),
			Questions:       []DiscoveryQuestion{question},
			AnsweredCount:   answeredCount,
			CurrentQuestion: &currentIdx,
			TotalQuestions:  &total,
			Context: ContextBlock{
				Rules:            rulesWithMode,
				ConcernReminders: reminders,
			},
			Transition: struct {
				OnComplete string `json:"onComplete"`
			}{OnComplete: cs("next --agent --answer=\"<answer>\"", specName)},
		}

		if len(agreedPremises) > 0 {
			agentOut.AgreedPremises = agreedPremises
		}
		if len(revisedPremises) > 0 {
			agentOut.RevisedPremises = revisedPremises
		}

		// Q1 enrichments
		if currentIdx == 0 {
			desc := ""
			if st.SpecDescription != nil {
				desc = *st.SpecDescription
			}
			if research := buildPreDiscoveryResearch(desc); research != nil {
				agentOut.PreDiscoveryResearch = research
			}

			planPath := ""
			if st.Discovery.PlanPath != nil {
				planPath = *st.Discovery.PlanPath
			}
			if planCtx := buildPlanContext(planPath); planCtx != nil {
				agentOut.PlanContext = planCtx
			} else if isRichDescription && !hasPersistedPrefills {
				agentOut.RichDescription = &RichDescriptionOutput{
					Provided:    true,
					Length:      len(specDescription),
					Content:     specDescription,
					Instruction: "The user provided a detailed description. For each question, extract relevant info and present as a pre-filled suggestion. IMPORTANT: When extracting answers from the description, mark each extraction as [STATED] (directly written by the user) or [INFERRED] (your interpretation). Present extractions individually for confirmation.",
				}
			}
		}

		// Pending follow-ups
		var pendingFU []state.FollowUp
		for _, f := range st.Discovery.FollowUps {
			if f.Status == "pending" {
				pendingFU = append(pendingFU, f)
			}
		}
		if len(pendingFU) > 0 {
			agentOut.PendingFollowUps = pendingFU
		}

		// Follow-up hints from last answer
		if len(st.Discovery.Answers) > 0 {
			lastAnswer := st.Discovery.Answers[len(st.Discovery.Answers)-1]
			if followHints := generateFollowUpHints(lastAnswer.Answer); len(followHints) > 0 {
				agentOut.FollowUpHints = followHints
			}
		}

		return agentOut
	}

	history := st.RevisitHistory
	var lastRevisit *state.RevisitEntry
	if len(history) > 0 {
		lastRevisit = &history[len(history)-1]
	}
	isRevisited := lastRevisit != nil

	revisitInstruction := "Conduct a thorough discovery conversation. FIRST: perform a pre-discovery codebase scan (README, CLAUDE.md, recent git log, TODOs, directory structure) and present a brief audit summary. THEN: challenge the user's spec description against your findings. THEN: ask the current discovery question, wait for the answer, and submit it immediately with `tddmaster next --answer=\"<answer>\"`. Continue one question at a time, offering concrete options based on codebase knowledge. AFTER questions: present a dream state table (current → this spec → future), scored expansion proposals, architectural decisions, and an error/rescue map. FINALLY: present a complete discovery synthesis for user confirmation before approve."
	if isRevisited {
		revisitInstruction = "This spec was revisited from EXECUTING. Previous discovery answers are preserved — review and revise as needed, or approve to regenerate tasks."
	}

	var questions []DiscoveryQuestion
	if currentQ != nil {
		questions = []DiscoveryQuestion{buildDiscoveryQuestion(*currentQ, st.Discovery.Prefills)}
	} else {
		questions = []DiscoveryQuestion{}
	}

	total := len(allQuestions)
	out := DiscoveryOutput{
		Phase:           "DISCOVERY",
		Instruction:     revisitInstruction,
		Questions:       questions,
		AnsweredCount:   answeredCount,
		CurrentQuestion: &currentIdx,
		TotalQuestions:  &total,
		Context: ContextBlock{
			Rules:            rulesWithMode,
			ConcernReminders: reminders,
		},
		Transition: struct {
			OnComplete string `json:"onComplete"`
		}{OnComplete: cs("next --answer=\"<answer>\"", specName)},
	}

	if currentUser != nil {
		out.CurrentUser = &struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}{Name: currentUser.Name, Email: currentUser.Email}
	}
	if len(specNotes) > 0 {
		out.Notes = specNotes
	}
	if len(agreedPremises) > 0 {
		out.AgreedPremises = agreedPremises
	}
	if len(revisedPremises) > 0 {
		out.RevisedPremises = revisedPremises
	}

	if isRevisited {
		trueVal := true
		reason := lastRevisit.Reason
		out.Revisited = &trueVal
		out.RevisitReason = &reason
		out.PreviousProgress = &PreviousProgress{
			CompletedTasks: lastRevisit.CompletedTasks,
			TotalTasks:     len(lastRevisit.CompletedTasks),
		}
		return out
	}

	// First-call enrichments
	if answeredCount == 0 {
		desc := ""
		if st.SpecDescription != nil {
			desc = *st.SpecDescription
		}
		if research := buildPreDiscoveryResearch(desc); research != nil {
			out.PreDiscoveryResearch = research
		}

		planPath := ""
		if st.Discovery.PlanPath != nil {
			planPath = *st.Discovery.PlanPath
		}
		if planCtx := buildPlanContext(planPath); planCtx != nil {
			out.PlanContext = planCtx
		} else if isRichDescription && !hasPersistedPrefills {
			out.RichDescription = &RichDescriptionOutput{
				Provided:    true,
				Length:      len(specDescription),
				Content:     specDescription,
				Instruction: "The user provided a detailed description. For each question, extract relevant info and present as a pre-filled suggestion.",
			}
		}
	}

	return out
}

func compileDiscoveryReview(
	st state.StateFile,
	activeConcerns []state.ConcernDefinition,
) DiscoveryReviewOutput {
	specName := st.Spec
	allQuestions := GetQuestionsWithExtras(activeConcerns)

	questionMap := make(map[string]string)
	for _, q := range allQuestions {
		questionMap[q.ID] = q.Text
	}

	var reviewAnswers []DiscoveryReviewAnswer
	for _, a := range st.Discovery.Answers {
		qText, ok := questionMap[a.QuestionID]
		if !ok {
			qText = a.QuestionID
		}
		reviewAnswers = append(reviewAnswers, DiscoveryReviewAnswer{
			QuestionID: a.QuestionID,
			Question:   qText,
			Answer:     a.Answer,
		})
	}

	splitProposal := AnalyzeForSplit(st.Discovery.Answers, "")

	// Sub-phase: approved + split detected → user decides
	if st.Discovery.Approved && splitProposal.Detected {
		return DiscoveryReviewOutput{
			Phase:       "DISCOVERY_REFINEMENT",
			Instruction: "Discovery answers are approved. tddmaster detected multiple independent work areas in this spec. Present the split proposal to the user and let them decide whether to keep as one spec or split into separate specs.",
			Answers:     reviewAnswers,
			Transition: struct {
				OnApprove string `json:"onApprove"`
				OnRevise  string `json:"onRevise"`
			}{
				OnApprove: cs("next --answer=\"keep\"", specName),
				OnRevise:  cs("next --answer='{\"revise\":{\"status_quo\":\"corrected answer\"}}'", specName),
			},
			SplitProposal: &splitProposal,
		}
	}

	// Sub-phase: approved + alternatives not presented
	alternativesPresented := st.Discovery.AlternativesPresented != nil && *st.Discovery.AlternativesPresented
	if st.Discovery.Approved && !alternativesPresented {
		subPhase := "alternatives"
		altFields := []string{"id", "name", "summary", "effort", "risk", "pros", "cons"}
		alt := &AlternativesOutput{
			Required:    true,
			Instruction: "Generate 2-3 approaches from discovery answers and codebase. Present via AskUserQuestion.",
		}
		alt.Format.Fields = altFields

		return DiscoveryReviewOutput{
			Phase:       "DISCOVERY_REFINEMENT",
			SubPhase:    &subPhase,
			Instruction: "Based on discovery answers, propose 2-3 distinct implementation approaches. Present each with name, summary, effort (S/M/L/XL), risk (Low/Med/High), pros, and cons. Ask the user to choose one, or skip.",
			Answers:     reviewAnswers,
			Transition: struct {
				OnApprove string `json:"onApprove"`
				OnRevise  string `json:"onRevise"`
			}{
				OnApprove: cs("next --answer='{\"approach\":\"A\",\"name\":\"...\",\"summary\":\"...\",\"effort\":\"M\",\"risk\":\"Low\"}'", specName),
				OnRevise:  cs("next --answer=\"skip\"", specName),
			},
			Alternatives: alt,
		}
	}

	// Batch warning
	batchWarning := ""
	if st.Discovery.BatchSubmitted != nil && *st.Discovery.BatchSubmitted {
		batchWarning = " IMPORTANT: These answers were BATCH-SUBMITTED (not confirmed one-by-one). You MUST present EVERY answer individually and get explicit user confirmation for each. Do NOT auto-approve."
	}

	// Build review checklist
	allDimensions := GetReviewDimensions(activeConcerns, nil)
	registryIDs := GetRegistryDimensionIds(activeConcerns)
	registrySet := make(map[string]bool)
	for _, id := range registryIDs {
		registrySet[id] = true
	}

	var reviewChecklist *ReviewChecklist
	if len(allDimensions) > 0 {
		var checklistDims []ReviewChecklistDimension
		hasRegistries := false
		for _, dim := range allDimensions {
			isReg := registrySet[dim.ID]
			if isReg {
				hasRegistries = true
			}
			checklistDims = append(checklistDims, ReviewChecklistDimension{
				ID:               dim.ID,
				Label:            dim.Label,
				Prompt:           dim.Prompt,
				EvidenceRequired: dim.EvidenceRequired,
				IsRegistry:       isReg,
				ConcernID:        dim.ConcernID,
			})
		}

		rc := &ReviewChecklist{
			Dimensions:  checklistDims,
			Instruction: "Before approving, review the plan against each dimension below. For dimensions marked evidenceRequired, cite specific files or code. Present findings to the user for each dimension via AskUserQuestion — one dimension at a time.",
		}
		if hasRegistries {
			regInstr := "Registry dimensions (isRegistry=true) require a structured table with every row filled. These tables will be included in the generated spec."
			rc.RegistryInstruction = &regInstr
		}
		reviewChecklist = rc
	}

	var instruction string
	if splitProposal.Detected {
		instruction = "Present ALL discovery answers to the user for review. ALSO present the split proposal — tddmaster detected multiple independent areas." + batchWarning
	} else {
		instruction = "Present ALL discovery answers to the user for review. The user must confirm or correct each answer before the spec can be generated. Use AskUserQuestion to ask for confirmation." + batchWarning
	}

	result := DiscoveryReviewOutput{
		Phase:       "DISCOVERY_REFINEMENT",
		Instruction: instruction,
		Answers:     reviewAnswers,
		Transition: struct {
			OnApprove string `json:"onApprove"`
			OnRevise  string `json:"onRevise"`
		}{
			OnApprove: cs("next --answer=\"approve\"", specName),
			OnRevise:  cs("next --answer='{\"revise\":{\"status_quo\":\"corrected answer\"}}'", specName),
		},
		ReviewChecklist: reviewChecklist,
	}
	if splitProposal.Detected {
		result.SplitProposal = &splitProposal
	}
	return result
}

func compileSpecDraft(st state.StateFile) SpecDraftOutput {
	specName := st.Spec
	edgeCases := spec.DeriveEdgeCases(st.Discovery.Answers, st.Discovery.Premises)

	if st.Classification == nil {
		specPath := ""
		if st.SpecState.Path != nil {
			specPath = *st.SpecState.Path
		}

		classTrue := true
		classPrompt := &ClassificationPrompt{
			Options: []struct {
				ID    string `json:"id"`
				Label string `json:"label"`
			}{
				{"involvesWebUI", "Web/Mobile UI — layouts, responsive design, visual components"},
				{"involvesCLI", "CLI/Terminal UI — spinners, progress bars, interactive prompts"},
				{"involvesPublicAPI", "Public API changes"},
				{"involvesMigration", "Data migration or schema changes"},
				{"involvesDataHandling", "Data handling or privacy"},
			},
			Instruction: "Select all that apply. Submit as JSON: `" +
				cs("next --answer='{\"involvesWebUI\":true,\"involvesCLI\":false,\"involvesPublicAPI\":false,...}'", specName) +
				"`. If none apply, answer with: `" +
				cs("next --answer=\"none\"", specName) + "`",
		}

		return SpecDraftOutput{
			Phase:       "SPEC_PROPOSAL",
			Instruction: "Before generating the spec, classify what this spec involves. Ask the user to select all that apply.",
			SpecPath:    specPath,
			EdgeCases:   edgeCases,
			Transition: struct {
				OnApprove string `json:"onApprove"`
			}{
				OnApprove: cs("next --answer='{\"involvesWebUI\":false,\"involvesCLI\":false,\"involvesPublicAPI\":false,\"involvesMigration\":false,\"involvesDataHandling\":false}'", specName),
			},
			ClassificationRequired: &classTrue,
			ClassificationPrompt:   classPrompt,
		}
	}

	specPath := ""
	if st.SpecState.Path != nil {
		specPath = *st.SpecState.Path
	}

	return SpecDraftOutput{
		Phase:       "SPEC_PROPOSAL",
		Instruction: "Spec draft is ready. Self-review before presenting to user.",
		SpecPath:    specPath,
		EdgeCases:   edgeCases,
		Transition: struct {
			OnApprove string `json:"onApprove"`
		}{OnApprove: cs("approve", specName)},
		SelfReview: &SelfReview{
			Required: true,
			Checks: []string{
				"Placeholder scan: no TBD, TODO, vague requirements",
				"Consistency: tasks match discovery, ACs match tasks",
				"Scope: single spec, not multiple independent subsystems",
				"Ambiguity: every AC has one interpretation",
				"Edge cases: discovery answers and revised premises are captured for test-writer coverage",
			},
			Instruction: "Review draft against these checks. If issues are found, send a refinement — DO NOT put a task list in `notes`. " +
				"Full task replacement: `next --answer='{\"refinement\":\"task-1: Title | task-2: Title | task-3: Title\"}'`. " +
				"Verb patch: `next --answer='{\"refinement\":{\"update\":{\"task-1\":\"New title\"},\"add\":[\"New task\"],\"remove\":[\"task-2\"]}}'`. " +
				"`notes` is reserved for free-form context only. Fix inline; do not ask the user to fix.",
		},
	}
}

func compileSpecApproved(st state.StateFile, config *state.NosManifest, parsedSpec *spec.ParsedSpec) SpecApprovedOutput {
	specName := st.Spec
	specPath := ""
	if st.SpecState.Path != nil {
		specPath = *st.SpecState.Path
	}
	out := SpecApprovedOutput{
		Phase:       "SPEC_APPROVED",
		Instruction: "Spec is approved and ready. When the user is ready to start, begin execution.",
		SpecPath:    specPath,
		Transition: struct {
			OnStart string `json:"onStart"`
		}{OnStart: cs("next --answer=\"start\"", specName)},
	}

	selectionPending := config != nil && config.IsTDDEnabled() &&
		(st.TaskTDDSelected == nil || !*st.TaskTDDSelected)
	if !selectionPending {
		return out
	}

	entries := buildTDDSelectionEntries(st, parsedSpec)
	if len(entries) == 0 {
		return out
	}

	out.Instruction = "Select TDD scope for this spec before starting execution. Some tasks (infrastructure setup, module downloads) often do not benefit from red/green/refactor."
	out.TaskTDDSelection = &TaskTDDSelectionOutput{
		Required:    true,
		Instruction: "Choose which tasks run with TDD. 'All' keeps current behavior; 'None' skips red/green/refactor for every task; 'Custom' lets you pick task-by-task.",
		Tasks:       entries,
		Answers: TaskTDDSelectionAnswers{
			All:    "tdd-all",
			None:   "tdd-none",
			Custom: `{"tddTasks":["task-1","task-3"]}`,
		},
	}
	return out
}

// buildTDDSelectionEntries returns the canonical task list for the TDD
// selection UI. Prefers StateFile.OverrideTasks when present, falling back to
// the parsed spec.md tasks.
func buildTDDSelectionEntries(st state.StateFile, parsedSpec *spec.ParsedSpec) []TaskTDDSelectionEntry {
	if len(st.OverrideTasks) > 0 {
		entries := make([]TaskTDDSelectionEntry, 0, len(st.OverrideTasks))
		for _, t := range st.OverrideTasks {
			entries = append(entries, TaskTDDSelectionEntry{
				ID:           t.ID,
				Title:        t.Title,
				SuggestedTDD: suggestTDDForTitle(t.Title),
			})
		}
		return entries
	}
	if parsedSpec == nil || len(parsedSpec.Tasks) == 0 {
		return nil
	}
	entries := make([]TaskTDDSelectionEntry, 0, len(parsedSpec.Tasks))
	for _, t := range parsedSpec.Tasks {
		entries = append(entries, TaskTDDSelectionEntry{
			ID:           t.ID,
			Title:        t.Title,
			SuggestedTDD: suggestTDDForTitle(t.Title),
		})
	}
	return entries
}

// nonTDDKeywordRe matches titles that almost never benefit from TDD — pure
// plumbing/scaffolding work. Used only to populate SuggestedTDD as an advisory
// hint; the user's explicit answer is the source of truth.
var nonTDDKeywordRe = regexp.MustCompile(`(?i)\b(download|install|scaffold|bootstrap|go\s+mod|go\.mod|init(?:ialize)?|create\s+(?:directory|folder|project|skeleton)|add\s+dependenc(?:y|ies)|configure\s+ci)\b`)

func suggestTDDForTitle(title string) bool {
	return !nonTDDKeywordRe.MatchString(title)
}

func isACRelevant(acText string, classification *state.SpecClassification) bool {
	if classification == nil {
		return false
	}

	lower := strings.ToLower(acText)

	if strings.Contains(lower, "mobile") || strings.Contains(lower, "layout") ||
		strings.Contains(lower, "interaction design") {
		return classification.InvolvesWebUI
	}

	if strings.Contains(lower, "ui state") || strings.Contains(lower, "skeleton ui") {
		return classification.InvolvesWebUI || classification.InvolvesCLI
	}

	if strings.Contains(lower, "readme") || strings.Contains(lower, "documentation updated") ||
		(strings.Contains(lower, "reflected in") && strings.Contains(lower, "docs")) {
		return true
	}

	if strings.Contains(lower, "api doc") || strings.Contains(lower, "public api") {
		return classification.InvolvesPublicAPI
	}

	if strings.Contains(lower, "migration") || strings.Contains(lower, "backward compat") ||
		strings.Contains(lower, "deprecat") {
		return classification.InvolvesMigration
	}

	if strings.Contains(lower, "audit trail") || strings.Contains(lower, "access control") ||
		strings.Contains(lower, "data handling") || strings.Contains(lower, "data retention") {
		return classification.InvolvesDataHandling
	}

	return true
}

func buildAcceptanceCriteria(
	activeConcerns []state.ConcernDefinition,
	verifyFailed bool,
	verifyOutput string,
	debt *state.DebtState,
	classification *state.SpecClassification,
	parsedSpec *spec.ParsedSpec,
	folderRuleCriteria []FolderRule,
	naItems []string,
) []AcceptanceCriterion {
	var criteria []AcceptanceCriterion
	naSet := make(map[string]bool, len(naItems))
	for _, id := range naItems {
		naSet[id] = true
	}

	acCounter := 0
	nextID := func() string {
		acCounter++
		return fmt.Sprintf("ac-%d", acCounter)
	}

	// Debt items
	if debt != nil {
		for _, item := range debt.Items {
			if naSet[item.ID] {
				continue
			}
			criteria = append(criteria, AcceptanceCriterion{
				ID:   item.ID,
				Text: fmt.Sprintf("[DEBT from iteration %d] %s", item.Since, item.Text),
			})
		}
	}

	// Verification failure
	if verifyFailed {
		truncated := verifyOutput
		if len(truncated) > 200 {
			truncated = truncated[:200]
		}
		criteria = append(criteria, AcceptanceCriterion{
			ID:   nextID(),
			Text: "[FAILED] Tests — fix this first: " + truncated,
		})
	}

	// Spec verification items
	if parsedSpec != nil {
		for _, item := range parsedSpec.Verification {
			id := nextID()
			if naSet[id] {
				continue
			}
			criteria = append(criteria, AcceptanceCriterion{ID: id, Text: item})
		}
	}

	// Concern-injected criteria
	for _, concern := range activeConcerns {
		for _, ac := range concern.AcceptanceCriteria {
			if !isACRelevant(ac, classification) {
				continue
			}
			id := nextID()
			if naSet[id] {
				continue
			}
			criteria = append(criteria, AcceptanceCriterion{
				ID:   id,
				Text: fmt.Sprintf("(%s) %s", concern.ID, ac),
			})
		}
	}

	// Folder rule criteria
	for _, fr := range folderRuleCriteria {
		id := nextID()
		if naSet[id] {
			continue
		}
		criteria = append(criteria, AcceptanceCriterion{
			ID:   id,
			Text: fmt.Sprintf("(folder: %s) %s", fr.Folder, fr.Rule),
		})
	}

	// Scope check
	if parsedSpec != nil {
		for _, t := range parsedSpec.Tasks {
			if len(t.Files) > 0 {
				criteria = append(criteria, AcceptanceCriterion{
					ID:   "scope-check",
					Text: fmt.Sprintf("Scope check: only files listed in task (%s) should be modified. Report any out-of-scope changes with justification.", strings.Join(t.Files, ", ")),
				})
				break
			}
		}
	}

	// Mandatory ACs
	criteria = append(criteria,
		AcceptanceCriterion{ID: "mandatory-tests", Text: "Tests written and passing for all new and changed behavior"},
		AcceptanceCriterion{ID: "mandatory-docs", Text: "Documentation updated for all public-facing changes"},
	)

	return criteria
}

// buildTDDVerificationContext returns phase-specific verification instructions for the verifier.
// Each phase spells out the expected exit-code contract and the JSON output shape the
// verifier must return so RecordTDDVerificationFull can make cycle transitions.
func buildTDDVerificationContext(cycle string) *TDDVerificationContext {
	var instruction string
	switch cycle {
	case state.TDDCycleRed:
		instruction = tddcontract.VerifierRedPhaseInstruction()
	case state.TDDCycleGreen:
		instruction = tddcontract.VerifierGreenPhaseInstruction("", "")
	case state.TDDCycleRefactor:
		instruction = tddcontract.VerifierRefactorPhaseInstruction("", "")
	default:
		return nil
	}
	return &TDDVerificationContext{Phase: cycle, Instruction: instruction}
}

// buildRefactorInstructions packages the verifier's refactor notes into an
// executor-facing directive. Returns nil when there are no notes to apply or
// when the executor has already consumed the current batch.
func buildRefactorInstructions(st state.StateFile, maxRounds int) *RefactorInstructions {
	if st.Execution.TDDCycle != state.TDDCycleRefactor {
		return nil
	}
	if st.Execution.LastVerification == nil {
		return nil
	}
	if st.Execution.RefactorApplied {
		// Executor already applied the latest batch — nothing to hand over now.
		return nil
	}
	notes := st.Execution.LastVerification.RefactorNotes
	if len(notes) == 0 {
		return nil
	}
	// Notes may originate from the merged GREEN scan (phase="green") or from a
	// post-executor REFACTOR scan (phase="refactor") — both are valid sources.
	return &RefactorInstructions{
		Notes:       notes,
		Instruction: "Apply each refactor note verbatim. Do NOT change test behavior — tests must still pass. When finished, report `refactorApplied: true` in your JSON output; the verifier will re-run tests.",
		Round:       st.Execution.RefactorRounds + 1,
		MaxRounds:   maxRounds,
	}
}

func compileExecution(
	st state.StateFile,
	activeConcerns []state.ConcernDefinition,
	rules []string,
	maxIterationsBeforeRestart int,
	parsedSpec *spec.ParsedSpec,
	folderRuleCriteria []FolderRule,
) ExecutionOutput {
	specName := st.Spec
	edgeCases := spec.DeriveEdgeCases(st.Discovery.Answers, st.Discovery.Premises)
	tensions := DetectTensions(activeConcerns)
	shouldRestart := st.Execution.Iteration >= maxIterationsBeforeRestart
	verifyFailed := st.Execution.LastVerification != nil && !st.Execution.LastVerification.Passed
	verifyOutputStr := ""
	if st.Execution.LastVerification != nil {
		verifyOutputStr = st.Execution.LastVerification.Output
	}

	// Find current task
	var specTasks []spec.ParsedTask
	if parsedSpec != nil {
		specTasks = parsedSpec.Tasks
	}
	completedIDs := st.Execution.CompletedTasks
	completedSet := make(map[string]bool, len(completedIDs))
	for _, id := range completedIDs {
		completedSet[id] = true
	}

	var nextTask *spec.ParsedTask
	for i := range specTasks {
		if !completedSet[specTasks[i].ID] {
			nextTask = &specTasks[i]
			break
		}
	}

	var taskBlock *TaskBlock
	if nextTask != nil {
		tb := TaskBlock{
			ID:             nextTask.ID,
			Title:          nextTask.Title,
			TotalTasks:     len(specTasks),
			CompletedTasks: len(completedIDs),
		}
		if len(nextTask.Files) > 0 {
			tb.Files = nextTask.Files
		}
		taskBlock = &tb
	}

	// Filter edge cases to those covered by the current task (if Covers is specified).
	// If Covers is empty (no mapping yet), all edge cases are included (backward-compat).
	taskEdgeCases := edgeCases
	if nextTask != nil && len(nextTask.Covers) > 0 {
		coverSet := make(map[string]bool, len(nextTask.Covers))
		for _, c := range nextTask.Covers {
			coverSet[strings.ToUpper(c)] = true
		}
		var filtered []string
		for i, ec := range edgeCases {
			ecID := fmt.Sprintf("EC-%d", i+1)
			if coverSet[ecID] {
				filtered = append(filtered, ec)
			}
		}
		taskEdgeCases = filtered
	}

	tier1Reminders, _ := SplitRemindersByTier(activeConcerns)

	// Status report flow
	if st.Execution.AwaitingStatusReport {
		var naItems []string
		if st.Execution.NaItems != nil {
			naItems = st.Execution.NaItems
		}
		criteria := buildAcceptanceCriteria(
			activeConcerns, verifyFailed, verifyOutputStr,
			st.Execution.Debt, st.Classification,
			parsedSpec, folderRuleCriteria, naItems,
		)

		// Detect batch task claims
		var batchTaskIDs []string
		if st.Execution.LastProgress != nil {
			var prevAnswer map[string]interface{}
			if err := json.Unmarshal([]byte(*st.Execution.LastProgress), &prevAnswer); err == nil {
				if completed, ok := prevAnswer["completed"].([]interface{}); ok {
					for _, id := range completed {
						if s, ok := id.(string); ok && strings.HasPrefix(s, "task-") {
							batchTaskIDs = append(batchTaskIDs, s)
						}
					}
				}
			}
		}

		batchInstruction := "Before this task is accepted, report your completion status against these acceptance criteria."
		if len(batchTaskIDs) >= 2 {
			batchInstruction = fmt.Sprintf("%d tasks reported complete. Report status against ALL relevant acceptance criteria.", len(batchTaskIDs))
		}

		statusTrue := true
		out := ExecutionOutput{
			Phase:       "EXECUTING",
			Instruction: batchInstruction,
			EdgeCases:   taskEdgeCases,
			Context: ContextBlock{
				Rules:            rules,
				ConcernReminders: tier1Reminders,
			},
			Transition: struct {
				OnComplete string `json:"onComplete"`
				OnBlocked  string `json:"onBlocked"`
				Iteration  int    `json:"iteration"`
			}{
				OnComplete: cs("next --answer='{\"completed\":[...],\"remaining\":[...],\"blocked\":[]}'", specName),
				OnBlocked:  cs("block \"reason\"", specName),
				Iteration:  st.Execution.Iteration,
			},
			StatusReportRequired: &statusTrue,
			StatusReport: &StatusReportRequest{
				Criteria: criteria,
				ReportFormat: struct {
					Completed string `json:"completed"`
					Remaining string `json:"remaining"`
					Blocked   string `json:"blocked"`
					NA        string `json:"na"`
					NewIssues string `json:"newIssues"`
				}{
					Completed: "list item IDs you finished (e.g., ['debt-1', 'ac-3']) with evidence",
					Remaining: "list item IDs not yet done",
					Blocked:   "list item IDs that need a decision from the user",
					NA:        "(optional) list item IDs that are not applicable to this task — they will be removed from future criteria",
					NewIssues: "(optional) list NEW issues discovered during implementation — free text, will be assigned debt IDs automatically",
				},
			},
		}
		if len(batchTaskIDs) >= 2 {
			out.BatchTasks = batchTaskIDs
		}
		if verifyFailed {
			trueVal := true
			truncated := verifyOutputStr
			if len(truncated) > 2000 {
				truncated = truncated[:2000]
			}
			out.VerificationFailed = &trueVal
			out.VerificationOutput = &truncated
		}
		return out
	}

	// Normal execution
	wasRejected := st.Execution.LastProgress != nil && strings.Contains(*st.Execution.LastProgress, "Task not accepted")
	var debtItems []state.DebtItem
	debtUnaddressed := 0
	if st.Execution.Debt != nil {
		debtItems = st.Execution.Debt.Items
		debtUnaddressed = st.Execution.Debt.UnaddressedIterations
	}

	taskInstruction := fmt.Sprintf("All tasks completed. Run `%s` to finish.", cs("done", specName))
	if taskBlock != nil {
		taskInstruction = fmt.Sprintf("Execute task %s: %s (%d/%d completed)",
			taskBlock.ID, taskBlock.Title, taskBlock.CompletedTasks, taskBlock.TotalTasks)
	}

	var baseInstruction string
	if verifyFailed {
		baseInstruction = "Verification FAILED. Fix the failing tests before continuing."
	} else if wasRejected && len(debtItems) > 0 {
		urgencySuffix := ""
		if debtUnaddressed >= 3 {
			urgencySuffix = fmt.Sprintf(" These items have been outstanding for %d iterations.", debtUnaddressed)
		}
		baseInstruction = fmt.Sprintf("Task not accepted — %d remaining item(s) must be addressed before this task can be completed.%s Address them, then submit a new status report.", len(debtItems), urgencySuffix)
	} else {
		baseInstruction = taskInstruction
	}

	out := ExecutionOutput{
		Phase:       "EXECUTING",
		Instruction: baseInstruction,
		Task:        taskBlock,
		EdgeCases:   taskEdgeCases,
		Context: ContextBlock{
			Rules:            rules,
			ConcernReminders: tier1Reminders,
		},
		Transition: struct {
			OnComplete string `json:"onComplete"`
			OnBlocked  string `json:"onBlocked"`
			Iteration  int    `json:"iteration"`
		}{
			OnComplete: cs("next --answer=\"...\"", specName),
			OnBlocked:  cs("block \"reason\"", specName),
			Iteration:  st.Execution.Iteration,
		},
	}
	if len(edgeCases) > 0 {
		out.Instruction += " Use the listed edge cases to drive test-writer coverage before implementation."
	}

	// Task rejection info
	if wasRejected && len(debtItems) > 0 {
		trueVal := true
		reason := fmt.Sprintf("%d remaining item(s) must be addressed.", len(debtItems))
		remaining := make([]string, len(debtItems))
		for i, d := range debtItems {
			remaining[i] = d.Text
		}
		out.TaskRejected = &trueVal
		out.RejectionReason = &reason
		out.RejectionRemaining = remaining
	}

	// Carry forward debt
	if st.Execution.Debt != nil && len(st.Execution.Debt.Items) > 0 {
		unaddressed := st.Execution.Debt.UnaddressedIterations
		debtNote := "These were not completed in a previous iteration. Address them BEFORE starting new work."
		if unaddressed >= 3 {
			debtNote = fmt.Sprintf("URGENT: These items have been unaddressed for %d iterations. Address them IMMEDIATELY before any new work.", unaddressed)
		}
		out.PreviousIterationDebt = &DebtCarryForward{
			FromIteration: st.Execution.Debt.FromIteration,
			Items:         st.Execution.Debt.Items,
			Note:          debtNote,
		}
	}

	if verifyFailed {
		trueVal := true
		truncated := verifyOutputStr
		if len(truncated) > 2000 {
			truncated = truncated[:2000]
		}
		out.VerificationFailed = &trueVal
		out.VerificationOutput = &truncated
	}

	// Tension gate
	if len(tensions) > 0 {
		var tensionParts []string
		for _, t := range tensions {
			tensionParts = append(tensionParts, strings.Join(t.Between, " vs ")+": "+t.Issue)
		}
		out.ConcernTensions = tensions
		out.Instruction = fmt.Sprintf("TENSION GATE: %d concern tension(s) detected: %s. You MUST present these to the user and get explicit resolution for each before proceeding. Use AskUserQuestion to ask which side to prioritize.", len(tensions), strings.Join(tensionParts, "; "))
	}

	if shouldRestart {
		trueVal := true
		name := ""
		if st.Spec != nil {
			name = *st.Spec
		}
		restartInstr := fmt.Sprintf("Context may be getting large after %d iterations. Consider starting a new conversation and running `%s` to resume - your progress is saved.", st.Execution.Iteration, cs("next", &name))
		out.RestartRecommended = &trueVal
		out.RestartInstruction = &restartInstr
	}

	// Promote prompt for last unpromoted decision
	for i := len(st.Decisions) - 1; i >= 0; i-- {
		d := st.Decisions[i]
		if !d.Promoted {
			lastProgressStr := ""
			if st.Execution.LastProgress != nil {
				lastProgressStr = *st.Execution.LastProgress
			}
			if strings.HasPrefix(lastProgressStr, "Resolved:") {
				out.PromotePrompt = &PromotePrompt{
					DecisionID: d.ID,
					Question:   d.Question,
					Choice:     d.Choice,
					Prompt: fmt.Sprintf("You just resolved a decision: \"%s\". Ask the user: \"Should this be a permanent rule for future specs too?\" If yes, run: `%s`",
						d.Choice, c(fmt.Sprintf("rule add \"%s\"", d.Choice))),
				}
			}
			break
		}
	}

	// Pre-execution review on first iteration
	if st.Execution.Iteration == 0 && !verifyFailed && !wasRejected {
		out.PreExecutionReview = &PreExecutionReview{
			Instruction: "Re-read spec before starting. Flag: missing info that will block mid-execution, wrong task order, unclear ACs. Better to catch now than mid-execution.",
		}
	}

	// Design checklist for beautiful-product
	for _, cc := range activeConcerns {
		if cc.ID == "beautiful-product" {
			out.DesignChecklist = &DesignChecklist{
				Required:    true,
				Instruction: "Before completing any UI task, rate your implementation 0-10 on these dimensions and include the ratings in your AC report:",
				Dimensions: []DesignChecklistDimension{
					{ID: "hierarchy", Label: "Information hierarchy — what does the user see first, second, third?"},
					{ID: "states", Label: "Interaction states — loading, empty, error, success all specified?"},
					{ID: "edge-cases", Label: "Edge cases — long text, zero results, slow connection handled?"},
					{ID: "intentionality", Label: "Overall intentionality — does this feel designed or generated?"},
				},
			}
			break
		}
	}

	return out
}

func compileBlocked(st state.StateFile) BlockedOutput {
	specName := st.Spec
	reason := "Unknown"
	if st.Execution.LastProgress != nil {
		reason = *st.Execution.LastProgress
	}
	return BlockedOutput{
		Phase:       "BLOCKED",
		Instruction: "A decision is needed. Ask the user.",
		Reason:      reason,
		Transition: struct {
			OnResolved string `json:"onResolved"`
		}{OnResolved: cs("next --answer=\"...\"", specName)},
	}
}

func compileCompleted(st state.StateFile) CompletedOutput {
	learningsTrue := true
	name := ""
	if st.Spec != nil {
		name = *st.Spec
	}

	out := CompletedOutput{
		Phase:            "COMPLETED",
		LearningsPending: &learningsTrue,
	}
	out.Summary.Spec = st.Spec
	out.Summary.Iterations = st.Execution.Iteration
	out.Summary.DecisionsCount = len(st.Decisions)
	out.Summary.CompletionReason = st.CompletionReason
	out.Summary.CompletionNote = st.CompletionNote

	out.LearningPrompt = &struct {
		Instruction string   `json:"instruction"`
		Examples    []string `json:"examples"`
	}{
		Instruction: fmt.Sprintf("LEARNING PENDING — Record learnings before moving on. For each insight, decide: one-time learning or permanent rule? One-time (\"assumed X, was Y\") → `learn \"text\"`. Permanent (\"always/never do X\") → `learn \"text\" --rule`. Run: `tddmaster spec %s learn \"text\"` or `learn \"text\" --rule`.", name),
		Examples: []string{
			fmt.Sprintf("tddmaster spec %s learn \"Assumed S3 SDK v2, was v3\"", name),
			fmt.Sprintf("tddmaster spec %s learn \"Always use Result types\" --rule", name),
			"tddmaster learn promote 1",
		},
	}

	return out
}

// =============================================================================
// Main Compile Function
// =============================================================================

// Compile builds the NextOutput for a given state.
func Compile(
	st state.StateFile,
	activeConcerns []state.ConcernDefinition,
	rules []string,
	config *state.NosManifest,
	parsedSpec *spec.ParsedSpec,
	folderRuleCriteria []FolderRule,
	idleContext *IdleContext,
	interactionHints *InteractionHints,
	currentUser *struct{ Name, Email string },
	tier2Count int,
) NextOutput {
	hints := DefaultHints
	if interactionHints != nil {
		hints = *interactionHints
	}

	meta := buildMeta(st, activeConcerns, hints)

	maxIter := 15
	allowGit := false
	if config != nil {
		if config.MaxIterationsBeforeRestart > 0 {
			maxIter = config.MaxIterationsBeforeRestart
		}
		allowGit = config.AllowGit
	}

	behavioral := buildBehavioral(st, maxIter, allowGit, activeConcerns, parsedSpec, hints)

	// Inject tier2 summary for EXECUTING phase
	if st.Phase == state.PhaseExecuting {
		_, tier2Reminders := SplitRemindersByTier(activeConcerns)
		totalT2 := tier2Count + len(tier2Reminders)
		if totalT2 > 0 {
			summary := fmt.Sprintf("%d file-specific rules delivered via PreToolUse hook when editing matching files.", totalT2)
			behavioral.Tier2Summary = &summary
		}
	}

	protocolGuide := buildProtocolGuide(st)
	roadmap := buildRoadmap(st.Phase)
	gate := buildGate(st, parsedSpec)

	// Build phase-specific output
	var phase string
	var discoveryData *DiscoveryOutput
	var discoveryReviewData *DiscoveryReviewOutput
	var specDraftData *SpecDraftOutput
	var specApprovedData *SpecApprovedOutput
	var executionData *ExecutionOutput
	var blockedData *BlockedOutput
	var completedData *CompletedOutput
	var idleData *IdleOutput

	switch st.Phase {
	case state.PhaseIdle:
		phase = "IDLE"
		allConcerns := defaults.DefaultConcerns()
		idle := compileIdle(activeConcerns, allConcerns, len(rules), idleContext)
		idleData = &idle

	case state.PhaseDiscovery:
		phase = "DISCOVERY"
		disc := compileDiscovery(st, activeConcerns, rules, currentUser)
		discoveryData = &disc

	case state.PhaseDiscoveryRefinement:
		phase = "DISCOVERY_REFINEMENT"
		dr := compileDiscoveryReview(st, activeConcerns)
		discoveryReviewData = &dr

	case state.PhaseSpecProposal:
		phase = "SPEC_PROPOSAL"
		sd := compileSpecDraft(st)
		specDraftData = &sd

	case state.PhaseSpecApproved:
		phase = "SPEC_APPROVED"
		sa := compileSpecApproved(st, config, parsedSpec)
		specApprovedData = &sa

	case state.PhaseExecuting:
		phase = "EXECUTING"
		exec := compileExecution(st, activeConcerns, rules, maxIter, parsedSpec, folderRuleCriteria)
		currentTaskUsesTDD := state.ShouldRunTDDForCurrentTask(st, config)
		if currentTaskUsesTDD && st.Execution.TDDCycle != "" {
			cycle := st.Execution.TDDCycle
			exec.TDDPhase = &cycle
			exec.TDDVerificationContext = buildTDDVerificationContext(cycle)
			maxRounds := 0
			if config != nil && config.Tdd != nil {
				maxRounds = config.Tdd.MaxRefactorRounds
			}
			exec.RefactorInstructions = buildRefactorInstructions(st, maxRounds)
		}
		if currentTaskUsesTDD &&
			st.Execution.LastVerification != nil && !st.Execution.LastVerification.Passed {
			maxRetries := 0
			if config.Tdd != nil {
				maxRetries = config.Tdd.MaxVerificationRetries
			}
			failCount := st.Execution.LastVerification.VerificationFailCount
			exec.TDDFailureReport = &TDDFailureReport{
				Reason:             "verification-failed",
				UncoveredEdgeCases: st.Execution.LastVerification.UncoveredEdgeCases,
				RetryCount:         failCount,
				MaxRetries:         maxRetries,
				WillBlock:          maxRetries > 0 && failCount >= maxRetries,
			}
		}
		executionData = &exec

	case state.PhaseBlocked:
		phase = "BLOCKED"
		bl := compileBlocked(st)
		blockedData = &bl

	case state.PhaseCompleted:
		phase = "COMPLETED"
		comp := compileCompleted(st)
		completedData = &comp

	default:
		phase = "IDLE"
		allConcerns := defaults.DefaultConcerns()
		idle := compileIdle(activeConcerns, allConcerns, len(rules), idleContext)
		idleData = &idle
	}

	result := NextOutput{
		Phase:               phase,
		Meta:                meta,
		Behavioral:          behavioral,
		Roadmap:             roadmap,
		Gate:                gate,
		ProtocolGuide:       protocolGuide,
		DiscoveryData:       discoveryData,
		DiscoveryReviewData: discoveryReviewData,
		SpecDraftData:       specDraftData,
		SpecApprovedData:    specApprovedData,
		ExecutionData:       executionData,
		BlockedData:         blockedData,
		CompletedData:       completedData,
		IdleData:            idleData,
	}

	// Build interactive options
	internalOpts := buildInteractiveOptions(st, activeConcerns, idleContext, config)
	if len(internalOpts) > 0 {
		opts := make([]InteractiveOption, len(internalOpts))
		cmdMap := make(map[string]string, len(internalOpts))
		for i, opt := range internalOpts {
			opts[i] = InteractiveOption{Label: opt.Label, Description: opt.Description}
			cmdMap[opt.Label] = opt.Command
		}

		toolHint := "AskUserQuestion"
		toolHintInstruction := "Use AskUserQuestion tool to present these options. Do NOT use prose."
		if hints.OptionPresentation != "tool" {
			toolHint = "prose-numbered-list"
			toolHintInstruction = "Present options as a numbered list. Ask user to pick a number."
		}

		result.InteractiveOptions = opts
		result.CommandMap = cmdMap
		result.ToolHint = &toolHint
		result.ToolHintInstruction = &toolHintInstruction
	}

	return result
}

// =============================================================================
// TDD Rules Injection
// =============================================================================

// tddBehavioralRules is the canonical set of TDD behavioral rules injected when
// tddMode is active.  Keeping them in a package-level slice lets tests verify the
// exact rule texts without coupling to buildBehavioral internals.
var tddBehavioralRules = []string{
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

// TDDRules returns the canonical TDD behavioral rules slice.
// Tests and callers can call this to retrieve the rules without hard-coding them.
func TDDRules() []string {
	out := make([]string, len(tddBehavioralRules))
	copy(out, tddBehavioralRules)
	return out
}

// InjectTDDRules appends TDD behavioral rules to rules and returns the combined
// slice.  The original slice is not modified.  When tddMode is false callers
// should skip this function entirely — the check is the caller's responsibility.
func InjectTDDRules(rules []string) []string {
	combined := make([]string, len(rules), len(rules)+len(tddBehavioralRules))
	copy(combined, rules)
	return append(combined, tddBehavioralRules...)
}

// =============================================================================
// Helpers
// =============================================================================

func strPtr(s string) *string {
	return &s
}
