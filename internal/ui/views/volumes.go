package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/components"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
)

// Volumes renders volume size, driver, and the containers holding each mount.
func Volumes(s style.Sheet, items []engine.Volume, cursor int, filter string, width, height int, now time.Time) string {
	return VolumeList(s, items, cursor, filter, width, height, now)
}

// VolumeList keeps the table compact enough for an 80-column terminal.
func VolumeList(s style.Sheet, items []engine.Volume, cursor int, filter string, width, height int, now time.Time) string {
	cell := max(4, (width-34)/3)
	lines := []string{s.Muted("filter " + emptyDash(filter)), fmt.Sprintf("  %s %s %s %s %9s", padCell("DRIVER", 10), padCell("VOLUME", cell), padCell("MOUNT POINT", cell), padCell("CONTAINERS", cell), "SIZE")}
	capacity := max(1, height-tokens.PanelFrame-tokens.InsetY*2-2)
	start := max(0, cursor-capacity+1)
	end := min(len(items), start+capacity)
	g := tokens.Icons(s.ASCII)
	for n := start; n < end; n++ {
		v := items[n]
		marker := " "
		if n == cursor {
			marker = g.Cursor
		}
		size := components.Bytes(max(0, v.Size))
		if v.Size < 0 {
			size = "n/a"
		}
		refs := fmt.Sprintf("%d refs", v.RefCount)
		if v.RefCount < 0 {
			refs = "? refs"
		}
		var names []string
		for _, c := range v.UsedBy {
			names = append(names, c.Name)
		}
		associated := strings.Join(names, ",")
		if associated == "" {
			associated = refs
		}
		line := fmt.Sprintf("%s %s %s %s %s %9s", marker, padCell(style.Clean(v.Driver), 10), padCell(style.Clean(v.Name), cell), padCell(style.Clean(v.Mountpoint), cell), padCell(style.Clean(associated), cell), size)
		if n == cursor {
			line = s.Text(line, s.Palette.Accent)
		}
		lines = append(lines, line)
	}
	if len(items) == 0 {
		return s.Panel("VOLUMES  0", components.Empty(s, "No volumes in the hold.", "Named volumes appear after a container mounts one.", max(1, width-tokens.PanelFrame-tokens.InsetX*2)), width, height, true)
	}
	return s.Panel(fmt.Sprintf("VOLUMES  %d", len(items)), strings.Join(lines, "\n"), width, height, true)
}

// VolumeDetail names every container that mounts the volume before offering removal.
func VolumeDetail(s style.Sheet, volume engine.Volume, width, height int, now time.Time) string {
	return s.Panel(volume.Name, strings.Join(volumeDetailLines(s, volume, now), "\n"), width, height, false)
}

func volumeDetailLines(s style.Sheet, volume engine.Volume, now time.Time) []string {
	lines := []string{style.Clean(volume.Driver) + " · " + style.Clean(volume.Scope), style.Clean(volume.Mountpoint), ""}
	if !volume.CreatedAt.IsZero() {
		lines = append(lines, "Created "+components.Age(volume.CreatedAt, now))
	}
	size := "size unavailable"
	if volume.Size >= 0 {
		size = components.Bytes(volume.Size) + " on disk"
	}
	refs := "reference count unavailable"
	if volume.RefCount >= 0 {
		refs = fmt.Sprintf("%d container reference(s)", volume.RefCount)
	}
	lines = append(lines, size, refs, "", s.Heading("USED BY"))
	if len(volume.UsedBy) == 0 {
		lines = append(lines, "No containers. Safe to remove.")
	}
	for _, c := range volume.UsedBy {
		lines = append(lines, style.Clean(c.Name)+"  "+components.Pill(s, c.State))
	}
	keys := make([]string, 0, len(volume.Labels))
	for key := range volume.Labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) > 0 {
		lines = append(lines, "", s.Heading("LABELS"))
		for _, key := range keys {
			lines = append(lines, style.Clean(key)+"="+style.Clean(volume.Labels[key]))
		}
	}
	lines = append(lines, "", "d remove unused   esc back")
	return lines
}
