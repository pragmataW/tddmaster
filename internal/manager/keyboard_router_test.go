
package manager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func baseState() ManagerState {
	return CreateInitialState()
}

func terminalState() ManagerState {
	s := CreateInitialState()
	s.Focus = FocusTerminal
	return s
}

// =============================================================================
// RouteKey — list focus
// =============================================================================

func TestRouteKey_ListFocus_CtrlC_Quit(t *testing.T) {
	action := RouteKey(baseState(), "c", true)
	assert.Equal(t, KeyActionQuit, action.Type)
}

func TestRouteKey_ListFocus_CtrlD_Quit(t *testing.T) {
	action := RouteKey(baseState(), "d", true)
	assert.Equal(t, KeyActionQuit, action.Type)
}

func TestRouteKey_ListFocus_CtrlT_NewTab(t *testing.T) {
	action := RouteKey(baseState(), "t", true)
	assert.Equal(t, KeyActionNewTab, action.Type)
}

func TestRouteKey_ListFocus_Tab_ToggleFocus(t *testing.T) {
	action := RouteKey(baseState(), "tab", false)
	assert.Equal(t, KeyActionToggleFocus, action.Type)
}

func TestRouteKey_ListFocus_Escape_ToggleFocus(t *testing.T) {
	action := RouteKey(baseState(), "escape", false)
	assert.Equal(t, KeyActionToggleFocus, action.Type)
}

func TestRouteKey_ListFocus_CtrlE_ToggleSpecs(t *testing.T) {
	action := RouteKey(baseState(), "e", true)
	assert.Equal(t, KeyActionToggleSpecs, action.Type)
}

func TestRouteKey_ListFocus_CtrlW_ToggleMonitor(t *testing.T) {
	action := RouteKey(baseState(), "w", true)
	assert.Equal(t, KeyActionToggleMonitor, action.Type)
}

func TestRouteKey_ListFocus_Up_Navigate(t *testing.T) {
	action := RouteKey(baseState(), "up", false)
	assert.Equal(t, KeyActionNavigate, action.Type)
	assert.Equal(t, DirectionUp, action.Direction)
}

func TestRouteKey_ListFocus_Down_Navigate(t *testing.T) {
	action := RouteKey(baseState(), "down", false)
	assert.Equal(t, KeyActionNavigate, action.Type)
	assert.Equal(t, DirectionDown, action.Direction)
}

func TestRouteKey_ListFocus_Return_Select(t *testing.T) {
	action := RouteKey(baseState(), "return", false)
	assert.Equal(t, KeyActionSelect, action.Type)
}

func TestRouteKey_ListFocus_N_NewTab(t *testing.T) {
	action := RouteKey(baseState(), "n", false)
	assert.Equal(t, KeyActionNewTab, action.Type)
}

func TestRouteKey_ListFocus_X_CloseTab(t *testing.T) {
	action := RouteKey(baseState(), "x", false)
	assert.Equal(t, KeyActionCloseTab, action.Type)
}

func TestRouteKey_ListFocus_Q_Quit(t *testing.T) {
	action := RouteKey(baseState(), "q", false)
	assert.Equal(t, KeyActionQuit, action.Type)
}

func TestRouteKey_ListFocus_Unknown_None(t *testing.T) {
	action := RouteKey(baseState(), "f", false)
	assert.Equal(t, KeyActionNone, action.Type)
}

// =============================================================================
// RouteKey — terminal focus
// =============================================================================

func TestRouteKey_TerminalFocus_CtrlC_Quit(t *testing.T) {
	action := RouteKey(terminalState(), "c", true)
	assert.Equal(t, KeyActionQuit, action.Type)
}

func TestRouteKey_TerminalFocus_Escape_ToggleFocus(t *testing.T) {
	action := RouteKey(terminalState(), "escape", false)
	assert.Equal(t, KeyActionToggleFocus, action.Type)
}

func TestRouteKey_TerminalFocus_AnyKey_Passthrough(t *testing.T) {
	action := RouteKey(terminalState(), "a", false)
	assert.Equal(t, KeyActionPassthrough, action.Type)
	assert.Equal(t, "a", action.Data)
}

func TestRouteKey_TerminalFocus_UpKey_Passthrough(t *testing.T) {
	action := RouteKey(terminalState(), "up", false)
	assert.Equal(t, KeyActionPassthrough, action.Type)
	assert.Equal(t, "up", action.Data)
}

// =============================================================================
// ToggleFocus
// =============================================================================

func TestToggleFocus_ListToTerminal(t *testing.T) {
	s := CreateInitialState()
	s.Focus = FocusList
	result := ToggleFocus(s)
	assert.Equal(t, FocusTerminal, result.Focus)
}

func TestToggleFocus_TerminalToList(t *testing.T) {
	s := CreateInitialState()
	s.Focus = FocusTerminal
	result := ToggleFocus(s)
	assert.Equal(t, FocusList, result.Focus)
}

// =============================================================================
// NavigateList
// =============================================================================

func TestNavigateList_EmptyList(t *testing.T) {
	s := CreateInitialState()
	result := NavigateList(s, DirectionDown, 0)
	assert.Equal(t, s.SelectedTabIndex, result.SelectedTabIndex)
}

func TestNavigateList_DownFromStart(t *testing.T) {
	s := CreateInitialState()
	s.SelectedTabIndex = 0
	result := NavigateList(s, DirectionDown, 3)
	assert.Equal(t, 1, result.SelectedTabIndex)
}

func TestNavigateList_UpFromMiddle(t *testing.T) {
	s := CreateInitialState()
	s.SelectedTabIndex = 2
	result := NavigateList(s, DirectionUp, 5)
	assert.Equal(t, 1, result.SelectedTabIndex)
}

func TestNavigateList_UpAtStart_Clamps(t *testing.T) {
	s := CreateInitialState()
	s.SelectedTabIndex = 0
	result := NavigateList(s, DirectionUp, 3)
	assert.Equal(t, 0, result.SelectedTabIndex)
}

func TestNavigateList_DownAtEnd_Clamps(t *testing.T) {
	s := CreateInitialState()
	s.SelectedTabIndex = 2
	result := NavigateList(s, DirectionDown, 3)
	assert.Equal(t, 2, result.SelectedTabIndex)
}

// =============================================================================
// Mouse routing
// =============================================================================

func TestIsInsidePanel_Inside(t *testing.T) {
	p := Panel{X: 1, Y: 1, Width: 10, Height: 5}
	assert.True(t, isInsidePanel(5, 3, p))
}

func TestIsInsidePanel_Outside(t *testing.T) {
	p := Panel{X: 1, Y: 1, Width: 10, Height: 5}
	assert.False(t, isInsidePanel(15, 3, p))
}

func TestIsInsidePanel_OnBorder(t *testing.T) {
	p := Panel{X: 1, Y: 1, Width: 10, Height: 5}
	// x < p.x + p.width means x=10 is NOT inside (10 < 11 is true, so it IS inside)
	assert.True(t, isInsidePanel(10, 3, p))
	// x = p.x + p.width means x=11 is NOT inside
	assert.False(t, isInsidePanel(11, 3, p))
}

func makeLayout() LayoutResult {
	return LayoutResult{
		Left:        Panel{ID: "left", X: 1, Y: 1, Width: 20, Height: 30},
		RightTop:    Panel{ID: "rightTop", X: 21, Y: 1, Width: 40, Height: 10},
		RightBottom: Panel{ID: "rightBottom", X: 21, Y: 11, Width: 40, Height: 20},
	}
}

func TestRouteMouseEvent_ScrollSpecs_Up(t *testing.T) {
	layout := makeLayout()
	// Panel.Left starts at X=1, so mx = event.X + 1 = 1 means event.X = 0
	event := MouseEvent{Type: "wheel", X: 5, Y: 10, Direction: "up"}
	action := RouteMouseEvent(event, layout, 5, FocusList)
	assert.Equal(t, MouseActionScrollSpecs, action.Type)
	assert.Equal(t, DirectionUp, action.Direction)
}

func TestRouteMouseEvent_ScrollSpecs_Down(t *testing.T) {
	layout := makeLayout()
	event := MouseEvent{Type: "wheel", X: 5, Y: 10, Direction: "down"}
	action := RouteMouseEvent(event, layout, 5, FocusList)
	assert.Equal(t, MouseActionScrollSpecs, action.Type)
	assert.Equal(t, DirectionDown, action.Direction)
}

func TestRouteMouseEvent_ClickSpec(t *testing.T) {
	layout := makeLayout()
	// Left panel Y=1, border at row 1, interior starts at Y=2
	// event.Y + 1 - panels.Left.Y - 1 = relRow
	// event.Y = 1 → my = 2, inside left panel (y=1, height=30)
	// relRow = 1+1 - 1 - 1 = 0
	event := MouseEvent{Type: "mousedown", Button: 0, X: 5, Y: 1}
	action := RouteMouseEvent(event, layout, 5, FocusList)
	assert.Equal(t, MouseActionClickSpec, action.Type)
	assert.Equal(t, 0, action.Index)
}

func TestRouteMouseEvent_ClickTerminal(t *testing.T) {
	layout := makeLayout()
	// RightBottom starts at X=21, so mx = event.X + 1 must be >= 21
	// event.X = 20 → mx = 21. Panel Y=11, height=20, so my must be 11..30
	// event.Y = 11 → my = 12
	event := MouseEvent{Type: "mousedown", Button: 0, X: 20, Y: 11}
	action := RouteMouseEvent(event, layout, 5, FocusList)
	assert.Equal(t, MouseActionClickTerminal, action.Type)
}

func TestRouteMouseEvent_ClickMonitor(t *testing.T) {
	layout := makeLayout()
	// RightTop starts at X=21, Y=1, Width=40, Height=10
	// event.X = 30 → mx = 31 (inside rightTop)
	// event.Y = 5 → my = 6 (inside rightTop Y=1..10)
	event := MouseEvent{Type: "mousedown", Button: 0, X: 30, Y: 5}
	action := RouteMouseEvent(event, layout, 5, FocusList)
	assert.Equal(t, MouseActionClickMonitor, action.Type)
}

func TestRouteMouseEvent_OutsideAllPanels_None(t *testing.T) {
	layout := makeLayout()
	event := MouseEvent{Type: "mousedown", Button: 0, X: 100, Y: 100}
	action := RouteMouseEvent(event, layout, 5, FocusList)
	assert.Equal(t, MouseActionNone, action.Type)
}
