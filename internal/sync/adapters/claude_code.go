// Package adapters provides coding-tool adapter implementations for tddmaster sync.
package adapters

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pragmataW/tddmaster/internal/state"
	statesync "github.com/pragmataW/tddmaster/internal/sync"
	"github.com/pragmataW/tddmaster/internal/sync/adapters/shared"
)

// =============================================================================
// Adapter
// =============================================================================

// ClaudeCodeAdapter implements ToolAdapter for Claude Code.
type ClaudeCodeAdapter struct{}

func (a *ClaudeCodeAdapter) ID() state.CodingToolId {
	return state.CodingToolClaudeCode
}

func (a *ClaudeCodeAdapter) Capabilities() statesync.ToolCapabilities {
	return statesync.ToolCapabilities{
		Rules:  true,
		Hooks:  true,
		Agents: true,
		Specs:  false,
		Mcp:    false,
		Interaction: statesync.InteractionHints{
			HasAskUserTool:        true,
			OptionPresentation:    "tool",
			HasSubAgentDelegation: true,
			SubAgentMethod:        "task",
			AskUserStrategy:       "ask_user_question",
		},
	}
}

func (a *ClaudeCodeAdapter) SyncRules(ctx statesync.SyncContext, options *statesync.SyncOptions) error {
	return syncClaudeMd(ctx.Root, ctx.Rules, options, ctx.CommandPrefix)
}

func (a *ClaudeCodeAdapter) SyncHooks(ctx statesync.SyncContext, _ *statesync.SyncOptions) error {
	return statesync.SyncHooks(ctx.Root, ctx.CommandPrefix)
}

func (a *ClaudeCodeAdapter) SyncAgents(ctx statesync.SyncContext, _ *statesync.SyncOptions) error {
	if err := generateAgentFile(ctx.Root, ctx.CommandPrefix, ctx.Rules, ctx.Manifest); err != nil {
		return err
	}
	if err := generateVerifierFile(ctx.Root, ctx.Rules, ctx.Manifest); err != nil {
		return err
	}
	if ctx.Manifest != nil && ctx.Manifest.TddMode {
		return generateClaudeCodeTestWriterFile(ctx.Root, ctx.Rules, ctx.Manifest)
	}
	return nil
}

func (a *ClaudeCodeAdapter) SyncSpecs(_ statesync.SyncContext, _ string) error {
	return nil // not supported
}

func (a *ClaudeCodeAdapter) SyncMcp(_ statesync.SyncContext) error {
	return nil // not supported
}

// =============================================================================
// CLAUDE.md generation
// =============================================================================

func claudeCodeConventionSources() shared.ConventionSources {
	return shared.ConventionSources{
		ProjectFile: "CLAUDE.md",
		HomeFile:    "~/.claude/CLAUDE.md",
	}
}

func buildClaudeSection(rules []string, options *statesync.SyncOptions, commandPrefix string) string {
	allowGit := false
	if options != nil {
		allowGit = options.AllowGit
	}

	lines := []string{
		shared.NosStart,
		"## tddmaster orchestrator",
		"",
		"State-driven orchestration. Do NOT read `.tddmaster/` files directly — tddmaster provides everything via JSON.",
		"",
		"### Protocol",
		"",
		"    " + commandPrefix + " spec <name> next                           # get instruction",
		"    " + commandPrefix + ` spec <name> next --answer="response"       # submit and advance`,
		"    " + commandPrefix + ` spec new "description"                     # create spec (name auto-generated)`,
		"",
		"Every spec command MUST include `spec <name>`. Use `" + commandPrefix + " spec list` for available specs.",
		"",
		"### Core rules",
		"",
		"- Call tddmaster ONCE per interaction. One question, one answer, one submit.",
		"- Call `next` at: conversation start, before file edits, after completing work, at decisions.",
		"- Never batch-submit. Never answer discovery questions yourself.",
		"- Never skip steps or infer decisions. Ask first. Explicit > Clever.",
		"- NEVER suggest bypassing or skipping tddmaster. Discovery is not overhead.",
		"- NEVER ask permission to run the next tddmaster command. After spec new → run next. After approve → run next. Each step has one next step. Just run it.",
		"- Execute tddmaster commands IMMEDIATELY — the output has all context needed.",
		"- Display `roadmap` before content. Display `gate` prominently.",
		"",
		"### Task recovery",
		"",
		"If a task was marked completed by mistake (wrong `completed: [...]` in status report,",
		"or user changed their mind):",
		"",
		"    " + commandPrefix + " undo                         # reset the most recently completed task",
		"    " + commandPrefix + " spec <name> task undo <id>   # reset a specific task by ID",
		"",
		"`undo` only flips the task flag — it does NOT rewind `next` iteration or TDD phase.",
		"If you accidentally ended the whole spec, use `" + commandPrefix + " spec <name> reopen --resume-execution`.",
		"Plain `reopen` returns to DISCOVERY for revision; `--resume-execution` restores EXECUTING/BLOCKED progress.",
		"`done` and `cancel` act on the ENTIRE spec, not a single task — never call them to \"undo a task\".",
		"",
		"### Interactive choices",
		"",
		"- Use AskUserQuestion for `interactiveOptions`. Use `commandMap` to resolve selections.",
		"- Listen-first step (mode not yet chosen, no user context): ask the single open-ended context question via AskUserQuestion with a free-form prompt. Do NOT fabricate options.",
		"- Premise challenge step: ask each premise via a separate AskUserQuestion call (agree / disagree / revise with free-form notes). Aggregate the answers client-side, then submit the final JSON payload.",
		"- On recurring patterns or corrections: ask 'Permanent rule?' → `" + commandPrefix + ` rule add "description"` + "``.",
	}

	if !allowGit {
		lines = append(lines,
			"",
			"### Git",
			"",
			"Read-only: log, diff, status, show, blame. No write commands (commit, push, checkout, etc.).",
		)
	}

	lines = append(lines,
		"",
		"### Discovery",
		"",
		"Listen first: after spec creation, ask user to share context before mode selection.",
		"Modes: full (default), validate, technical-depth, ship-fast, explore.",
		"Pre-scan codebase before questions. Challenge premises. Propose alternatives.",
		"",
		"### Execution",
		"",
		"- Re-read files before and after editing. Files >500 LOC: read in chunks.",
		"- Run type-check + lint after every edit. Never mark AC passed if type-check fails.",
		"- If search returns few results, re-run narrower — assume truncation.",
		"- Clean dead code before structural refactors on files >300 LOC.",
		"- Complete the spec — no mid-execution pauses or checkpoints.",
		"- `meta` block has resume context for session start or after compaction.",
		"",
		"### TDD Protocol",
		"",
		"When `TDDEnabled` is `true`, tddmaster enforces Red-Green-Refactor cycles.",
		"The executor context includes `tddPhase` (`red`|`green`|`refactor`|`\"\"`), `tddVerificationContext`, and `tddFailureReport`.",
		"- `red` — write failing tests ONLY; do NOT write implementation code",
		"- `green` — implement clean, working code that makes the failing tests pass (do not artificially minimise the solution)",
		"- `refactor` — improve structure without changing behavior",
		"Include `tddPhase` in status reports. When `tddFailureReport` is present, address `failedACs` before reporting done.",
	)

	if len(rules) > 0 {
		lines = append(lines, "", "### Active Rules", "")
		for _, rule := range rules {
			if strings.ContainsRune(rule, '\n') {
				lines = append(lines, rule, "")
			} else {
				lines = append(lines, "- "+rule)
			}
		}
	}

	lines = append(lines, shared.NosEnd)

	return strings.Join(lines, "\n")
}

func syncClaudeMd(root string, rules []string, options *statesync.SyncOptions, commandPrefix string) error {
	if commandPrefix == "" {
		commandPrefix = "tddmaster"
	}

	filePath := filepath.Join(root, "CLAUDE.md")
	section := buildClaudeSection(rules, options, commandPrefix)

	var content string
	if data, err := os.ReadFile(filePath); err == nil {
		content = string(data)
		startIdx := strings.Index(content, shared.NosStart)
		endIdx := strings.Index(content, shared.NosEnd)
		if startIdx != -1 && endIdx != -1 {
			content = content[:startIdx] + section + content[endIdx+len(shared.NosEnd):]
		} else {
			content = strings.TrimRight(content, "\n") + "\n\n" + section + "\n"
		}
	} else {
		content = section + "\n"
	}

	return os.WriteFile(filePath, []byte(content), 0o644)
}

// =============================================================================
// Agent file generation
// =============================================================================

// resolveVerifierCommands returns the type-check and test commands for agent
// templates. When manifest is nil or manifest.TestRunner is nil, it defaults to
// "deno check" and "deno test" for backward compatibility.
func resolveVerifierCommands(manifest *state.Manifest) (typeCheckCmd, testCmd string) {
	if manifest != nil && manifest.TestRunner != nil {
		runner := *manifest.TestRunner
		return runner, runner
	}
	return "deno check", "deno test"
}

func generateAgentFile(root, commandPrefix string, rules []string, manifest *state.Manifest) error {
	agentDir := filepath.Join(root, ".claude", "agents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return err
	}

	preamble := shared.ConventionsPreamble(root, claudeCodeConventionSources(), rules, manifest.ShouldInjectConventions())

	lines := []string{
		"---",
		"name: tddmaster-executor",
		`description: "Executes a single tddmaster task."`,
		"tools: Read, Edit, MultiEdit, Write, Bash, Grep, Glob, LS",
		"model: sonnet",
		"---",
		"",
		preamble + shared.ExecutorInstructions(commandPrefix),
	}
	content := strings.Join(lines, "\n") + "\n"

	return os.WriteFile(filepath.Join(agentDir, "tddmaster-executor.md"), []byte(content), 0o644)
}

func generateClaudeCodeTestWriterFile(root string, rules []string, manifest *state.Manifest) error {
	agentDir := filepath.Join(root, ".claude", "agents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return err
	}

	preamble := shared.ConventionsPreamble(root, claudeCodeConventionSources(), rules, manifest.ShouldInjectConventions())

	lines := []string{
		"---",
		"name: test-writer",
		`description: "Writes tests FIRST following TDD principles. Reads the project rule set from .claude/rules/ before writing any test."`,
		"tools: Read, Edit, MultiEdit, Write, Bash, Grep, Glob, LS",
		"model: sonnet",
		"---",
		"",
		preamble + shared.TestWriterInstructions("`.claude/rules/`"),
		"",
	}
	content := strings.Join(lines, "\n")

	return os.WriteFile(filepath.Join(agentDir, "test-writer.md"), []byte(content), 0o644)
}

func generateVerifierFile(root string, rules []string, manifest *state.Manifest) error {
	agentDir := filepath.Join(root, ".claude", "agents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return err
	}

	typeCheckCmd, testCmd := resolveVerifierCommands(manifest)

	tddMode := manifest != nil && manifest.TddMode
	skipVerify := manifest != nil && manifest.SkipVerify
	var verifierInstructions string
	if tddMode {
		verifierInstructions = shared.VerifierInstructionsAllPhases(typeCheckCmd, testCmd, skipVerify)
	} else {
		verifierInstructions = shared.VerifierInstructions(typeCheckCmd, testCmd)
	}

	preamble := shared.ConventionsPreamble(root, claudeCodeConventionSources(), rules, manifest.ShouldInjectConventions())

	lines := []string{
		"---",
		"name: tddmaster-verifier",
		`description: "Independently verifies completed task work. Read-only. Never sees the executor's context."`,
		"tools: Read, Bash, Grep, Glob, LS",
		"model: sonnet",
		"---",
		"",
		preamble + verifierInstructions,
		"",
		"The orchestrator will use your report for the tddmaster status report.",
		"",
	}

	return os.WriteFile(filepath.Join(agentDir, "tddmaster-verifier.md"), []byte(strings.Join(lines, "\n")), 0o644)
}
