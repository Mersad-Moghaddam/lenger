package views

import (
	"charm.land/lipgloss/v2"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/chart"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/components"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
)

// Containers composes the responsive list and structured inspection view.
func Containers(s style.Sheet, items []engine.Container, cursor int, selected map[string]bool, filter string, inspect *engine.ContainerInspection, showDetail, raw bool, stats map[string]engine.ContainerStats, cpu map[string]float64, history map[string][]float64, width, height int, now time.Time) string {
	if len(items) == 0 {
		return ContainerList(s, items, 0, selected, filter, stats, cpu, history, width, height, now)
	}
	cursor = max(0, min(cursor, len(items)-1))
	if width < tokens.DetailWidth {
		if showDetail {
			return ContainerDetail(s, items[cursor], inspect, raw, stats, cpu, history, width, height, now)
		}
		return ContainerList(s, items, cursor, selected, filter, stats, cpu, history, width, height, now)
	}
	left := max(44, width*2/5)
	right := width - left - tokens.Gap
	result := lipgloss.JoinHorizontal(lipgloss.Top,
		ContainerList(s, items, cursor, selected, filter, stats, cpu, history, left, height, now),
		strings.Repeat(" ", tokens.Gap),
		ContainerDetail(s, items[cursor], inspect, raw, stats, cpu, history, right, height, now),
	)
	return style.Fit(result, width, height)
}

// ContainerList renders a bounded chart-table with measured resource samples.
func ContainerList(s style.Sheet, items []engine.Container, cursor int, selected map[string]bool, filter string, stats map[string]engine.ContainerStats, cpu map[string]float64, history map[string][]float64, width, height int, now time.Time) string {
	lines := []string{s.Muted("filter " + emptyDash(filter))}
	const rowHeight = 2
	available := max(1, height-tokens.PanelFrame-tokens.InsetY*2-1)
	capacity := max(1, available/rowHeight)
	start := max(0, cursor-capacity+1)
	end := min(len(items), start+capacity)
	g := tokens.Icons(s.ASCII)
	contentWidth := max(1, width-tokens.PanelFrame-tokens.InsetX*2)
	for n := start; n < end; n++ {
		c := items[n]
		marker := " "
		if n == cursor {
			marker = g.Cursor
		}
		if selected[c.ID] {
			marker = g.Selected
		}
		state, status := listContainerStatus(c)
		nameWidth := max(8, min(24, width/3))
		imageWidth := max(8, width-nameWidth-19)
		name := padCell(style.Clean(c.Name), nameWidth)
		image := padCell(style.Clean(c.Image), imageWidth)
		if status == "" {
			status = state
		}
		pill := components.StatusPill(s, state, status)
		line := fmt.Sprintf("%s %s %s %s", marker, name, pill, image)
		if n == cursor {
			line = s.Text(line, s.Palette.Accent)
		}
		lines = append(lines, line)
		if rowHeight == 2 {
			resource := "cpu —  mem —"
			if sample, ok := stats[c.ID]; ok && sample.Current(c.State, now) {
				cpuText := "  — "
				if percent, sampled := cpu[c.ID]; sampled && sample.CPUAvailable {
					cpuText = fmt.Sprintf("%4.1f%%", percent)
				}
				spark := ""
				if sample.CPUAvailable {
					spark = chart.Sparkline(history[c.ID], 10, s.ASCII)
				}
				memory := "—"
				if sample.MemoryAvailable {
					memory = components.Bytes(int64(sample.Memory))
				}
				resource = fmt.Sprintf("cpu %s %s  mem %s", cpuText, spark, memory)
			}
			context := []string{}
			if c.Project != "" {
				context = append(context, c.Project)
			}
			if len(c.Ports) > 0 {
				context = append(context, c.Ports[0])
			}
			if c.Running() && !c.Created.IsZero() {
				context = append(context, "up "+uptime(c.Created, now))
			}
			second := "  " + resource
			if len(context) > 0 {
				second += "  " + strings.Join(context, " · ")
			}
			second = style.Fit(style.Clean(second), contentWidth, 1)
			if n == cursor {
				second = s.Text(second, s.Palette.Accent)
			} else {
				second = s.Muted(second)
			}
			lines = append(lines, second)
		}
	}
	if len(items) == 0 {
		next := "Run an image with r, or / to change the filter."
		return s.Panel("CONTAINERS  0", components.Empty(s, "No containers at anchor.", next, max(1, width-tokens.PanelFrame-tokens.InsetX*2)), width, height, true)
	}
	return s.Panel(fmt.Sprintf("CONTAINERS  %d", len(items)), strings.Join(lines, "\n"), width, height, true)
}

var exitStatusPattern = regexp.MustCompile(`^Exited \(([0-9]+)\)(?: |$)`)

// Docker list status is evidence only when its canonical exit form is present;
// missing/unknown status must not manufacture an exit code of zero.
func listContainerStatus(c engine.Container) (string, string) {
	if c.State == "restarting" {
		return "restarting", ""
	}
	if c.State == "dead" || c.Health == "unhealthy" {
		return "unhealthy", ""
	}
	if c.Health == "starting" || c.State == "created" {
		return "starting", ""
	}
	if c.State == "exited" {
		if match := exitStatusPattern.FindStringSubmatch(c.Status); match != nil {
			code, err := strconv.Atoi(match[1])
			if err == nil {
				role := "exited"
				if code != 0 {
					role = "failed"
				}
				return role, fmt.Sprintf("Exited (%d)", code)
			}
		}
	}
	return c.State, ""
}

// ContainerDetail renders structured metadata first and optional masked JSON.
func ContainerDetail(s style.Sheet, c engine.Container, inspect *engine.ContainerInspection, raw bool, stats map[string]engine.ContainerStats, cpu map[string]float64, history map[string][]float64, width, height int, now time.Time) string {
	lines := []string{
		engine.ShortID(c.ID) + "  " + components.Pill(s, c.State),
		style.Clean(c.Image), "",
	}
	if c.Running() {
		lines = append(lines, s.Heading("RESOURCES"))
		if sample, ok := stats[c.ID]; ok {
			spark := chart.Sparkline(history[c.ID], 18, s.ASCII)
			cpuText := "sampling"
			if percent, sampled := cpu[c.ID]; sampled {
				cpuText = fmt.Sprintf("%5.1f%%", percent)
			}
			lines = append(lines, "CPU: "+cpuText+"  "+spark)
			lines = append(lines, "Memory: "+components.Bytes(int64(sample.Memory))+" / "+components.Bytes(int64(sample.MemoryLimit)))
			lines = append(lines, "Network: rx "+components.Bytes(int64(sample.NetworkRX))+" · tx "+components.Bytes(int64(sample.NetworkTX)))
			lines = append(lines, "Block I/O: r "+components.Bytes(int64(sample.BlockRead))+" · w "+components.Bytes(int64(sample.BlockWrite)))
			lines = append(lines, fmt.Sprintf("PIDs: %d", sample.PIDs))
		} else {
			lines = append(lines, "Sampling measured stats…")
		}
		lines = append(lines, "")
	}
	if inspect == nil {
		lines = append(lines, "Press i or Enter to load inspection.")
	} else {
		if raw {
			lines = append(lines, s.Heading("SANITIZED JSON"), "Environment values and labels are masked.")
			data, err := json.MarshalIndent(inspect, "", "  ")
			if err == nil {
				for _, line := range strings.Split(string(data), "\n") {
					lines = append(lines, style.Clean(line))
				}
			}
			lines = append(lines, "", "J structured view")
			return s.Panel(c.Name, strings.Join(lines, "\n"), width, height, false)
		}
		lines = append(lines, s.Heading("STATE"))
		lines = append(lines, "Status: "+style.Clean(inspect.Status))
		if inspect.Health != "" {
			lines = append(lines, "Health: "+style.Clean(inspect.Health))
		}
		lines = append(lines, "Uptime: "+uptime(inspect.StartedAt, now))
		if inspect.Restart != "" {
			lines = append(lines, "Restart: "+style.Clean(inspect.Restart))
		}
		if inspect.ExitCode != 0 {
			lines = append(lines, fmt.Sprintf("Exit code: %d", inspect.ExitCode))
		}
		if inspect.HealthCheck != nil {
			lines = append(lines, "", s.Heading("HEALTHCHECK PROBE"))
			probe := inspect.HealthCheck
			lines = append(lines,
				fmt.Sprintf("Status: %s   Exit code: %d   Failing streak: %d",
					style.Clean(probe.Status),
					probe.ExitCode,
					probe.FailingStreak,
				),
			)
			if !probe.End.IsZero() {
				lines = append(lines, fmt.Sprintf("Last checked: %s", components.Age(probe.End, now)))
			}
			if probe.Output != "" {
				cleanOut := strings.ReplaceAll(strings.TrimSpace(probe.Output), "\r", "")
				outLines := strings.Split(cleanOut, "\n")
				lines = append(lines, "Probe output:")
				for _, ol := range outLines[:min(3, len(outLines))] {
					lines = append(lines, "  "+s.Muted(style.Fit(style.Clean(ol), max(10, width-tokens.InsetX*2-tokens.PanelFrame-4), 1)))
				}
			}
		}
		lines = append(lines, "", s.Heading("NETWORK"))
		lines = append(lines, "Mode: "+emptyDash(inspect.NetworkMode))
		lines = append(lines, "Networks: "+emptyDash(strings.Join(inspect.Networks, ", ")))
		lines = append(lines, "Ports: "+emptyDash(strings.Join(inspect.Ports, ", ")))
		lines = append(lines, "", s.Heading("MOUNTS"))
		if len(inspect.Mounts) == 0 {
			lines = append(lines, "No mounts.")
		}
		for _, m := range inspect.Mounts {
			mode := "rw"
			if !m.RW {
				mode = "ro"
			}
			source := m.Source
			if m.Name != "" {
				source = m.Name
			}
			lines = append(lines, fmt.Sprintf("%s -> %s  %s  %s", style.Clean(source), style.Clean(m.Destination), style.Clean(m.Type), mode))
		}
		lines = append(lines, "", s.Heading("ENVIRONMENT (MASKED)"))
		if len(inspect.Env) == 0 {
			lines = append(lines, "No environment entries.")
		}
		for _, value := range inspect.Env {
			lines = append(lines, style.Clean(value))
		}
		lines = append(lines, "", s.Heading("COMMAND (MASKED)"))
		lines = append(lines, "Entrypoint: "+emptyDash(style.Clean(strings.Join(inspect.Entrypoint, " "))))
		lines = append(lines, "Command: "+emptyDash(style.Clean(strings.Join(inspect.Command, " "))))
		keys := make([]string, 0, len(inspect.Labels))
		for key := range inspect.Labels {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		if len(keys) > 0 {
			lines = append(lines, "", s.Heading("LABELS (MASKED)"))
			for _, key := range keys {
				lines = append(lines, style.Clean(key)+"="+style.Clean(inspect.Labels[key]))
			}
		}
		lines = append(lines, "", "J sanitized JSON")
	}
	if c.Project != "" {
		lines = append(lines, "", s.Heading("PROJECT"), style.Clean(c.Project)+" / "+style.Clean(c.Service))
	}
	return s.Panel(c.Name, strings.Join(lines, "\n"), width, height, false)
}

func uptime(started, now time.Time) string {
	if started.IsZero() {
		return "not running"
	}
	d := now.Sub(started)
	if d < 0 {
		return "just now"
	}
	if d >= 24*time.Hour {
		return fmt.Sprintf("%dd %dh", int(d.Hours()/24), int(d.Hours())%24)
	}
	if d >= time.Hour {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dm", max(0, int(d.Minutes())))
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return style.Clean(value)
}

func padCell(value string, width int) string {
	value = style.Fit(value, width, 1)
	return value + strings.Repeat(" ", max(0, width-lipgloss.Width(value)))
}
