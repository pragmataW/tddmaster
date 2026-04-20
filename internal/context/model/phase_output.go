package model

import "github.com/pragmataW/tddmaster/internal/state"

// BlockedOutput is the output for the BLOCKED phase.
type BlockedOutput struct {
	Phase       string             `json:"phase"`
	Instruction string             `json:"instruction"`
	Reason      string             `json:"reason"`
	Transition  TransitionResolved `json:"transition"`
}

// SpecSummary describes an existing spec.
type SpecSummary struct {
	Name      string  `json:"name"`
	Phase     string  `json:"phase"`
	Iteration int     `json:"iteration"`
	Detail    *string `json:"detail,omitempty"`
}

// IdleOutput is the output for the IDLE phase.
type IdleOutput struct {
	Phase             string        `json:"phase"`
	Instruction       string        `json:"instruction"`
	Welcome           string        `json:"welcome"`
	ExistingSpecs     []SpecSummary `json:"existingSpecs"`
	AvailableConcerns []ConcernInfo `json:"availableConcerns"`
	ActiveConcerns    []string      `json:"activeConcerns"`
	ActiveRulesCount  int           `json:"activeRulesCount"`
	BehavioralNote    *string       `json:"behavioralNote,omitempty"`
	Hint              *string       `json:"hint,omitempty"`
}

// IdleContext provides extra context for the IDLE phase.
type IdleContext struct {
	ExistingSpecs []SpecSummary
	RulesCount    *int
}

// CompletionSummary is the summary block of CompletedOutput.
type CompletionSummary struct {
	Spec             *string                 `json:"spec"`
	Iterations       int                     `json:"iterations"`
	DecisionsCount   int                     `json:"decisionsCount"`
	CompletionReason *state.CompletionReason `json:"completionReason"`
	CompletionNote   *string                 `json:"completionNote"`
}

// LearningPrompt asks the user to record insights after completion.
type LearningPrompt struct {
	Instruction string   `json:"instruction"`
	Examples    []string `json:"examples"`
}

// CompletedOutput is the output for the COMPLETED phase.
type CompletedOutput struct {
	Phase                 string            `json:"phase"`
	Summary               CompletionSummary `json:"summary"`
	LearningPrompt        *LearningPrompt   `json:"learningPrompt,omitempty"`
	LearningsPending      *bool             `json:"learningsPending,omitempty"`
	StaleDiagrams         []StaleDiagram    `json:"staleDiagrams,omitempty"`
	StaleDiagramsBlocking *bool             `json:"staleDiagramsBlocking,omitempty"`
}
