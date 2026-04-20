package meta

import (
	"fmt"
	"time"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/state"
)

// BuildProtocolGuide returns a protocol reminder for new sessions or when the
// last call is stale (> StaleSessionMS). Returns nil when the session is
// active and the caller already saw a protocol guide recently.
func BuildProtocolGuide(r Renderer, st state.StateFile) *model.ProtocolGuide {
	if st.LastCalledAt == nil {
		return &model.ProtocolGuide{
			What:         "tddmaster orchestrates your work: IDLE → DISCOVERY → DISCOVERY_REFINEMENT → SPEC_PROPOSAL → SPEC_APPROVED → EXECUTING → DONE → IDLE",
			How:          fmt.Sprintf("Run `%s` for instructions. Submit results with `%s`. Never make architectural decisions without asking.", r.CS("next", st.Spec), r.CS("next --answer=\"...\"", st.Spec)),
			CurrentPhase: string(st.Phase),
		}
	}

	lastCalledStr := *st.LastCalledAt
	lastCalled, err := time.Parse(time.RFC3339, lastCalledStr)
	if err != nil {
		lastCalled, err = time.Parse("2006-01-02T15:04:05.000Z", lastCalledStr)
	}
	if err == nil {
		elapsed := time.Since(lastCalled).Milliseconds()
		if elapsed > model.StaleSessionMS {
			return &model.ProtocolGuide{
				What:         "tddmaster orchestrates your work: IDLE → DISCOVERY → DISCOVERY_REFINEMENT → SPEC_PROPOSAL → SPEC_APPROVED → EXECUTING → DONE → IDLE",
				How:          fmt.Sprintf("Run `%s` for instructions. Submit results with `%s`. Never make architectural decisions without asking.", r.CS("next", st.Spec), r.CS("next --answer=\"...\"", st.Spec)),
				CurrentPhase: string(st.Phase),
			}
		}
	}
	return nil
}
