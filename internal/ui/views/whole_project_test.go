package views

import (
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/theme"
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
	"time"
)

func TestWholeListAvailabilityAndStatus(t *testing.T) {
	p, err := theme.Bundled("midnight")
	if err != nil {
		t.Fatal(err)
	}
	s := style.Sheet{Palette: p, Plain: true, ASCII: true}
	now := time.Now()
	for _, tc := range []struct {
		name   string
		sample engine.ContainerStats
		state  string
		want   string
		reject string
	}{
		{"known zero", engine.ContainerStats{At: now, MemoryAvailable: true}, "running", "mem 0 B", ""},
		{"absent", engine.ContainerStats{At: now}, "running", "mem —", "mem 0 B"},
		{"failed poll", engine.ContainerStats{At: now, Memory: 123456789, State: engine.TelemetryUnavailable}, "running", "mem —", "117.7"},
		{"stale", engine.ContainerStats{At: now.Add(-time.Hour), Memory: 123456789, MemoryAvailable: true}, "running", "mem —", "117.7"},
		{"stopped", engine.ContainerStats{At: now, Memory: 123456789, MemoryAvailable: true}, "exited", "mem —", "117.7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := ansi.Strip(ContainerList(s, []engine.Container{{ID: "a", Name: "a", State: tc.state}}, 0, nil, "", map[string]engine.ContainerStats{"a": tc.sample}, nil, nil, 160, 20, now))
			if !strings.Contains(body, tc.want) || (tc.reject != "" && strings.Contains(body, tc.reject)) {
				t.Fatal(body)
			}
		})
	}
	for _, tc := range []struct{ state, health, status, want string }{{"restarting", "", "", "restarting"}, {"running", "starting", "", "starting"}, {"exited", "", "Exited (137) 2 minutes ago", "Exited (137)"}, {"exited", "", "Exited (0) 2 minutes ago", "Exited (0)"}, {"exited", "", "", "exited"}} {
		body := ansi.Strip(ContainerList(s, []engine.Container{{ID: "a", Name: "a", State: tc.state, Health: tc.health, Status: tc.status}}, 0, nil, "", nil, nil, nil, 160, 20, now))
		if !strings.Contains(body, tc.want) {
			t.Fatalf("status %s: %s", tc.want, body)
		}
	}
	for _, tc := range []struct {
		c    engine.Container
		role string
	}{{engine.Container{State: "exited", Status: "Exited (137) yesterday"}, "failed"}, {engine.Container{State: "exited", Status: "Exited (0) yesterday"}, "exited"}, {engine.Container{State: "exited"}, "exited"}, {engine.Container{State: "restarting"}, "restarting"}, {engine.Container{State: "running", Health: "starting"}, "starting"}} {
		role, _ := listContainerStatus(tc.c)
		if role != tc.role {
			t.Fatalf("semantic role %s want %s", role, tc.role)
		}
	}
}

func TestWholeLeanResourceTables(t *testing.T) {
	now := time.Now()
	volume := engine.Volume{Name: "volume", Driver: "local", Mountpoint: "/mount", Size: 1024, UsedBy: []engine.Container{{Name: "attached"}}}
	network := engine.Network{Name: "network", Driver: "bridge", Scope: "local", Subnets: []string{"172.1.0.0/16"}, Gateways: []string{"172.1.0.1"}}
	for _, width := range []int{80, 160} {
		v := Volumes(plainSheet(), []engine.Volume{volume}, 0, "", width, 24, now)
		n := Networks(plainSheet(), []engine.Network{network}, 0, "", width, 24, now)
		for _, want := range []string{"DRIVER", "VOLUME", "MOUNT POINT", "CONTAINERS", "SIZE", "local", "/mount", "attached"} {
			if !strings.Contains(v, want) {
				t.Fatalf("volume table %d missing %s", width, want)
			}
		}
		for _, want := range []string{"NAME", "DRIVER", "SCOPE", "SUBNET", "GATEWAY", "CONNECTED", "bridge", "local"} {
			if !strings.Contains(n, want) {
				t.Fatalf("network table %d missing %s", width, want)
			}
		}
		if strings.Contains(v, "USED BY") || strings.Contains(n, "NETWORK DETAIL") {
			t.Fatal("automatic detail pane retained")
		}
	}
}
