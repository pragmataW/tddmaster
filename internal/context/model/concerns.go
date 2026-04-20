package model

import "github.com/pragmataW/tddmaster/internal/state"

// ConcernTension represents a tension between two active concerns that must be
// surfaced to the user before execution.
type ConcernTension struct {
	Between []string `json:"between"`
	Issue   string   `json:"issue"`
}

// TaggedReviewDimension is a ReviewDimension with its source concern ID attached.
type TaggedReviewDimension struct {
	state.ReviewDimension
	ConcernID string `json:"concernId"`
}

// ConcernInfo describes an available concern presented at IDLE.
type ConcernInfo struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}
