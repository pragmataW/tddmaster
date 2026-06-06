package manifest

const (
	defaultCommand      = "tddmaster"
	defaultMaxIteration = 15
)

type Manifest struct {
	SelectedTools           []ToolID `json:"selectedTools"`
	MaxIterationBeforeStart int      `json:"maxIterationBeforeStart"`
	Command                 string   `json:"command"`
}

func Defaults() Manifest {
	return Manifest{
		SelectedTools:           []ToolID{},
		MaxIterationBeforeStart: defaultMaxIteration,
		Command:                 defaultCommand,
	}
}

func Normalize(m *Manifest) {
	seen := make(map[ToolID]struct{})
	deduped := make([]ToolID, 0, len(m.SelectedTools))
	for _, t := range m.SelectedTools {
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			deduped = append(deduped, t)
		}
	}
	m.SelectedTools = deduped

	if m.MaxIterationBeforeStart <= 0 {
		m.MaxIterationBeforeStart = defaultMaxIteration
	}

	if m.Command == "" {
		m.Command = defaultCommand
	}
}
