package views

import (
	"fmt"
	"strings"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/chart"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/components"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
)

// CleanLayerCommand removes build-step boilerplate from Docker history commands.
func CleanLayerCommand(cmd string) string {
	c := strings.TrimSpace(cmd)
	c = strings.TrimPrefix(c, "/bin/sh -c #(nop) ")
	c = strings.TrimPrefix(c, "/bin/sh -c ")
	c = strings.TrimPrefix(c, "#(nop) ")
	return style.Clean(strings.TrimSpace(c))
}

// LayerBreakdown renders the visual layer sounding and highlights the biggest layer (IMG-07).
func LayerBreakdown(s style.Sheet, layers []engine.Layer, width int) string {
	if len(layers) == 0 {
		return s.Muted("No layer history available.")
	}

	var totalSize int64
	var biggestIdx = -1
	var biggestSize int64

	sizes := make([]int64, len(layers))
	for i, l := range layers {
		sizes[i] = l.Size
		totalSize += l.Size
		if l.Size > biggestSize {
			biggestSize = l.Size
			biggestIdx = i
		}
	}

	var lines []string

	// Proportional bar ribbon
	ribbonWidth := max(10, width-tokens.InsetX*2-tokens.PanelFrame)
	segments := chart.Segments(sizes, ribbonWidth)
	var ribbon strings.Builder
	for n, cells := range segments {
		glyph := "█"
		if s.ASCII {
			glyph = "#"
		}
		color := s.Palette.Series[n%len(s.Palette.Series)]
		if n == biggestIdx {
			color = s.Palette.Danger
		}
		ribbon.WriteString(s.Text(strings.Repeat(glyph, cells), color))
	}
	lines = append(lines, ribbon.String())

	if biggestIdx >= 0 && biggestSize > 0 {
		biggestCmd := CleanLayerCommand(layers[biggestIdx].Command)
		biggestCmd = style.Fit(biggestCmd, max(12, width-28), 1)
		pct := float64(biggestSize) / float64(max(1, totalSize)) * 100
		callout := fmt.Sprintf("Biggest layer: %s (%.1f%%) — %s", components.Bytes(biggestSize), pct, biggestCmd)
		lines = append(lines, s.Text(callout, s.Palette.Warning))
	}

	lines = append(lines, "")

	// Layer enumeration
	for i, l := range layers {
		idStr := l.ID
		if len(idStr) > 12 {
			idStr = idStr[:12]
		}
		if idStr == "" || idStr == "<missing>" {
			idStr = fmt.Sprintf("layer-%02d", i+1)
		}

		sizeStr := components.Bytes(l.Size)
		badge := "        "
		if i == biggestIdx && l.Size > 0 {
			badge = s.Text("[BIGGEST]", s.Palette.Danger)
		}

		cmdClean := CleanLayerCommand(l.Command)
		cmdFit := style.Fit(cmdClean, max(10, width-36), 1)

		lines = append(lines, fmt.Sprintf("%-10s %8s %s %s", idStr, sizeStr, badge, cmdFit))
	}

	return strings.Join(lines, "\n")
}
