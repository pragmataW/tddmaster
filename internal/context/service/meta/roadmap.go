package meta

import (
	"strings"

	"github.com/pragmataW/tddmaster/internal/context/model"
	"github.com/pragmataW/tddmaster/internal/state"
)

// BuildRoadmap returns the phase roadmap string with the current phase
// highlighted.
func BuildRoadmap(phase state.Phase) string {
	if phase == state.PhaseBlocked {
		parts := make([]string, len(model.RoadmapPhases))
		for i, p := range model.RoadmapPhases {
			if p.Key == "EXECUTING" {
				parts[i] = "[ EXECUTING (BLOCKED) ]"
			} else {
				parts[i] = p.Label
			}
		}
		return strings.Join(parts, " → ")
	}

	parts := make([]string, len(model.RoadmapPhases))
	for i, p := range model.RoadmapPhases {
		switch {
		case p.Key == "IDLE" && phase == state.PhaseIdle:
			parts[i] = "[ IDLE ]"
		case state.Phase(p.Key) == phase:
			parts[i] = "[ " + p.Label + " ]"
		default:
			parts[i] = p.Label
		}
	}
	return strings.Join(parts, " → ")
}
