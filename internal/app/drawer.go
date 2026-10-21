package app

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Mersad-Moghaddam/lenger/internal/config"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/chart"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/masterdetail"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/views"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/views/inspector"
)

type drawerInspectMsg struct {
	gen  uint64
	id   string
	data engine.ContainerInspection
	err  error
}
type timedMetric struct {
	at  time.Time
	cpu float64
}

func (m *Model) healthEvidence(c engine.ContainerInspection) inspector.HealthEvidence {
	d := c.DiagnosticEvidence()
	e := inspector.HealthEvidence{InspectedAt: c.InspectedAt, ObservedAt: m.peakAt[c.ID], SessionStarted: m.sessionStarted, ConfiguredMemory: d.ConfiguredMemory, DaemonError: d.DaemonError, DaemonErrorKnown: d.DaemonErrorKnown, ProbeExecutionObserved: d.ProbeExecutionObserved, HealthHistory: d.HealthHistory, HealthHistoryKnown: d.HealthHistoryKnown}
	if h := d.HealthConfig; h != nil {
		e.HealthConfig = &inspector.HealthCheckEvidence{Command: h.Command, Interval: h.Interval, Timeout: h.Timeout, StartPeriod: h.StartPeriod, Retries: h.Retries}
	}
	return e
}

func (m *Model) metricWindows(w int) string {
	at := time.Now()
	var minute, five []float64
	for _, s := range m.metricSamples[m.drawerID] {
		if !s.at.After(at) && !s.at.Before(at.Add(-5*time.Minute)) {
			five = append(five, s.cpu)
			if !s.at.Before(at.Add(-time.Minute)) {
				minute = append(minute, s.cpu)
			}
		}
	}
	started := "unavailable"
	if t := m.metricStarted[m.drawerID]; !t.IsZero() {
		started = t.Format(time.RFC3339)
	}
	return fmt.Sprintf("Collection started %s\nCPU 60s %s\nCPU 5min %s\nPartial session windows; no pre-session history.\n", started, chart.Sparkline(minute, max(1, w-9), m.sheet.ASCII), chart.Sparkline(five, max(1, w-10), m.sheet.ASCII))
}

type drawerStatsMsg struct {
	request uint64
	gen     uint64
	id      string
	data    engine.ContainerStats
	err     error
}
type drawerStatsTick struct {
	gen uint64
	id  string
}
type drawerMaskTick struct{ gen uint64 }
type drawerRevealMsg struct {
	gen   uint64
	id    string
	items []inspector.EnvItem
	err   error
}

func (m *Model) closeDrawer() {
	if sample, ok := m.stats[m.drawerID]; ok && sample.State == engine.TelemetryActive {
		sample.State = engine.TelemetrySampled
		m.stats[m.drawerID] = sample
	}
	if m.drawerStatsCancel != nil {
		m.drawerStatsCancel()
		m.drawerStatsCancel = nil
	}
	m.healthCommandScroll = 0
	if m.overlay == "health-command" {
		m.overlay = ""
	}
	m.healthCommandRevealed = false
	if m.overlay == "port-preview" {
		m.overlay = ""
	}
	m.portPreview = inspector.PortPreview{}
	m.portMapping = ""
	m.portContainer = ""
	m.drawerEnv.RemaskAll()
	if m.drawerCancel != nil {
		m.drawerCancel()
		m.drawerCancel = nil
	}
	m.stopLogs()
	m.logsGen++
	m.drawerGen++
	m.statsLoading = false
	m.statsTicking = false
	m.drawerEnv = inspector.EnvViewModel{}
	m.focus.CloseDrawer()
	m.containerDetail = false
}

func (m *Model) dragDivider(x int) tea.Cmd {
	l := masterdetail.ComputeLayout(m.width, m.height, true, m.drawerRatio)
	if l.Mode != masterdetail.ModeSplit {
		return nil
	}
	next := float64(x) / float64(max(1, m.width-1))
	l.AdjustRatio(next - l.Ratio)
	m.drawerRatio = l.Ratio
	return m.persistRatio()
}
func (m *Model) drawerMouse(x, y int) (bool, tea.Cmd) {
	if y < tokens.HeaderHeight {
		return false, nil
	}
	if y >= m.height-tokens.FooterHeight {
		return true, nil
	}
	l := masterdetail.ComputeLayout(m.width, m.height, m.focus.DrawerOpen, m.drawerRatio)
	if m.focus.DrawerOpen && l.Mode == masterdetail.ModeSplit && x == l.ListWidth {
		m.dividerDragging = true
		return true, nil
	}
	inDrawer := m.focus.DrawerOpen && (m.focus.Zoomed || l.Mode == masterdetail.ModeOverlay || x > l.ListWidth)
	row := y - tokens.HeaderHeight
	if inDrawer {
		headerRows := 1
		dw := l.InspectorWidth
		if m.focus.Zoomed {
			dw = m.width
		}
		if dw < 100 {
			headerRows = 5
		}
		if l.Mode == masterdetail.ModeOverlay || m.focus.Zoomed {
			if row == 0 {
				m.closeDrawer()
				return true, nil
			}
			row--
		}
		if row < headerRows {
			oldTab := m.focus.ActiveTab
			if headerRows == 5 {
				m.focus.ActiveTab = max(0, min(4, row))
			} else {
				offset := x
				if l.Mode == masterdetail.ModeSplit {
					offset -= l.ListWidth + 1
				}
				labels := []string{"Health & Diagnostics", "Live Logs", "Metrics", "Env & Ports", "Timeline"}
				if c := m.containerIns[m.drawerID]; c.ExitCode != 0 || c.State == "restarting" || c.Health == "unhealthy" {
					labels[0] += " (ERR)"
				}
				at := 0
				for i, label := range labels {
					if i == oldTab {
						label = "[" + label + "]"
					}
					at += lipgloss.Width(label) + 3
					if offset < at {
						m.focus.ActiveTab = i
						break
					}
				}
			}
			m.focus.Area = FocusDrawerTabs
			m.drawerOffsets[oldTab] = m.drawerScroll
			m.drawerScroll = m.drawerOffsets[m.focus.ActiveTab]
			m.drawerLatch = false
			m.normalizeDrawerControls()
			return true, nil
		}
		m.focus.Area = FocusDrawerControls
		if m.focus.ActiveTab == 3 {
			m.drawerEnv.RecordActivity(time.Now())
		}
		if m.focus.ActiveTab == 3 {
			_, owners := m.envVisualRows(dw)
			at := row - headerRows + m.drawerScroll
			if at >= 0 && at < len(owners) && owners[at] >= 0 {
				m.focus.ControlIdx = owners[at]
				return m.drawerKey("enter")
			}
			return true, nil
		}
		if m.focus.ActiveTab == 0 && row-headerRows < 3 {
			m.focus.ControlIdx = row - headerRows
			return m.drawerKey("enter")
		}
		return true, nil
	}
	if m.grouped {
		h := m.height - tokens.HeaderHeight - tokens.FooterHeight
		if row < 1 || row >= h {
			return true, nil
		}
		rows := m.groupedRows()
		index := m.groupWindowStart(h) + row - 1
		if index < 0 || index >= len(rows) {
			return true, nil
		}
		m.focus.Area = FocusList
		m.groupCursor = index
		target := rows[index]
		if target.id == "" {
			m.closeDrawer()
			m.groupCollapsed[target.project] = !m.groupCollapsed[target.project]
			return true, nil
		}
		for i, c := range m.containers() {
			if c.ID == target.id {
				m.containerCursor = i
				return true, m.openDrawer()
			}
		}
		return true, nil
	}
	if y >= tokens.HeaderHeight+tokens.PanelFrame+1 {
		capacity := max(1, (m.height-tokens.HeaderHeight-tokens.FooterHeight-tokens.PanelFrame-tokens.InsetY*2-1)/2)
		start := max(0, m.containerCursor-capacity+1)
		index := start + (y-tokens.HeaderHeight-tokens.PanelFrame-1)/2
		if index >= 0 && index < len(m.containers()) {
			m.containerCursor = index
			return true, m.openDrawer()
		}
	}
	return true, nil
}

func (m *Model) openDrawer() tea.Cmd {
	items := m.containers()
	if len(items) == 0 {
		m.closeDrawer()
		return nil
	}
	c := items[m.containerCursor]
	if m.grouped {
		for i, row := range m.groupedRows() {
			if row.id == c.ID {
				m.groupCursor = i
				break
			}
		}
	}
	if m.focus.DrawerOpen && m.drawerID == c.ID {
		return nil
	}
	wasOpen := m.focus.DrawerOpen
	active := m.focus.ActiveTab
	m.closeDrawer()
	m.drawerID = c.ID
	m.drawerCtx, m.drawerCancel = context.WithCancel(m.ctx)
	initial := 0
	if c.Running() {
		initial = 1
	}
	if wasOpen {
		initial = active
	}
	m.focus.OpenDrawer(initial)
	m.drawerLatch = !wasOpen
	m.containerDetail = true
	m.drawerScroll = 0
	m.drawerOffsets = [5]int{}
	m.drawerInspected = false
	m.drawerEnvFetched = false
	m.pendingPrevious = false
	m.statsError = false
	m.drawerLogs = inspector.LogsViewModel{Follow: true}
	m.logsID = c.ID
	delete(m.stats, c.ID)
	delete(m.statCPU, c.ID)
	m.statHistory[c.ID] = nil
	m.normalizeDrawerControls()
	gen, id, ctx, service := m.drawerGen, c.ID, m.drawerCtx, m.service
	inspect := func() tea.Msg {
		data, err := service.InspectContainer(ctx, id)
		return drawerInspectMsg{gen, id, data, err}
	}
	return tea.Batch(inspect, m.streamDrawerLogs(false), m.loadStats(), tea.Tick(time.Second, func(time.Time) tea.Msg { return drawerMaskTick{gen} }))
}

func (m *Model) rebindDrawerSelection() tea.Cmd {
	if !m.focus.DrawerOpen {
		return nil
	}
	items := m.containers()
	if len(items) == 0 {
		m.closeDrawer()
		return nil
	}
	if items[m.containerCursor].ID != m.drawerID {
		return m.openDrawer()
	}
	return nil
}

func (m *Model) requestPreviousLogs() tea.Cmd {
	m.drawerOffsets[m.focus.ActiveTab] = m.drawerScroll
	m.focus.ActiveTab = 1
	m.drawerScroll = m.drawerOffsets[1]
	m.drawerLatch = false
	m.normalizeDrawerControls()
	if !m.drawerInspected {
		m.pendingPrevious = true
		return nil
	}
	return m.streamDrawerLogs(true)
}

func (m *Model) normalizeDrawerControls() {
	n := 1
	switch m.focus.ActiveTab {
	case 0:
		n = 3
	case 3:
		n = len(m.drawerEnv.EnvItems)
		if i, ok := m.containerIns[m.drawerID]; ok {
			n += len(i.Ports)
		}
	}
	m.focus.ControlMax = n
	m.focus.ControlIdx = max(0, min(m.focus.ControlIdx, n-1))
}

func (m *Model) drawerKey(k string) (bool, tea.Cmd) {
	if m.tab != 0 || !m.focus.DrawerOpen {
		return false, nil
	}
	before := m.focus.ActiveTab
	if m.focus.ActiveTab == 3 {
		m.drawerEnv.RecordActivity(time.Now())
	}
	if m.focus.HandleKey(k) {
		if !m.focus.DrawerOpen {
			m.closeDrawer()
		}
		if before != m.focus.ActiveTab {
			m.drawerOffsets[before] = m.drawerScroll
			m.drawerScroll = m.drawerOffsets[m.focus.ActiveTab]
			m.drawerLatch = false
		}
		m.normalizeDrawerControls()
		m.ensureEnvControlVisible()
		return true, nil
	}
	if k == "[" || k == "]" {
		l := masterdetail.ComputeLayout(m.width, m.height, true, m.drawerRatio)
		d := .05
		if k == "[" {
			d = -d
		}
		l.AdjustRatio(d)
		m.drawerRatio = l.Ratio
		return true, m.persistRatio()
	}
	if m.focus.Area == FocusList {
		if k == "down" || k == "j" || k == "up" || k == "k" {
			d := 1
			if k == "up" || k == "k" {
				d = -1
			}
			old := m.containerCursor
			m.moveCursor(k, d)
			if old != m.containerCursor {
				return true, m.openDrawer()
			}
			return true, nil
		}
		return false, nil
	}
	if m.focus.Area == FocusDrawerTabs && (k == "enter" || k == "space" || k == " ") {
		return true, nil
	}
	if m.focus.ActiveTab == 0 && k == "c" {
		m.healthCommandRevealed = false
		m.overlay = "health-command"
		return true, nil
	}
	if m.focus.ActiveTab == 0 && k == "enter" && m.focus.Area == FocusDrawerControls {
		switch m.focus.ControlIdx {
		case 0:
			return true, m.requestPreviousLogs()
		case 1:
			if _, ok := m.containerByID(m.drawerID); !ok {
				return true, m.notify("Inspected container is no longer available.")
			}
			m.actionIDs = []string{m.drawerID}
			m.actionName = engine.ActionRestart
			m.overlay = "container-action"
			return true, nil
		case 2:
			m.overlay = "health-command"
			return true, nil
		}
	}
	if m.focus.ActiveTab == 3 {
		m.drawerEnv.RecordActivity(time.Now())
		if k == "tab" || k == "shift+tab" {
			m.ensureEnvControlVisible()
		}
		if k == "u" {
			m.drawerEnv.RevealAll(time.Now())
			return true, nil
		}
		if (k == "space" || k == " " || k == "enter") && m.focus.Area == FocusDrawerControls {
			idx := m.focus.ControlIdx
			if idx < len(m.drawerEnv.EnvItems) {
				m.drawerEnv.RevealItem(idx, time.Now())
				return true, nil
			}
			ports := m.containerIns[m.drawerID].Ports
			p := idx - len(m.drawerEnv.EnvItems)
			if p >= 0 && p < len(ports) {
				preview, err := inspector.PreviewPort(ports[p])
				if err != nil {
					return true, m.notify("Port has no safe browser preview.")
				}
				m.portPreview = preview
				m.portMapping = ports[p]
				m.portContainer = m.drawerID
				m.overlay = "port-preview"
			}
			return true, nil
		}
	}
	if k == "up" || k == "k" || k == "down" || k == "j" {
		d := 1
		if k == "up" || k == "k" {
			d = -1
		}
		if m.focus.ActiveTab == 1 {
			m.drawerLogs.Scroll(d)
		} else {
			m.drawerScroll = max(0, m.drawerScroll+d)
		}
		return true, nil
	}
	if m.focus.ActiveTab == 1 {
		switch k {
		case "y":
			w, h := m.drawerViewportSize()
			visible := m.drawerLogs.Render(m.sheet, w, h)
			return true, tea.Batch(tea.SetClipboard(visible), m.notify("Copied visible log viewport."))
		case "f", "G", "end":
			m.drawerLogs.SetFollow(true)
			return true, nil
		case "l":
			if m.drawerLogs.IsPrevious {
				return true, m.streamDrawerLogs(false)
			}
		case "/":
			return true, m.openInput("drawer-log-filter")
		case "v":
			switch m.drawerLogs.LevelFilter {
			case "error":
				m.drawerLogs.LevelFilter = "warn"
			case "warn":
				m.drawerLogs.LevelFilter = "all"
			default:
				m.drawerLogs.LevelFilter = "error"
			}
			return true, nil
		}
	}
	if m.focus.ActiveTab == 0 && k == "p" {
		return true, m.requestPreviousLogs()
	}
	return false, nil
}

func (m *Model) persistRatio() tea.Cmd {
	if m.ratioWriter == nil {
		m.ratioWriter = &config.StateWriter{}
	}
	write := m.ratioWriter.Queue(config.State{ListRatio: m.drawerRatio})
	return func() tea.Msg {
		if err := write(); err != nil {
			return execDoneMsg{err}
		}
		return nil
	}
}

func (m *Model) ensureEnvControlVisible() {
	if m.focus.ActiveTab != 3 || m.focus.Area != FocusDrawerControls {
		return
	}
	w, h := m.drawerViewportSize()
	_, owners := m.envVisualRows(w)
	row := 0
	for i, owner := range owners {
		if owner == m.focus.ControlIdx {
			row = i
			break
		}
	}
	if row < m.drawerScroll {
		m.drawerScroll = row
	}
	if row >= m.drawerScroll+h {
		m.drawerScroll = max(0, row-h+1)
	}
}

func (m *Model) drawerViewportSize() (int, int) {
	l := masterdetail.ComputeLayout(m.width, m.height-tokens.HeaderHeight-tokens.FooterHeight, true, m.drawerRatio)
	w := l.InspectorWidth
	if m.focus.Zoomed {
		w = m.width
	}
	rows := 1
	if w < 100 {
		rows = 5
	}
	if m.focus.Zoomed || l.Mode == masterdetail.ModeOverlay {
		rows++
	}
	return w, max(0, l.Height-rows)
}

func (m *Model) fetchDrawerEnvironment() tea.Cmd {
	gen, id, ctx, service := m.drawerGen, m.drawerID, m.drawerCtx, m.service
	return func() tea.Msg {
		raw, err := service.FetchEnvironmentForLiveReveal(ctx, id)
		return drawerRevealMsg{gen, id, inspector.ParseEnv(raw), err}
	}
}

func (m *Model) confirmPortPreview() tea.Cmd {
	preview, err := inspector.PreviewPort(m.portMapping)
	valid := err == nil && m.focus.DrawerOpen && m.portContainer == m.drawerID && preview == m.portPreview
	found := false
	for _, p := range m.containerIns[m.drawerID].Ports {
		if p == m.portMapping {
			found = true
		}
	}
	m.overlay = ""
	if !valid || !found {
		return nil
	}
	url := preview.URL
	return func() tea.Msg {
		name, args := "xdg-open", []string{url}
		if runtime.GOOS == "darwin" {
			name = "open"
		}
		// No shell and no environment-controlled command interpolation.
		err := exec.Command(name, args...).Run()
		return execDoneMsg{err}
	}
}

func (m *Model) streamDrawerLogs(previous bool) tea.Cmd {
	if previous && m.containerIns[m.drawerID].StartedAt.IsZero() {
		return m.notify("Previous logs unavailable: inspect has no start boundary.")
	}
	m.stopLogs()
	m.logsGen++
	m.drawerLogs = inspector.LogsViewModel{Follow: !previous, IsPrevious: previous}
	opts := engine.LogOptions{Follow: !previous, Tail: 500, Previous: previous}
	if previous {
		opts.Until = m.containerIns[m.drawerID].StartedAt
		m.drawerLogs.PreviousUntil = opts.Until
	}
	gen, id, service := m.logsGen, m.drawerID, m.service
	ctx, cancel := context.WithCancel(m.drawerCtx)
	m.logsCancel = cancel
	ch := make(chan tea.Msg, 256)
	m.logCh = ch
	worker := func() tea.Msg {
		err := service.Logs(ctx, id, opts, func(entry engine.LogEntry) error {
			select {
			case ch <- logLineMsg{gen, entry}:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		select {
		case ch <- logEndMsg{gen, err}:
		case <-ctx.Done():
		}
		return nil
	}
	return tea.Batch(worker, waitLog(ctx, ch, gen))
}

func (m *Model) renderContainers(w, h int) string {
	l := masterdetail.ComputeLayout(w, h, m.focus.DrawerOpen, m.drawerRatio)
	now := time.Now()
	if m.fixture {
		now = m.clock
	}
	list := views.ContainerList(m.sheet, m.containers(), m.containerCursor, m.selectedContain, m.containerFilter, m.stats, m.statCPU, m.statHistory, l.ListWidth, h, now)
	if m.grouped {
		list = m.renderGroupedList(l.ListWidth, h)
	}
	if !m.focus.DrawerOpen {
		return list
	}
	dw := l.InspectorWidth
	if m.focus.Zoomed {
		dw = w
	}
	labels := []string{"Health & Diagnostics", "Live Logs", "Metrics", "Env & Ports", "Timeline"}
	if c, ok := m.containerIns[m.drawerID]; ok && (c.ExitCode != 0 || c.State == "restarting" || c.Health == "unhealthy") {
		labels[0] += " (ERR)"
	}
	for i := range labels {
		if i == m.focus.ActiveTab {
			labels[i] = "[" + labels[i] + "]"
		}
	}
	// Labels wrap rather than becoming undiscoverable on a narrow split.
	header := strings.Join(labels, " · ") + "\n"
	if dw < 100 {
		header = strings.Join(labels, "\n") + "\n"
	}
	if l.Mode == masterdetail.ModeOverlay || m.focus.Zoomed {
		header = "[Esc] Back\n" + header
	}
	hh := max(0, h-lipgloss.Height(header))
	c, ok := m.containerIns[m.drawerID]
	body := "Loading inspection…"
	if ok {
		switch m.focus.ActiveTab {
		case 0:
			_, observed := m.peakAt[m.drawerID]
			rich := inspector.RenderHealthTabWithEvidence(m.sheet, c, m.peakMemory[m.drawerID], observed, m.healthEvidence(c), 4096, 2048)
			body = inspector.RenderHealthTabWithEvidence(m.sheet, c, m.peakMemory[m.drawerID], observed, m.healthEvidence(c), 4096, len(strings.Split(strings.TrimRight(rich, " \n"), "\n")))
		case 1:
			body = m.drawerLogs.Render(m.sheet, dw, hh)
		case 2:
			sample, ok := m.stats[m.drawerID]
			if !ok {
				sample.State = engine.TelemetryUnavailable
			}
			if c.State != "running" {
				sample.State = engine.TelemetrySampled
			}
			age := time.Since(sample.At)
			if !sample.At.IsZero() && age > 10*time.Second {
				sample.State = engine.TelemetryUnavailable
				sample.CPUAvailable = false
				sample.MemoryAvailable = false
				sample.NetworkAvailable = false
				sample.BlockAvailable = false
				sample.PIDsAvailable = false
			}
			body = m.metricWindows(dw) + inspector.RenderMetricsTab(m.sheet, sample, m.statHistory[m.drawerID], c.State == "running", dw, max(0, hh+m.drawerScroll-4))
			if m.statsError {
				body = "Telemetry unavailable: last poll failed; retained samples are historical.\n" + body
			} else if !sample.At.IsZero() {
				body = fmt.Sprintf("Last sample %s · age %s\n", sample.At.Format(time.RFC3339), max(time.Duration(0), age).Truncate(time.Second)) + body
			}
		case 3:
			rows, _ := m.envVisualRows(dw)
			body = strings.Join(rows, "\n")
		case 4:
			items, incomplete := inspector.BuildIncidentSequence(c, m.eventHistory)
			// Latest compatibility probe is execution evidence too; coalesce it
			// with the full history instead of presenting the same run twice.
			if p := c.HealthCheck; p != nil {
				found := false
				for _, probe := range c.DiagnosticEvidence().HealthHistory {
					if probe.End.Equal(p.End) {
						found = true
					}
				}
				if found {
					filtered := items[:0]
					for _, item := range items {
						if item.Timestamp.Equal(p.End) && (item.Type == "health_ok" || item.Type == "health_warn") {
							continue
						}
						filtered = append(filtered, item)
					}
					items = filtered
				}
			}
			d := c.DiagnosticEvidence()
			for _, probe := range d.HealthHistory {
				if probe.End.IsZero() {
					continue
				}
				kind, outcome := "probe", "Probe exit code 0"
				if probe.ExitCode != 0 {
					kind = "probe"
					outcome = fmt.Sprintf("Probe exit code %d", probe.ExitCode)
				}
				duplicate := false
				for _, item := range items {
					if item.Timestamp.Equal(probe.End) && item.Type == kind {
						duplicate = true
					}
				}
				if !duplicate {
					items = append(items, inspector.TimelineItem{Timestamp: probe.End, Type: kind, Summary: "Completed health probe", Outcome: outcome})
				}
			}
			sort.SliceStable(items, func(i, j int) bool { return items[i].Timestamp.Before(items[j].Timestamp) })
			body = inspector.RenderTimelineTab(m.sheet, items, incomplete, 4096, 2048)
		}
	}
	lines := wrapVisual(strings.TrimRight(body, " \n"), dw)
	off := min(m.drawerScroll, max(0, len(lines)-1))
	if m.focus.ActiveTab != 1 {
		body = strings.Join(lines[off:], "\n")
	}
	if m.focus.ActiveTab == 0 {
		body = m.healthControls() + "\n" + body
	}
	drawer := style.Fit(header+body, dw, h)
	if l.Mode == masterdetail.ModeOverlay || m.focus.Zoomed {
		return drawer
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, list, style.Fit("│", 1, h), drawer)
}
