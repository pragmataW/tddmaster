package phases

import (
	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/state"
)

// CompileBlocked renders the BLOCKED phase payload with the last recorded
// progress string as the blocking reason.
func CompileBlocked(r Renderer, st state.StateFile) model.BlockedOutput {
	reason := "Unknown"
	if st.Execution.LastProgress != nil {
		reason = *st.Execution.LastProgress
	}
	return model.BlockedOutput{
		Phase:       "BLOCKED",
		Instruction: model.BlockedInstruction,
		Reason:      reason,
		Transition: model.TransitionResolved{
			OnResolved: r.CS("next --answer=\"...\"", st.Spec),
		},
	}
}
