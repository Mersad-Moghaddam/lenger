package app

// FocusArea identifies a stop in the list and inspector focus loop.
type FocusArea int

const (
	FocusList FocusArea = iota
	FocusDrawerTabs
	FocusDrawerControls
)

// FocusManager manages keyboard focus independently of container and view state.
// ControlMax is the number of focusable controls, not the last control index.
// The caller supplies it after opening a drawer or changing its active tab.
type FocusManager struct {
	Area       FocusArea
	ActiveTab  int
	ControlIdx int
	ControlMax int
	DrawerOpen bool
	Zoomed     bool
}

// OpenDrawer selects a caller-provided smart initial tab and keeps list focus.
func (f *FocusManager) OpenDrawer(initialTab int) {
	f.ActiveTab = min(4, max(0, initialTab))
	f.DrawerOpen = true
	f.Area = FocusList
	f.Zoomed = false
	f.ControlIdx, f.ControlMax = 0, 0
}

// CloseDrawer returns focus to the list and discards transient drawer state.
func (f *FocusManager) CloseDrawer() {
	f.DrawerOpen, f.Zoomed = false, false
	f.Area = FocusList
	f.ControlIdx, f.ControlMax = 0, 0
}

// ToggleZoom has no effect when the drawer is closed.
func (f *FocusManager) ToggleZoom() {
	if f.DrawerOpen {
		f.Zoomed = !f.Zoomed
	}
}

// HandleKey consumes only focus, drawer dismissal, and zoom keys. Enter is
// intentionally left to the caller: list activation needs container context,
// while control activation belongs to the active tab's view.
func (f *FocusManager) HandleKey(key string) bool {
	if !f.DrawerOpen {
		return false
	}
	switch key {
	case "esc":
		if f.Zoomed {
			f.Zoomed = false
		} else {
			f.CloseDrawer()
		}
	case "z":
		f.ToggleZoom()
	case "tab":
		switch f.Area {
		case FocusList:
			f.Area, f.ControlIdx = FocusDrawerTabs, 0
		case FocusDrawerTabs:
			f.ControlIdx = 0
			if f.ControlMax > 0 {
				f.Area = FocusDrawerControls
			} else {
				f.Area = FocusList
			}
		case FocusDrawerControls:
			if f.ControlIdx < f.ControlMax-1 {
				f.ControlIdx++
			} else {
				f.Area, f.ControlIdx = FocusList, 0
			}
		}
	case "shift+tab":
		switch f.Area {
		case FocusList:
			if f.ControlMax > 0 {
				f.Area, f.ControlIdx = FocusDrawerControls, f.ControlMax-1
			} else {
				f.Area, f.ControlIdx = FocusDrawerTabs, 0
			}
		case FocusDrawerTabs:
			f.Area, f.ControlIdx = FocusList, 0
		case FocusDrawerControls:
			if f.ControlIdx > 0 {
				f.ControlIdx--
			} else {
				f.Area, f.ControlIdx = FocusDrawerTabs, 0
			}
		}
	case "left", "h", "right", "l":
		if f.Area != FocusDrawerTabs {
			return false
		}
		next := f.ActiveTab
		if key == "left" || key == "h" {
			next--
		} else {
			next++
		}
		next = min(4, max(0, next))
		if next != f.ActiveTab {
			f.ActiveTab = next
			f.ControlIdx, f.ControlMax = 0, 0
		}
	default:
		return false
	}
	return true
}
