// Package model contains pure data shapes for tddmaster state: phase enums,
// discovery/execution structures, manifest/concern schemas, and session types.
// It has no I/O and no mutation logic — those live in internal/state/service.
package model

// Phase enumerates the high-level workflow states a spec can be in.
type Phase string

const (
	PhaseUninitialized       Phase = "UNINITIALIZED"
	PhaseIdle                Phase = "IDLE"
	PhaseDiscovery           Phase = "DISCOVERY"
	PhaseDiscoveryRefinement Phase = "DISCOVERY_REFINEMENT"
	PhaseSpecProposal        Phase = "SPEC_PROPOSAL"
	PhaseSpecApproved        Phase = "SPEC_APPROVED"
	PhaseExecuting           Phase = "EXECUTING"
	PhaseBlocked             Phase = "BLOCKED"
	PhaseCompleted           Phase = "COMPLETED"
)

// CompletionReason enumerates why a spec reached PhaseCompleted.
type CompletionReason string

const (
	CompletionReasonDone      CompletionReason = "done"
	CompletionReasonCancelled CompletionReason = "cancelled"
	CompletionReasonWontfix   CompletionReason = "wontfix"
)

// DiscoveryMode selects how aggressively discovery asks questions.
type DiscoveryMode string

const (
	DiscoveryModeFull           DiscoveryMode = "full"
	DiscoveryModeValidate       DiscoveryMode = "validate"
	DiscoveryModeTechnicalDepth DiscoveryMode = "technical-depth"
	DiscoveryModeShipFast       DiscoveryMode = "ship-fast"
	DiscoveryModeExplore        DiscoveryMode = "explore"
)

// ValidTransitions defines which phase transitions are allowed.
var ValidTransitions = map[Phase][]Phase{
	PhaseUninitialized:       {PhaseIdle},
	PhaseIdle:                {PhaseDiscovery, PhaseCompleted},
	PhaseDiscovery:           {PhaseDiscoveryRefinement, PhaseCompleted},
	PhaseDiscoveryRefinement: {PhaseDiscoveryRefinement, PhaseSpecProposal, PhaseCompleted},
	PhaseSpecProposal:        {PhaseSpecProposal, PhaseSpecApproved, PhaseCompleted},
	PhaseSpecApproved:        {PhaseExecuting, PhaseCompleted},
	PhaseExecuting:           {PhaseCompleted, PhaseBlocked},
	PhaseBlocked:             {PhaseExecuting, PhaseCompleted},
	PhaseCompleted:           {PhaseIdle, PhaseDiscovery, PhaseExecuting, PhaseBlocked},
}
