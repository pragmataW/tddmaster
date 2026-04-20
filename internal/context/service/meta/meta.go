// Package meta assembles the non-phase-specific chrome attached to every
// NextOutput: meta block, roadmap, gate, protocol guide and interactive
// options. Functions here accept a renderer interface so they don't import the
// orchestrator package and trigger an import cycle.
package meta

import (
	"fmt"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/state"
)

// Renderer is the minimal command-builder interface required by this package.
// service.Renderer satisfies it via duck typing.
type Renderer interface {
	C(sub string) string
	CS(sub string, specName *string) string
}

// BuildEnforcement returns the EnforcementInfo based on the caller's capabilities.
func BuildEnforcement(hints model.InteractionHints) *model.EnforcementInfo {
	if hints.HasSubAgentDelegation {
		return &model.EnforcementInfo{
			Level: "enforced",
			Capabilities: []string{
				"PreToolUse file edit gate",
				"Git write guard",
				"Stop iteration tracking",
				"PostToolUse file logging",
				"Sub-agent delegation",
			},
		}
	}
	return &model.EnforcementInfo{
		Level:        "behavioral",
		Capabilities: []string{"Behavioral rules only"},
		Gaps: []string{
			"File edits not blocked in non-execution phases",
			"Git write commands not blocked",
			"No iteration tracking",
			"No file change logging",
			"No sub-agent delegation available",
		},
	}
}

// Build assembles the MetaBlock for the current state.
func Build(r Renderer, st state.StateFile, activeConcerns []state.ConcernDefinition, hints model.InteractionHints) model.MetaBlock {
	var resumeHint string

	switch st.Phase {
	case state.PhaseIdle:
		resumeHint = fmt.Sprintf("No active spec. Start one with: `%s`", r.C("spec new --name=<slug> \"description\""))
	case state.PhaseDiscovery:
		name := ""
		if st.Spec != nil {
			name = *st.Spec
		}
		resumeHint = fmt.Sprintf("Discovery in progress for \"%s\". %d questions answered so far.", name, len(st.Discovery.Answers))
	case state.PhaseDiscoveryRefinement:
		resumeHint = fmt.Sprintf("Discovery answers ready for review. %d answers collected. Waiting for user confirmation.", len(st.Discovery.Answers))
	case state.PhaseSpecProposal:
		path := ""
		if st.SpecState.Path != nil {
			path = *st.SpecState.Path
		}
		resumeHint = fmt.Sprintf("Spec draft ready for review at %s. Waiting for approval.", path)
	case state.PhaseSpecApproved:
		name := ""
		if st.Spec != nil {
			name = *st.Spec
		}
		resumeHint = fmt.Sprintf("Spec \"%s\" is approved. Waiting to start execution.", name)
	case state.PhaseExecuting:
		name := ""
		if st.Spec != nil {
			name = *st.Spec
		}
		if st.Execution.LastProgress != nil {
			resumeHint = fmt.Sprintf("Executing \"%s\", iteration %d. Last progress: %s. Continue with the current task.", name, st.Execution.Iteration, *st.Execution.LastProgress)
		} else {
			resumeHint = fmt.Sprintf("Executing \"%s\", iteration %d. Start the first task.", name, st.Execution.Iteration)
		}
	case state.PhaseBlocked:
		progress := "Unknown"
		if st.Execution.LastProgress != nil {
			progress = *st.Execution.LastProgress
		}
		resumeHint = fmt.Sprintf("Execution blocked: %s. Ask the user to resolve.", progress)
	case state.PhaseCompleted:
		name := ""
		if st.Spec != nil {
			name = *st.Spec
		}
		resumeHint = fmt.Sprintf("Spec \"%s\" completed in %d iterations.", name, st.Execution.Iteration)
	default:
		resumeHint = fmt.Sprintf("Run `%s` to get started.", r.CS("next", st.Spec))
	}

	concernIDs := make([]string, len(activeConcerns))
	for i, c := range activeConcerns {
		concernIDs[i] = c.ID
	}

	return model.MetaBlock{
		Protocol:       fmt.Sprintf("Run `%s` to submit results and advance", r.CS("next --answer=\"...\"", st.Spec)),
		Spec:           st.Spec,
		Branch:         st.Branch,
		Iteration:      st.Execution.Iteration,
		LastProgress:   st.Execution.LastProgress,
		ActiveConcerns: concernIDs,
		ResumeHint:     resumeHint,
		Enforcement:    BuildEnforcement(hints),
	}
}
