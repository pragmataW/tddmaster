package model

// ClassificationOption is a single item the user may select in the classification prompt.
type ClassificationOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// ClassificationPrompt asks the user to classify the spec.
type ClassificationPrompt struct {
	Options     []ClassificationOption `json:"options"`
	Instruction string                 `json:"instruction"`
}

// SelfReview is a self-review checklist for the spec draft.
type SelfReview struct {
	Required    bool     `json:"required"`
	Checks      []string `json:"checks"`
	Instruction string   `json:"instruction"`
}

// SpecDraftOutput is the output for the SPEC_PROPOSAL phase.
type SpecDraftOutput struct {
	Phase                  string                `json:"phase"`
	Instruction            string                `json:"instruction"`
	SpecPath               string                `json:"specPath"`
	EdgeCases              []string              `json:"edgeCases,omitempty"`
	Transition             TransitionApprove     `json:"transition"`
	ClassificationRequired *bool                 `json:"classificationRequired,omitempty"`
	ClassificationPrompt   *ClassificationPrompt `json:"classificationPrompt,omitempty"`
	SelfReview             *SelfReview           `json:"selfReview,omitempty"`
	Saved                  *bool                 `json:"saved,omitempty"`
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

// TaskTDDSelectionOutput describes the per-task TDD selection sub-step shown
// after a spec is approved when spec-level TDD is enabled and the selection
// has not yet been made.
type TaskTDDSelectionOutput struct {
	Required    bool                    `json:"required"`
	Instruction string                  `json:"instruction"`
	Tasks       []TaskTDDSelectionEntry `json:"tasks"`
	Answers     TaskTDDSelectionAnswers `json:"answers"`
}

// ImportantTaskSelectionEntry describes a single task offered in the bulk
// "mark important?" review step.
type ImportantTaskSelectionEntry struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// ImportantTaskSelectionOutput describes the discovery bulk-mark sub-step
// shown after a spec is approved when the Important Task Gate is enabled and
// the user has not yet reviewed the task list.
type ImportantTaskSelectionOutput struct {
	Required    bool                          `json:"required"`
	Instruction string                        `json:"instruction"`
	Tasks       []ImportantTaskSelectionEntry `json:"tasks"`
	AnswerShape string                        `json:"answerShape"` // e.g. `{"importantTaskIds":["task-1","task-3"]}`
}

// SpecApprovedOutput is the output for the SPEC_APPROVED phase.
type SpecApprovedOutput struct {
	Phase                  string                        `json:"phase"`
	Instruction            string                        `json:"instruction"`
	SpecPath               string                        `json:"specPath"`
	Transition             TransitionStart               `json:"transition"`
	Saved                  *bool                         `json:"saved,omitempty"`
	TaskTDDSelection       *TaskTDDSelectionOutput       `json:"taskTDDSelection,omitempty"`
	ImportantTaskSelection *ImportantTaskSelectionOutput `json:"importantTaskSelection,omitempty"`
}
