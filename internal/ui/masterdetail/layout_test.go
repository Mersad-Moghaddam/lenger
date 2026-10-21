package masterdetail

import (
	"math"
	"testing"
)

// These fixtures catch incorrect mode boundaries, divider accounting, minimum
// widths, ratio direction/defaulting, and dimension normalization.
func TestComputeLayout(t *testing.T) {
	tests := []struct {
		name          string
		width, height int
		open          bool
		ratio         float64
		want          Layout
	}{
		{"closed wide", 200, 40, false, .35, Layout{ModeListOnly, 200, 0, 40, .35}},
		{"closed narrow", 80, 24, false, .4, Layout{ModeListOnly, 80, 0, 24, .4}},
		{"below breakpoint", 86, 24, true, .35, Layout{ModeOverlay, 86, 86, 24, .35}},
		{"at breakpoint", 87, 24, true, .35, Layout{ModeSplit, 32, 54, 24, .35}},
		{"default ratio", 201, 40, true, 0, Layout{ModeSplit, 70, 130, 40, .35}},
		{"divider and truncation", 200, 40, true, .35, Layout{ModeSplit, 69, 130, 40, .35}},
		{"minimum list", 120, 24, true, .2, Layout{ModeSplit, 32, 87, 24, .2}},
		{"minimum inspector", 100, 24, true, .5, Layout{ModeSplit, 45, 54, 24, .5}},
		{"low ratio clamped", 201, 24, true, .1, Layout{ModeSplit, 40, 160, 24, .2}},
		{"high ratio clamped", 201, 24, true, .9, Layout{ModeSplit, 100, 100, 24, .5}},
		{"negative ratio defaults", 201, 24, true, -1, Layout{ModeSplit, 70, 130, 24, .35}},
		{"nan ratio defaults", 201, 24, true, math.NaN(), Layout{ModeSplit, 70, 130, 24, .35}},
		{"infinite ratio defaults", 201, 24, true, math.Inf(1), Layout{ModeSplit, 70, 130, 24, .35}},
		{"negative infinite ratio defaults", 201, 24, true, math.Inf(-1), Layout{ModeSplit, 70, 130, 24, .35}},
		{"zero open", 0, 0, true, 0, Layout{ModeOverlay, 0, 0, 0, .35}},
		{"negative open", -20, -5, true, 0, Layout{ModeOverlay, 0, 0, 0, .35}},
		{"negative closed", -20, -5, false, 0, Layout{ModeListOnly, 0, 0, 0, .35}},
		{"zero height split", 87, 0, true, .35, Layout{ModeSplit, 32, 54, 0, .35}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeLayout(tt.width, tt.height, tt.open, tt.ratio)
			if got != tt.want {
				t.Fatalf("ComputeLayout() = %+v, want %+v", got, tt.want)
			}
			if got.Mode == ModeSplit && got.ListWidth+1+got.InspectorWidth != tt.width {
				t.Fatal("split must reserve exactly one divider cell")
			}
		})
	}
}

// Reusing the configured ratio across resize must not latch overlay mode or
// replace the preference with a minimum-constrained effective ratio.
func TestComputeLayoutResize(t *testing.T) {
	l := ComputeLayout(201, 40, true, .4)
	for _, width := range []int{86, 87, 80, 201} {
		l = ComputeLayout(width, 30, true, l.Ratio)
		if l.Ratio != .4 || l.Height != 30 {
			t.Fatalf("resize lost preference or height: %+v", l)
		}
		if width < 87 {
			if l.Mode != ModeOverlay || l.InspectorWidth != width {
				t.Fatalf("width %d: expected full-width overlay, got %+v", width, l)
			}
		} else if l.Mode != ModeSplit || l.ListWidth < 32 || l.InspectorWidth < 54 || l.ListWidth+1+l.InspectorWidth != width {
			t.Fatalf("width %d: invalid split %+v", width, l)
		}
	}
	if l.ListWidth != 80 || l.InspectorWidth != 120 {
		t.Fatalf("resize did not restore preferred split: %+v", l)
	}
}

// Adjustments change the preference only; the next ComputeLayout handles sizing.
func TestAdjustRatio(t *testing.T) {
	tests := []struct {
		name               string
		ratio, delta, want float64
	}{
		{"expand", .35, .05, .4},
		{"shrink", .35, -.05, .3},
		{"upper clamp", .4, 1, .5},
		{"lower clamp", .3, -1, .2},
		{"zero value", 0, .05, .4},
		{"invalid preference", math.NaN(), .05, .4},
		{"nan delta ignored", .4, math.NaN(), .4},
		{"infinite delta ignored", .4, math.Inf(1), .4},
		{"negative infinite delta ignored", .4, math.Inf(-1), .4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := Layout{Mode: ModeSplit, ListWidth: 70, InspectorWidth: 130, Height: 40, Ratio: tt.ratio}
			l.AdjustRatio(tt.delta)
			if math.IsNaN(l.Ratio) || math.Abs(l.Ratio-tt.want) > 1e-12 {
				t.Fatalf("AdjustRatio() ratio = %v, want %v", l.Ratio, tt.want)
			}
			if l.Mode != ModeSplit || l.ListWidth != 70 || l.InspectorWidth != 130 || l.Height != 40 {
				t.Fatalf("adjustment changed geometry: %+v", l)
			}
		})
	}
}
