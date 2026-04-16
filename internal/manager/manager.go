
// TUI manager — bubbletea Model/Update/View implementation.

package manager

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// =============================================================================
// Messages
// =============================================================================

// QuitMsg is sent when the user requests to quit.
type QuitMsg struct{}

// NewTabMsg is sent when the user requests a new tab.
type NewTabMsg struct {
	Tab ManagerTab
}

// CloseTabMsg is sent when the user requests to close a tab.
type CloseTabMsg struct {
	TabID string
}

// AppendBufferMsg appends data to a tab's buffer.
type AppendBufferMsg struct {
	TabID string
	Data  string
}

// UpdateTabPhaseMsg updates the phase of a tab.
type UpdateTabPhaseMsg struct {
	TabID string
	Phase string
}

// =============================================================================
// Model
// =============================================================================

// Model is the bubbletea model for the TUI manager.
type Model struct {
	State  ManagerState
	Width  int
	Height int
}

// NewModel creates a new Model with the given dimensions.
func NewModel(width, height int) Model {
	return Model{
		State:  CreateInitialState(),
		Width:  width,
		Height: height,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case QuitMsg:
		m.State.Running = false
		return m, tea.Quit

	case NewTabMsg:
		m.State = CreateTab(m.State, msg.Tab)
		return m, nil

	case CloseTabMsg:
		m.State = CloseTab(m.State, msg.TabID)
		return m, nil

	case AppendBufferMsg:
		for i := range m.State.Tabs {
			if m.State.Tabs[i].ID == msg.TabID {
				AppendToBuffer(&m.State.Tabs[i], msg.Data)
				break
			}
		}
		return m, nil

	case UpdateTabPhaseMsg:
		for i := range m.State.Tabs {
			if m.State.Tabs[i].ID == msg.TabID {
				phase := msg.Phase
				m.State.Tabs[i].Phase = &phase
				break
			}
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	ctrl := isCtrl(msg)
	key := normalizeKey(msg)

	action := RouteKey(m.State, key, ctrl)

	switch action.Type {
	case KeyActionQuit:
		m.State.Running = false
		return m, tea.Quit

	case KeyActionToggleFocus:
		m.State = ToggleFocus(m.State)

	case KeyActionNavigate:
		itemCount := len(m.State.Tabs)
		if m.State.Focus == FocusList {
			// When in list focus, navigate with spec count placeholder
			// (actual spec list navigation uses tab count)
			m.State = NavigateList(m.State, action.Direction, itemCount)
		}

	case KeyActionSelect:
		// Select the currently highlighted tab
		m.State = SwitchTab(m.State, m.State.SelectedTabIndex)

	case KeyActionNewTab:
		// Caller should send NewTabMsg to actually create tabs
		// Here we create a free tab by default
		tab := ManagerTab{
			ID:        fmt.Sprintf("tab-%d", len(m.State.Tabs)+1),
			Mode:      TabModeFree,
			SessionID: fmt.Sprintf("session-%d", len(m.State.Tabs)+1),
			Buffer:    []string{},
		}
		m.State = CreateTab(m.State, tab)

	case KeyActionCloseTab:
		if activeTab := GetActiveTab(m.State); activeTab != nil {
			m.State = CloseTab(m.State, activeTab.ID)
		}

	case KeyActionToggleSpecs:
		m.State.SpecsVisible = !m.State.SpecsVisible

	case KeyActionToggleMonitor:
		m.State.MonitorVisible = !m.State.MonitorVisible
	}

	return m, nil
}

func isCtrl(msg tea.KeyMsg) bool {
	return strings.HasPrefix(msg.String(), "ctrl+")
}

func normalizeKey(msg tea.KeyMsg) string {
	s := msg.String()
	// bubbletea uses "ctrl+c", "ctrl+d", etc.
	if strings.HasPrefix(s, "ctrl+") {
		return s[5:]
	}
	switch s {
	case "up":
		return "up"
	case "down":
		return "down"
	case "tab":
		return "tab"
	case "esc":
		return "escape"
	case "enter":
		return "return"
	}
	return s
}

// View implements tea.Model.
func (m Model) View() string {
	if m.Width == 0 || m.Height == 0 {
		return "Initializing..."
	}

	var sb strings.Builder

	// Layout: left panel (specs), right panels (monitor top, terminal bottom)
	// Leave 1 row for tab bar above terminal
	leftWidth := m.Width / 3
	rightWidth := m.Width - leftWidth
	monitorHeight := m.Height / 3
	terminalHeight := m.Height - monitorHeight - 1 // -1 for tab bar

	// Panels (1-based coordinates)
	leftPanel := Panel{ID: "left", X: 1, Y: 1, Width: leftWidth, Height: m.Height}
	rightTopPanel := Panel{ID: "rightTop", X: leftWidth + 1, Y: 1, Width: rightWidth, Height: monitorHeight}
	rightBottomPanel := Panel{ID: "rightBottom", X: leftWidth + 1, Y: monitorHeight + 2, Width: rightWidth, Height: terminalHeight}

	_ = leftPanel
	_ = rightTopPanel
	_ = rightBottomPanel

	// Render spec list (left panel)
	if m.State.SpecsVisible {
		specList := RenderSpecList(
			[]SpecInfo{}, // specs come from external data
			m.State.Tabs,
			m.State.SelectedTabIndex,
			leftPanel.X, leftPanel.Y, leftPanel.Width, leftPanel.Height,
		)
		sb.WriteString(specList)
	}

	// Render monitor (right top)
	if m.State.MonitorVisible {
		activeTab := GetActiveTab(m.State)
		monitor := RenderMonitor(
			activeTab,
			rightTopPanel.X, rightTopPanel.Y, rightTopPanel.Width, rightTopPanel.Height,
			nil,
		)
		sb.WriteString(monitor)
	}

	// Render terminal panel (right bottom, with tab bar above)
	activeTab := GetActiveTab(m.State)
	terminal := RenderTerminalPanel(
		activeTab,
		rightBottomPanel.X, rightBottomPanel.Y, rightBottomPanel.Width, rightBottomPanel.Height,
		m.State.Tabs,
		m.State.SelectedTabIndex,
	)
	sb.WriteString(terminal)

	return sb.String()
}
