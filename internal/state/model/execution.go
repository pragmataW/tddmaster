package model

const (
	TDDCycleRed      = "red"
	TDDCycleGreen    = "green"
	TDDCycleRefactor = "refactor"
)

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
	FailedACs             []string       `json:"failedACs,omitempty"`
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
	FromIteration         int        `json:"fromIteration"`
	UnaddressedIterations int        `json:"unaddressedIterations"`
}

type SpecTask struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Completed  bool     `json:"completed"`
	Covers     []string `json:"covers,omitempty"`
	TDDEnabled *bool    `json:"tddEnabled,omitempty"`
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
	TDDCycle             string              `json:"tddCycle,omitempty"  yaml:"tddCycle,omitempty"`
	RefactorRounds       int                 `json:"refactorRounds,omitempty"`
	RefactorApplied      bool                `json:"refactorApplied,omitempty"`
	PendingRefactorNotes []RefactorNote      `json:"pendingRefactorNotes,omitempty"`
}

type SpecState struct {
	Path   *string `json:"path"`
	Status string  `json:"status"`
}

type Decision struct {
	ID        string `json:"id"`
	Question  string `json:"question"`
	Choice    string `json:"choice"`
	Promoted  bool   `json:"promoted"`
	Timestamp string `json:"timestamp"`
}

type RevisitEntry struct {
	From           Phase    `json:"from"`
	Reason         string   `json:"reason"`
	CompletedTasks []string `json:"completedTasks"`
	Timestamp      string   `json:"timestamp"`
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
