
// Package context provides context compilation and concern operations for tddmaster.
package context

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pragmataW/tddmaster/internal/defaults"
	"github.com/pragmataW/tddmaster/internal/state"
)

// =============================================================================
// String Constants (S1192: avoid duplicate string literals)
// =============================================================================

const (
	concernStrUIElement           = "ui element"
	concernStrDesignIntentionality = "design intentionality"
	concernStrInteractionStates   = "interaction states"
	concernStrEdgeCaseCheck       = "edge case check"
	concernStrLoadingState        = "loading state"
	concernStrAPIDoc              = "api doc"
	concernStrEndpointShouldBe    = "endpoint should be"
	concernStrMoveFast            = "move-fast"
	concernStrWellEngineered      = "well-engineered"
)

// =============================================================================
// Concern Loader
// =============================================================================

// LoadConcerns loads all concern JSON files from a directory.
func LoadConcerns(dirPath string) ([]state.ConcernDefinition, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return []state.ConcernDefinition{}, nil // Directory doesn't exist yet
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	var concerns []state.ConcernDefinition
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dirPath, name))
		if err != nil {
			return nil, err
		}
		var c state.ConcernDefinition
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, err
		}
		concerns = append(concerns, c)
	}

	return concerns, nil
}

// LoadDefaultConcerns returns the embedded default concern definitions.
func LoadDefaultConcerns() []state.ConcernDefinition {
	return defaults.DefaultConcerns()
}

// =============================================================================
// Concern Operations
// =============================================================================

// GetConcernExtras collects extras for a specific question from all concerns.
func GetConcernExtras(concerns []state.ConcernDefinition, questionID string) []state.ConcernExtra {
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

// GetReminders collects reminders from concerns, filtered by classification when available.
func GetReminders(concerns []state.ConcernDefinition, classification *state.SpecClassification) []string {
	var reminders []string

	for _, concern := range concerns {
		for _, reminder := range concern.Reminders {
			if classification != nil {
				lower := strings.ToLower(reminder)

				// AI slop / design-specific → only when involvesWebUI
				if (strings.Contains(lower, "slop") || strings.Contains(lower, concernStrUIElement) ||
					strings.Contains(lower, concernStrDesignIntentionality) ||
					strings.Contains(lower, concernStrInteractionStates) ||
					strings.Contains(lower, concernStrEdgeCaseCheck) ||
					strings.Contains(lower, concernStrLoadingState)) &&
					!classification.InvolvesWebUI {
					continue
				}

				// API documentation → only when involvesPublicAPI
				if (strings.Contains(lower, concernStrAPIDoc) || strings.Contains(lower, concernStrEndpointShouldBe)) &&
					!classification.InvolvesPublicAPI {
					continue
				}
			}

			reminders = append(reminders, concern.ID+": "+reminder)
		}
	}

	return reminders
}

// isFileSpecificReminder checks if a reminder is file-type-specific (UI/API/migration).
func isFileSpecificReminder(reminder string) bool {
	lower := strings.ToLower(reminder)
	return strings.Contains(lower, "slop") || strings.Contains(lower, concernStrUIElement) ||
		strings.Contains(lower, concernStrDesignIntentionality) ||
		strings.Contains(lower, concernStrInteractionStates) ||
		strings.Contains(lower, concernStrEdgeCaseCheck) ||
		strings.Contains(lower, concernStrLoadingState) ||
		strings.Contains(lower, concernStrAPIDoc) || strings.Contains(lower, concernStrEndpointShouldBe) ||
		strings.Contains(lower, "migration") || strings.Contains(lower, "rollback")
}

// SplitRemindersByTier splits reminders into tier1 (general) and tier2 (file-specific).
func SplitRemindersByTier(concerns []state.ConcernDefinition) (tier1 []string, tier2 []string) {
	for _, concern := range concerns {
		for _, reminder := range concern.Reminders {
			prefixed := concern.ID + ": " + reminder
			if isFileSpecificReminder(reminder) {
				tier2 = append(tier2, prefixed)
			} else {
				tier1 = append(tier1, prefixed)
			}
		}
	}
	return tier1, tier2
}

// GetTier2RemindersForFile returns tier2 reminders applicable to a specific file.
func GetTier2RemindersForFile(concerns []state.ConcernDefinition, filePath string, classification *state.SpecClassification) []string {
	ext := ""
	if idx := strings.LastIndex(filePath, "."); idx >= 0 {
		ext = filePath[idx:]
	}

	uiExts := map[string]bool{".tsx": true, ".jsx": true, ".html": true, ".css": true, ".svelte": true, ".vue": true}
	apiExts := map[string]bool{".ts": true, ".go": true, ".py": true, ".rs": true}
	isUI := uiExts[ext]
	isAPI := apiExts[ext]

	var reminders []string

	for _, concern := range concerns {
		for _, reminder := range concern.Reminders {
			if !isFileSpecificReminder(reminder) {
				continue
			}

			lower := strings.ToLower(reminder)

			// UI reminders → only for UI files
			if (strings.Contains(lower, "slop") || strings.Contains(lower, concernStrUIElement) ||
				strings.Contains(lower, concernStrDesignIntentionality) ||
				strings.Contains(lower, concernStrInteractionStates) ||
				strings.Contains(lower, concernStrEdgeCaseCheck) ||
				strings.Contains(lower, concernStrLoadingState)) && !isUI {
				continue
			}

			// API reminders → only for API files with involvesPublicAPI
			if (strings.Contains(lower, concernStrAPIDoc) || strings.Contains(lower, concernStrEndpointShouldBe)) &&
				(!isAPI || classification == nil || !classification.InvolvesPublicAPI) {
				continue
			}

			reminders = append(reminders, concern.ID+": "+reminder)
		}
	}

	return reminders
}

// =============================================================================
// Tensions
// =============================================================================

// ConcernTension represents a tension between two active concerns.
type ConcernTension struct {
	Between []string `json:"between"`
	Issue   string   `json:"issue"`
}

// DetectTensions detects tensions between active concerns.
func DetectTensions(activeConcerns []state.ConcernDefinition) []ConcernTension {
	var tensions []ConcernTension

	ids := make(map[string]bool)
	for _, c := range activeConcerns {
		ids[c.ID] = true
	}

	if ids[concernStrMoveFast] && ids["compliance"] {
		tensions = append(tensions, ConcernTension{
			Between: []string{concernStrMoveFast, "compliance"},
			Issue:   "Speed vs traceability — shortcuts may violate audit requirements.",
		})
	}

	if ids[concernStrMoveFast] && ids["long-lived"] {
		tensions = append(tensions, ConcernTension{
			Between: []string{concernStrMoveFast, "long-lived"},
			Issue:   "Shipping speed vs maintainability — tech debt decisions need human approval.",
		})
	}

	if ids["beautiful-product"] && ids[concernStrMoveFast] {
		tensions = append(tensions, ConcernTension{
			Between: []string{"beautiful-product", concernStrMoveFast},
			Issue:   "Design polish vs speed — which UI states can be deferred?",
		})
	}

	if ids[concernStrWellEngineered] && ids[concernStrMoveFast] {
		tensions = append(tensions, ConcernTension{
			Between: []string{concernStrWellEngineered, concernStrMoveFast},
			Issue:   "Engineering rigor vs shipping speed — which quality dimensions (tests, observability, security hardening) can be deferred to v2?",
		})
	}

	if ids[concernStrWellEngineered] && ids["learning-project"] {
		tensions = append(tensions, ConcernTension{
			Between: []string{concernStrWellEngineered, "learning-project"},
			Issue:   "Engineering standards vs experimentation freedom — how much test/security/observability rigor is appropriate for an experiment?",
		})
	}

	return tensions
}

// =============================================================================
// Review Dimensions
// =============================================================================

// TaggedReviewDimension is a ReviewDimension with its source concern ID.
type TaggedReviewDimension struct {
	state.ReviewDimension
	ConcernID string `json:"concernId"`
}

// GetReviewDimensions collects review dimensions from active concerns, filtered by classification.
func GetReviewDimensions(activeConcerns []state.ConcernDefinition, classification *state.SpecClassification) []TaggedReviewDimension {
	var dimensions []TaggedReviewDimension

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
			dimensions = append(dimensions, TaggedReviewDimension{
				ReviewDimension: dim,
				ConcernID:       concern.ID,
			})
		}
	}

	return dimensions
}

// GetRegistryDimensionIds collects all registry dimension IDs from active concerns.
func GetRegistryDimensionIds(activeConcerns []state.ConcernDefinition) []string {
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
