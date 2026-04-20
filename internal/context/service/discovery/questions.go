package discovery

import (
	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/context/service/concerns"
	"github.com/pragmataW/tddmaster/internal/state"
)

// GetQuestionsWithExtras returns all questions enriched with built-in and
// concern-specific extras.
func GetQuestionsWithExtras(activeConcerns []state.ConcernDefinition) []model.QuestionWithExtras {
	result := make([]model.QuestionWithExtras, len(model.Questions))
	for i, q := range model.Questions {
		var extras []state.ConcernExtra
		for _, bi := range model.BuiltInExtras {
			if bi.QuestionID == q.ID {
				extras = append(extras, state.ConcernExtra{
					QuestionID: bi.QuestionID,
					Text:       bi.Text,
				})
			}
		}
		extras = append(extras, concerns.GetExtras(activeConcerns, q.ID)...)
		result[i] = model.QuestionWithExtras{
			Question: q,
			Extras:   extras,
		}
	}
	return result
}

// isDiscoveryComplete returns true when all questions have been answered.
func isDiscoveryComplete(answers []state.DiscoveryAnswer) bool {
	answered := make(map[string]bool, len(answers))
	for _, a := range answers {
		answered[a.QuestionID] = true
	}
	for _, q := range model.Questions {
		if !answered[q.ID] {
			return false
		}
	}
	return true
}
