package ruleform

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type rgb struct{ r, g, b int }

var brandAnchors = []rgb{
	{124, 58, 237},
	{99, 102, 241},
	{56, 189, 248},
	{45, 212, 191},
	{56, 189, 248},
	{99, 102, 241},
}

func lerp(a, b, t float64) int {
	return int(a + (b-a)*t)
}

func gradientAt(pos float64) rgb {
	if pos < 0 {
		pos = 0
	}
	if pos > 1 {
		pos = 1
	}
	seg := pos * float64(len(brandAnchors)-1)
	i := int(seg)
	if i >= len(brandAnchors)-1 {
		return brandAnchors[len(brandAnchors)-1]
	}
	t := seg - float64(i)
	a, b := brandAnchors[i], brandAnchors[i+1]
	return rgb{
		r: lerp(float64(a.r), float64(b.r), t),
		g: lerp(float64(a.g), float64(b.g), t),
		b: lerp(float64(a.b), float64(b.b), t),
	}
}

func (c rgb) hex() string {
	return fmt.Sprintf("#%02x%02x%02x", c.r, c.g, c.b)
}

func (c rgb) color() lipgloss.Color {
	return lipgloss.Color(c.hex())
}

func gradientLine(text string, offset, span int) string {
	runes := []rune(text)
	if span <= 0 {
		span = 1
	}
	var out strings.Builder
	for i, ch := range runes {
		if ch == ' ' {
			out.WriteByte(' ')
			continue
		}
		pos := float64(((i+offset)%span)+span) / float64(span)
		if pos > 1 {
			pos -= 1
		}
		st := lipgloss.NewStyle().Foreground(gradientAt(pos).color()).Bold(true)
		out.WriteString(st.Render(string(ch)))
	}
	return out.String()
}
