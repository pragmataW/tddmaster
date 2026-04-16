
// Rendering helpers — ANSI primitives used across manager panels.

package manager

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/ansi"
)

// moveTo returns an ANSI escape sequence to move the cursor to (row, col).
// Both row and col are 1-based.
func moveTo(row, col int) string {
	return fmt.Sprintf("\x1b[%d;%dH", row, col)
}

// visibleLength returns the number of visible (non-ANSI) characters in s.
func visibleLength(s string) int {
	return ansi.PrintableRuneWidth(s)
}

// truncateVisible truncates s to at most maxWidth visible characters.
func truncateVisible(s string, maxWidth int) string {
	return lipgloss.NewStyle().MaxWidth(maxWidth).Render(s)
}

// drawBox draws a rounded box border and returns the ANSI string.
// x, y are 1-based. width and height include the border.
func drawBox(x, y, width, height int, title string) string {
	var sb strings.Builder

	// Top border
	sb.WriteString(moveTo(y, x))
	sb.WriteRune('\u256D') // ╭
	topInner := width - 2
	if title != "" {
		titleStr := " " + title + " "
		titleLen := len(titleStr)
		if titleLen > topInner {
			titleLen = topInner
			titleStr = titleStr[:titleLen]
		}
		leftDash := (topInner - titleLen) / 2
		rightDash := topInner - titleLen - leftDash
		sb.WriteString(strings.Repeat("\u2500", leftDash))
		sb.WriteString(styleBold.Render(titleStr))
		sb.WriteString(strings.Repeat("\u2500", rightDash))
	} else {
		sb.WriteString(strings.Repeat("\u2500", topInner))
	}
	sb.WriteRune('\u256E') // ╮

	// Side borders
	for row := 1; row < height-1; row++ {
		sb.WriteString(moveTo(y+row, x))
		sb.WriteRune('\u2502') // │
		sb.WriteString(moveTo(y+row, x+width-1))
		sb.WriteRune('\u2502') // │
	}

	// Bottom border
	sb.WriteString(moveTo(y+height-1, x))
	sb.WriteRune('\u2570') // ╰
	sb.WriteString(strings.Repeat("\u2500", width-2))
	sb.WriteRune('\u256F') // ╯

	return sb.String()
}

// fillBox draws a box with the given lines filling the interior.
func fillBox(x, y, width, height int, title string, lines []string) string {
	var sb strings.Builder
	sb.WriteString(drawBox(x, y, width, height, title))

	innerWidth := width - 2
	innerHeight := height - 2

	for row := 0; row < innerHeight; row++ {
		sb.WriteString(moveTo(y+1+row, x+1))
		if row < len(lines) {
			truncated := truncateVisible(lines[row], innerWidth)
			pad := innerWidth - visibleLength(truncated)
			if pad < 0 {
				pad = 0
			}
			sb.WriteString(truncated)
			sb.WriteString(strings.Repeat(" ", pad))
		} else {
			sb.WriteString(strings.Repeat(" ", innerWidth))
		}
	}

	return sb.String()
}
