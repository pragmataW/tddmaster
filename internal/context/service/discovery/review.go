package discovery

import (
	"fmt"
	"strings"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/context/service/concerns"
	"github.com/pragmataW/tddmaster/internal/context/service/split"
	"github.com/pragmataW/tddmaster/internal/state"
)

// CompileReview renders the DISCOVERY_REFINEMENT phase — answer review,
// optional split proposal, alternatives prompt, and review checklist.
func CompileReview(r Renderer, st state.StateFile, activeConcerns []state.ConcernDefinition) model.DiscoveryReviewOutput {
	allQuestions := GetQuestionsWithExtras(activeConcerns)
	questionMap := make(map[string]string)
	for _, q := range allQuestions {
		questionMap[q.ID] = q.Text
	}

	var reviewAnswers []model.DiscoveryReviewAnswer
	for _, a := range st.Discovery.Answers {
		qText, ok := questionMap[a.QuestionID]
		if !ok {
			qText = a.QuestionID
		}
		reviewAnswers = append(reviewAnswers, model.DiscoveryReviewAnswer{
			QuestionID: a.QuestionID,
			Question:   qText,
			Answer:     a.Answer,
		})
	}
	reviewSummary := buildReviewSummary(reviewAnswers)

	splitProposal := split.Analyze(st.Discovery.Answers, "")

	if st.Discovery.Approved && splitProposal.Detected {
		return model.DiscoveryReviewOutput{
			Phase:         "DISCOVERY_REFINEMENT",
			Instruction:   model.DiscoveryReviewSplitApprovedInstruction,
			Answers:       reviewAnswers,
			ReviewSummary: reviewSummary,
			Transition: model.TransitionApproveRevise{
				OnApprove: r.CS("next --answer=\"keep\"", st.Spec),
				OnRevise:  r.CS("next --answer='{\"revise\":{\"status_quo\":\"corrected answer\"}}'", st.Spec),
			},
			SplitProposal: &splitProposal,
		}
	}

	alternativesPresented := st.Discovery.AlternativesPresented != nil && *st.Discovery.AlternativesPresented
	if st.Discovery.Approved && !alternativesPresented {
		subPhase := "alternatives"
		alt := &model.AlternativesOutput{
			Required:    true,
			Instruction: model.AlternativesInstruction,
		}
		alt.Format.Fields = model.AlternativesFields

		return model.DiscoveryReviewOutput{
			Phase:         "DISCOVERY_REFINEMENT",
			SubPhase:      &subPhase,
			Instruction:   model.DiscoveryReviewAlternativesInstruction,
			Answers:       reviewAnswers,
			ReviewSummary: reviewSummary,
			Transition: model.TransitionApproveRevise{
				OnApprove: r.CS("next --answer='{\"approach\":\"A\",\"name\":\"...\",\"summary\":\"...\",\"effort\":\"M\",\"risk\":\"Low\"}'", st.Spec),
				OnRevise:  r.CS("next --answer=\"skip\"", st.Spec),
			},
			Alternatives: alt,
		}
	}

	batchWarning := ""
	if st.Discovery.BatchSubmitted != nil && *st.Discovery.BatchSubmitted {
		batchWarning = model.BatchWarning
	}

	allDimensions := concerns.GetReviewDimensions(activeConcerns, nil)
	registryIDs := concerns.GetRegistryDimensionIDs(activeConcerns)
	registrySet := make(map[string]bool)
	for _, id := range registryIDs {
		registrySet[id] = true
	}

	var reviewChecklist *model.ReviewChecklist
	if len(allDimensions) > 0 {
		var checklistDims []model.ReviewChecklistDimension
		hasRegistries := false
		for _, dim := range allDimensions {
			isReg := registrySet[dim.ID]
			if isReg {
				hasRegistries = true
			}
			checklistDims = append(checklistDims, model.ReviewChecklistDimension{
				ID:               dim.ID,
				Label:            dim.Label,
				Prompt:           dim.Prompt,
				EvidenceRequired: dim.EvidenceRequired,
				IsRegistry:       isReg,
				ConcernID:        dim.ConcernID,
			})
		}

		rc := &model.ReviewChecklist{
			Dimensions:  checklistDims,
			Instruction: model.ReviewChecklistInstruction,
		}
		if hasRegistries {
			regInstr := model.ReviewChecklistRegistryInstruction
			rc.RegistryInstruction = &regInstr
		}
		reviewChecklist = rc
	}

	var instruction string
	if splitProposal.Detected {
		instruction = model.DiscoveryReviewSplitInstruction + batchWarning
	} else {
		instruction = model.DiscoveryReviewDefaultInstruction + batchWarning
	}

	result := model.DiscoveryReviewOutput{
		Phase:         "DISCOVERY_REFINEMENT",
		Instruction:   instruction,
		Answers:       reviewAnswers,
		ReviewSummary: reviewSummary,
		Transition: model.TransitionApproveRevise{
			OnApprove: r.CS("next --answer=\"approve\"", st.Spec),
			OnRevise:  r.CS("next --answer='{\"revise\":{\"status_quo\":\"corrected answer\"}}'", st.Spec),
		},
		ReviewChecklist: reviewChecklist,
	}
	if splitProposal.Detected {
		result.SplitProposal = &splitProposal
	}
	return result
}

func buildReviewSummary(answers []model.DiscoveryReviewAnswer) string {
	if len(answers) == 0 {
		return ""
	}
	lines := make([]string, 0, len(answers)*2)
	for i, answer := range answers {
		lines = append(lines, fmt.Sprintf("%d. [%s] %s", i+1, answer.QuestionID, answer.Question))
		lines = append(lines, "Answer: "+answer.Answer)
	}
	return strings.Join(lines, "\n")
}
