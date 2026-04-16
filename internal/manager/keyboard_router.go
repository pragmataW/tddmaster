
// Focus management and key routing.

package manager

// KeyActionType represents the type of action to perform in response to a key press.
type KeyActionType string

const (
	KeyActionNavigate     KeyActionType = "navigate"
	KeyActionSelect       KeyActionType = "select"
	KeyActionNewTab       KeyActionType = "newTab"
	KeyActionCloseTab     KeyActionType = "closeTab"
	KeyActionQuit         KeyActionType = "quit"
	KeyActionToggleFocus  KeyActionType = "toggleFocus"
	KeyActionToggleSpecs  KeyActionType = "toggleSpecs"
	KeyActionToggleMonitor KeyActionType = "toggleMonitor"
	KeyActionPassthrough  KeyActionType = "passthrough"
	KeyActionNone         KeyActionType = "none"
)

// NavigationDirection represents the direction to navigate.
type NavigationDirection string

const (
	DirectionUp   NavigationDirection = "up"
	DirectionDown NavigationDirection = "down"
)

// KeyAction represents an action to perform in response to a key press.
type KeyAction struct {
	Type      KeyActionType
	Direction NavigationDirection // only for KeyActionNavigate
	Data      string              // only for KeyActionPassthrough
}

// RouteKey routes a keypress based on current focus mode.
// Returns the action to perform.
func RouteKey(state ManagerState, key string, ctrl bool) KeyAction {
	// In terminal focus: only intercept Ctrl+C (quit) and Escape (switch to list).
	// Everything else passes through to the PTY for maximum terminal compatibility.
	if state.Focus == FocusTerminal {
		if ctrl && key == "c" {
			return KeyAction{Type: KeyActionQuit}
		}
		if key == "escape" {
			return KeyAction{Type: KeyActionToggleFocus}
		}
		return KeyAction{Type: KeyActionPassthrough, Data: key}
	}

	// List focus: full shortcut set
	if ctrl && key == "c" {
		return KeyAction{Type: KeyActionQuit}
	}
	if ctrl && key == "d" {
		return KeyAction{Type: KeyActionQuit}
	}
	if ctrl && key == "t" {
		return KeyAction{Type: KeyActionNewTab}
	}
	if key == "tab" || key == "escape" {
		return KeyAction{Type: KeyActionToggleFocus}
	}
	if ctrl && key == "e" {
		return KeyAction{Type: KeyActionToggleSpecs}
	}
	if ctrl && key == "w" {
		return KeyAction{Type: KeyActionToggleMonitor}
	}

	if state.Focus == FocusList {
		switch key {
		case "up":
			return KeyAction{Type: KeyActionNavigate, Direction: DirectionUp}
		case "down":
			return KeyAction{Type: KeyActionNavigate, Direction: DirectionDown}
		case "return":
			return KeyAction{Type: KeyActionSelect}
		case "n":
			return KeyAction{Type: KeyActionNewTab}
		case "x":
			return KeyAction{Type: KeyActionCloseTab}
		case "q":
			return KeyAction{Type: KeyActionQuit}
		default:
			return KeyAction{Type: KeyActionNone}
		}
	}

	// Terminal focus: pass everything through
	return KeyAction{Type: KeyActionPassthrough, Data: key}
}

// ToggleFocus toggles the focus between list and terminal.
func ToggleFocus(state ManagerState) ManagerState {
	newState := state
	if state.Focus == FocusList {
		newState.Focus = FocusTerminal
	} else {
		newState.Focus = FocusList
	}
	return newState
}

// NavigateList navigates the list selection, clamped to [0, itemCount).
func NavigateList(state ManagerState, direction NavigationDirection, itemCount int) ManagerState {
	if itemCount == 0 {
		return state
	}
	newState := state
	current := state.SelectedTabIndex
	switch direction {
	case DirectionUp:
		if current <= 0 {
			newState.SelectedTabIndex = 0
		} else {
			newState.SelectedTabIndex = current - 1
		}
	case DirectionDown:
		if current >= itemCount-1 {
			newState.SelectedTabIndex = itemCount - 1
		} else {
			newState.SelectedTabIndex = current + 1
		}
	}
	return newState
}

// =============================================================================
// Mouse routing
// =============================================================================

// MouseActionType represents the type of mouse action.
type MouseActionType string

const (
	MouseActionClickSpec      MouseActionType = "clickSpec"
	MouseActionClickTerminal  MouseActionType = "clickTerminal"
	MouseActionClickMonitor   MouseActionType = "clickMonitor"
	MouseActionScrollSpecs    MouseActionType = "scrollSpecs"
	MouseActionScrollTerminal MouseActionType = "scrollTerminal"
	MouseActionNone           MouseActionType = "none"
)

// MouseAction represents an action to perform in response to a mouse event.
type MouseAction struct {
	Type      MouseActionType
	Index     int                 // only for MouseActionClickSpec
	Direction NavigationDirection // only for scroll actions
}

// Panel represents a rectangular region on the screen (1-based coordinates).
type Panel struct {
	ID     string
	X      int
	Y      int
	Width  int
	Height int
}

// LayoutResult holds the panels returned by the layout engine.
type LayoutResult struct {
	Left        Panel
	RightTop    Panel
	RightBottom Panel
}

// MouseEvent represents a mouse event.
type MouseEvent struct {
	Type      string // "mousedown", "wheel"
	Button    int
	X         int
	Y         int
	Direction string // "up" or "down" for wheel events
}

func isInsidePanel(mx, my int, p Panel) bool {
	return mx >= p.X && mx < p.X+p.Width && my >= p.Y && my < p.Y+p.Height
}

// RouteMouseEvent routes a mouse event based on which panel it hits.
func RouteMouseEvent(
	event MouseEvent,
	panels LayoutResult,
	specItemCount int,
	focus Focus,
) MouseAction {
	// Mouse coords are 0-based from the event, panels are 1-based
	mx := event.X + 1
	my := event.Y + 1

	// Click/scroll in spec list
	if isInsidePanel(mx, my, panels.Left) {
		if event.Type == "wheel" {
			dir := DirectionDown
			if event.Direction == "up" {
				dir = DirectionUp
			}
			return MouseAction{Type: MouseActionScrollSpecs, Direction: dir}
		}
		if event.Type == "mousedown" && event.Button == 0 {
			relRow := event.Y + 1 - panels.Left.Y - 1
			if relRow >= 0 && relRow < specItemCount {
				return MouseAction{Type: MouseActionClickSpec, Index: relRow}
			}
		}
		return MouseAction{Type: MouseActionNone}
	}

	// Click/scroll in terminal panel
	if isInsidePanel(mx, my, panels.RightBottom) {
		if event.Type == "wheel" {
			dir := DirectionDown
			if event.Direction == "up" {
				dir = DirectionUp
			}
			return MouseAction{Type: MouseActionScrollTerminal, Direction: dir}
		}
		if focus != FocusTerminal {
			return MouseAction{Type: MouseActionClickTerminal}
		}
		return MouseAction{Type: MouseActionNone}
	}

	// Click in monitor → focus spec list
	if isInsidePanel(mx, my, panels.RightTop) {
		return MouseAction{Type: MouseActionClickMonitor}
	}

	return MouseAction{Type: MouseActionNone}
}
