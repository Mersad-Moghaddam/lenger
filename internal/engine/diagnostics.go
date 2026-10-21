package engine

import (
	"slices"
	"time"
)

// DiagnosticEvidence contains potentially sensitive daemon evidence. Obtain it
// deliberately through ContainerInspection.DiagnosticEvidence; do not render raw
// fields without an explicit reveal flow. Zero timestamps mean absent evidence.
type DiagnosticEvidence struct {
	ConfiguredMemory       *uint64
	DaemonError            string `json:"-"`
	DaemonErrorKnown       bool
	ProbeExecutionObserved bool
	HealthConfig           *HealthCheckConfig
	HealthHistory          []HealthProbe
	HealthHistoryKnown     bool
}

// HealthCheckConfig is the daemon's configuration, not evidence of execution.
type HealthCheckConfig struct {
	Command                        []string `json:"-"`
	Interval, Timeout, StartPeriod time.Duration
	Retries                        int
}

// MaxHealthHistory bounds retained latest daemon entries without inventing events.
const MaxHealthHistory = 32

// WithDiagnosticEvidence attaches a detached snapshot for engine adapters.
func (i ContainerInspection) WithDiagnosticEvidence(e DiagnosticEvidence) ContainerInspection {
	i.diagnostics = cloneDiagnosticEvidence(e)
	return i
}

// DiagnosticEvidence deliberately reveals a detached diagnostic snapshot.
func (i ContainerInspection) DiagnosticEvidence() DiagnosticEvidence {
	return cloneDiagnosticEvidence(i.diagnostics)
}

func cloneDiagnosticEvidence(e DiagnosticEvidence) DiagnosticEvidence {
	if e.ConfiguredMemory != nil {
		v := *e.ConfiguredMemory
		e.ConfiguredMemory = &v
	}
	if e.HealthConfig != nil {
		v := *e.HealthConfig
		v.Command = slices.Clone(v.Command)
		e.HealthConfig = &v
	}
	if len(e.HealthHistory) > MaxHealthHistory {
		e.HealthHistory = e.HealthHistory[len(e.HealthHistory)-MaxHealthHistory:]
	}
	e.HealthHistory = slices.Clone(e.HealthHistory)
	return e
}

// String and GoString keep ordinary diagnostic formatting secret-safe.
func (i ContainerInspection) String() string   { return "ContainerInspection{diagnostics:redacted}" }
func (i ContainerInspection) GoString() string { return i.String() }
func (e DiagnosticEvidence) String() string    { return "DiagnosticEvidence{raw:redacted}" }
func (e DiagnosticEvidence) GoString() string  { return e.String() }
func (c HealthCheckConfig) String() string     { return "HealthCheckConfig{command:redacted}" }
func (c HealthCheckConfig) GoString() string   { return c.String() }
func (p HealthProbe) String() string           { return "HealthProbe{output:redacted}" }
func (p HealthProbe) GoString() string         { return p.String() }
