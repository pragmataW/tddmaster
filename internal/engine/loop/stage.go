package loop

import (
	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/rules"
	"github.com/pragmataW/tddmaster/internal/spec"
)

const (
	StageIDGate     = "gate"
	StageIDRed      = "red"
	StageIDGreen    = "green"
	StageIDRefactor = "refactor"
	StageIDExecutor = "executor"
	StageIDVerifier = "verifier"
)

type Stage interface {
	ID() string
	Applies(ctx ExecCtx) bool
	Prompt(ctx ExecCtx) engine.Action
	OnReport(ctx ExecCtx, report StageReport) (ExecCtx, error)
}

type ExecCtx struct {
	Settings          spec.Settings
	Task              spec.Task
	State             spec.ExecState
	TaskIdx           int
	MaxRefactorRounds int
	UserContext       string
	Rules             rules.Set
}

type TraceReportEntry struct {
	TestFilePath string   `json:"testFilePath"`
	FunctionName string   `json:"functionName"`
	TaskID       string   `json:"taskId"`
	AC           []string `json:"ac"`
	EC           []string `json:"ec"`
}

type RefactorNote = spec.RefactorNote

type StageReport struct {
	Passed             bool           `json:"passed"`
	Phase              string         `json:"phase"`
	FailedACs          []string       `json:"failedACs"`
	RefactorNotes      []RefactorNote `json:"refactorNotes,omitempty"`
	UncoveredEdgeCases []string       `json:"uncoveredEdgeCases"`
	Completed          []string       `json:"completed"`
	Blocked            []string       `json:"blocked"`
	FilesModified      []string       `json:"filesModified"`
	RefactorApplied    bool           `json:"refactorApplied"`
	Plan               *spec.TaskPlan `json:"plan,omitempty"`
	Accepted           bool           `json:"accepted"`
	PlanFeedback       string         `json:"planFeedback"`
	TestsWritten       []string           `json:"testsWritten"`
	Traceability       []TraceReportEntry `json:"traceability"`
	FileCoverage       []FileCoverageEntry `json:"fileCoverage,omitempty"`
}

func (r StageReport) RefactorNotesPresent() bool {
	return len(r.RefactorNotes) > 0
}

func (r StageReport) EffectivePassed() bool {
	return r.Passed && len(r.FailedACs) == 0 && len(r.UncoveredEdgeCases) == 0
}
