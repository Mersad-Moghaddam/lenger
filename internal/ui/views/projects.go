package views

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/components"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
)

// Projects renders the multi-container compose project list and service detail view (PRJ-01).
func Projects(s style.Sheet, items []engine.ComposeProject, cursor int, filter string, streamOutput []string, width, height int, now time.Time) string {
	if len(items) == 0 {
		return ProjectList(s, items, 0, filter, width, height, now)
	}
	cursor = max(0, min(cursor, len(items)-1))
	if width < tokens.DetailWidth {
		return ProjectList(s, items, cursor, filter, width, height, now)
	}
	left := width * 3 / 7
	right := width - left - tokens.Gap
	result := lipgloss.JoinHorizontal(lipgloss.Top,
		ProjectList(s, items, cursor, filter, left, height, now),
		strings.Repeat(" ", tokens.Gap),
		ProjectDetail(s, items[cursor], streamOutput, right, height, now),
	)
	return style.Fit(result, width, height)
}

// ProjectList renders the responsive table of discovered Compose projects.
func ProjectList(s style.Sheet, items []engine.ComposeProject, cursor int, filter string, width, height int, now time.Time) string {
	lines := []string{s.Muted("filter " + emptyDash(filter))}
	capacity := max(1, height-tokens.PanelFrame-tokens.InsetY*2-1)
	start := max(0, cursor-capacity+1)
	end := min(len(items), start+capacity)
	g := tokens.Icons(s.ASCII)

	for n := start; n < end; n++ {
		p := items[n]
		marker := " "
		if n == cursor {
			marker = g.Cursor
		}
		statusPill := components.Pill(s, p.Status)
		svcSummary := fmt.Sprintf("%d svcs", len(p.Services))
		nameWidth := max(10, width-28)
		name := padCell(style.Clean(p.Name), nameWidth)
		line := fmt.Sprintf("%s %s %s %8s", marker, name, statusPill, svcSummary)
		if n == cursor {
			line = s.Text(line, s.Palette.Accent)
		}
		lines = append(lines, line)
	}

	if len(items) == 0 {
		return s.Panel("PROJECTS  0", components.Empty(s, "No Docker Compose projects found.", "Projects are auto-discovered from container labels.", max(1, width-tokens.PanelFrame-tokens.InsetX*2)), width, height, true)
	}
	return s.Panel(fmt.Sprintf("PROJECTS  %d", len(items)), strings.Join(lines, "\n"), width, height, true)
}

// ProjectDetail renders the services, working directory, and live compose stream logs.
func ProjectDetail(s style.Sheet, project engine.ComposeProject, streamOutput []string, width, height int, now time.Time) string {
	var lines []string
	lines = append(lines,
		style.Clean(project.Name)+"  "+components.Pill(s, project.Status),
		"",
	)

	if project.WorkingDir != "" {
		lines = append(lines, s.Heading("DIRECTORY"), style.Clean(project.WorkingDir), "")
	}
	if len(project.ConfigFiles) > 0 {
		lines = append(lines, s.Heading("CONFIG FILES"), style.Clean(strings.Join(project.ConfigFiles, ", ")), "")
	}

	lines = append(lines, s.Heading(fmt.Sprintf("SERVICES (%d)", len(project.Services))))
	if len(project.Containers) == 0 {
		lines = append(lines, "No containers associated with this project.")
	} else {
		for _, c := range project.Containers {
			svcName := c.Service
			if svcName == "" {
				svcName = "-"
			}
			ports := ""
			if len(c.Ports) > 0 {
				ports = " · " + c.Ports[0]
			}
			line := fmt.Sprintf("%-14s %s  %s%s",
				style.Clean(svcName),
				style.Clean(c.Name),
				components.Pill(s, c.State),
				ports,
			)
			lines = append(lines, line)
		}
	}

	if len(streamOutput) > 0 {
		lines = append(lines, "", s.Heading("COMPOSE STREAM LOGS"))
		tailStart := max(0, len(streamOutput)-10)
		for _, logLine := range streamOutput[tailStart:] {
			lines = append(lines, style.Fit(style.Clean(logLine), max(10, width-tokens.InsetX*2-tokens.PanelFrame), 1))
		}
	} else {
		lines = append(lines, "", s.Muted("u up · d down · r restart · / filter"))
	}

	body := strings.Join(lines, "\n")
	return s.Panel("COMPOSE PROJECT", body, width, height, false)
}
