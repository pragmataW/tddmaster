package ruleform

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestLerp_Midpoint(t *testing.T) {
	got := lerp(0, 10, 0.5)
	if got != 5 {
		t.Errorf("lerp(0,10,0.5) = %d, want 5", got)
	}
}

func TestLerp_Zero(t *testing.T) {
	got := lerp(0, 10, 0)
	if got != 0 {
		t.Errorf("lerp(0,10,0) = %d, want 0", got)
	}
}

func TestLerp_One(t *testing.T) {
	got := lerp(0, 10, 1)
	if got != 10 {
		t.Errorf("lerp(0,10,1) = %d, want 10", got)
	}
}

func TestGradientAt_ZeroReturnsFirstAnchor(t *testing.T) {
	got := gradientAt(0)
	want := brandAnchors[0]
	if got != want {
		t.Errorf("gradientAt(0) = %v, want %v", got, want)
	}
}

func TestGradientAt_OneReturnsLastAnchor(t *testing.T) {
	got := gradientAt(1)
	want := brandAnchors[len(brandAnchors)-1]
	if got != want {
		t.Errorf("gradientAt(1) = %v, want %v", got, want)
	}
}

func TestGradientAt_NegativeClampsToFirst(t *testing.T) {
	got := gradientAt(-0.5)
	want := brandAnchors[0]
	if got != want {
		t.Errorf("gradientAt(-0.5) = %v, want %v", got, want)
	}
}

func TestGradientAt_GreaterThanOneClampsToLast(t *testing.T) {
	got := gradientAt(1.5)
	want := brandAnchors[len(brandAnchors)-1]
	if got != want {
		t.Errorf("gradientAt(1.5) = %v, want %v", got, want)
	}
}

func TestGradientAt_MidValueFieldsInRange(t *testing.T) {
	got := gradientAt(0.5)
	if got.r < 0 || got.r > 255 {
		t.Errorf("gradientAt(0.5).r = %d, want 0..255", got.r)
	}
	if got.g < 0 || got.g > 255 {
		t.Errorf("gradientAt(0.5).g = %d, want 0..255", got.g)
	}
	if got.b < 0 || got.b > 255 {
		t.Errorf("gradientAt(0.5).b = %d, want 0..255", got.b)
	}
}

func TestRGB_Hex_KnownValue(t *testing.T) {
	c := rgb{124, 58, 237}
	got := c.hex()
	want := "#7c3aed"
	if got != want {
		t.Errorf("rgb{124,58,237}.hex() = %q, want %q", got, want)
	}
}

func TestRGB_Hex_ZeroValue(t *testing.T) {
	c := rgb{0, 0, 0}
	got := c.hex()
	if got != "#000000" {
		t.Errorf("rgb{0,0,0}.hex() = %q, want \"#000000\"", got)
	}
}

func TestRGB_Hex_MaxValue(t *testing.T) {
	c := rgb{255, 255, 255}
	got := c.hex()
	if got != "#ffffff" {
		t.Errorf("rgb{255,255,255}.hex() = %q, want \"#ffffff\"", got)
	}
}

func TestRGB_Color_EqualsHex(t *testing.T) {
	c := rgb{124, 58, 237}
	got := c.color()
	want := lipgloss.Color(c.hex())
	if got != want {
		t.Errorf("rgb.color() = %q, want %q", got, want)
	}
}

func TestGradientLine_NonEmpty(t *testing.T) {
	got := gradientLine("hello", 0, 18)
	if len(got) == 0 {
		t.Error("gradientLine(\"hello\", 0, 18) returned empty string")
	}
}

func TestGradientLine_PreservesSpaceCount(t *testing.T) {
	text := "hi world"
	got := gradientLine(text, 0, 18)
	spaceCount := strings.Count(got, " ")
	wantSpaces := strings.Count(text, " ")
	if spaceCount != wantSpaces {
		t.Errorf("gradientLine space count = %d, want %d", spaceCount, wantSpaces)
	}
}

func TestGradientLine_SpanZeroNoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("gradientLine panicked with span=0: %v", r)
		}
	}()
	gradientLine("hello", 0, 0)
}

func TestGradientLine_SpanNegativeNoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("gradientLine panicked with span=-1: %v", r)
		}
	}()
	gradientLine("hello", 0, -1)
}

func TestGradientLine_SpaceOnlyInput(t *testing.T) {
	got := gradientLine("   ", 0, 10)
	if strings.Count(got, " ") != 3 {
		t.Errorf("gradientLine with spaces only: got %d spaces, want 3", strings.Count(got, " "))
	}
}

func TestGradientLine_MixedSpaceAndChar(t *testing.T) {
	got := gradientLine(" a b", 5, 20)
	if len(got) == 0 {
		t.Error("gradientLine with mixed space/char returned empty")
	}
	if strings.Count(got, " ") != 2 {
		t.Errorf("gradientLine mixed: got %d spaces, want 2", strings.Count(got, " "))
	}
}

func TestGradientLine_LargeOffset(t *testing.T) {
	got := gradientLine("abc", 9999, 18)
	if len(got) == 0 {
		t.Error("gradientLine with large offset returned empty")
	}
}
