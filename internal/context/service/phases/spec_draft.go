package phases

import (
	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
)

// CompileSpecDraft renders the SPEC_PROPOSAL phase. When a spec has no
// classification yet, the classification prompt is attached instead of the
// self-review block.
func CompileSpecDraft(r Renderer, st state.StateFile) model.SpecDraftOutput {
	edgeCases := spec.DeriveEdgeCases(st.Discovery.Answers, st.Discovery.Premises)

	specPath := ""
	if st.SpecState.Path != nil {
		specPath = *st.SpecState.Path
	}

	if st.Classification == nil {
		classTrue := true
		classPrompt := &model.ClassificationPrompt{
			Options: model.ClassificationOptions,
			Instruction: "Select all that apply. Submit as JSON: `" +
				r.CS("next --answer='{\"involvesWebUI\":true,\"involvesCLI\":false,\"involvesPublicAPI\":false,...}'", st.Spec) +
				"`. If none apply, answer with: `" +
				r.CS("next --answer=\"none\"", st.Spec) + "`",
		}

		return model.SpecDraftOutput{
			Phase:       "SPEC_PROPOSAL",
			Instruction: model.SpecClassifyInstruction,
			SpecPath:    specPath,
			EdgeCases:   edgeCases,
			Transition: model.TransitionApprove{
				OnApprove: r.CS("next --answer='{\"involvesWebUI\":false,\"involvesCLI\":false,\"involvesPublicAPI\":false,\"involvesMigration\":false,\"involvesDataHandling\":false}'", st.Spec),
			},
			ClassificationRequired: &classTrue,
			ClassificationPrompt:   classPrompt,
		}
	}

	return model.SpecDraftOutput{
		Phase:       "SPEC_PROPOSAL",
		Instruction: model.SpecDraftReadyInstruction,
		SpecPath:    specPath,
		EdgeCases:   edgeCases,
		Transition: model.TransitionApprove{
			OnApprove: r.CS("approve", st.Spec),
		},
		SelfReview: &model.SelfReview{
			Required:    true,
			Checks:      model.SpecSelfReviewChecks,
			Instruction: model.SpecSelfReviewInstructionFmt,
		},
	}
}
