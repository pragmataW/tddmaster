package initform

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

func brandTheme() *huh.Theme {
	t := huh.ThemeCharm()

	var (
		indigo = lipgloss.Color("#6366f1")
		violet = lipgloss.Color("#a78bfa")
		cyan   = lipgloss.Color("#38bdf8")
		teal   = lipgloss.Color("#2dd4bf")
		slate  = lipgloss.Color("#94a3b8")
		red    = lipgloss.Color("#f87171")
	)

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
