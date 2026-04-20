package model

import "github.com/pragmataW/tddmaster/internal/state"

// Question represents one discovery question.
type Question struct {
	ID       string   `json:"id"`
	Text     string   `json:"text"`
	Concerns []string `json:"concerns"`
}

// QuestionWithExtras is a Question enriched with concern-specific extras.
type QuestionWithExtras struct {
	Question
	Extras []state.ConcernExtra `json:"extras"`
}

// Questions is the canonical list of hardcoded discovery questions (v0.1).
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

// BuiltInExtra is a concern-extra-shaped addendum that is always injected,
// regardless of active concerns.
type BuiltInExtra struct {
	QuestionID string
	Text       string
}

// BuiltInExtras are always injected regardless of active concerns.
var BuiltInExtras = []BuiltInExtra{
	{
		QuestionID: "verification",
		Text:       "What tests should be written? (unit, integration, e2e — be specific about what behavior to test)",
	},
	{
		QuestionID: "verification",
		Text:       "What documentation needs updating? (README, API docs, CHANGELOG, inline comments)",
	},
}
