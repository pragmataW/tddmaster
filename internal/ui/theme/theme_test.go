package theme

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestTheme_ac1_ReturnsNonNilHuhTheme(t *testing.T) {
	got := Theme()

	if got == nil {
		t.Fatal("Theme() returned nil, want a non-nil *huh.Theme")
	}
}

func TestTheme_ac1_AppliesBrandFocusedTitleStyle(t *testing.T) {
	got := Theme()

	if got == nil {
		t.Fatal("Theme() returned nil, want a non-nil *huh.Theme")
	}

	if !got.Focused.Title.GetBold() {
		t.Error("Theme().Focused.Title should be bold")
	}

	if got.Focused.Title.GetForeground() != ColorCyan {
		t.Errorf("Theme().Focused.Title foreground = %v, want %v", got.Focused.Title.GetForeground(), ColorCyan)
	}
}

func TestTheme_ac1_AppliesBrandErrorMessageStyle(t *testing.T) {
	got := Theme()

	if got == nil {
		t.Fatal("Theme() returned nil, want a non-nil *huh.Theme")
	}

	if got.Focused.ErrorMessage.GetForeground() != ColorRed {
		t.Errorf("Theme().Focused.ErrorMessage foreground = %v, want %v", got.Focused.ErrorMessage.GetForeground(), ColorRed)
	}
}

func TestPalette_ac1_ExposesSharedColorConstants(t *testing.T) {
	cases := map[string]struct {
		got  lipgloss.Color
		want lipgloss.Color
	}{
		"indigo": {ColorIndigo, lipgloss.Color("#6366f1")},
		"violet": {ColorViolet, lipgloss.Color("#a78bfa")},
		"cyan":   {ColorCyan, lipgloss.Color("#38bdf8")},
		"teal":   {ColorTeal, lipgloss.Color("#2dd4bf")},
		"slate":  {ColorSlate, lipgloss.Color("#94a3b8")},
		"red":    {ColorRed, lipgloss.Color("#f87171")},
	}

	for name, c := range cases {
		if c.got != c.want {
			t.Errorf("Color%s = %v, want %v", name, c.got, c.want)
		}
	}
}

func TestBoxStyles_ac1_ExposesSharedBorderStyle(t *testing.T) {
	if BorderStyle.GetBorderStyle() != lipgloss.RoundedBorder() {
		t.Errorf("BorderStyle border = %v, want RoundedBorder", BorderStyle.GetBorderStyle())
	}

	if got := BorderStyle.GetPaddingTop(); got != 1 {
		t.Errorf("BorderStyle padding top = %d, want 1", got)
	}
	if got := BorderStyle.GetPaddingBottom(); got != 1 {
		t.Errorf("BorderStyle padding bottom = %d, want 1", got)
	}
	if got := BorderStyle.GetPaddingLeft(); got != 2 {
		t.Errorf("BorderStyle padding left = %d, want 2", got)
	}
	if got := BorderStyle.GetPaddingRight(); got != 2 {
		t.Errorf("BorderStyle padding right = %d, want 2", got)
	}
}

func TestBoxStyles_ac1_ExposesSuccessErrorMutedStyles(t *testing.T) {
	if SuccessStyle.GetForeground() != ColorTeal {
		t.Errorf("SuccessStyle foreground = %v, want %v", SuccessStyle.GetForeground(), ColorTeal)
	}

	if ErrorStyle.GetForeground() != ColorRed {
		t.Errorf("ErrorStyle foreground = %v, want %v", ErrorStyle.GetForeground(), ColorRed)
	}

	if MutedStyle.GetForeground() != ColorSlate {
		t.Errorf("MutedStyle foreground = %v, want %v", MutedStyle.GetForeground(), ColorSlate)
	}
}

func TestBoxStyles_ac1_StylesAreDistinctFromEachOther(t *testing.T) {
	if SuccessStyle.GetForeground() == ErrorStyle.GetForeground() {
		t.Error("SuccessStyle and ErrorStyle should not share the same foreground color")
	}
	if SuccessStyle.GetForeground() == MutedStyle.GetForeground() {
		t.Error("SuccessStyle and MutedStyle should not share the same foreground color")
	}
	if ErrorStyle.GetForeground() == MutedStyle.GetForeground() {
		t.Error("ErrorStyle and MutedStyle should not share the same foreground color")
	}
}
