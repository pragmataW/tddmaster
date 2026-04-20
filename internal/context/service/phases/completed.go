package phases

import (
	"fmt"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/state"
)

// CompileCompleted renders the COMPLETED phase payload — completion summary
// plus the learning-capture prompt.
func CompileCompleted(st state.StateFile) model.CompletedOutput {
	learningsTrue := true
	name := ""
	if st.Spec != nil {
		name = *st.Spec
	}

	summary := model.CompletionSummary{
		Spec:             st.Spec,
		Iterations:       st.Execution.Iteration,
		DecisionsCount:   len(st.Decisions),
		CompletionReason: st.CompletionReason,
		CompletionNote:   st.CompletionNote,
	}

	return model.CompletedOutput{
		Phase:            "COMPLETED",
		Summary:          summary,
		LearningsPending: &learningsTrue,
		LearningPrompt: &model.LearningPrompt{
			Instruction: fmt.Sprintf("LEARNING PENDING — Record learnings before moving on. For each insight, decide: one-time learning or permanent rule? One-time (\"assumed X, was Y\") → `learn \"text\"`. Permanent (\"always/never do X\") → `learn \"text\" --rule`. Run: `tddmaster spec %s learn \"text\"` or `learn \"text\" --rule`.", name),
			Examples: []string{
				fmt.Sprintf("tddmaster spec %s learn \"Assumed S3 SDK v2, was v3\"", name),
				fmt.Sprintf("tddmaster spec %s learn \"Always use Result types\" --rule", name),
				"tddmaster learn promote 1",
			},
		},
	}
}
