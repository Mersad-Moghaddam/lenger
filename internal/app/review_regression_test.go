package app

import (
	tea "charm.land/bubbletea/v2"
	"errors"
	"fmt"
	"github.com/Mersad-Moghaddam/lenger/internal/core/services"
	"github.com/Mersad-Moghaddam/lenger/internal/core/store"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/engine/fake"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/masterdetail"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/views/inspector"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
	"time"
)

func TestReviewI1ListOnlyGeometry(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	for _, w := range []int{120, 160, 300} {
		m.width = w
		body := m.renderContainers(w, 30)
		first := strings.Split(ansi.Strip(body), "\n")[0]
		if strings.Count(first, "┌")+strings.Count(first, "+") > 2 {
			t.Fatalf("nested legacy split at %d: %s", w, first)
		}
		if strings.Contains(body, "RESOURCES") {
			t.Fatal("closed drawer contains legacy detail")
		}
		if !strings.Contains(first, strings.Repeat("─", w/2)) && !strings.Contains(first, strings.Repeat("-", w/2)) {
			t.Fatalf("list not full width %d: %s", w, first)
		}
	}
}

func TestReviewI2SelectionRebind(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	integrationKey(m, tea.KeyEnter, "")
	old := m.drawerID
	items := m.containers()
	idx := 1
	if items[idx].ID == old {
		idx = 2
	}
	id := items[idx].ID
	integrationKey(m, tea.KeyExtended, "container:"+id)
	if m.drawerID != id || !m.focus.DrawerOpen {
		t.Fatal("palette did not rebind drawer")
	}
	m.overlay = "filter"
	m.input.SetValue("no-matches-ever")
	integrationKey(m, tea.KeyEnter, "")
	if m.focus.DrawerOpen {
		t.Fatal("empty filter retained old target")
	}
}

func TestReviewI3StatsRequestReplacement(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	integrationKey(m, tea.KeyEnter, "")
	// Both requests belong to the same drawer generation; their completion order is inverted.
	m.statsLoading = false
	first := m.loadStats()
	if first == nil {
		t.Fatal("no first sample")
	}
	old := first().(drawerStatsMsg)
	m.Update(drawerInspectMsg{gen: m.drawerGen, id: m.drawerID, data: engine.ContainerInspection{ID: m.drawerID, State: "exited"}})
	m.statsLoading = false
	next := m.loadStats()
	if next == nil {
		t.Fatal("no replacement sample")
	}
	canceled := make(chan struct{})
	original := m.drawerStatsCancel
	m.drawerStatsCancel = func() { original(); close(canceled) }
	m.Update(old)
	if !m.statsLoading {
		t.Fatal("old result cleared replacement loading")
	}
	select {
	case <-canceled:
		t.Fatal("old result canceled replacement")
	default:
	}
}

func TestReviewI4TelemetryErrorAndElapsedWindow(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	integrationKey(m, tea.KeyEnter, "")
	m.focus.ActiveTab = 2
	id := m.drawerID
	m.containerIns[id] = engine.ContainerInspection{ID: id, State: "running"}
	at := time.Now().Add(-10 * time.Minute)
	m.metricSamples[id] = []timedMetric{{at, 83}}
	m.stats[id] = engine.ContainerStats{At: at, Memory: 123, MemoryAvailable: true}
	m.Update(drawerStatsMsg{request: m.drawerStatsRequest, gen: m.drawerGen, id: id, err: errors.New("daemon unavailable")})
	if !m.statsError || m.stats[id].MemoryAvailable || len(m.statHistory[id]) != 0 {
		t.Fatal("failed sample retained current availability")
	}
	text := m.Render()
	if !strings.Contains(strings.ToLower(text), "unavailable") {
		t.Fatal("failed sample labeled current")
	}
	if strings.Contains(m.metricWindows(60), "█") {
		t.Fatal("expired sample remains in current window")
	}
}

func TestReviewI5ScrolledGroupMouse(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	m.height = 18
	m.grouped = true
	m.store.Containers = nil
	for i := 0; i < 25; i++ {
		m.store.Containers = append(m.store.Containers, engine.Container{ID: fmt.Sprint(i), Name: fmt.Sprintf("c%02d", i), Project: fmt.Sprintf("p%02d", i)})
	}
	m.groupCursor = 40
	h := m.height - tokens.HeaderHeight - tokens.FooterHeight
	start := max(0, m.groupCursor-(h-3))
	want := m.groupedRows()[start]
	m.mouse(3, tokens.HeaderHeight+1)
	if m.groupCursor != start {
		t.Fatalf("clicked row %d wanted %d (%s)", m.groupCursor, start, want.project)
	}
}

func TestReviewI6ControlFocusGate(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	integrationKey(m, tea.KeyEnter, "")
	m.focus.ActiveTab = 3
	m.focus.Area = FocusDrawerTabs
	m.containerIns[m.drawerID] = engine.ContainerInspection{ID: m.drawerID}
	m.drawerEnv.EnvItems = inspector.ParseEnv([]string{"TOKEN=secret"})
	m.normalizeDrawerControls()
	integrationKey(m, tea.KeyEnter, "")
	if m.drawerEnv.EnvItems[0].Revealed {
		t.Fatal("tab bar Enter revealed control")
	}
	integrationKey(m, tea.KeyTab, "")
	if !strings.Contains(m.Render(), "> TOKEN=") {
		t.Fatal("focused row lacks visible identity")
	}
}

func TestReviewI7LongCommandAccessible(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	integrationKey(m, tea.KeyEnter, "")
	m.overlay = "health-command"
	m.healthCommandRevealed = true
	m.containerIns[m.drawerID] = engine.ContainerInspection{ID: m.drawerID}.WithDiagnosticEvidence(engine.DiagnosticEvidence{HealthConfig: &engine.HealthCheckConfig{Command: []string{strings.Repeat("x", 300) + "TAIL_EVIDENCE"}}})
	_, text := m.overlayContent()
	for _, line := range strings.Split(ansi.Strip(text), "\n") {
		if len(line) > 75 {
			t.Fatal("long evidence not visually wrapped")
		}
	}
	if !strings.Contains(strings.ReplaceAll(text, "\n", ""), "TAIL_EVIDENCE") {
		t.Fatal("evidence tail lost")
	}
}

func TestReviewI8PendingInspectPrevious(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	integrationKey(m, tea.KeyEnter, "")
	m.focus.ActiveTab = 0
	m.focus.Area = FocusDrawerControls
	delete(m.containerIns, m.drawerID)
	gen, id := m.drawerGen, m.drawerID
	cmd := integrationKey(m, 'p', "p")
	if cmd != nil {
		t.Fatal("historical request issued before actual boundary")
	}
	if m.drawerLatch {
		t.Fatal("explicit previous navigation retained latch")
	}
	at := time.Now()
	_, cmd = m.Update(drawerInspectMsg{gen: gen, id: id, data: engine.ContainerInspection{ID: id, State: "exited", StartedAt: at}})
	if m.focus.ActiveTab != 1 || !m.drawerLogs.PreviousUntil.Equal(at) || cmd == nil {
		t.Fatal("pending previous request did not use fresh boundary")
	}
}

func TestReviewSelectionRemovalAndFilterTarget(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	integrationKey(m, tea.KeyEnter, "")
	c := m.containers()[1]
	m.overlay = "filter"
	m.input.SetValue(c.Name)
	integrationKey(m, tea.KeyEnter, "")
	if m.drawerID != c.ID || m.containerIDs()[0] != m.drawerID {
		t.Fatal("filtered action/drawer target differs")
	}
	snap := store.Snapshot{}
	for _, item := range m.store.Containers {
		if item.ID != c.ID {
			snap.Containers = append(snap.Containers, item)
		}
	}
	m.Update(snapshotMsg{data: snap})
	if m.focus.DrawerOpen {
		t.Fatal("removed selected container retained drawer")
	}
}

func TestReviewEnvWrappedHitMap(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	integrationKey(m, tea.KeyEnter, "")
	m.focus.ActiveTab = 3
	m.focus.Area = FocusDrawerControls
	m.containerIns[m.drawerID] = engine.ContainerInspection{ID: m.drawerID}
	m.drawerEnv.EnvItems = inspector.ParseEnv([]string{"LONG=" + strings.Repeat("x", 200), "TOKEN=second"})
	m.normalizeDrawerControls()
	rows, owners := m.envVisualRows(54)
	if len(rows) != len(owners) {
		t.Fatal("geometry diverged")
	}
	for i, owner := range owners {
		if owner == 0 && i > 1 {
			m.drawerScroll = i
			m.width = 87
			l := masterdetail.ComputeLayout(m.width, m.height, true, m.drawerRatio)
			m.mouse(l.ListWidth+3, tokens.HeaderHeight+5)
			if !m.drawerEnv.EnvItems[0].Revealed || m.drawerEnv.EnvItems[1].Revealed {
				t.Fatal("wrapped continuation activated wrong env")
			}
			return
		}
	}
	t.Fatal("missing wrapped continuation")
}

func TestReviewLongTimelineVisualScroll(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	integrationKey(m, tea.KeyEnter, "")
	m.focus.ActiveTab = 4
	m.focus.Area = FocusDrawerControls
	id := m.drawerID
	m.containerIns[id] = engine.ContainerInspection{ID: id, Created: time.Now()}
	m.eventHistory = []engine.Event{{ID: id, Kind: "container", Action: "start", At: time.Now()}}
	m.focus.Zoomed = true
	m.width = 80
	text := m.Render()
	if !strings.Contains(text, "Timeline & Events") {
		t.Fatal("timeline missing")
	}
	// Timestamp plus outcome exceeds the 77-cell default pane. Its final words
	// must survive in a subsequent visual row, rather than Fit clipping them.
	if !strings.Contains(strings.Join(strings.Fields(ansi.Strip(text)), ""), "Observeddaemonevent") {
		t.Fatalf("wrapped timeline outcome inaccessible: %s", text)
	}
}

func TestReviewActiveEnvTabsDoNotRefetchAfterRemask(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	integrationKey(m, tea.KeyEnter, "")
	id := m.drawerID
	m.drawerEnvFetched = true
	m.drawerEnv.EnvItems = inspector.ParseEnv([]string{"TOKEN=secret"})
	m.drawerEnv.RevealAll(time.Now().Add(-61 * time.Second))
	m.focus.ActiveTab = 3
	m.focus.Area = FocusDrawerTabs
	integrationKey(m, tea.KeyTab, "")
	m.Update(drawerMaskTick{m.drawerGen})
	if !m.drawerEnv.EnvItems[0].Revealed {
		t.Fatal("active control navigation treated as idle")
	}
	m.drawerEnv.LastActivity = time.Now().Add(-61 * time.Second)
	m.Update(drawerMaskTick{m.drawerGen})
	_, cmd := m.Update(drawerInspectMsg{gen: m.drawerGen, id: id, data: engine.ContainerInspection{ID: id, State: "running"}})
	if cmd != nil {
		t.Fatal("background inspect refetched raw env after auto-remask")
	}
}

func TestReviewRound2GroupedServiceMouse(t *testing.T) {
	for _, drawerFocused := range []bool{false, true} {
		t.Run(fmt.Sprint(drawerFocused), func(t *testing.T) {
			m := fixtureModel(t)
			m.tab = 0
			m.height = 18
			m.grouped = true
			m.store.Containers = nil
			for i := 0; i < 25; i++ {
				m.store.Containers = append(m.store.Containers, engine.Container{ID: fmt.Sprint(i), Name: fmt.Sprintf("c%02d", i), Project: fmt.Sprintf("p%02d", i)})
			}
			if drawerFocused {
				integrationKey(m, tea.KeyEnter, "")
				m.focus.Area = FocusDrawerControls
			}
			m.groupCursor = 40
			h := m.height - tokens.HeaderHeight - tokens.FooterHeight
			start := m.groupWindowStart(h)
			rows := m.groupedRows()
			idx := start
			if rows[idx].id == "" {
				idx++
			}
			target := rows[idx].id
			m.mouse(3, tokens.HeaderHeight+1+idx-start)
			if m.containers()[m.containerCursor].ID != target || m.drawerID != target || !m.focus.DrawerOpen || m.focus.Area != FocusList {
				t.Fatalf("group click target=%s selection=%s drawer=%s focus=%v", target, m.containers()[m.containerCursor].ID, m.drawerID, m.focus.Area)
			}
			if ids := m.containerIDs(); len(ids) != 1 || ids[0] != target {
				t.Fatalf("action targets %v instead of %s", ids, target)
			}
			old := m.groupCursor
			m.mouse(3, m.height-tokens.FooterHeight)
			if m.groupCursor != old {
				t.Fatal("footer click changed selection")
			}
		})
	}
}

func TestReviewRound2GlobalLogsOffset(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	integrationKey(m, tea.KeyEnter, "")
	m.focus.ActiveTab = 4
	m.drawerScroll = 7
	m.drawerOffsets[1] = 3
	m.focus.Area = FocusList
	integrationKey(m, 'l', "l")
	if m.drawerOffsets[4] != 7 || m.drawerScroll != 3 || m.focus.ActiveTab != 1 {
		t.Fatal("global Logs entry lost per-tab offsets")
	}
}

func TestReviewStatsCanceledCompletionAfterReplacement(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	e := &synchronizedDrawerEngine{Engine: fake.New(), logsStarted: make(chan struct{}, 4), logsStopped: make(chan struct{}, 4), statsStarted: make(chan struct{}, 4), statsStopped: make(chan struct{}, 4)}
	m.service = services.New(e)
	integrationKey(m, tea.KeyEnter, "")
	// Opening creates request 1; isolate request 2 so its cancellation is delayed
	// until a replacement has actually entered the engine.
	m.drawerStatsCancel()
	m.statsLoading = false
	first := m.loadStats()
	done := make(chan drawerStatsMsg, 1)
	go func() { done <- first().(drawerStatsMsg) }()
	awaitDrawerSignal(t, e.statsStarted)
	m.Update(drawerInspectMsg{gen: m.drawerGen, id: m.drawerID, data: engine.ContainerInspection{ID: m.drawerID, State: "exited"}})
	awaitDrawerSignal(t, e.statsStopped)
	second := m.loadStats()
	nextDone := make(chan drawerStatsMsg, 1)
	go func() { nextDone <- second().(drawerStatsMsg) }()
	awaitDrawerSignal(t, e.statsStarted)
	m.Update(<-done)
	if !m.statsLoading {
		t.Fatal("canceled completion stole live replacement")
	}
	integrationKey(m, tea.KeyEscape, "")
	awaitDrawerSignal(t, e.statsStopped)
	select {
	case <-nextDone:
	case <-time.After(3 * time.Second):
		t.Fatal("replacement did not complete")
	}
}
