package model

// CurrentUser identifies the active user contributing to discovery.
type CurrentUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// SpecNote is a free-form note attached to the spec by any user.
type SpecNote struct {
	Text string `json:"text"`
	User string `json:"user"`
}

// RevisedPremise captures a premise the user disagreed with and rewrote.
type RevisedPremise struct {
	Original string `json:"original"`
	Revision string `json:"revision"`
}

// StaleDiagram flags a Mermaid/PlantUML block whose referenced code drifted.
type StaleDiagram struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Reason string `json:"reason"`
}
