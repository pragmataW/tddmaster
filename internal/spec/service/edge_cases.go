package service

import (
	"regexp"
	"strings"

	"github.com/pragmataW/tddmaster/internal/state"
)

var (
	edgeCasePrefixRe  = regexp.MustCompile(`(?i)^(edge[- ]?cases?|watch(?:\s+out)?\s+for|consider)\s*[:\-]\s*`)
	edgeCaseKeywordRe = regexp.MustCompile(`(?i)\b(edge[- ]?cases?|empty|zero|missing|invalid|duplicate|timeout|slow|latency|partial|retry|offline|boundary|large|long|unicode|whitespace|nil|null|404|409|429|500|error|failure|fallback|concurrent|race)\b`)
)

// DeriveEdgeCases extracts concrete edge cases from two intentional sources:
// (1) the explicit "edge_cases" answer (parsed literally, no keyword filter),
// and (2) disagreed/revised premises. Keyword harvesting from unrelated
// discovery answers was removed because it bled sentences containing
// incidental words like "error", "fallback", "nil", "race" from other
// sections into the Edge Cases list.
// The resulting list is de-duplicated and preserves first-seen order within
// each group.
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

	for _, answer := range answers {
		if answer.QuestionID == "edge_cases" {
			for _, item := range toBulletList(answer.Answer) {
				appendLiteral(item)
			}
		}
	}

	for _, premise := range premises {
		if premise.Revision != nil && strings.TrimSpace(*premise.Revision) != "" {
			appendDerived(*premise.Revision)
			continue
		}
		if !premise.Agreed {
			appendDerived(premise.Text)
		}
	}

	return append(literalECs, derivedECs...)
}

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
