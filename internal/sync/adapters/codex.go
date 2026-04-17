package adapters

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/pragmataW/tddmaster/internal/state"
	statesync "github.com/pragmataW/tddmaster/internal/sync"
	"github.com/pragmataW/tddmaster/internal/sync/adapters/shared"
)

// =============================================================================
// File Paths
// =============================================================================

const codexHooksDir = ".codex"
const codexHooksFile = "hooks.json"
const codexAgentsDir = ".codex/agents"
const codexConfigFile = ".codex/config.toml"

// =============================================================================
// CodexAdapter
// =============================================================================

// CodexAdapter implements ToolAdapter for OpenAI Codex CLI.
type CodexAdapter struct{}

func (a *CodexAdapter) ID() state.CodingToolId {
	return state.CodingToolCodex
}

func (a *CodexAdapter) Capabilities() statesync.ToolCapabilities {
	return statesync.ToolCapabilities{
		Rules:  true,
		Hooks:  true,
		Agents: true,
		Specs:  false,
		Mcp:    true,
		Interaction: statesync.InteractionHints{
			HasAskUserTool:        false,
			OptionPresentation:    "prose",
			HasSubAgentDelegation: true,
			SubAgentMethod:        "spawn",
			AskUserStrategy:       "tddmaster_block",
		},
	}
}

func (a *CodexAdapter) SyncRules(ctx statesync.SyncContext, options *statesync.SyncOptions) error {
	return shared.SyncAgentsMd(ctx, options)
}

func (a *CodexAdapter) SyncHooks(ctx statesync.SyncContext, _ *statesync.SyncOptions) error {
	hooksDir := filepath.Join(ctx.Root, codexHooksDir)
	hooksPath := filepath.Join(hooksDir, codexHooksFile)

	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return err
	}

	// Read existing hooks (preserve non-tddmaster hooks)
	var existingHooks []map[string]interface{}
	if data, err := os.ReadFile(hooksPath); err == nil {
		var parsed map[string]interface{}
		if json.Unmarshal(data, &parsed) == nil {
			if hooks, ok := parsed["hooks"]; ok {
				if hooksSlice, ok := hooks.([]interface{}); ok {
					for _, h := range hooksSlice {
						if hm, ok := h.(map[string]interface{}); ok {
							existingHooks = append(existingHooks, hm)
						}
					}
				}
			}
		}
	}

	// Filter out previous tddmaster-managed hooks
	var userHooks []map[string]interface{}
	for _, h := range existingHooks {
		if _, isNos := h["_tddmaster"]; !isNos {
			userHooks = append(userHooks, h)
		}
	}

	// Build fresh tddmaster hooks and merge
	tddmasterHooks := buildCodexHooksConfig(ctx.CommandPrefix)
	allHooks := append(userHooks, tddmasterHooks...) //nolint:gocritic

	merged := map[string]interface{}{"hooks": allHooks}
	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	return os.WriteFile(hooksPath, data, 0o644)
}

func (a *CodexAdapter) SyncAgents(ctx statesync.SyncContext, _ *statesync.SyncOptions) error {
	agentsDir := filepath.Join(ctx.Root, codexAgentsDir)
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		return err
	}

	if err := os.WriteFile(
		filepath.Join(agentsDir, "tddmaster-executor.toml"),
		[]byte(buildCodexExecutorAgentToml(ctx.CommandPrefix, ctx.Manifest)),
		0o644,
	); err != nil {
		return err
	}

	if err := os.WriteFile(
		filepath.Join(agentsDir, "tddmaster-verifier.toml"),
		[]byte(buildCodexVerifierAgentToml(ctx.Manifest)),
		0o644,
	); err != nil {
		return err
	}

	if ctx.Manifest != nil && ctx.Manifest.TddMode {
		return os.WriteFile(
			filepath.Join(agentsDir, "test-writer.toml"),
			[]byte(buildCodexTestWriterAgentToml()),
			0o644,
		)
	}
	return nil
}

func (a *CodexAdapter) SyncSpecs(_ statesync.SyncContext, _ string) error {
	return nil
}

func (a *CodexAdapter) SyncMcp(ctx statesync.SyncContext) error {
	configPath := filepath.Join(ctx.Root, codexConfigFile)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}

	existing := ""
	if data, err := os.ReadFile(configPath); err == nil {
		existing = string(data)
	}

	// Remove existing [mcp_servers.tddmaster] section if present, then append
	re := strings.NewReplacer() // dummy; we do manual regex-like replacement below
	_ = re

	// Simple removal of [mcp_servers.tddmaster] block
	cleaned := removeMcpTddmasterSection(existing)

	mcpSection := buildCodexMcpToml(ctx.CommandPrefix)
	var merged string
	if cleaned != "" {
		merged = strings.TrimRight(cleaned, "\n") + "\n\n" + mcpSection
	} else {
		merged = mcpSection
	}

	return os.WriteFile(configPath, []byte(merged), 0o644)
}

// removeMcpTddmasterSection removes [mcp_servers.tddmaster] block from TOML content.
func removeMcpTddmasterSection(content string) string {
	startMarker := "[mcp_servers.tddmaster]"
	startIdx := strings.Index(content, startMarker)
	if startIdx == -1 {
		return content
	}

	// Find the end: next [ section or end of string
	rest := content[startIdx+len(startMarker):]
	endIdx := strings.Index(rest, "\n[")
	if endIdx == -1 {
		return strings.TrimRight(content[:startIdx], "\n")
	}
	return strings.TrimRight(content[:startIdx], "\n") + rest[endIdx+1:]
}

// =============================================================================
// Content generators
// =============================================================================

func buildCodexHooksConfig(commandPrefix string) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"_tddmaster": true,
			"event":      "SessionStart",
			"command":    commandPrefix + " invoke-hook session-start",
			"timeout":    5000,
		},
		{
			"_tddmaster": true,
			"event":      "PreToolUse",
			"command":    commandPrefix + " invoke-hook pre-tool-use",
			"timeout":    5000,
		},
		{
			"_tddmaster": true,
			"event":      "PostToolUse",
			"command":    commandPrefix + " invoke-hook post-file-write",
			"timeout":    3000,
		},
		{
			"_tddmaster": true,
			"event":      "Stop",
			"command":    commandPrefix + " invoke-hook stop",
			"timeout":    10000,
		},
	}
}

func buildCodexExecutorAgentToml(commandPrefix string, manifest *state.Manifest) string {
	instructions := strings.Join([]string{
		"You are executing a single task from a tddmaster spec.",
		"Your ONLY job is to complete the task described in the prompt.",
		"Follow all behavioral rules provided in the prompt.",
		"Do NOT start new tasks, explore unrelated code, or make architectural decisions.",
		"If the task is too vague to execute, say so immediately.",
		"",
		"## TDD Context",
		"If `tddPhase` is present in your task context, follow RGR discipline:",
		"- `red` — write failing tests ONLY; do NOT write implementation code",
		"- `green` — implement minimum code to make failing tests pass",
		"- `refactor` — improve structure without changing behavior; tests must still pass",
		"If `tddFailureReport` is present, read `failedACs` and address them before anything else.",
		"Include `tddPhase` in your JSON report when it is set.",
		"",
		"## Refactor Mode",
		"If `refactorInstructions` is present in your task context, this is a REFACTOR round:",
		"1. Apply each note in `refactorInstructions.notes` verbatim.",
		"2. Do NOT change test behavior — the full test suite must still pass.",
		"3. Report `refactorApplied: true` in your JSON output.",
		"The verifier will re-run tests after your changes to confirm behavior is preserved.",
		"",
		"## Reporting",
		"When finished, provide a structured JSON summary:",
		`{"completed": ["<item IDs done>"], "remaining": ["<item IDs not done>"],`,
		` "blocked": ["<item IDs needing decisions>"], "filesModified": ["<paths>"],`,
		` "tddPhase": "<phase or omit>", "refactorApplied": true|false}`,
		"The orchestrator will submit this to `" + commandPrefix + " next --answer` on your behalf.",
	}, "\n")

	return strings.Join([]string{
		`name = "tddmaster-executor"`,
		`description = "Executes a single tddmaster task. Follows spec behavioral rules and reports structured results."`,
		`developer_instructions = """`,
		instructions,
		`"""`,
		"",
	}, "\n")
}

func buildCodexVerifierAgentToml(manifest *state.Manifest) string {
	typeCheckCmd, testCmd := resolveVerifierCommands(manifest)

	// TDD mode guard: only inject RED/GREEN/REFACTOR phase blocks when TDD is enabled.
	// Non-TDD projects must use VerifierInstructions — sending TDD phase blocks to a
	// non-TDD verifier causes confusion and incorrect phase-specific behavior.
	tddMode := manifest != nil && manifest.TddMode
	var baseInstructions string
	if tddMode {
		baseInstructions = shared.VerifierInstructionsAllPhases(typeCheckCmd, testCmd)
	} else {
		baseInstructions = shared.VerifierInstructions(typeCheckCmd, testCmd)
	}
	instructions := baseInstructions + "\n\nThe orchestrator will use this report for the tddmaster status report."

	return strings.Join([]string{
		`name = "tddmaster-verifier"`,
		`description = "Independently verifies completed task work. Read-only. Never sees the executor's context."`,
		`developer_instructions = """`,
		instructions,
		`"""`,
		"",
	}, "\n")
}

func buildCodexTestWriterAgentToml() string {
	instructions := shared.TestWriterInstructions("the `AGENTS.md` file")

	return strings.Join([]string{
		`name = "test-writer"`,
		`description = "Writes tests FIRST following TDD principles. Reads the project rule set from AGENTS.md before writing any test."`,
		`developer_instructions = """`,
		instructions,
		`"""`,
		"",
	}, "\n")
}

func buildCodexMcpToml(commandPrefix string) string {
	parts := strings.Fields(commandPrefix)
	command := "npx"
	if len(parts) > 0 {
		command = parts[0]
	}

	argParts := append(parts[1:], "mcp-serve") //nolint:gocritic
	args := make([]string, len(argParts))
	for i, a := range argParts {
		args[i] = `"` + a + `"`
	}

	return `[mcp_servers.tddmaster]` + "\n" +
		`command = "` + command + `"` + "\n" +
		`args = [` + strings.Join(args, ", ") + `]` + "\n"
}
