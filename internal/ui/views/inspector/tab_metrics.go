package inspector

import (
	"fmt"
	"math"
	"strings"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
)

// RenderMetricsTab renders measured resources and derived CPU percentages from
// this session. cpuHistory must contain interval percentages (CPUPercent), not
// cumulative CPU counters, in chronological order. It must be reset on container
// changes. When stats.CPUAvailable is true, the final history element must be
// the interval ending at stats.At. For an unavailable interval, the caller must
// append an invalid marker (for example NaN) or clear history, never leave an
// older finite interval in the final slot. The view cannot verify alignment
// because history has no timestamps. History without stats is retained graph
// evidence only, not a current CPU measurement. No sample cadence or
// collection-start time is inferred by this view.
func RenderMetricsTab(s style.Sheet, stats engine.ContainerStats, cpuHistory []float64, isAlive bool, w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	hasSample := !stats.At.IsZero() || stats.CPUAvailable || stats.MemoryAvailable || stats.NetworkAvailable || stats.BlockAvailable || stats.PIDsAvailable
	validHistory := make([]float64, 0, len(cpuHistory))
	for _, value := range cpuHistory {
		if metricsValidPercent(value) {
			validHistory = append(validHistory, value)
		}
	}
	if !isAlive && !hasSample && len(validHistory) == 0 {
		return style.Fit("Unavailable (container exited prior to session)", w, h)
	}
	if stats.State == engine.TelemetryUnavailable {
		return style.Fit("Telemetry unavailable", w, h)
	}
	lines := []string{s.Heading("Resource metrics")}
	if stats.State == engine.TelemetrySampled || (!isAlive && (hasSample || len(validHistory) > 0)) {
		lines = append(lines, s.Muted("Retained session sample (not live)"))
	} else {
		lines = append(lines, s.Muted("Current session telemetry"))
	}
	cpu := "CPU: Unavailable"
	if stats.CPUAvailable {
		cpu = "CPU: Collecting initial samples..."
		// Only the latest interval is current. An invalid latest interval must
		// not silently promote an older history point to a current reading.
		if len(cpuHistory) > 0 && metricsValidPercent(cpuHistory[len(cpuHistory)-1]) {
			value := cpuHistory[len(cpuHistory)-1]
			cpu = fmt.Sprintf("CPU: %.1f%% %s", value, metricsGauge(s, value, w))
		}
	}
	lines = append(lines, cpu)
	memory := "Memory: Unavailable"
	if stats.MemoryAvailable {
		if stats.MemoryLimit == 0 {
			memory = "Memory: " + metricsBytes(stats.Memory) + " / No memory limit configured"
		} else {
			percent := float64(stats.Memory) / float64(stats.MemoryLimit) * 100
			memory = fmt.Sprintf("Memory: %s / %s (%.1f%%) %s", metricsMemoryBytes(stats.Memory), metricsMemoryBytes(stats.MemoryLimit), percent, metricsGauge(s, percent, w))
		}
	}
	lines = append(lines, memory)
	network, block := "Network I/O: Unavailable", "Block I/O: Unavailable"
	if stats.NetworkAvailable {
		network = fmt.Sprintf("Network I/O: In %s / Out %s", metricsBytes(stats.NetworkRX), metricsBytes(stats.NetworkTX))
	}
	if stats.BlockAvailable {
		block = fmt.Sprintf("Block I/O: Read %s / Write %s", metricsBytes(stats.BlockRead), metricsBytes(stats.BlockWrite))
	}
	lines = append(lines, network, block, "", s.Heading("Session CPU history"))
	if len(validHistory) == 0 {
		lines = append(lines, "Collecting initial samples...")
	} else {
		lines = append(lines, metricsSparkline(validHistory, s.ASCII, w))
	}
	lines = append(lines, s.Muted("Session-only history; historical data before this session is not available."))
	return style.Fit(strings.Join(lines, "\n"), w, h)
}

func metricsValidPercent(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}

func metricsMemoryBytes(value uint64) string {
	if value < 1<<20 {
		return metricsBytes(value)
	}
	n := float64(value) / (1 << 20)
	if n == math.Trunc(n) {
		return fmt.Sprintf("%.0f MiB", n)
	}
	return fmt.Sprintf("%.1f MiB", n)
}

// The visual meter saturates at one full bar; the numeric reading never does.
func metricsGauge(s style.Sheet, percent float64, width int) string {
	n := min(20, max(1, width-2))
	filled := int(math.Round(math.Min(percent, 100) / 100 * float64(n)))
	full, empty := "█", "░"
	if s.ASCII {
		full, empty = "#", "-"
	}
	role := s.Palette.Success
	if percent > 90 {
		role = s.Palette.Danger
	} else if percent >= 70 {
		role = s.Palette.Warning
	}
	return "[" + s.Text(strings.Repeat(full, filled)+strings.Repeat(empty, n-filled), role) + "]"
}

func metricsBytes(value uint64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	n, unit := float64(value), 0
	for n >= 1024 && unit < len(units)-1 {
		n /= 1024
		unit++
	}
	if n == math.Trunc(n) {
		return fmt.Sprintf("%.0f %s", n, units[unit])
	}
	return fmt.Sprintf("%.1f %s", n, units[unit])
}

func metricsSparkline(history []float64, ascii bool, width int) string {
	if len(history) > width {
		history = history[len(history)-width:]
	}
	glyphs := []rune("▁▂▃▄▅▆▇█")
	if ascii {
		glyphs = []rune(".:-=+*#@")
	}
	peak := 0.0
	for _, value := range history {
		peak = math.Max(peak, value)
	}
	var out strings.Builder
	for _, value := range history {
		level := 0
		if peak > 0 {
			level = int(value / peak * float64(len(glyphs)-1))
		}
		out.WriteRune(glyphs[level])
	}
	return out.String()
}
