package spec

import (
	"strconv"
	"strings"
	"time"
)

type Criterion struct {
	ID    string `json:"id"`
	Given string `json:"given,omitempty"`
	When  string `json:"when,omitempty"`
	Then  string `json:"then"`
	Raw   string `json:"raw,omitempty"`
}

type TraceEntry struct {
	FunctionName string   `json:"functionName"`
	TaskID       string   `json:"taskId"`
	CriterionIDs []string `json:"criterionIds,omitempty"`
	EC           []string `json:"ec"`
}

type Traceability struct {
	Entries  map[string][]TraceEntry       `json:"entries"`
	Coverage map[string]map[string]float64 `json:"coverage,omitempty"`
}

type RefactorNote struct {
	File       string `json:"file"`
	Suggestion string `json:"suggestion"`
	Rationale  string `json:"rationale"`
}

const (
	PhaseInitial    = "spec-settings"
	StatusDraft     = "draft"
	StatusExecuting = "executing"
	StatusCompleted = "completed"
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
	RuleLearningEnabled      bool `json:"ruleLearningEnabled"`
}

type WorktreeRef struct {
	Path   string `json:"path"`
	Branch string `json:"branch"`
}

type ExecState struct {
	TDDCycle           string             `json:"tddCycle,omitempty"`
	Implemented        bool               `json:"implemented,omitempty"`
	RefactorRounds     int                `json:"refactorRounds,omitempty"`
	RefactorApplied    bool               `json:"refactorApplied,omitempty"`
	PlanApproved       bool               `json:"planApproved,omitempty"`
	PlanAttempts       int                `json:"planAttempts,omitempty"`
	PlanFeedback       string             `json:"planFeedback,omitempty"`
	Plan               *TaskPlan          `json:"plan,omitempty"`
	Worktree           *WorktreeRef       `json:"worktree,omitempty"`
	LastFailedACs      []string           `json:"lastFailedACs,omitempty"`
	LastUncoveredEC    []string           `json:"lastUncoveredEC,omitempty"`
	LastCoverage       map[string]float64 `json:"lastCoverage,omitempty"`
	LastModifiedFiles  []string           `json:"lastModifiedFiles,omitempty"`
	CoverageUnreported bool               `json:"coverageUnreported,omitempty"`
	RefactorNotes      []RefactorNote     `json:"refactorNotes,omitempty"`
}

type Progress struct {
	Spec       string    `json:"spec"`
	Status     string    `json:"status"`
	Tasks      []Task    `json:"tasks"`
	TaskSeq    int       `json:"taskSeq,omitempty"`
	UpdatedAt  time.Time `json:"updatedAt"`
	Iterations int       `json:"iterations,omitempty"`
}

type Task struct {
	ID              string         `json:"id"`
	Title           string         `json:"title"`
	Criteria        []Criterion    `json:"criteria,omitempty"`
	Done            bool           `json:"done"`
	TDDEnabled      bool           `json:"tddEnabled"`
	Important       bool           `json:"important"`
	EdgeCases       []string       `json:"edgeCases,omitempty"`
	RefactorNotes   []RefactorNote `json:"refactorNotes,omitempty"`
	FailedACReasons []string       `json:"failedAcReasons,omitempty"`
	DependsOn       []string       `json:"dependsOn,omitempty"`
	Blocked         bool           `json:"blocked,omitempty"`
	BlockedReason   string         `json:"blockedReason,omitempty"`
	Exec            *ExecState     `json:"exec,omitempty"`
}

type TaskPlan struct {
	TaskID         string   `json:"taskId"`
	Assumptions    []string `json:"assumptions"`
	TouchedFiles   []string `json:"touchedFiles"`
	DesignPatterns []string `json:"designPatterns"`
	BestPractices  []string `json:"bestPractices"`
	Approach       string   `json:"approach"`
}

const (
	CriterionIDPrefix = "ac-"
	TaskIDPrefix      = "task-"
)

func AssignCriterionIDs(t *Task) {
	const prefix = CriterionIDPrefix
	maxSuffix := 0
	for _, c := range t.Criteria {
		if !strings.HasPrefix(c.ID, prefix) {
			continue
		}
		n, err := strconv.Atoi(strings.TrimPrefix(c.ID, prefix))
		if err != nil {
			continue
		}
		if n > maxSuffix {
			maxSuffix = n
		}
	}
	for i := range t.Criteria {
		if t.Criteria[i].ID != "" {
			continue
		}
		maxSuffix++
		t.Criteria[i].ID = prefix + strconv.Itoa(maxSuffix)
	}
}

func DefaultSettings() Settings {
	return Settings{TDDEnabled: true, SkipVerifierEnabled: false, ImportantTaskGateEnabled: false, MinTestCoverage: 80, RuleLearningEnabled: false}
}

func (s *Settings) ClampCoverage() {
	if s.MinTestCoverage < 0 {
		s.MinTestCoverage = 0
	}
	if s.MinTestCoverage > 100 {
		s.MinTestCoverage = 100
	}
}
