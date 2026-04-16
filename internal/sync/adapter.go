
// Package sync provides rule loading and tool syncing utilities for tddmaster.
package sync

import (
	"github.com/pragmataW/tddmaster/internal/state"
)

// =============================================================================
// Capabilities
// =============================================================================

// InteractionHints describes how a tool presents interactive choices and
// delegates to sub-agents.
type InteractionHints struct {
	// HasAskUserTool indicates whether the tool has an AskUserQuestion-style tool.
	HasAskUserTool bool `json:"hasAskUserTool"`
	// OptionPresentation: "tool" uses AskUserQuestion, "prose" uses numbered lists.
	OptionPresentation string `json:"optionPresentation"`
	// HasSubAgentDelegation indicates whether the tool can delegate work to sub-agents.
	HasSubAgentDelegation bool `json:"hasSubAgentDelegation"`
	// SubAgentMethod is the mechanism for spawning sub-agents.
	SubAgentMethod string `json:"subAgentMethod"`
}

// ToolCapabilities describes what a tool adapter can generate.
type ToolCapabilities struct {
	Rules       bool             `json:"rules"`
	Hooks       bool             `json:"hooks"`
	Agents      bool             `json:"agents"`
	Specs       bool             `json:"specs"`
	Mcp         bool             `json:"mcp"`
	Interaction InteractionHints `json:"interaction"`
}

// =============================================================================
// Context & Options
// =============================================================================

// SyncContext holds shared parameters every handler receives.
type SyncContext struct {
	Root          string
	Rules         []string
	CommandPrefix string
	// Manifest holds TDD-specific project settings (e.g. TestRunner).
	// May be nil when no manifest is present; adapters must handle nil gracefully.
	Manifest *state.Manifest
}

// SyncOptions holds tool-specific options (e.g. Claude Code's AllowGit).
type SyncOptions struct {
	AllowGit bool
}

// =============================================================================
// Adapter Interface
// =============================================================================

// ToolAdapter is the contract that every coding-tool adapter must satisfy.
type ToolAdapter interface {
	// ID returns which coding tool this adapter serves.
	ID() state.CodingToolId

	// Capabilities returns what this adapter is capable of generating.
	Capabilities() ToolCapabilities

	// SyncRules generates rule/instruction files (required — all tools produce these).
	SyncRules(ctx SyncContext, options *SyncOptions) error

	// SyncHooks generates hook configurations (optional).
	SyncHooks(ctx SyncContext, options *SyncOptions) error

	// SyncAgents generates agent configurations (optional).
	SyncAgents(ctx SyncContext, options *SyncOptions) error

	// SyncSpecs generates project-spec artifacts (optional).
	SyncSpecs(ctx SyncContext, specPath string) error

	// SyncMcp generates MCP configuration (optional).
	SyncMcp(ctx SyncContext) error
}
