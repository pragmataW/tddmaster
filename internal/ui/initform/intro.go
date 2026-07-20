package initform

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pragmataW/tddmaster/internal/ui/theme"
)

var logoLines = []string{
	" ████████╗██████╗ ██████╗ ",
	" ╚══██╔══╝██╔══██╗██╔══██╗",
	"    ██║   ██║  ██║██║  ██║ ",
	"    ██║   ██║  ██║██║  ██║ ",
	"    ██║   ██████╔╝██████╔╝ ",
	"    ╚═╝   ╚═════╝ ╚═════╝  ",
}

const tagline = "TDD-driven spec orchestration for AI-assisted development"

type introFrameMsg struct{}

func introTick() tea.Cmd {
	return tea.Tick(58*time.Millisecond, func(time.Time) tea.Msg {
		return introFrameMsg{}
	})
}

type introModel struct {
	frame   int
	quitting bool
}

func (m introModel) Init() tea.Cmd {
	return introTick()
}

func (m introModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		m.quitting = true
		return m, tea.Quit
	case introFrameMsg:
		m.frame++
		if m.frame >= 78 {
			m.quitting = true
			return m, tea.Quit
		}
		return m, introTick()
	}
	return m, nil
}

func (m introModel) View() string {
	if m.quitting {
		return ""
	}

	var body string

	revealed := m.frame / 2
	for i, line := range logoLines {
		if i > revealed {
			break
		}
		body += gradientLine(line, m.frame*2-i, 18) + "\n"
	}

	subStart := len(logoLines)*2 + 2
	if m.frame > subStart {
		typed := m.frame - subStart
		if typed > len(tagline) {
			typed = len(tagline)
		}
		subStyle := lipgloss.NewStyle().Foreground(theme.ColorSlate).Italic(true)
		caret := ""
		if typed < len(tagline) && m.frame%2 == 0 {
			caret = lipgloss.NewStyle().Foreground(theme.ColorCyan).Render("▌")
		}
		body += "\n  " + subStyle.Render(tagline[:typed]) + caret
	}

	box := theme.BorderStyle.Padding(1, 3).Render(body)

	return "\n" + box + "\n"
}

func PlayIntro() {
	if p := tea.NewProgram(introModel{}); p != nil {
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "intro animation skipped: %v\n", err)
		}
	}
}
