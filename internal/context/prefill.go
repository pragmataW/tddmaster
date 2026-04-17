package context

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/pragmataW/tddmaster/internal/state"
)

const richUserContextThreshold = 200

// ExtractUserContextPrefills turns a rich listen-first message into reviewable discovery suggestions.
func ExtractUserContextPrefills(raw string) []state.DiscoveryPrefillQuestion {
	trimmed := strings.TrimSpace(raw)
	if utf8.RuneCountInString(trimmed) <= richUserContextThreshold {
		return nil
	}

	segments := splitPrefillSegments(trimmed)
	results := make([]state.DiscoveryPrefillQuestion, 0, len(Questions))
	for _, question := range Questions {
		items := extractPrefillsForQuestion(question.ID, segments, trimmed)
		if len(items) == 0 {
			continue
		}
		results = append(results, state.DiscoveryPrefillQuestion{
			QuestionID: question.ID,
			Items:      items,
		})
	}
	return results
}

func extractPrefillsForQuestion(questionID string, segments []string, fullText string) []state.DiscoveryPrefillItem {
	items := make([]state.DiscoveryPrefillItem, 0, 3)
	for _, segment := range segments {
		if kind, ok := classifyStatedPrefill(questionID, segment); ok {
			items = appendPrefillItem(items, kind, segment, fmt.Sprintf("Quoted from initial context: %q", clipPrefillBasis(segment)))
		}
	}

	if len(items) > 0 {
		return items
	}

	inferredText, basis, ok := classifyInferredPrefill(questionID, fullText)
	if !ok {
		return nil
	}
	return appendPrefillItem(items, "INFERRED", inferredText, basis)
}

func classifyStatedPrefill(questionID, segment string) (string, bool) {
	lower := strings.ToLower(segment)
	switch questionID {
	case "status_quo":
		return "STATED", containsAny(lower, "today", "currently", "right now", "manual", "without this", "current workflow", "current process")
	case "ambition":
		return "STATED", containsAny(lower, "1-star", "10-star", "mvp", "phase 2", "later", "eventually", "future", "ideal", "stretch", "v1", "v2")
	case "reversibility":
		return "STATED", containsAny(lower, "irreversible", "one-way", "can't undo", "cannot undo", "rollback", "roll back", "migration", "drop column", "drop table", "rename", "remove field", "delete data", "public api")
	case "user_impact":
		return "STATED", containsAny(lower, "existing users", "current users", "breaking change", "backward compatible", "user-facing", "behavior change", "api consumers", "customers")
	case "verification":
		return "STATED", containsAny(lower, "verify", "test", "acceptance", "qa", "documentation", "docs", "done when", "success means", "regression")
	case "scope_boundary":
		return "STATED", containsAny(lower, "out of scope", "not in scope", "won't", "will not", "should not", "do not", "doesn't need", "not for v1", "defer")
	case "edge_cases":
		return "STATED", containsAny(lower, "edge case", "timeout", "error", "failure", "retry", "duplicate", "partial", "race", "concurrent", "empty", "nil", "null", "offline", "latency", "429", "500", "exception")
	default:
		return "", false
	}
}

func classifyInferredPrefill(questionID, fullText string) (string, string, bool) {
	lower := strings.ToLower(fullText)
	switch questionID {
	case "status_quo":
		if containsAny(lower, "manual", "spreadsheet", "workaround", "replace", "today", "currently") {
			return "There is an existing workflow or workaround that this change replaces or improves.",
				"Initial context mentions a manual/current workflow signal.", true
		}
	case "ambition":
		if containsAny(lower, "mvp", "phase 2", "later", "eventually", "future", "stretch") {
			return "The user is implying a staged rollout between an MVP slice and a more complete follow-up.",
				"Initial context includes phased-delivery language such as MVP, later, or future work.", true
		}
	case "reversibility":
		if containsAny(lower, "migration", "schema", "database", "backfill", "rename", "drop", "persisted", "public api") {
			return "This likely includes a hard-to-reverse change and needs rollback or compatibility planning.",
				"Initial context mentions migration/schema/API signals that are often one-way or costly to undo.", true
		}
	case "user_impact":
		if containsAny(lower, "existing users", "current users", "customer", "client", "regression", "backward compatible", "api consumer") {
			return "This likely affects existing user behavior or client expectations and needs regression attention.",
				"Initial context mentions existing-user, client, or regression signals.", true
		}
	case "verification":
		if containsAny(lower, "bug", "regression", "correct", "verify", "test", "acceptance", "documentation") {
			return "Verification should prove the change works and that adjacent behavior does not regress.",
				"Initial context mentions bug/regression/correctness signals that imply explicit verification coverage.", true
		}
	case "scope_boundary":
		if containsAny(lower, "mvp", "phase 2", "later", "defer", "not in scope", "out of scope") {
			return "Some adjacent ideas are intentionally deferred and should be stated as out of scope.",
				"Initial context includes phased-scope or defer language.", true
		}
	case "edge_cases":
		if containsAny(lower, "timeout", "retry", "partial", "concurrent", "migration", "external api", "sync", "import", "failure", "error") {
			return "Protective tests are likely needed for failure, retry, and partial-state edge cases.",
				"Initial context mentions failure or integration signals that usually create edge-case risk.", true
		}
	}
	return "", "", false
}

func splitPrefillSegments(raw string) []string {
	normalized := strings.NewReplacer("\r\n", "\n", "\r", "\n", "\t", " ").Replace(raw)
	fields := strings.Fields(normalized)
	if len(fields) == 0 {
		return nil
	}
	normalized = strings.Join(fields, " ")

	splitter := func(r rune) bool {
		switch r {
		case '.', '!', '?', ';', '\n':
			return true
		default:
			return false
		}
	}

	parts := strings.FieldsFunc(normalized, splitter)
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		segment := strings.TrimSpace(part)
		if utf8.RuneCountInString(segment) < 12 {
			continue
		}
		segments = append(segments, segment)
	}
	return segments
}

func appendPrefillItem(items []state.DiscoveryPrefillItem, itemType, text, basis string) []state.DiscoveryPrefillItem {
	normalized := normalizePrefillText(text)
	for _, existing := range items {
		if normalizePrefillText(existing.Text) == normalized {
			return items
		}
	}
	return append(items, state.DiscoveryPrefillItem{
		Type:  itemType,
		Text:  strings.TrimSpace(text),
		Basis: strings.TrimSpace(basis),
	})
}

func normalizePrefillText(text string) string {
	return strings.ToLower(strings.Join(strings.Fields(text), " "))
}

func clipPrefillBasis(text string) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= 120 {
		return string(runes)
	}
	return string(runes[:117]) + "..."
}

func containsAny(text string, patterns ...string) bool {
	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}
