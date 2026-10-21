package inspector

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
)

func TestTimelineChronologyAndDedup(t *testing.T) {
	at := time.Date(2026, 10, 21, 7, 0, 0, 0, time.UTC)
	c := engine.ContainerInspection{ID: "target", Created: at, StartedAt: at.Add(time.Minute), HealthCheck: &engine.HealthProbe{End: at.Add(2 * time.Minute), ExitCode: 1, Output: "SECRET=raw-probe-output"}}
	events := []engine.Event{
		{ID: c.ID, Kind: "container", Action: "restart", At: at.Add(4 * time.Minute)},
		{ID: c.ID, Kind: "container", Action: "oom", At: at.Add(3 * time.Minute)},
		{ID: c.ID, Kind: "container", Action: "start", At: c.StartedAt},
		{ID: c.ID, Kind: "container", Action: "create", At: at},
		{ID: c.ID, Kind: "container", Action: "oom", At: at.Add(3 * time.Minute)},
	}
	original := append([]engine.Event(nil), events...)
	items, incomplete := BuildIncidentSequence(c, events)
	if incomplete {
		t.Fatal("matching creation evidence should cover the known creation boundary")
	}
	var types []string
	for _, item := range items {
		types = append(types, item.Type)
		if strings.Contains(item.Summary+item.Outcome, "raw-probe-output") {
			t.Fatal("probe diagnostics leaked")
		}
	}
	if want := []string{"create", "start", "health_warn", "oom", "restart"}; !reflect.DeepEqual(types, want) {
		t.Fatalf("types = %v, want %v", types, want)
	}
	if !reflect.DeepEqual(events, original) {
		t.Fatal("mutated caller events")
	}
}

func TestTimelineFiltersAndIncompleteCoverage(t *testing.T) {
	at := time.Date(2026, 10, 21, 7, 0, 0, 0, time.UTC)
	c := engine.ContainerInspection{ID: "target", Created: at}
	for _, events := range [][]engine.Event{
		nil,
		{{ID: "unrelated", Kind: "container", Action: "create", At: at}},
		{{ID: c.ID, Kind: "image", Action: "create", At: at}},
		{{ID: c.ID, Kind: "container", Action: "create"}},
		{{ID: c.ID, Kind: "container", Action: "start", At: at.Add(time.Minute)}},
		{{ID: c.ID, Kind: "container", Action: "create", At: at, Err: errors.New("secret stream failure")}},
	} {
		items, incomplete := BuildIncidentSequence(c, events)
		if !incomplete {
			t.Fatalf("coverage should be incomplete: %+v", events)
		}
		for _, item := range items {
			if item.Timestamp.IsZero() {
				t.Fatal("zero timestamp emitted")
			}
			if item.Type != "create" && item.Type != "start" {
				t.Fatalf("unrelated event: %+v", item)
			}
		}
		if len(events) > 0 && events[0].Action == "create" && len(items) != 1 {
			t.Fatalf("unrelated/invalid event included: %+v", items)
		}
	}
}

func TestTimelineEmptyAndUnknownHealth(t *testing.T) {
	items, incomplete := BuildIncidentSequence(engine.ContainerInspection{OOMKilled: true, HealthCheck: &engine.HealthProbe{ExitCode: 1}}, nil)
	if len(items) != 0 || !incomplete {
		t.Fatalf("empty evidence = %+v, %v", items, incomplete)
	}
	items, _ = BuildIncidentSequence(engine.ContainerInspection{}, []engine.Event{{Kind: "container", Action: "oom", At: time.Now()}})
	if len(items) != 0 {
		t.Fatal("empty IDs must not match")
	}
}

func TestTimelineEqualTimestampAndInspectExit(t *testing.T) {
	at := time.Date(2026, 10, 21, 7, 0, 0, 0, time.UTC)
	c := engine.ContainerInspection{ID: "target", Created: at, StartedAt: at, FinishedAt: at, ExitCode: 137, OOMKilled: true}
	events := []engine.Event{{ID: c.ID, Kind: "container", Action: "restart", At: at}, {ID: c.ID, Kind: "container", Action: "oom", At: at}}
	items, _ := BuildIncidentSequence(c, events)
	for i := 0; i < 10; i++ {
		got, _ := BuildIncidentSequence(c, events)
		if !reflect.DeepEqual(got, items) {
			t.Fatal("unstable equal timestamps")
		}
	}
	var types []string
	for _, item := range items {
		types = append(types, item.Type)
	}
	if !reflect.DeepEqual(types, []string{"create", "start", "die", "restart", "oom"}) {
		t.Fatalf("stable insertion order lost: %v", types)
	}
	if !strings.Contains(items[2].Outcome, "137") {
		t.Fatal("inspect exit evidence missing")
	}
	withoutEvents, _ := BuildIncidentSequence(c, nil)
	for _, item := range withoutEvents {
		if item.Type == "oom" {
			t.Fatal("invented OOM timestamp from inspect")
		}
	}
}

func TestTimelineHealthEvidence(t *testing.T) {
	at := time.Date(2026, 10, 21, 7, 0, 0, 0, time.UTC)
	c := engine.ContainerInspection{ID: "target", HealthCheck: &engine.HealthProbe{Start: at, End: at.Add(time.Second), Status: "healthy", Output: "TOKEN=secret"}}
	items, _ := BuildIncidentSequence(c, []engine.Event{{ID: c.ID, Kind: "container", Action: "health_status: unhealthy", At: at.Add(2 * time.Second)}})
	if len(items) != 2 || items[0].Type != "health_ok" || items[1].Type != "health_warn" {
		t.Fatalf("health evidence = %+v", items)
	}
	if !items[0].Timestamp.Equal(c.HealthCheck.End) {
		t.Fatal("probe should use completed execution timestamp")
	}
	c.HealthCheck.End = time.Time{}
	items, _ = BuildIncidentSequence(c, nil)
	if len(items) != 0 {
		t.Fatal("unfinished probe must not be presented as completed")
	}
}

func TestTimelineRenderIntegrityAndBounds(t *testing.T) {
	s := style.Sheet{Plain: true, ASCII: true}
	at := time.Date(2026, 10, 21, 7, 0, 0, 0, time.UTC)
	items := []TimelineItem{{Timestamp: at, Type: "oom", Summary: "OOM", Outcome: "observed\x1b\nunsafe"}}
	got := RenderTimelineTab(s, items, true, 160, 8)
	for _, want := range []string{"incomplete", "unknown", "2026-10-21", "oom", "observed", "ago"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q: %q", want, got)
		}
	}
	if strings.Contains(got, "no longer retained") || strings.Contains(got, "\x1b") {
		t.Fatalf("unsafe notice/content: %q", got)
	}
	if strings.Contains(RenderTimelineTab(s, items, false, 160, 8), "incomplete") {
		t.Fatal("spurious notice")
	}
	if !strings.Contains(RenderTimelineTab(s, nil, true, 160, 8), "No timestamped") {
		t.Fatal("missing empty state")
	}
	for _, size := range [][2]int{{0, 4}, {4, 0}, {-1, 4}, {1, 1}, {20, 3}} {
		got := RenderTimelineTab(s, items, true, size[0], size[1])
		if size[0] <= 0 || size[1] <= 0 {
			if got != "" {
				t.Fatal("nonempty invalid bounds")
			}
			continue
		}
		lines := strings.Split(got, "\n")
		if len(lines) > size[1] {
			t.Fatal("height overflow")
		}
		for _, line := range lines {
			if lipgloss.Width(line) > size[0] {
				t.Fatal("width overflow")
			}
		}
	}
}
