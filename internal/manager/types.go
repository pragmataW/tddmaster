
// Manager types — shared across all manager modules.

package manager

// TabMode represents the mode of a manager tab.
type TabMode string

const (
	TabModeSpec TabMode = "spec"
	TabModeFree TabMode = "free"
)

// Focus represents the focus state of the manager.
type Focus string

const (
	FocusList     Focus = "list"
	FocusTerminal Focus = "terminal"
)

// ManagerTab represents a single tab in the TUI manager.
type ManagerTab struct {
	ID        string
	Spec      *string // nil if no spec
	Mode      TabMode
	SessionID string
	Buffer    []string
	Active    bool
	Phase     *string // nil if no phase
}

// ManagerState represents the full state of the TUI manager.
type ManagerState struct {
	Tabs              []ManagerTab
	SelectedTabIndex  int
	Focus             Focus
	Running           bool
	SpecsVisible      bool
	MonitorVisible    bool
}

// CreateInitialState returns a new ManagerState with default values.
func CreateInitialState() ManagerState {
	return ManagerState{
		Tabs:             []ManagerTab{},
		SelectedTabIndex: -1,
		Focus:            FocusList,
		Running:          true,
		SpecsVisible:     true,
		MonitorVisible:   true,
	}
}
