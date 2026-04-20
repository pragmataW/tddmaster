package model

// InteractiveOption is a single choice presented to the user. Command is the
// exact shell invocation the caller should run when the user picks this option.
// Command is omitted from the public JSON contract; NextOutput pairs a label-only
// view with a separate commandMap for the consumer.
type InteractiveOption struct {
	Label       string `json:"label"`
	Description string `json:"description"`
	Command     string `json:"-"`
}

// DiscoveryModeOption is a single discovery mode choice presented at the mode
// selection sub-step. The same list is the source of truth for both
// ModeSelectionOutput.Options and the top-level InteractiveOption array emitted
// for PhaseDiscovery.
type DiscoveryModeOption struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// DiscoveryModeOptions returns the canonical list of discovery modes.
func DiscoveryModeOptions() []DiscoveryModeOption {
	return []DiscoveryModeOption{
		{"full", "Full discovery", "Standard 7 questions with all concern extras. Default for new features."},
		{"validate", "Validate my plan", "I already know what I want — challenge my assumptions, find gaps."},
		{"technical-depth", "Technical depth", "Focus on architecture, data flow, performance, integration points."},
		{"ship-fast", "Ship fast", "Minimum viable scope. What can we defer? What's the MVP?"},
		{"explore", "Explore scope", "Think bigger. 10x version? Adjacent opportunities? What are we missing?"},
	}
}

// RoadmapPhase is a single stop on the overall workflow timeline.
type RoadmapPhase struct {
	Key   string
	Label string
}

// RoadmapPhases is the canonical ordered list of phases rendered in the roadmap banner.
var RoadmapPhases = []RoadmapPhase{
	{"IDLE", "IDLE"},
	{"DISCOVERY", "DISCOVERY"},
	{"DISCOVERY_REFINEMENT", "REFINEMENT"},
	{"SPEC_PROPOSAL", "PROPOSAL"},
	{"SPEC_APPROVED", "APPROVED"},
	{"EXECUTING", "EXECUTING"},
	{"COMPLETED", "COMPLETED"},
	{"IDLE_END", "IDLE"},
}
