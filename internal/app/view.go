package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/components"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/views"
)

// View composes one bounded frame and opts into the alternate screen.
func (m *Model) View() tea.View {
	v := tea.NewView(m.Render())
	v.AltScreen = true
	v.WindowTitle = "Lenger · Anchor your containers"
	if m.config.UI.Mouse {
		v.MouseMode = tea.MouseModeCellMotion
	}
	return v
}

// Render returns the same screen used by the interactive app and offline gallery.
func (m *Model) Render() string {
	w, h := m.width, m.height
	s := m.sheet
	if w < tokens.MinWidth || h < tokens.MinHeight {
		return style.Fit(fmt.Sprintf("Lenger needs at least %d×%d cells.\nCurrent size: %d×%d. Resize or press q to quit.", tokens.MinWidth, tokens.MinHeight, w, h), w, h)
	}
	status, state := "docker · local", "ok"
	if m.fixture {
		status, state = "fixture engine · offline", "muted"
	}
	if m.loading {
		status, state = "connecting…", "loading"
	}
	if m.err != nil {
		status, state = "docker · unavailable", "error"
	}
	brand := s.Gradient(tokens.Icons(s.ASCII).Anchor+" LENGER") + "   " + s.Muted("Anchor your containers.")
	chip := components.StatusChip(s, status, state)
	gap := max(1, w-lipgloss.Width(brand)-lipgloss.Width(chip))
	header := brand + strings.Repeat(" ", gap) + chip + "\n" + components.Tabs(s, m.tab, w) + "\n"
	height := h - tokens.HeaderHeight - tokens.FooterHeight
	var body string
	switch {
	case m.loading:
		body = s.Panel("CONNECTING", "Loading local images and containers…\nThe interface remains available while Docker responds.", w, height, false)
	case m.err != nil:
		body = s.Panel("DOCKER NEEDS ATTENTION", m.err.Message+"\n\n"+m.err.Suggestion+"\n\nR retry   q quit\nFor an offline preview: LENGER_ENGINE=fake lenger", w, height, true)
	case m.tab == 0:
		items := m.containers()
		var inspect *engine.ContainerInspection
		if len(items) > 0 {
			if i, ok := m.containerIns[items[m.containerCursor].ID]; ok {
				inspect = &i
			}
		}
		body = views.Containers(s, items, m.containerCursor, m.selectedContain, m.containerFilter, inspect, m.containerDetail, m.containerRaw, m.stats, m.statCPU, m.statHistory, w, height, m.clock)
	case m.tab == 1:
		items := m.items()
		var inspect *engine.Inspection
		if len(items) > 0 {
			if i, ok := m.inspections[items[m.cursor].ID]; ok {
				inspect = &i
			}
		}
		body = views.Images(s, items, m.cursor, m.selected, inspect, m.detail, []string{"name", "size", "age"}[m.sortMode], m.filter, w, height, m.clock)
	case m.tab == 2:
		body = views.Volumes(s, m.volumes(), m.volumeCursor, m.volumeFilter, w, height, m.clock)
	case m.tab == 3:
		body = views.Networks(s, m.networks(), m.networkCursor, m.networkFilter, w, height, m.clock)
	default:
		items := m.containers()
		var inspect *engine.ContainerInspection
		if len(items) > 0 {
			if i, ok := m.containerIns[items[m.containerCursor].ID]; ok {
				inspect = &i
			}
		}
		body = views.Containers(s, items, m.containerCursor, m.selectedContain, m.containerFilter, inspect, m.containerDetail, m.containerRaw, m.stats, m.statCPU, m.statHistory, w, height, m.clock)
	}
	footer := "? help   : palette   c cleanup   p pull   / filter   J jobs   q quit"
	switch m.tab {
	case 0:
		footer = "x start  s stop  r restart  p pause  d remove  l logs  e exec  i inspect  / filter  c cleanup"
	case 1:
		footer = "p pull  r run  i inspect  d remove  o sort  / filter  space select  c cleanup"
	case 2:
		footer = "d remove unused  / filter  c cleanup  R refresh"
	case 3:
		footer = "c create  d remove  / filter  R refresh  q quit"
	}
	running := 0
	for _, j := range m.jobList {
		if j.State == "running" {
			running++
		}
	}
	footer += fmt.Sprintf("   %s %d jobs", tokens.Icons(s.ASCII).Pull, running)
	second := m.toast
	if second == "" {
		second = "Local daemon. No accounts. No telemetry."
	}
	base := style.Fit(header+style.Fit(body, w, height)+"\n"+style.Fit(s.Muted(footer)+"\n"+s.Muted(style.Clean(second)), w, tokens.FooterHeight), w, h)
	if m.overlay != "" {
		title, content := m.overlayContent()
		return components.Modal(s, base, title, content, w, h)
	}
	return base
}

func (m *Model) overlayContent() (string, string) {
	s := m.sheet
	switch m.overlay {
	case "help":
		return "Keyboard chart", components.Help()
	case "pull-input":
		return "Pull image", m.input.View() + "\n\nUse repository:tag or repository@sha256:digest.\nDocker's configured credential helpers are respected.\n\nenter pull   esc back"
	case "filter":
		title, scope := "Filter images", "repository, tag, or image ID"
		switch m.tab {
		case 0:
			title, scope = "Filter containers", "name, image, state, project, or ID"
		case 2:
			title, scope = "Filter volumes", "name, driver, or mountpoint"
		case 3:
			title, scope = "Filter networks", "name, driver, scope, or ID"
		}
		return title, m.input.View() + "\n\nMatches " + scope + ".\nenter apply   esc back"
	case "run":
		return "Run image", m.runOverlay()
	case "compose-action":
		return "Compose Project Action", m.composeActionOverlay()
	case "create-network":
		return "Create Docker Network", m.createNetworkOverlay()
	case "remove-network":
		return "Confirm Network Removal", m.removeNetworkOverlay()
	case "container-action":
		return "Confirm container action", m.containerActionOverlay()
	case "logs":
		return "Container logs", m.logsOverlay()
	case "log-filter":
		return "Filter logs", m.input.View() + "\n\nCase-insensitive regular expression.\nenter apply   esc back"
	case "cleanup":
		return "Smart cleanup preview", m.cleanupOverlay()
	case "cleanup-confirm":
		return "Confirm cleanup", m.cleanupConfirmOverlay()
	case "remove-volume":
		return "Confirm volume removal", m.removeVolumeOverlay()
	case "exec":
		return "Exec shell", m.execOverlay()
	case "registry-search":
		return "Registry Search & Tag Browser", m.registrySearchOverlay()
	case "tag-image":
		return "Tag Image", m.tagImageOverlay()
	case "push-image":
		return "Push Image", m.pushImageOverlay()
	case "export-image":
		return "Export Image Tarball", m.exportImageOverlay()
	case "import-image":
		return "Import Image Tarball", m.importImageOverlay()
	case "daemon-mirrors":
		return "Registry Mirrors Configuration", m.daemonMirrorsOverlay()
	case "palette":
		lines := []string{m.input.View(), ""}
		results := m.paletteActions()
		start := max(0, m.paletteCursor-7)
		for n := start; n < min(len(results), start+8); n++ {
			prefix := "  "
			if n == m.paletteCursor {
				prefix = tokens.Icons(s.ASCII).Cursor + " "
			}
			lines = append(lines, prefix+style.Clean(results[n].name))
		}
		if len(results) > 0 {
			index := min(m.paletteCursor, len(results)-1)
			lines = append(lines, "", s.Muted(style.Clean(results[index].command)))
		}
		if len(results) == 0 {
			lines = append(lines, "No matching actions or images.")
		}
		lines = append(lines, "", "↑/↓ choose   enter run   esc close")
		return "Command palette", strings.Join(lines, "\n")
	case "pull":
		for _, j := range m.jobList {
			if j.ID == m.jobID {
				return "Pull image", components.Pull(s, j, min(m.width, tokens.ModalWidth)-tokens.InsetX*2-tokens.PanelFrame)
			}
		}
		return "Pull image", "Starting pull…\n\nesc hide   J jobs"
	case "jobs":
		lines := []string{}
		for _, j := range m.jobList {
			lines = append(lines, fmt.Sprintf("#%d %s %3.0f%% %s", j.ID, style.Clean(j.Target), j.Fraction()*100, components.Pill(s, j.State)))
		}
		if len(lines) == 0 {
			lines = append(lines, "No jobs yet. Press p to pull an image.")
		}
		return "Session jobs", strings.Join(lines, "\n") + "\n\nenter open most recent   esc close"
	case "remove":
		ids := m.removalIDs()
		lines := []string{"Remove the following local images?", ""}
		for _, id := range ids {
			for _, i := range m.store.Images {
				if i.ID == id {
					lines = append(lines, style.Clean(i.Name())+"  "+components.Bytes(i.Size))
					for _, c := range i.UsedBy {
						lines = append(lines, "  IN USE by "+style.Clean(c.Name)+" — removal will be refused")
					}
				}
			}
		}
		lines = append(lines, "", "No force flag. Containers and volumes are not removed.", "Equivalent: docker image rm "+strings.Join(ids, " "), "", "y confirm   n / esc cancel")
		return "Confirm image removal", strings.Join(lines, "\n")
	}
	return "Lenger", "esc close"
}
