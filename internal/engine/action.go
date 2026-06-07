package engine

type ActionType string

const (
	ActionAsk      ActionType = "ask"
	ActionInstruct ActionType = "instruct"
	ActionNotify   ActionType = "notify"
	ActionTerminal ActionType = "terminal"
	ActionError    ActionType = "error"
)

type InputFormat string

const (
	FormatJSON InputFormat = "json"
	FormatText InputFormat = "text"
	FormatFlag InputFormat = "flag"
)

type ExpectedInput struct {
	Format    InputFormat `json:"format,omitempty"`
	SubmitCmd string      `json:"submitCmd,omitempty"`
	Example   string      `json:"example,omitempty"`
}

type InteractiveOption struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

type Action struct {
	Action             ActionType          `json:"action"`
	Instruction        string              `json:"instruction"`
	MultiSelect        bool                `json:"multiSelect,omitempty"`
	CommandMap         map[string]string   `json:"commandMap,omitempty"`
	DelegateAgent      string              `json:"delegateAgent,omitempty"`
	ExpectedInput      ExpectedInput       `json:"expectedInput"`
	InteractiveOptions []InteractiveOption `json:"interactiveOptions,omitempty"`
}
