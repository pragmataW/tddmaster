
package context

import (
	"github.com/pragmataW/tddmaster/internal/state"
)

// =============================================================================
// Question Definition
// =============================================================================

// Question represents a discovery question.
type Question struct {
	ID       string   `json:"id"`
	Text     string   `json:"text"`
	Concerns []string `json:"concerns"`
}

// =============================================================================
// Hardcoded Questions (v0.1)
// =============================================================================

// Questions is the list of hardcoded discovery questions.
var Questions = []Question{
	{
		ID:       "status_quo",
		Text:     "What does the user do today without this feature?",
		Concerns: []string{"product:status_quo", "eng:replace_scope", "qa:regression_risk"},
	},
	{
		ID:   "ambition",
		Text: "Describe the 1-star and 10-star versions.",
		Concerns: []string{
			"product:scope_direction",
			"eng:complexity_tier",
			"qa:test_depth",
		},
	},
	{
		ID:   "reversibility",
		Text: "Does this change involve an irreversible decision?",
		Concerns: []string{
			"product:one_way_door",
			"eng:migration_strategy",
			"qa:verification_stringency",
		},
	},
	{
		ID:   "user_impact",
		Text: "Does this change affect existing users' behavior?",
		Concerns: []string{
			"product:breaking_change",
			"eng:backward_compat",
			"qa:regression_tests",
		},
	},
	{
		ID:   "verification",
		Text: "How do you verify this works correctly?",
		Concerns: []string{
			"product:success_metric",
			"eng:test_strategy",
			"qa:acceptance_criteria",
		},
	},
	{
		ID:       "scope_boundary",
		Text:     "What should this feature NOT do?",
		Concerns: []string{"product:focus", "eng:out_of_scope", "qa:negative_tests"},
	},
	{
		ID:   "edge_cases",
		Text: "Which boundary conditions, error states, or exceptional inputs could cause this change to misbehave? List cases that need protective tests.",
		Concerns: []string{
			"qa:edge_coverage",
			"eng:test_strategy",
			"product:risk_surface",
		},
	},
}

// =============================================================================
// Question With Extras
// =============================================================================

// QuestionWithExtras is a Question enriched with concern-specific extras.
type QuestionWithExtras struct {
	Question
	Extras []state.ConcernExtra `json:"extras"`
}

// builtInExtras are always injected regardless of active concerns.
var builtInExtras = []struct {
	QuestionID string
	Text       string
}{
	{
		QuestionID: "verification",
		Text:       "What tests should be written? (unit, integration, e2e — be specific about what behavior to test)",
	},
	{
		QuestionID: "verification",
		Text:       "What documentation needs updating? (README, API docs, CHANGELOG, inline comments)",
	},
}

// GetQuestionsWithExtras returns all questions enriched with built-in and concern-specific extras.
func GetQuestionsWithExtras(activeConcerns []state.ConcernDefinition) []QuestionWithExtras {
	result := make([]QuestionWithExtras, len(Questions))
	for i, q := range Questions {
		var extras []state.ConcernExtra

		// Built-in extras first
		for _, bi := range builtInExtras {
			if bi.QuestionID == q.ID {
				extras = append(extras, state.ConcernExtra{
					QuestionID: bi.QuestionID,
					Text:       bi.Text,
				})
			}
		}

		// Concern-specific extras
		concernExtras := GetConcernExtras(activeConcerns, q.ID)
		extras = append(extras, concernExtras...)

		result[i] = QuestionWithExtras{
			Question: q,
			Extras:   extras,
		}
	}
	return result
}

// GetNextUnanswered returns the first unanswered question, or nil if all answered.
func GetNextUnanswered(questions []QuestionWithExtras, answers []state.DiscoveryAnswer) *QuestionWithExtras {
	answeredIDs := make(map[string]bool, len(answers))
	for _, a := range answers {
		answeredIDs[a.QuestionID] = true
	}

	for i := range questions {
		if !answeredIDs[questions[i].ID] {
			return &questions[i]
		}
	}

	return nil
}

// IsDiscoveryComplete returns true when all questions have been answered.
func IsDiscoveryComplete(answers []state.DiscoveryAnswer) bool {
	answeredIDs := make(map[string]bool, len(answers))
	for _, a := range answers {
		answeredIDs[a.QuestionID] = true
	}

	for _, q := range Questions {
		if !answeredIDs[q.ID] {
			return false
		}
	}

	return true
}
