// Package views renders resource screens using shared styles and components.
package views

import (
	"charm.land/lipgloss/v2"
	"fmt"
	"github.com/Mersad-Moghaddam/lenger/internal/core/store"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/chart"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/components"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
	"strings"
)

// Dashboard shows real counts while clearly identifying unavailable telemetry.
func Dashboard(s style.Sheet, data store.Store, events []engine.Event, width, height int, fixture bool) string {
	running, exited, unhealthy := 0, 0, 0
	for _, c := range data.Containers {
		switch c.State {
		case "running":
			running++
		case "exited":
			exited++
		}
		if strings.Contains(c.Status, "unhealthy") {
			unhealthy++
		}
	}
	var disk, unused int64
	for _, i := range data.Images {
		disk += i.Size
		if len(i.UsedBy) == 0 {
			unused += i.Size
		}
	}
	health := fmt.Sprintf("%3d  %s\n%3d  %s\n%3d  %s", running, components.Pill(s, "running"), exited, components.Pill(s, "exited"), unhealthy, components.Pill(s, "unhealthy"))
	attention := fmt.Sprintf("%d unhealthy containers\n%s unused image bytes\n\nUnused is not a cleanup estimate.\nShared layers may remain on disk.", unhealthy, components.Bytes(unused))
	leftWidth := max(24, width/4)
	rightWidth := width - leftWidth - tokens.Gap
	topHeight := max(7, height/2)
	left := s.Panel("HEALTH", health, leftWidth, topHeight, false) + "\n" + s.Panel("NEEDS ATTENTION", attention, leftWidth, max(1, height-topHeight-1), false)
	chartWidth := (rightWidth - tokens.Gap) / 2
	cpu, memory := "Not subscribed\nLive stats arrive in M2.", "Not subscribed\nLive stats arrive in M2."
	if fixture {
		series := []float64{2, 4, 7, 18, 12, 9, 4, 5, 8, 11, 15, 13, 10, 12, 18}
		cpu = "18%  fixture data\n" + chart.Braille(series, max(1, chartWidth-tokens.InsetX*2-tokens.PanelFrame), 2, s.ASCII)
		memory = "2.4 GB  fixture data\n" + chart.Braille([]float64{1, 1, 2, 2, 3, 3, 4, 5, 5, 5, 5}, max(1, chartWidth-tokens.InsetX*2-tokens.PanelFrame), 2, s.ASCII)
	}
	top := lipgloss.JoinHorizontal(lipgloss.Top, s.Panel("CPU", cpu, chartWidth, 7, false), strings.Repeat(" ", tokens.Gap), s.Panel("MEMORY", memory, rightWidth-chartWidth-tokens.Gap, 7, false))
	fraction := 0.0
	if disk > 0 {
		fraction = float64(disk-unused) / float64(disk)
	}
	diskWidth := max(1, rightWidth-tokens.InsetX*2-tokens.PanelFrame)
	diskBody := fmt.Sprintf("%s local · %3.0f%% referenced\n%s\n%s unused; shared bytes may overlap.", components.Bytes(disk), fraction*100, chart.Sounding(fraction, diskWidth, s.ASCII), components.Bytes(unused))
	if disk == 0 {
		diskBody = components.Empty(s, "No local images.", "Press p to pull one.", diskWidth)
	}
	eventLines := []string{}
	for n := len(events) - 1; n >= 0 && len(eventLines) < 5; n-- {
		e := events[n]
		eventLines = append(eventLines, e.At.Format("15:04:05")+"  "+style.Clean(e.Kind+" "+e.Action+" "+engine.ShortID(e.ID)))
	}
	if len(eventLines) == 0 {
		eventLines = append(eventLines, "Waiting for Docker events.")
	}
	right := top + "\n" + s.Panel("SOUNDINGS · IMAGE STORAGE", diskBody, rightWidth, 8, false) + "\n" + s.Panel("EVENTS", strings.Join(eventLines, "\n"), rightWidth, max(1, height-18), false)
	return style.Fit(lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", tokens.Gap), right), width, height)
}

// Stub makes deferred resource screens explicit instead of implying live data.
func Stub(s style.Sheet, name, requirements string, width, height int) string {
	return s.Panel(name, "This workspace is reserved for "+requirements+".\n\nImages and pulling are available now.\nPress 2 for images, p to pull, or ? for help.", width, height, false)
}
