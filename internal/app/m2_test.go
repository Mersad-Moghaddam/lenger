package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

func TestProjectsNavigationAndComposeAction(t *testing.T) {
	m := fixtureModel(t)
	// Switch to projects tab
	_, _ = m.Update(tea.KeyPressMsg{Code: '4', Text: "4"})
	if m.tab != 3 {
		t.Fatalf("expected tab 3 for Projects, got %d", m.tab)
	}

	projs := m.projects()
	if len(projs) == 0 {
		t.Fatal("expected discovered Compose projects in fixture")
	}
	if projs[0].Name != "storefront" {
		t.Fatalf("expected storefront project, got %s", projs[0].Name)
	}

	// Press u to trigger Compose Up
	_, _ = m.Update(tea.KeyPressMsg{Code: 'u', Text: "u"})
	if m.overlay != "compose-action" {
		t.Fatalf("expected compose-action overlay, got %s", m.overlay)
	}
	if m.composeAction != engine.ComposeUp {
		t.Fatalf("expected ComposeUp action, got %s", m.composeAction)
	}

	// Cancel overlay
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.overlay != "" {
		t.Fatalf("expected overlay closed on Esc, got %s", m.overlay)
	}

	// Press d to trigger Compose Down
	_, _ = m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if m.overlay != "compose-action" || m.composeAction != engine.ComposeDown {
		t.Fatalf("expected compose down overlay, got %s / %s", m.overlay, m.composeAction)
	}
	if m.composeVolumes {
		t.Fatal("composeVolumes should default to false")
	}

	// Press v to toggle volume removal
	_, _ = m.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	if !m.composeVolumes {
		t.Fatal("expected composeVolumes to toggle to true on 'v'")
	}

	// Confirm action
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command scheduled on compose confirm")
	}
	if m.overlay != "" {
		t.Fatal("expected overlay closed on compose confirm")
	}
}

func TestNetworksNavigationAndCreation(t *testing.T) {
	m := fixtureModel(t)
	// Switch to networks tab (6 key -> index 5)
	_, _ = m.Update(tea.KeyPressMsg{Code: '6', Text: "6"})
	if m.tab != 5 {
		t.Fatalf("expected tab 5 for Networks, got %d", m.tab)
	}

	nets := m.networks()
	if len(nets) == 0 {
		t.Fatal("expected networks in fixture")
	}

	// Press c to trigger Create Network
	_, _ = m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	if m.overlay != "create-network" {
		t.Fatalf("expected create-network overlay, got %s", m.overlay)
	}
	if m.createNetDriver != "bridge" {
		t.Fatalf("expected default driver bridge, got %s", m.createNetDriver)
	}

	// Press tab to toggle driver to overlay
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.createNetDriver != "overlay" {
		t.Fatalf("expected driver toggled to overlay, got %s", m.createNetDriver)
	}

	// Set network name and confirm
	m.input.SetValue("my-mesh")
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command scheduled on create network confirm")
	}
	if m.overlay != "" {
		t.Fatal("expected overlay closed after create confirm")
	}

	// Pressing 'd' on a network with attached containers notifies safety warning
	// Find storefront_default which has containers
	for i, n := range m.networks() {
		if len(n.Containers) > 0 {
			m.networkCursor = i
			break
		}
	}
	_, _ = m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if m.overlay == "remove-network" {
		t.Fatal("expected safety check to block removing network with active containers")
	}
	if !strings.Contains(m.toast, "active containers attached") {
		t.Fatalf("expected safety toast warning, got %q", m.toast)
	}
}

func TestSystemInfoTabLoadsDaemonTelemetry(t *testing.T) {
	m := fixtureModel(t)
	// Switch to system tab (7 key -> index 6)
	_, cmd := m.Update(tea.KeyPressMsg{Code: '7', Text: "7"})
	if cmd == nil || m.tab != 6 {
		t.Fatalf("expected tab 6 for System with commands scheduled, got %d", m.tab)
	}

	// Feed daemon info message
	daemon := engine.DaemonInfo{
		ServerVersion: "27.3.1",
		APIVersion:    "1.47",
		OS:            "Ubuntu 24.04.1 LTS",
		Arch:          "x86_64",
		KernelVersion: "6.11.0-17-generic",
		Driver:        "overlay2",
		CgroupDriver:  "systemd",
		CgroupVersion: "2",
		NCPU:          16,
		MemTotal:      34_359_738_368,
	}
	_, _ = m.Update(daemonInfoMsg{data: daemon, err: nil})
	if !m.daemonLoaded || m.daemonInfo.ServerVersion != "27.3.1" {
		t.Fatalf("daemonInfo not updated in model: %#v", m.daemonInfo)
	}

	// Render view
	text := m.Render()
	for _, want := range []string{"DAEMON INFORMATION", "27.3.1", "Ubuntu 24.04.1 LTS", "x86_64", "16 CPUs"} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered system tab missing %q:\n%s", want, text)
		}
	}
}
