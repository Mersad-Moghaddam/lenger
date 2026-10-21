// Package chart renders bounded, font-independent terminal data visualizations.
package chart

import (
	"math"
	"strings"
)

// Bar draws eighth-cell progress, clamping invalid input and width.
func Bar(fraction float64, width int, ascii bool) string {
	if width <= 0 {
		return ""
	}
	if math.IsNaN(fraction) {
		fraction = 0
	}
	fraction = max(0, min(1, fraction))
	if ascii {
		full := int(fraction * float64(width))
		return strings.Repeat("#", full) + strings.Repeat("-", width-full)
	}
	steps := int(fraction * float64(width) * 8)
	full := steps / 8
	partial := steps % 8
	out := strings.Repeat("█", full)
	if full < width && partial > 0 {
		out += string([]rune(" ▏▎▍▌▋▊▉")[partial])
		full++
	}
	return out + strings.Repeat("░", width-full)
}

// Sparkline samples a series into exactly width cells.
func Sparkline(values []float64, width int, ascii bool) string {
	if width <= 0 {
		return ""
	}
	if len(values) == 0 {
		return strings.Repeat(" ", width)
	}
	levels := []rune("▁▂▃▄▅▆▇█")
	if ascii {
		levels = []rune(" .:-=+*#")
	}
	peak := 0.0
	for _, v := range values {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			peak = max(peak, v)
		}
	}
	var out strings.Builder
	for x := 0; x < width; x++ {
		v := values[min(len(values)-1, x*len(values)/width)]
		if math.IsNaN(v) || math.IsInf(v, 0) {
			v = 0
		}
		level := 0
		if peak > 0 {
			level = int(max(0, min(1, v/peak)) * float64(len(levels)-1))
		}
		out.WriteRune(levels[level])
	}
	return out.String()
}

// Braille draws an area series at two-by-four dots per terminal cell.
func Braille(values []float64, width, height int, ascii bool) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if ascii {
		return strings.Repeat(Sparkline(values, width, true)+"\n", height-1) + Sparkline(values, width, true)
	}
	grid := make([][]rune, height)
	for y := range grid {
		grid[y] = make([]rune, width)
		for x := range grid[y] {
			grid[y][x] = '\u2800'
		}
	}
	peak := 0.0
	for _, v := range values {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			peak = max(peak, v)
		}
	}
	bits := [2][4]uint{{0, 1, 2, 6}, {3, 4, 5, 7}}
	if len(values) > 0 && peak > 0 {
		for x := 0; x < width*2; x++ {
			v := values[min(len(values)-1, x*len(values)/(width*2))]
			if math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			h := int(max(0, min(1, v/peak)) * float64(height*4))
			for y := height*4 - h; y < height*4; y++ {
				grid[y/4][x/2] |= rune(1 << bits[x%2][y%4])
			}
		}
	}
	lines := make([]string, height)
	for y := range grid {
		lines[y] = string(grid[y])
	}
	return strings.Join(lines, "\n")
}

// Segments distributes cells proportionally, preserving the exact total width.
func Segments(sizes []int64, width int) []int {
	out := make([]int, len(sizes))
	var total int64
	for _, s := range sizes {
		total += max(0, s)
	}
	if total == 0 || width <= 0 {
		return out
	}
	used := 0
	var cumulative int64
	for n, s := range sizes {
		cumulative += max(0, s)
		end := int(float64(cumulative) / float64(total) * float64(width))
		out[n] = end - used
		used = end
	}
	return out
}
