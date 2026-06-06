package initform

import (
	"fmt"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pragmataW/tddmaster/internal/scaffold"
)

type outroLine struct {
	icon  string
	text  string
	color lipgloss.Color
}

type outroRevealMsg struct{}

func outroTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return outroRevealMsg{}
	})
}

type outroModel struct {
	lines    []outroLine
	shown    int
	frame    int
	command  string
	quitting bool
}

func buildOutroLines(res scaffold.Result) []outroLine {
	var lines []outroLine
	for _, f := range res.FilesWritten {
		name := filepath.Base(f)
		dir := filepath.Base(filepath.Dir(f))
		lines = append(lines, outroLine{icon: "✓", text: dir + "/" + name, color: lipgloss.Color("#2dd4bf")})
	}
	for _, id := range res.Adapters {
		lines = append(lines, outroLine{icon: "✓", text: fmt.Sprintf(".claude/agents/tddmaster-*.md (%s)", id), color: lipgloss.Color("#2dd4bf")})
	}
	for _, w := range res.Warnings {
		lines = append(lines, outroLine{icon: "!", text: w, color: lipgloss.Color("#fbbf24")})
	}
	return lines
}

func (m outroModel) Init() tea.Cmd {
	return outroTick()
}

func (m outroModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		m.shown = len(m.lines)
		m.quitting = true
		return m, tea.Quit
	case outroRevealMsg:
		m.frame++
		if m.shown < len(m.lines) {
			m.shown++
			return m, outroTick()
		}
		if m.frame < m.shown+25 {
			return m, outroTick()
		}
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m outroModel) View() string {
	if m.quitting {
		return ""
	}

	header := gradientLine("  files written", m.frame, 18)

	var body string
	body += header + "\n\n"

	for i := 0; i < m.shown; i++ {
		ln := m.lines[i]
		iconSt := lipgloss.NewStyle().Foreground(ln.color).Bold(true)
		textSt := lipgloss.NewStyle().Foreground(lipgloss.Color("#e2e8f0"))
		body += "  " + iconSt.Render(ln.icon) + " " + textSt.Render(ln.text) + "\n"
	}

	if m.shown >= len(m.lines) && m.command != "" {
		next := lipgloss.NewStyle().Foreground(lipgloss.Color("#2dd4bf")).Bold(true).
			Render(fmt.Sprintf("→ next: %s spec new \"description\"", m.command))
		body += "\n  " + next + "\n"
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6366f1")).
		Padding(1, 2).
		Render(body)

	return "\n" + box + "\n"
}

func PlayOutro(res scaffold.Result, command string) {
	m := outroModel{lines: buildOutroLines(res), command: command}
	if len(m.lines) == 0 {
		fmt.Println(RenderSummary(res, command))
		return
	}
	if p := tea.NewProgram(m); p != nil {
		_, _ = p.Run()
	}
}
