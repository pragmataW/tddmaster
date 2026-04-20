package service

import (
	"fmt"
	"strings"

	"github.com/pragmataW/tddmaster/internal/state"
)

// renderOptions bundles optional override inputs so Render keeps its positional
// signature stable for existing test callers.
type renderOptions struct {
	overrideTasks      []state.SpecTask
	overrideOutOfScope []string
}

// RenderOption customizes Render output. Use the WithX helpers to set each
// field — callers that pass no options get the classic auto-derived behavior.
type RenderOption func(*renderOptions)

// WithTaskOverride replaces the auto-derived Tasks list with the provided
// SpecTask slice. IDs are preserved as-is (no renumbering).
func WithTaskOverride(tasks []state.SpecTask) RenderOption {
	return func(o *renderOptions) { o.overrideTasks = tasks }
}

// WithOutOfScopeOverride appends additional out-of-scope items on top of the
// scope-boundary answer.
func WithOutOfScopeOverride(items []string) RenderOption {
	return func(o *renderOptions) { o.overrideOutOfScope = items }
}

// Render builds a full spec.md markdown document from the provided data.
// When tddMode is true, test-related tasks are moved to the beginning of the
// auto-derived task list.
func Render(
	specName string,
	answers []state.DiscoveryAnswer,
	premises []state.Premise,
	concerns []state.ConcernDefinition,
	decisions []state.Decision,
	classification *state.SpecClassification,
	customACs []state.CustomAC,
	specNotes []state.SpecNote,
	transitionHistory []state.PhaseTransition,
	tddMode bool,
	options ...RenderOption,
) string {
	var opts renderOptions
	for _, apply := range options {
		apply(&opts)
	}

	var lines []string

	lines = append(lines, fmt.Sprintf("# Spec: %s", specName), "")
	lines = append(lines, "## Status: draft", "")

	lines = appendConcernsHeader(lines, concerns)
	lines = appendDiscoveryAnswers(lines, answers)
	lines = appendConcernSections(lines, concerns, classification)
	lines = appendRegistryTables(lines, concerns)
	lines = appendDecisions(lines, decisions)
	lines = appendOutOfScope(lines, answers, opts.overrideOutOfScope)
	lines = appendEdgeCases(lines, answers, premises)
	lines = appendTasks(lines, answers, decisions, opts.overrideTasks, tddMode)
	lines = appendVerification(lines, answers)
	lines = appendCustomACs(lines, customACs)
	lines = appendNotes(lines, specNotes)
	lines = appendTransitionHistory(lines, transitionHistory)

	return strings.Join(lines, "\n")
}

func appendConcernsHeader(lines []string, concerns []state.ConcernDefinition) []string {
	if len(concerns) == 0 {
		return lines
	}
	ids := make([]string, len(concerns))
	for i, c := range concerns {
		ids[i] = c.ID
	}
	return append(lines, fmt.Sprintf("## Concerns: %s", strings.Join(ids, ", ")), "")
}

func appendDiscoveryAnswers(lines []string, answers []state.DiscoveryAnswer) []string {
	lines = append(lines, "## Discovery Answers", "")
	for _, a := range answers {
		if a.QuestionID == "edge_cases" || a.QuestionID == "scope_boundary" {
			continue
		}
		lines = append(lines, fmt.Sprintf("### %s", a.QuestionID), "", a.Answer, "")
	}
	return lines
}

func appendConcernSections(lines []string, concerns []state.ConcernDefinition, classification *state.SpecClassification) []string {
	for _, concern := range concerns {
		if len(concern.SpecSections) == 0 {
			continue
		}
		relevance := checkSectionRelevance(concern, classification)
		for _, section := range concern.SpecSections {
			if !relevance[section] {
				continue
			}
			lines = append(lines,
				fmt.Sprintf("## %s (%s)", section, concern.ID),
				"",
				"_To be addressed during execution._",
				"",
			)
		}
	}
	return lines
}

func appendRegistryTables(lines []string, concerns []state.ConcernDefinition) []string {
	for _, concern := range concerns {
		if len(concern.Registries) == 0 {
			continue
		}
		for _, regID := range concern.Registries {
			dim := findDimension(concern.ReviewDimensions, regID)
			if dim == nil {
				continue
			}
			lines = append(lines,
				fmt.Sprintf("## %s (%s)", dim.Label, concern.ID),
				"",
				fmt.Sprintf("_%s_", dim.Prompt),
				"",
			)
			lines = append(lines, registryTableLines(regID)...)
			lines = append(lines, "")
		}
	}
	return lines
}

func findDimension(dims []state.ReviewDimension, id string) *state.ReviewDimension {
	for i := range dims {
		if dims[i].ID == id {
			return &dims[i]
		}
	}
	return nil
}

func registryTableLines(regID string) []string {
	switch regID {
	case "error-rescue":
		return []string{
			"| Codepath | What Can Go Wrong | Exception Class | Rescued? | Recovery Action | User Sees |",
			"|----------|-------------------|-----------------|----------|----------------|-----------|",
			"| _To be filled during execution._ | | | | | |",
		}
	case "failure-modes":
		return []string{
			"| Codepath | Failure Mode | Rescued? | Tested? | User Sees | Logged? |",
			"|----------|--------------|----------|---------|-----------|---------|",
			"| _To be filled during execution._ | | | | | |",
		}
	case "test-plan":
		return []string{
			"| Behavior | Test Layer | Happy Path | Failure Path | Edge Case |",
			"|----------|------------|------------|--------------|-----------|",
			"| _To be filled during execution._ | | | | |",
		}
	default:
		return []string{"_To be filled during execution._"}
	}
}

func appendDecisions(lines []string, decisions []state.Decision) []string {
	if len(decisions) == 0 {
		return lines
	}
	lines = append(lines,
		"## Decisions",
		"",
		"| # | Decision | Choice | Promoted |",
		"|---|----------|--------|----------|",
	)
	for i, d := range decisions {
		promoted := "no"
		if d.Promoted {
			promoted = "yes"
		}
		lines = append(lines, fmt.Sprintf("| %d | %s | %s | %s |", i+1, d.Question, d.Choice, promoted))
	}
	return append(lines, "")
}

func appendOutOfScope(lines []string, answers []state.DiscoveryAnswer, overrides []string) []string {
	scopeAnswer := findAnswer(answers, "scope_boundary")
	if scopeAnswer == nil && len(overrides) == 0 {
		return lines
	}
	lines = append(lines, "## Out of Scope", "")
	if scopeAnswer != nil {
		for _, item := range toBulletList(scopeAnswer.Answer) {
			lines = append(lines, fmt.Sprintf("- %s", item))
		}
	}
	for _, item := range overrides {
		lines = append(lines, fmt.Sprintf("- %s", item))
	}
	return append(lines, "")
}

func appendEdgeCases(lines []string, answers []state.DiscoveryAnswer, premises []state.Premise) []string {
	edgeCases := DeriveEdgeCases(answers, premises)
	if len(edgeCases) == 0 {
		return lines
	}
	lines = append(lines, "## Edge Cases", "")
	for _, item := range edgeCases {
		lines = append(lines, fmt.Sprintf("- %s", item))
	}
	return append(lines, "")
}

func appendTasks(lines []string, answers []state.DiscoveryAnswer, decisions []state.Decision, overrides []state.SpecTask, tddMode bool) []string {
	lines = append(lines, "## Tasks", "")
	if len(overrides) > 0 {
		for _, t := range overrides {
			checkbox := "[ ]"
			if t.Completed {
				checkbox = "[x]"
			}
			lines = append(lines, fmt.Sprintf("- %s %s: %s", checkbox, t.ID, t.Title))
			if len(t.Covers) > 0 {
				lines = append(lines, fmt.Sprintf("  Covers: %s", strings.Join(t.Covers, ", ")))
			}
		}
	} else {
		for i, task := range deriveTasks(answers, decisions, tddMode) {
			lines = append(lines, fmt.Sprintf("- [ ] task-%d: %s", i+1, task))
		}
	}
	return append(lines, "")
}

func appendVerification(lines []string, answers []state.DiscoveryAnswer) []string {
	lines = append(lines, "## Verification", "")
	if verificationAnswer := findAnswer(answers, "verification"); verificationAnswer != nil {
		for _, item := range toBulletList(verificationAnswer.Answer) {
			lines = append(lines, fmt.Sprintf("- %s", item))
		}
	} else {
		lines = append(lines, "_To be defined._")
	}
	return append(lines, "")
}

func appendCustomACs(lines []string, customACs []state.CustomAC) []string {
	if len(customACs) == 0 {
		return lines
	}
	lines = append(lines, "## Custom Acceptance Criteria", "")
	for _, ac := range customACs {
		lines = append(lines, fmt.Sprintf("- %s _-- %s, %s_", ac.Text, ac.User, string(ac.AddedInPhase)))
	}
	return append(lines, "")
}

func appendNotes(lines []string, specNotes []state.SpecNote) []string {
	var filtered []state.SpecNote
	for _, n := range specNotes {
		if !strings.HasPrefix(n.Text, "[TASK] ") {
			filtered = append(filtered, n)
		}
	}
	if len(filtered) == 0 {
		return lines
	}
	lines = append(lines, "## Notes", "")
	for _, note := range filtered {
		lines = append(lines, fmt.Sprintf("- %s _-- %s, %s_", note.Text, note.User, string(note.Phase)))
	}
	return append(lines, "")
}

func appendTransitionHistory(lines []string, history []state.PhaseTransition) []string {
	if len(history) == 0 {
		return lines
	}
	lines = append(lines,
		"## Transition History",
		"",
		"| From | To | User | Timestamp | Reason |",
		"|------|----|------|-----------|--------|",
	)
	for _, t := range history {
		reason := "-"
		if t.Reason != nil {
			reason = *t.Reason
		}
		lines = append(lines, fmt.Sprintf("| %s | %s | %s | %s | %s |",
			string(t.From), string(t.To), t.User, t.Timestamp, reason))
	}
	return append(lines, "")
}
