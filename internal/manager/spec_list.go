
// Left panel — renders spec list.

package manager

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// SpecInfo holds display info for a spec in the list.
type SpecInfo struct {
	Name             string
	Phase            *string // nil if no phase
	HasActiveSession bool
}

// ListItem represents an item in the spec list for rendering.
type ListItem struct {
	Label      string
	Badge      string
	BadgeColor string
	Active     bool
	Dimmed     bool
	Selectable bool
}

var (
	styleGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleRed    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styleCyan   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styleDim    = lipgloss.NewStyle().Faint(true)
	styleBold   = lipgloss.NewStyle().Bold(true)
)

func abbreviatePhase(phase *string) string {
	if phase == nil {
		return "\u2014"
	}
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
	if abbr, ok := phaseMap[*phase]; ok {
		return abbr
	}
	if len(*phase) > 4 {
		return (*phase)[:4]
	}
	return *phase
}

func phaseColor(phase *string) string {
	if phase == nil {
		return "dim"
	}
	switch *phase {
	case "EXECUTING":
		return "green"
	case "DISCOVERY", "DISCOVERY_REFINEMENT":
		return "cyan"
	case "BLOCKED":
		return "red"
	case "SPEC_PROPOSAL", "SPEC_APPROVED":
		return "yellow"
	case "COMPLETED":
		return "dim"
	default:
		return "dim"
	}
}

func colorize(text, color string) string {
	switch color {
	case "green":
		return styleGreen.Render(text)
	case "yellow":
		return styleYellow.Render(text)
	case "red":
		return styleRed.Render(text)
	case "cyan":
		return styleCyan.Render(text)
	case "dim":
		return styleDim.Render(text)
	default:
		return text
	}
}

// BuildListItems converts specs and tabs into renderable list items.
func BuildListItems(specs []SpecInfo, tabs []ManagerTab) []ListItem {
	tabSpecs := make(map[string]bool)
	for _, t := range tabs {
		if t.Spec != nil {
			tabSpecs[*t.Spec] = true
		}
	}

	items := make([]ListItem, 0, len(specs))
	for _, s := range specs {
		phase := s.Phase
		items = append(items, ListItem{
			Label:      s.Name,
			Badge:      abbreviatePhase(phase),
			BadgeColor: phaseColor(phase),
			Active:     tabSpecs[s.Name],
			Dimmed:     phase != nil && *phase == "COMPLETED",
			Selectable: true,
		})
	}

	if len(items) == 0 {
		items = append(items, ListItem{
			Label:      "No specs yet",
			Dimmed:     true,
			Selectable: false,
		})
	}

	return items
}

// RenderSpecList renders the spec list panel as a string.
// panelX, panelY, panelW, panelH are 1-based screen coordinates.
func RenderSpecList(
	specs []SpecInfo,
	tabs []ManagerTab,
	selectedIndex int,
	panelX, panelY, panelW, panelH int,
) string {
	items := BuildListItems(specs, tabs)

	innerWidth := panelW - 2
	viewportHeight := panelH - 2

	clampedIndex := selectedIndex
	if clampedIndex < 0 {
		clampedIndex = 0
	}
	if clampedIndex >= len(items) {
		clampedIndex = len(items) - 1
	}

	// Compute scroll offset to keep selected item visible
	scrollOffset := 0
	if clampedIndex >= viewportHeight {
		scrollOffset = clampedIndex - viewportHeight + 1
	}

	var sb strings.Builder

	// Draw border
	sb.WriteString(drawBox(panelX, panelY, panelW, panelH, "Specs"))

	// Render visible items
	for vi := 0; vi < viewportHeight; vi++ {
		idx := scrollOffset + vi
		row := panelY + 1 + vi

		sb.WriteString(moveTo(row, panelX+1))

		if idx >= len(items) {
			sb.WriteString(strings.Repeat(" ", innerWidth))
			continue
		}

		item := items[idx]
		isSelectable := item.Selectable
		selected := idx == selectedIndex && isSelectable

		var bullet string
		if item.Active {
			bullet = styleGreen.Render("\u25CF")
		} else if selected {
			bullet = "\u25B8"
		} else {
			bullet = " "
		}

		var badge string
		if item.Badge != "" {
			badge = " " + colorize(fmt.Sprintf("[%s]", item.Badge), item.BadgeColor)
		}

		var label string
		if item.Dimmed {
			label = styleDim.Render(item.Label)
		} else {
			label = item.Label
		}

		line := fmt.Sprintf(" %s %s%s", bullet, label, badge)
		truncated := truncateVisible(line, innerWidth)
		pad := innerWidth - visibleLength(truncated)
		if pad < 0 {
			pad = 0
		}

		if selected {
			full := truncated + strings.Repeat(" ", pad)
			sb.WriteString(lipgloss.NewStyle().Reverse(true).Render(full))
		} else {
			sb.WriteString(truncated + strings.Repeat(" ", pad))
		}
	}

	return sb.String()
}
