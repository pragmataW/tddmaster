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

	for _, f := range res.FilesWritten {
		name := filepath.Base(f)
		dir := filepath.Dir(f)
		sb.WriteString(fmt.Sprintf("  %s/%s\n", filepath.Base(dir), name))
	}

	for _, id := range res.Adapters {
		sb.WriteString(fmt.Sprintf("  .claude/agents/tddmaster-*.md (%s)\n", id))
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
