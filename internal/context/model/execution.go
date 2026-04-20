package model

import "github.com/pragmataW/tddmaster/internal/state"

// AcceptanceCriterion is a single AC in the status report.
type AcceptanceCriterion struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// ReportFormat describes the shape of a status report submission.
type ReportFormat struct {
	Completed string `json:"completed"`
	Remaining string `json:"remaining"`
	Blocked   string `json:"blocked"`
	NA        string `json:"na"`
	NewIssues string `json:"newIssues"`
}

// StatusReportRequest asks the agent to report AC status.
type StatusReportRequest struct {
	Criteria     []AcceptanceCriterion `json:"criteria"`
	ReportFormat ReportFormat          `json:"reportFormat"`
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

// DesignChecklist is the design checklist for the beautiful-product concern.
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
	Phase                  string                  `json:"phase"`
	Instruction            string                  `json:"instruction"`
	Task                   *TaskBlock              `json:"task,omitempty"`
	BatchTasks             []string                `json:"batchTasks,omitempty"`
	EdgeCases              []string                `json:"edgeCases,omitempty"`
	Context                ContextBlock            `json:"context"`
	Transition             TransitionExecution     `json:"transition"`
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
	VerifierRequired       bool                    `json:"verifierRequired"`
}
