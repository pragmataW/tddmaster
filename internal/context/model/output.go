package model

// ClearContextAction signals that the caller should clear its scratchpad before rendering.
type ClearContextAction struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

// GateInfo describes the gate at the current phase transition.
type GateInfo struct {
	Message string `json:"message"`
	Action  string `json:"action"`
	Phase   string `json:"phase"`
}

// ProtocolGuide is shown on first call or after a stale session.
type ProtocolGuide struct {
	What         string `json:"what"`
	How          string `json:"how"`
	CurrentPhase string `json:"currentPhase"`
}

// EnforcementInfo describes the enforcement level for the active tool.
type EnforcementInfo struct {
	Level        string   `json:"level"` // "enforced" | "behavioral"
	Capabilities []string `json:"capabilities"`
	Gaps         []string `json:"gaps,omitempty"`
}

// MetaBlock is the self-documenting resume context included in every output.
type MetaBlock struct {
	Protocol       string           `json:"protocol"`
	Spec           *string          `json:"spec"`
	Branch         *string          `json:"branch"`
	Iteration      int              `json:"iteration"`
	LastProgress   *string          `json:"lastProgress"`
	ActiveConcerns []string         `json:"activeConcerns"`
	ResumeHint     string           `json:"resumeHint"`
	Enforcement    *EnforcementInfo `json:"enforcement,omitempty"`
}

// BehavioralBlock contains phase-aware guardrails for agent behaviour.
type BehavioralBlock struct {
	ModeOverride *string  `json:"modeOverride,omitempty"`
	Rules        []string `json:"rules"`
	CoreReminder []string `json:"coreReminder,omitempty"`
	Tone         string   `json:"tone"`
	Urgency      *string  `json:"urgency,omitempty"`
	OutOfScope   []string `json:"outOfScope,omitempty"`
	Tier2Summary *string  `json:"tier2Summary,omitempty"`
}

// ContextBlock is the rules+reminders context injected into certain phase outputs.
type ContextBlock struct {
	Rules            []string `json:"rules"`
	ConcernReminders []string `json:"concernReminders"`
}

// NextOutput is the top-level JSON output for `tddmaster next`.
type NextOutput struct {
	Phase string `json:"phase"`

	Meta                MetaBlock           `json:"meta"`
	Behavioral          BehavioralBlock     `json:"behavioral"`
	Roadmap             string              `json:"roadmap"`
	Gate                *GateInfo           `json:"gate,omitempty"`
	ProtocolGuide       *ProtocolGuide      `json:"protocolGuide,omitempty"`
	ClearContext        *ClearContextAction `json:"clearContext,omitempty"`
	InteractiveOptions  []InteractiveOption `json:"interactiveOptions,omitempty"`
	CommandMap          map[string]string   `json:"commandMap,omitempty"`
	ToolHint            *string             `json:"toolHint,omitempty"`
	ToolHintInstruction *string             `json:"toolHintInstruction,omitempty"`

	DiscoveryData       *DiscoveryOutput       `json:"discoveryData,omitempty"`
	DiscoveryReviewData *DiscoveryReviewOutput `json:"discoveryReviewData,omitempty"`
	SpecDraftData       *SpecDraftOutput       `json:"specDraftData,omitempty"`
	SpecApprovedData    *SpecApprovedOutput    `json:"specApprovedData,omitempty"`
	ExecutionData       *ExecutionOutput       `json:"executionData,omitempty"`
	BlockedData         *BlockedOutput         `json:"blockedData,omitempty"`
	CompletedData       *CompletedOutput       `json:"completedData,omitempty"`
	IdleData            *IdleOutput            `json:"idleData,omitempty"`
}

// TransitionOnComplete is the per-phase transition hint rendered in DISCOVERY output.
type TransitionOnComplete struct {
	OnComplete string `json:"onComplete"`
}

// TransitionApproveRevise is the per-phase transition hint rendered in DISCOVERY_REFINEMENT.
type TransitionApproveRevise struct {
	OnApprove string `json:"onApprove"`
	OnRevise  string `json:"onRevise"`
}

// TransitionApprove is rendered when only one forward action (approve) is available.
type TransitionApprove struct {
	OnApprove string `json:"onApprove"`
}

// TransitionStart is rendered at SPEC_APPROVED — execution hasn't started yet.
type TransitionStart struct {
	OnStart string `json:"onStart"`
}

// TransitionExecution carries the execution loop's forward/blocked/iteration hints.
type TransitionExecution struct {
	OnComplete string `json:"onComplete"`
	OnBlocked  string `json:"onBlocked"`
	Iteration  int    `json:"iteration"`
}

// TransitionResolved is rendered at BLOCKED — the user must unblock.
type TransitionResolved struct {
	OnResolved string `json:"onResolved"`
}
