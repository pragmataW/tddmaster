package initform

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pragmataW/tddmaster/internal/scaffold"
	"github.com/pragmataW/tddmaster/internal/ui/theme"
)

var (
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	boldStyle   = lipgloss.NewStyle().Bold(true)
)

func RenderSummary(res scaffold.Result, command string) string {
	var sb strings.Builder

	sb.WriteString(boldStyle.Render("Files written:"))
	sb.WriteString("\n")

	for _, line := range summaryFileLines(res.FilesWritten) {
		sb.WriteString("  " + line + "\n")
	}

	if len(res.Warnings) > 0 {
		sb.WriteString("\n")
		sb.WriteString(yellowStyle.Render("Warnings:"))
		sb.WriteString("\n")
		for _, w := range res.Warnings {
			sb.WriteString(yellowStyle.Render("  ! " + w))
			sb.WriteString("\n")
		}
	}

	if command != "" {
		sb.WriteString("\n")
		sb.WriteString(greenStyle.Render(fmt.Sprintf("Next step: %s start <slug>", command)))
	}

	return theme.BorderStyle.Render(sb.String())
}

// summaryFileLines turns the raw written-file list into display lines. Absolute
// paths are shortened to their last two segments; a directory that received more
// than two agent files collapses into a single glob so the summary stays short
// without dropping anything the way a hardcoded adapter line used to.
func summaryFileLines(files []string) []string {
	order := make([]string, 0, len(files))
	byDir := make(map[string][]string, len(files))
	for _, f := range files {
		display := f
		if filepath.IsAbs(f) {
			display = filepath.Join(filepath.Base(filepath.Dir(f)), filepath.Base(f))
		}
		dir := filepath.Dir(display)
		if _, seen := byDir[dir]; !seen {
			order = append(order, dir)
		}
		byDir[dir] = append(byDir[dir], display)
	}

	var lines []string
	for _, dir := range order {
		entries := byDir[dir]
		if len(entries) > 2 && dir != "." {
			lines = append(lines, filepath.Join(dir, "tddmaster-*"+filepath.Ext(entries[0])))
			continue
		}
		lines = append(lines, entries...)
	}
	return lines
}
