package app

import "testing"

func TestFocusInitialAndOpenDrawer(t *testing.T) {
	f := FocusManager{}
	if f.Area != FocusList || f.DrawerOpen || f.Zoomed {
		t.Fatalf("unexpected initial state: %+v", f)
	}
	for _, tc := range []struct{ initial, want int }{{-1, 0}, {0, 0}, {2, 2}, {4, 4}, {9, 4}} {
		f = FocusManager{Area: FocusDrawerControls, ControlIdx: 2, ControlMax: 3, Zoomed: true}
		f.OpenDrawer(tc.initial)
		if !f.DrawerOpen || f.Area != FocusList || f.ActiveTab != tc.want || f.Zoomed || f.ControlIdx != 0 || f.ControlMax != 0 {
			t.Fatalf("OpenDrawer(%d): %+v", tc.initial, f)
		}
	}
}

func TestFocusForwardAndReverseLoop(t *testing.T) {
	f := FocusManager{}
	f.OpenDrawer(0)
	f.ControlMax = 3
	want := []struct {
		area FocusArea
		idx  int
	}{{FocusDrawerTabs, 0}, {FocusDrawerControls, 0}, {FocusDrawerControls, 1}, {FocusDrawerControls, 2}, {FocusList, 0}}
	for i, state := range want {
		if !f.HandleKey("tab") || f.Area != state.area || f.ControlIdx != state.idx {
			t.Fatalf("forward step %d: %+v", i, f)
		}
	}
	want = []struct {
		area FocusArea
		idx  int
	}{{FocusDrawerControls, 2}, {FocusDrawerControls, 1}, {FocusDrawerControls, 0}, {FocusDrawerTabs, 0}, {FocusList, 0}}
	for i, state := range want {
		if !f.HandleKey("shift+tab") || f.Area != state.area || f.ControlIdx != state.idx {
			t.Fatalf("reverse step %d: %+v", i, f)
		}
	}
}

func TestFocusEmptyControlsAreSkipped(t *testing.T) {
	for _, count := range []int{0, -1} {
		f := FocusManager{DrawerOpen: true, ControlMax: count}
		for _, area := range []FocusArea{FocusDrawerTabs, FocusList} {
			if !f.HandleKey("tab") || f.Area != area {
				t.Fatalf("count %d forward: %+v", count, f)
			}
		}
		for _, area := range []FocusArea{FocusDrawerTabs, FocusList} {
			if !f.HandleKey("shift+tab") || f.Area != area {
				t.Fatalf("count %d reverse: %+v", count, f)
			}
		}
	}
}

func TestFocusTabSelectionIsClampedAndResetsControls(t *testing.T) {
	for _, tc := range []struct {
		key         string
		start, want int
	}{{"left", 0, 0}, {"h", 2, 1}, {"right", 4, 4}, {"l", 2, 3}, {"left", -9, 0}, {"right", 9, 4}} {
		f := FocusManager{DrawerOpen: true, Area: FocusDrawerTabs, ActiveTab: tc.start, ControlIdx: 2, ControlMax: 3}
		if !f.HandleKey(tc.key) || f.ActiveTab != tc.want {
			t.Fatalf("%+v: %+v", tc, f)
		}
		if tc.want != tc.start && (f.ControlIdx != 0 || f.ControlMax != 0) {
			t.Fatalf("tab change retained stale controls: %+v", f)
		}
	}
}

func TestFocusZoomAndEscapeLifecycle(t *testing.T) {
	for _, area := range []FocusArea{FocusList, FocusDrawerTabs, FocusDrawerControls} {
		f := FocusManager{DrawerOpen: true, Area: area, ControlIdx: 2, ControlMax: 3}
		if !f.HandleKey("z") || !f.Zoomed {
			t.Fatalf("zoom from %d: %+v", area, f)
		}
		if !f.HandleKey("esc") || f.Zoomed || !f.DrawerOpen || f.Area != area {
			t.Fatalf("first escape: %+v", f)
		}
		if !f.HandleKey("esc") || f.DrawerOpen || f.Area != FocusList || f.ControlIdx != 0 || f.ControlMax != 0 {
			t.Fatalf("second escape: %+v", f)
		}
	}
	f := FocusManager{DrawerOpen: true}
	f.ToggleZoom()
	f.ToggleZoom()
	if f.Zoomed {
		t.Fatal("second toggle must unzoom")
	}
	f.Zoomed, f.ControlIdx, f.ControlMax = true, 2, 3
	f.CloseDrawer()
	if f.DrawerOpen || f.Zoomed || f.Area != FocusList || f.ControlIdx != 0 || f.ControlMax != 0 {
		t.Fatalf("close must clear transient state: %+v", f)
	}
	f.ToggleZoom()
	if f.Zoomed {
		t.Fatal("closed drawer must not zoom")
	}
}

func TestFocusUnhandledKeysDoNotMutateState(t *testing.T) {
	for _, state := range []FocusManager{{}, {DrawerOpen: true, Area: FocusList}, {DrawerOpen: true, Area: FocusDrawerControls, ControlMax: 3}} {
		keys := []string{"enter", "up", "down", "j", "k", "q", "r", "s", "?", "/", "[", "]", "left", "right", "h", "l"}
		if !state.DrawerOpen {
			keys = append(keys, "tab", "shift+tab", "esc", "z")
		}
		for _, key := range keys {
			f := state
			if f.HandleKey(key) || f != state {
				t.Fatalf("key %q must pass through unchanged: before %+v after %+v", key, state, f)
			}
		}
	}
}
