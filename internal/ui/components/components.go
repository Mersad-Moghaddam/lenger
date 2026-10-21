// Package components renders shared, token-driven terminal widgets.
package components

import (
	"charm.land/lipgloss/v2"
	"fmt"
	"github.com/Mersad-Moghaddam/lenger/internal/core/jobs"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/chart"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
	"strings"
	"time"
)

// TabNames returns navigation labels without shared mutable state.
func TabNames() []string {
	return []string{"Dashboard", "Images", "Containers", "Projects", "Volumes", "Networks", "System"}
}

// Tabs renders compact numbered navigation with a signature active underline.
func Tabs(s style.Sheet, active, width int) string {
	var labels, underline strings.Builder
	for n, name := range TabNames() {
		label := fmt.Sprintf("%d %s", n+1, name)
		if width < tokens.DetailWidth {
			label = fmt.Sprintf("%d %.3s", n+1, name)
		}
		label += "  "
		if n == active {
			labels.WriteString(s.Heading(label))
			line := "━"
			if s.ASCII {
				line = "="
			}
			underline.WriteString(s.Gradient(strings.Repeat(line, lipgloss.Width(label)-tokens.Gap)) + strings.Repeat(" ", tokens.Gap))
		} else {
			labels.WriteString(s.Muted(label))
			underline.WriteString(strings.Repeat(" ", lipgloss.Width(label)))
		}
	}
	return style.Fit(labels.String()+"\n"+underline.String(), width, tokens.InsetY*2)
}

// Pill combines an explicit status word and a non-color status indicator.
func Pill(s style.Sheet, state string) string {
	g := tokens.Icons(s.ASCII)
	icon, role := g.Stopped, s.Palette.Muted
	switch state {
	case "running", "succeeded", "healthy":
		icon, role = g.Running, s.Palette.Success
	case "paused":
		icon, role = g.Paused, s.Palette.Warning
	case "failed", "unhealthy":
		icon, role = g.Error, s.Palette.Danger
	}
	return s.Text(icon+" "+style.Clean(state), role)
}

// StatusChip renders one daemon or resource state with a glyph, word and role.
func StatusChip(s style.Sheet, label, state string) string {
	glyph, role := "●", s.Palette.Success
	switch state {
	case "loading", "paused", "warning":
		glyph, role = "◌", s.Palette.Warning
	case "error", "unhealthy":
		glyph, role = "✕", s.Palette.Danger
	case "offline", "muted":
		glyph, role = "○", s.Palette.Muted
	}
	if s.ASCII {
		glyph = map[string]string{"●": "+", "◌": "~", "✕": "!", "○": "-"}[glyph]
	}
	return s.Text(glyph+" "+style.Clean(label), role)
}

// Empty anchors an empty view with a small harbor mark and one next action.
// The mark is decorative; the sentence carries the instruction.
func Empty(s style.Sheet, title, next string, width int) string {
	g := tokens.Icons(s.ASCII).Anchor
	art := []string{"      " + g, "  ~~~~~~~~~~~~~", " ~  ~  ~  ~  ~"}
	if s.ASCII {
		art = []string{"      @", "  ~~~~~~~~~~~~~", " ~  ~  ~  ~  ~"}
	}
	lines := append(art, "", s.Heading(title), s.Muted(next))
	body := strings.Join(lines, "\n")
	if width > 0 && !s.Plain {
		body = lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(body)
	}
	return body
}

// Bytes returns a compact decimal byte quantity.
func Bytes(size int64) string {
	switch {
	case size >= 1_000_000_000:
		return fmt.Sprintf("%.1f GB", float64(size)/1_000_000_000)
	case size >= 1_000_000:
		return fmt.Sprintf("%.1f MB", float64(size)/1_000_000)
	case size >= 1000:
		return fmt.Sprintf("%.1f kB", float64(size)/1000)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// Age renders deterministic ages relative to a caller-provided clock.
func Age(created, now time.Time) string {
	d := now.Sub(created)
	if d < time.Second {
		return "just now"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}

// Modal composites a bounded raised panel over a dimmed, single-frame backdrop.
func Modal(s style.Sheet, backdrop, title, body string, width, height int) string {
	w := min(tokens.ModalWidth, width-tokens.Gap*2)
	h := min(height-tokens.Gap*2, lipgloss.Height(body)+tokens.PanelFrame+tokens.InsetY)
	// Keep the last action hints visible even in the smallest supported viewport.
	lines := strings.Split(body, "\n")
	capacity := max(1, h-tokens.PanelFrame-1)
	if len(lines) > capacity {
		lines = append(lines[:max(0, capacity-2)], "… content truncated", lines[len(lines)-1])
		body = strings.Join(lines, "\n")
	}
	panel := s.Panel(title, body, w, h, true)
	dim := backdrop
	if !s.Plain {
		dim = lipgloss.NewStyle().Foreground(lipgloss.Color(s.Palette.Faint)).Background(lipgloss.Color(s.Palette.BG)).Render(stripStyles(backdrop))
	}
	return style.Fit(lipgloss.NewCompositor(lipgloss.NewLayer(dim), lipgloss.NewLayer(panel).X((width-w)/2).Y((height-h)/2).Z(1)).Render(), width, height)
}

func stripStyles(text string) string {
	var out strings.Builder
	escape := false
	for _, r := range text {
		if r == '\x1b' {
			escape = true
			continue
		}
		if escape {
			if r >= '@' && r <= '~' && r != '[' {
				escape = false
			}
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}

// Pull displays real layer states and known-byte completion without a fake ETA.
func Pull(s style.Sheet, j jobs.Job, width int) string {
	barWidth := max(1, width-tokens.InsetX*2)
	lines := []string{style.Clean(j.Target) + "  " + Pill(s, j.State), fmt.Sprintf("%3.0f%%  %s/s  ETA %s", j.Fraction()*100, Bytes(int64(j.Speed)), j.ETA.Round(time.Second)), s.Gradient(chart.Bar(j.Fraction(), barWidth, s.ASCII)), ""}
	for _, l := range j.Layers {
		fraction := 0.0
		if l.Total > 0 {
			fraction = float64(l.Current) / float64(l.Total)
		}
		lines = append(lines, fmt.Sprintf("%s  %s  %s", style.Clean(l.ID), chart.Bar(fraction, max(8, barWidth/3), s.ASCII), style.Clean(l.Status)))
	}
	if j.Err != nil {
		lines = append(lines, "", s.Text(j.Err.Message, s.Palette.Danger), j.Err.Suggestion)
	}
	lines = append(lines, "", "esc hide (keeps pulling)   x cancel   R retry   J jobs")
	return strings.Join(lines, "\n")
}

// Help returns the implemented key map rather than advertising deferred actions.
func Help() string {
	return "1–7       Switch view\nj/k ↑/↓   Move through the active list\nenter / i Inspect the selected resource\nspace     Toggle selection (images/containers)\nshift+↓/↑ Extend selection\n/         Filter images, containers, or volumes\no         Cycle image name / size / age sort\np         Pull image · pause container\nr         Run image · restart container\nx / s     Start / stop container\nK         Kill container (confirmation)\nd         Remove resource (never forced)\nl         Container logs (follow, pause, filter, save)\ne         Exec a shell in a running container\ny         Copy equivalent CLI command\nc         Smart cleanup checklist\nJ         Job history and active pulls\n: ctrl+p  Command palette\nD         Toggle compact density\nR         Refresh / retry pull\n?         This help\nesc       Back / hide overlay\nq ctrl+c  Quit and cancel jobs"
}
