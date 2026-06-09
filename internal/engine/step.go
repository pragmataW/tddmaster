package engine

import "encoding/json"

type StepID string

type RunResult struct {
	Done         bool
	Contribution json.RawMessage
	Action       Action
}

type StepDef struct {
	ID       StepID
	Prompt   func(*Context) Action
	Validate func(answer []byte) error
	Emit     func(answer []byte) error
	Run      func(*Context, []byte) (RunResult, error)
}

type StepProgress struct {
	Step     StepID          `json:"step"`
	Answered bool            `json:"answered"`
	Answer   json.RawMessage `json:"answer,omitempty"`
}
