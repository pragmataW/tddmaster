package meta

import (
	"fmt"
	"strings"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/state"
)

// BuildInteractiveOptions returns the per-phase interactive options presented
// to the user. Returns nil when the current phase has no choices.
func BuildInteractiveOptions(
	r Renderer,
	st state.StateFile,
	activeConcerns []state.ConcernDefinition,
	idleContext *model.IdleContext,
	config *state.NosManifest,
) []model.InteractiveOption {
	switch st.Phase {
	case state.PhaseIdle:
		return buildIdleOptions(r, activeConcerns, idleContext)
	case state.PhaseDiscoveryRefinement:
		return buildDiscoveryRefinementOptions(r, st)
	case state.PhaseSpecProposal:
		return buildSpecProposalOptions(r, st)
	case state.PhaseSpecApproved:
		return buildSpecApprovedOptions(r, st, config)
	case state.PhaseDiscovery:
		return buildDiscoveryOptions(r, st)
	case state.PhaseExecuting:
		return nil
	case state.PhaseBlocked:
		return []model.InteractiveOption{
			{
				Label:       "Resolve block",
				Description: "Provide a resolution to unblock execution",
				Command:     r.CS("next --answer=\"resolution\"", st.Spec),
			},
			{
				Label:       "Reset spec",
				Description: "Abandon this spec and start over",
				Command:     r.CS("reset", st.Spec),
			},
		}
	case state.PhaseCompleted:
		return []model.InteractiveOption{
			{
				Label:       "New spec",
				Description: "Start a new feature spec",
				Command:     r.C("spec new --name=<slug> \"description\""),
			},
			{
				Label:       "Reopen spec",
				Description: "Reopen this spec for revision",
				Command:     r.CS("reopen", st.Spec),
			},
			{
				Label:       "Check status",
				Description: "Review completed spec summary",
				Command:     r.C("status"),
			},
		}
	}
	return nil
}

func buildIdleOptions(r Renderer, activeConcerns []state.ConcernDefinition, idleContext *model.IdleContext) []model.InteractiveOption {
	var specs []model.SpecSummary
	if idleContext != nil {
		specs = idleContext.ExistingSpecs
	}

	var continuable []model.SpecSummary
	for _, s := range specs {
		if s.Phase != "COMPLETED" {
			continuable = append(continuable, s)
		}
	}

	var opts []model.InteractiveOption
	if len(activeConcerns) == 0 {
		opts = append(opts, model.InteractiveOption{
			Label:       "Add concerns (Recommended)",
			Description: "Shape how discovery and specs work by adding project concerns",
			Command:     r.C("concern add <id> [<id2> ...]"),
		})
	}

	opts = append(opts, model.InteractiveOption{
		Label:       "Start a new spec",
		Description: "Tell me what you want to build — a one-liner, detailed requirements, meeting notes, anything",
		Command:     r.C("spec new \"description\""),
	})

	for i, sp := range continuable {
		if i >= model.ContinuableSpecsCap {
			break
		}
		detail := fmt.Sprintf("Iteration %d", sp.Iteration)
		if sp.Detail != nil {
			detail = *sp.Detail
		}
		specName := sp.Name
		opts = append(opts, model.InteractiveOption{
			Label:       fmt.Sprintf("Continue: %s (%s)", sp.Name, sp.Phase),
			Description: detail,
			Command:     r.CS("next", &specName),
		})
	}

	if len(activeConcerns) > 0 {
		ids := make([]string, len(activeConcerns))
		for i, cc := range activeConcerns {
			ids[i] = cc.ID
		}
		opts = append(opts, model.InteractiveOption{
			Label:       "Edit concerns",
			Description: "Currently: " + strings.Join(ids, ", "),
			Command:     r.C("concern list"),
		})
	}

	if len(opts) > model.IdleOptionsCap {
		opts = opts[:model.IdleOptionsCap]
	}
	return opts
}

func buildDiscoveryRefinementOptions(r Renderer, st state.StateFile) []model.InteractiveOption {
	if st.Discovery.Approved {
		return []model.InteractiveOption{
			{
				Label:       "Keep as one spec",
				Description: "All work in a single spec",
				Command:     r.CS("next --answer=\"keep\"", st.Spec),
			},
			{
				Label:       "Split into separate specs",
				Description: "Create one spec per independent area",
				Command:     r.CS("next --answer=\"split\"", st.Spec),
			},
		}
	}
	return []model.InteractiveOption{
		{
			Label:       "Approve all answers",
			Description: "Answers look correct — generate the spec",
			Command:     r.CS("next --answer=\"approve\"", st.Spec),
		},
		{
			Label:       "Revise answers",
			Description: "Correct one or more discovery answers",
			Command:     r.CS("next --answer='{\"revise\":{...}}'", st.Spec),
		},
	}
}

func buildSpecProposalOptions(r Renderer, st state.StateFile) []model.InteractiveOption {
	return []model.InteractiveOption{
		{
			Label:       "Approve spec",
			Description: "Review looks good — approve and move to execution",
			Command:     r.CS("approve", st.Spec),
		},
		{
			Label:       "Refine spec",
			Description: "Submit refinements to improve tasks or sections",
			Command:     r.CS("next --answer='{\"refinement\":\"...\"}'", st.Spec),
		},
		{
			Label:       "Save for later",
			Description: "Keep the draft as-is. Others can review, add ACs, notes, or tasks. Come back anytime to approve.",
			Command:     r.CS("next --answer=\"save\"", st.Spec),
		},
		{
			Label:       "Start over",
			Description: "Reset the spec and start fresh",
			Command:     r.CS("reset", st.Spec),
		},
	}
}

func buildSpecApprovedOptions(r Renderer, st state.StateFile, config *state.NosManifest) []model.InteractiveOption {
	needsTDDSelection := config != nil && config.IsTDDEnabled() &&
		(st.TaskTDDSelected == nil || !*st.TaskTDDSelected)
	if needsTDDSelection {
		return []model.InteractiveOption{
			{
				Label:       "TDD for all tasks",
				Description: "Every task follows red → green → refactor (current behavior)",
				Command:     r.CS("next --answer=\"tdd-all\"", st.Spec),
			},
			{
				Label:       "No TDD",
				Description: "Skip red/green/refactor for every task — run executor → verifier only",
				Command:     r.CS("next --answer=\"tdd-none\"", st.Spec),
			},
			{
				Label:       "Pick per task",
				Description: "Use AskUserQuestion with multiSelect over specApprovedData.taskTDDSelection.tasks, then submit {\"tddTasks\":[...IDs...]}",
				Command:     r.CS("next --answer='{\"tddTasks\":[\"task-1\",\"task-3\"]}'", st.Spec),
			},
			{
				Label:       "Save for later",
				Description: "Spec is approved but don't start execution yet. Others can still add ACs or notes.",
				Command:     r.CS("next --answer=\"save\"", st.Spec),
			},
		}
	}
	return []model.InteractiveOption{
		{
			Label:       "Start execution",
			Description: "Begin implementing the tasks",
			Command:     r.CS("next --answer=\"start\"", st.Spec),
		},
		{
			Label:       "Save for later",
			Description: "Spec is approved but don't start execution yet. Others can still add ACs or notes.",
			Command:     r.CS("next --answer=\"save\"", st.Spec),
		},
	}
}

func buildDiscoveryOptions(r Renderer, st state.StateFile) []model.InteractiveOption {
	mode := st.Discovery.Mode
	hasUserContext := st.Discovery.UserContext != nil && len(*st.Discovery.UserContext) > 0
	hasDescription := st.SpecDescription != nil && len(*st.SpecDescription) > 0
	hasPlan := st.Discovery.PlanPath != nil
	answeredCount := len(st.Discovery.Answers)

	if mode == nil && hasDescription && answeredCount == 0 && !hasPlan && hasUserContext {
		modes := model.DiscoveryModeOptions()
		opts := make([]model.InteractiveOption, 0, len(modes))
		for _, m := range modes {
			opts = append(opts, model.InteractiveOption{
				Label:       m.Label,
				Description: m.Description,
				Command:     r.CS("next --answer=\""+m.ID+"\"", st.Spec),
			})
		}
		return opts
	}
	return nil
}
