
// Package output provides CLI output formatting and command prefix utilities.
//
// Port of tddmaster/output/mode.ts.
package output

import "os"

// =============================================================================
// Types
// =============================================================================

// Mode represents the audience mode: agent (Claude Code, Codex, OpenCode) or human.
type Mode string

const (
	ModeAgent Mode = "agent"
	ModeHuman Mode = "human"
)

// Interaction represents whether the session is interactive (TTY) or non-interactive (piped, CI).
type Interaction string

const (
	InteractionInteractive    Interaction = "interactive"
	InteractionNonInteractive Interaction = "non-interactive"
)

// ModeConfig is a minimal interface for manifest config used by DetectMode.
// Mirrors the relevant fields from schema.NosManifest / state.StateFile.
type ModeConfig interface {
	GetAgentMode() *bool
}

// =============================================================================
// Detection
// =============================================================================

// DetectMode detects audience from args, config, and environment.
//
//  1. Explicit --agent / --human flag in args
//  2. Persisted agentMode in manifest config
//  3. Environment detection (CLAUDE_CODE, CURSOR_* env vars, CI, etc.)
func DetectMode(args []string, config ModeConfig) Mode {
	// 1. Explicit flag
	for _, arg := range args {
		if arg == "--agent" {
			return ModeAgent
		}
		if arg == "--human" {
			return ModeHuman
		}
	}

	// 2. Persisted in manifest
	if config != nil {
		if m := config.GetAgentMode(); m != nil {
			if *m {
				return ModeAgent
			}
			return ModeHuman
		}
	}

	// 3. Environment detection
	return detectAudienceFromEnv()
}

// DetectInteraction detects whether the session is interactive from args and environment.
func DetectInteraction(args []string) Interaction {
	for _, arg := range args {
		if arg == "--non-interactive" {
			return InteractionNonInteractive
		}
	}

	return detectInteractionFromEnv()
}

// StripModeFlag removes mode-related flags (--agent, --human, --non-interactive) from args.
func StripModeFlag(args []string) []string {
	if args == nil {
		return []string{}
	}

	result := make([]string, 0, len(args))
	for _, a := range args {
		if a != "--agent" && a != "--human" && a != "--non-interactive" {
			result = append(result, a)
		}
	}
	return result
}

// =============================================================================
// Environment-based detection (mirrors @eser/shell/env)
// =============================================================================

// detectAudienceFromEnv detects agent vs human from environment variables.
// Checks common agent environment variables set by Claude Code, Codex, OpenCode, etc.
func detectAudienceFromEnv() Mode {
	// Claude Code sets CLAUDE_CODE environment variable
	if os.Getenv("CLAUDE_CODE") != "" {
		return ModeAgent
	}
	// Generic agent marker
	if os.Getenv("AGENT_MODE") != "" {
		return ModeAgent
	}
	// CI environments are typically non-interactive but not necessarily "agent"
	// Default to human for terminal usage
	return ModeHuman
}

// detectInteractionFromEnv detects interactive vs non-interactive from environment.
func detectInteractionFromEnv() Interaction {
	// Standard CI environment variables
	if os.Getenv("CI") != "" {
		return InteractionNonInteractive
	}
	// Check if stdout is a TTY — non-TTY means piped/non-interactive
	fi, err := os.Stdout.Stat()
	if err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
		return InteractionInteractive
	}
	return InteractionNonInteractive
}
