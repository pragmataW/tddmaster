package spec

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pragmataW/tddmaster/internal/state"
)

// =============================================================================
// Helpers
// =============================================================================

// sentenceSplitRe matches sentence-ending periods followed by space+uppercase.
// It does NOT match periods inside filenames, extensions, version numbers, or URLs.
// Semicolons are NOT split points — they're legitimate punctuation in long
// technical answers and splitting on them fragmented content mid-sentence.
var sentenceSplitRe = regexp.MustCompile(`\.(?:\s+[A-Z])`)

// toBulletList splits text into list items by line breaks or sentence boundaries.
// Does NOT split on dots inside filenames, extensions, abbreviations,
// version numbers, or URLs, nor on semicolons within a sentence.
func toBulletList(text string) []string {
	// Split on line breaks first
	rawLines := strings.Split(text, "\n")
	var lines []string
	for _, l := range rawLines {
		t := strings.TrimSpace(l)
		if len(t) > 0 {
			lines = append(lines, t)
		}
	}

	if len(lines) > 1 {
		return lines
	}

	// Single block — split on sentence-ending periods (". A").
	// We use FindAllStringIndex to locate split points, then slice manually
	// so we preserve the character after the dot (uppercase letter of next sentence).
	idxs := sentenceSplitRe.FindAllStringIndex(text, -1)
	if len(idxs) == 0 {
		trimmed := strings.TrimSpace(text)
		if len(trimmed) > 5 {
			return []string{trimmed}
		}
		return []string{}
	}

	var parts []string
	prev := 0
	for _, loc := range idxs {
		// Match is ". A" (len 3). Keep the period with the first part; next
		// sentence starts at loc[0]+2 (skip ". ").
		parts = append(parts, strings.TrimSpace(text[prev:loc[0]+1]))
		prev = loc[0] + 2
	}
	if prev < len(text) {
		parts = append(parts, strings.TrimSpace(text[prev:]))
	}

	var result []string
	for _, p := range parts {
		if len(p) > 5 {
			result = append(result, p)
		}
	}
	return result
}

// =============================================================================
// DeriveTasks
// =============================================================================

var tenStarRe = regexp.MustCompile(`(?is)10[- ]?star[:\s]+(.+?)(?:\n|$)`)
var fiveStarRe = regexp.MustCompile(`(?is)5[- ]?star[:\s]+(.+?)(?:\n|$)`)
var oneStarPrefixRe = regexp.MustCompile(`(?i)1[- ]?star[:\s]+[^.]*\.\s*`)
var leadingArticleRe = regexp.MustCompile(`(?i)^(the|a|an|with|plus|also)\s+`)
var goalPrefixRe = regexp.MustCompile(`(?i)^(the\s+)?(target|goal|objective)[:\s]+`)
var trailingPuncRe = regexp.MustCompile(`[.\x{2026}]+$`)
var bulletPrefixRe = regexp.MustCompile(`^\s*[-\x{2022}*]\s*`)
var edgeCasePrefixRe = regexp.MustCompile(`(?i)^(edge[- ]?cases?|watch(?:\s+out)?\s+for|consider)\s*[:\-]\s*`)
var edgeCaseKeywordRe = regexp.MustCompile(`(?i)\b(edge[- ]?cases?|empty|zero|missing|invalid|duplicate|timeout|slow|latency|partial|retry|offline|boundary|large|long|unicode|whitespace|nil|null|404|409|429|500|error|failure|fallback|concurrent|race)\b`)
var ellipsisRune = '\u2026'

// isTestTask reports whether a task string is a test-related task.
// Test tasks are identified by containing keywords like "test" or "tests".
func isTestTask(task string) bool {
	lower := strings.ToLower(task)
	return strings.Contains(lower, "test")
}

// DeriveTasks derives tasks from discovery answers and decisions.
// When tddMode is true, test-related tasks are moved to the beginning of the list
// to enforce test-first ordering.
func DeriveTasks(answers []state.DiscoveryAnswer, decisions []state.Decision, tddMode bool) []string {
	var tasks []string

	// From Q2 (ambition) — extract the implementation goal as ONE task.
	var ambition *state.DiscoveryAnswer
	for i := range answers {
		if answers[i].QuestionID == "ambition" {
			ambition = &answers[i]
			break
		}
	}
	if ambition != nil {
		text := ambition.Answer

		var goalText string
		if m := tenStarRe.FindStringSubmatch(text); m != nil {
			goalText = strings.TrimSpace(m[1])
		} else if m := fiveStarRe.FindStringSubmatch(text); m != nil {
			goalText = strings.TrimSpace(m[1])
		} else {
			goalText = strings.TrimSpace(oneStarPrefixRe.ReplaceAllString(text, ""))
		}

		// Clean: strip leading articles/filler, garbled prefixes
		cleaned := goalPrefixRe.ReplaceAllString(leadingArticleRe.ReplaceAllString(goalText, ""), "")
		cleaned = strings.TrimSpace(cleaned)

		// Capitalize first letter
		if len(cleaned) > 0 {
			cleaned = strings.ToUpper(cleaned[:1]) + cleaned[1:]
		}

		// Strip trailing period or ellipsis
		cleaned = strings.TrimSpace(trailingPuncRe.ReplaceAllString(cleaned, ""))

		// Trim to reasonable length
		if len(cleaned) > 140 {
			cleaned = cleaned[:137] + "..."
		}

		if len(cleaned) > 3 {
			tasks = append(tasks, cleaned)
		}
	}

	// From Q5 (verification) — verification tasks, kept whole.
	var verification *state.DiscoveryAnswer
	for i := range answers {
		if answers[i].QuestionID == "verification" {
			verification = &answers[i]
			break
		}
	}
	if verification != nil {
		bulletRe := regexp.MustCompile(`^\s*[-\x{2022}*]\s*`)
		for _, line := range strings.Split(verification.Answer, "\n") {
			item := strings.TrimSpace(bulletRe.ReplaceAllString(line, ""))
			if len(item) > 0 {
				tasks = append(tasks, item)
			}
		}
	}

	// From decisions — expansion proposals that were accepted become tasks.
	for _, d := range decisions {
		lower := strings.ToLower(d.Choice)
		if strings.Contains(lower, "accepted") || strings.Contains(lower, "add to scope") {
			taskText := regexp.MustCompile(`(?i)^should\s+(we|i)\s+`).ReplaceAllString(d.Question, "")
			taskText = strings.TrimRight(taskText, "?")
			taskText = strings.TrimSpace(taskText)
			if len(taskText) > 0 {
				capitalized := strings.ToUpper(taskText[:1]) + taskText[1:]
				tasks = append(tasks, capitalized)
			}
		}
	}

	// If no tasks could be derived, prompt the user
	if len(tasks) == 0 {
		tasks = append(tasks, "_Tasks need to be defined before execution. Add tasks manually or run discovery with more detail._")
	}

	// Always append mandatory test + docs tasks
	tasks = append(tasks, "Write or update tests for all new and changed behavior")
	tasks = append(tasks, "Update documentation for all public-facing changes (README, API docs, CHANGELOG)")

	// When tddMode is enabled, reorder so test-related tasks come first.
	if tddMode {
		var testTasks []string
		var otherTasks []string
		for _, task := range tasks {
			if isTestTask(task) {
				testTasks = append(testTasks, task)
			} else {
				otherTasks = append(otherTasks, task)
			}
		}
		tasks = append(testTasks, otherTasks...)
	}

	return tasks
}

// =============================================================================
// DeriveEdgeCases
// =============================================================================

func normalizeEdgeCaseCandidate(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}

	trimmed = bulletPrefixRe.ReplaceAllString(trimmed, "")
	trimmed = edgeCasePrefixRe.ReplaceAllString(trimmed, "")
	trimmed = strings.Join(strings.Fields(trimmed), " ")

	return strings.TrimSpace(trimmed)
}

func isEdgeCaseCandidate(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}

	if edgeCaseKeywordRe.MatchString(trimmed) {
		return true
	}

	lower := strings.ToLower(trimmed)
	return strings.HasPrefix(lower, "if ") ||
		strings.HasPrefix(lower, "when ") ||
		strings.HasPrefix(lower, "unless ") ||
		strings.HasPrefix(lower, "without ")
}

// DeriveEdgeCases extracts concrete edge cases from two intentional sources:
// (1) the explicit "edge_cases" answer (parsed literally, no keyword filter),
// and (2) disagreed/revised premises. Keyword harvesting from unrelated
// discovery answers was removed because it bled sentences containing
// incidental words like "error", "fallback", "nil", "race" from other
// sections into the Edge Cases list.
// The resulting list is de-duplicated and preserves first-seen order
// within each group.
func DeriveEdgeCases(answers []state.DiscoveryAnswer, premises []state.Premise) []string {
	var literalECs []string
	var derivedECs []string
	seen := make(map[string]bool)

	appendLiteral := func(text string) {
		candidate := normalizeEdgeCaseCandidate(text)
		if candidate == "" {
			return
		}
		key := strings.ToLower(candidate)
		if seen[key] {
			return
		}
		seen[key] = true
		literalECs = append(literalECs, candidate)
	}

	appendDerived := func(text string) {
		candidate := normalizeEdgeCaseCandidate(text)
		if candidate == "" || !isEdgeCaseCandidate(candidate) {
			return
		}
		key := strings.ToLower(candidate)
		if seen[key] {
			return
		}
		seen[key] = true
		derivedECs = append(derivedECs, candidate)
	}

	// Pass 1: literal parse from explicit edge_cases answer (no keyword filter).
	for _, answer := range answers {
		if answer.QuestionID == "edge_cases" {
			for _, item := range toBulletList(answer.Answer) {
				appendLiteral(item)
			}
		}
	}

	// Pass 2: derive from premise revisions and disagreed premises only.
	// (Keyword harvesting from unrelated answers was removed — it caused
	// unrelated sentences to leak into Edge Cases.)
	for _, premise := range premises {
		if premise.Revision != nil && strings.TrimSpace(*premise.Revision) != "" {
			appendDerived(*premise.Revision)
			continue
		}
		if !premise.Agreed {
			appendDerived(premise.Text)
		}
	}

	// Literal (explicit) first, premise-derived after.
	return append(literalECs, derivedECs...)
}

// =============================================================================
// Section relevance check
// =============================================================================

// checkSectionRelevance checks if a concern section is relevant based on classification.
func checkSectionRelevance(concern state.ConcernDefinition, classification *state.SpecClassification) map[string]bool {
	relevance := make(map[string]bool, len(concern.SpecSections))

	// No classification — default all to not relevant (clean spec)
	if classification == nil {
		for _, section := range concern.SpecSections {
			relevance[section] = false
		}
		return relevance
	}

	for _, section := range concern.SpecSections {
		lower := strings.ToLower(section)

		if strings.Contains(lower, "design") || strings.Contains(lower, "mobile") ||
			strings.Contains(lower, "layout") || strings.Contains(lower, "interaction") {
			relevance[section] = classification.InvolvesWebUI
		} else if strings.Contains(lower, "contributor") || strings.Contains(lower, "public api") ||
			strings.Contains(lower, "api surface") {
			relevance[section] = classification.InvolvesPublicAPI
		} else if strings.Contains(lower, "migration") || strings.Contains(lower, "deprecation") ||
			strings.Contains(lower, "backward") || strings.Contains(lower, "compatibility") {
			relevance[section] = classification.InvolvesMigration
		} else if strings.Contains(lower, "audit") || strings.Contains(lower, "access control") ||
			strings.Contains(lower, "data handling") {
			relevance[section] = classification.InvolvesDataHandling
		} else {
			relevance[section] = true
		}
	}

	return relevance
}

// =============================================================================
// RenderSpec
// =============================================================================

// renderOptions bundles optional override inputs so RenderSpec keeps its
// positional signature stable for existing test callers.
type renderOptions struct {
	overrideTasks      []state.SpecTask
	overrideOutOfScope []string
}

// RenderOption customizes RenderSpec output. Use the WithX helpers to set each
// field — callers that pass no options get the classic auto-derived behavior.
type RenderOption func(*renderOptions)

// WithTaskOverride replaces the auto-derived Tasks list with the provided
// SpecTask slice. IDs are preserved as-is (no renumbering). Pass nil/empty to
// keep the auto-derived list.
func WithTaskOverride(tasks []state.SpecTask) RenderOption {
	return func(o *renderOptions) { o.overrideTasks = tasks }
}

// WithOutOfScopeOverride appends additional out-of-scope items on top of the
// scope-boundary answer.
func WithOutOfScopeOverride(items []string) RenderOption {
	return func(o *renderOptions) { o.overrideOutOfScope = items }
}

// RenderSpec renders a full spec.md markdown document from the provided data.
// When tddMode is true, test-related tasks are moved to the beginning of the task list.
func RenderSpec(
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

	lines = append(lines, fmt.Sprintf("# Spec: %s", specName))
	lines = append(lines, "")
	lines = append(lines, "## Status: draft")
	lines = append(lines, "")

	if len(concerns) > 0 {
		ids := make([]string, len(concerns))
		for i, c := range concerns {
			ids[i] = c.ID
		}
		lines = append(lines, fmt.Sprintf("## Concerns: %s", strings.Join(ids, ", ")))
		lines = append(lines, "")
	}

	// Summary from discovery
	lines = append(lines, "## Discovery Answers")
	lines = append(lines, "")

	for _, answer := range answers {
		lines = append(lines, fmt.Sprintf("### %s", answer.QuestionID))
		lines = append(lines, "")
		lines = append(lines, answer.Answer)
		lines = append(lines, "")
	}

	// Concern-specific sections — only render relevant ones
	for _, concern := range concerns {
		if len(concern.SpecSections) > 0 {
			relevance := checkSectionRelevance(concern, classification)

			for _, section := range concern.SpecSections {
				if !relevance[section] {
					continue
				}
				lines = append(lines, fmt.Sprintf("## %s (%s)", section, concern.ID))
				lines = append(lines, "")
				lines = append(lines, "_To be addressed during execution._")
				lines = append(lines, "")
			}
		}
	}

	// Registry tables — structured tables from concern review dimensions
	for _, concern := range concerns {
		if len(concern.Registries) == 0 {
			continue
		}

		for _, regID := range concern.Registries {
			var dim *state.ReviewDimension
			for i := range concern.ReviewDimensions {
				if concern.ReviewDimensions[i].ID == regID {
					dim = &concern.ReviewDimensions[i]
					break
				}
			}
			if dim == nil {
				continue
			}

			lines = append(lines, fmt.Sprintf("## %s (%s)", dim.Label, concern.ID))
			lines = append(lines, "")
			lines = append(lines, fmt.Sprintf("_%s_", dim.Prompt))
			lines = append(lines, "")

			switch regID {
			case "error-rescue":
				lines = append(lines, "| Codepath | What Can Go Wrong | Exception Class | Rescued? | Recovery Action | User Sees |")
				lines = append(lines, "|----------|-------------------|-----------------|----------|----------------|-----------|")
				lines = append(lines, "| _To be filled during execution._ | | | | | |")
			case "failure-modes":
				lines = append(lines, "| Codepath | Failure Mode | Rescued? | Tested? | User Sees | Logged? |")
				lines = append(lines, "|----------|--------------|----------|---------|-----------|---------|")
				lines = append(lines, "| _To be filled during execution._ | | | | | |")
			case "test-plan":
				lines = append(lines, "| Behavior | Test Layer | Happy Path | Failure Path | Edge Case |")
				lines = append(lines, "|----------|------------|------------|--------------|-----------|")
				lines = append(lines, "| _To be filled during execution._ | | | | |")
			default:
				lines = append(lines, "_To be filled during execution._")
			}
			lines = append(lines, "")
		}
	}

	// Decisions
	if len(decisions) > 0 {
		lines = append(lines, "## Decisions")
		lines = append(lines, "")
		lines = append(lines, "| # | Decision | Choice | Promoted |")
		lines = append(lines, "|---|----------|--------|----------|")

		for i, d := range decisions {
			promoted := "no"
			if d.Promoted {
				promoted = "yes"
			}
			lines = append(lines, fmt.Sprintf("| %d | %s | %s | %s |", i+1, d.Question, d.Choice, promoted))
		}

		lines = append(lines, "")
	}

	// Out of Scope — formatted as bullet list
	var scopeAnswer *state.DiscoveryAnswer
	for i := range answers {
		if answers[i].QuestionID == "scope_boundary" {
			scopeAnswer = &answers[i]
			break
		}
	}

	if scopeAnswer != nil || len(opts.overrideOutOfScope) > 0 {
		lines = append(lines, "## Out of Scope")
		lines = append(lines, "")
		if scopeAnswer != nil {
			for _, item := range toBulletList(scopeAnswer.Answer) {
				lines = append(lines, fmt.Sprintf("- %s", item))
			}
		}
		for _, item := range opts.overrideOutOfScope {
			lines = append(lines, fmt.Sprintf("- %s", item))
		}
		lines = append(lines, "")
	}

	edgeCases := DeriveEdgeCases(answers, premises)
	if len(edgeCases) > 0 {
		lines = append(lines, "## Edge Cases")
		lines = append(lines, "")
		for _, item := range edgeCases {
			lines = append(lines, fmt.Sprintf("- %s", item))
		}
		lines = append(lines, "")
	}

	// Tasks — user override wins, otherwise auto-derived from discovery.
	lines = append(lines, "## Tasks")
	lines = append(lines, "")
	if len(opts.overrideTasks) > 0 {
		// Use struct-based override: IDs are stable, Completed state is respected.
		for _, t := range opts.overrideTasks {
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
		// Auto-derive tasks and number from 1.
		for i, task := range DeriveTasks(answers, decisions, tddMode) {
			lines = append(lines, fmt.Sprintf("- [ ] task-%d: %s", i+1, task))
		}
	}
	lines = append(lines, "")

	// Verification (from verification answer)
	var verificationAnswer *state.DiscoveryAnswer
	for i := range answers {
		if answers[i].QuestionID == "verification" {
			verificationAnswer = &answers[i]
			break
		}
	}

	lines = append(lines, "## Verification")
	lines = append(lines, "")
	if verificationAnswer != nil {
		items := toBulletList(verificationAnswer.Answer)
		for _, item := range items {
			lines = append(lines, fmt.Sprintf("- %s", item))
		}
	} else {
		lines = append(lines, "_To be defined._")
	}
	lines = append(lines, "")

	// Custom Acceptance Criteria
	if len(customACs) > 0 {
		lines = append(lines, "## Custom Acceptance Criteria")
		lines = append(lines, "")
		for _, ac := range customACs {
			lines = append(lines, fmt.Sprintf("- %s _-- %s, %s_", ac.Text, ac.User, string(ac.AddedInPhase)))
		}
		lines = append(lines, "")
	}

	// Notes (multi-user annotations, excluding task notes)
	var filteredNotes []state.SpecNote
	for _, n := range specNotes {
		if !strings.HasPrefix(n.Text, "[TASK] ") {
			filteredNotes = append(filteredNotes, n)
		}
	}
	if len(filteredNotes) > 0 {
		lines = append(lines, "## Notes")
		lines = append(lines, "")
		for _, note := range filteredNotes {
			lines = append(lines, fmt.Sprintf("- %s _-- %s, %s_", note.Text, note.User, string(note.Phase)))
		}
		lines = append(lines, "")
	}

	// Phase transition history
	if len(transitionHistory) > 0 {
		lines = append(lines, "## Transition History")
		lines = append(lines, "")
		lines = append(lines, "| From | To | User | Timestamp | Reason |")
		lines = append(lines, "|------|----|------|-----------|--------|")
		for _, t := range transitionHistory {
			reason := "-"
			if t.Reason != nil {
				reason = *t.Reason
			}
			lines = append(lines, fmt.Sprintf("| %s | %s | %s | %s | %s |",
				string(t.From), string(t.To), t.User, t.Timestamp, reason))
		}
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// suppress unused import warning for ellipsisRune
var _ = ellipsisRune
