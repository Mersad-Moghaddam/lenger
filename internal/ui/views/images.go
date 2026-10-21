package views

import (
	"charm.land/lipgloss/v2"
	"fmt"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/chart"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/components"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
	"sort"
	"strings"
	"time"
)

// ImageList displays only the visible window, keeping large collections cheap.
func ImageList(s style.Sheet, items []engine.Image, cursor int, selected map[string]bool, sortName, filter string, width, height int, now time.Time) string {
	lines := []string{s.Muted("sort " + sortName + "  / " + style.Clean(filter))}
	capacity := max(1, height-tokens.PanelFrame-tokens.InsetY*2)
	start := max(0, cursor-capacity+1)
	end := min(len(items), start+capacity)
	for n := start; n < end; n++ {
		i := items[n]
		marker := " "
		g := tokens.Icons(s.ASCII)
		if n == cursor {
			marker = g.Cursor
		}
		if selected[i.ID] {
			marker = g.Selected
		}
		status := "unused"
		if len(i.UsedBy) > 0 {
			status = "in use"
		}
		if i.Dangling {
			status = "dangling"
		}
		nameWidth := max(8, width-39)
		name := style.Fit(style.Clean(i.Name()), nameWidth, 1)
		line := fmt.Sprintf("%s %s %8s %s", marker, name, components.Bytes(i.Size), status)
		if width >= tokens.DetailWidth/2 {
			line += "  " + components.Age(i.Created, now)
		}
		if n == cursor {
			line = s.Text(line, s.Palette.Accent)
		}
		lines = append(lines, line)
	}
	if len(items) == 0 {
		next := "Press p to pull an image, or / to change the filter."
		return s.Panel(fmt.Sprintf("IMAGES  %d", len(items)), components.Empty(s, "No images in this harbor.", next, max(1, width-tokens.PanelFrame-tokens.InsetX*2)), width, height, true)
	}
	return s.Panel(fmt.Sprintf("IMAGES  %d", len(items)), strings.Join(lines, "\n"), width, height, true)
}

// ImageDetail shows selected metadata; all sensitive fields arrive masked.
func ImageDetail(s style.Sheet, image engine.Image, inspect *engine.Inspection, width, height int, now time.Time) string {
	lines := []string{image.ShortID() + "  " + components.Bytes(image.Size), components.Age(image.Created, now), ""}
	if inspect == nil {
		lines = append(lines, "Press i or Enter to load inspection and layers.")
	} else {
		lines = append(lines, style.Clean(inspect.Platform), "", s.Heading("LAYERS"))
		sizes := make([]int64, len(inspect.Layers))
		for n, l := range inspect.Layers {
			sizes[n] = l.Size
		}
		segments := chart.Segments(sizes, max(1, width-tokens.InsetX*2-tokens.PanelFrame))
		var ribbon strings.Builder
		for n, cells := range segments {
			glyph := "█"
			if s.ASCII {
				glyph = "#"
			}
			ribbon.WriteString(s.Text(strings.Repeat(glyph, cells), s.Palette.Series[n%len(s.Palette.Series)]))
		}
		lines = append(lines, ribbon.String())
		for n, l := range inspect.Layers {
			if n >= 4 {
				lines = append(lines, fmt.Sprintf("… %d more layers", len(inspect.Layers)-n))
				break
			}
			lines = append(lines, components.Bytes(l.Size)+"  "+style.Clean(l.Command))
		}
		lines = append(lines, "", s.Heading("CONFIG"), "Entrypoint: "+style.Clean(strings.Join(inspect.Entrypoint, " ")), "Command: "+style.Clean(strings.Join(inspect.Command, " ")), "Ports: "+style.Clean(strings.Join(inspect.Ports, ", ")))
		lines = append(lines, "", s.Heading("ENVIRONMENT (MASKED)"))
		for _, v := range inspect.Env {
			lines = append(lines, style.Clean(v))
		}
		keys := make([]string, 0, len(inspect.Labels))
		for k := range inspect.Labels {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if len(keys) > 0 {
			lines = append(lines, "", s.Heading("LABELS (MASKED)"))
			for _, k := range keys {
				lines = append(lines, style.Clean(k)+"="+style.Clean(inspect.Labels[k]))
			}
		}
		if len(inspect.Digests) > 0 {
			lines = append(lines, "", s.Heading("DIGESTS"))
			for _, digest := range inspect.Digests {
				lines = append(lines, style.Clean(digest))
			}
		}
	}
	lines = append(lines, "", s.Heading("USED BY"))
	if len(image.UsedBy) == 0 {
		lines = append(lines, "No containers reference this image.")
	}
	for _, c := range image.UsedBy {
		lines = append(lines, style.Clean(c.Name)+"  "+components.Pill(s, c.State))
	}
	return s.Panel(image.Name(), strings.Join(lines, "\n"), width, height, false)
}

// Images composes responsive list/detail layouts from one selected resource.
func Images(s style.Sheet, items []engine.Image, cursor int, selected map[string]bool, inspect *engine.Inspection, showDetail bool, sortName, filter string, width, height int, now time.Time) string {
	if len(items) == 0 {
		return ImageList(s, items, 0, selected, sortName, filter, width, height, now)
	}
	cursor = max(0, min(cursor, len(items)-1))
	if width < tokens.DetailWidth {
		if showDetail {
			return ImageDetail(s, items[cursor], inspect, width, height, now)
		}
		return ImageList(s, items, cursor, selected, sortName, filter, width, height, now)
	}
	left := (width - tokens.Gap) / 2
	right := width - left - tokens.Gap
	if width >= tokens.WideWidth {
		left = width / 3
		right = width/3 - tokens.Gap
	}
	result := lipgloss.JoinHorizontal(lipgloss.Top, ImageList(s, items, cursor, selected, sortName, filter, left, height, now), strings.Repeat(" ", tokens.Gap), ImageDetail(s, items[cursor], inspect, right, height, now))
	if width >= tokens.WideWidth {
		remaining := width - lipgloss.Width(result) - tokens.Gap
		result = lipgloss.JoinHorizontal(lipgloss.Top, result, strings.Repeat(" ", tokens.Gap), s.Panel("IMAGE ACTIVITY", "Live container stats arrive in M2.\n\nNo synthetic live data is shown.", remaining, height, false))
	}
	return result
}
