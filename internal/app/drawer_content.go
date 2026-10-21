package app

import (
	"fmt"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/views/inspector"
	"github.com/charmbracelet/x/ansi"
	"strings"
)

// wrapVisual retains every sanitized visual cell before viewport clipping.
func wrapVisual(text string, w int) []string {
	var rows []string
	for _, line := range strings.Split(text, "\n") {
		rows = append(rows, strings.Split(ansi.Hardwrap(strings.TrimRight(line, " "), max(1, w), true), "\n")...)
	}
	return rows
}

// envVisualRows is shared by rendering, keyboard visibility and mouse hit tests.
// A wrapped continuation maps to its owning control, never a neighboring secret.
func (m *Model) envVisualRows(w int) ([]string, []int) {
	var rows []string
	var owners []int
	add := func(text string, owner int) {
		for _, line := range wrapVisual(text, w) {
			rows = append(rows, line)
			owners = append(owners, owner)
		}
	}
	add("Environment · Space reveal · u reveal all · 60s idle remask", -1)
	for i, item := range m.drawerEnv.EnvItems {
		value := item.String()
		if item.Revealed {
			value = style.Clean(item.Key) + "=" + style.Clean(item.Value)
		}
		prefix := "  "
		if m.focus.Area == FocusDrawerControls && m.focus.ControlIdx == i {
			prefix = "> "
		}
		add(prefix+value, i)
	}
	add("Ports · Enter preview", -1)
	for i, p := range m.containerIns[m.drawerID].Ports {
		idx := len(m.drawerEnv.EnvItems) + i
		prefix := "  "
		if m.focus.Area == FocusDrawerControls && m.focus.ControlIdx == idx {
			prefix = "> "
		}
		add(prefix+style.Clean(p), idx)
		preview, err := inspector.PreviewPort(p)
		if err == nil {
			add("Confirm before opening: "+preview.URL, -1)
			if preview.Warning != "" {
				add(preview.Warning, -1)
			}
		} else {
			add("Display only: no safe HTTP preview", -1)
		}
	}
	return rows, owners
}

func (m *Model) healthControls() string {
	labels := []string{"Previous instance logs", "Restart container", "Healthcheck command / evidence"}
	var lines []string
	for i, label := range labels {
		prefix := "  "
		if m.focus.Area == FocusDrawerControls && m.focus.ControlIdx == i {
			prefix = "> "
		}
		w, _ := m.drawerViewportSize()
		lines = append(lines, style.Fit(fmt.Sprintf("%s%d. %s", prefix, i+1, label), w, 1))
	}
	return strings.Join(lines, "\n")
}
