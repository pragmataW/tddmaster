package spec

import "time"

type TraceEntry struct {
	FunctionName string   `json:"functionName"`
	TaskID       string   `json:"taskId"`
	AC           []string `json:"ac"`
	EC           []string `json:"ec"`
}

type Traceability struct {
	Entries  map[string][]TraceEntry `json:"entries"`
	Coverage map[string]int          `json:"coverage,omitempty"`
}

const (
	PhaseInitial     = "spec-settings"
	StatusDraft      = "draft"
	StatusExecuting  = "executing"
	StatusCompleted  = "completed"
)

type State struct {
	Version   int                 `json:"version"`
	Slug      string              `json:"slug"`
	Phase     string              `json:"phase"`
	Answers   map[string][]Answer `json:"answers"`
	CreatedAt time.Time           `json:"createdAt"`
	UpdatedAt time.Time           `json:"updatedAt"`
}

type Answer struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Settings struct {
	TDDEnabled               bool `json:"tddEnabled"`
	SkipVerifierEnabled      bool `json:"skipVerifierEnabled"`
	ImportantTaskGateEnabled bool `json:"importantTaskGateEnabled"`
	MinTestCoverage          int  `json:"minTestCoverage"`
}

type ExecState struct {
	Iteration       int               `json:"iteration"`
	TDDCycle        string            `json:"tddCycle,omitempty"`
	Implemented     bool              `json:"implemented,omitempty"`
	RefactorRounds  int               `json:"refactorRounds,omitempty"`
	RefactorApplied bool              `json:"refactorApplied,omitempty"`
	ApprovedPlans   []string          `json:"approvedPlans,omitempty"`
	PlanAttempts    map[string]int    `json:"planAttempts,omitempty"`
	PlanFeedback    map[string]string `json:"planFeedback,omitempty"`
	TaskPlans       map[string]TaskPlan `json:"taskPlans,omitempty"`
	LastFailedACs     []string          `json:"lastFailedACs,omitempty"`
	LastUncoveredEC   []string          `json:"lastUncoveredEC,omitempty"`
	LastCoverage      map[string]int    `json:"lastCoverage,omitempty"`
	LastModifiedFiles []string          `json:"lastModifiedFiles,omitempty"`
	CoverageUnreported bool             `json:"coverageUnreported,omitempty"`
}

type Progress struct {
	Spec      string     `json:"spec"`
	Status    string     `json:"status"`
	Tasks     []Task     `json:"tasks"`
	UpdatedAt time.Time  `json:"updatedAt"`
	Execution *ExecState `json:"execution,omitempty"`
}

type Task struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	AC         []string `json:"ac"`
	Done       bool     `json:"done"`
	TDDEnabled bool     `json:"tddEnabled"`
	Important  bool     `json:"important"`
	EdgeCases  []string `json:"edgeCases,omitempty"`
}

type TaskPlan struct {
	TaskID         string   `json:"taskId"`
	Assumptions    []string `json:"assumptions"`
	TouchedFiles   []string `json:"touchedFiles"`
	DesignPatterns []string `json:"designPatterns"`
	BestPractices  []string `json:"bestPractices"`
	Approach       string   `json:"approach"`
}

func DefaultSettings() Settings {
	return Settings{TDDEnabled: true, SkipVerifierEnabled: false, ImportantTaskGateEnabled: false, MinTestCoverage: 80}
}

func (s *Settings) ClampCoverage() {
	if s.MinTestCoverage < 0 {
		s.MinTestCoverage = 0
	}
	if s.MinTestCoverage > 100 {
		s.MinTestCoverage = 100
	}
}
