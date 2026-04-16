
package manager

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeTab(id string) ManagerTab {
	return ManagerTab{
		ID:        id,
		Mode:      TabModeFree,
		SessionID: "sess-" + id,
		Buffer:    []string{},
	}
}

// =============================================================================
// CreateTab
// =============================================================================

func TestCreateTab_AddsTab(t *testing.T) {
	s := CreateInitialState()
	tab := makeTab("tab-1")
	result := CreateTab(s, tab)

	assert.Len(t, result.Tabs, 1)
	assert.Equal(t, "tab-1", result.Tabs[0].ID)
}

func TestCreateTab_SelectedIndexIsNewTab(t *testing.T) {
	s := CreateInitialState()
	s = CreateTab(s, makeTab("tab-1"))
	s = CreateTab(s, makeTab("tab-2"))

	// After two creates, selectedTabIndex should be 1 (last created)
	assert.Equal(t, 1, s.SelectedTabIndex)
}

func TestCreateTab_PreservesExistingTabs(t *testing.T) {
	s := CreateInitialState()
	s = CreateTab(s, makeTab("tab-1"))
	s = CreateTab(s, makeTab("tab-2"))

	assert.Len(t, s.Tabs, 2)
	assert.Equal(t, "tab-1", s.Tabs[0].ID)
	assert.Equal(t, "tab-2", s.Tabs[1].ID)
}

// =============================================================================
// CloseTab
// =============================================================================

func TestCloseTab_RemovesTab(t *testing.T) {
	s := CreateInitialState()
	s = CreateTab(s, makeTab("tab-1"))
	s = CreateTab(s, makeTab("tab-2"))

	result := CloseTab(s, "tab-1")
	assert.Len(t, result.Tabs, 1)
	assert.Equal(t, "tab-2", result.Tabs[0].ID)
}

func TestCloseTab_UnknownID_NoChange(t *testing.T) {
	s := CreateInitialState()
	s = CreateTab(s, makeTab("tab-1"))

	result := CloseTab(s, "nonexistent")
	assert.Len(t, result.Tabs, 1)
}

func TestCloseTab_ClampsSelectedIndex(t *testing.T) {
	s := CreateInitialState()
	s = CreateTab(s, makeTab("tab-1"))
	s = CreateTab(s, makeTab("tab-2"))
	s = CreateTab(s, makeTab("tab-3"))
	s.SelectedTabIndex = 2

	result := CloseTab(s, "tab-3")
	assert.Len(t, result.Tabs, 2)
	assert.Equal(t, 1, result.SelectedTabIndex)
}

// =============================================================================
// SwitchTab
// =============================================================================

func TestSwitchTab_ValidIndex(t *testing.T) {
	s := CreateInitialState()
	s = CreateTab(s, makeTab("tab-1"))
	s = CreateTab(s, makeTab("tab-2"))

	result := SwitchTab(s, 0)
	assert.Equal(t, 0, result.SelectedTabIndex)
}

func TestSwitchTab_NegativeIndex_NoChange(t *testing.T) {
	s := CreateInitialState()
	s = CreateTab(s, makeTab("tab-1"))
	s.SelectedTabIndex = 0

	result := SwitchTab(s, -1)
	assert.Equal(t, 0, result.SelectedTabIndex)
}

func TestSwitchTab_OutOfBounds_NoChange(t *testing.T) {
	s := CreateInitialState()
	s = CreateTab(s, makeTab("tab-1"))
	s.SelectedTabIndex = 0

	result := SwitchTab(s, 5)
	assert.Equal(t, 0, result.SelectedTabIndex)
}

// =============================================================================
// AppendToBuffer
// =============================================================================

func TestAppendToBuffer_BasicAppend(t *testing.T) {
	tab := makeTab("tab-1")
	AppendToBuffer(&tab, "line1\nline2")

	assert.Equal(t, []string{"line1", "line2"}, tab.Buffer)
}

func TestAppendToBuffer_CapsAt1000Lines(t *testing.T) {
	tab := makeTab("tab-1")
	// Add 1001 lines
	data := strings.Repeat("x\n", 1001)
	AppendToBuffer(&tab, data)

	assert.LessOrEqual(t, len(tab.Buffer), maxBufferLines+1) // +1 for trailing empty from split
}

func TestAppendToBuffer_PreservesOldLines(t *testing.T) {
	tab := makeTab("tab-1")
	AppendToBuffer(&tab, "first")
	AppendToBuffer(&tab, "second")

	assert.Equal(t, []string{"first", "second"}, tab.Buffer)
}

// =============================================================================
// GetActiveTab
// =============================================================================

func TestGetActiveTab_NoTabs_ReturnsNil(t *testing.T) {
	s := CreateInitialState()
	assert.Nil(t, GetActiveTab(s))
}

func TestGetActiveTab_ValidIndex(t *testing.T) {
	s := CreateInitialState()
	s = CreateTab(s, makeTab("tab-1"))
	s.SelectedTabIndex = 0

	tab := GetActiveTab(s)
	assert.NotNil(t, tab)
	assert.Equal(t, "tab-1", tab.ID)
}

func TestGetActiveTab_NegativeIndex_ReturnsNil(t *testing.T) {
	s := CreateInitialState()
	s = CreateTab(s, makeTab("tab-1"))
	s.SelectedTabIndex = -1

	assert.Nil(t, GetActiveTab(s))
}

func TestGetActiveTab_IndexOutOfBounds_ReturnsNil(t *testing.T) {
	s := CreateInitialState()
	s = CreateTab(s, makeTab("tab-1"))
	s.SelectedTabIndex = 5

	assert.Nil(t, GetActiveTab(s))
}
