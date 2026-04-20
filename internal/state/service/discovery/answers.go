package discovery

import "github.com/pragmataW/tddmaster/internal/state/model"

// NormalizeAnswer normalizes a DiscoveryAnswer to AttributedDiscoveryAnswer.
// Old-format answers (just questionId+answer) become attributed with an
// "Unknown User" placeholder.
func NormalizeAnswer(answer model.DiscoveryAnswer) model.AttributedDiscoveryAnswer {
	return model.AttributedDiscoveryAnswer{
		QuestionID: answer.QuestionID,
		Answer:     answer.Answer,
		User:       "Unknown User",
		Email:      "",
		Timestamp:  "",
		Type:       "original",
	}
}

// NormalizeAttributedAnswer returns the attributed answer as-is.
func NormalizeAttributedAnswer(answer model.AttributedDiscoveryAnswer) model.AttributedDiscoveryAnswer {
	return answer
}

// GetAnswersForQuestion returns all answers for a specific question, normalized.
func GetAnswersForQuestion(answers []model.DiscoveryAnswer, questionID string) []model.AttributedDiscoveryAnswer {
	result := make([]model.AttributedDiscoveryAnswer, 0)
	for _, a := range answers {
		if a.QuestionID == questionID {
			result = append(result, NormalizeAnswer(a))
		}
	}
	return result
}

// GetCombinedAnswer returns the combined answer text for a question (all contributors).
func GetCombinedAnswer(answers []model.DiscoveryAnswer, questionID string) string {
	qAnswers := GetAnswersForQuestion(answers, questionID)
	if len(qAnswers) == 0 {
		return ""
	}
	if len(qAnswers) == 1 {
		return qAnswers[0].Answer
	}
	result := ""
	for i, a := range qAnswers {
		if i > 0 {
			result += "\n\n"
		}
		result += a.Answer + " -- *" + a.User + "*"
	}
	return result
}

// GetPrefillsForQuestion returns a copy of prefill items for the given question.
func GetPrefillsForQuestion(prefills []model.DiscoveryPrefillQuestion, questionID string) []model.DiscoveryPrefillItem {
	for _, p := range prefills {
		if p.QuestionID != questionID {
			continue
		}
		result := make([]model.DiscoveryPrefillItem, len(p.Items))
		copy(result, p.Items)
		return result
	}
	return nil
}
