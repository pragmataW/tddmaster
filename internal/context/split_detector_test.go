
package context_test

import (
	"regexp"
	"testing"

	ctx "github.com/pragmataW/tddmaster/internal/context"
	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Helpers
// =============================================================================

func makeAnswers(overrides map[string]string) []state.DiscoveryAnswer {
	defaults := map[string]string{
		"status_quo":    "Users manually upload photos via email.",
		"ambition":      "1-star: basic upload. 10-star: drag-and-drop with preview.",
		"reversibility": "No irreversible decisions.",
		"user_impact":   "No breaking changes.",
		"verification":  "Unit tests and manual QA.",
		"scope_boundary": "No video support.",
	}

	merged := make(map[string]string)
	for k, v := range defaults {
		merged[k] = v
	}
	for k, v := range overrides {
		merged[k] = v
	}

	var answers []state.DiscoveryAnswer
	for questionID, answer := range merged {
		answers = append(answers, state.DiscoveryAnswer{
			QuestionID: questionID,
			Answer:     answer,
		})
	}
	return answers
}

// =============================================================================
// AnalyzeForSplit
// =============================================================================

func TestAnalyzeForSplit_FalseForSingleArea(t *testing.T) {
	answers := makeAnswers(map[string]string{
		"status_quo": "Users manually upload photos via email.",
		"ambition":   "Add a drag-and-drop photo upload widget.",
	})

	result := ctx.AnalyzeForSplit(answers, "")

	assert.False(t, result.Detected)
	assert.Len(t, result.Proposals, 0)
}

func TestAnalyzeForSplit_DetectsNumberedListInStatusQuo(t *testing.T) {
	answers := makeAnswers(map[string]string{
		"status_quo": "(1) Log messages are too verbose and flood stdout (2) Bot chat responses use wrong gender pronouns",
	})

	result := ctx.AnalyzeForSplit(answers, "")

	assert.True(t, result.Detected)
	assert.Len(t, result.Proposals, 2)

	found := false
	for _, p := range result.Proposals {
		for _, id := range p.RelevantAnswers {
			if id == "status_quo" {
				found = true
			}
		}
	}
	assert.True(t, found)
}

func TestAnalyzeForSplit_DetectsDotNumberedListInAmbition(t *testing.T) {
	answers := makeAnswers(map[string]string{
		"ambition": "1. Fix the log level configuration to reduce noise. 2. Restore bot gender detection from user profiles.",
	})

	result := ctx.AnalyzeForSplit(answers, "")

	assert.True(t, result.Detected)
	assert.Len(t, result.Proposals, 2)
}

func TestAnalyzeForSplit_DetectsAdditionallySeparation(t *testing.T) {
	answers := makeAnswers(map[string]string{
		"status_quo": "Cursor shader is stale and renders incorrectly. Additionally, bot gender detection is broken in chat.",
	})

	result := ctx.AnalyzeForSplit(answers, "")

	assert.True(t, result.Detected)
	assert.Len(t, result.Proposals, 2)
}

func TestAnalyzeForSplit_DetectsAndPatternWithUnrelatedNouns(t *testing.T) {
	answers := makeAnswers(map[string]string{
		"ambition": "fix log levels AND restore bot gender detection",
	})

	result := ctx.AnalyzeForSplit(answers, "")

	assert.True(t, result.Detected)
	assert.Len(t, result.Proposals, 2)
}

func TestAnalyzeForSplit_FalseForTightlyCoupledAreas(t *testing.T) {
	answers := makeAnswers(map[string]string{
		"status_quo": "(1) Add UserType enum to schema (2) Use the new UserType in the handler",
	})

	result := ctx.AnalyzeForSplit(answers, "")

	// Should detect coupling via shared PascalCase "UserType" and "use" pattern
	assert.False(t, result.Detected)
}

func TestAnalyzeForSplit_FalseWhenTotalTasksSmall(t *testing.T) {
	answers := makeAnswers(map[string]string{
		"status_quo": "(1) typo (2) typo2",
	})

	result := ctx.AnalyzeForSplit(answers, "")

	// If detected, total must be > 3
	if result.Detected {
		total := 0
		for _, p := range result.Proposals {
			total += p.EstimatedTasks
		}
		assert.Greater(t, total, 3)
	}
}

func TestAnalyzeForSplit_FalseInShipFastMode(t *testing.T) {
	answers := makeAnswers(map[string]string{
		"status_quo": "(1) Log messages are too verbose (2) Bot chat uses wrong gender",
	})

	result := ctx.AnalyzeForSplit(answers, "ship-fast")

	assert.False(t, result.Detected)
	assert.Len(t, result.Proposals, 0)
}

func TestAnalyzeForSplit_GeneratesSlugifiedNames(t *testing.T) {
	answers := makeAnswers(map[string]string{
		"status_quo": "(1) Log messages are too verbose and flood stdout (2) Bot chat responses use wrong gender pronouns",
	})

	result := ctx.AnalyzeForSplit(answers, "")

	assert.True(t, result.Detected)
	slugRe := regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)
	for _, p := range result.Proposals {
		assert.True(t, slugRe.MatchString(p.Name), "slug should match pattern: %s", p.Name)
		assert.LessOrEqual(t, len(p.Name), 50)
	}
}

func TestAnalyzeForSplit_AssignsRelevantAnswerIds(t *testing.T) {
	answers := makeAnswers(map[string]string{
		"status_quo": "(1) Log messages are too verbose (2) Bot chat uses wrong gender",
	})

	result := ctx.AnalyzeForSplit(answers, "")

	assert.True(t, result.Detected)
	for _, p := range result.Proposals {
		assert.Greater(t, len(p.RelevantAnswers), 0)
		for _, id := range p.RelevantAnswers {
			assert.Greater(t, len(id), 0)
		}
	}
}

func TestAnalyzeForSplit_DetectsOrdinalPattern(t *testing.T) {
	answers := makeAnswers(map[string]string{
		"ambition": "First: reduce log noise by fixing the log level config. Second: restore the bot gender detection pipeline.",
	})

	result := ctx.AnalyzeForSplit(answers, "")

	assert.True(t, result.Detected)
	assert.Len(t, result.Proposals, 2)
}

func TestAnalyzeForSplit_ReturnsReasonString(t *testing.T) {
	answers := makeAnswers(map[string]string{
		"status_quo": "(1) Log messages too verbose (2) Bot chat broken",
	})

	result := ctx.AnalyzeForSplit(answers, "")

	assert.True(t, result.Detected)
	assert.Greater(t, len(result.Reason), 0)
	assert.Contains(t, result.Reason, "2")
}

func TestAnalyzeForSplit_Detects3IndependentAreas(t *testing.T) {
	answers := makeAnswers(map[string]string{
		"status_quo": "(1) Log messages are too verbose and flood stdout (2) Bot chat responses use wrong gender pronouns (3) Missing asset loading fails silently",
	})

	result := ctx.AnalyzeForSplit(answers, "")

	assert.True(t, result.Detected)
	assert.Len(t, result.Proposals, 3)
	assert.Contains(t, result.Reason, "3")
}
