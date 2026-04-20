package service

import (
	"strings"

	"github.com/pragmataW/tddmaster/internal/state"
)

// checkSectionRelevance maps each concern section name to a boolean indicating
// whether it should be rendered, using the active spec classification. When
// classification is nil, ALL sections default to not-relevant (the legacy
// "clean spec" behavior); see docs/bugs.md S1 for the known edge case where
// generic sections are dropped.
func checkSectionRelevance(concern state.ConcernDefinition, classification *state.SpecClassification) map[string]bool {
	relevance := make(map[string]bool, len(concern.SpecSections))

	if classification == nil {
		for _, section := range concern.SpecSections {
			relevance[section] = false
		}
		return relevance
	}

	for _, section := range concern.SpecSections {
		lower := strings.ToLower(section)

		switch {
		case containsAny(lower, "design", "mobile", "layout", "interaction"):
			relevance[section] = classification.InvolvesWebUI
		case containsAny(lower, "contributor", "public api", "api surface"):
			relevance[section] = classification.InvolvesPublicAPI
		case containsAny(lower, "migration", "deprecation", "backward", "compatibility"):
			relevance[section] = classification.InvolvesMigration
		case containsAny(lower, "audit", "access control", "data handling"):
			relevance[section] = classification.InvolvesDataHandling
		default:
			relevance[section] = true
		}
	}

	return relevance
}

func containsAny(haystack string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(haystack, n) {
			return true
		}
	}
	return false
}
