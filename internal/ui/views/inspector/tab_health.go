package inspector

import (
	"fmt"
	"strings"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
)

type InitialTab int

const (
	TabHealth InitialTab = iota
	TabLogs
	TabMetrics
	TabEnv
	TabTimeline
)

// DetermineInitialTab is an opening-time decision only. The caller must latch
// the result, not recompute the active tab on background inspection updates.
func DetermineInitialTab(c engine.ContainerInspection) InitialTab {
	if c.State != "running" {
		return TabHealth
	}
	if c.Health != "" && c.Health != "healthy" {
		return TabHealth
	}
	if c.HealthCheck != nil && c.HealthCheck.Status != "healthy" {
		return TabHealth
	}
	return TabLogs
}

// healthEnum deliberately does not echo arbitrary metadata: even fields that
// usually hold short enums can carry unexpected terminal controls or secrets.
func healthEnum(value string, allowed ...string) string {
	for _, candidate := range allowed {
		if value == candidate {
			return candidate
		}
	}
	return "unknown"
}

// HealthEvidence is optional inspect/session metadata supplied by the caller.
// Zero times mean unknown. ConfiguredMemory nil means unknown; zero means no
// configured limit. ObservedAt is the peak sample's time, not the latest poll.
// Known flags distinguish absent data from a known empty error/history.
type HealthEvidence struct {
	InspectedAt, ObservedAt, SessionStarted time.Time
	ConfiguredMemory                        *uint64
	DaemonError                             string
	DaemonErrorKnown                        bool
	ProbeExecutionObserved                  bool
	HealthConfig                            *HealthCheckEvidence
	HealthHistory                           []engine.HealthProbe
	HealthHistoryKnown                      bool
}

// HealthCheckEvidence carries inspect configuration without authorizing command
// disclosure. Command is never rendered by this diagnostic view.
type HealthCheckEvidence struct {
	Command                        []string
	Interval, Timeout, StartPeriod time.Duration
	Retries                        int
}

// RenderHealthTab separates daemon facts, session observations and inference.
// No timestamp is invented: the current inputs carry neither inspection time
// nor the time of the peak sample. Probe output is always withheld because
// key-based redaction alone cannot safely sanitize arbitrary command output.
func RenderHealthTab(s style.Sheet, c engine.ContainerInspection, peakMemory uint64, peakObserved bool, w, h int) string {
	return RenderHealthTabWithEvidence(s, c, peakMemory, peakObserved, HealthEvidence{}, w, h)
}

// RenderHealthTabWithEvidence retains a compact essential summary and reserves
// the action footer before optional detail. Upstream metadata wiring is required.
func RenderHealthTabWithEvidence(s style.Sheet, c engine.ContainerInspection, peakMemory uint64, peakObserved bool, e HealthEvidence, w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	state := healthEnum(c.State, "created", "running", "paused", "restarting", "removing", "exited", "dead")
	health := "unavailable (no probe metadata)"
	if c.Health != "" {
		health = healthEnum(c.Health, "healthy", "unhealthy", "starting")
	} else if c.HealthCheck != nil {
		health = healthEnum(c.HealthCheck.Status, "healthy", "unhealthy", "starting")
	}
	policy := healthEnum(c.Restart, "no", "always", "unless-stopped", "on-failure")
	if c.Restart == "" {
		policy = "unavailable"
	}
	inspectTime := "Inspect timestamp: unavailable"
	if !e.InspectedAt.IsZero() {
		inspectTime = "Inspect timestamp: " + e.InspectedAt.Format(time.RFC3339)
	}
	healthLine := "Health: " + health
	if c.Health != "" && c.HealthCheck != nil && c.Health != c.HealthCheck.Status {
		healthLine = "Health conflict: aggregate " + health + "; probe " + healthEnum(c.HealthCheck.Status, "healthy", "unhealthy", "starting")
	}
	lines := []string{
		s.Heading("[Docker Engine Inspect]"),
		"State: " + state,
		healthLine,
		fmt.Sprintf("ExitCode: %d · OOMKilled: %t", c.ExitCode, c.OOMKilled),
	}
	details := []string{inspectTime}
	if c.RestartCount >= 0 {
		details = append(details, fmt.Sprintf("RestartCount: %d", c.RestartCount))
	} else {
		details = append(details, "RestartCount: unavailable")
	}
	details = append(details, "Restart policy: "+policy)
	daemonError := "Daemon error: unavailable"
	if e.DaemonErrorKnown {
		daemonError = "Daemon error: none"
		if e.DaemonError != "" {
			daemonError = "Daemon error: withheld (may contain secrets)"
		}
	}
	details = append(details, daemonError)
	if e.HealthConfig == nil {
		details = append(details, "Healthcheck configuration/history: unavailable (configuration not supplied)")
	} else {
		details = append(details, "Healthcheck command: withheld (may contain secrets)", "Healthcheck interval: "+e.HealthConfig.Interval.String(), "Healthcheck timeout: "+e.HealthConfig.Timeout.String(), "Healthcheck start period: "+e.HealthConfig.StartPeriod.String())
		if e.HealthConfig.Retries >= 0 {
			details = append(details, fmt.Sprintf("Healthcheck retries: %d", e.HealthConfig.Retries))
		}
	}
	if !e.HealthHistoryKnown {
		details = append(details, "Healthcheck history: unavailable")
	} else {
		details = append(details, fmt.Sprintf("Healthcheck history: %d records", len(e.HealthHistory)))
		for _, probe := range e.HealthHistory {
			if probe.End.IsZero() {
				details = append(details, "History probe execution: unavailable")
				continue
			}
			details = append(details, "History probe completed: "+probe.End.Format(time.RFC3339), fmt.Sprintf("History probe exit code: %d", probe.ExitCode))
			if !probe.Start.IsZero() && !probe.End.Before(probe.Start) {
				details = append(details, "History probe duration: "+probe.End.Sub(probe.Start).String())
			}
			details = append(details, "History probe output: withheld (may contain secrets)")
		}
	}
	if health == "starting" || (c.HealthCheck != nil && c.HealthCheck.Status == "starting") {
		lines = append(lines, s.Muted("Health: health checks are initializing"))
	} else if health == "unknown" {
		lines = append(lines, s.Muted("Health status unknown; initialization cannot be confirmed"))
	}
	if probe := c.HealthCheck; probe != nil {
		details = append(details,
			"Latest probe status: "+healthEnum(probe.Status, "healthy", "unhealthy", "starting"),
		)
		if probe.FailingStreak >= 0 {
			details = append(details, fmt.Sprintf("Failing streak: %d", probe.FailingStreak))
		} else {
			details = append(details, "Failing streak: unavailable")
		}
		if probe.End.IsZero() && !e.ProbeExecutionObserved {
			lines = append(lines, "Probe execution: unavailable (not yet recorded)")
		} else {
			lines = append(lines, fmt.Sprintf("Probe exit code: %d", probe.ExitCode))
			if !probe.Start.IsZero() && !probe.End.IsZero() && !probe.End.Before(probe.Start) {
				details = append(details, "Probe duration: "+probe.End.Sub(probe.Start).String())
			} else {
				details = append(details, s.Muted("Probe duration: unavailable"))
			}
			if probe.Output != "" {
				details = append(details, s.Muted("Probe output: withheld (may contain secrets)"))
			} else {
				details = append(details, "Probe output: empty")
			}
		}
	}
	// Details remain in the inspect tier, but are lower priority than essentials.
	telemetry := []string{s.Heading("[Telemetry: Docker API ContainerStats]")}
	if peakObserved {
		peak := "0 B"
		if peakMemory != 0 {
			peak = fmt.Sprintf("%.2f MiB (%d B)", float64(peakMemory)/(1024*1024), peakMemory)
		}
		telemetry = append(telemetry, "Peak memory: "+peak+" (observed this session)")
	} else {
		absence := "Unavailable (no session sample)"
		switch c.State {
		case "exited", "dead":
			if !e.SessionStarted.IsZero() && !c.FinishedAt.IsZero() && c.FinishedAt.Before(e.SessionStarted) {
				absence = "Unavailable (container exited prior to telemetry session)"
			}
		case "running", "paused", "restarting":
			absence = "Collecting (no session sample yet)"
		}
		telemetry = append(telemetry, "Peak memory: "+absence)
	}
	sampleTime := "Sample timestamp: unavailable"
	if peakObserved && !e.ObservedAt.IsZero() {
		sampleTime = "Sample timestamp: " + e.ObservedAt.Format(time.RFC3339)
	}
	limit := "Memory limit: unavailable"
	if e.ConfiguredMemory != nil {
		limit = fmt.Sprintf("Memory limit: %d B", *e.ConfiguredMemory)
		if *e.ConfiguredMemory == 0 {
			limit += " (unlimited)"
		}
	}
	telemetry = append(telemetry, sampleTime, limit)
	hypothesis := "Hypothesis: No specific cause established; inspect logs and healthcheck configuration."
	switch {
	case c.OOMKilled:
		hypothesis = "Hypothesis: memory pressure may explain termination; inspect memory limits and workload."
	case c.ExitCode == 137:
		hypothesis = "Hypothesis: SIGKILL may explain exit 137; this alone does not establish an OOM kill."
	case health == "unhealthy" || (c.HealthCheck != nil && c.HealthCheck.Status == "unhealthy"):
		hypothesis = "Hypothesis: The healthcheck may be failing; review its configuration and dependencies."
	case health == "starting":
		hypothesis = "Hypothesis: Healthcheck initialization may still be in progress."
	}
	footer := []string{s.Heading("[Diagnostic Hypothesis & Next Actions]"), s.Text(hypothesis, s.Palette.Warning),
		"[p] View Previous Instance Logs", "[r] Restart", "[c] View Failing Healthcheck Command",
	}
	// Keep all essential tiers and actions at supported drawer sizes. Optional
	// inspect details consume only spare rows, never push the footer offscreen.
	spare := h - len(lines) - len(telemetry) - len(footer)
	if spare > 0 {
		lines = append(lines, details[:min(spare, len(details))]...)
	}
	lines = append(lines, telemetry...)
	lines = append(lines, footer...)
	return style.Fit(strings.Join(lines, "\n"), w, h)
}
