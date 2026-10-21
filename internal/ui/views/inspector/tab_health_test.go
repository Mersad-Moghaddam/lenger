package inspector

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
)

func TestHealthInitialTab(t *testing.T) {
	if TabHealth != 0 || TabLogs != 1 || TabMetrics != 2 || TabEnv != 3 || TabTimeline != 4 {
		t.Fatal("initial tab constants changed")
	}
	for _, tc := range []struct {
		name string
		c    engine.ContainerInspection
		want InitialTab
	}{
		{"exited137", engine.ContainerInspection{State: "exited", ExitCode: 137}, TabHealth},
		{"exited0", engine.ContainerInspection{State: "exited"}, TabHealth},
		{"healthy", engine.ContainerInspection{State: "running", Health: "healthy", HealthCheck: &engine.HealthProbe{Status: "healthy"}}, TabLogs},
		{"nilProbe", engine.ContainerInspection{State: "running"}, TabLogs},
		{"probeHealthy", engine.ContainerInspection{State: "running", HealthCheck: &engine.HealthProbe{Status: "healthy"}}, TabLogs},
		{"unhealthy", engine.ContainerInspection{State: "running", Health: "unhealthy"}, TabHealth},
		{"starting", engine.ContainerInspection{State: "running", HealthCheck: &engine.HealthProbe{Status: "starting"}}, TabHealth},
		{"paused", engine.ContainerInspection{State: "paused"}, TabHealth},
		{"restarting", engine.ContainerInspection{State: "restarting"}, TabHealth},
		{"unknownState", engine.ContainerInspection{State: "future-state"}, TabHealth},
		{"unknownHealth", engine.ContainerInspection{State: "running", Health: "unknown"}, TabHealth},
		{"emptyProbe", engine.ContainerInspection{State: "running", HealthCheck: &engine.HealthProbe{}}, TabHealth},
		{"conflictingProbe", engine.ContainerInspection{State: "running", Health: "healthy", HealthCheck: &engine.HealthProbe{Status: "unhealthy"}}, TabHealth},
		{"zero", engine.ContainerInspection{}, TabHealth},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := DetermineInitialTab(tc.c); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func healthText(c engine.ContainerInspection, peak uint64, observed bool) string {
	return RenderHealthTab(style.Sheet{Plain: true}, c, peak, observed, 180, 80)
}

func TestHealthProvenance(t *testing.T) {
	got := healthText(engine.ContainerInspection{State: "exited", ExitCode: 137, OOMKilled: true, RestartCount: 2, Restart: "on-failure"}, 64*1024*1024, true)
	inspect := strings.Index(got, "[Docker Engine Inspect]")
	stats := strings.Index(got, "[Telemetry: Docker API ContainerStats]")
	hypothesis := strings.Index(got, "[Diagnostic Hypothesis & Next Actions]")
	if inspect < 0 || stats <= inspect || hypothesis <= stats {
		t.Fatalf("missing or unordered provenance sections:\n%s", got)
	}
	for _, fact := range []string{"State: exited", "ExitCode: 137", "OOMKilled: true", "RestartCount: 2", "Restart policy: on-failure"} {
		if !strings.Contains(got[inspect:stats], fact) {
			t.Errorf("missing inspect fact %q", fact)
		}
	}
	if !strings.Contains(got[stats:hypothesis], "Peak memory: 64.00 MiB") {
		t.Error("missing session peak")
	}
	for _, text := range []string{"Hypothesis:", "[p] View Previous Instance Logs", "[r] Restart", "[c] View Failing Healthcheck Command"} {
		if !strings.Contains(got[hypothesis:], text) {
			t.Errorf("missing hypothesis/action %q", text)
		}
	}
}

func TestHealthNilProbeAndMissingPeaks(t *testing.T) {
	for _, state := range []string{"running", "paused", "restarting", "exited", "dead", "unknown"} {
		got := healthText(engine.ContainerInspection{State: state}, 999, false)
		if !strings.Contains(got, "State: "+state) || !strings.Contains(got, "Health: unavailable (no probe metadata)") {
			t.Errorf("nil probe state %s:\n%s", state, got)
		}
		if state == "exited" || state == "dead" {
			if !strings.Contains(got, "Unavailable (no session sample)") || strings.Contains(got, "prior to telemetry") {
				t.Error("wrapper cannot attribute exit to prior session")
			}
		} else if state == "unknown" {
			if !strings.Contains(got, "Unavailable (no session sample)") {
				t.Error("unknown must not claim active collection")
			}
		} else if !strings.Contains(got, "Collecting (no session sample yet)") {
			t.Error("alive must not claim exited")
		}
		if strings.Contains(got, "999") {
			t.Error("unobserved peak leaked")
		}
	}
	if got := healthText(engine.ContainerInspection{State: "exited"}, 0, true); !strings.Contains(got, "Peak memory: 0 B") || strings.Contains(got, "container exited prior") {
		t.Error("observed zero must be valid")
	}
}

func TestHealthEvidenceAndCompactActions(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	zero := uint64(0)
	c := engine.ContainerInspection{State: "running", Health: "healthy", HealthCheck: &engine.HealthProbe{Status: "unhealthy", ExitCode: 0}}
	e := HealthEvidence{InspectedAt: now, ObservedAt: now.Add(-time.Minute), SessionStarted: now.Add(-time.Hour), ConfiguredMemory: &zero, ProbeExecutionObserved: true, DaemonErrorKnown: true, DaemonError: "credential-secret", HealthConfig: &HealthCheckEvidence{Command: []string{"credential-secret"}, Interval: time.Second}, HealthHistory: []engine.HealthProbe{{End: now, ExitCode: 0, Output: "credential-secret"}}, HealthHistoryKnown: true}
	full := RenderHealthTabWithEvidence(style.Sheet{Plain: true}, c, 0, true, e, 180, 80)
	for _, text := range []string{now.Format(time.RFC3339), e.ObservedAt.Format(time.RFC3339), "Memory limit: 0 B (unlimited)", "Probe exit code: 0", "Health conflict: aggregate healthy; probe unhealthy", "Daemon error: withheld", "Healthcheck interval: 1s", "History probe exit code: 0"} {
		if !strings.Contains(full, text) {
			t.Errorf("missing evidence %q", text)
		}
	}
	if strings.Contains(full, "credential-secret") {
		t.Fatal("optional evidence leaked secrets")
	}
	for _, size := range [][2]int{{60, 20}, {80, 24}, {100, 18}} {
		got := RenderHealthTabWithEvidence(style.Sheet{Plain: true}, c, 0, true, e, size[0], size[1])
		for _, text := range []string{"[Docker Engine Inspect]", "State: running", "Health conflict:", "ExitCode: 0", "OOMKilled: false", "[Telemetry: Docker API ContainerStats]", "Peak memory: 0 B", "Hypothesis:", "[p] View Previous Instance Logs", "[r] Restart", "[c] View Failing Healthcheck Command"} {
			if !strings.Contains(got, text) {
				t.Errorf("size %v missing essential %q:\n%s", size, text, got)
			}
		}
	}
	for _, probeStatus := range []string{"starting", "unknown"} {
		c.HealthCheck.Status = probeStatus
		if got := RenderHealthTabWithEvidence(style.Sheet{Plain: true}, c, 0, false, e, 180, 80); !strings.Contains(got, "Health conflict: aggregate healthy; probe "+probeStatus) {
			t.Fatal("conflict not explicit")
		}
	}
	c.Health, c.HealthCheck.Status = "unhealthy", "healthy"
	if got := healthText(c, 0, false); !strings.Contains(got, "Health conflict: aggregate unhealthy; probe healthy") {
		t.Fatal("reverse conflict not explicit")
	}
	c.State, c.FinishedAt = "exited", now.Add(-2*time.Hour)
	if got := RenderHealthTabWithEvidence(style.Sheet{Plain: true}, c, 0, false, e, 180, 80); !strings.Contains(got, "container exited prior to telemetry session") {
		t.Fatal("known prior-session exit not attributed")
	}
	c.FinishedAt = now
	if got := RenderHealthTabWithEvidence(style.Sheet{Plain: true}, c, 0, false, e, 180, 80); strings.Contains(got, "container exited prior") {
		t.Fatal("post-session exit incorrectly attributed")
	}
}

func TestHealthNoInventedEvidence(t *testing.T) {
	start := time.Date(2026, 10, 20, 1, 2, 3, 0, time.UTC)
	got := healthText(engine.ContainerInspection{State: "exited", ExitCode: 137, StartedAt: start}, 1, true)
	if !strings.Contains(got, "OOMKilled: false") || strings.Contains(got, "OOMKilled: true") {
		t.Error("exit137 is not proof of OOM")
	}
	for _, text := range []string{"Inspect timestamp: unavailable", "Sample timestamp: unavailable", "Memory limit: unavailable", "Daemon error: unavailable", "Healthcheck configuration/history: unavailable"} {
		if !strings.Contains(got, text) {
			t.Errorf("missing domain gap %q", text)
		}
	}
	stats := strings.Index(got, "[Telemetry:")
	if strings.Contains(got[stats:], start.Format(time.RFC3339)) {
		t.Error("container start presented as sampling time")
	}
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "memory pressure") && !strings.Contains(line, "Hypothesis:") {
			t.Error("inference not prefixed")
		}
	}
}

func TestHealthProbeAndSecretPolicy(t *testing.T) {
	start := time.Date(2026, 10, 21, 1, 0, 0, 0, time.UTC)
	c := engine.ContainerInspection{State: "running", Health: "starting", Env: []string{"API_TOKEN=very-secret-token", "PUBLIC=innocent"}, Command: []string{"secret-command"}, HealthCheck: &engine.HealthProbe{Status: "starting", FailingStreak: 3, Start: start, End: start.Add(2 * time.Second), ExitCode: 1, Output: "API_TOKEN=very-secret-token\nhttps://user:password@host arbitrary-secret\x1b[2J"}}
	got := healthText(c, 0, false)
	for _, text := range []string{"Health: starting", "health checks are initializing", "Failing streak: 3", "Probe exit code: 1", "Probe duration: 2s", "Probe output: withheld (may contain secrets)"} {
		if !strings.Contains(got, text) {
			t.Errorf("missing probe fact %q", text)
		}
	}
	for _, secret := range []string{"very-secret-token", "arbitrary-secret", "password", "secret-command", "innocent", "\x1b[2J"} {
		if strings.Contains(got, secret) {
			t.Errorf("diagnostics leaked %q", secret)
		}
	}
	// Even metadata strings can contain unexpected untrusted payloads.
	c.State, c.Health, c.Restart, c.HealthCheck.Status = "secret-state", "secret-health", "secret-policy", "secret-probe"
	got = healthText(c, 0, false)
	if strings.Contains(got, "secret-") {
		t.Error("unrecognized metadata must not be echoed")
	}
}

func TestHealthBounds(t *testing.T) {
	for _, size := range [][2]int{{0, 5}, {5, 0}, {-1, 2}, {1, 1}, {40, 12}, {120, 60}} {
		got := RenderHealthTab(style.Sheet{Plain: true}, engine.ContainerInspection{}, 0, false, size[0], size[1])
		if size[0] <= 0 || size[1] <= 0 {
			if got != "" {
				t.Error("nonpositive rectangle")
			}
			continue
		}
		if lipgloss.Width(got) != size[0] || lipgloss.Height(got) != size[1] {
			t.Errorf("wrong rectangle %v", size)
		}
	}
}

func TestHealthProbeExecutionValidity(t *testing.T) {
	for _, status := range []string{"", "starting"} {
		got := healthText(engine.ContainerInspection{State: "running", HealthCheck: &engine.HealthProbe{Status: status}}, 0, false)
		if strings.Contains(got, "Probe exit code: 0") || strings.Contains(got, "Probe output: empty") {
			t.Fatal("missing execution presented as successful probe")
		}
		if !strings.Contains(got, "Probe execution: unavailable (not yet recorded)") {
			t.Fatal("missing execution explanation")
		}
	}
	got := healthText(engine.ContainerInspection{State: "running", HealthCheck: &engine.HealthProbe{End: time.Now(), ExitCode: 0}}, 0, false)
	if !strings.Contains(got, "Probe exit code: 0") {
		t.Fatal("recorded zero exit code must remain valid")
	}
}

func TestHealthCompactExecutionAndInvalidCounters(t *testing.T) {
	c := engine.ContainerInspection{State: "running", Health: "starting", RestartCount: -1, HealthCheck: &engine.HealthProbe{Status: "starting", FailingStreak: -1}}
	for _, height := range []int{18, 20, 24} {
		got := RenderHealthTab(style.Sheet{Plain: true}, c, 0, false, 80, height)
		for _, text := range []string{"Probe execution: unavailable", "health checks are initializing", "[p] View Previous Instance Logs", "[r] Restart", "[c] View Failing Healthcheck Command"} {
			if !strings.Contains(got, text) {
				t.Errorf("height %d missing %q", height, text)
			}
		}
	}
	got := healthText(c, 0, false)
	if strings.Contains(got, "RestartCount: -1") || strings.Contains(got, "Failing streak: -1") || !strings.Contains(got, "RestartCount: unavailable") || !strings.Contains(got, "Failing streak: unavailable") {
		t.Fatal("invalid counter rendered as verified fact")
	}
}
