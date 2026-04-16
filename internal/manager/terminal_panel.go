
// Right-bottom panel — renders PTY output.
// Tab bar renders one row ABOVE the panel border.

package manager

import (
	"math"
	"strings"
)

// RenderTerminalPanel renders the terminal panel (and tab bar above it) as a string.
func RenderTerminalPanel(
	tab *ManagerTab,
	panelX, panelY, panelW, panelH int,
	allTabs []ManagerTab,
	activeTabIndex int,
) string {
	title := "Terminal"

	// Tab bar sits one row ABOVE the panel border
	tabBarRow := panelY - 1
	tabBarRendered := RenderTabBar(allTabs, activeTabIndex, panelW, tabBarRow, panelX)

	// Draw box border
	border := drawBox(panelX, panelY, panelW, panelH, title)

	// No tabs — centered message inside panel
	if len(allTabs) == 0 {
		msg := styleDim.Render("No tabs \u2014 press n to create one")
		msgLen := visibleLength(msg)
		padLine := strings.Repeat(" ", maxInt(0, panelW-2))

		var sb strings.Builder
		sb.WriteString(tabBarRendered)
		sb.WriteString(border)

		midRow := int(math.Floor(float64(panelH-2) / 2))
		for r := 1; r < panelH-1; r++ {
			sb.WriteString(moveTo(panelY+r, panelX+1))
			if r == midRow {
				leftPad := maxInt(0, (panelW-2-msgLen)/2)
				rightPad := maxInt(0, panelW-2-leftPad-msgLen)
				sb.WriteString(strings.Repeat(" ", leftPad))
				sb.WriteString(msg)
				sb.WriteString(strings.Repeat(" ", rightPad))
			} else {
				sb.WriteString(padLine)
			}
		}
		return sb.String()
	}

	// Tab exists but no output yet — blank interior
	if tab == nil || len(tab.Buffer) == 0 {
		padLine := strings.Repeat(" ", maxInt(0, panelW-2))
		var sb strings.Builder
		sb.WriteString(tabBarRendered)
		sb.WriteString(border)
		for r := 1; r < panelH-1; r++ {
			sb.WriteString(moveTo(panelY+r, panelX+1))
			sb.WriteString(padLine)
		}
		return sb.String()
	}

	// Fallback: raw buffer lines
	innerHeight := panelH - 2
	var visibleLines []string
	if len(tab.Buffer) <= innerHeight {
		visibleLines = tab.Buffer
	} else {
		visibleLines = tab.Buffer[len(tab.Buffer)-innerHeight:]
	}

	return tabBarRendered + fillBox(panelX, panelY, panelW, panelH, title, visibleLines)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
