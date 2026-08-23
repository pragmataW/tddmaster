package loop

import (
	"github.com/pragmataW/tddmaster/internal/engine"
	"github.com/pragmataW/tddmaster/internal/projectcmds"
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

type SiblingTask struct {
	ID    string
	Title string
	Files []string
}

type TaskOverview struct {
	ID     string
	Title  string
	Status string
	Self   bool
}

type SpecIntent struct {
	StatusQuo          string
	Ambition           string
	Reversibility      string
	UserImpact         string
	ChallengedPremises []string
}

func (i SpecIntent) Empty() bool {
	return i.StatusQuo == "" && i.Ambition == "" && i.Reversibility == "" &&
		i.UserImpact == "" && len(i.ChallengedPremises) == 0
}

type ProjectCommand = projectcmds.Command

type TraceRef struct {
	TestFilePath string
	FunctionName string
	AC           []string
	EC           []string
}

type ExecCtx struct {
	Settings          spec.Settings
	Task              spec.Task
	State             spec.ExecState
	TaskIdx           int
	MaxRefactorRounds int
	Slug              string
	Mode              string
	Worktrees         bool
	UserContext       string
	ScopeBoundary     string
	VerificationHint  string
	ProjectCommands   []ProjectCommand
	Intent            SpecIntent
	Overview          []TaskOverview
	Trace             []TraceRef
	Siblings          []SiblingTask
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
	TaskID             string              `json:"taskId,omitempty"`
	Passed             bool                `json:"passed"`
	Phase              string              `json:"phase"`
	Reason             string              `json:"reason,omitempty"`
	Output             string              `json:"output,omitempty"`
	FailedACs          []string            `json:"failedACs"`
	RefactorNotes      []RefactorNote      `json:"refactorNotes,omitempty"`
	UncoveredEdgeCases []string            `json:"uncoveredEdgeCases"`
	Completed          []string            `json:"completed"`
	Blocked            []string            `json:"blocked"`
	FilesModified      []string            `json:"filesModified"`
	RefactorApplied    bool                `json:"refactorApplied"`
	Plan               *spec.TaskPlan      `json:"plan,omitempty"`
	Accepted           bool                `json:"accepted"`
	PlanFeedback       string              `json:"planFeedback"`
	TestsWritten       []string            `json:"testsWritten"`
	Traceability       []TraceReportEntry  `json:"traceability"`
	FileCoverage       []FileCoverageEntry `json:"fileCoverage,omitempty"`
}

func (r StageReport) RefactorNotesPresent() bool {
	return len(r.RefactorNotes) > 0
}

// RefactorRoundPassed decides a refactor apply round when no verifier follows it.
// The executor's own report is then the only signal available, so applying the
// notes without reporting a failure or a blocker counts as passing — `passed`
// may be absent from an executor report and must not silently stall the cycle.
func (r StageReport) RefactorRoundPassed() bool {
	if !r.RefactorApplied {
		return r.EffectivePassed()
	}
	return len(r.FailedACs) == 0 && len(r.UncoveredEdgeCases) == 0 && len(r.Blocked) == 0
}

func (r StageReport) HasStageResult() bool {
	return r.Passed || r.Phase != "" || r.RefactorApplied || r.Accepted ||
		r.Plan != nil || r.PlanFeedback != "" || r.Reason != "" || r.Output != "" ||
		len(r.Completed) > 0 || len(r.FailedACs) > 0 || len(r.UncoveredEdgeCases) > 0 ||
		len(r.TestsWritten) > 0 || len(r.Traceability) > 0 || len(r.FileCoverage) > 0 ||
		len(r.FilesModified) > 0 || len(r.RefactorNotes) > 0
}

func (r StageReport) HasGateAnswer() bool {
	return r.Accepted || r.Plan != nil || r.PlanFeedback != ""
}

func (r StageReport) EffectivePassed() bool {
	return r.Passed && len(r.FailedACs) == 0 && len(r.UncoveredEdgeCases) == 0
}
