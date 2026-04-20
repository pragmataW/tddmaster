// Package model holds the pure data structures exchanged between `tddmaster next`
// and its callers. No I/O, no business logic — only shapes.
package model

// InteractionHints describes how the active coding tool presents options and delegates.
type InteractionHints struct {
	HasAskUserTool        bool   `json:"hasAskUserTool"`
	OptionPresentation    string `json:"optionPresentation"` // "tool" | "prose"
	HasSubAgentDelegation bool   `json:"hasSubAgentDelegation"`
	SubAgentMethod        string `json:"subAgentMethod"`  // "task" | "delegation" | "spawn" | "fleet" | "none"
	AskUserStrategy       string `json:"askUserStrategy"` // "ask_user_question" | "tddmaster_block"
}

// DefaultHints describes the Claude Code baseline: AskUserQuestion tool + Agent-based delegation.
var DefaultHints = InteractionHints{
	HasAskUserTool:        true,
	OptionPresentation:    "tool",
	HasSubAgentDelegation: true,
	SubAgentMethod:        "task",
	AskUserStrategy:       "ask_user_question",
}
