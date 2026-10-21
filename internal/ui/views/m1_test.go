package views

import (
	"strings"
	"testing"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
)

func plainSheet() style.Sheet {
	return style.Sheet{Plain: true, ASCII: true}
}

func TestContainerListKeepsStateAndUptimeVisible(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	items := []engine.Container{{
		ID: "0123456789abcdef", Name: "web-1", Image: "nginx:1.27-alpine",
		State: "running", Status: "Up 2 hours", Created: now.Add(-2 * time.Hour),
		Ports: []string{"8080:80"}, Project: "storefront",
	}}
	text := Containers(plainSheet(), items, 0, nil, "", nil, false, false, nil, nil, nil, 80, 24, now)
	for _, want := range []string{"web-1", "running", "up 2h", "storefront"} {
		if !strings.Contains(text, want) {
			t.Fatalf("container list missing %q:\n%s", want, text)
		}
	}
}

func TestContainerDetailMasksAndOffersSanitizedJSON(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	item := engine.Container{ID: "0123456789abcdef", Name: "web-1", Image: "nginx:1.27-alpine", State: "running"}
	inspect := &engine.ContainerInspection{
		ID: item.ID, Name: item.Name, Image: item.Image, State: "running",
		Status: "Up 2 hours", StartedAt: now.Add(-2 * time.Hour),
		Ports: []string{"8080:80"}, Networks: []string{"bridge"},
		Env: []string{"TOKEN=••••"}, Labels: map[string]string{"owner": "••••"},
	}
	stats := map[string]engine.ContainerStats{item.ID: {
		Memory: 96_000_000, MemoryLimit: 1_000_000_000, NetworkRX: 48_000_000, NetworkTX: 16_000_000,
		BlockRead: 6_000_000, BlockWrite: 3_000_000, PIDs: 8,
	}}
	cpu := map[string]float64{item.ID: 18.4}
	history := map[string][]float64{item.ID: {12, 14, 18, 16, 20, 18.4}}
	text := Containers(plainSheet(), []engine.Container{item}, 0, nil, "", inspect, true, false, stats, cpu, history, 80, 32, now)
	if !strings.Contains(text, "ENVIRONMENT (MASKED)") || !strings.Contains(text, "TOKEN=••••") {
		t.Fatalf("structured inspection did not expose masked metadata:\n%s", text)
	}
	if !strings.Contains(text, "RESOURCES") || !strings.Contains(text, "CPU:  18.4%") {
		t.Fatalf("structured inspection did not expose measured resources:\n%s", text)
	}
	raw := Containers(plainSheet(), []engine.Container{item}, 0, nil, "", inspect, true, true, stats, cpu, history, 80, 50, now)
	if !strings.Contains(raw, "SANITIZED JSON") || !strings.Contains(raw, "TOKEN=••••") {
		t.Fatalf("sanitized JSON view is incomplete:\n%s", raw)
	}
}

func TestVolumeAndSystemViewsRenderMeasuredFields(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	volume := engine.Volume{
		Name: "scratch-data", Driver: "local", Scope: "local", Mountpoint: "/var/lib/docker/volumes/scratch-data/_data",
		CreatedAt: now.Add(-3 * time.Hour), Size: 42_000_000, RefCount: 0,
	}
	volumes := Volumes(plainSheet(), []engine.Volume{volume}, 0, "", 120, 30, now)
	if !strings.Contains(volumes, "scratch-data") || !strings.Contains(volumes, "42.0 MB") || !strings.Contains(volumes, "Safe to remove") {
		t.Fatalf("volume detail is incomplete:\n%s", volumes)
	}
	usage := engine.SystemUsage{
		Images:      engine.UsageCategory{Total: 4, Active: 3, Size: 500_000_000, Reclaimable: 100_000_000},
		Containers:  engine.UsageCategory{Total: 3, Active: 2},
		Volumes:     engine.UsageCategory{Total: 2, Active: 1, Size: 50_000_000, Reclaimable: 10_000_000},
		BuildCache:  engine.UsageCategory{Total: 1, Size: 700_000_000, Reclaimable: 700_000_000},
		Used:        1_250_000_000,
		Reclaimable: 810_000_000,
	}
	daemon := engine.DaemonInfo{ServerVersion: "27.3.1", Arch: "x86_64", OS: "Ubuntu 24.04.1 LTS", NCPU: 16, MemTotal: 34_359_738_368}
	system := System(plainSheet(), usage, daemon, true, nil, 120, 30, false)
	for _, want := range []string{"LOCAL STORAGE", "RECLAIMABLE SHARE", "build cache", "65%", "Docker Engine: 27.3.1"} {
		if !strings.Contains(system, want) {
			t.Fatalf("system view missing %q:\n%s", want, system)
		}
	}
}
