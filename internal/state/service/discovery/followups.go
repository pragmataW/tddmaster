package discovery

import (
	"fmt"
	"time"

	"github.com/pragmataW/tddmaster/internal/state/model"
)

const maxFollowupsPerQuestion = 3

// AddFollowUp adds a follow-up question to an answered discovery question.
// Silently caps at 3 per parent question.
func AddFollowUp(state model.StateFile, parentQuestionID, question, createdBy string) model.StateFile {
	if state.Discovery.FollowUps == nil {
		state.Discovery.FollowUps = []model.FollowUp{}
	}

	parentCount := 0
	for _, f := range state.Discovery.FollowUps {
		if f.ParentQuestionID == parentQuestionID {
			parentCount++
		}
	}
	if parentCount >= maxFollowupsPerQuestion {
		return state
	}

	// ID suffix: Q3a, Q3b, Q3c
	id := fmt.Sprintf("%s%c", parentQuestionID, rune('a'+parentCount))

	followUp := model.FollowUp{
		ID:               id,
		ParentQuestionID: parentQuestionID,
		Question:         question,
		Answer:           nil,
		Status:           "pending",
		CreatedBy:        createdBy,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
	}

	state.Discovery.FollowUps = append(state.Discovery.FollowUps, followUp)
	return state
}

// AnswerFollowUp answers a follow-up question.
func AnswerFollowUp(state model.StateFile, followUpID, answer string) model.StateFile {
	if state.Discovery.FollowUps == nil {
		return state
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for i, f := range state.Discovery.FollowUps {
		if f.ID == followUpID && f.Status == "pending" {
			state.Discovery.FollowUps[i].Answer = &answer
			state.Discovery.FollowUps[i].Status = "answered"
			state.Discovery.FollowUps[i].AnsweredAt = &now
		}
	}
	return state
}

// SkipFollowUp skips a follow-up question.
func SkipFollowUp(state model.StateFile, followUpID string) model.StateFile {
	if state.Discovery.FollowUps == nil {
		return state
	}
	for i, f := range state.Discovery.FollowUps {
		if f.ID == followUpID && f.Status == "pending" {
			state.Discovery.FollowUps[i].Status = "skipped"
		}
	}
	return state
}

// GetPendingFollowUps returns all pending follow-ups.
func GetPendingFollowUps(state model.StateFile) []model.FollowUp {
	result := make([]model.FollowUp, 0)
	for _, f := range state.Discovery.FollowUps {
		if f.Status == "pending" {
			result = append(result, f)
		}
	}
	return result
}

// GetFollowUpsForQuestion returns all follow-ups for a specific parent question.
func GetFollowUpsForQuestion(state model.StateFile, parentQuestionID string) []model.FollowUp {
	result := make([]model.FollowUp, 0)
	for _, f := range state.Discovery.FollowUps {
		if f.ParentQuestionID == parentQuestionID {
			result = append(result, f)
		}
	}
	return result
}

// AddDelegation delegates a discovery question to another contributor.
func AddDelegation(state model.StateFile, questionID, delegatedTo, delegatedBy string) model.StateFile {
	if state.Discovery.Delegations == nil {
		state.Discovery.Delegations = []model.Delegation{}
	}

	delegation := model.Delegation{
		QuestionID:  questionID,
		DelegatedTo: delegatedTo,
		DelegatedBy: delegatedBy,
		Status:      "pending",
		DelegatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	state.Discovery.Delegations = append(state.Discovery.Delegations, delegation)
	return state
}

// AnswerDelegation answers a delegated question.
func AnswerDelegation(state model.StateFile, questionID, answer, answeredBy string) model.StateFile {
	if state.Discovery.Delegations == nil {
		return state
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for i, d := range state.Discovery.Delegations {
		if d.QuestionID == questionID && d.Status == "pending" {
			state.Discovery.Delegations[i].Status = "answered"
			state.Discovery.Delegations[i].Answer = &answer
			state.Discovery.Delegations[i].AnsweredBy = &answeredBy
			state.Discovery.Delegations[i].AnsweredAt = &now
		}
	}
	return state
}

// GetPendingDelegations returns all pending delegations.
func GetPendingDelegations(state model.StateFile) []model.Delegation {
	result := make([]model.Delegation, 0)
	for _, d := range state.Discovery.Delegations {
		if d.Status == "pending" {
			result = append(result, d)
		}
	}
	return result
}
