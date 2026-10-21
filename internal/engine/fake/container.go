package fake

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// InspectContainer returns deterministic structured metadata, including a
// fixture secret which the service must mask.
func (e *Engine) InspectContainer(ctx context.Context, id string) (engine.ContainerInspection, error) {
	if err := ctx.Err(); err != nil {
		return engine.ContainerInspection{}, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	n := slices.IndexFunc(e.containers, func(c engine.Container) bool { return c.ID == id || c.Name == id })
	if n < 0 {
		return engine.ContainerInspection{}, fmt.Errorf("container not found")
	}
	c := e.containers[n]
	out := engine.ContainerInspection{
		ID: c.ID, Name: c.Name, Image: c.Image, ImageID: c.ImageID,
		Platform: "linux/amd64", State: c.State, Status: c.Status, Health: c.Health,
		Created: c.Created, StartedAt: c.Created, Restart: "unless-stopped",
		NetworkMode: "bridge", Networks: []string{"storefront_default"},
		Env:        []string{"PATH=/usr/local/bin:/usr/bin", "DB_PASSWORD=fixture-secret", "TZ=UTC"},
		Ports:      slices.Clone(c.Ports),
		Mounts:     slices.Clone(c.Mounts),
		Labels:     maps.Clone(c.Labels),
		TTY:        false,
		Entrypoint: []string{"/docker-entrypoint.sh"},
		Command:    []string{"server", "--verbose"},
	}
	if len(out.Labels) == 0 {
		out.Labels = map[string]string{"org.opencontainers.image.title": c.Name}
	}
	if c.Health != "" {
		out.HealthCheck = &engine.HealthProbe{
			Status:        c.Health,
			FailingStreak: 0,
			Start:         time.Now().UTC().Add(-15 * time.Second),
			End:           time.Now().UTC().Add(-14 * time.Second),
			ExitCode:      0,
			Output:        "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n{\"status\":\"pass\",\"uptime\":10800}",
		}
	}
	if c.State == "exited" {
		out.FinishedAt = c.Created.Add(time.Hour)
		out.StartedAt = time.Time{}
	}
	return out, nil
}

// ContainerStats returns a deterministic sample that changes with wall time so
// the UI can exercise delta-based CPU calculations without a daemon.
func (e *Engine) ContainerStats(ctx context.Context, id string) (engine.ContainerStats, error) {
	if err := ctx.Err(); err != nil {
		return engine.ContainerStats{}, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	n := slices.IndexFunc(e.containers, func(c engine.Container) bool { return c.ID == id || c.Name == id })
	if n < 0 {
		return engine.ContainerStats{}, fmt.Errorf("container not found")
	}
	c := e.containers[n]
	if !c.Running() {
		return engine.ContainerStats{}, fmt.Errorf("container is not running")
	}
	memory := []uint64{96_000_000, 512_000_000, 18_000_000}[n%3]
	network := []uint64{48_000_000, 12_000_000, 3_000_000}[n%3]
	block := []uint64{6_000_000, 92_000_000, 1_000_000}[n%3]
	percent := []float64{18.4, 7.1, 0.8}[n%3]
	now := time.Now().UTC()
	system := uint64(now.UnixMilli()) * 1_000_000
	online := uint32(4)
	return engine.ContainerStats{
		At: now, CPUTotal: uint64(float64(system) / 100 * percent / float64(online)),
		SystemTotal: system, OnlineCPUs: online, Memory: memory, MemoryLimit: 1_000_000_000,
		NetworkRX: network, NetworkTX: network / 3, BlockRead: block, BlockWrite: block / 2,
		PIDs: uint64(8 + n),
	}, nil
}

// CreateContainer adds a fixture container in the created state.
func (e *Engine) CreateContainer(ctx context.Context, spec engine.RunSpec) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if spec.Name != "" && slices.ContainsFunc(e.containers, func(c engine.Container) bool { return c.Name == spec.Name }) {
		return "", fmt.Errorf("container name is already in use: %s", spec.Name)
	}
	var image engine.Image
	found := false
	for _, i := range e.images {
		if i.ID == spec.Image || i.Name() == spec.Image || slices.Contains(i.RepoTags, spec.Image) {
			image, found = i, true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("image not found: %s", spec.Image)
	}
	name := spec.Name
	if name == "" {
		name = fmt.Sprintf("lenger-%d", len(e.containers)+1)
	}
	id := e.nextContainerIDLocked()
	mounts, err := fixtureMounts(spec.Volumes)
	if err != nil {
		return "", err
	}
	c := engine.Container{
		ID: id, Name: name, ImageID: image.ID, Image: image.Name(),
		State: "created", Status: "Created", Created: time.Now().UTC(),
		Ports: slices.Clone(spec.Ports), Mounts: mounts,
		Labels: map[string]string{"lenger.run": "true"},
	}
	e.containers = append(e.containers, c)
	e.refreshVolumeRefsLocked()
	e.broadcastLocked(engine.Event{ID: id, Kind: "container", Action: "create", At: time.Now()})
	return id, nil
}

// ActContainer applies a deterministic lifecycle transition.
func (e *Engine) ActContainer(ctx context.Context, id string, action engine.ContainerAction) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	n := slices.IndexFunc(e.containers, func(c engine.Container) bool { return c.ID == id || c.Name == id })
	if n < 0 {
		return fmt.Errorf("container not found")
	}
	c := &e.containers[n]
	switch action {
	case engine.ActionStart:
		if c.State == "paused" {
			return fmt.Errorf("container is paused; unpause it first")
		}
		c.State, c.Status = "running", "Up less than a second"
	case engine.ActionStop:
		if !c.Running() {
			return fmt.Errorf("container is not running")
		}
		c.State, c.Status = "exited", "Exited (0)"
	case engine.ActionRestart:
		if c.State == "paused" {
			return fmt.Errorf("container is paused; unpause it first")
		}
		c.State, c.Status = "running", "Up less than a second"
	case engine.ActionPause:
		if c.State != "running" {
			return fmt.Errorf("only a running container can be paused")
		}
		c.State, c.Status = "paused", "Up less than a second (Paused)"
	case engine.ActionUnpause:
		if c.State != "paused" {
			return fmt.Errorf("container is not paused")
		}
		c.State, c.Status = "running", "Up less than a second"
	case engine.ActionKill:
		if !c.Running() {
			return fmt.Errorf("container is not running")
		}
		c.State, c.Status = "exited", "Exited (137)"
	case engine.ActionRemove:
		if c.Running() {
			return fmt.Errorf("container is running; stop it first")
		}
		e.containers = slices.Delete(e.containers, n, n+1)
		e.refreshVolumeRefsLocked()
	default:
		return fmt.Errorf("unsupported container action %q", action)
	}
	e.broadcastLocked(engine.Event{ID: id, Kind: "container", Action: string(action), At: time.Now()})
	return nil
}

func (e *Engine) nextContainerIDLocked() string {
	for n := 1; ; n++ {
		id := fmt.Sprintf("c%04d", n)
		if !slices.ContainsFunc(e.containers, func(c engine.Container) bool { return c.ID == id }) {
			return id
		}
	}
}

func (e *Engine) refreshVolumeRefsLocked() {
	counts := map[string]int{}
	for _, c := range e.containers {
		for _, m := range c.Mounts {
			if m.Type == "volume" && m.Name != "" {
				counts[m.Name]++
			}
		}
	}
	for n := range e.volumes {
		e.volumes[n].RefCount = counts[e.volumes[n].Name]
	}
}

func fixtureMounts(values []string) ([]engine.Mount, error) {
	result := make([]engine.Mount, 0, len(values))
	for _, value := range values {
		parts := strings.SplitN(value, ":", 3)
		if len(parts) < 2 || !strings.HasPrefix(parts[1], "/") {
			return nil, fmt.Errorf("invalid mount %q; use source:/absolute/target[:ro]", value)
		}
		mode := ""
		if len(parts) == 3 {
			mode = parts[2]
		}
		kind := "volume"
		name := parts[0]
		if strings.HasPrefix(parts[0], "/") {
			kind, name = "bind", ""
		}
		result = append(result, engine.Mount{Type: kind, Name: name, Source: parts[0], Destination: parts[1], Mode: mode, RW: mode != "ro"})
	}
	return result, nil
}
