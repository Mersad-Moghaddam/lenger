// Package masterdetail computes responsive list and inspector pane sizes.
package masterdetail

import "math"

type LayoutMode int

const (
	ModeListOnly LayoutMode = iota
	ModeSplit
	ModeOverlay
)

// Layout holds pane geometry and the preferred list ratio. Ratio remains the
// user preference even when minimum pane widths constrain the actual split.
type Layout struct {
	Mode           LayoutMode
	ListWidth      int
	InspectorWidth int
	Height         int
	Ratio          float64
}

// ComputeLayout reserves one divider cell in split mode. Overlay mode keeps
// the underlying list full-width while the inspector covers it. Nonpositive
// dimensions are normalized to zero; an open drawer still selects overlay
// mode when there is no horizontal space.
func ComputeLayout(totalWidth, totalHeight int, drawerOpen bool, ratio float64) Layout {
	totalWidth = max(0, totalWidth)
	totalHeight = max(0, totalHeight)
	l := Layout{
		Mode:      ModeListOnly,
		ListWidth: totalWidth,
		Height:    totalHeight,
		Ratio:     normalizeRatio(ratio),
	}
	if !drawerOpen {
		return l
	}
	if totalWidth < 87 {
		l.Mode = ModeOverlay
		l.InspectorWidth = totalWidth
		return l
	}
	l.Mode = ModeSplit
	available := totalWidth - 1
	l.ListWidth = max(32, min(int(float64(available)*l.Ratio), available-54))
	l.InspectorWidth = available - l.ListWidth
	return l
}

// AdjustRatio updates the preferred ratio without recomputing geometry. Call
// ComputeLayout with the new ratio to apply it. Non-finite deltas are ignored.
func (l *Layout) AdjustRatio(delta float64) {
	l.Ratio = normalizeRatio(l.Ratio)
	if math.IsNaN(delta) || math.IsInf(delta, 0) {
		return
	}
	l.Ratio = max(.20, min(.50, l.Ratio+delta))
}

// Zero, negative, and non-finite preferences use the 35% default. Positive
// preferences are clamped to the adjustable 20–50% range.
func normalizeRatio(ratio float64) float64 {
	if ratio <= 0 || math.IsNaN(ratio) || math.IsInf(ratio, 0) {
		return .35
	}
	return max(.20, min(.50, ratio))
}
