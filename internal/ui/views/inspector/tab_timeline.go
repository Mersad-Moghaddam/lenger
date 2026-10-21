package inspector

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
)

// TimelineItem is timestamped evidence, not an inferred lifecycle transition.
type TimelineItem struct {
	Timestamp time.Time
	Type      string
	Summary   string
	Outcome   string
}

// BuildIncidentSequence merges inspect evidence and container-scoped events.
// The bool means coverage is incomplete or unknown, NOT proven daemon retention
// loss. Even false only means the known creation boundary was observed; the
// session event stream cannot certify an uninterrupted historical record.
func BuildIncidentSequence(c engine.ContainerInspection, events []engine.Event) ([]TimelineItem, bool) {
	items := make([]TimelineItem, 0)
	add := func(at time.Time, kind, summary, outcome string) {
		if at.IsZero() {
			return
		}
		items = append(items, TimelineItem{at, kind, summary, outcome})
	}
	add(c.Created, "create", "Container created", "Recorded by inspect")
	add(c.StartedAt, "start", "Container started", "Latest start recorded by inspect")
	add(c.FinishedAt, "die", "Container exited", fmt.Sprintf("Exit code %d (inspect)", c.ExitCode))
	// Only a completed latest probe is available; do not imply a full health log
	// or expose arbitrary command output, which may contain credentials.
	if probe := c.HealthCheck; probe != nil && !probe.End.IsZero() {
		if probe.ExitCode != 0 {
			add(probe.End, "health_warn", "Latest health probe failed", fmt.Sprintf("Probe exit code %d", probe.ExitCode))
		} else if probe.Status == "healthy" {
			add(probe.End, "health_ok", "Latest health probe succeeded", "Probe exit code 0")
		} else if probe.Status == "unhealthy" {
			add(probe.End, "health_warn", "Latest health probe completed", "Reported status: unhealthy; probe exit code 0")
		}
	}
	creationObserved := false
	streamError := false
	for _, event := range events {
		if event.Err != nil {
			streamError = true
			continue
		}
		if c.ID == "" || event.ID != c.ID || event.Kind != "container" || event.At.IsZero() {
			continue
		}
		switch event.Action {
		case "create":
			if !c.Created.IsZero() && event.At.Equal(c.Created) {
				creationObserved = true
			}
			add(event.At, "create", "Container created", "Observed daemon event")
		case "start":
			add(event.At, "start", "Container started", "Observed daemon event")
		case "die":
			add(event.At, "die", "Container exited", "Observed daemon event; exit code unavailable")
		case "restart":
			add(event.At, "restart", "Container restarted", "Observed daemon event")
		case "oom":
			add(event.At, "oom", "Out-of-memory event", "Observed daemon OOM event")
		case "health_status: unhealthy":
			add(event.At, "health_warn", "Health became unhealthy", "Observed daemon health transition")
		case "health_status: healthy":
			add(event.At, "health_ok", "Health became healthy", "Observed daemon health transition")
		case "health_status: starting":
			add(event.At, "health_start", "Health checks starting", "Observed daemon health transition")
		}
	}
	// Stable sorting preserves source order for simultaneous distinct events.
	sort.SliceStable(items, func(i, j int) bool { return items[i].Timestamp.Before(items[j].Timestamp) })
	type evidenceKey struct {
		at   time.Time
		kind string
	}
	seen := make(map[evidenceKey]bool)
	deduped := items[:0]
	for _, item := range items {
		key := evidenceKey{item.Timestamp.UTC(), item.Type}
		if !seen[key] {
			seen[key] = true
			deduped = append(deduped, item)
		}
	}
	return deduped, !creationObserved || streamError
}

// RenderTimelineTab renders bounded chronological evidence. The caller owns
// viewport scrolling. Unknown coverage must not be described as daemon deletion.
func RenderTimelineTab(s style.Sheet, items []TimelineItem, hasTruncatedHistory bool, w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	lines := []string{s.Heading("Timeline & Events")}
	if hasTruncatedHistory {
		lines = append(lines, s.Muted("History coverage is incomplete or unknown; earlier events may be unavailable."))
	}
	now := time.Now()
	count := 0
	for _, item := range items {
		if item.Timestamp.IsZero() {
			continue
		}
		count++
		glyph := timelineGlyph(item.Type, s.ASCII)
		lines = append(lines, fmt.Sprintf("%s %s (%s) %s — %s: %s", glyph,
			item.Timestamp.UTC().Format(time.RFC3339Nano), timelineRelativeTime(item.Timestamp, now),
			style.Clean(item.Type), style.Clean(item.Summary), style.Clean(item.Outcome)))
	}
	if count == 0 {
		lines = append(lines, s.Muted("No timestamped lifecycle evidence available."))
	}
	return style.Fit(strings.Join(lines, "\n"), w, h)
}

func timelineGlyph(kind string, ascii bool) string {
	if ascii {
		switch kind {
		case "oom", "health_warn", "die":
			return "!"
		default:
			return ">"
		}
	}
	switch kind {
	case "create":
		return "○"
	case "start":
		return "▶"
	case "health_warn":
		return "⚠"
	case "oom", "die":
		return "×"
	case "restart":
		return "↻"
	default:
		return "•"
	}
}

func timelineRelativeTime(at, now time.Time) string {
	delta := now.Sub(at)
	future := delta < 0
	if future {
		delta = -delta
	}
	var label string
	switch {
	case delta < time.Minute:
		label = fmt.Sprintf("%ds", int(delta.Seconds()))
	case delta < time.Hour:
		label = fmt.Sprintf("%dm", int(delta.Minutes()))
	case delta < 24*time.Hour:
		label = fmt.Sprintf("%dh", int(delta.Hours()))
	default:
		label = fmt.Sprintf("%dd", int(delta.Hours()/24))
	}
	if future {
		return "in " + label
	}
	return label + " ago"
}
