package app

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"fmt"
	"github.com/Mersad-Moghaddam/lenger/internal/core/services"
	"github.com/Mersad-Moghaddam/lenger/internal/core/store"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/engine/docker"
	"github.com/charmbracelet/x/ansi"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWholeSpaceResourceRouting(t *testing.T) {
	for tab := 0; tab < 4; tab++ {
		t.Run(string(rune('0'+tab)), func(t *testing.T) {
			m := fixtureModel(t)
			m.tab = tab
			integrationKey(m, tea.KeySpace, "")
			count := 0
			for _, v := range m.selectedContain {
				if v {
					count++
				}
			}
			if (tab == 0 && count != 1) || (tab != 0 && count != 0) {
				t.Fatalf("tab %d selected %d containers", tab, count)
			}
		})
	}
}

func TestWholeListTelemetryLifecycle(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	m.store.Containers = []engine.Container{{ID: "a", Name: "a", State: "running"}}
	m.containerCursor = 0
	m.containerFilter = ""
	at := time.Now()
	m.clock = at
	m.stats["a"] = engine.ContainerStats{At: at, Memory: 123456789, MemoryAvailable: true, CPUAvailable: true}
	m.statCPU["a"] = 42
	integrationKey(m, tea.KeyEnter, "")
	m.stats["a"] = engine.ContainerStats{At: at, Memory: 123456789, MemoryAvailable: true, CPUAvailable: true}
	m.statCPU["a"] = 42
	integrationKey(m, tea.KeyEsc, "")
	body := ansi.Strip(m.renderContainers(160, 25))
	if strings.Contains(body, "42.0%") || strings.Contains(body, "117.7") || strings.Contains(body, "123.5") {
		t.Fatal("closed drawer telemetry rendered current: " + body)
	}
	integrationKey(m, tea.KeyEnter, "")
	m.stats["a"] = engine.ContainerStats{At: at, Memory: 123456789, MemoryAvailable: true, CPUAvailable: true}
	m.statCPU["a"] = 42
	m.Update(drawerStatsMsg{gen: m.drawerGen, id: m.drawerID, request: m.drawerStatsRequest, err: fmt.Errorf("sample failed")})
	body = ansi.Strip(m.renderContainers(160, 25))
	if strings.Contains(body, "42.0%") || strings.Contains(body, "123.5") {
		t.Fatal("error-after-success rendered current: " + body)
	}
}

func TestWholeFilteredStackAndIdentity(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	m.store.Containers = []engine.Container{{ID: "a", Name: "a", Project: "stack"}, {ID: "b", Name: "b", Labels: map[string]string{"com.docker.compose.project": "stack"}}, {ID: "c", Name: "c", Project: "other"}}
	m.grouped = true
	m.containerFilter = "a"
	m.groupCursor = 0
	integrationKey(m, tea.KeyExtended, "s")
	if len(m.stackIDs) != 2 {
		t.Fatalf("stack targets %v", m.stackIDs)
	}
	if m.actionName != engine.ActionStop || m.stackIDs[0] != "a" || m.stackIDs[1] != "b" {
		t.Fatalf("not exact canonical stop: %s %v", m.actionName, m.stackIDs)
	}
	integrationKey(m, tea.KeyEsc, "")
	integrationKey(m, tea.KeyExtended, "r")
	if m.actionName != engine.ActionRestart || len(m.stackIDs) != 2 {
		t.Fatal("restart did not target whole project")
	}
	m.closeOverlay()
	m.containerFilter = ""
	m.groupCursor = 2
	m.containerCursor = 1
	// Rows: stack header,a,b,other header,c. Preserve b when a disappears.
	m.Update(snapshotMsg{data: store.Snapshot{Containers: []engine.Container{{ID: "b", Name: "b", Project: "stack"}}}})
	rows := m.groupedRows()
	if rows[m.groupCursor].id != "b" {
		t.Fatalf("identity drift: %#v cursor %d", rows, m.groupCursor)
	}
}

func TestWholeHealthRestartIsSingular(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	integrationKey(m, tea.KeyEnter, "")
	id := m.drawerID
	for _, c := range m.containers() {
		if c.ID != id {
			m.selectedContain[c.ID] = true
			break
		}
	}
	m.focus.ActiveTab = 0
	m.focus.Area = FocusDrawerControls
	m.focus.ControlIdx = 1
	integrationKey(m, tea.KeyEnter, "")
	if len(m.actionIDs) != 1 || m.actionIDs[0] != id {
		t.Fatalf("Health targets %v; inspected %s", m.actionIDs, id)
	}
}

func TestWholeResourcePruneRouting(t *testing.T) {
	for tab, kind := range map[int]services.CleanupKind{1: services.CleanupDanglingImages, 2: services.CleanupUnusedVolumes} {
		m := fixtureModel(t)
		m.tab = tab
		integrationKey(m, tea.KeyExtended, "p")
		kinds := m.cleanupKinds()
		if m.overlay != "cleanup" || len(kinds) != 1 || kinds[0] != kind {
			t.Fatalf("tab %d: %s %v", tab, m.overlay, kinds)
		}
		integrationKey(m, tea.KeyDown, "")
		integrationKey(m, tea.KeySpace, "")
		if len(m.cleanupKinds()) != 1 || m.cleanupKinds()[0] != kind {
			t.Fatal("resource prune escaped category scope")
		}
		integrationKey(m, tea.KeyEnter, "")
		if m.overlay != "cleanup-confirm" {
			t.Fatal("missing confirmation")
		}
		if tab == 2 {
			integrationKey(m, tea.KeyEnter, "")
			if m.overlay != "cleanup-confirm" {
				t.Fatal("volumes deleted without typed confirmation")
			}
		}
		integrationKey(m, tea.KeyEsc, "")
		integrationKey(m, tea.KeyEsc, "")
		if m.overlay != "" {
			t.Fatal("cancel did not close")
		}
	}
}

func TestWholeNarrowNetworkEnterEscape(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 3
	m.width = 80
	m.height = 24
	var endpoints []engine.NetworkEndpoint
	for i := 0; i < 60; i++ {
		endpoints = append(endpoints, engine.NetworkEndpoint{Name: fmt.Sprintf("endpoint-%02d", i), IPv4: fmt.Sprintf("172.28.0.%d/16", i+2), IPv6: "2001:db8::1234/64"})
	}
	m.store.Networks = []engine.Network{{ID: "n", Name: "test-network", Subnets: []string{"172.28.0.0/16"}, Gateways: []string{"172.28.0.1"}, Containers: endpoints}}
	m.networkFilter = "test-network"
	integrationKey(m, tea.KeyEnter, "")
	if !m.detail {
		t.Fatal("network Enter did not open detail")
	}
	if !strings.Contains(m.Render(), "172.28.0.0/16") {
		t.Fatal("missing subnet in narrow detail")
	}
	for range 65 {
		integrationKey(m, tea.KeyDown, "")
	}
	if !strings.Contains(m.Render(), "endpoint-59") {
		t.Fatal("last endpoint inaccessible by scrolling")
	}
	if m.networkCursor != 0 {
		t.Fatal("detail navigation moved list selection")
	}
	integrationKey(m, tea.KeyEsc, "")
	if m.detail || m.networkFilter != "test-network" || m.networkCursor != 0 {
		t.Fatal("Esc did not preserve network context")
	}
}

func TestWholeDockerPortsReachDrawer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"Id":"port-container","State":{"Status":"running"},"NetworkSettings":{"Ports":{"80/tcp":[{"HostIp":"0.0.0.0","HostPort":"8080"}],"81/tcp":[{"HostIp":"::","HostPort":"8081"}],"53/udp":[{"HostIp":"0.0.0.0","HostPort":"5353"}]}}}`)
	}))
	defer server.Close()
	t.Setenv("DOCKER_HOST", server.URL)
	t.Setenv("DOCKER_API_VERSION", "1.52")
	t.Setenv("DOCKER_CERT_PATH", "")
	t.Setenv("DOCKER_TLS_VERIFY", "")
	e, err := docker.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := e.Close(); err != nil {
			t.Error(err)
		}
	})
	inspected, err := e.InspectContainer(context.Background(), "port-container")
	if err != nil {
		t.Fatal(err)
	}
	m := fixtureModel(t)
	m.tab = 0
	m.store.Containers = []engine.Container{{ID: "port-container", Name: "ports", State: "running"}}
	m.containerCursor = 0
	m.containerFilter = ""
	integrationKey(m, tea.KeyEnter, "")
	m.Update(drawerInspectMsg{gen: m.drawerGen, id: m.drawerID, data: inspected})
	m.focus.ActiveTab = 3
	m.focus.Area = FocusDrawerControls
	for i, p := range inspected.Ports {
		if strings.Contains(p, "8080") {
			m.focus.ControlIdx = i
		}
	}
	rows, _ := m.envVisualRows(100)
	text := strings.Join(rows, "\n")
	if !strings.Contains(text, "http://127.0.0.1:8080") || !strings.Contains(text, "http://[::1]:8081") || strings.Count(text, "Security") < 3 {
		t.Fatalf("adapter drawer content: %s", text)
	}
	integrationKey(m, tea.KeyEnter, "")
	if m.overlay != "port-preview" {
		t.Fatalf("port did not reach preview: %s", m.overlay)
	}
	integrationKey(m, tea.KeyEsc, "")
}

func TestWholeGroupedSnapshotFilterCollapseIdentity(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	m.store.Containers = []engine.Container{{ID: "a", Name: "a", Project: "stack"}, {ID: "b", Name: "b", Project: "stack"}, {ID: "c", Name: "zzonly", Project: "z"}}
	m.containerFilter = ""
	m.containerOrder = map[string]int{"a": 0, "b": 1, "c": 2}
	m.grouped = true
	m.groupCursor = 2
	m.containerCursor = 1
	integrationKey(m, tea.KeyEnter, "")
	id := m.drawerID
	m.Update(snapshotMsg{data: store.Snapshot{Containers: []engine.Container{{ID: "b", Name: "b", Project: "stack"}, {ID: "c", Name: "zzonly", Project: "z"}}}})
	if m.currentGroupedRow().id != id || m.drawerID != id {
		t.Fatalf("snapshot highlight/drawer diverged: %#v %s", m.currentGroupedRow(), m.drawerID)
	}
	integrationKey(m, tea.KeyExtended, "r")
	if len(m.actionIDs) != 1 || m.actionIDs[0] != id {
		t.Fatalf("snapshot action %v", m.actionIDs)
	}
	m.closeOverlay()
	m.overlay = "filter"
	m.input.SetValue("b")
	integrationKey(m, tea.KeyEnter, "")
	if m.currentGroupedRow().id != id || m.drawerID != id {
		t.Fatal("filter lost surviving service")
	}
	m.setActiveFilter("zzonly")
	m.rebindDrawerSelection()
	if m.focus.DrawerOpen || m.currentGroupedRow().id != "" {
		t.Fatal("missing service did not fall back to header and close drawer")
	}
	integrationKey(m, tea.KeyEnter, "")
	if !m.groupCollapsed["z"] || m.currentGroupedRow().project != "z" {
		t.Fatal("collapse lost header")
	}
	integrationKey(m, tea.KeyEnter, "")
	if m.groupCollapsed["z"] {
		t.Fatal("header did not expand")
	}
}
