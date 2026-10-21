package chart

import (
	"charm.land/lipgloss/v2"
	"math"
	"strings"
	"testing"
)

func TestSoundingIsBoundedAndReachesFull(t *testing.T) {
	for _, ascii := range []bool{false, true} {
		for _, width := range []int{0, 1, 12, 80} {
			lines := strings.Split(Sounding(.5, width, ascii), "\n")
			if width == 0 {
				if len(lines) != 1 || lines[0] != "" {
					t.Fatal("zero-width sounding should be empty")
				}
				continue
			}
			if len(lines) != 3 {
				t.Fatalf("sounding has %d lines, want 3", len(lines))
			}
			for _, line := range lines {
				if got := lipgloss.Width(line); got != width {
					t.Fatalf("sounding width %d != %d", got, width)
				}
			}
		}
		full := strings.Split(Sounding(1, 24, ascii), "\n")[2]
		if strings.ContainsAny(full, "░╌-") {
			t.Fatalf("full gauge retains an empty cell: %q", full)
		}
	}
}

func TestChartWidths(t *testing.T) {
	for _, ascii := range []bool{false, true} {
		for _, width := range []int{0, 1, 8, 80} {
			for _, fraction := range []float64{-1, 0, .5, 1, 2, math.NaN()} {
				if got := lipgloss.Width(Bar(fraction, width, ascii)); got != width {
					t.Fatalf("bar width %d != %d", got, width)
				}
			}
			for _, line := range strings.Split(Braille([]float64{0, 1, 2}, width, 2, ascii), "\n") {
				if got := lipgloss.Width(line); got != width {
					t.Fatalf("chart width %d != %d", got, width)
				}
			}
		}
	}
}
