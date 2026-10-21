// Package style derives shared Lip Gloss styles from semantic tokens.
package style

import (
	"charm.land/lipgloss/v2"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/theme"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
	"strings"
	"unicode"
)

// Sheet holds a palette and rendering capabilities, without global styles.
type Sheet struct {
	Palette      tokens.Palette
	Plain, ASCII bool
}

// Text renders a flat semantic foreground color.
func (s Sheet) Text(text, role string) string {
	if s.Plain {
		return text
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(role)).Render(text)
}

// Muted renders secondary text with the palette's readable muted role.
func (s Sheet) Muted(text string) string { return s.Text(text, s.Palette.Muted) }

// Heading renders a compact title with bold primary text.
func (s Sheet) Heading(text string) string {
	if s.Plain {
		return text
	}
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(s.Palette.Text)).Render(text)
}

// Clean removes all terminal controls and line separators from untrusted data.
func Clean(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			return ' '
		}
		return r
	}, value)
}

// Fit clips trusted styled content to an exact terminal rectangle.
func Fit(text string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	lines := strings.Split(text, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for n := range lines {
		lines[n] = lipgloss.NewStyle().MaxWidth(width).Render(lines[n])
		lines[n] += strings.Repeat(" ", max(0, width-lipgloss.Width(lines[n])))
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines, "\n")
}

// Panel renders bounded content with one shared spacing and border convention.
func (s Sheet) Panel(title, body string, width, height int, focus bool) string {
	border := lipgloss.RoundedBorder()
	if s.ASCII {
		border = lipgloss.NormalBorder()
		border.Top = "-"
		border.Bottom = "-"
		border.Left = "|"
		border.Right = "|"
		border.TopLeft = "+"
		border.TopRight = "+"
		border.BottomLeft = "+"
		border.BottomRight = "+"
	}
	inner := max(0, width-tokens.PanelFrame)
	content := Fit(s.Heading(Clean(title))+"\n"+body, max(0, inner-tokens.InsetX*2), max(0, height-tokens.PanelFrame))
	st := lipgloss.NewStyle().Border(border).Padding(0, tokens.InsetX)
	if !s.Plain {
		role := s.Palette.Border
		if focus {
			role = s.Palette.BorderFocus
		}
		st = st.BorderForeground(lipgloss.Color(role)).Background(lipgloss.Color(s.Palette.Surface)).Foreground(lipgloss.Color(s.Palette.Text))
	}
	return Fit(st.Render(content), width, height)
}

// Gradient is reserved for the wordmark, tab underline and progress fills.
func (s Sheet) Gradient(text string) string {
	if s.Plain {
		return text
	}
	runes := []rune(text)
	var out strings.Builder
	for n, r := range runes {
		t := float64(n) / float64(max(1, len(runes)-1))
		out.WriteString(s.Text(string(r), theme.Blend(s.Palette.Accent, s.Palette.Accent2, t)))
	}
	return out.String()
}
