// Package fake provides deterministic, cancellable fixtures without Docker.
package fake

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// Engine stores an isolated fixture daemon. It is safe for concurrent services.
type Engine struct {
	mu         sync.Mutex
	images     []engine.Image
	containers []engine.Container
	volumes    []engine.Volume
	networks   []engine.Network
	buildCache int64
	subs       map[uint64]chan engine.Event
	next       uint64
	// Delay controls the scripted pull and follow-log cadence; zero is immediate.
	Delay time.Duration
	// PullError injects a pull failure before progress starts.
	PullError error
}

// New returns a fresh, reproducible fixture daemon.
func New() *Engine {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	refs := []struct {
		name string
		size int64
	}{{"nginx:1.27-alpine", 48}, {"postgres:16", 432}, {"redis:7.4", 117}, {"golang:1.23", 843}, {"node:22-slim", 241}, {"ubuntu:24.04", 78}}
	e := &Engine{subs: make(map[uint64]chan engine.Event), Delay: 80 * time.Millisecond, buildCache: 700_000_000}
	for n, r := range refs {
		e.images = append(e.images, engine.Image{ID: fmt.Sprintf("sha256:%064x", n+1), RepoTags: []string{r.name}, Size: r.size * 1000 * 1000, Created: now.Add(-time.Duration(n+1) * 24 * time.Hour)})
	}
	e.volumes = []engine.Volume{
		{Name: "web-content", Driver: "local", Mountpoint: "/var/lib/docker/volumes/web-content/_data", Scope: "local", CreatedAt: now.Add(-72 * time.Hour), Size: 320_000_000, RefCount: 1},
		{Name: "postgres-data", Driver: "local", Mountpoint: "/var/lib/docker/volumes/postgres-data/_data", Scope: "local", CreatedAt: now.Add(-240 * time.Hour), Size: 1_420_000_000, RefCount: 1},
		{Name: "redis-data", Driver: "local", Mountpoint: "/var/lib/docker/volumes/redis-data/_data", Scope: "local", CreatedAt: now.Add(-120 * time.Hour), Size: 96_000_000, RefCount: 1},
		{Name: "scratch-data", Driver: "local", Mountpoint: "/var/lib/docker/volumes/scratch-data/_data", Scope: "local", CreatedAt: now.Add(-12 * time.Hour), Size: 64_000_000, RefCount: 0},
	}
	e.networks = []engine.Network{
		{
			ID: "bridge-id", Name: "bridge", Driver: "bridge", Scope: "local",
			Subnets: []string{"172.17.0.0/16"}, Gateways: []string{"172.17.0.1"},
			Created: now.Add(-30 * 24 * time.Hour),
			Containers: []engine.NetworkEndpoint{
				{ContainerID: "web1", Name: "web-1", IPv4: "172.17.0.2/16"},
				{ContainerID: "db1", Name: "postgres-1", IPv4: "172.17.0.3/16"},
			},
		},
		{
			ID: "host-id", Name: "host", Driver: "host", Scope: "local",
			Created: now.Add(-30 * 24 * time.Hour),
		},
		{
			ID: "none-id", Name: "none", Driver: "null", Scope: "local", Internal: true,
			Created: now.Add(-30 * 24 * time.Hour),
		},
		{
			ID: "storefront-net-id", Name: "storefront_default", Driver: "bridge", Scope: "local",
			Subnets: []string{"172.28.0.0/16"}, Gateways: []string{"172.28.0.1"}, Attachable: true,
			Created: now.Add(-72 * time.Hour),
			Labels: map[string]string{
				"com.docker.compose.project": "storefront",
				"com.docker.compose.network": "default",
			},
			Containers: []engine.NetworkEndpoint{
				{ContainerID: "web1", Name: "web-1", IPv4: "172.28.0.2/16"},
				{ContainerID: "db1", Name: "postgres-1", IPv4: "172.28.0.3/16"},
				{ContainerID: "redis1", Name: "redis-1", IPv4: "172.28.0.4/16"},
			},
		},
	}
	e.containers = []engine.Container{
		{ID: "web1", Name: "web-1", ImageID: e.images[0].ID, Image: e.images[0].Name(), State: "running", Status: "Up 3 hours", Health: "healthy", Created: now.Add(-3 * time.Hour), Ports: []string{"8080->80/tcp"}, Mounts: []engine.Mount{{Type: "volume", Name: "web-content", Source: "/var/lib/docker/volumes/web-content/_data", Destination: "/usr/share/nginx/html", RW: true}}, Labels: map[string]string{"com.docker.compose.project": "storefront", "com.docker.compose.service": "web"}, Project: "storefront", Service: "web", ProjectDir: "/opt/storefront", ProjectConfigFiles: []string{"/opt/storefront/docker-compose.yml"}},
		{ID: "db1", Name: "postgres-1", ImageID: e.images[1].ID, Image: e.images[1].Name(), State: "running", Status: "Up 1 day", Health: "healthy", Created: now.Add(-24 * time.Hour), Ports: []string{"5432/tcp"}, Mounts: []engine.Mount{{Type: "volume", Name: "postgres-data", Source: "/var/lib/docker/volumes/postgres-data/_data", Destination: "/var/lib/postgresql/data", RW: true}}, Labels: map[string]string{"com.docker.compose.project": "storefront", "com.docker.compose.service": "db"}, Project: "storefront", Service: "db", ProjectDir: "/opt/storefront", ProjectConfigFiles: []string{"/opt/storefront/docker-compose.yml"}},
		{ID: "redis1", Name: "redis-1", ImageID: e.images[2].ID, Image: e.images[2].Name(), State: "exited", Status: "Exited (0)", Created: now.Add(-48 * time.Hour), Ports: []string{"6379/tcp"}, Mounts: []engine.Mount{{Type: "volume", Name: "redis-data", Source: "/var/lib/docker/volumes/redis-data/_data", Destination: "/data", RW: true}}, Labels: map[string]string{"com.docker.compose.project": "storefront", "com.docker.compose.service": "cache"}, Project: "storefront", Service: "cache", ProjectDir: "/opt/storefront", ProjectConfigFiles: []string{"/opt/storefront/docker-compose.yml"}},
	}
	return e
}

// ListImages returns a copy so callers cannot mutate fixture state.
func (e *Engine) ListImages(ctx context.Context) ([]engine.Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	result := slices.Clone(e.images)
	for n := range result {
		result[n].RepoTags = slices.Clone(result[n].RepoTags)
		result[n].Digests = slices.Clone(result[n].Digests)
		result[n].UsedBy = nil
	}
	return result, nil
}

// ListContainers returns a detached copy of the fixture containers.
func (e *Engine) ListContainers(ctx context.Context) ([]engine.Container, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	result := make([]engine.Container, len(e.containers))
	copy(result, e.containers)
	for n := range result {
		result[n].Labels = maps.Clone(result[n].Labels)
		result[n].Ports = slices.Clone(result[n].Ports)
		result[n].Mounts = slices.Clone(result[n].Mounts)
	}
	return result, nil
}

// InspectImage returns scripted metadata containing a deliberately masked secret.
func (e *Engine) InspectImage(ctx context.Context, id string) (engine.Inspection, error) {
	items, err := e.ListImages(ctx)
	if err != nil {
		return engine.Inspection{}, err
	}
	for _, i := range items {
		if i.ID == id {
			return engine.Inspection{ID: id, Platform: "linux/amd64", Env: []string{"PATH=/usr/local/bin:/usr/bin", "DEMO_TOKEN=fixture-secret"}, Entrypoint: []string{"/docker-entrypoint.sh"}, Command: []string{"server"}, Ports: []string{"80/tcp"}, Digests: []string{"fixture@" + id}, Labels: map[string]string{"org.opencontainers.image.title": i.Name()}, Layers: []engine.Layer{{ID: "base", Command: "FROM alpine", Size: i.Size / 8}, {ID: "install", Command: "RUN install application", Size: i.Size * 7 / 8}, {ID: "config", Command: "CMD server"}}}, nil
		}
	}
	return engine.Inspection{}, fmt.Errorf("image not found")
}

// RemoveImage refuses images still referenced by a container.
func (e *Engine) RemoveImage(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	e.mu.Lock()
	for _, c := range e.containers {
		if c.ImageID == id {
			e.mu.Unlock()
			return fmt.Errorf("image is in use by container %s", c.Name)
		}
	}
	idx := slices.IndexFunc(e.images, func(i engine.Image) bool { return i.ID == id })
	if idx < 0 {
		e.mu.Unlock()
		return fmt.Errorf("image not found")
	}
	e.images = slices.Delete(e.images, idx, idx+1)
	e.broadcastLocked(engine.Event{ID: id, Kind: "image", Action: "delete", At: time.Now()})
	e.mu.Unlock()
	return nil
}

// Pull emits a layer script and only adds an image after successful completion.
func (e *Engine) Pull(ctx context.Context, ref string, emit func(engine.Progress) error) error {
	if e.PullError != nil {
		return e.PullError
	}
	for step := 0; step <= 20; step++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if e.Delay > 0 {
			timer := time.NewTimer(e.Delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
		for n, id := range []string{"a3f1c9e2b7d4", "7c0d8e5a1f92", "e19b44c07a63"} {
			current := int64(min(20, max(0, step-n*2))) * 1_000_000
			status := "Downloading"
			if current == 20_000_000 {
				status = "Pull complete"
			}
			if err := emit(engine.Progress{ID: id, Status: status, Current: current, Total: 20_000_000}); err != nil {
				return err
			}
		}
	}
	for _, id := range []string{"a3f1c9e2b7d4", "7c0d8e5a1f92", "e19b44c07a63"} {
		if err := emit(engine.Progress{ID: id, Status: "Pull complete", Current: 20_000_000, Total: 20_000_000}); err != nil {
			return err
		}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if !slices.ContainsFunc(e.images, func(i engine.Image) bool { return i.Name() == ref }) {
		e.images = append(e.images, engine.Image{ID: fmt.Sprintf("sha256:%064x", len(e.images)+100), RepoTags: []string{ref}, Size: 60_000_000, Created: time.Now()})
	}
	e.broadcastLocked(engine.Event{Kind: "image", Action: "pull", At: time.Now()})
	return nil
}

// Events waits without idle timers and unregisters on cancellation.
func (e *Engine) Events(ctx context.Context, emit func(engine.Event) error) error {
	e.mu.Lock()
	e.next++
	id := e.next
	ch := make(chan engine.Event, 32)
	e.subs[id] = ch
	e.mu.Unlock()
	defer func() { e.mu.Lock(); delete(e.subs, id); e.mu.Unlock() }()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev := <-ch:
			if err := emit(ev); err != nil {
				return err
			}
		}
	}
}

func (e *Engine) broadcastLocked(ev engine.Event) {
	for _, ch := range e.subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

// Close releases no external resources; subscriptions follow their contexts.
func (e *Engine) Close() error { return nil }
