// Package state provides types and state management for tddmaster.
package state

import (
	"encoding/json"
	"fmt"
)

// =============================================================================
// Phases
// =============================================================================

type Phase string

const (
	PhaseUninitialized       Phase = "UNINITIALIZED"
	PhaseIdle                Phase = "IDLE"
	PhaseDiscovery           Phase = "DISCOVERY"
	PhaseDiscoveryRefinement Phase = "DISCOVERY_REFINEMENT"
	PhaseSpecProposal        Phase = "SPEC_PROPOSAL"
	PhaseSpecApproved        Phase = "SPEC_APPROVED"
	PhaseExecuting           Phase = "EXECUTING"
	PhaseBlocked             Phase = "BLOCKED"
	PhaseCompleted           Phase = "COMPLETED"
)

// =============================================================================
// TDD Cycle
// =============================================================================

const (
	TDDCycleRed      = "red"
	TDDCycleGreen    = "green"
	TDDCycleRefactor = "refactor"
)

type CompletionReason string

const (
	CompletionReasonDone      CompletionReason = "done"
	CompletionReasonCancelled CompletionReason = "cancelled"
	CompletionReasonWontfix   CompletionReason = "wontfix"
)

type DiscoveryMode string

const (
	DiscoveryModeFull           DiscoveryMode = "full"
	DiscoveryModeValidate       DiscoveryMode = "validate"
	DiscoveryModeTechnicalDepth DiscoveryMode = "technical-depth"
	DiscoveryModeShipFast       DiscoveryMode = "ship-fast"
	DiscoveryModeExplore        DiscoveryMode = "explore"
)

// =============================================================================
// Discovery
// =============================================================================

type DiscoveryAnswer struct {
	QuestionID string `json:"questionId"`
	Answer     string `json:"answer"`
}

// AttributedDiscoveryAnswer is an extended discovery answer with attribution.
// Old format (just questionId+answer) still works via NormalizeAnswer.
type AttributedDiscoveryAnswer struct {
	QuestionID string  `json:"questionId"`
	Answer     string  `json:"answer"`
	User       string  `json:"user"`
	Email      string  `json:"email"`
	Timestamp  string  `json:"timestamp"`
	Type       string  `json:"type"`                 // "original" | "addition" | "revision"
	Confidence *int    `json:"confidence,omitempty"` // 1-10
	Basis      *string `json:"basis,omitempty"`
}

// ConfidenceFinding is a confidence-scored finding from agent analysis.
type ConfidenceFinding struct {
	Finding    string `json:"finding"`
	Confidence int    `json:"confidence"` // 1-10
	Basis      string `json:"basis"`
}

type Premise struct {
	Text      string  `json:"text"`
	Agreed    bool    `json:"agreed"`
	Revision  *string `json:"revision,omitempty"`
	User      string  `json:"user"`
	Timestamp string  `json:"timestamp"`
}

type SelectedApproach struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Summary   string `json:"summary"`
	Effort    string `json:"effort"`
	Risk      string `json:"risk"`
	User      string `json:"user"`
	Timestamp string `json:"timestamp"`
}

type PhaseTransition struct {
	From      Phase   `json:"from"`
	To        Phase   `json:"to"`
	User      string  `json:"user"`
	Email     string  `json:"email"`
	Timestamp string  `json:"timestamp"`
	Reason    *string `json:"reason,omitempty"`
}

type CustomAC struct {
	ID           string `json:"id"`
	Text         string `json:"text"`
	User         string `json:"user"`
	Email        string `json:"email"`
	Timestamp    string `json:"timestamp"`
	AddedInPhase Phase  `json:"addedInPhase"`
}

type SpecNote struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	User      string `json:"user"`
	Email     string `json:"email"`
	Timestamp string `json:"timestamp"`
	Phase     Phase  `json:"phase"`
}

type FollowUp struct {
	ID               string  `json:"id"`
	ParentQuestionID string  `json:"parentQuestionId"`
	Question         string  `json:"question"`
	Answer           *string `json:"answer"`
	Status           string  `json:"status"` // "pending" | "answered" | "skipped"
	CreatedBy        string  `json:"createdBy"`
	CreatedAt        string  `json:"createdAt"`
	AnsweredAt       *string `json:"answeredAt,omitempty"`
}

type Delegation struct {
	QuestionID  string  `json:"questionId"`
	DelegatedTo string  `json:"delegatedTo"`
	DelegatedBy string  `json:"delegatedBy"`
	Status      string  `json:"status"` // "pending" | "answered"
	DelegatedAt string  `json:"delegatedAt"`
	Answer      *string `json:"answer,omitempty"`
	AnsweredBy  *string `json:"answeredBy,omitempty"`
	AnsweredAt  *string `json:"answeredAt,omitempty"`
}

type DiscoveryState struct {
	Answers               []DiscoveryAnswer `json:"answers"`
	Completed             bool              `json:"completed"`
	CurrentQuestion       int               `json:"currentQuestion"`
	Audience              string            `json:"audience"` // "agent" | "human"
	Approved              bool              `json:"approved"`
	PlanPath              *string           `json:"planPath"`
	Mode                  *DiscoveryMode    `json:"mode,omitempty"`
	Premises              []Premise         `json:"premises,omitempty"`
	SelectedApproach      *SelectedApproach `json:"selectedApproach,omitempty"`
	PremisesCompleted     *bool             `json:"premisesCompleted,omitempty"`
	AlternativesPresented *bool             `json:"alternativesPresented,omitempty"`
	Contributors          []string          `json:"contributors,omitempty"`
	Delegations           []Delegation      `json:"delegations,omitempty"`
	FollowUps             []FollowUp        `json:"followUps,omitempty"`
	UserContext           *string           `json:"userContext,omitempty"`
	UserContextProcessed  *bool             `json:"userContextProcessed,omitempty"`
	// Jidoka C1: answers were batch-submitted by agent and need user confirmation.
	BatchSubmitted *bool `json:"batchSubmitted,omitempty"`
}

// =============================================================================
// Spec
// =============================================================================

type SpecState struct {
	Path   *string `json:"path"`
	Status string  `json:"status"` // "none" | "draft" | "approved"
}

// =============================================================================
// Execution
// =============================================================================

type RefactorNote struct {
	File       string `json:"file"`
	Suggestion string `json:"suggestion"`
	Rationale  string `json:"rationale"`
}

type VerificationResult struct {
	Passed                bool           `json:"passed"`
	Output                string         `json:"output"`
	Timestamp             string         `json:"timestamp"`
	UncoveredEdgeCases    []string       `json:"uncoveredEdgeCases,omitempty"`
	VerificationFailCount int            `json:"verificationFailCount,omitempty"`
	RefactorNotes         []RefactorNote `json:"refactorNotes,omitempty"`
	Phase                 string         `json:"phase,omitempty"`
}

type StatusReport struct {
	Completed []string `json:"completed"`
	Remaining []string `json:"remaining"`
	Blocked   []string `json:"blocked"`
	Iteration int      `json:"iteration"`
	Timestamp string   `json:"timestamp"`
}

type DebtItem struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Since int    `json:"since"`
}

type DebtState struct {
	Items                 []DebtItem `json:"items"`
	FromIteration         int        `json:"fromIteration"` // kept for backward compat
	UnaddressedIterations int        `json:"unaddressedIterations"`
}

type SpecTask struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Completed  bool     `json:"completed"`
	Covers     []string `json:"covers,omitempty"`     // EC IDs this task covers, e.g. ["EC-1","EC-3"]
	TDDEnabled *bool    `json:"tddEnabled,omitempty"` // nil = fall back to spec-level TddMode
}

type SpecClassification struct {
	InvolvesWebUI        bool `json:"involvesWebUI"`
	InvolvesCLI          bool `json:"involvesCLI"`
	InvolvesPublicAPI    bool `json:"involvesPublicAPI"`
	InvolvesMigration    bool `json:"involvesMigration"`
	InvolvesDataHandling bool `json:"involvesDataHandling"`
}

type ExecutionState struct {
	Iteration            int                 `json:"iteration"`
	LastProgress         *string             `json:"lastProgress"`
	ModifiedFiles        []string            `json:"modifiedFiles"`
	LastVerification     *VerificationResult `json:"lastVerification"`
	AwaitingStatusReport bool                `json:"awaitingStatusReport"`
	Debt                 *DebtState          `json:"debt"`
	CompletedTasks       []string            `json:"completedTasks"`
	DebtCounter          int                 `json:"debtCounter"`
	NaItems              []string            `json:"naItems"`
	ConfidenceFindings   []ConfidenceFinding `json:"confidenceFindings,omitempty"`
	TDDCycle             string              `json:"tddCycle,omitempty" yaml:"tddCycle,omitempty"`
	RefactorRounds       int                 `json:"refactorRounds,omitempty"`
	RefactorApplied      bool                `json:"refactorApplied,omitempty"`
}

// =============================================================================
// Decision
// =============================================================================

type Decision struct {
	ID        string `json:"id"`
	Question  string `json:"question"`
	Choice    string `json:"choice"`
	Promoted  bool   `json:"promoted"`
	Timestamp string `json:"timestamp"`
}

// =============================================================================
// Revisit History
// =============================================================================

type RevisitEntry struct {
	From           Phase    `json:"from"`
	Reason         string   `json:"reason"`
	CompletedTasks []string `json:"completedTasks"`
	Timestamp      string   `json:"timestamp"`
}

// =============================================================================
// State File (.tddmaster/.state/state.json)
// =============================================================================

type StateFile struct {
	Version            string              `json:"version"`
	Phase              Phase               `json:"phase"`
	Spec               *string             `json:"spec"`
	SpecDescription    *string             `json:"specDescription"`
	Branch             *string             `json:"branch"`
	Discovery          DiscoveryState      `json:"discovery"`
	SpecState          SpecState           `json:"specState"`
	Execution          ExecutionState      `json:"execution"`
	Decisions          []Decision          `json:"decisions"`
	LastCalledAt       *string             `json:"lastCalledAt"`
	Classification     *SpecClassification `json:"classification"`
	CompletionReason   *CompletionReason   `json:"completionReason"`
	CompletedAt        *string             `json:"completedAt"`
	CompletionNote     *string             `json:"completionNote"`
	ReopenedFrom       *string             `json:"reopenedFrom"`
	RevisitHistory     []RevisitEntry      `json:"revisitHistory"`
	TransitionHistory  []PhaseTransition   `json:"transitionHistory,omitempty"`
	CustomACs          []CustomAC          `json:"customACs,omitempty"`
	SpecNotes          []SpecNote          `json:"specNotes,omitempty"`
	OverrideTasks      []SpecTask          `json:"overrideTasks,omitempty"`
	OverrideOutOfScope []string            `json:"overrideOutOfScope,omitempty"`
	TaskTDDSelected    *bool               `json:"taskTDDSelected,omitempty"` // per-task TDD selection completed gate
	LastAnswer         *AnswerFingerprint  `json:"lastAnswer,omitempty"`      // idempotency for --answer retries
}

// AnswerFingerprint records the last successfully processed answer so that
// re-submissions (error retries, duplicate deliveries) are idempotent within
// the same phase. A replay in the same (phase, hash) tuple is a no-op; the
// same text in a later phase is treated as fresh input.
type AnswerFingerprint struct {
	Phase     Phase  `json:"phase"`
	Hash      string `json:"hash"`      // sha256(trimmed answer)[:8] hex = 16 chars
	Timestamp string `json:"timestamp"` // RFC3339
}

// CreateInitialState creates a new default StateFile.
func CreateInitialState() StateFile {
	return StateFile{
		Version:         "0.1.0",
		Phase:           PhaseIdle,
		Spec:            nil,
		SpecDescription: nil,
		Branch:          nil,
		Discovery: DiscoveryState{
			Answers:         []DiscoveryAnswer{},
			Completed:       false,
			CurrentQuestion: 0,
			Audience:        "human",
			Approved:        false,
			PlanPath:        nil,
		},
		SpecState: SpecState{Path: nil, Status: "none"},
		Execution: ExecutionState{
			Iteration:            0,
			LastProgress:         nil,
			ModifiedFiles:        []string{},
			LastVerification:     nil,
			AwaitingStatusReport: false,
			Debt:                 nil,
			CompletedTasks:       []string{},
			DebtCounter:          0,
			NaItems:              []string{},
		},
		Decisions:        []Decision{},
		LastCalledAt:     nil,
		Classification:   nil,
		CompletionReason: nil,
		CompletedAt:      nil,
		CompletionNote:   nil,
		ReopenedFrom:     nil,
		RevisitHistory:   []RevisitEntry{},
	}
}

// =============================================================================
// Config (tddmaster section in .tddmaster/manifest.yml)
// =============================================================================

type ProjectTraits struct {
	Languages  []string `json:"languages"  yaml:"languages"`
	Frameworks []string `json:"frameworks" yaml:"frameworks"`
	CI         []string `json:"ci"         yaml:"ci"`
	TestRunner *string  `json:"testRunner" yaml:"testRunner"`
}

type CodingToolId string

const (
	CodingToolClaudeCode CodingToolId = "claude-code"
	CodingToolOpencode   CodingToolId = "opencode"
	CodingToolCodex      CodingToolId = "codex"
)

type UserConfig struct {
	Name  string `json:"name"  yaml:"name"`
	Email string `json:"email" yaml:"email"`
}

type NosManifest struct {
	Concerns                   []string       `json:"concerns"                   yaml:"concerns"`
	Tools                      []CodingToolId `json:"tools"                      yaml:"tools"`
	DefaultRunner              string         `json:"defaultRunner,omitempty"    yaml:"defaultRunner,omitempty"`
	Project                    ProjectTraits  `json:"project"                    yaml:"project"`
	MaxIterationsBeforeRestart int            `json:"maxIterationsBeforeRestart" yaml:"maxIterationsBeforeRestart"`
	Tdd                        *Manifest      `json:"tdd,omitempty"              yaml:"tdd,omitempty"`
	VerifyCommand              *string        `json:"verifyCommand"              yaml:"verifyCommand"`
	AllowGit                   bool           `json:"allowGit"                   yaml:"allowGit"`
	Command                    string         `json:"command"                    yaml:"command"`
	User                       *UserConfig    `json:"user,omitempty"             yaml:"user,omitempty"`
}

// IsTDDEnabled returns true when the TDD workflow is enabled in this manifest.
// It returns false when the Tdd field is nil or when TddMode is explicitly false.
func (m NosManifest) IsTDDEnabled() bool {
	return m.Tdd != nil && m.Tdd.TddMode
}

// CreateInitialManifest creates a new default NosManifest.
func CreateInitialManifest(
	concerns []string,
	tools []CodingToolId,
	project ProjectTraits,
) NosManifest {
	return NosManifest{
		Concerns:                   concerns,
		Tools:                      tools,
		Project:                    project,
		MaxIterationsBeforeRestart: 15,
		Tdd:                        &Manifest{TddMode: true, MaxVerificationRetries: 3, MaxRefactorRounds: 3},
		VerifyCommand:              nil,
		AllowGit:                   false,
		Command:                    "tddmaster",
	}
}

// =============================================================================
// Discovery Answer Helpers (backward-compatible normalization)
// =============================================================================

// NormalizeAnswer normalizes a DiscoveryAnswer to AttributedDiscoveryAnswer.
// Handles both old format (just questionId+answer) and new format (with user, email, timestamp, type).
func NormalizeAnswer(answer DiscoveryAnswer) AttributedDiscoveryAnswer {
	return AttributedDiscoveryAnswer{
		QuestionID: answer.QuestionID,
		Answer:     answer.Answer,
		User:       "Unknown User",
		Email:      "",
		Timestamp:  "",
		Type:       "original",
	}
}

// NormalizeAttributedAnswer returns the attributed answer as-is (already normalized).
func NormalizeAttributedAnswer(answer AttributedDiscoveryAnswer) AttributedDiscoveryAnswer {
	return answer
}

// GetAnswersForQuestion returns all answers for a specific question, normalized.
func GetAnswersForQuestion(answers []DiscoveryAnswer, questionID string) []AttributedDiscoveryAnswer {
	var result []AttributedDiscoveryAnswer
	for _, a := range answers {
		if a.QuestionID == questionID {
			result = append(result, NormalizeAnswer(a))
		}
	}
	if result == nil {
		result = []AttributedDiscoveryAnswer{}
	}
	return result
}

// GetCombinedAnswer returns the combined answer text for a question (all contributors).
func GetCombinedAnswer(answers []DiscoveryAnswer, questionID string) string {
	qAnswers := GetAnswersForQuestion(answers, questionID)
	if len(qAnswers) == 0 {
		return ""
	}
	if len(qAnswers) == 1 {
		return qAnswers[0].Answer
	}
	result := ""
	for i, a := range qAnswers {
		if i > 0 {
			result += "\n\n"
		}
		result += a.Answer + " -- *" + a.User + "*"
	}
	return result
}

// =============================================================================
// Concern Definition (.tddmaster/concerns/*.json)
// =============================================================================

type ConcernExtra struct {
	QuestionID string `json:"questionId"`
	Text       string `json:"text"`
}

type ReviewDimensionScope string

const (
	ReviewDimensionScopeAll  ReviewDimensionScope = "all"
	ReviewDimensionScopeUI   ReviewDimensionScope = "ui"
	ReviewDimensionScopeAPI  ReviewDimensionScope = "api"
	ReviewDimensionScopeData ReviewDimensionScope = "data"
)

type ReviewDimension struct {
	ID               string               `json:"id"`
	Label            string               `json:"label"`
	Prompt           string               `json:"prompt"`
	EvidenceRequired bool                 `json:"evidenceRequired"`
	Scope            ReviewDimensionScope `json:"scope"`
}

type ConcernDefinition struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	Description        string            `json:"description"`
	Extras             []ConcernExtra    `json:"extras"`
	SpecSections       []string          `json:"specSections"`
	Reminders          []string          `json:"reminders"`
	AcceptanceCriteria []string          `json:"acceptanceCriteria"`
	ReviewDimensions   []ReviewDimension `json:"reviewDimensions,omitempty"`
	Registries         []string          `json:"registries,omitempty"`
	DreamStatePrompt   *string           `json:"dreamStatePrompt,omitempty"`
}

// UnmarshalJSON handles backward-compatible migration of OverrideTasks from
// []string (old format) to []SpecTask (new format). Old state files with
// string arrays are automatically upgraded on first read.
func (s *StateFile) UnmarshalJSON(data []byte) error {
	// Use a type alias to avoid infinite recursion during unmarshal.
	type StateFileAlias StateFile
	aux := &struct {
		OverrideTasks json.RawMessage `json:"overrideTasks,omitempty"`
		*StateFileAlias
	}{
		StateFileAlias: (*StateFileAlias)(s),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if len(aux.OverrideTasks) == 0 || string(aux.OverrideTasks) == "null" {
		s.OverrideTasks = nil
		return nil
	}
	// Try new format: []SpecTask
	var tasks []SpecTask
	if err := json.Unmarshal(aux.OverrideTasks, &tasks); err == nil {
		s.OverrideTasks = tasks
		return nil
	}
	// Fallback: old format []string — upgrade each entry to a SpecTask.
	var strs []string
	if err := json.Unmarshal(aux.OverrideTasks, &strs); err != nil {
		return fmt.Errorf("overrideTasks: cannot unmarshal as []SpecTask or []string: %w", err)
	}
	s.OverrideTasks = make([]SpecTask, len(strs))
	for i, title := range strs {
		s.OverrideTasks[i] = SpecTask{
			ID:        fmt.Sprintf("task-%d", i+1),
			Title:     title,
			Completed: false,
		}
	}
	return nil
}
