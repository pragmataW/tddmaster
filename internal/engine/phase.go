package engine

type PhaseID string

const PhaseComplete PhaseID = "completed"

type PhaseDef struct {
	ID     PhaseID
	Driver Driver
}

func NextPhase(defs []PhaseDef, current PhaseID) PhaseID {
	for i, d := range defs {
		if d.ID == current {
			if i+1 < len(defs) {
				return defs[i+1].ID
			}
			return PhaseComplete
		}
	}
	return PhaseComplete
}

type PhaseProgress struct {
	Phase   PhaseID          `json:"phase"`
	Modules []ModuleProgress `json:"modules"`
}
