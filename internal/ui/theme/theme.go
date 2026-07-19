package theme

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

const (
	ColorIndigo = lipgloss.Color("#6366f1")
	ColorViolet = lipgloss.Color("#a78bfa")
	ColorCyan   = lipgloss.Color("#38bdf8")
	ColorTeal   = lipgloss.Color("#2dd4bf")
	ColorSlate  = lipgloss.Color("#94a3b8")
	ColorRed    = lipgloss.Color("#f87171")
)

var (
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			BorderForeground(ColorIndigo)

	SuccessStyle = lipgloss.NewStyle().Foreground(ColorTeal)
	ErrorStyle   = lipgloss.NewStyle().Foreground(ColorRed)
	MutedStyle   = lipgloss.NewStyle().Foreground(ColorSlate)
)

func Theme() *huh.Theme {
	t := huh.ThemeCharm()

	t.Focused.Base = t.Focused.Base.BorderForeground(ColorIndigo)
	t.Focused.Title = t.Focused.Title.Foreground(ColorCyan).Bold(true)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(ColorTeal).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(ColorSlate)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(ColorTeal)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(ColorTeal)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(ColorTeal)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(ColorTeal)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Background(ColorIndigo).Foreground(lipgloss.Color("#ffffff")).Bold(true)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(ColorSlate)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(ColorRed)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(ColorRed)
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(ColorCyan)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(ColorViolet)

	t.Blurred.Title = t.Blurred.Title.Foreground(ColorSlate)
	t.Blurred.NoteTitle = t.Blurred.NoteTitle.Foreground(ColorSlate)

	return t
}
