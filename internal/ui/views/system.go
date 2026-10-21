package views

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

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

// System shows measured disk usage and host/daemon information (SYS-03).
func System(s style.Sheet, usage engine.SystemUsage, daemon engine.DaemonInfo, loaded bool, failure *engine.ClassifiedError, width, height int, fixture bool) string {
	if !loaded {
		body := "Reading disk usage and daemon telemetry from Docker…\n\nThe system tab does not invent capacity or telemetry."
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

	if width < tokens.DetailWidth {
		if daemon.ServerVersion != "" {
			lines = append(lines, "", s.Heading("DAEMON INFORMATION"),
				fmt.Sprintf("Docker Engine: %s (%s)", style.Clean(daemon.ServerVersion), style.Clean(daemon.Arch)),
				fmt.Sprintf("OS: %s", style.Clean(daemon.OS)),
				fmt.Sprintf("CPUs: %d · Memory: %s", daemon.NCPU, components.Bytes(daemon.MemTotal)),
			)
		}
		return s.Panel("SYSTEM · DISK USAGE", strings.Join(lines, "\n"), width, height, true)
	}

	left := width * 3 / 5
	right := width - left - tokens.Gap

	var infoLines []string
	infoLines = append(infoLines, s.Heading("DAEMON INFORMATION"))
	if daemon.ServerVersion != "" {
		infoLines = append(infoLines,
			"Docker Engine: "+style.Clean(daemon.ServerVersion)+" (API "+style.Clean(daemon.APIVersion)+")",
			"Host OS: "+style.Clean(daemon.OS)+" ("+style.Clean(daemon.Arch)+")",
			"Kernel: "+style.Clean(daemon.KernelVersion),
			"Storage Driver: "+style.Clean(daemon.Driver),
			fmt.Sprintf("Cgroup: %s (v%s)", style.Clean(daemon.CgroupDriver), style.Clean(daemon.CgroupVersion)),
			fmt.Sprintf("Hardware: %d CPUs · %s RAM", daemon.NCPU, components.Bytes(daemon.MemTotal)),
			"Root Dir: "+style.Clean(daemon.RootDir),
		)
	} else {
		infoLines = append(infoLines, "Daemon telemetry unavailable.")
	}

	infoLines = append(infoLines, "", s.Heading("SMART CLEANUP"),
		"Preview candidates before deletion.",
		"dangling images, unused volumes, containers",
		"",
		"Press c to open checklist.",
		"",
		s.Muted("Volumes require confirmation. No forced prunes."),
	)

	top := s.Panel("SYSTEM · DISK USAGE", strings.Join(lines, "\n"), left, height, true)
	side := s.Panel("DAEMON & CLEANUP", strings.Join(infoLines, "\n"), right, height, false)
	return style.Fit(lipgloss.JoinHorizontal(lipgloss.Top, top, strings.Repeat(" ", tokens.Gap), side), width, height)
}
