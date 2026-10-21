// Package app owns the Bubble Tea router, input state and asynchronous messages.
package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/Mersad-Moghaddam/lenger/internal/config"
	"github.com/Mersad-Moghaddam/lenger/internal/core/jobs"
	"github.com/Mersad-Moghaddam/lenger/internal/core/services"
	"github.com/Mersad-Moghaddam/lenger/internal/core/store"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/anim"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/components"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
)

type snapshotMsg struct {
	data store.Snapshot
	err  error
}
type inspectMsg struct {
	id   string
	data engine.Inspection
	err  error
}
type containerInspectMsg struct {
	id   string
	data engine.ContainerInspection
	err  error
}
type containerActionMsg struct {
	action engine.ContainerAction
	ids    []string
	errs   map[string]error
}
type usageMsg struct {
	data engine.SystemUsage
	err  error
}
type statsMsg struct {
	data map[string]engine.ContainerStats
}
type statsTickMsg struct{}
type runMsg struct {
	id  string
	err error
}
type cleanupPlanMsg struct {
	plan services.CleanupPlan
	err  error
}
type cleanupDoneMsg struct {
	result services.CleanupResult
	err    error
}
type volumeRemovedMsg struct {
	name string
	err  error
}
type execDoneMsg struct{ err error }
type logLineMsg struct {
	gen   uint64
	entry engine.LogEntry
}
type logEndMsg struct {
	gen uint64
	err error
}
type logSavedMsg struct {
	path string
	err  error
}
type removedMsg map[string]error
type jobsMsg []jobs.Job
type eventMsg engine.Event
type themeMsg tokens.Palette
type pollMsg struct{}
type toastMsg uint64

type daemonInfoMsg struct {
	data engine.DaemonInfo
	err  error
}
type composeLineMsg struct {
	line string
}
type composeDoneMsg struct {
	action engine.ComposeAction
	err    error
}
type networkCreatedMsg struct {
	id  string
	err error
}
type networkRemovedMsg struct {
	id  string
	err error
}

type registrySearchMsg struct {
	query   string
	results []engine.RegistrySearchResult
	err     error
}
type remoteTagsMsg struct {
	repo string
	tags []engine.RemoteTag
	err  error
}
type imageUpdateMsg struct {
	id     string
	status engine.ImageUpdateStatus
	err    error
}
type vulnScanMsg struct {
	id     string
	report engine.VulnerabilityReport
	err    error
}
type pushProgressMsg struct {
	progress engine.Progress
}
type pushDoneMsg struct {
	ref string
	err error
}
type imageTaggedMsg struct {
	sourceID  string
	targetRef string
	err       error
}
type imageSavedMsg struct {
	path string
	err  error
}
type imageLoadedMsg struct {
	path string
	err  error
}
type daemonMirrorsMsg struct {
	diff      string
	newConfig string
	err       error
}

// Model contains session-local state; pointer ownership stays on the model thread.
type Model struct {
	// Dependencies and lifetime are fixed at construction.
	ctx     context.Context
	cancel  context.CancelFunc
	service *services.Service
	config  config.Config

	// Resources are owned by the Bubble Tea model thread.
	sheet        style.Sheet
	store        store.Store
	inspections  map[string]engine.Inspection
	containerIns map[string]engine.ContainerInspection
	eventHistory []engine.Event
	err          *engine.ClassifiedError

	// Disk usage is loaded lazily for the System tab and cleanup preview.
	usage        engine.SystemUsage
	usageLoaded  bool
	usageLoading bool
	usageErr     *engine.ClassifiedError

	// Container resource samples are keyed by ID and refreshed only on tab 2.
	stats        map[string]engine.ContainerStats
	statCPU      map[string]float64
	statHistory  map[string][]float64
	statsLoading bool
	statsTicking bool

	// Workers communicate through detached snapshots and cancellation-aware channels.
	jobs    *jobs.Manager
	jobList []jobs.Job
	jobID   uint64
	events  chan engine.Event
	themes  chan tokens.Palette

	// Navigation and presentation state.
	width           int
	height          int
	tab             int
	cursor          int
	containerCursor int
	volumeCursor    int
	projectCursor   int
	networkCursor   int
	sortMode        int
	paletteCursor   int
	selected        map[string]bool
	selectedContain map[string]bool
	filter          string
	containerFilter string
	volumeFilter    string
	projectFilter   string
	networkFilter   string
	overlay         string
	toast           string
	toastID         uint64
	input           textinput.Model
	loading         bool
	refreshing      bool
	detail          bool
	containerDetail bool
	containerRaw    bool
	fixture         bool
	compact         bool
	refreshPending  bool
	completedJobs   map[uint64]bool
	motion          anim.Spring
	clock           time.Time

	// Host daemon information (SYS-03)
	daemonInfo    engine.DaemonInfo
	daemonLoaded  bool
	daemonLoading bool

	// Docker compose state and log stream (PRJ-01..PRJ-05)
	composeStream  []string
	composeAction  engine.ComposeAction
	composeProject string
	composeVolumes bool
	composeCh      chan tea.Msg

	// Network creation & removal state (NET-01)
	createNetName   string
	createNetDriver string
	removeNetwork   string

	// Container logs are a capped ring owned by the model thread.
	logs           []engine.LogEntry
	logsID         string
	logsFollow     bool
	logsPaused     bool
	logsTimestamps bool
	logsWrap       bool
	logsFilter     string
	logsNotice     string
	logsGen        uint64
	logsCancel     context.CancelFunc
	logCh          chan tea.Msg

	// Modal workflows keep their editable state on the model thread.
	runImage        string
	runForm         runForm
	runField        int
	runErr          string
	runBusy         bool
	actionIDs       []string
	actionName      engine.ContainerAction
	cleanupPlan     services.CleanupPlan
	cleanupLoaded   bool
	cleanupLoading  bool
	cleanupErr      *engine.ClassifiedError
	cleanupSelected map[services.CleanupKind]bool
	cleanupCursor   int
	removeVolume    string
	execContainer   string
	execShell       int

	// Registry search & tag browser state (IMG-06)
	registryQuery        string
	registryResults      []engine.RegistrySearchResult
	registryCursor       int
	registryTags         []engine.RemoteTag
	registryTagCursor    int
	registrySearching    bool
	registryBrowsingTags bool
	registryErr          string

	// Tag, Push, Save, Load state (IMG-08)
	tagSourceID    string
	tagSourceImage string
	pushImageRef   string
	pushProgress   []engine.Progress
	pushBusy       bool
	pushErr        string
	exportImageID  string

	// Daemon mirrors diff state (REG-02, REG-03)
	mirrorDiffText  string
	mirrorNewConfig string
	mirrorSudoCmd   string
}

// New creates a model without performing I/O or scheduling animation.
func New(ctx context.Context, s *services.Service, c config.Config, p tokens.Palette) *Model {
	ctx, cancel := context.WithCancel(ctx)
	input := textinput.New()
	input.SetVirtualCursor(false)
	input.CharLimit = 255
	input.SetWidth(tokens.ModalWidth - tokens.InsetX*4)
	_, plain := os.LookupEnv("NO_COLOR")
	return &Model{
		ctx:             ctx,
		cancel:          cancel,
		service:         s,
		config:          c,
		sheet:           style.Sheet{Palette: p, Plain: plain, ASCII: c.UI.ASCII},
		jobs:            jobs.New(),
		events:          make(chan engine.Event, 64),
		themes:          make(chan tokens.Palette, 1),
		width:           120,
		height:          40,
		selected:        make(map[string]bool),
		selectedContain: make(map[string]bool),
		cleanupSelected: make(map[services.CleanupKind]bool),
		inspections:     make(map[string]engine.Inspection),
		containerIns:    make(map[string]engine.ContainerInspection),
		stats:           make(map[string]engine.ContainerStats),
		statCPU:         make(map[string]float64),
		statHistory:     make(map[string][]float64),
		completedJobs:   make(map[uint64]bool),
		input:           input,
		loading:         true,
		logsTimestamps:  true,
		motion:          anim.Spring{Reduced: c.UI.ReduceMotion || plain},
		clock:           time.Now(),
	}
}

// SetFixture labels synthetic telemetry when the explicit fake engine is used.
func (m *Model) SetFixture(value bool) { m.fixture = value }

// Init starts independent resource, event and theme commands outside Update.
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.refresh(), m.startEvents(), m.waitEvent(), m.waitJobs(), m.poll(), m.loadDaemonInfo()}
	if m.config.UI.Theme == "omarchy" {
		cmds = append(cmds, m.startTheme(), m.waitTheme())
	}
	return tea.Batch(cmds...)
}

func (m *Model) refresh() tea.Cmd {
	if m.refreshing {
		m.refreshPending = true
		return nil
	}
	m.refreshing = true
	ctx, service := m.ctx, m.service
	return func() tea.Msg { batch, err := service.Snapshot(ctx); return snapshotMsg{batch, err} }
}

// Update routes messages; all network, filesystem and engine work uses commands.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = max(0, msg.Width)
		m.height = max(0, msg.Height)
		m.input.SetWidth(max(1, min(tokens.ModalWidth, m.width)-tokens.InsetX*4))
		return m, nil
	case snapshotMsg:
		m.refreshing = false
		m.loading = false
		m.clock = time.Now()
		if msg.err != nil {
			m.err = engine.Classify(msg.err)
		} else {
			m.err = nil
			m.store.Replace(msg.data)
			m.clamp()
		}
		if m.refreshPending {
			m.refreshPending = false
			return m, tea.Batch(m.refresh(), m.loadStats())
		}
		if m.tab == 2 && m.err == nil {
			return m, m.loadStats()
		}
		return m, nil
	case daemonInfoMsg:
		m.daemonLoading = false
		if msg.err == nil {
			m.daemonInfo = msg.data
			m.daemonLoaded = true
		}
		return m, nil
	case composeLineMsg:
		m.composeStream = append(m.composeStream, msg.line)
		if len(m.composeStream) > 200 {
			m.composeStream = m.composeStream[len(m.composeStream)-200:]
		}
		return m, m.waitComposeCh()
	case composeDoneMsg:
		text := fmt.Sprintf("Compose %s completed.", msg.action)
		if msg.err != nil {
			text = fmt.Sprintf("Compose %s failed: %s", msg.action, engine.Classify(msg.err).Message)
		}
		m.closeOverlay()
		return m, tea.Batch(m.notify(text), m.refresh())
	case networkCreatedMsg:
		text := "Network created."
		if msg.err != nil {
			text = engine.Classify(msg.err).Message
		}
		m.closeOverlay()
		return m, tea.Batch(m.notify(text), m.refresh())
	case networkRemovedMsg:
		text := "Network removed."
		if msg.err != nil {
			text = engine.Classify(msg.err).Message
		}
		m.closeOverlay()
		return m, tea.Batch(m.notify(text), m.refresh())
	case containerInspectMsg:
		if msg.err != nil {
			return m, m.notify(engine.Classify(msg.err).Message)
		}
		m.containerIns[msg.id] = msg.data
		return m, nil
	case containerActionMsg:
		failures := 0
		for _, err := range msg.errs {
			if err != nil {
				failures++
			}
		}
		if msg.action == engine.ActionRemove && failures == 0 {
			for _, id := range msg.ids {
				delete(m.selectedContain, id)
				delete(m.containerIns, id)
			}
		}
		text := "Container action completed."
		if failures > 0 {
			text = "Some container actions failed; refresh and check the current state."
		}
		return m, tea.Batch(m.notify(text), m.refresh(), m.loadUsage(), m.loadStats())
	case usageMsg:
		m.usageLoading = false
		if msg.err != nil {
			m.usageErr = engine.Classify(msg.err)
			m.usageLoaded = false
		} else {
			m.usage = msg.data
			m.usageLoaded = true
			m.usageErr = nil
		}
		return m, nil
	case statsMsg:
		m.statsLoading = false
		for id, sample := range msg.data {
			if previous, ok := m.stats[id]; ok && sample.SystemTotal > previous.SystemTotal {
				deltaSystem := sample.SystemTotal - previous.SystemTotal
				deltaCPU := sample.CPUTotal - previous.CPUTotal
				percent := 0.0
				if deltaCPU <= deltaSystem*64 {
					percent = float64(deltaCPU) / float64(deltaSystem) * 100
					percent *= float64(max(1, sample.OnlineCPUs))
					percent = max(0, min(999, percent))
				}
				m.statCPU[id] = percent
				history := append(m.statHistory[id], percent)
				if len(history) > 32 {
					history = history[len(history)-32:]
				}
				m.statHistory[id] = history
			}
			m.stats[id] = sample
		}
		if m.tab == 2 {
			return m, m.statsTick()
		}
		return m, nil
	case statsTickMsg:
		m.statsTicking = false
		if m.tab != 2 || !m.hasRunningContainers() {
			return m, nil
		}
		return m, tea.Batch(m.loadStats(), m.statsTick())
	case runMsg:
		m.runBusy = false
		if msg.err != nil {
			m.runErr = engine.Classify(msg.err).Message
			return m, nil
		}
		m.closeOverlay()
		m.tab = 2
		m.containerDetail = false
		return m, tea.Batch(m.notify("Container started. Logs are one key away."), m.refresh(), m.loadUsage(), m.loadStats())
	case cleanupPlanMsg:
		m.cleanupLoading = false
		m.cleanupLoaded = true
		if msg.err != nil {
			m.cleanupErr = engine.Classify(msg.err)
			return m, nil
		}
		m.cleanupErr = nil
		m.cleanupPlan = msg.plan
		m.cleanupCursor = 0
		m.cleanupSelected = defaultCleanupSelection()
		return m, nil
	case cleanupDoneMsg:
		m.cleanupLoading = false
		if msg.err != nil {
			return m, m.notify(engine.Classify(msg.err).Message)
		}
		m.closeOverlay()
		text := fmt.Sprintf("Cleanup removed %d item(s).", msg.result.Removed)
		if msg.result.Reclaimed > 0 {
			prefix := ""
			if msg.result.Estimated {
				prefix = "up to "
			}
			text += " " + prefix + components.Bytes(msg.result.Reclaimed) + " reclaimable."
		}
		if len(msg.result.Errors) > 0 {
			text += fmt.Sprintf(" %d item(s) were skipped or failed.", len(msg.result.Errors))
		}
		return m, tea.Batch(m.notify(text), m.refresh(), m.loadUsage())
	case volumeRemovedMsg:
		if msg.err != nil {
			return m, m.notify(engine.Classify(msg.err).Message)
		}
		m.closeOverlay()
		return m, tea.Batch(m.notify("Volume removed."), m.refresh(), m.loadUsage())
	case execDoneMsg:
		if msg.err != nil {
			return m, m.notify("Exec exited with an error: " + engine.Classify(msg.err).Message)
		}
		return m, m.refresh()
	case logLineMsg:
		if msg.gen != m.logsGen {
			return m, nil
		}
		m.logs = append(m.logs, msg.entry)
		if len(m.logs) > 50_000 {
			m.logs = m.logs[len(m.logs)-50_000:]
		}
		return m, waitLog(m.logCh, m.logsGen, m.ctx)
	case logEndMsg:
		if msg.gen != m.logsGen {
			return m, nil
		}
		m.logsCancel = nil
		m.logsFollow = false
		if msg.err != nil && m.ctx.Err() == nil && !errors.Is(msg.err, context.Canceled) {
			m.logsNotice = engine.Classify(msg.err).Message
		}
		return m, nil
	case logSavedMsg:
		if msg.err != nil {
			return m, m.notify("Could not save logs: " + msg.err.Error())
		}
		return m, m.notify("Logs saved to " + msg.path)
	case pullStartedMsg:
		m.jobID = msg.id
		// The notification loop owns completion bookkeeping, including pulls
		// that finish before their first running snapshot is displayed.
		return m, nil
	case inspectMsg:
		if msg.err != nil {
			return m, m.notify(engine.Classify(msg.err).Message)
		}
		m.inspections[msg.id] = msg.data
		return m, nil
	case registrySearchMsg:
		m.registrySearching = false
		if msg.err != nil {
			m.registryErr = engine.Classify(msg.err).Message
		} else {
			m.registryResults = msg.results
			m.registryCursor = 0
			m.registryErr = ""
		}
		return m, nil
	case remoteTagsMsg:
		m.registrySearching = false
		if msg.err != nil {
			m.registryErr = engine.Classify(msg.err).Message
		} else {
			m.registryTags = msg.tags
			m.registryTagCursor = 0
			m.registryBrowsingTags = true
		}
		return m, nil
	case imageUpdateMsg:
		if msg.err != nil {
			return m, m.notify("Update check failed: " + engine.Classify(msg.err).Message)
		}
		ins := m.inspections[msg.id]
		ins.UpdateStatus = &msg.status
		m.inspections[msg.id] = ins
		notice := fmt.Sprintf("%s is up to date.", msg.status.Ref)
		if msg.status.HasUpdate {
			notice = fmt.Sprintf("Update available for %s!", msg.status.Ref)
		}
		return m, m.notify(notice)
	case vulnScanMsg:
		if msg.err != nil {
			return m, m.notify("Scan failed: " + engine.Classify(msg.err).Message)
		}
		ins := m.inspections[msg.id]
		ins.Vulnerabilities = &msg.report
		m.inspections[msg.id] = ins
		notice := fmt.Sprintf("Scan completed (%s): %d Critical, %d High", msg.report.Scanner, msg.report.Critical, msg.report.High)
		if msg.report.Error != "" {
			notice = "Scan notice: " + msg.report.Error
		}
		return m, m.notify(notice)
	case pushProgressMsg:
		m.pushProgress = append(m.pushProgress, msg.progress)
		if len(m.pushProgress) > 30 {
			m.pushProgress = m.pushProgress[len(m.pushProgress)-30:]
		}
		return m, nil
	case pushDoneMsg:
		m.pushBusy = false
		if msg.err != nil {
			m.pushErr = engine.Classify(msg.err).Message
			return m, nil
		}
		m.closeOverlay()
		return m, m.notify("Push completed successfully: " + msg.ref)
	case imageTaggedMsg:
		if msg.err != nil {
			return m, m.notify("Tag failed: " + engine.Classify(msg.err).Message)
		}
		m.closeOverlay()
		return m, tea.Batch(m.notify("Tagged: "+msg.targetRef), m.refresh())
	case imageSavedMsg:
		if msg.err != nil {
			return m, m.notify("Export failed: " + engine.Classify(msg.err).Message)
		}
		m.closeOverlay()
		return m, m.notify("Image exported to " + msg.path)
	case imageLoadedMsg:
		if msg.err != nil {
			return m, m.notify("Import failed: " + engine.Classify(msg.err).Message)
		}
		m.closeOverlay()
		return m, tea.Batch(m.notify("Image imported from "+msg.path), m.refresh())
	case daemonMirrorsMsg:
		if msg.err != nil {
			return m, m.notify("Daemon diff failed: " + engine.Classify(msg.err).Message)
		}
		m.mirrorDiffText = msg.diff
		m.mirrorNewConfig = msg.newConfig
		m.mirrorSudoCmd = fmt.Sprintf("echo %q | sudo tee /etc/docker/daemon.json && sudo systemctl reload docker", msg.newConfig)
		m.overlay = "daemon-mirrors"
		return m, nil
	case removedMsg:
		failures := 0
		for id, err := range msg {
			if err != nil {
				failures++
			} else {
				delete(m.selected, id)
				delete(m.inspections, id)
			}
		}
		text := "Image removal completed."
		if failures > 0 {
			text = "Some images could not be removed; they may still be in use. Refresh and inspect their references."
		}
		return m, tea.Batch(m.notify(text), m.refresh())
	case jobsMsg:
		finished := false
		if m.completedJobs == nil {
			m.completedJobs = make(map[uint64]bool)
		}
		for _, j := range msg {
			if j.State != "running" && !m.completedJobs[j.ID] {
				m.completedJobs[j.ID] = true
				finished = true
			}
		}
		m.jobList = []jobs.Job(msg)
		cmd := m.waitJobs()
		if finished {
			return m, tea.Batch(cmd, m.refresh())
		}
		return m, cmd
	case eventMsg:
		e := engine.Event(msg)
		m.eventHistory = append(m.eventHistory, e)
		if len(m.eventHistory) > 100 {
			m.eventHistory = m.eventHistory[len(m.eventHistory)-100:]
		}
		cmd := m.waitEvent()
		if store.Invalidate(e) {
			m.usageLoaded = false
			cmds := []tea.Cmd{cmd, m.refresh()}
			if m.tab == 6 {
				cmds = append(cmds, m.loadUsage())
			}
			return m, tea.Batch(cmds...)
		}
		return m, cmd
	case eventStreamStoppedMsg:
		return m, m.notify("Docker event stream stopped. Safety polling remains active; restart Lenger to resubscribe.")
	case themeMsg:
		m.sheet.Palette = tokens.Palette(msg)
		return m, m.waitTheme()
	case pollMsg:
		cmd := m.poll()
		if !m.refreshing {
			return m, tea.Batch(cmd, m.refresh())
		}
		return m, cmd
	case toastMsg:
		if uint64(msg) == m.toastID {
			m.toast = ""
		}
		return m, nil
	case anim.Tick:
		m.motion.Step()
		return m, m.motion.Next()
	case tea.MouseClickMsg:
		return m, m.mouse(msg.Mouse().X, msg.Mouse().Y)
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, m.quit()
		}
		return m, m.key(msg)
	case tea.PasteMsg:
		switch m.overlay {
		case "pull-input", "palette", "filter", "log-filter", "remove-volume":
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		case "cleanup-confirm":
			if m.cleanupSelected[services.CleanupUnusedVolumes] {
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
		case "run":
			if m.runField != 4 {
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
		}
	}
	return m, nil
}

func (m *Model) quit() tea.Cmd {
	if m.logsCancel != nil {
		m.logsCancel()
	}
	m.jobs.CancelAll()
	m.cancel()
	return tea.Quit
}
func (m *Model) notify(text string) tea.Cmd {
	m.toast = text
	m.toastID++
	id := m.toastID
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return toastMsg(id) })
}
func (m *Model) items() []engine.Image {
	items := make([]engine.Image, 0, len(m.store.Images))
	query := strings.ToLower(m.filter)
	for _, i := range m.store.Images {
		if strings.Contains(strings.ToLower(strings.Join(i.RepoTags, " ")+" "+i.ID), query) {
			items = append(items, i)
		}
	}
	sort.SliceStable(items, func(a, b int) bool {
		switch m.sortMode {
		case 1:
			if items[a].Size != items[b].Size {
				return items[a].Size > items[b].Size
			}
		case 2:
			if !items[a].Created.Equal(items[b].Created) {
				return items[a].Created.After(items[b].Created)
			}
		}
		return items[a].Name() < items[b].Name()
	})
	return items
}
func (m *Model) clamp() {
	m.cursor = max(0, min(m.cursor, len(m.items())-1))
	m.containerCursor = max(0, min(m.containerCursor, len(m.containers())-1))
	m.volumeCursor = max(0, min(m.volumeCursor, len(m.volumes())-1))
	m.projectCursor = max(0, min(m.projectCursor, len(m.projects())-1))
	m.networkCursor = max(0, min(m.networkCursor, len(m.networks())-1))
	m.cleanupCursor = max(0, min(m.cleanupCursor, len(services.CleanupKinds())-1))
}

func (m *Model) containers() []engine.Container {
	query := strings.ToLower(m.containerFilter)
	items := make([]engine.Container, 0, len(m.store.Containers))
	for _, c := range m.store.Containers {
		haystack := strings.ToLower(strings.Join([]string{c.Name, c.Image, c.State, c.Status, c.Project, c.Service, c.ID}, " "))
		if strings.Contains(haystack, query) {
			items = append(items, c)
		}
	}
	sort.SliceStable(items, func(a, b int) bool { return items[a].Name < items[b].Name })
	return items
}

func (m *Model) projects() []engine.ComposeProject {
	query := strings.ToLower(m.projectFilter)
	items := make([]engine.ComposeProject, 0, len(m.store.Projects))
	for _, p := range m.store.Projects {
		haystack := strings.ToLower(p.Name + " " + p.Status + " " + strings.Join(p.Services, " "))
		if strings.Contains(haystack, query) {
			items = append(items, p)
		}
	}
	sort.SliceStable(items, func(a, b int) bool { return items[a].Name < items[b].Name })
	return items
}

func (m *Model) networks() []engine.Network {
	query := strings.ToLower(m.networkFilter)
	items := make([]engine.Network, 0, len(m.store.Networks))
	for _, n := range m.store.Networks {
		haystack := strings.ToLower(n.Name + " " + n.Driver + " " + n.Scope + " " + n.ID)
		if strings.Contains(haystack, query) {
			items = append(items, n)
		}
	}
	sort.SliceStable(items, func(a, b int) bool { return items[a].Name < items[b].Name })
	return items
}

func (m *Model) volumes() []engine.Volume {
	query := strings.ToLower(m.volumeFilter)
	items := make([]engine.Volume, 0, len(m.store.Volumes))
	for _, v := range m.store.Volumes {
		haystack := strings.ToLower(v.Name + " " + v.Driver + " " + v.Mountpoint)
		if strings.Contains(haystack, query) {
			items = append(items, v)
		}
	}
	sort.SliceStable(items, func(a, b int) bool { return items[a].Name < items[b].Name })
	return items
}

func (m *Model) activeFilter() string {
	switch m.tab {
	case 2:
		return m.containerFilter
	case 3:
		return m.projectFilter
	case 4:
		return m.volumeFilter
	case 5:
		return m.networkFilter
	default:
		return m.filter
	}
}

func (m *Model) setActiveFilter(value string) {
	switch m.tab {
	case 2:
		m.containerFilter = value
		m.containerCursor = 0
	case 3:
		m.projectFilter = value
		m.projectCursor = 0
	case 4:
		m.volumeFilter = value
		m.volumeCursor = 0
	case 5:
		m.networkFilter = value
		m.networkCursor = 0
	default:
		m.filter = value
		m.cursor = 0
	}
}

func (m *Model) inspect() tea.Cmd {
	items := m.items()
	if len(items) == 0 {
		return nil
	}
	id := items[m.cursor].ID
	m.detail = true
	if _, ok := m.inspections[id]; ok {
		return nil
	}
	ctx, service := m.ctx, m.service
	return func() tea.Msg { i, err := service.Inspect(ctx, id); return inspectMsg{id, i, err} }
}

func (m *Model) inspectContainer() tea.Cmd {
	items := m.containers()
	if len(items) == 0 {
		return nil
	}
	c := items[m.containerCursor]
	m.containerDetail = true
	if _, ok := m.containerIns[c.ID]; ok {
		return nil
	}
	ctx, service := m.ctx, m.service
	return func() tea.Msg {
		data, err := service.InspectContainer(ctx, c.ID)
		return containerInspectMsg{id: c.ID, data: data, err: err}
	}
}

// Gallery seeds a model for rendering only, without starting Init or background I/O.
func (m *Model) Gallery(batch store.Snapshot, tab int, overlay string, inspection *engine.Inspection) {
	m.store.Replace(batch)
	m.tab = tab
	m.overlay = overlay
	m.loading = false
	m.fixture = true
	m.clock = time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	if inspection != nil {
		inspection.UpdateStatus = &engine.ImageUpdateStatus{
			Ref:          batch.Images[0].Name(),
			LocalDigest:  "sha256:local",
			RemoteDigest: "sha256:remote-newer",
			HasUpdate:    true,
			CheckedAt:    m.clock,
		}
		inspection.Vulnerabilities = &engine.VulnerabilityReport{
			Scanner:   "trivy",
			Critical:  1,
			High:      3,
			Medium:    7,
			Low:       12,
			Fixable:   4,
			ScannedAt: m.clock,
		}
		m.inspections[inspection.ID] = *inspection
	}
	m.usage = galleryUsage(batch)
	m.usageLoaded = true
	m.daemonInfo = batch.Daemon
	m.daemonLoaded = true
	m.seedGalleryStats(batch)
	if len(batch.Containers) > 0 {
		if data, err := m.service.InspectContainer(m.ctx, batch.Containers[0].ID); err == nil {
			m.containerIns[data.ID] = data
		}
	}
	switch overlay {
	case "compose-action":
		if len(m.store.Projects) > 0 {
			m.composeProject = m.store.Projects[0].Name
			m.composeAction = engine.ComposeDown
			m.composeVolumes = true
		}
	case "create-network":
		m.createNetName = "custom-bridge"
		m.createNetDriver = "bridge"
		m.input.SetValue("custom-bridge")
	case "remove-network":
		if len(batch.Networks) > 0 {
			m.removeNetwork = batch.Networks[0].ID
		}
	case "pull":
		m.jobID = 1
		m.jobList = []jobs.Job{{ID: 1, Target: "redis:7.4", State: "running", Current: 37_200_000, Total: 60_000_000, Speed: 12_400_000, ETA: 9 * time.Second, Layers: []engine.Progress{{ID: "a3f1c9e2b7d4", Status: "Pull complete", Current: 20_000_000, Total: 20_000_000}, {ID: "7c0d8e5a1f92", Status: "Downloading", Current: 17_200_000, Total: 40_000_000}}}}
	case "run":
		if len(batch.Images) > 0 {
			m.runImage = batch.Images[0].Name()
			m.runForm = runForm{name: "lenger-web", ports: "8080:80", env: "APP_ENV=production", restart: 1}
			m.runField = 0
			m.input.SetValue(m.runForm.name)
			_ = m.input.Focus()
		}
	case "logs":
		if len(batch.Containers) > 0 {
			m.logsID = batch.Containers[0].ID
			m.logsFollow = false
			m.logsPaused = true
			m.logsNotice = "fixture buffer · stream detached"
			m.logs = []engine.LogEntry{
				{At: m.clock.Add(-4 * time.Second), Stream: "stdout", Text: "server listening on :8080"},
				{At: m.clock.Add(-3 * time.Second), Stream: "stdout", Text: "connected to postgres:5432"},
				{At: m.clock.Add(-2 * time.Second), Stream: "stderr", Text: "warn: cache miss for /api/health"},
				{At: m.clock.Add(-time.Second), Stream: "stdout", Text: "GET /api/health 200 2.4ms"},
			}
		}
	case "cleanup":
		if plan, err := m.service.CleanupPreview(m.ctx); err == nil {
			m.cleanupPlan = plan
			m.cleanupLoaded = true
			m.cleanupSelected = defaultCleanupSelection()
		}
	case "container-action":
		if len(batch.Containers) > 0 {
			m.actionIDs = []string{batch.Containers[0].ID}
			m.actionName = engine.ActionRemove
		}
	case "exec":
		if len(batch.Containers) > 0 {
			m.execContainer = batch.Containers[0].ID
		}
	case "remove-volume":
		for _, v := range batch.Volumes {
			if v.RefCount == 0 && len(v.UsedBy) == 0 {
				m.removeVolume = v.Name
				break
			}
		}
	case "registry-search":
		m.registryQuery = "nginx"
		m.registryResults = []engine.RegistrySearchResult{
			{Name: "nginx", Description: "Official build of Nginx.", Stars: 20100, Official: true},
			{Name: "bitnami/nginx", Description: "Bitnami container image for NGINX.", Stars: 430, Official: false, Automated: true},
		}
		m.input.SetValue("nginx")
	case "tag-image":
		if len(batch.Images) > 0 {
			m.tagSourceID = batch.Images[0].ID
			m.tagSourceImage = batch.Images[0].Name()
			m.input.SetValue("registry.example.com/web:1.27")
		}
	case "push-image":
		m.pushImageRef = "registry.example.com/web:1.27"
		m.pushBusy = true
		m.pushProgress = []engine.Progress{
			{ID: "layer-1", Status: "Pushed", Current: 10485760, Total: 10485760},
			{ID: "layer-2", Status: "Pushing", Current: 5242880, Total: 10485760},
		}
	case "export-image":
		if len(batch.Images) > 0 {
			m.exportImageID = batch.Images[0].ID
			m.input.SetValue("/tmp/nginx-export.tar")
		}
	case "import-image":
		m.input.SetValue("/tmp/app-bundle.tar")
	case "daemon-mirrors":
		m.mirrorDiffText = "--- /etc/docker/daemon.json\n+++ /etc/docker/daemon.json\n+   \"registry-mirrors\": [\"https://mirror.gcr.io\"]"
		m.mirrorSudoCmd = "echo '{\"registry-mirrors\": [\"https://mirror.gcr.io\"]}' | sudo tee /etc/docker/daemon.json && sudo systemctl reload docker"
	}
}

func (m *Model) seedGalleryStats(batch store.Snapshot) {
	for n, c := range batch.Containers {
		if !c.Running() {
			continue
		}
		percent := []float64{18.4, 7.1, 0.8}[n%3]
		memory := []uint64{96_000_000, 512_000_000, 18_000_000}[n%3]
		m.stats[c.ID] = engine.ContainerStats{
			At: m.clock, SystemTotal: 1_000_000_000,
			CPUTotal: uint64(1_000_000_000 / 100 * percent / 4), OnlineCPUs: 4,
			Memory: memory, MemoryLimit: 1_000_000_000,
			NetworkRX: memory / 2, NetworkTX: memory / 6,
			BlockRead: memory / 8, BlockWrite: memory / 16, PIDs: uint64(8 + n),
		}
		m.statCPU[c.ID] = percent
		history := make([]float64, 0, 16)
		for i := 0; i < 16; i++ {
			wave := 0.72 + 0.28*float64((i*7+n*3)%10)/10
			history = append(history, percent*wave)
		}
		m.statHistory[c.ID] = history
	}
}

func galleryUsage(batch store.Snapshot) engine.SystemUsage {
	var usage engine.SystemUsage
	for _, image := range batch.Images {
		usage.Images.Total++
		usage.Images.Size += max(0, image.Size)
		if len(image.UsedBy) > 0 {
			usage.Images.Active++
		} else {
			usage.Images.Reclaimable += max(0, image.Size)
		}
	}
	for _, container := range batch.Containers {
		usage.Containers.Total++
		if container.Running() {
			usage.Containers.Active++
		}
	}
	for _, volume := range batch.Volumes {
		usage.Volumes.Total++
		usage.Volumes.Size += max(0, volume.Size)
		if len(volume.UsedBy) > 0 {
			usage.Volumes.Active++
		} else {
			usage.Volumes.Reclaimable += max(0, volume.Size)
		}
	}
	usage.BuildCache = engine.UsageCategory{Total: 1, Size: 700_000_000, Reclaimable: 700_000_000}
	usage.Used = usage.Images.Size + usage.Containers.Size + usage.Volumes.Size + usage.BuildCache.Size
	usage.Reclaimable = usage.Images.Reclaimable + usage.Containers.Reclaimable + usage.Volumes.Reclaimable + usage.BuildCache.Reclaimable
	return usage
}
