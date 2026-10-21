package views

import (
	"charm.land/lipgloss/v2"
	"fmt"
	"strings"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/chart"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/components"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
)

type usageRow struct {
	label      string
	category   engine.UsageCategory
	role       string
	reclaim    int64
	showActive bool
}

// System shows measured disk usage and the cleanup entry point. It never
// presents a fabricated capacity: Docker reports used and reclaimable bytes.
func System(s style.Sheet, usage engine.SystemUsage, loaded bool, failure *engine.ClassifiedError, width, height int, fixture bool) string {
	if !loaded {
		body := "Reading disk usage from Docker…\n\nThe system tab does not invent capacity or telemetry."
		if failure != nil {
			body = failure.Message + "\n\n" + failure.Suggestion + "\n\nR retry"
		}
		return s.Panel("SYSTEM", body, width, height, true)
	}
	rows := []usageRow{
		{"images", usage.Images, s.Palette.Accent2, usage.Images.Reclaimable, true},
		{"containers", usage.Containers, s.Palette.Info, usage.Containers.Reclaimable, true},
		{"volumes", usage.Volumes, s.Palette.Accent, usage.Volumes.Reclaimable, true},
		{"build cache", usage.BuildCache, s.Palette.Warning, usage.BuildCache.Reclaimable, false},
	}
	labelWidth := max(12, min(18, width/5))
	var lines []string
	if fixture {
		lines = append(lines, s.Muted("fixture engine · disk usage is synthetic"))
	}
	lines = append(lines, s.Heading("LOCAL STORAGE"), components.Bytes(usage.Used)+" in use · "+components.Bytes(usage.Reclaimable)+" reclaimable", "")
	sizes := make([]int64, len(rows))
	for n, row := range rows {
		sizes[n] = max(0, row.category.Size)
	}
	barWidth := max(8, width-tokens.PanelFrame-tokens.InsetX*2)
	segments := chart.Segments(sizes, barWidth)
	var ribbon strings.Builder
	for n, cells := range segments {
		glyph := "█"
		if s.ASCII {
			glyph = "#"
		}
		ribbon.WriteString(s.Text(strings.Repeat(glyph, cells), rows[n].role))
	}
	if usage.Used > 0 {
		lines = append(lines, ribbon.String(), "")
	}
	header := fmt.Sprintf("%-*s %5s %10s %10s", labelWidth, "category", "count", "size", "reclaim")
	lines = append(lines, s.Muted(header))
	for _, row := range rows {
		count := fmt.Sprintf("%d", row.category.Total)
		if row.showActive {
			count = fmt.Sprintf("%d/%d", row.category.Active, row.category.Total)
		}
		line := fmt.Sprintf("%-*s %5s %10s %10s",
			labelWidth, row.label, count, components.Bytes(max(0, row.category.Size)), components.Bytes(max(0, row.reclaim)))
		lines = append(lines, s.Text(line, row.role))
	}
	lines = append(lines, "", s.Heading("RECLAIMABLE SHARE"))
	fraction := 0.0
	if usage.Used > 0 {
		fraction = float64(usage.Reclaimable) / float64(usage.Used)
	}
	lines = append(lines, chart.Sounding(min(1, fraction), max(1, width-tokens.PanelFrame-tokens.InsetX*2), s.ASCII))
	lines = append(lines, fmt.Sprintf("%.0f%% of used storage is currently reclaimable.", fraction*100))
	body := strings.Join(lines, "\n")
	if width < tokens.DetailWidth {
		return s.Panel("SYSTEM · DISK USAGE", body, width, height, true)
	}
	left := width * 3 / 5
	right := width - left - tokens.Gap
	cleanup := s.Heading("SMART CLEANUP") + "\n\n" +
		"Preview every candidate before anything is removed.\n\n" +
		"dangling images\nunused images\nstopped containers\nunused volumes\nbuild cache\n\n" +
		"Volumes require typing the word volumes because their data cannot be recovered.\n\n" +
		"Press c to open the checklist.\n\n" +
		s.Muted("No force flags. No volume removal without a separate confirmation.")
	top := s.Panel("SYSTEM · DISK USAGE", body, left, height, true)
	side := s.Panel("CLEANUP", cleanup, right, height, false)
	return style.Fit(lipgloss.JoinHorizontal(lipgloss.Top, top, strings.Repeat(" ", tokens.Gap), side), width, height)
}
