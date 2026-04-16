
package context_test

import (
	"testing"

	ctx "github.com/pragmataW/tddmaster/internal/context"
	"github.com/pragmataW/tddmaster/internal/defaults"
	"github.com/pragmataW/tddmaster/internal/state"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Helpers
// =============================================================================

func allAnswersFixture() []state.DiscoveryAnswer {
	answers := make([]state.DiscoveryAnswer, len(ctx.Questions))
	for i, q := range ctx.Questions {
		answers[i] = state.DiscoveryAnswer{
			QuestionID: q.ID,
			Answer:     "answer-" + q.ID,
		}
	}
	return answers
}

// =============================================================================
// GetQuestionsWithExtras
// =============================================================================

func TestGetQuestionsWithExtras_Returns7QuestionsWithNoConcerns(t *testing.T) {
	result := ctx.GetQuestionsWithExtras([]state.ConcernDefinition{})

	assert.Len(t, result, 7)
	assert.Equal(t, "status_quo", result[0].ID)
	assert.Equal(t, "scope_boundary", result[5].ID)
	assert.Equal(t, "edge_cases", result[6].ID)
}

func TestGetQuestionsWithExtras_InjectsExtrasFromConcerns(t *testing.T) {
	allConcerns := defaults.DefaultConcerns()
	var openSource state.ConcernDefinition
	for _, c := range allConcerns {
		if c.ID == "open-source" {
			openSource = c
			break
		}
	}

	result := ctx.GetQuestionsWithExtras([]state.ConcernDefinition{openSource})

	var statusQuo *ctx.QuestionWithExtras
	for i := range result {
		if result[i].ID == "status_quo" {
			statusQuo = &result[i]
			break
		}
	}

	assert.NotNil(t, statusQuo)
	assert.Greater(t, len(statusQuo.Extras), 0)
	assert.Equal(t, "Is this workaround common in the community?", statusQuo.Extras[0].Text)
}

func TestGetQuestionsWithExtras_MultipleConCernsStackExtras(t *testing.T) {
	allConcerns := defaults.DefaultConcerns()

	var openSource, beautiful state.ConcernDefinition
	for _, c := range allConcerns {
		switch c.ID {
		case "open-source":
			openSource = c
		case "beautiful-product":
			beautiful = c
		}
	}

	result := ctx.GetQuestionsWithExtras([]state.ConcernDefinition{openSource, beautiful})

	var statusQuo *ctx.QuestionWithExtras
	for i := range result {
		if result[i].ID == "status_quo" {
			statusQuo = &result[i]
			break
		}
	}

	assert.NotNil(t, statusQuo)
	// open-source adds 1 extra for status_quo, beautiful-product adds 2
	assert.Equal(t, 3, len(statusQuo.Extras))
}

// =============================================================================
// GetNextUnanswered
// =============================================================================

func TestGetNextUnanswered_ReturnsFirstWhenNoAnswers(t *testing.T) {
	qs := ctx.GetQuestionsWithExtras([]state.ConcernDefinition{})
	next := ctx.GetNextUnanswered(qs, []state.DiscoveryAnswer{})

	assert.NotNil(t, next)
	assert.Equal(t, "status_quo", next.ID)
}

func TestGetNextUnanswered_SkipsAnsweredQuestions(t *testing.T) {
	qs := ctx.GetQuestionsWithExtras([]state.ConcernDefinition{})
	answers := []state.DiscoveryAnswer{
		{QuestionID: "status_quo", Answer: "done"},
	}
	next := ctx.GetNextUnanswered(qs, answers)

	assert.NotNil(t, next)
	assert.Equal(t, "ambition", next.ID)
}

func TestGetNextUnanswered_ReturnsNilWhenAllAnswered(t *testing.T) {
	qs := ctx.GetQuestionsWithExtras([]state.ConcernDefinition{})
	next := ctx.GetNextUnanswered(qs, allAnswersFixture())

	assert.Nil(t, next)
}

// =============================================================================
// IsDiscoveryComplete
// =============================================================================

func TestIsDiscoveryComplete_FalseWithPartialAnswers(t *testing.T) {
	partial := []state.DiscoveryAnswer{
		{QuestionID: "status_quo", Answer: "x"},
		{QuestionID: "ambition", Answer: "x"},
	}

	assert.False(t, ctx.IsDiscoveryComplete(partial))
}

func TestIsDiscoveryComplete_TrueWhenAllAnswered(t *testing.T) {
	assert.True(t, ctx.IsDiscoveryComplete(allAnswersFixture()))
}

func TestIsDiscoveryComplete_TrueEvenWithExtraUnknownIds(t *testing.T) {
	answers := append(allAnswersFixture(), state.DiscoveryAnswer{
		QuestionID: "unknown_extra",
		Answer:     "x",
	})

	assert.True(t, ctx.IsDiscoveryComplete(answers))
}
