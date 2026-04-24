package model

import "github.com/pragmataW/tddmaster/internal/state"

// DiscoveryQuestion is a discovery question as presented in output.
type DiscoveryQuestion struct {
	ID       string                       `json:"id"`
	Text     string                       `json:"text"`
	Concerns []string                     `json:"concerns"`
	Extras   []string                     `json:"extras"`
	Prefills []state.DiscoveryPrefillItem `json:"prefills,omitempty"`
}

// PreDiscoveryResearch is injected when tech version terms are detected in description.
type PreDiscoveryResearch struct {
	Required       bool     `json:"required"`
	Instruction    string   `json:"instruction"`
	ExtractedTerms []string `json:"extractedTerms"`
}

// PreviousProgress summarises prior work for revisit states.
type PreviousProgress struct {
	CompletedTasks []string `json:"completedTasks"`
	TotalTasks     int      `json:"totalTasks"`
}

// ModeSelectionOutput provides mode selection options for discovery.
type ModeSelectionOutput struct {
	Required    bool                  `json:"required"`
	Instruction string                `json:"instruction"`
	Options     []DiscoveryModeOption `json:"options"`
}

// PremiseChallengeOutput asks the agent to challenge spec premises.
type PremiseChallengeOutput struct {
	Required    bool     `json:"required"`
	Instruction string   `json:"instruction"`
	Prompts     []string `json:"prompts"`
}

// RichDescriptionOutput signals a rich user description is available for pre-fill.
type RichDescriptionOutput struct {
	Provided    bool   `json:"provided"`
	Length      int    `json:"length"`
	Content     string `json:"content"`
	Instruction string `json:"instruction"`
}

// AlternativesFormat carries the expected field set for alternative proposals.
type AlternativesFormat struct {
	Fields []string `json:"fields"`
}

// AlternativesOutput asks the agent to propose implementation alternatives.
type AlternativesOutput struct {
	Required    bool               `json:"required"`
	Instruction string             `json:"instruction"`
	Format      AlternativesFormat `json:"format"`
}

// DiscoveryOutput is the output for the DISCOVERY phase.
type DiscoveryOutput struct {
	Phase           string               `json:"phase"`
	Instruction     string               `json:"instruction"`
	Questions       []DiscoveryQuestion  `json:"questions"`
	AnsweredCount   int                  `json:"answeredCount"`
	CurrentQuestion *int                 `json:"currentQuestion,omitempty"`
	TotalQuestions  *int                 `json:"totalQuestions,omitempty"`
	Context         ContextBlock         `json:"context"`
	Transition      TransitionOnComplete `json:"transition"`

	Revisited            *bool                   `json:"revisited,omitempty"`
	RevisitReason        *string                 `json:"revisitReason,omitempty"`
	PreviousProgress     *PreviousProgress       `json:"previousProgress,omitempty"`
	PreDiscoveryResearch *PreDiscoveryResearch   `json:"preDiscoveryResearch,omitempty"`
	CurrentUser          *CurrentUser            `json:"currentUser,omitempty"`
	Notes                []SpecNote              `json:"notes,omitempty"`
	ModeSelection        *ModeSelectionOutput    `json:"modeSelection,omitempty"`
	PremiseChallenge     *PremiseChallengeOutput `json:"premiseChallenge,omitempty"`
	RichDescription      *RichDescriptionOutput  `json:"richDescription,omitempty"`
	AgreedPremises       []string                `json:"agreedPremises,omitempty"`
	RevisedPremises      []RevisedPremise        `json:"revisedPremises,omitempty"`
	FollowUpHints        []string                `json:"followUpHints,omitempty"`
	PendingFollowUps     []state.FollowUp        `json:"pendingFollowUps,omitempty"`
	PreviousLearnings    []string                `json:"previousLearnings,omitempty"`
}
