package model

// ProgressTask is a single task entry in progress.json.
type ProgressTask struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// ProgressDecision mirrors the spec-level decision for the progress snapshot.
type ProgressDecision struct {
	Question string `json:"question"`
	Choice   string `json:"choice"`
	Promoted bool   `json:"promoted"`
}

// ProgressFile is the full progress.json document.
type ProgressFile struct {
	Spec      string             `json:"spec"`
	Status    string             `json:"status"`
	Tasks     []ProgressTask     `json:"tasks"`
	Decisions []ProgressDecision `json:"decisions"`
	Debt      []any              `json:"debt"`
	UpdatedAt string             `json:"updatedAt"`
}
