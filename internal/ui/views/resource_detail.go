package views

import (
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"time"
)

func detailRows(lines []string, width int) []string {
	var rows []string
	for _, line := range lines {
		rows = append(rows, strings.Split(ansi.Hardwrap(line, max(1, width), true), "\n")...)
	}
	return rows
}

func VolumeDetailRows(s style.Sheet, v engine.Volume, width int, now time.Time) []string {
	return detailRows(volumeDetailLines(s, v, now), width)
}
