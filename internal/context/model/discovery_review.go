package model

// DiscoveryReviewAnswer is a single answer in the discovery review.
type DiscoveryReviewAnswer struct {
	QuestionID string `json:"questionId"`
	Question   string `json:"question"`
	Answer     string `json:"answer"`
}

// ReviewChecklistDimension is a single dimension in the review checklist.
type ReviewChecklistDimension struct {
	ID               string `json:"id"`
	Label            string `json:"label"`
	Prompt           string `json:"prompt"`
	EvidenceRequired bool   `json:"evidenceRequired"`
	IsRegistry       bool   `json:"isRegistry"`
	ConcernID        string `json:"concernId"`
}

// ReviewChecklist is the full review checklist for discovery refinement.
type ReviewChecklist struct {
	Dimensions          []ReviewChecklistDimension `json:"dimensions"`
	Instruction         string                     `json:"instruction"`
	RegistryInstruction *string                    `json:"registryInstruction,omitempty"`
}

// DiscoveryReviewOutput is the output for the DISCOVERY_REFINEMENT phase.
type DiscoveryReviewOutput struct {
	Phase           string                  `json:"phase"`
	Instruction     string                  `json:"instruction"`
	Answers         []DiscoveryReviewAnswer `json:"answers"`
	ReviewSummary   string                  `json:"reviewSummary,omitempty"`
	Transition      TransitionApproveRevise `json:"transition"`
	SplitProposal   *SplitProposal          `json:"splitProposal,omitempty"`
	SubPhase        *string                 `json:"subPhase,omitempty"`
	Alternatives    *AlternativesOutput     `json:"alternatives,omitempty"`
	ReviewChecklist *ReviewChecklist        `json:"reviewChecklist,omitempty"`
}
