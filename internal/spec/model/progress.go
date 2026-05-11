package model

// ProgressTask is a single task entry in progress.json.
type ProgressTask struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	Important bool   `json:"important,omitempty"`
}

// ProgressDecision mirrors the spec-level decision for the progress snapshot.
type ProgressDecision struct {
	Question string `json:"question"`
	Choice   string `json:"choice"`
	Promoted bool   `json:"promoted"`
}

// ProgressTaskPlan is the user-approved implementation plan for an
// "important"-flagged task. Produced by the tddmaster-planner subagent,
// reviewed by the user, then persisted under ProgressFile.TaskPlans so the
// executor receives it for every TDD phase of the task.
type ProgressTaskPlan struct {
	TaskID         string   `json:"taskId"`
	Assumptions    []string `json:"assumptions"`
	TouchedFiles   []string `json:"touchedFiles"`
	DesignPatterns []string `json:"designPatterns"`
	BestPractices  []string `json:"bestPractices"`
	Approach       string   `json:"approach"`
	AttemptCount   int      `json:"attemptCount"`
	ApprovedAt     string   `json:"approvedAt"`
	ApprovedBy     string   `json:"approvedBy"`
}

// ProgressFile is the full progress.json document.
type ProgressFile struct {
	Spec      string             `json:"spec"`
	Status    string             `json:"status"`
	Tasks     []ProgressTask     `json:"tasks"`
	Decisions []ProgressDecision `json:"decisions"`
	Debt      []any              `json:"debt"`
	TaskPlans []ProgressTaskPlan `json:"taskPlans,omitempty"`
	UpdatedAt string             `json:"updatedAt"`
}
