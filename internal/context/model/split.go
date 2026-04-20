package model

// SplitProposalItem is a single proposed sub-spec in a split proposal.
type SplitProposalItem struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	EstimatedTasks  int      `json:"estimatedTasks"`
	RelevantAnswers []string `json:"relevantAnswers"`
}

// SplitProposal is the result of analysing discovery answers for potential spec splits.
type SplitProposal struct {
	Detected  bool                `json:"detected"`
	Reason    string              `json:"reason"`
	Proposals []SplitProposalItem `json:"proposals"`
}
