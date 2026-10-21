// Package engine defines daemon-independent resource and streaming contracts.
package engine

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Image is a local image summary and its container references.
type Image struct {
	ID                string
	RepoTags, Digests []string
	Size, Shared      int64
	Created           time.Time
	Dangling          bool
	UsedBy            []Container
}

// Name returns a human-readable reference, including untagged images.
func (i Image) Name() string {
	if len(i.RepoTags) > 0 {
		return i.RepoTags[0]
	}
	return "<none>:<none>"
}

// ShortID strips the digest algorithm and truncates the identifier safely.
func (i Image) ShortID() string { return ShortID(i.ID) }

// ShortID returns at most twelve characters of a digest.
func ShortID(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// Container is the lightweight information needed by lists and usage indices.
type Container struct {
	ID, Name, ImageID, Image, State, Status, Health string
	Created                                         time.Time
	Ports                                           []string
	Mounts                                          []Mount
	Project, Service, ProjectDir                    string
	ProjectConfigFiles                              []string
	Labels                                          map[string]string
}

// Running reports whether the container has a live main process.
func (c Container) Running() bool {
	return c.State == "running" || c.State == "paused" || c.State == "restarting"
}

// Mount is a named volume or bind mount visible without inspecting a container.
type Mount struct {
	Type, Name, Source, Destination, Mode string
	RW                                    bool
}

// HealthProbe holds daemon execution facts (CTR-07), not inferred health transitions.
// Status and FailingStreak describe the current aggregate ONLY on the compatibility
// ContainerInspection.HealthCheck. In DiagnosticEvidence.HealthHistory these fields
// are always zero: daemon logs do not record historical aggregate health states.
type HealthProbe struct {
	// ExecutionObserved requires daemon execution timestamps, not a default exit code.
	ExecutionObserved bool
	Status            string
	FailingStreak     int
	Start, End        time.Time
	ExitCode          int
	Output            string `json:"-"`
}

// ContainerInspection is structured, SDK-independent container metadata.
type ContainerInspection struct {
	ID, Name, Image, ImageID, Platform          string
	State, Health, Status, NetworkMode, Restart string
	Created, StartedAt, FinishedAt              time.Time
	ExitCode, RestartCount                      int
	OOMKilled, TTY                              bool
	Env                                         []string
	Entrypoint, Command                         []string `json:"-"`
	Ports                                       []string
	Networks                                    []string
	Mounts                                      []Mount
	Labels                                      map[string]string
	HealthCheck                                 *HealthProbe
	// InspectedAt is successful inspect fetch completion, not container start.
	InspectedAt time.Time
	diagnostics DiagnosticEvidence
}

// RunSpec contains the options exposed by the run-from-image wizard.
type RunSpec struct {
	Name, Image, Network, Restart string
	Command, Ports, Env, Volumes  []string
}

// ContainerStats is one non-streaming sample of measured container resources.
// CPU and system totals are cumulative; callers derive a percentage from two
// samples taken close together.
type ContainerStats struct {
	State                                                                          TelemetryState
	CPUAvailable, MemoryAvailable, NetworkAvailable, BlockAvailable, PIDsAvailable bool
	At                                                                             time.Time
	CPUTotal, SystemTotal                                                          uint64
	OnlineCPUs                                                                     uint32
	Memory, MemoryLimit                                                            uint64
	NetworkRX, NetworkTX                                                           uint64
	BlockRead, BlockWrite                                                          uint64
	PIDs                                                                           uint64
}

// Current is the shared quick-stat policy. Unknown timestamps, stopped
// collection, non-running lifecycles and samples older than ten seconds are
// never presented as live measurements. Availability remains per metric.
func (s ContainerStats) Current(state string, now time.Time) bool {
	return state == "running" && s.State == TelemetryActive && !s.At.IsZero() && !s.At.After(now) && now.Sub(s.At) <= 10*time.Second
}

// TelemetryState distinguishes fresh samples from session history and absence.
type TelemetryState int

const (
	TelemetryActive TelemetryState = iota
	TelemetrySampled
	TelemetryUnavailable
)

// CPUPercent derives a percentage from cumulative samples. A measured zero is
// available; missing samples, resets, and non-positive intervals are not.
func (s ContainerStats) CPUPercent(previous ContainerStats) (float64, bool) {
	if !s.CPUAvailable || !previous.CPUAvailable || s.State == TelemetryUnavailable || previous.State == TelemetryUnavailable ||
		!s.At.After(previous.At) || s.OnlineCPUs == 0 || s.SystemTotal <= previous.SystemTotal || s.CPUTotal < previous.CPUTotal {
		return 0, false
	}
	return float64(s.CPUTotal-previous.CPUTotal) / float64(s.SystemTotal-previous.SystemTotal) * float64(s.OnlineCPUs) * 100, true
}

// LogOptions controls one bounded or following container log stream.
type LogOptions struct {
	Follow bool
	Tail   int
	Until  time.Time
	// Previous requests a bounded historical read. Until must be the restart boundary.
	Previous bool
}

// LogEntry is one demultiplexed log line with its optional daemon timestamp.
type LogEntry struct {
	At           time.Time
	Stream, Text string
}

// Volume is a daemon volume summary with usage data when the daemon supplies it.
type Volume struct {
	Name, Driver, Mountpoint, Scope string
	CreatedAt                       time.Time
	Size                            int64
	RefCount                        int
	Labels                          map[string]string
	UsedBy                          []Container
}

// UsageCategory is one row in the disk-usage breakdown.
type UsageCategory struct {
	Total, Active int
	Size          int64
	Reclaimable   int64
}

// SystemUsage is the daemon disk-usage snapshot used by the soundings panel.
type SystemUsage struct {
	Images, Containers, Volumes, BuildCache UsageCategory
	Used, Reclaimable                       int64
}

// ContainerAction is a safe, non-forced lifecycle operation.
type ContainerAction string

// Container lifecycle actions supported by the engine layer.
const (
	ActionStart   ContainerAction = "start"
	ActionStop    ContainerAction = "stop"
	ActionRestart ContainerAction = "restart"
	ActionPause   ContainerAction = "pause"
	ActionUnpause ContainerAction = "unpause"
	ActionKill    ContainerAction = "kill"
	ActionRemove  ContainerAction = "remove"
)

// Layer records an image history entry, including metadata-only entries.
type Layer struct {
	ID, Command string
	Size        int64
}

// VulnerabilityReport records security scanning findings from Trivy, Grype, or Scout (IMG-10).
type VulnerabilityReport struct {
	Scanner   string
	Critical  int
	High      int
	Medium    int
	Low       int
	Fixable   int
	ScannedAt time.Time
	Error     string
}

// ImageUpdateStatus tracks whether an image's remote tag digest has drifted from the local image (IMG-09).
type ImageUpdateStatus struct {
	Ref          string
	LocalDigest  string
	RemoteDigest string
	HasUpdate    bool
	CheckedAt    time.Time
}

// Inspection contains structured image metadata. Services mask values by default.
type Inspection struct {
	ID, Platform                             string
	Env, Entrypoint, Command, Ports, Digests []string
	Labels                                   map[string]string
	Layers                                   []Layer
	Vulnerabilities                          *VulnerabilityReport
	UpdateStatus                             *ImageUpdateStatus
}

// Progress is one Docker pull stream update.
type Progress struct {
	ID, Status     string
	Current, Total int64
}

// Event reports a resource change or stream connection state.
type Event struct {
	ID, Kind, Action string
	At               time.Time
	Err              error
}

// NetworkEndpoint represents a container attached to a Docker network.
type NetworkEndpoint struct {
	ContainerID, Name, IPv4, IPv6, MacAddress string
}

// Network represents a Docker bridge, overlay, macvlan, or custom network.
type Network struct {
	ID, Name, Driver, Scope string
	Internal, Attachable    bool
	Ingress, IPv6           bool
	Subnets, Gateways       []string
	Containers              []NetworkEndpoint
	Labels                  map[string]string
	Created                 time.Time
}

// DaemonInfo contains system-wide engine and host telemetry (SYS-03).
type DaemonInfo struct {
	ID, Name                                           string
	ServerVersion, APIVersion, MinAPIVersion           string
	OS, OSType, Arch, KernelVersion                    string
	Driver, LoggingDriver, CgroupDriver, CgroupVersion string
	RootDir                                            string
	NCPU                                               int
	MemTotal                                           int64
	SecurityOptions, Warnings                          []string
}

// ComposeAction is an operation performed on a Compose project.
type ComposeAction string

// Compose lifecycle actions supported by the engine layer.
const (
	ComposeUp      ComposeAction = "up"
	ComposeDown    ComposeAction = "down"
	ComposeRestart ComposeAction = "restart"
	ComposePull    ComposeAction = "pull"
)

// ComposeOptions configures execution of a compose action.
type ComposeOptions struct {
	ProjectDir    string
	ConfigFiles   []string
	ProjectName   string
	RemoveVolumes bool
	Services      []string
}

// ComposeProject represents a multi-container Docker Compose application (PRJ-01).
type ComposeProject struct {
	Name, WorkingDir string
	ConfigFiles      []string
	Services         []string
	Status           string
	Containers       []Container
}

// RegistrySearchResult is an entry from searching a remote container registry (IMG-06).
type RegistrySearchResult struct {
	Name        string
	Description string
	Stars       int
	Official    bool
	Automated   bool
}

// RemoteTag is a published tag with its manifest digest and timestamp (IMG-06).
type RemoteTag struct {
	Name      string
	Digest    string
	Size      int64
	UpdatedAt time.Time
}

// Engine abstracts local daemon access. Stream callbacks must not be retained.
type Engine interface {
	ListImages(context.Context) ([]Image, error)
	ListContainers(context.Context) ([]Container, error)
	InspectImage(context.Context, string) (Inspection, error)
	RemoveImage(context.Context, string) error
	Pull(context.Context, string, func(Progress) error) error

	ListVolumes(context.Context) ([]Volume, error)
	RemoveVolume(context.Context, string) error
	SystemUsage(context.Context) (SystemUsage, error)

	InspectContainer(context.Context, string) (ContainerInspection, error)
	ContainerStats(context.Context, string) (ContainerStats, error)
	CreateContainer(context.Context, RunSpec) (string, error)
	ActContainer(context.Context, string, ContainerAction) error
	Logs(context.Context, string, LogOptions, func(LogEntry) error) error
	PruneBuildCache(context.Context) (int64, error)

	ListNetworks(context.Context) ([]Network, error)
	InspectNetwork(context.Context, string) (Network, error)
	CreateNetwork(context.Context, string, string) (string, error)
	RemoveNetwork(context.Context, string) error
	ConnectNetwork(context.Context, string, string) error
	DisconnectNetwork(context.Context, string, string, bool) error

	DaemonInfo(context.Context) (DaemonInfo, error)
	Compose(context.Context, ComposeAction, ComposeOptions, func(string) error) error

	SearchRegistry(context.Context, string) ([]RegistrySearchResult, error)
	ListRemoteTags(context.Context, string) ([]RemoteTag, error)
	CheckImageUpdate(context.Context, string, string) (ImageUpdateStatus, error)
	TagImage(context.Context, string, string) error
	PushImage(context.Context, string, func(Progress) error) error
	SaveImage(context.Context, string, string) error
	LoadImage(context.Context, string) error

	Events(context.Context, func(Event) error) error
	Close() error
}

// ErrNotImplemented identifies deliberately deferred capabilities.
type ErrNotImplemented struct{ Requirement string }

func (e ErrNotImplemented) Error() string {
	return fmt.Sprintf("%s is not implemented yet", e.Requirement)
}
