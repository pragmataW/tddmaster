
package manager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateInitialState(t *testing.T) {
	s := CreateInitialState()

	assert.Empty(t, s.Tabs, "Tabs should be empty initially")
	assert.Equal(t, -1, s.SelectedTabIndex, "SelectedTabIndex should be -1 initially")
	assert.Equal(t, FocusList, s.Focus, "Focus should be FocusList initially")
	assert.True(t, s.Running, "Running should be true initially")
	assert.True(t, s.SpecsVisible, "SpecsVisible should be true initially")
	assert.True(t, s.MonitorVisible, "MonitorVisible should be true initially")
}

func TestManagerTab_NilSpec(t *testing.T) {
	tab := ManagerTab{
		ID:        "tab-1",
		Mode:      TabModeFree,
		SessionID: "sess-1",
		Buffer:    []string{},
	}

	assert.Nil(t, tab.Spec, "Spec should be nil for free tab")
	assert.Nil(t, tab.Phase, "Phase should be nil for new tab")
}

func TestManagerTab_WithSpec(t *testing.T) {
	specName := "my-spec"
	phase := "EXECUTING"
	tab := ManagerTab{
		ID:        "tab-2",
		Spec:      &specName,
		Mode:      TabModeSpec,
		SessionID: "sess-2",
		Buffer:    []string{},
		Phase:     &phase,
	}

	assert.NotNil(t, tab.Spec)
	assert.Equal(t, "my-spec", *tab.Spec)
	assert.Equal(t, "EXECUTING", *tab.Phase)
	assert.Equal(t, TabModeSpec, tab.Mode)
}

func TestTabModeConstants(t *testing.T) {
	assert.Equal(t, TabMode("spec"), TabModeSpec)
	assert.Equal(t, TabMode("free"), TabModeFree)
}

func TestFocusConstants(t *testing.T) {
	assert.Equal(t, Focus("list"), FocusList)
	assert.Equal(t, Focus("terminal"), FocusTerminal)
}
