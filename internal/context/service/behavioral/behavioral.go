// Package behavioral builds the per-phase BehavioralBlock — the rules and
// tone that tell the caller (the LLM orchestrator) how to behave during the
// current phase. The block is produced fresh for every Compile() call.
package behavioral

import (
	"fmt"
	"strings"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/context/service/concerns"
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
)

// Renderer is the minimal command-builder interface required by this package.
type Renderer interface {
	C(sub string) string
	CS(sub string, specName *string) string
}

func strPtr(s string) *string { return &s }

// pickAskMethod returns the fragment plugged into mandatoryRules based on the
// active interaction hints.
func pickAskMethod(hints model.InteractionHints) string {
	switch {
	case hints.AskUserStrategy == "tddmaster_block":
		return model.AskMethodBlock
	case !hints.HasAskUserTool:
		return model.AskMethodProse
	default:
		return model.AskMethodAskUser
	}
}

// buildMandatoryRules assembles the preamble that every phase shares.
func buildMandatoryRules(allowGit bool, hints model.InteractionHints) []string {
	var rules []string
	if !allowGit {
		rules = append(rules, model.GitReadonlyRule)
	}
	ask := pickAskMethod(hints)
	rules = append(rules,
		model.MandatoryRuleProgress,
		fmt.Sprintf(model.MandatoryRuleNeverSkipFmt, ask),
		model.MandatoryRuleRoadmapFirst,
		model.MandatoryRuleNoBypass,
		model.MandatoryRuleNoPermission,
		model.MandatoryRuleListenFirst,
		model.MandatoryRuleAdaptiveQ,
		model.MandatoryRuleConfidence,
	)
	return rules
}

// Build returns the phase-specific behavioural block.
func Build(
	r Renderer,
	st state.StateFile,
	maxIterationsBeforeRestart int,
	allowGit bool,
	activeConcerns []state.ConcernDefinition,
	parsedSpec *spec.ParsedSpec,
	hints model.InteractionHints,
) model.BehavioralBlock {
	stale := st.Execution.Iteration >= maxIterationsBeforeRestart
	mandatory := buildMandatoryRules(allowGit, hints)

	var scopeItems []string
	if parsedSpec != nil {
		scopeItems = parsedSpec.OutOfScope
	}

	switch st.Phase {
	case state.PhaseIdle:
		return phaseIdleBehavioral(r, st, mandatory, hints)
	case state.PhaseDiscovery:
		return phaseDiscoveryBehavioral(mandatory, activeConcerns, hints)
	case state.PhaseDiscoveryRefinement:
		return phaseDiscoveryRefinementBehavioral(mandatory, hints)
	case state.PhaseSpecProposal:
		return phaseSpecProposalBehavioral(r, st, mandatory, hints)
	case state.PhaseSpecApproved:
		return phaseSpecApprovedBehavioral(mandatory, hints)
	case state.PhaseExecuting:
		return phaseExecutingBehavioral(r, st, mandatory, scopeItems, hints, stale)
	case state.PhaseBlocked:
		return model.BehavioralBlock{
			Rules: append(mandatory,
				"Present the decision to the user exactly as described.",
				"Do not suggest a preferred option unless the user asks for your opinion.",
				"After the user decides, relay the answer immediately. Do not elaborate.",
			),
			Tone: model.ToneBlocked,
		}
	case state.PhaseCompleted:
		return model.BehavioralBlock{
			Rules: append(mandatory,
				"Report the completion summary. Do not start new work.",
				"If the user wants to continue, they start a new spec.",
			),
			Tone: model.ToneCompleted,
		}
	default:
		return model.BehavioralBlock{
			Rules: append(mandatory,
				fmt.Sprintf("Run `%s` to get your instructions.", r.CS("next", st.Spec)),
				"Do not take action without tddmaster guidance.",
			),
			Tone: model.ToneDefault,
		}
	}
}

func phaseIdleBehavioral(_ Renderer, _ state.StateFile, mandatory []string, hints model.InteractionHints) model.BehavioralBlock {
	optionRule := "Pass interactiveOptions DIRECTLY to AskUserQuestion options array (header max 12 chars). Use commandMap to resolve selections. For availableConcerns: AskUserQuestion with multiSelect:true, max 4 per question — split across questions if needed. Present ALL concerns."
	if hints.OptionPresentation != "tool" {
		optionRule = "Present interactiveOptions as numbered list. Use commandMap to resolve selections. Present ALL availableConcerns as numbered list for multiselect."
	}
	rules := append([]string{
		"If the user described a feature/bug/task, create a spec immediately: `tddmaster spec new \"description\"` — name is auto-generated. Do NOT present menus or ask 'What would you like to do?' unless the conversation has no prior context.",
	}, append(mandatory,
		optionRule,
		"Encourage full context: 'Tell me what you want to build — one-liner, detailed requirements, meeting notes, anything.' Slug is auto-generated. Pass full text to `tddmaster spec new \"...\"`.",
		"After spec new, listen first, then ask the user to choose a discovery mode: full, validate, technical-depth, ship-fast, or explore.",
		"Every task gets a spec. No exceptions. A one-liner fix, a config change, a 'simple' refactor — all get specs. The spec can be short but it must exist. 'Too simple for a spec' is the anti-pattern.",
	)...)

	return model.BehavioralBlock{
		Rules: rules,
		Tone:  model.ToneIdle,
	}
}

func phaseDiscoveryBehavioral(mandatory []string, activeConcerns []state.ConcernDefinition, hints model.InteractionHints) model.BehavioralBlock {
	var questionMethod, subStepMethod string
	switch {
	case hints.AskUserStrategy == "tddmaster_block":
		questionMethod = "Ask one question at a time via `tddmaster block \"question\"`. One question per interaction."
		subStepMethod = "Listen-first: ask the single open-ended context question via `tddmaster block`. Mode selection: present the provided interactiveOptions one by one via `tddmaster block`. Premise challenge: one `tddmaster block` call per premise."
	case !hints.HasAskUserTool:
		questionMethod = "Ask one question at a time as text."
		subStepMethod = "Listen-first: ask the single open-ended context question as plain text. Mode selection: present the provided interactiveOptions as a numbered list. Premise challenge: ask one premise at a time as plain text."
	default:
		questionMethod = "Ask each question via AskUserQuestion. One question per call."
		subStepMethod = "Listen-first step: ask the single open-ended context question via AskUserQuestion (free-form, no fabricated options). Mode selection: use AskUserQuestion with the provided interactiveOptions — never present them as prose or a numbered list. Premise challenge: one AskUserQuestion per premise, aggregate client-side, then submit the JSON payload."
	}

	dreamPrompts := concerns.GetDreamStatePrompts(activeConcerns)
	var dreamBase string
	if len(dreamPrompts) > 0 {
		dreamBase = "After answers, " + strings.Join(dreamPrompts, " Also: ")
	} else {
		dreamBase = "After answers, synthesize CURRENT STATE → THIS SPEC → 6-MONTH IDEAL vision."
	}

	rules := append(mandatory,
		fmt.Sprintf("%s Never answer questions yourself. Never submit answers without user confirmation. Pre-fill suggested answers from detailed descriptions — user must confirm each. With a fully formed plan, keep discovery brief by confirming pre-filled answers one at a time, but MUST still run premise challenge and alternatives.", questionMethod),
		subStepMethod,
		"DO NOT create, edit, or write any files.",
		"DO NOT run shell commands that modify state.",
		"You MAY read files and run read-only commands (cat, ls, grep, git log, git diff).",
		"Pre-discovery: (1) pre-discovery codebase scan — read README, CLAUDE.md, design docs, last 20 commits, TODOs, existing specs, directory structure. Present a brief audit summary. (2) If `preDiscoveryResearch.required`, web-search every `extractedTerms` entry — report versions, API changes, deprecations. (3) Ask discovery mode using the real options: A) Full discovery B) Validate my plan C) Technical depth D) Ship fast E) Explore scope. Adapt emphasis accordingly.",
		"Before starting discovery questions, challenge the user's initial spec description against codebase findings. Flag: hidden complexity, conflicts with existing code, scope mismatch, overlapping modules. Ask clarifying follow-ups.",
		"When asking questions, offer concrete options from codebase knowledge alongside the open-ended question (e.g., 'I see three scenarios: A)... B)... C)... D) Something else'). Push back on vague answers. Follow up on short answer with 'Can you be more specific?'",
		fmt.Sprintf("%s Then: (1) expansion opportunities as numbered proposals with effort (S/M/L/XL), risk, completeness delta — options: Add/Defer/Skip. (2) Architectural decisions that BLOCK implementation — present with options, RECOMMENDATION, completeness scores. Unresolved = risk flag. (3) Error/rescue map: codepath | failure mode | handling. Flag CRITICAL GAPS as decisions.", dreamBase),
		"Present DISCOVERY SUMMARY for confirmation: intent, scope, dream state, expansions, architectural decisions, error map. Ask for confirmation before generating spec. Keep discovery sequential: submit each confirmed answer as its own `tddmaster next --answer` call.",
	)

	return model.BehavioralBlock{
		ModeOverride: strPtr(model.ModeOverrideDiscovery),
		Rules:        rules,
		Tone:         model.ToneDiscovery,
	}
}

func phaseDiscoveryRefinementBehavioral(mandatory []string, hints model.InteractionHints) model.BehavioralBlock {
	var confirmQ string
	switch {
	case hints.AskUserStrategy == "tddmaster_block":
		confirmQ = "Use `tddmaster block \"Are these answers correct, or would you like to revise any?\"` to pause for user review."
	case !hints.HasAskUserTool:
		confirmQ = "Ask the user: 'Are these answers correct, or would you like to revise any?' Present approval and revision as numbered options."
	default:
		confirmQ = "Use AskUserQuestion to ask: 'Are these answers correct, or would you like to revise any?'"
	}
	return model.BehavioralBlock{
		ModeOverride: strPtr(model.ModeOverrideDiscoveryRefinement),
		Rules: append(mandatory,
			"DO NOT create, edit, or write any files.",
			"Present ALL discovery answers to the user clearly, one by one.",
			"Before you ask for approval or revision, render `discoveryReviewData.reviewSummary` verbatim. If it is missing, render every item in `discoveryReviewData.answers` yourself.",
			"Do NOT jump straight to interactiveOptions. The answer review must appear before approve/revise/split choices.",
			confirmQ,
			"If the user approves, run the approve command.",
			"If the user wants to revise, collect their corrections and submit them.",
			"You MUST NOT approve on behalf of the user. The user must explicitly confirm.",
			"If tddmaster output contains a splitProposal, present it to the user with the exact options shown. Do NOT split or merge specs on your own. Do NOT recommend one option over the other unless the user asks for your opinion. The user decides.",
		),
		Tone: model.ToneDiscoveryRefinement,
	}
}

func phaseSpecProposalBehavioral(r Renderer, st state.StateFile, mandatory []string, hints model.InteractionHints) model.BehavioralBlock {
	delegations := st.Discovery.Delegations
	var delegationRules []string
	if len(delegations) > 0 {
		var lines []string
		pendingCount := 0
		answeredCount := 0
		for _, d := range delegations {
			status := "PENDING"
			if d.Status == "answered" {
				status = "ANSWERED ✓"
				answeredCount++
			} else {
				pendingCount++
			}
			lines = append(lines, fmt.Sprintf("- %s: delegated to %s — %s", d.QuestionID, d.DelegatedTo, status))
		}
		suffix := fmt.Sprintf("\n%d delegation(s) answered. Approve is allowed.", answeredCount)
		if pendingCount > 0 {
			suffix = fmt.Sprintf("\nApprove BLOCKED — %d pending delegation(s).", pendingCount)
		}
		delegationRules = append(delegationRules, "DELEGATION STATUS:\n"+strings.Join(lines, "\n")+suffix)
	}

	var classifyQ string
	switch {
	case hints.AskUserStrategy == "tddmaster_block":
		classifyQ = "When presenting classification options, present them as a numbered list. Use `tddmaster block` if you need the user to pick. Do NOT infer classification yourself."
	case !hints.HasAskUserTool:
		classifyQ = "When presenting classification options, present them as a numbered list with multiselect (user picks multiple numbers). Do NOT infer classification yourself."
	default:
		classifyQ = "When presenting classification options, use AskUserQuestion with multiSelect:true. Do NOT infer classification yourself."
	}

	rules := append(append(mandatory, delegationRules...),
		"DO NOT create, edit, or write any files.",
		"Read the spec and present a summary to the user.",
		"Flag any tasks that are too vague to execute.",
		"Flag any missing acceptance criteria.",
		"No placeholders in specs. If a task has 'TBD', 'TODO', 'to be determined', 'details to follow', or 'implement appropriate X' — fill in the detail or remove the task and add it as an open question.",
		"Ask the user if they want to refine before approving.",
		classifyQ,
		"When generating or refining tasks, include a 'Files:' hint listing likely files to create/modify. Format: 'Files: `path/to/file.ts`, `path/to/other.ts`'. Hints, not constraints — helps sub-agents load right context.",
		"If you identify issues in the spec (vague tasks, irrelevant sections, missing acceptance criteria), submit a refinement via: `"+
			r.CS("next --answer='{\"refinement\":\"task-1: Add upload endpoint, task-2: Add validation middleware, task-3: Write integration tests\"}'", st.Spec)+
			"`. The spec will be updated and you can review again.",
	)

	return model.BehavioralBlock{
		ModeOverride: strPtr(model.ModeOverrideSpecProposal),
		Rules:        rules,
		Tone:         model.ToneSpecProposal,
	}
}

func phaseSpecApprovedBehavioral(mandatory []string, hints model.InteractionHints) model.BehavioralBlock {
	var confirmQ string
	switch {
	case hints.AskUserStrategy == "tddmaster_block":
		confirmQ = "Before starting execution, show the spec summary and use `tddmaster block \"Confirm execution?\"` to pause for user go-ahead."
	case !hints.HasAskUserTool:
		confirmQ = "Before starting execution, show the spec summary to the user and ask for final confirmation. Present 'Start execution' and 'Not yet' as numbered options."
	default:
		confirmQ = "Before starting execution, show the spec summary to the user and ask for final confirmation via AskUserQuestion."
	}
	return model.BehavioralBlock{
		Rules: append(mandatory,
			"The spec is approved but execution has not started.",
			"Do not start coding until the user triggers execution.",
			"If the user wants changes, they must reset and re-spec.",
			confirmQ,
		),
		Tone: model.ToneSpecApproved,
	}
}

func phaseExecutingBehavioral(
	r Renderer,
	st state.StateFile,
	mandatory []string,
	scopeItems []string,
	hints model.InteractionHints,
	stale bool,
) model.BehavioralBlock {
	reportCmd := r.CS("next --answer='{\"completed\":[...],\"remaining\":[...],\"blocked\":[]}'", st.Spec)
	verifierScope := "Verifier scope: (1) AC verification with evidence. (2) Plan alignment — does implementation match task description? Flag deviations. (3) Code quality — follows existing patterns? Flag style breaks, missing error handling. Categorize findings: CRITICAL (blocks completion), IMPORTANT (should fix), SUGGESTION (nice to have)."

	var spawnInstruction, verifyInstruction string
	switch hints.SubAgentMethod {
	case "task":
		spawnInstruction = fmt.Sprintf("Spawn tddmaster-executor via Agent tool. Pass: task title, description, ACs, rules, scope constraints, concern reminders, file paths. Report via `%s`.", reportCmd)
		verifyInstruction = fmt.Sprintf("After executor completes, spawn tddmaster-verifier with changed files + ACs + test commands. Never trust executor self-report alone. %s", verifierScope)
	case "spawn":
		spawnInstruction = fmt.Sprintf("Use spawn_agent for tddmaster-executor. Pass: task, ACs, rules, scope, file paths. Use wait_agent to collect. Report via `%s`.", reportCmd)
		verifyInstruction = fmt.Sprintf("After executor completes, spawn tddmaster-verifier with changed files + ACs + test commands. %s", verifierScope)
	case "fleet":
		spawnInstruction = fmt.Sprintf("Use /fleet for parallel executors. Pass each: task, ACs, rules, scope, file paths. Report via `%s`.", reportCmd)
		verifyInstruction = fmt.Sprintf("After fleet completes, run verification pass. %s", verifierScope)
	case "delegation":
		spawnInstruction = fmt.Sprintf("Use agent delegation for tddmaster-executor. Pass: task, ACs, rules, scope, file paths. Report via `%s`.", reportCmd)
		verifyInstruction = fmt.Sprintf("After executor completes, delegate to tddmaster-verifier with changed files + ACs + test commands. %s", verifierScope)
	default:
		spawnInstruction = fmt.Sprintf("Execute tasks sequentially yourself. Verify (type-check + tests) after each. Report via `%s`.", reportCmd)
		verifyInstruction = ""
	}

	hasSubAgents := hints.SubAgentMethod != "none"
	var orchestratorRule string
	if hasSubAgents {
		orchestratorRule = fmt.Sprintf("You are the orchestrator. NEVER edit files directly — delegate ALL edits to tddmaster-executor. %s On sub-agent failure, fall back to direct execution and note it in status.", spawnInstruction)
	} else {
		orchestratorRule = spawnInstruction
	}

	base := append(mandatory, orchestratorRule)
	if verifyInstruction != "" {
		base = append(base, verifyInstruction)
	}
	base = append(base,
		"Do not explore beyond current task. Do not refactor outside scope. Do not add features, tests, or docs not in the spec. timebox context reads — the deliverable is working code.",
	)
	if hasSubAgents {
		base = append(base, "Show a dispatch table: | Step | Agent | Files | Tasks | Est. |. Separate executor for implementation vs tests. Batch tightly-coupled files; parallelize independent modules.")
	}
	base = append(base,
		"Edit discipline: (1) Re-read file before editing. (2) Re-read after to confirm. (3) Files >500 LOC: read in chunks. (4) Run type-check + lint after edits — never mark AC passed if type-check fails. (5) If search returns few results, re-run narrower — assume truncation.",
		fmt.Sprintf("On recurring patterns or corrections, ask: 'Permanent rule or just this task?' If permanent: `%s`. Never write to `.tddmaster/rules/` directly.", r.C("rule add \"<description>\"")),
		"Before structural refactors on files >300 LOC, remove dead code first. Do NOT suggest pausing or stopping mid-spec — execute to completion. The user decides when to stop.",
		"RATIONALIZATION ALERT: Never use 'should work now', 'looks correct', 'I'm confident', 'seems right', 'probably passes'. Run the command, read the output, report what happened. Evidence, not belief.",
		"TDD: (1) Write test. (2) Run it — MUST fail. If it passes before implementation, the test is wrong. (3) Implement. (4) Run test — must pass. Skipping step 2 means the test is unverified.",
		"If execution output includes `edgeCases`, pass them explicitly to the test-writer. Those cases are mandatory red-phase coverage before implementation.",
		"Parallel vs serial sub-agents: PARALLEL when tasks touch different files with no shared state. SERIAL when tasks modify same files or depend on each other. When unsure, default to serial.",
	)
	if hasSubAgents {
		base = append(base, "VERIFICATION REQUIRED: After EVERY task completion, you MUST spawn tddmaster-verifier before reporting done. If you skip verification and self-report, the status report will flag it. No exceptions — 'it looks correct' is not verification.")
	} else {
		base = append(base, "VERIFICATION REQUIRED: After EVERY task completion, run type-check + tests before reporting done. Evidence of passing tests must be included in the status report.")
	}

	if st.Execution.LastVerification != nil && !st.Execution.LastVerification.Passed {
		base = append(base, "Tests are failing. Fix ONLY the failing tests. Do not refactor passing code.")
	}

	behavioral := model.BehavioralBlock{
		Rules:      base,
		Tone:       model.ToneExecuting,
		OutOfScope: scopeItems,
	}
	if len(scopeItems) == 0 {
		behavioral.OutOfScope = nil
	}

	if stale {
		urgency := fmt.Sprintf("%d+ iterations — context degrading. Finish current task, recommend fresh session.", st.Execution.Iteration)
		behavioral.Urgency = &urgency
	}

	return behavioral
}
