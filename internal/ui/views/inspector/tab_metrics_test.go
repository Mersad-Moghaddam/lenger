package inspector

import (
	"math"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
	"github.com/charmbracelet/x/ansi"
)

func metricsTestStats() engine.ContainerStats {
	return engine.ContainerStats{At: time.Date(2026, 10, 21, 8, 0, 0, 0, time.UTC), CPUAvailable: true, MemoryAvailable: true, NetworkAvailable: true, BlockAvailable: true, Memory: 210 << 20, MemoryLimit: 1024 << 20, NetworkRX: 1 << 10, NetworkTX: 2 << 20, BlockRead: 3 << 10, BlockWrite: 4 << 20}
}

func TestMetricsUnavailable(t *testing.T) {
	for _, stats := range []engine.ContainerStats{{}, {State: engine.TelemetryUnavailable}} {
		got := RenderMetricsTab(style.Sheet{Plain: true}, stats, nil, false, 120, 20)
		if !strings.Contains(got, "Unavailable (container exited prior to session)") {
			t.Fatalf("missing prior-session fallback: %s", got)
		}
		if strings.Contains(got, "0.0%") || strings.Contains(got, "No memory limit configured") {
			t.Fatalf("invented measurements: %s", got)
		}
	}
	stats := metricsTestStats()
	stats.State = engine.TelemetryUnavailable
	got := RenderMetricsTab(style.Sheet{Plain: true}, stats, []float64{99}, true, 120, 20)
	if !strings.Contains(got, "Telemetry unavailable") || strings.Contains(got, "99.0%") || strings.Contains(got, "210 MiB") {
		t.Fatalf("explicit unavailability ignored: %s", got)
	}
}

func TestMetricsDeadHistoryOnly(t *testing.T) {
	got := RenderMetricsTab(style.Sheet{Plain: true}, engine.ContainerStats{}, []float64{0, 42}, false, 140, 20)
	for _, want := range []string{"Retained session sample (not live)", "Session CPU history", "▁█", "CPU: Unavailable", "Memory: Unavailable", "Network I/O: Unavailable", "Block I/O: Unavailable"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q: %s", want, got)
		}
	}
	if strings.Contains(got, "Current session telemetry") || strings.Contains(got, "CPU: 42.0%") {
		t.Errorf("history claimed as current telemetry: %s", got)
	}
	for _, history := range [][]float64{nil, {math.NaN(), math.Inf(1), -1}} {
		got := RenderMetricsTab(style.Sheet{Plain: true}, engine.ContainerStats{}, history, false, 140, 20)
		if !strings.Contains(got, "Unavailable (container exited prior to session)") || strings.Contains(got, "Retained session sample") {
			t.Errorf("invalid history claimed as retained measurement: %s", got)
		}
	}
}

func TestMetricsCPUHistory(t *testing.T) {
	for _, tc := range []struct {
		name    string
		history []float64
		want    string
	}{
		{"first interval", nil, "CPU: Collecting initial samples..."},
		{"measured zero", []float64{0}, "CPU: 0.0%"},
		{"multicore", []float64{10, 250}, "CPU: 250.0%"},
		{"invalid", []float64{math.NaN(), math.Inf(1), -1}, "CPU: Collecting initial samples..."},
		{"invalid latest does not reuse stale reading", []float64{42, math.NaN()}, "CPU: Collecting initial samples..."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := RenderMetricsTab(style.Sheet{Plain: true}, metricsTestStats(), tc.history, true, 140, 20)
			if !strings.Contains(got, tc.want) || !strings.Contains(got, "historical data before this session is not available") {
				t.Fatalf("unexpected CPU: %s", got)
			}
			if tc.name == "invalid latest does not reuse stale reading" && strings.Contains(got, "CPU: 42.0%") {
				t.Fatalf("older interval promoted to current sample: %s", got)
			}
		})
	}
	stats := metricsTestStats()
	stats.CPUAvailable = false
	got := RenderMetricsTab(style.Sheet{Plain: true}, stats, []float64{42}, true, 140, 20)
	if !strings.Contains(got, "CPU: Unavailable") {
		t.Fatalf("missing CPU presented as measured: %s", got)
	}
}

func TestMetricsMemoryAndIO(t *testing.T) {
	stats := metricsTestStats()
	got := RenderMetricsTab(style.Sheet{Plain: true}, stats, nil, true, 140, 20)
	for _, want := range []string{"210 MiB / 1024 MiB (20.5%)", "Network I/O: In 1 KiB / Out 2 MiB", "Block I/O: Read 3 KiB / Write 4 MiB"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q: %s", want, got)
		}
	}
	stats.MemoryLimit = 0
	got = RenderMetricsTab(style.Sheet{Plain: true}, stats, nil, true, 140, 20)
	if !strings.Contains(got, "No memory limit configured") || strings.Contains(got, "NaN") {
		t.Fatalf("bad unlimited memory: %s", got)
	}
	stats.MemoryAvailable = false
	stats.NetworkAvailable = false
	stats.BlockAvailable = false
	got = RenderMetricsTab(style.Sheet{Plain: true}, stats, nil, true, 140, 20)
	for _, want := range []string{"Memory: Unavailable", "Network I/O: Unavailable", "Block I/O: Unavailable"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q: %s", want, got)
		}
	}
	if strings.Contains(got, "No memory limit configured") {
		t.Fatal("unknown config called unlimited")
	}
}

func TestMetricsThresholdStyling(t *testing.T) {
	s := style.Sheet{Palette: tokens.Palette{Success: "#00ff00", Warning: "#ffff00", Danger: "#ff0000"}}
	for _, tc := range []struct {
		percent uint64
		role    string
	}{{0, s.Palette.Success}, {69, s.Palette.Success}, {70, s.Palette.Warning}, {90, s.Palette.Warning}, {91, s.Palette.Danger}, {150, s.Palette.Danger}} {
		stats := metricsTestStats()
		stats.MemoryLimit = 100
		stats.Memory = tc.percent
		got := RenderMetricsTab(s, stats, nil, true, 140, 20)
		// Inspect the rendered gauge's actual ANSI styling, not color-name prose.
		var memory string
		for _, line := range strings.Split(got, "\n") {
			if strings.HasPrefix(ansi.Strip(line), "Memory:") {
				memory = line
			}
		}
		bar := lipgloss.NewStyle().Foreground(lipgloss.Color(tc.role)).Render(metricsGaugeFillForTest(tc.percent))
		if !strings.Contains(memory, bar) || !strings.Contains(bar, "\x1b[") {
			t.Fatalf("%d%% gauge missing styled fill %q: %q", tc.percent, bar, memory)
		}
	}
}

func metricsGaugeFillForTest(percent uint64) string {
	filled := int(math.Round(math.Min(float64(percent), 100) / 100 * 20))
	return strings.Repeat("█", filled) + strings.Repeat("░", 20-filled)
}

func TestMetricsSparklineAndRetained(t *testing.T) {
	stats := metricsTestStats()
	stats.State = engine.TelemetrySampled
	for _, ascii := range []bool{false, true} {
		got := RenderMetricsTab(style.Sheet{Plain: true, ASCII: ascii}, stats, []float64{0, 50, 100}, false, 140, 20)
		want := "▁▄█"
		if ascii {
			want = ".=@"
		}
		if !strings.Contains(got, want) || !strings.Contains(got, "Retained session sample") || !strings.Contains(got, "Session CPU history") {
			t.Fatalf("bad retained history: %s", got)
		}
	}
	got := RenderMetricsTab(style.Sheet{Plain: true}, stats, nil, false, 140, 20)
	if !strings.Contains(got, "Collecting initial samples...") {
		t.Fatal(got)
	}
}

func TestMetricsBounds(t *testing.T) {
	for _, size := range [][2]int{{0, 5}, {5, 0}, {1, 1}, {20, 5}, {80, 12}} {
		got := RenderMetricsTab(style.Sheet{Plain: true}, metricsTestStats(), []float64{1, 2}, true, size[0], size[1])
		if size[0] <= 0 || size[1] <= 0 {
			if got != "" {
				t.Fatal("nonempty zero area")
			}
			continue
		}
		if lipgloss.Height(got) != size[1] || lipgloss.Width(got) > size[0] {
			t.Fatalf("bad bounds %v: %q", size, got)
		}
	}
}
