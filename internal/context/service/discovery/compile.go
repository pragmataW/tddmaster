// Package discovery compiles the DISCOVERY and DISCOVERY_REFINEMENT phase
// outputs: listen-first, mode selection, premise challenge, per-question
// question presentation (normal + agent mode), and review summary with
// optional split proposals.
package discovery

import (
	"fmt"
	"strings"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/context/service/concerns"
	"github.com/pragmataW/tddmaster/internal/state"
)

// Renderer is the minimal command-builder interface required by this package.
type Renderer interface {
	C(sub string) string
	CS(sub string, specName *string) string
}

// attachCommonFields fills in the optional user/notes/premises fields that
// every DISCOVERY branch ships identically. Centralising this kills the
// repeat-yourself inline struct literals that used to sit in every branch.
func attachCommonFields(
	out *model.DiscoveryOutput,
	currentUser *model.CurrentUser,
	specNotes []model.SpecNote,
	agreedPremises []string,
	revisedPremises []model.RevisedPremise,
) {
	if currentUser != nil {
		u := *currentUser
		out.CurrentUser = &u
	}
	if len(specNotes) > 0 {
		out.Notes = specNotes
	}
	if len(agreedPremises) > 0 {
		out.AgreedPremises = agreedPremises
	}
	if len(revisedPremises) > 0 {
		out.RevisedPremises = revisedPremises
	}
}

func collectSpecNotes(st state.StateFile) []model.SpecNote {
	var notes []model.SpecNote
	for _, n := range st.SpecNotes {
		if strings.HasPrefix(n.Text, "[TASK] ") {
			continue
		}
		notes = append(notes, model.SpecNote{Text: n.Text, User: n.User})
	}
	return notes
}

func collectPremises(st state.StateFile) (agreed []string, revised []model.RevisedPremise) {
	for _, p := range st.Discovery.Premises {
		if p.Agreed {
			agreed = append(agreed, p.Text)
		} else if p.Revision != nil {
			revised = append(revised, model.RevisedPremise{Original: p.Text, Revision: *p.Revision})
		}
	}
	return
}

// Compile renders the DISCOVERY phase. Dispatches across listen-first, mode
// selection, premise challenge, all-answered, agent, revisited, and normal
// branches.
func Compile(
	r Renderer,
	st state.StateFile,
	activeConcerns []state.ConcernDefinition,
	rules []string,
	currentUser *model.CurrentUser,
) model.DiscoveryOutput {
	allQuestions := GetQuestionsWithExtras(activeConcerns)
	answeredCount := len(st.Discovery.Answers)
	allAnswered := isDiscoveryComplete(st.Discovery.Answers)
	isAgent := st.Discovery.Audience == "agent"

	specNotes := collectSpecNotes(st)
	agreedPremises, revisedPremises := collectPremises(st)

	hasUserContext := st.Discovery.UserContext != nil && len(*st.Discovery.UserContext) > 0
	hasDescription := st.SpecDescription != nil && len(*st.SpecDescription) > 0
	hasPlan := st.Discovery.PlanPath != nil
	mode := st.Discovery.Mode

	reminders := concerns.GetReminders(activeConcerns, nil)
	baseCtx := model.ContextBlock{Rules: rules, ConcernReminders: reminders}

	// Listen-first step.
	if mode == nil && !hasUserContext && answeredCount == 0 && !hasPlan && hasDescription {
		out := model.DiscoveryOutput{
			Phase:         "DISCOVERY",
			Instruction:   model.DiscoveryListenFirstInstruction,
			Questions:     []model.DiscoveryQuestion{},
			AnsweredCount: 0,
			Context:       baseCtx,
			Transition:    model.TransitionOnComplete{OnComplete: r.CS("next --answer=\"<user context or just start>\"", st.Spec)},
		}
		attachCommonFields(&out, currentUser, specNotes, nil, nil)
		return out
	}

	// Mode selection step.
	if mode == nil && hasDescription && answeredCount == 0 && !hasPlan {
		ms := &model.ModeSelectionOutput{
			Required:    true,
			Instruction: model.ModeSelectionOutputInstruction,
			Options:     model.DiscoveryModeOptions(),
		}
		out := model.DiscoveryOutput{
			Phase:         "DISCOVERY",
			Instruction:   model.DiscoveryModeSelectionInstruction,
			Questions:     []model.DiscoveryQuestion{},
			AnsweredCount: 0,
			Context:       baseCtx,
			Transition:    model.TransitionOnComplete{OnComplete: r.CS("next --answer=\"<mode>\"", st.Spec)},
			ModeSelection: ms,
		}
		attachCommonFields(&out, currentUser, nil, nil, nil)
		return out
	}

	// Premise challenge step.
	premisesCompleted := st.Discovery.PremisesCompleted != nil && *st.Discovery.PremisesCompleted
	if mode != nil && !premisesCompleted && !allAnswered {
		planNote := ""
		if st.Discovery.PlanPath != nil {
			planNote = " and the plan document"
		}
		out := model.DiscoveryOutput{
			Phase:         "DISCOVERY",
			Instruction:   model.DiscoveryPremiseInstruction,
			Questions:     []model.DiscoveryQuestion{},
			AnsweredCount: 0,
			Context:       baseCtx,
			Transition:    model.TransitionOnComplete{OnComplete: r.CS("next --answer='{\"premises\":[]}'", st.Spec)},
			PremiseChallenge: &model.PremiseChallengeOutput{
				Required:    true,
				Instruction: fmt.Sprintf(model.PremiseChallengeInstructionFmt, planNote),
				Prompts:     model.PremiseChallengePrompts,
			},
		}
		attachCommonFields(&out, currentUser, nil, nil, nil)
		return out
	}

	// Mode-specific rules.
	var modeRules []string
	if mode != nil {
		modeRules = getModeRules(*mode)
	}
	rulesWithMode := append(rules, modeRules...)

	// Rich description context.
	specDescription := ""
	if st.SpecDescription != nil {
		specDescription = *st.SpecDescription
	}
	isRichDescription := len(specDescription) > model.RichDescriptionThreshold
	hasPersistedPrefills := len(st.Discovery.Prefills) > 0

	if allAnswered {
		history := st.RevisitHistory
		var lastRevisit *state.RevisitEntry
		if len(history) > 0 {
			lastRevisit = &history[len(history)-1]
		}

		instrBase := fmt.Sprintf("All discovery questions answered. Run: `%s`", r.CS("approve", st.Spec))
		if lastRevisit != nil {
			instrBase = "This spec was revisited from EXECUTING. All previous answers are preserved. Review and approve, or revise answers before regenerating the spec."
		}

		base := model.DiscoveryOutput{
			Phase:         "DISCOVERY",
			Instruction:   instrBase,
			Questions:     []model.DiscoveryQuestion{},
			AnsweredCount: answeredCount,
			Context:       model.ContextBlock{Rules: rules, ConcernReminders: []string{}},
			Transition:    model.TransitionOnComplete{OnComplete: r.CS("approve", st.Spec)},
		}
		attachCommonFields(&base, currentUser, specNotes, nil, nil)

		if lastRevisit != nil {
			trueVal := true
			reason := lastRevisit.Reason
			base.Revisited = &trueVal
			base.RevisitReason = &reason
			base.PreviousProgress = &model.PreviousProgress{
				CompletedTasks: lastRevisit.CompletedTasks,
				TotalTasks:     len(lastRevisit.CompletedTasks),
			}
		}
		return base
	}

	currentQ, currentIdx := selectCurrentDiscoveryQuestion(allQuestions, st.Discovery.Answers, st.Discovery.CurrentQuestion)

	// Agent mode: return only the current question.
	if isAgent {
		if currentQ == nil {
			return model.DiscoveryOutput{
				Phase:         "DISCOVERY",
				Instruction:   fmt.Sprintf("All discovery questions answered. Run: `%s`", r.CS("approve", st.Spec)),
				Questions:     []model.DiscoveryQuestion{},
				AnsweredCount: answeredCount,
				Context:       model.ContextBlock{Rules: rules, ConcernReminders: []string{}},
				Transition:    model.TransitionOnComplete{OnComplete: r.CS("approve", st.Spec)},
			}
		}

		question := buildDiscoveryQuestion(*currentQ, st.Discovery.Prefills)
		total := len(allQuestions)
		agentOut := model.DiscoveryOutput{
			Phase: "DISCOVERY",
			Instruction: fmt.Sprintf("Ask this question to the user using AskUserQuestion. Submit the answer with: `%s`",
				r.CS("next --agent --answer=\"<answer>\"", st.Spec)),
			Questions:       []model.DiscoveryQuestion{question},
			AnsweredCount:   answeredCount,
			CurrentQuestion: &currentIdx,
			TotalQuestions:  &total,
			Context:         model.ContextBlock{Rules: rulesWithMode, ConcernReminders: reminders},
			Transition:      model.TransitionOnComplete{OnComplete: r.CS("next --agent --answer=\"<answer>\"", st.Spec)},
		}

		attachCommonFields(&agentOut, nil, nil, agreedPremises, revisedPremises)

		if currentIdx == 0 {
			if research := buildPreDiscoveryResearch(specDescription); research != nil {
				agentOut.PreDiscoveryResearch = research
			}
			planPath := ""
			if st.Discovery.PlanPath != nil {
				planPath = *st.Discovery.PlanPath
			}
			if planCtx := buildPlanContext(planPath); planCtx != nil {
				agentOut.PlanContext = planCtx
			} else if isRichDescription && !hasPersistedPrefills {
				agentOut.RichDescription = &model.RichDescriptionOutput{
					Provided:    true,
					Length:      len(specDescription),
					Content:     specDescription,
					Instruction: model.RichDescriptionInstructionAgent,
				}
			}
		}

		var pendingFU []state.FollowUp
		for _, f := range st.Discovery.FollowUps {
			if f.Status == "pending" {
				pendingFU = append(pendingFU, f)
			}
		}
		if len(pendingFU) > 0 {
			agentOut.PendingFollowUps = pendingFU
		}

		if len(st.Discovery.Answers) > 0 {
			lastAnswer := st.Discovery.Answers[len(st.Discovery.Answers)-1]
			if followHints := generateFollowUpHints(lastAnswer.Answer); len(followHints) > 0 {
				agentOut.FollowUpHints = followHints
			}
		}
		return agentOut
	}

	history := st.RevisitHistory
	var lastRevisit *state.RevisitEntry
	if len(history) > 0 {
		lastRevisit = &history[len(history)-1]
	}
	isRevisited := lastRevisit != nil

	revisitInstruction := model.DiscoveryNormalInstruction
	if isRevisited {
		revisitInstruction = model.DiscoveryRevisitedInstruction
	}

	var questions []model.DiscoveryQuestion
	if currentQ != nil {
		questions = []model.DiscoveryQuestion{buildDiscoveryQuestion(*currentQ, st.Discovery.Prefills)}
	} else {
		questions = []model.DiscoveryQuestion{}
	}

	total := len(allQuestions)
	out := model.DiscoveryOutput{
		Phase:           "DISCOVERY",
		Instruction:     revisitInstruction,
		Questions:       questions,
		AnsweredCount:   answeredCount,
		CurrentQuestion: &currentIdx,
		TotalQuestions:  &total,
		Context:         model.ContextBlock{Rules: rulesWithMode, ConcernReminders: reminders},
		Transition:      model.TransitionOnComplete{OnComplete: r.CS("next --answer=\"<answer>\"", st.Spec)},
	}

	attachCommonFields(&out, currentUser, specNotes, agreedPremises, revisedPremises)

	if isRevisited {
		trueVal := true
		reason := lastRevisit.Reason
		out.Revisited = &trueVal
		out.RevisitReason = &reason
		out.PreviousProgress = &model.PreviousProgress{
			CompletedTasks: lastRevisit.CompletedTasks,
			TotalTasks:     len(lastRevisit.CompletedTasks),
		}
		return out
	}

	if answeredCount == 0 {
		if research := buildPreDiscoveryResearch(specDescription); research != nil {
			out.PreDiscoveryResearch = research
		}
		planPath := ""
		if st.Discovery.PlanPath != nil {
			planPath = *st.Discovery.PlanPath
		}
		if planCtx := buildPlanContext(planPath); planCtx != nil {
			out.PlanContext = planCtx
		} else if isRichDescription && !hasPersistedPrefills {
			out.RichDescription = &model.RichDescriptionOutput{
				Provided:    true,
				Length:      len(specDescription),
				Content:     specDescription,
				Instruction: model.RichDescriptionInstructionUser,
			}
		}
	}

	return out
}
