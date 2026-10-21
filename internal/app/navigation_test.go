package app

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Mersad-Moghaddam/lenger/internal/config"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/components"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
)

func TestNavigationResourceTabsAndContextPreservation(t *testing.T) {
	expectedTabs := []string{"Containers", "Images", "Volumes", "Networks"}
	gotTabs := components.TabNames()
	if !reflect.DeepEqual(gotTabs, expectedTabs) {
		t.Fatalf("expected tabs %v, got %v", expectedTabs, gotTabs)
	}

	// Verify New() initializes with default tab 0 (Containers)
	rawModel := New(context.Background(), nil, config.Defaults(), tokens.Palette{})
	if rawModel.tab != 0 {
		t.Fatalf("expected New() to default to tab 0 (Containers), got %d", rawModel.tab)
	}

	m := fixtureModel(t)
	m.tab = 0
	if m.tab != 0 {
		t.Fatalf("expected default tab 0 (Containers), got %d", m.tab)
	}

	// Set container cursor and filter
	m.containerCursor = 1
	m.containerFilter = "redis"

	// Mock active streaming cancellation function on drawer
	streamCanceled := false
	m.drawerCancel = func() {
		streamCanceled = true
	}

	// Switch to tab 1 (Images, shortcut '2')
	m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	if m.tab != 1 {
		t.Fatalf("expected tab 1 (Images) after pressing '2', got %d", m.tab)
	}
	if !streamCanceled {
		t.Fatal("expected drawerCancel to be called when leaving Containers tab")
	}
	if m.drawerCancel != nil {
		t.Fatal("expected drawerCancel to be cleared after switching tabs")
	}

	// Verify container context was preserved
	if m.containerCursor != 1 {
		t.Fatalf("expected containerCursor 1 to be preserved, got %d", m.containerCursor)
	}
	if m.containerFilter != "redis" {
		t.Fatalf("expected containerFilter 'redis' to be preserved, got %q", m.containerFilter)
	}

	// Switch back to tab 0 (Containers, shortcut '1')
	m.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
	if m.tab != 0 {
		t.Fatalf("expected tab 0 (Containers) after pressing '1', got %d", m.tab)
	}
	if m.containerCursor != 1 {
		t.Fatalf("expected containerCursor 1 still preserved, got %d", m.containerCursor)
	}
	if m.containerFilter != "redis" {
		t.Fatalf("expected containerFilter 'redis' still preserved, got %q", m.containerFilter)
	}
}

func TestNavigationFilterTitlesAndContainerSelection(t *testing.T) {
	m := fixtureModel(t)
	for tab, title := range []string{"Filter containers", "Filter images", "Filter volumes", "Filter networks"} {
		m.tab = tab
		m.overlay = "filter"
		got, _ := m.overlayContent()
		if got != title {
			t.Fatalf("tab %d: got %q, want %q", tab, got, title)
		}
	}
}

func TestNavigationCopyEquivalentResourceIndices(t *testing.T) {
	for tab, prefix := range []string{"docker inspect ", "docker image inspect ", "docker volume inspect ", "docker network inspect "} {
		m := fixtureModel(t)
		m.tab = tab
		m.copyEquivalent()
		if !strings.HasPrefix(m.toast, "Copied: "+prefix) {
			t.Errorf("tab %d copied %q, want %q", tab, m.toast, prefix)
		}
	}
}

func TestNavigationPaletteResourceShortcuts(t *testing.T) {
	want := map[string]string{"Containers": "1", "Images": "2", "Volumes": "3", "Networks": "4"}
	for _, a := range actions() {
		if a.name == "Dashboard" || a.name == "Projects" || a.name == "System info" {
			t.Errorf("removed top-level navigation remains: %s", a.name)
		}
		if key, ok := want[a.name]; ok {
			if a.key != key {
				t.Errorf("%s shortcut %s, want %s", a.name, a.key, key)
			}
			delete(want, a.name)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing resource actions: %v", want)
	}
}

func TestNavigationPaletteSelectionSuspendsDrawer(t *testing.T) {
	for _, resource := range []string{"image", "network"} {
		t.Run(resource, func(t *testing.T) {
			m := fixtureModel(t)
			m.tab = 0
			canceled := false
			m.drawerCancel = func() { canceled = true }
			id := m.items()[0].ID
			if resource == "network" {
				id = m.networks()[0].ID
			}
			key := resource + ":" + id
			m.key(tea.KeyPressMsg{Code: tea.KeyExtended, Text: key})
			if !canceled || m.drawerCancel != nil {
				t.Fatal("palette resource selection bypassed drawer cancellation")
			}
		})
	}
}

func TestNavigationPaletteContainerResult(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 1
	m.openInput("palette")
	id := m.containers()[0].ID
	for n, a := range m.paletteActions() {
		if a.key == "container:"+id {
			m.paletteCursor = n
			m.overlayKey(tea.KeyPressMsg{Code: tea.KeyEnter})
			if m.tab != 0 || !m.containerDetail {
				t.Fatalf("container palette result opened tab %d, detail %v", m.tab, m.containerDetail)
			}
			return
		}
	}
	t.Fatal("missing container palette result")
}

func TestNavigationRemovedShortcutsDoNotNavigate(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	for _, key := range []rune{'5', '6', '7'} {
		m.key(tea.KeyPressMsg{Code: key, Text: string(key)})
		if m.tab != 0 {
			t.Fatalf("removed shortcut %c changed tab to %d", key, m.tab)
		}
	}
}

func TestNavigationRoundTripPreservesContainerViewport(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	m.containerFilter = "storefront"
	m.containerCursor = 2
	id := m.containers()[m.containerCursor].ID
	m.selectedContain[id] = true
	before := m.Render()
	for tab := 1; tab <= 3; tab++ {
		m.switchTab(tab)
		m.moveCursor("down", 1)
		m.setActiveFilter("unrelated-resource-filter")
	}
	m.switchTab(0)
	if m.containerFilter != "storefront" || m.containerCursor != 2 || m.containers()[m.containerCursor].ID != id || !m.selectedContain[id] {
		t.Fatal("resource navigation changed container context")
	}
	if got := m.Render(); got != before {
		t.Fatal("container viewport changed after resource round trip")
	}
}

// A palette navigation without an activated drawer must not sample the whole
// Containers tab. Drawer worker startup/cancellation is covered by integration.
func TestNavigationPaletteReturnDoesNotRunIdleStatsWorker(t *testing.T) {
	for _, resource := range []string{"container", "project"} {
		for _, found := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/found=%v", resource, found), func(t *testing.T) {
				m := fixtureModel(t)
				m.tab = 1
				m.statsLoading = false
				id := "missing-result"
				if found {
					if resource == "container" {
						id = m.containers()[0].ID
					} else {
						id = m.projects()[0].Name
					}
				}
				cmd := m.key(tea.KeyPressMsg{Code: tea.KeyExtended, Text: resource + ":" + id})
				statsResults := 0
				var run func(tea.Cmd)
				run = func(cmd tea.Cmd) {
					if cmd == nil {
						return
					}
					switch msg := cmd().(type) {
					case tea.BatchMsg:
						for _, child := range msg {
							run(child)
						}
					case statsMsg:
						statsResults++
						if len(msg.data) == 0 {
							t.Fatal("returned worker did not sample running containers")
						}
						m.Update(msg)
					}
				}
				// A found container starts a real streaming drawer. Do not execute
				// its long-lived Batch synchronously in this unit test.
				if resource == "container" && found {
					if cmd == nil || !m.focus.DrawerOpen || m.drawerID != id {
						t.Fatal("palette activation did not open selected drawer")
					}
					m.closeDrawer()
					return
				}
				run(cmd)
				if statsResults != 0 || m.statsLoading {
					t.Fatalf("stats results=%d, loading=%v; idle navigation must not poll", statsResults, m.statsLoading)
				}
				if next := m.loadStats(); next != nil {
					t.Fatal("closed drawer scheduled stats")
				}
			})
		}
	}
}

func TestNavigationMouseDepartureCancelsDrawer(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	m.width = 120
	m.containerCursor = 1
	m.containerFilter = "storefront"
	canceled := 0
	m.drawerCancel = func() { canceled++ }
	m.mouse(15, 1) // Images header, after "1 Containers" and its gap.
	if m.tab != 1 || canceled != 1 || m.drawerCancel != nil {
		t.Fatalf("mouse departure: tab=%d cancellations=%d", m.tab, canceled)
	}
	m.mouse(0, 1)
	if m.tab != 0 || canceled != 1 || m.containerCursor != 1 || m.containerFilter != "storefront" {
		t.Fatal("mouse round trip lost container state or repeated cancellation")
	}
}
