package phases

import (
	"regexp"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/spec"
	"github.com/pragmataW/tddmaster/internal/state"
)

// nonTDDKeywordRe matches titles that almost never benefit from TDD — pure
// plumbing/scaffolding work. Used only to populate SuggestedTDD as an advisory
// hint; the user's explicit answer is the source of truth.
var nonTDDKeywordRe = regexp.MustCompile(`(?i)\b(download|install|scaffold|bootstrap|go\s+mod|go\.mod|init(?:ialize)?|create\s+(?:directory|folder|project|skeleton)|add\s+dependenc(?:y|ies)|configure\s+ci)\b`)

func suggestTDDForTitle(title string) bool {
	return !nonTDDKeywordRe.MatchString(title)
}

// buildTDDSelectionEntries returns the canonical task list for the TDD
// selection UI. Prefers StateFile.OverrideTasks when present, falling back to
// the parsed spec.md tasks.
func buildTDDSelectionEntries(st state.StateFile, parsedSpec *spec.ParsedSpec) []model.TaskTDDSelectionEntry {
	if len(st.OverrideTasks) > 0 {
		entries := make([]model.TaskTDDSelectionEntry, 0, len(st.OverrideTasks))
		for _, t := range st.OverrideTasks {
			entries = append(entries, model.TaskTDDSelectionEntry{
				ID:           t.ID,
				Title:        t.Title,
				SuggestedTDD: suggestTDDForTitle(t.Title),
			})
		}
		return entries
	}
	if parsedSpec == nil || len(parsedSpec.Tasks) == 0 {
		return nil
	}
	entries := make([]model.TaskTDDSelectionEntry, 0, len(parsedSpec.Tasks))
	for _, t := range parsedSpec.Tasks {
		entries = append(entries, model.TaskTDDSelectionEntry{
			ID:           t.ID,
			Title:        t.Title,
			SuggestedTDD: suggestTDDForTitle(t.Title),
		})
	}
	return entries
}

// CompileSpecApproved renders the SPEC_APPROVED phase. When per-task TDD
// selection is still pending, the TDD selection block is attached.
func CompileSpecApproved(r Renderer, st state.StateFile, config *state.NosManifest, parsedSpec *spec.ParsedSpec) model.SpecApprovedOutput {
	specPath := ""
	if st.SpecState.Path != nil {
		specPath = *st.SpecState.Path
	}
	out := model.SpecApprovedOutput{
		Phase:       "SPEC_APPROVED",
		Instruction: model.SpecApprovedWaitingInstruction,
		SpecPath:    specPath,
		Transition: model.TransitionStart{
			OnStart: r.CS("next --answer=\"start\"", st.Spec),
		},
	}

	tddSelectionPending := config != nil && config.IsTDDEnabled() &&
		(st.TaskTDDSelected == nil || !*st.TaskTDDSelected)
	importantSelectionPending := config != nil && config.IsImportantTaskGateEnabled() &&
		(st.ImportantTasksReviewed == nil || !*st.ImportantTasksReviewed)

	if tddSelectionPending {
		entries := buildTDDSelectionEntries(st, parsedSpec)
		if len(entries) > 0 {
			out.Instruction = model.SpecApprovedTDDInstruction
			out.TaskTDDSelection = &model.TaskTDDSelectionOutput{
				Required:    true,
				Instruction: model.TaskTDDSelectionInstruction,
				Tasks:       entries,
				Answers: model.TaskTDDSelectionAnswers{
					All:    "tdd-all",
					None:   "tdd-none",
					Custom: `{"tddTasks":["task-1","task-3"]}`,
				},
			}
		}
	}

	if importantSelectionPending {
		entries := buildImportantSelectionEntries(st, parsedSpec)
		if len(entries) > 0 {
			out.ImportantTaskSelection = &model.ImportantTaskSelectionOutput{
				Required:    true,
				Instruction: "Use AskUserQuestion with multiSelect=true over importantTaskSelection.tasks. Each task the user picks will pass through the tddmaster-planner subagent for a plan-first review before execution. Submit the selected IDs as JSON: {\"importantTaskIds\":[\"task-1\",\"task-3\"]}. Submit {\"importantTaskIds\":[]} to skip without flagging any.",
				Tasks:       entries,
				AnswerShape: `{"importantTaskIds":["task-1","task-3"]}`,
			}
			if out.TaskTDDSelection == nil {
				out.Instruction = "Before starting execution, review the task list and flag any 'important' tasks (architectural risk, security paths, core abstractions). Each flagged task pauses for a tddmaster-planner plan-first review."
			}
		}
	}
	return out
}

// buildImportantSelectionEntries returns the task list shown in the bulk
// "mark important?" review. Prefers OverrideTasks (state-authoritative) and
// falls back to the parsed spec.
func buildImportantSelectionEntries(st state.StateFile, parsedSpec *spec.ParsedSpec) []model.ImportantTaskSelectionEntry {
	if len(st.OverrideTasks) > 0 {
		entries := make([]model.ImportantTaskSelectionEntry, 0, len(st.OverrideTasks))
		for _, t := range st.OverrideTasks {
			entries = append(entries, model.ImportantTaskSelectionEntry{ID: t.ID, Title: t.Title})
		}
		return entries
	}
	if parsedSpec == nil || len(parsedSpec.Tasks) == 0 {
		return nil
	}
	entries := make([]model.ImportantTaskSelectionEntry, 0, len(parsedSpec.Tasks))
	for _, t := range parsedSpec.Tasks {
		entries = append(entries, model.ImportantTaskSelectionEntry{ID: t.ID, Title: t.Title})
	}
	return entries
}
