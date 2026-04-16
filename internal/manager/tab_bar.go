
// Tab bar — renders the horizontal tab strip at the top of the TUI.

package manager

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func abbreviatePhaseStr(phase string) string {
	phaseMap := map[string]string{
		"DISCOVERY":            "DISC",
		"DISCOVERY_REFINEMENT": "REVW",
		"SPEC_PROPOSAL":        "DRFT",
		"SPEC_APPROVED":        "APPR",
		"EXECUTING":            "EXEC",
		"BLOCKED":              "BLKD",
		"COMPLETED":            "DONE",
		"IDLE":                 "IDLE",
	}
	if abbr, ok := phaseMap[phase]; ok {
		return abbr
	}
	if len(phase) > 4 {
		return phase[:4]
	}
	return phase
}

func phaseColorStr(phase *string) string {
	if phase == nil {
		return "dim"
	}
	return phaseColor(phase)
}

// RenderTabBar renders the tab bar row as a string.
// row and col are 1-based screen coordinates.
func RenderTabBar(tabs []ManagerTab, activeIndex, width, row, col int) string {
	var sb strings.Builder

	sb.WriteString(moveTo(row, col))

	if len(tabs) == 0 {
		empty := styleDim.Render(" No tabs \u2014 press n to create one ")
		emptyLen := visibleLength(empty)
		pad := width - emptyLen
		if pad < 0 {
			pad = 0
		}
		sb.WriteString(empty + strings.Repeat(" ", pad))
		return sb.String()
	}

	activeStyle := lipgloss.NewStyle().Reverse(true)

	remaining := width
	for i, tab := range tabs {
		if remaining <= 0 {
			break
		}

		label := "IDLE"
		if tab.Spec != nil {
			label = *tab.Spec
		}

		var badge string
		if tab.Phase != nil {
			abbr := abbreviatePhaseStr(*tab.Phase)
			col := phaseColorStr(tab.Phase)
			badge = " " + colorize(fmt.Sprintf("[%s]", abbr), col)
		}

		closable := " x"
		tabText := fmt.Sprintf(" %s%s%s ", label, badge, closable)

		if i == activeIndex {
			rendered := activeStyle.Render(tabText)
			sb.WriteString(rendered)
			remaining -= visibleLength(tabText)
		} else {
			sb.WriteString(tabText)
			remaining -= visibleLength(tabText)
		}
	}

	if remaining > 0 {
		sb.WriteString(strings.Repeat(" ", remaining))
	}

	return sb.String()
}
