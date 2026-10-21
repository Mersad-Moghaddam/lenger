package app

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"fmt"
	"github.com/Mersad-Moghaddam/lenger/internal/core/services"
	"github.com/Mersad-Moghaddam/lenger/internal/core/store"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/engine/fake"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/masterdetail"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/views/inspector"
	"strings"
	"testing"
	"time"
)

type synchronizedDrawerEngine struct {
	engine.Engine
	logsStarted  chan struct{}
	logsStopped  chan struct{}
	statsStarted chan struct{}
	statsStopped chan struct{}
}

func (e *synchronizedDrawerEngine) Logs(ctx context.Context, id string, o engine.LogOptions, emit func(engine.LogEntry) error) error {
	e.logsStarted <- struct{}{}
	<-ctx.Done()
	e.logsStopped <- struct{}{}
	return ctx.Err()
}
func (e *synchronizedDrawerEngine) ContainerStats(ctx context.Context, id string) (engine.ContainerStats, error) {
	e.statsStarted <- struct{}{}
	<-ctx.Done()
	e.statsStopped <- struct{}{}
	return engine.ContainerStats{}, ctx.Err()
}
func awaitDrawerSignal(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("drawer worker did not synchronize")
	}
}
func runDrawerCommands(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			go runDrawerCommands(c)
		}
	}
}

func TestMasterDetailIntegrationWorkerLifetime(t *testing.T) {
	m := fixtureModel(t)
	e := &synchronizedDrawerEngine{Engine: fake.New(), logsStarted: make(chan struct{}, 4), logsStopped: make(chan struct{}, 4), statsStarted: make(chan struct{}, 4), statsStopped: make(chan struct{}, 4)}
	m.service = services.New(e)
	integrationKey(m, '1', "1")
	for i, c := range m.containers() {
		if c.Running() {
			m.containerCursor = i
			break
		}
	}
	cmd := integrationKey(m, tea.KeyEnter, "")
	go runDrawerCommands(cmd)
	awaitDrawerSignal(t, e.logsStarted)
	awaitDrawerSignal(t, e.statsStarted)
	oldGen, oldID := m.drawerGen, m.drawerID
	integrationKey(m, tea.KeyEscape, "")
	awaitDrawerSignal(t, e.logsStopped)
	awaitDrawerSignal(t, e.statsStopped)
	cmd = integrationKey(m, tea.KeyEnter, "")
	go runDrawerCommands(cmd)
	awaitDrawerSignal(t, e.logsStarted)
	awaitDrawerSignal(t, e.statsStarted)
	m.Update(drawerStatsMsg{gen: oldGen, id: oldID})
	if !m.statsLoading {
		t.Fatal("stale stats cleared newer loading state")
	}
	integrationKey(m, '2', "2")
	awaitDrawerSignal(t, e.logsStopped)
	awaitDrawerSignal(t, e.statsStopped)
}

func TestMasterDetailIntegrationEnvTimerAndPortConfirmation(t *testing.T) {
	m := fixtureModel(t)
	integrationKey(m, '1', "1")
	integrationKey(m, tea.KeyEnter, "")
	m.focus.ActiveTab = 3
	m.focus.Area = FocusDrawerControls
	m.drawerEnv.EnvItems = inspector.ParseEnv([]string{"API_TOKEN=acceptance-secret"})
	integrationKey(m, ' ', " ")
	if !m.drawerEnv.EnvItems[0].Revealed {
		t.Fatal("space did not reveal intended row")
	}
	m.drawerEnv.LastActivity = time.Now().Add(-61 * time.Second)
	m.Update(drawerMaskTick{m.drawerGen})
	if m.drawerEnv.EnvItems[0].Revealed {
		t.Fatal("timer failed to remask without keys")
	}
	c := m.containerIns[m.drawerID]
	c.Ports = []string{"0.0.0.0:8080->80/tcp"}
	m.containerIns[m.drawerID] = c
	m.focus.ControlIdx = 1
	integrationKey(m, tea.KeyEnter, "")
	if m.overlay != "port-preview" {
		t.Fatal("port activation did not require confirmation")
	}
	_, content := m.overlayContent()
	if !strings.Contains(content, "http://127.0.0.1:8080") || !strings.Contains(content, "non-loopback") {
		t.Fatalf("preview lacks independent URL and warning: %s", content)
	}
}

func integrationKey(m *Model, code rune, text string) tea.Cmd {
	_, cmd := m.Update(tea.KeyPressMsg{Code: code, Text: text})
	return cmd
}

func TestMasterDetailIntegration(t *testing.T) {
	m := fixtureModel(t)
	integrationKey(m, '1', "1")
	if strings.Contains(m.Render(), "Health & Diagnostics") {
		t.Fatal("drawer visible on launch")
	}
	integrationKey(m, tea.KeyEnter, "")
	if !strings.Contains(m.Render(), "Health & Diagnostics") {
		t.Fatal("Enter did not render five-tab drawer")
	}
	for _, label := range []string{"Live Logs", "Metrics", "Env & Ports", "Timeline"} {
		if !strings.Contains(m.Render(), label) {
			t.Errorf("missing drawer tab %s", label)
		}
	}
	integrationKey(m, tea.KeyTab, "")
	integrationKey(m, tea.KeyRight, "")
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	if !strings.Contains(m.Render(), "[Esc] Back") {
		t.Fatal("narrow drawer lacks overlay back indicator")
	}
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	integrationKey(m, tea.KeyEscape, "")
	if strings.Contains(m.Render(), "Health & Diagnostics") {
		t.Fatal("Escape did not close drawer")
	}
}

func TestMasterDetailIntegrationRatioPersistence(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := fixtureModel(t)
	integrationKey(m, '1', "1")
	integrationKey(m, tea.KeyEnter, "")
	cmd := integrationKey(m, ']', "]")
	runDrawerCommands(cmd)
	n := fixtureModel(t)
	if n.drawerRatio < .399 || n.drawerRatio > .401 {
		t.Fatalf("ratio did not persist: %v", n.drawerRatio)
	}
}

func TestMasterDetailIntegrationGroupMapping(t *testing.T) {
	m := fixtureModel(t)
	integrationKey(m, '1', "1")
	integrationKey(m, 'g', "g")
	if !strings.Contains(m.Render(), "storefront") {
		t.Fatal("compose group not visible")
	}
	integrationKey(m, tea.KeyEnter, "")
	if m.focus.DrawerOpen {
		t.Fatal("group header activation opened wrong container")
	}
	integrationKey(m, 's', "s")
	if m.overlay != "stack-stop" {
		t.Fatal("group stop lacks explicit confirmation")
	}
	_, text := m.overlayContent()
	if !strings.Contains(text, "storefront") || !strings.Contains(text, "volume") {
		t.Fatal("stack confirmation lacks named target and volume policy")
	}
}

func TestMasterDetailIntegrationLatchAndBackgroundIdentity(t *testing.T) {
	m := fixtureModel(t)
	integrationKey(m, '1', "1")
	integrationKey(m, tea.KeyEnter, "")
	id, gen := m.drawerID, m.drawerGen
	m.Update(drawerInspectMsg{gen: gen, id: id, data: engine.ContainerInspection{ID: id, State: "running"}})
	if m.focus.ActiveTab != 1 {
		t.Fatal("running inspection did not latch Logs")
	}
	m.Update(drawerInspectMsg{gen: gen, id: id, data: engine.ContainerInspection{ID: id, State: "exited", ExitCode: 137}})
	if m.focus.ActiveTab != 1 || !strings.Contains(m.Render(), "Health & Diagnostics (ERR)") {
		t.Fatal("health update hijacked tab or omitted ERR badge")
	}
	m.containerFilter = ""
	m.containerCursor = min(2, len(m.containers())-1)
	selected := m.containers()[m.containerCursor].ID
	snap := store.Snapshot{Containers: append([]engine.Container{{ID: "new-first", Name: "aaa-first"}}, m.store.Containers...)}
	m.Update(snapshotMsg{data: snap})
	if m.containers()[m.containerCursor].ID != selected {
		t.Fatal("background insertion moved selected identity")
	}
}

func TestMasterDetailIntegrationPreviousAndStaleLogs(t *testing.T) {
	m := fixtureModel(t)
	integrationKey(m, '1', "1")
	integrationKey(m, tea.KeyEnter, "")
	c := m.containerIns[m.drawerID]
	c.StartedAt = time.Now()
	m.containerIns[m.drawerID] = c
	m.Update(drawerInspectMsg{gen: m.drawerGen, id: m.drawerID, data: c})
	m.focus.ActiveTab = 0
	m.focus.Area = FocusDrawerControls
	old := m.logsGen
	integrationKey(m, 'p', "p")
	if !m.drawerLogs.IsPrevious || !m.drawerLogs.PreviousUntil.Equal(c.StartedAt) {
		t.Fatal("previous run boundary missing")
	}
	integrationKey(m, 'l', "l")
	if m.drawerLogs.IsPrevious || !m.drawerLogs.Follow {
		t.Fatal("resume live retained historical state")
	}
	m.Update(logLineMsg{gen: old, entry: engine.LogEntry{Text: "stale"}})
	if m.drawerLogs.Buffer != nil && m.drawerLogs.Buffer.Len() != 0 {
		t.Fatal("stale logs entered fresh buffer")
	}
}

func TestMasterDetailIntegrationMouse(t *testing.T) {
	m := fixtureModel(t)
	integrationKey(m, '1', "1")
	m.mouse(5, tokens.HeaderHeight+tokens.PanelFrame+1)
	if !m.focus.DrawerOpen {
		t.Fatal("container click failed to open drawer")
	}
	l := masterdetail.ComputeLayout(m.width, m.height, true, m.drawerRatio)
	m.mouse(l.ListWidth+3, tokens.HeaderHeight+3)
	if m.focus.ActiveTab != 3 {
		t.Fatal("drawer tab click failed")
	}
}

func TestMasterDetailIntegrationScrolledRoundTrip(t *testing.T) {
	m := fixtureModel(t)
	integrationKey(m, '1', "1")
	m.store.Containers = nil
	for i := 0; i < 60; i++ {
		m.store.Containers = append(m.store.Containers, engine.Container{ID: fmt.Sprintf("id-%02d", i), Name: fmt.Sprintf("service-%02d", i), State: "running"})
	}
	m.containerFilter = "service"
	for i := 0; i < 43; i++ {
		integrationKey(m, tea.KeyDown, "")
	}
	before := m.Render()
	id := m.containers()[m.containerCursor].ID
	integrationKey(m, '2', "2")
	integrationKey(m, '1', "1")
	if m.Render() != before || m.containers()[m.containerCursor].ID != id {
		t.Fatal("scrolled viewport/filter/identity lost")
	}
}

func TestMasterDetailIntegrationControls(t *testing.T) {
	m := fixtureModel(t)
	integrationKey(m, '1', "1")
	integrationKey(m, tea.KeyEnter, "")
	m.focus.ActiveTab = 0
	m.normalizeDrawerControls()
	integrationKey(m, tea.KeyTab, "")
	integrationKey(m, tea.KeyTab, "")
	if m.focus.Area != FocusDrawerControls {
		t.Fatal("Tab did not focus controls")
	}
	m.focus.ControlIdx = 1
	integrationKey(m, tea.KeyEnter, "")
	if m.overlay != "container-action" {
		t.Fatal("restart control lacks confirmation")
	}
}

func TestMasterDetailIntegrationTimedMetrics(t *testing.T) {
	m := fixtureModel(t)
	integrationKey(m, '1', "1")
	integrationKey(m, tea.KeyEnter, "")
	m.focus.ActiveTab = 2
	id, gen := m.drawerID, m.drawerGen
	at := time.Now()
	m.Update(drawerInspectMsg{gen: gen, id: id, data: engine.ContainerInspection{ID: id, State: "running"}})
	m.focus.ActiveTab = 2
	m.Update(drawerStatsMsg{request: m.drawerStatsRequest, gen: gen, id: id, data: engine.ContainerStats{CPUAvailable: true, At: at, SystemTotal: 100, CPUTotal: 10, OnlineCPUs: 1}})
	m.Update(drawerStatsMsg{request: m.drawerStatsRequest, gen: gen, id: id, data: engine.ContainerStats{CPUAvailable: true, At: at.Add(time.Second), SystemTotal: 200, CPUTotal: 20, OnlineCPUs: 1}})
	text := m.Render()
	if !strings.Contains(text, "60s") || !strings.Contains(text, "5min") || !strings.Contains(text, "Collection started") {
		t.Fatal("metrics lacks timed session windows/provenance")
	}
}

func TestMasterDetailIntegrationHealthCommandReveal(t *testing.T) {
	m := fixtureModel(t)
	integrationKey(m, '1', "1")
	integrationKey(m, tea.KeyEnter, "")
	c := engine.ContainerInspection{ID: m.drawerID, State: "running"}.WithDiagnosticEvidence(engine.DiagnosticEvidence{HealthConfig: &engine.HealthCheckConfig{Command: []string{"CMD-SHELL", "curl /health"}}})
	m.containerIns[m.drawerID] = c
	m.focus.ActiveTab = 0
	m.focus.Area = FocusDrawerControls
	integrationKey(m, 'c', "c")
	if m.overlay != "health-command" {
		t.Fatal("c did not open explicit safe health command viewer")
	}
	_, text := m.overlayContent()
	if strings.Contains(text, "curl /health") {
		t.Fatal("command shown before explicit reveal")
	}
	integrationKey(m, tea.KeyEnter, "")
	_, text = m.overlayContent()
	if !strings.Contains(text, "curl /health") {
		t.Fatal("confirmed command unavailable")
	}
}

func TestMasterDetailIntegrationStableBackgroundOrder(t *testing.T) {
	m := fixtureModel(t)
	integrationKey(m, '1', "1")
	before := m.containers()
	first := before[0].ID
	snap := store.Snapshot{Containers: append([]engine.Container(nil), m.store.Containers...)}
	for i := range snap.Containers {
		if snap.Containers[i].ID == first {
			snap.Containers[i].Name = "zzzz-renamed"
		}
	}
	m.Update(snapshotMsg{data: snap})
	if m.containers()[0].ID != first {
		t.Fatal("background rename reordered existing rows")
	}
}

func TestMasterDetailIntegrationCaptureAndRemask(t *testing.T) {
	m := fixtureModel(t)
	integrationKey(m, '1', "1")
	integrationKey(m, tea.KeyEnter, "")
	c := m.containerIns[m.drawerID]
	c.ID = m.drawerID
	m.containerIns[m.drawerID] = c
	m.focus.ActiveTab = 3
	m.focus.Area = FocusDrawerControls
	m.drawerEnv.EnvItems = inspector.ParseEnv([]string{"API_TOKEN=capture-secret"})
	m.drawerEnv.RevealAll(time.Now())
	if !strings.Contains(m.Render(), "capture-secret") {
		t.Fatal("explicit reveal not rendered")
	}
	if strings.Contains(m.RenderDiagnostic(), "capture-secret") {
		t.Fatal("diagnostic capture leaked revealed env")
	}
	if !m.drawerEnv.EnvItems[0].Revealed {
		t.Fatal("capture mutated live reveal state")
	}
	integrationKey(m, tea.KeyEscape, "")
	if len(m.drawerEnv.EnvItems) != 0 {
		t.Fatal("close retained raw environment")
	}
}

func TestMasterDetailIntegrationControlNormalizationAndZoom(t *testing.T) {
	m := fixtureModel(t)
	integrationKey(m, '1', "1")
	integrationKey(m, tea.KeyEnter, "")
	m.focus.ControlIdx = 99
	m.focus.ActiveTab = 0
	m.normalizeDrawerControls()
	if m.focus.ControlIdx != 2 {
		t.Fatal("ControlMax count not normalized")
	}
	integrationKey(m, 'z', "z")
	integrationKey(m, tea.KeyEscape, "")
	if m.focus.Zoomed || !m.focus.DrawerOpen {
		t.Fatal("first Escape must unzoom only")
	}
	integrationKey(m, tea.KeyEscape, "")
	integrationKey(m, 'z', "z")
	if m.focus.DrawerOpen || m.focus.Zoomed {
		t.Fatal("closed drawer zoomed")
	}
}
