package ruleform

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg struct{}

func tick() tea.Cmd {
	return tea.Tick(58*time.Millisecond, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

type phase int

const (
	phaseIntro phase = iota
	phaseForm
	phaseSuccess
	phaseDone
)

type formState struct {
	target   string
	filename string
	body     string
}

type model struct {
	root    string
	phase   phase
	frame   int
	written string
	err     error
	state   *formState
	form    *huh.Form
}

func buildForm(s *formState) *huh.Form {
	targets := Targets()
	opts := make([]huh.Option[string], len(targets))
	for i, t := range targets {
		opts[i] = huh.NewOption(t, t)
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Target").
				Options(opts...).
				Value(&s.target),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Rule filename").
				Value(&s.filename),
		),
		huh.NewGroup(
			huh.NewText().
				Title("Rule body").
				CharLimit(65536).
				Value(&s.body),
		),
	).WithTheme(brandTheme())
}

func newModel(root string) model {
	s := &formState{target: Targets()[0]}
	return model{
		root:  root,
		phase: phaseIntro,
		state: s,
		form:  buildForm(s),
	}
}

func (m model) Init() tea.Cmd {
	return tick()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.phase {
	case phaseIntro:
		switch msg.(type) {
		case tickMsg:
			m.frame++
			if m.frame >= 40 {
				m.phase = phaseForm
				return m, m.form.Init()
			}
			return m, tick()
		case tea.KeyMsg:
			m.phase = phaseForm
			return m, m.form.Init()
		}

	case phaseForm:
		f, cmd := m.form.Update(msg)
		if fm, ok := f.(*huh.Form); ok {
			m.form = fm
		}
		if m.form.State == huh.StateCompleted {
			written, err := WriteRule(m.root, m.state.target, m.state.filename, m.state.body)
			if err != nil {
				m.err = err
				m.phase = phaseDone
				return m, tea.Quit
			}
			m.written = written
			m.phase = phaseSuccess
			m.frame = 0
			return m, tick()
		}
		if m.form.State == huh.StateAborted {
			m.phase = phaseDone
			return m, tea.Quit
		}
		return m, cmd

	case phaseSuccess:
		switch msg.(type) {
		case tickMsg:
			m.frame++
			if m.frame >= 30 {
				m.phase = phaseDone
				return m, tea.Quit
			}
			return m, tick()
		case tea.KeyMsg:
			m.phase = phaseDone
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	switch m.phase {
	case phaseIntro:
		return m.viewIntro()
	case phaseForm:
		return m.viewForm()
	case phaseSuccess:
		return m.viewSuccess()
	default:
		return ""
	}
}

func (m model) viewIntro() string {
	title := "  add rule"
	banner := gradientLine(title, m.frame*2, 18)

	tagline := "  add a rule to guide your AI agents"
	subStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#94a3b8")).Italic(true)

	revealed := m.frame - 5
	if revealed < 0 {
		revealed = 0
	}
	if revealed > len(tagline) {
		revealed = len(tagline)
	}

	body := banner + "\n\n" + subStyle.Render(tagline[:revealed])

	return "\n" + lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6366f1")).
		Padding(1, 3).
		Render(body) + "\n"
}

func (m model) viewForm() string {
	return "\n" + m.form.View()
}

func (m model) viewSuccess() string {
	rel := m.written
	if r, err := filepath.Rel(m.root, m.written); err == nil {
		rel = r
	}

	checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#2dd4bf")).Bold(true)
	pathStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#e2e8f0"))
	header := gradientLine("  rule written", m.frame, 18)

	body := header + "\n\n"
	body += "  " + checkStyle.Render("✓") + " " + pathStyle.Render(rel) + "\n"

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6366f1")).
		Padding(1, 2).
		Render(body)

	return "\n" + box + "\n"
}

func brandTheme() *huh.Theme {
	t := huh.ThemeCharm()

	indigo := lipgloss.Color("#6366f1")
	violet := lipgloss.Color("#a78bfa")
	cyan := lipgloss.Color("#38bdf8")
	teal := lipgloss.Color("#2dd4bf")
	slate := lipgloss.Color("#94a3b8")
	red := lipgloss.Color("#f87171")

	t.Focused.Base = t.Focused.Base.BorderForeground(indigo)
	t.Focused.Title = t.Focused.Title.Foreground(cyan).Bold(true)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(teal).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(slate)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(teal)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(teal)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(teal)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(teal)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Background(indigo).Foreground(lipgloss.Color("#ffffff")).Bold(true)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(slate)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(red)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(red)
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(cyan)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(violet)

	t.Blurred.Title = t.Blurred.Title.Foreground(slate)
	t.Blurred.NoteTitle = t.Blurred.NoteTitle.Foreground(slate)

	return t
}

func Run(root string) error {
	m := newModel(root)
	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil
		}
		return fmt.Errorf("ui: %w", err)
	}
	if fm, ok := result.(model); ok {
		if fm.err != nil {
			if errors.Is(fm.err, huh.ErrUserAborted) {
				return nil
			}
			return fm.err
		}
	}
	return nil
}
