package chart

import (
	"charm.land/lipgloss/v2"
	"math"
	"strings"
	"testing"
)

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
