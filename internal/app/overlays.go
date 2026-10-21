package app

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Mersad-Moghaddam/lenger/internal/core/services"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/components"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
)

func (m *Model) runPreview() (engine.RunSpec, error) {
	form := m.runForm
	if m.runField != 4 {
		form.set(m.runField, strings.TrimSpace(m.input.Value()))
	}
	spec, err := form.spec(m.runImage)
	if err != nil {
		return spec, err
	}
	return services.NormalizeRunSpec(spec)
}

func (m *Model) runOverlay() string {
	s := m.sheet
	lines := []string{s.Heading("IMAGE"), "  " + style.Clean(m.runImage), ""}
	for n, label := range runLabels {
		marker := "  "
		if n == m.runField {
			marker = tokens.Icons(s.ASCII).Cursor + " "
		}
		value := m.runForm.value(n)
		if n == m.runField && n != 4 {
			value = m.input.View()
		} else if n == m.runField {
			value = "‹ " + value + " ›"
		} else if strings.TrimSpace(value) == "" {
			value = "—"
		}
		lines = append(lines, fmt.Sprintf("%s%-9s %s", marker, label, style.Clean(value)))
	}
	spec, err := m.runPreview()
	if err != nil {
		lines = append(lines, "", s.Text("Form error: "+style.Clean(err.Error()), s.Palette.Danger))
	} else {
		lines = append(lines, "", s.Heading("EQUIVALENT COMMAND"), style.Clean(services.RunCommand(spec)))
	}
	if m.runErr != "" {
		lines = append(lines, "", s.Text(style.Clean(m.runErr), s.Palette.Danger))
	}
	if m.runBusy {
		lines = append(lines, "", s.Muted("Launching container…"))
	}
	lines = append(lines, "", "tab/↑↓ field   ←/→ restart   enter run   esc cancel")
	lines = append(lines, s.Muted("The preview is generated from this form; no shell is involved."))
	return strings.Join(lines, "\n")
}

func (m *Model) containerActionOverlay() string {
	s := m.sheet
	lines := []string{s.Heading(strings.ToUpper(string(m.actionName))), ""}
	command := "docker " + string(m.actionName)
	if m.actionName == engine.ActionRemove {
		command = "docker rm"
	}
	for _, id := range m.actionIDs {
		name := id
		state := ""
		if c, ok := m.containerByID(id); ok {
			name, state = c.Name, c.State
		}
		lines = append(lines, style.Clean(name)+"  "+components.Pill(s, state))
		command += " " + id
	}
	lines = append(lines, "")
	if m.actionName == engine.ActionRemove {
		lines = append(lines, "No force. Running containers must be stopped first.")
		lines = append(lines, "Named and anonymous volumes are preserved.")
	}
	if m.actionName == engine.ActionKill {
		lines = append(lines, s.Text("Kill sends SIGKILL immediately; unsaved process state is lost.", s.Palette.Danger))
	}
	lines = append(lines, "", s.Heading("EQUIVALENT COMMAND"), style.Clean(command), "", "y confirm   n / esc cancel")
	return strings.Join(lines, "\n")
}

func (m *Model) logsOverlay() string {
	s := m.sheet
	contentWidth := max(20, min(tokens.ModalWidth, m.width-tokens.Gap*2)-tokens.PanelFrame-tokens.InsetX*2)
	bodyLimit := max(8, m.height-7)
	name := m.logsID
	for _, c := range m.store.Containers {
		if c.ID == m.logsID {
			name = c.Name
			break
		}
	}
	state := "snapshot"
	if m.logsPaused {
		state = "paused"
	} else if m.logsFollow {
		state = "following"
	}
	header := fmt.Sprintf("%s  %s  %s", style.Clean(name), components.Pill(s, state), fmt.Sprintf("%d lines", len(m.logs)))
	lines := []string{s.Muted(header)}
	flags := []string{}
	if m.logsTimestamps {
		flags = append(flags, "timestamps")
	}
	if m.logsWrap {
		flags = append(flags, "wrap")
	}
	if m.logsFilter != "" {
		flags = append(flags, "filter /"+style.Clean(m.logsFilter)+"/")
	}
	if len(flags) > 0 {
		lines = append(lines, s.Muted(strings.Join(flags, " · ")))
	}
	if m.logsNotice != "" {
		lines = append(lines, s.Text(style.Clean(m.logsNotice), s.Palette.Warning))
	}
	budget := max(1, bodyLimit-len(lines)-2)
	rendered := make([]string, 0, budget)
	for _, line := range m.logLines(50_000) {
		if m.logsWrap {
			rendered = append(rendered, wrapText(line, contentWidth)...)
		} else {
			rendered = append(rendered, style.Fit(line, contentWidth, 1))
		}
	}
	if len(rendered) == 0 {
		rendered = []string{s.Muted("No log lines yet.")}
	}
	if len(rendered) > budget {
		rendered = rendered[len(rendered)-budget:]
	}
	lines = append(lines, rendered...)
	lines = append(lines, "", "p pause  f follow  t timestamps  w wrap  / filter  c copy  s save  esc close")
	return strings.Join(lines, "\n")
}

func (m *Model) cleanupOverlay() string {
	s := m.sheet
	lines := []string{}
	if m.cleanupLoading && !m.cleanupLoaded {
		lines = append(lines, "Reading disk usage and listing exact candidates…", "", "No mutation starts before this preview is confirmed.")
		return strings.Join(lines, "\n")
	}
	if m.cleanupErr != nil {
		lines = append(lines, s.Text(m.cleanupErr.Message, s.Palette.Danger), m.cleanupErr.Suggestion, "", "R retry   esc close")
		return strings.Join(lines, "\n")
	}
	lines = append(lines, s.Heading("CLEANUP CHECKLIST"), "Preview exactly what will be removed, then confirm.", "")
	groups := map[services.CleanupKind]services.CleanupGroup{}
	for _, group := range m.cleanupPlan.Groups {
		groups[group.Kind] = group
	}
	for n, kind := range services.CleanupKinds() {
		group := groups[kind]
		marker := "  "
		if n == m.cleanupCursor {
			marker = tokens.Icons(s.ASCII).Cursor + " "
		}
		check := "[ ]"
		if m.cleanupSelected[kind] {
			check = "[x]"
		}
		lines = append(lines, fmt.Sprintf("%s%s %-20s %3d  %9s", marker, check, kind, group.Count, components.Bytes(group.Size)))
	}
	lines = append(lines, "", s.Heading("SELECTED"), m.cleanupSummary())
	items := m.selectedCleanupItems()
	if len(items) > 0 {
		maxItems := max(1, m.height-21)
		for n, item := range items {
			if n >= maxItems {
				lines = append(lines, fmt.Sprintf("… %d more", len(items)-n))
				break
			}
			lines = append(lines, "  "+style.Clean(item.Name))
		}
	} else {
		lines = append(lines, "  No candidates in the selected categories.")
	}
	lines = append(lines, "", "space toggle  enter confirm  R reload  esc close")
	if m.cleanupSelected[services.CleanupUnusedVolumes] {
		lines = append(lines, s.Text("Volumes require typing the word volumes on the next screen.", s.Palette.Warning))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) cleanupConfirmOverlay() string {
	s := m.sheet
	lines := []string{s.Heading("CONFIRM CLEANUP"), "", m.cleanupSummary(), ""}
	items := m.selectedCleanupItems()
	maxItems := max(1, m.height-18)
	for n, item := range items {
		if n >= maxItems {
			lines = append(lines, fmt.Sprintf("… %d more", len(items)-n))
			break
		}
		lines = append(lines, "  "+style.Clean(item.Name)+"  "+components.Bytes(item.Size))
	}
	lines = append(lines, "")
	if m.cleanupSelected[services.CleanupUnusedVolumes] {
		lines = append(lines, s.Text("Volume data cannot be recovered.", s.Palette.Danger))
		lines = append(lines, "Type the word volumes to confirm:", m.input.View())
		lines = append(lines, "", "enter confirm   esc back")
	} else {
		lines = append(lines, "This action is non-forced and revalidates every candidate.", "", "enter confirm   esc back")
	}
	if m.cleanupLoading {
		lines = append(lines, "", s.Muted("Removing selected candidates…"))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) removeVolumeOverlay() string {
	s := m.sheet
	lines := []string{
		s.Heading("REMOVE VOLUME"), "",
		style.Clean(m.removeVolume), "",
		s.Text("Volume contents cannot be recovered.", s.Palette.Danger),
		"No force. The daemon refuses removal while a container mounts it.",
		"", "Type volumes to confirm:", m.input.View(), "",
		"enter remove   esc cancel",
	}
	return strings.Join(lines, "\n")
}

func (m *Model) execOverlay() string {
	s := m.sheet
	name := m.execContainer
	for _, c := range m.store.Containers {
		if c.ID == m.execContainer {
			name = c.Name
			break
		}
	}
	lines := []string{s.Heading("EXEC SHELL"), style.Clean(name), "", "The TUI suspends while docker exec runs; it restores on exit.", ""}
	for n, shell := range execShells {
		marker := "  "
		if n == m.execShell {
			marker = tokens.Icons(s.ASCII).Cursor + " "
		}
		lines = append(lines, marker+shell)
	}
	lines = append(lines, "", "j/k choose   enter run   esc cancel")
	return strings.Join(lines, "\n")
}

func (m *Model) selectedCleanupItems() []services.CleanupItem {
	var out []services.CleanupItem
	for _, item := range m.cleanupPlan.Items {
		if m.cleanupSelected[item.Kind] {
			out = append(out, item)
		}
	}
	return out
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	if lipgloss.Width(text) <= width {
		return []string{text}
	}
	var lines []string
	var current strings.Builder
	for _, word := range strings.Fields(text) {
		if current.Len() > 0 && lipgloss.Width(current.String())+1+lipgloss.Width(word) > width {
			lines = append(lines, current.String())
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteByte(' ')
		}
		if lipgloss.Width(word) > width {
			if current.Len() > 0 {
				lines = append(lines, current.String())
				current.Reset()
			}
			runes := []rune(word)
			for len(runes) > width {
				lines = append(lines, string(runes[:width]))
				runes = runes[width:]
			}
			current.WriteString(string(runes))
			continue
		}
		current.WriteString(word)
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}
