// Package app owns the Bubble Tea router, input state and asynchronous messages.
package app

import (
	"context"
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
type removedMsg map[string]error
type jobsMsg []jobs.Job
type eventMsg engine.Event
type themeMsg tokens.Palette
type pollMsg struct{}
type toastMsg uint64

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
	eventHistory []engine.Event
	err          *engine.ClassifiedError

	// Workers communicate through detached snapshots and cancellation-aware channels.
	jobs    *jobs.Manager
	jobList []jobs.Job
	jobID   uint64
	events  chan engine.Event
	themes  chan tokens.Palette

	// Navigation and presentation state.
	width          int
	height         int
	tab            int
	cursor         int
	sortMode       int
	paletteCursor  int
	selected       map[string]bool
	filter         string
	overlay        string
	toast          string
	toastID        uint64
	input          textinput.Model
	loading        bool
	refreshing     bool
	detail         bool
	fixture        bool
	compact        bool
	refreshPending bool
	completedJobs  map[uint64]bool
	motion         anim.Spring
	clock          time.Time
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
		ctx:           ctx,
		cancel:        cancel,
		service:       s,
		config:        c,
		sheet:         style.Sheet{Palette: p, Plain: plain, ASCII: c.UI.ASCII},
		jobs:          jobs.New(),
		events:        make(chan engine.Event, 64),
		themes:        make(chan tokens.Palette, 1),
		width:         120,
		height:        40,
		selected:      make(map[string]bool),
		inspections:   make(map[string]engine.Inspection),
		completedJobs: make(map[uint64]bool),
		input:         input,
		loading:       true,
		motion:        anim.Spring{Reduced: c.UI.ReduceMotion || plain},
		clock:         time.Now(),
	}
}

// SetFixture labels synthetic telemetry when the explicit fake engine is used.
func (m *Model) SetFixture(value bool) { m.fixture = value }

// Init starts independent resource, event and theme commands outside Update.
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.refresh(), m.startEvents(), m.waitEvent(), m.waitJobs(), m.poll()}
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
			return m, m.refresh()
		}
		return m, nil
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
			return m, tea.Batch(cmd, m.refresh())
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
		if m.overlay == "pull-input" || m.overlay == "palette" || m.overlay == "filter" {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *Model) quit() tea.Cmd { m.jobs.CancelAll(); m.cancel(); return tea.Quit }
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
func (m *Model) clamp() { m.cursor = max(0, min(m.cursor, len(m.items())-1)) }
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

// Gallery seeds a model for rendering only, without starting Init or background I/O.
func (m *Model) Gallery(batch store.Snapshot, tab int, overlay string, inspection *engine.Inspection) {
	m.store.Replace(batch)
	m.tab = tab
	m.overlay = overlay
	m.loading = false
	m.fixture = true
	m.clock = time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	if inspection != nil {
		m.inspections[inspection.ID] = *inspection
	}
	if overlay == "pull" {
		m.jobID = 1
		m.jobList = []jobs.Job{{ID: 1, Target: "redis:7.4", State: "running", Current: 37_200_000, Total: 60_000_000, Speed: 12_400_000, ETA: 9 * time.Second, Layers: []engine.Progress{{ID: "a3f1c9e2b7d4", Status: "Pull complete", Current: 20_000_000, Total: 20_000_000}, {ID: "7c0d8e5a1f92", Status: "Downloading", Current: 17_200_000, Total: 40_000_000}}}}
	}
}
