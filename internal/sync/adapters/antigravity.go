package adapters

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pragmataW/tddmaster/internal/state"
	statesync "github.com/pragmataW/tddmaster/internal/sync"
	"github.com/pragmataW/tddmaster/internal/sync/adapters/shared"
)

// =============================================================================
// Adapter
// =============================================================================

// AntigravityAdapter implements ToolAdapter for Google Antigravity CLI.
type AntigravityAdapter struct{}

func (a *AntigravityAdapter) ID() state.CodingToolId {
	return state.CodingToolAntigravity
}

func (a *AntigravityAdapter) Capabilities() statesync.ToolCapabilities {
	return statesync.ToolCapabilities{
		Rules:  true,
		Hooks:  true,
		Agents: true,
		Specs:  true,
		Mcp:    false,
		Interaction: statesync.InteractionHints{
			HasAskUserTool:        true,
			OptionPresentation:    "tool",
			HasSubAgentDelegation: true,
			SubAgentMethod:        "invoke_subagent",
			AskUserStrategy:       "ask_user_question",
		},
	}
}

func (a *AntigravityAdapter) SyncRules(ctx statesync.SyncContext, options *statesync.SyncOptions) error {
	return syncAntigravityMd(ctx.Root, ctx.Rules, options, ctx.CommandPrefix)
}

func (a *AntigravityAdapter) SyncHooks(ctx statesync.SyncContext, _ *statesync.SyncOptions) error {
	hooksDir := filepath.Join(ctx.Root, ".agents")
	hooksPath := filepath.Join(hooksDir, "hooks.json")

	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return err
	}

	// Read existing hooks (preserve non-tddmaster hooks)
	existingHooks := make(map[string]interface{})
	if data, err := os.ReadFile(hooksPath); err == nil {
		_ = json.Unmarshal(data, &existingHooks)
	}

	// Overwrite tddmaster hooks block
	existingHooks["tddmaster"] = buildAntigravityHooksConfig(ctx.CommandPrefix)

	data, err := json.MarshalIndent(existingHooks, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	return os.WriteFile(hooksPath, data, 0o644)
}

func (a *AntigravityAdapter) SyncAgents(ctx statesync.SyncContext, _ *statesync.SyncOptions) error {
	if err := generateAntigravityAgentFile(ctx.Root, ctx.CommandPrefix, ctx.Rules, ctx.Manifest); err != nil {
		return err
	}
	if err := generateAntigravityVerifierFile(ctx.Root, ctx.Rules, ctx.Manifest); err != nil {
		return err
	}
	if ctx.NosManifest != nil && ctx.NosManifest.IsImportantTaskGateEnabled() {
		if err := generateAntigravityPlannerFile(ctx.Root, ctx.CommandPrefix, ctx.Rules, ctx.Manifest); err != nil {
			return err
		}
	}
	if ctx.Manifest != nil && ctx.Manifest.TddMode {
		return generateAntigravityTestWriterFile(ctx.Root, ctx.Rules, ctx.Manifest)
	}
	return nil
}

func (a *AntigravityAdapter) SyncSpecs(ctx statesync.SyncContext, specPath string) error {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil // skip gracefully
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return nil
	}

	segments := strings.Split(filepath.ToSlash(specPath), "/")
	specName := "unknown"
	if len(segments) >= 2 {
		specName = segments[len(segments)-2]
	}

	parsed := parseAntigravitySpecMd(content)

	skillsDir := filepath.Join(ctx.Root, ".agents", "skills")
	specSkillDir := filepath.Join(skillsDir, specName)
	if err := os.MkdirAll(specSkillDir, 0o755); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(specSkillDir, "SKILL.md"), []byte(buildAntigravitySkillMd(parsed)), 0o644)
}

func (a *AntigravityAdapter) SyncMcp(_ statesync.SyncContext) error {
	return nil // not supported
}

// =============================================================================
// AGENTS.md generation
// =============================================================================

func antigravityConventionSources() shared.ConventionSources {
	return shared.ConventionSources{
		ProjectFile: "AGENTS.md",
		HomeFile:    "~/.gemini/GEMINI.md",
	}
}

func buildAntigravitySection(rules []string, options *statesync.SyncOptions, commandPrefix string) string {
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
		"### Important task gate",
		"",
		"Optional gate behind manifest `tddmaster.importantTaskGate`. When enabled, tasks flagged `important` pause before execution for a plan-first review by the `tddmaster-planner` subagent.",
		"",
		"Mark / unmark tasks (also offered as a bulk multiSelect during SPEC_APPROVED when the gate is on):",
		"",
		"    " + commandPrefix + " spec <name> task mark-important <id>",
		"    " + commandPrefix + " spec <name> task unmark-important <id>",
		"",
		"When the active task is important and has no approved plan, `next` returns an `importantTaskGate` block with `delegateAgent: \"tddmaster-planner\"`. Required behavior:",
		"",
		"- Invoke `tddmaster-planner` (read-only: Read/Grep/Glob/LS) with task scope plus `priorFeedback` when present. Do NOT edit files in this phase.",
		"- Present planner output via AskUserQuestion with options `accept | revise | reject`.",
		"- On accept: submit `" + commandPrefix + ` spec <name> next --answer='{"plan":{...},"accepted":true}'` + "`.",
		"- On revise/reject: submit `" + commandPrefix + ` spec <name> next --answer='{"planFeedback":"<reason>"}'` + "` — gate re-fires with `priorFeedback` populated and `attemptCount` incremented. Address every point in the next planner spawn.",
		"",
		"Once approved, the plan is persisted to `progress.json` (`taskPlans[]`) and embedded as `approvedPlan` in every subsequent executor spawn for that task. `touchedFiles` is binding — if work needs a file outside the list, STOP and report.",
		"",
		"Toggle outside init/sync:",
		"",
		"    " + commandPrefix + " config important-gate on|off|status",
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

func syncAntigravityMd(root string, rules []string, options *statesync.SyncOptions, commandPrefix string) error {
	if commandPrefix == "" {
		commandPrefix = "tddmaster"
	}

	filePath := filepath.Join(root, "AGENTS.md")
	section := buildAntigravitySection(rules, options, commandPrefix)

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
// Agent files generation
// =============================================================================

func buildAntigravityToolsList() string {
	return "run_command, view_file, replace_file_content, multi_replace_file_content, write_to_file, list_dir, grep_search, ask_question, ask_permission, invoke_subagent, send_message"
}

func generateAntigravityAgentFile(root, commandPrefix string, rules []string, manifest *state.Manifest) error {
	agentDir := filepath.Join(root, ".agents", "agents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return err
	}

	preamble := shared.ConventionsPreamble(root, antigravityConventionSources(), rules, manifest.ShouldInjectConventions())

	lines := []string{
		"---",
		"name: tddmaster-executor",
		`description: "Executes a single tddmaster task."`,
		"tools: " + buildAntigravityToolsList(),
		"model: gemini-2.0-flash",
		"---",
		"",
		preamble + shared.ExecutorInstructions(commandPrefix),
	}
	content := strings.Join(lines, "\n") + "\n"

	return os.WriteFile(filepath.Join(agentDir, "tddmaster-executor.md"), []byte(content), 0o644)
}

func generateAntigravityTestWriterFile(root string, rules []string, manifest *state.Manifest) error {
	agentDir := filepath.Join(root, ".agents", "agents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return err
	}

	preamble := shared.ConventionsPreamble(root, antigravityConventionSources(), rules, manifest.ShouldInjectConventions())

	lines := []string{
		"---",
		"name: test-writer",
		`description: "Writes tests FIRST following TDD principles. Reads the project rule set from .agents/rules/ before writing any test."`,
		"tools: " + buildAntigravityToolsList(),
		"model: gemini-2.0-flash",
		"---",
		"",
		preamble + shared.TestWriterInstructions("`.agents/rules/`"),
		"",
	}
	content := strings.Join(lines, "\n")

	return os.WriteFile(filepath.Join(agentDir, "test-writer.md"), []byte(content), 0o644)
}

func generateAntigravityVerifierFile(root string, rules []string, manifest *state.Manifest) error {
	agentDir := filepath.Join(root, ".agents", "agents")
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

	preamble := shared.ConventionsPreamble(root, antigravityConventionSources(), rules, manifest.ShouldInjectConventions())

	lines := []string{
		"---",
		"name: tddmaster-verifier",
		`description: "Independently verifies completed task work. Read-only. Never sees the executor's context."`,
		"tools: run_command, view_file, list_dir, grep_search",
		"model: gemini-2.0-flash",
		"---",
		"",
		preamble + verifierInstructions,
		"",
		"The orchestrator will use your report for the tddmaster status report.",
		"",
	}

	return os.WriteFile(filepath.Join(agentDir, "tddmaster-verifier.md"), []byte(strings.Join(lines, "\n")), 0o644)
}

func generateAntigravityPlannerFile(root, commandPrefix string, rules []string, manifest *state.Manifest) error {
	agentDir := filepath.Join(root, ".agents", "agents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return err
	}

	preamble := shared.ConventionsPreamble(root, antigravityConventionSources(), rules, manifest.ShouldInjectConventions())

	lines := []string{
		"---",
		"name: tddmaster-planner",
		`description: "Produces a structured implementation plan for an important tddmaster task. Read-only — does not edit code."`,
		"tools: run_command, view_file, list_dir, grep_search, ask_question",
		"model: gemini-2.0-flash",
		"---",
		"",
		preamble + shared.PlannerInstructions(commandPrefix),
		"",
	}

	return os.WriteFile(filepath.Join(agentDir, "tddmaster-planner.md"), []byte(strings.Join(lines, "\n")), 0o644)
}

// =============================================================================
// Hooks configuration builder
// =============================================================================

func buildAntigravityHooksConfig(commandPrefix string) map[string]interface{} {
	return map[string]interface{}{
		"enabled": true,
		"PreToolUse": []interface{}{
			map[string]interface{}{
				"matcher": "Write|Edit|MultiEdit|Bash|write_to_file|replace_file_content|multi_replace_file_content|run_command",
				"hooks": []interface{}{
					map[string]interface{}{
						"command": fmt.Sprintf("%s invoke-hook pre-tool-use", commandPrefix),
					},
				},
			},
		},
		"PostToolUse": []interface{}{
			map[string]interface{}{
				"matcher": "Write|Edit|MultiEdit|write_to_file|replace_file_content|multi_replace_file_content",
				"hooks": []interface{}{
					map[string]interface{}{
						"command": fmt.Sprintf("%s invoke-hook post-file-write", commandPrefix),
					},
				},
			},
			map[string]interface{}{
				"matcher": "Bash|run_command",
				"hooks": []interface{}{
					map[string]interface{}{
						"command": fmt.Sprintf("%s invoke-hook post-bash", commandPrefix),
					},
				},
			},
		},
		"Stop": []interface{}{
			map[string]interface{}{
				"hooks": []interface{}{
					map[string]interface{}{
						"command": fmt.Sprintf("%s invoke-hook stop", commandPrefix),
					},
				},
			},
		},
		"SessionStart": []interface{}{
			map[string]interface{}{
				"hooks": []interface{}{
					map[string]interface{}{
						"command": fmt.Sprintf("%s invoke-hook session-start", commandPrefix),
					},
				},
			},
		},
	}
}

// =============================================================================
// Spec parser/builder
// =============================================================================

type antigravitySpecParsed struct {
	title            string
	concerns         string
	discoveryAnswers string
	contributorGuide string
	publicAPI        string
	outOfScope       string
	tasks            string
	verification     string
}

var antigravityH1Re = regexp.MustCompile(`(?m)^# Spec:\s*(.+)$`)

func parseAntigravitySpecMd(content string) antigravitySpecParsed {
	var parsed antigravitySpecParsed

	if m := antigravityH1Re.FindStringSubmatch(content); m != nil {
		parsed.title = strings.TrimSpace(m[1])
	} else {
		parsed.title = "Untitled"
	}

	sections := map[string]string{}
	parts := strings.Split(content, "\n## ")
	for _, part := range parts {
		newlineIdx := strings.IndexByte(part, '\n')
		if newlineIdx == -1 {
			continue
		}
		heading := strings.ToLower(strings.TrimSpace(part[:newlineIdx]))
		body := strings.TrimSpace(part[newlineIdx+1:])
		sections[heading] = body
	}

	findSection := func(prefix string) string {
		for k, v := range sections {
			if strings.HasPrefix(k, strings.ToLower(prefix)) {
				return v
			}
		}
		return ""
	}

	parsed.concerns = findSection("concerns")
	parsed.discoveryAnswers = findSection("discovery answers")
	parsed.contributorGuide = findSection("contributor guide")
	parsed.publicAPI = findSection("public api")
	parsed.outOfScope = findSection("out of scope")
	parsed.tasks = findSection("tasks")
	parsed.verification = findSection("verification")

	return parsed
}

func buildAntigravitySkillMd(spec antigravitySpecParsed) string {
	lines := []string{
		"---",
		"name: " + spec.title,
		`description: "tddmaster spec: ` + spec.title + `"`,
		"---",
		"",
		"# " + spec.title,
		"",
	}

	if spec.discoveryAnswers != "" {
		lines = append(lines, "## Overview", "", spec.discoveryAnswers, "")
	}
	if spec.concerns != "" {
		lines = append(lines, "## Concerns", "", spec.concerns, "")
	}
	if spec.contributorGuide != "" {
		lines = append(lines, "## Contributor Guide", "", spec.contributorGuide, "")
	}
	if spec.publicAPI != "" {
		lines = append(lines, "## Public API", "", spec.publicAPI, "")
	}
	if spec.outOfScope != "" {
		lines = append(lines, "## Out of Scope", "", spec.outOfScope, "")
	}
	if spec.tasks != "" {
		lines = append(lines, "## Tasks", "", spec.tasks, "")
	}
	if spec.verification != "" {
		lines = append(lines, "## Verification", "", spec.verification, "")
	}

	return strings.Join(lines, "\n")
}
