
// Tab lifecycle — create, switch, close tabs. Buffer management (last 1000 lines).

package manager

import "strings"

const maxBufferLines = 1000

// CreateTab adds a new tab to the state and selects it.
func CreateTab(state ManagerState, tab ManagerTab) ManagerState {
	newState := state
	newState.Tabs = append(append([]ManagerTab{}, state.Tabs...), tab)
	newState.SelectedTabIndex = len(state.Tabs)
	return newState
}

// CloseTab removes the tab with the given ID from the state.
func CloseTab(state ManagerState, tabID string) ManagerState {
	idx := -1
	for i, t := range state.Tabs {
		if t.ID == tabID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return state
	}

	newTabs := make([]ManagerTab, 0, len(state.Tabs)-1)
	for i, t := range state.Tabs {
		if i != idx {
			newTabs = append(newTabs, t)
		}
	}

	newState := state
	newState.Tabs = newTabs
	selectedIdx := state.SelectedTabIndex
	if selectedIdx >= len(newTabs) {
		selectedIdx = len(newTabs) - 1
	}
	newState.SelectedTabIndex = selectedIdx
	return newState
}

// SwitchTab selects the tab at the given index.
func SwitchTab(state ManagerState, index int) ManagerState {
	if index < 0 || index >= len(state.Tabs) {
		return state
	}
	newState := state
	newState.SelectedTabIndex = index
	return newState
}

// AppendToBuffer appends data to a tab's buffer, capping at maxBufferLines.
func AppendToBuffer(tab *ManagerTab, data string) {
	lines := strings.Split(data, "\n")
	tab.Buffer = append(tab.Buffer, lines...)
	if len(tab.Buffer) > maxBufferLines {
		tab.Buffer = tab.Buffer[len(tab.Buffer)-maxBufferLines:]
	}
}

// GetActiveTab returns the currently active tab, or nil if none.
func GetActiveTab(state ManagerState) *ManagerTab {
	if state.SelectedTabIndex < 0 || state.SelectedTabIndex >= len(state.Tabs) {
		return nil
	}
	tab := state.Tabs[state.SelectedTabIndex]
	return &tab
}
