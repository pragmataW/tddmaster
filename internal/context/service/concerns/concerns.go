// Package concerns collects concern-derived reminders, review dimensions,
// tensions, and registry pointers from active concern definitions. All
// filtering honours the spec classification when one is available.
package concerns

import (
	"strings"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/defaults"
	"github.com/pragmataW/tddmaster/internal/state"
)

// LoadDefault returns the embedded default concern definitions.
func LoadDefault() []state.ConcernDefinition {
	return defaults.DefaultConcerns()
}

// GetExtras collects extras for a specific question from all concerns.
func GetExtras(concerns []state.ConcernDefinition, questionID string) []state.ConcernExtra {
	var extras []state.ConcernExtra
	for _, concern := range concerns {
		for _, extra := range concern.Extras {
			if extra.QuestionID == questionID {
				extras = append(extras, extra)
			}
		}
	}
	return extras
}

// reminderAllowedByClassification reports whether a scoped reminder should be
// surfaced given the active spec classification. Tier1 reminders (empty scope)
// always pass.
func reminderAllowedByClassification(r state.ConcernReminder, classification *state.SpecClassification) bool {
	if !r.HasScope() || classification == nil {
		return true
	}
	if r.AppliesToScope(state.ConcernReminderScopeUI) && !classification.InvolvesWebUI {
		return false
	}
	if r.AppliesToScope(state.ConcernReminderScopeAPI) && !classification.InvolvesPublicAPI {
		return false
	}
	return true
}

// GetReminders collects reminders from concerns, filtered by classification when available.
func GetReminders(concerns []state.ConcernDefinition, classification *state.SpecClassification) []string {
	var out []string
	for _, concern := range concerns {
		for _, reminder := range concern.Reminders {
			if !reminderAllowedByClassification(reminder, classification) {
				continue
			}
			out = append(out, concern.ID+": "+reminder.Text)
		}
	}
	return out
}

// SplitRemindersByTier splits reminders into tier1 (general) and tier2 (scoped).
func SplitRemindersByTier(concerns []state.ConcernDefinition) (tier1 []string, tier2 []string) {
	for _, concern := range concerns {
		for _, reminder := range concern.Reminders {
			prefixed := concern.ID + ": " + reminder.Text
			if reminder.HasScope() {
				tier2 = append(tier2, prefixed)
			} else {
				tier1 = append(tier1, prefixed)
			}
		}
	}
	return tier1, tier2
}

// uiFileExtensions and apiFileExtensions classify file paths for tier2 delivery.
var uiFileExtensions = map[string]bool{".tsx": true, ".jsx": true, ".html": true, ".css": true, ".svelte": true, ".vue": true}
var apiFileExtensions = map[string]bool{".ts": true, ".go": true, ".py": true, ".rs": true}

// Tier2ForFile returns tier2 reminders applicable to a specific file.
func Tier2ForFile(concerns []state.ConcernDefinition, filePath string, classification *state.SpecClassification) []string {
	ext := ""
	if idx := strings.LastIndex(filePath, "."); idx >= 0 {
		ext = filePath[idx:]
	}
	isUI := uiFileExtensions[ext]
	isAPI := apiFileExtensions[ext]

	var out []string
	for _, concern := range concerns {
		for _, reminder := range concern.Reminders {
			if !reminder.HasScope() {
				continue
			}
			if reminder.AppliesToScope(state.ConcernReminderScopeUI) && !isUI {
				continue
			}
			if reminder.AppliesToScope(state.ConcernReminderScopeAPI) &&
				(!isAPI || classification == nil || !classification.InvolvesPublicAPI) {
				continue
			}
			out = append(out, concern.ID+": "+reminder.Text)
		}
	}
	return out
}

// DetectTensions detects tensions between active concerns.
func DetectTensions(activeConcerns []state.ConcernDefinition) []model.ConcernTension {
	ids := make(map[string]bool, len(activeConcerns))
	for _, c := range activeConcerns {
		ids[c.ID] = true
	}

	var tensions []model.ConcernTension
	if ids["move-fast"] && ids["compliance"] {
		tensions = append(tensions, model.ConcernTension{
			Between: []string{"move-fast", "compliance"},
			Issue:   "Speed vs traceability — shortcuts may violate audit requirements.",
		})
	}
	if ids["move-fast"] && ids["long-lived"] {
		tensions = append(tensions, model.ConcernTension{
			Between: []string{"move-fast", "long-lived"},
			Issue:   "Shipping speed vs maintainability — tech debt decisions need human approval.",
		})
	}
	if ids["beautiful-product"] && ids["move-fast"] {
		tensions = append(tensions, model.ConcernTension{
			Between: []string{"beautiful-product", "move-fast"},
			Issue:   "Design polish vs speed — which UI states can be deferred?",
		})
	}
	if ids["well-engineered"] && ids["move-fast"] {
		tensions = append(tensions, model.ConcernTension{
			Between: []string{"well-engineered", "move-fast"},
			Issue:   "Engineering rigor vs shipping speed — which quality dimensions (tests, observability, security hardening) can be deferred to v2?",
		})
	}
	if ids["well-engineered"] && ids["learning-project"] {
		tensions = append(tensions, model.ConcernTension{
			Between: []string{"well-engineered", "learning-project"},
			Issue:   "Engineering standards vs experimentation freedom — how much test/security/observability rigor is appropriate for an experiment?",
		})
	}
	return tensions
}

// GetReviewDimensions collects review dimensions from active concerns, filtered by classification.
func GetReviewDimensions(activeConcerns []state.ConcernDefinition, classification *state.SpecClassification) []model.TaggedReviewDimension {
	var dimensions []model.TaggedReviewDimension
	for _, concern := range activeConcerns {
		for _, dim := range concern.ReviewDimensions {
			if classification != nil {
				if dim.Scope == "ui" && !classification.InvolvesWebUI {
					continue
				}
				if dim.Scope == "api" && !classification.InvolvesPublicAPI {
					continue
				}
				if dim.Scope == "data" && !classification.InvolvesDataHandling {
					continue
				}
			}
			dimensions = append(dimensions, model.TaggedReviewDimension{
				ReviewDimension: dim,
				ConcernID:       concern.ID,
			})
		}
	}
	return dimensions
}

// GetRegistryDimensionIDs collects all registry dimension IDs from active concerns.
func GetRegistryDimensionIDs(activeConcerns []state.ConcernDefinition) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, concern := range activeConcerns {
		for _, reg := range concern.Registries {
			if !seen[reg] {
				seen[reg] = true
				ids = append(ids, reg)
			}
		}
	}
	return ids
}

// GetDreamStatePrompts collects dream state prompts from active concerns.
func GetDreamStatePrompts(activeConcerns []state.ConcernDefinition) []string {
	var prompts []string
	for _, c := range activeConcerns {
		if c.DreamStatePrompt != nil && len(*c.DreamStatePrompt) > 0 {
			prompts = append(prompts, *c.DreamStatePrompt)
		}
	}
	return prompts
}
