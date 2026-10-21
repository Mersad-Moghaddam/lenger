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

// ListNetworks returns a cloned list of fixture networks.
func (e *Engine) ListNetworks(ctx context.Context) ([]engine.Network, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	items := make([]engine.Network, 0, len(e.networks))
	for _, n := range e.networks {
		items = append(items, cloneNetwork(n))
	}
	slices.SortFunc(items, func(a, b engine.Network) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return items, nil
}

// InspectNetwork looks up a network by ID or name.
func (e *Engine) InspectNetwork(ctx context.Context, id string) (engine.Network, error) {
	if err := ctx.Err(); err != nil {
		return engine.Network{}, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, n := range e.networks {
		if n.ID == id || n.Name == id {
			return cloneNetwork(n), nil
		}
	}
	return engine.Network{}, fmt.Errorf("network not found: %s", id)
}

// CreateNetwork appends a new user-defined network.
func (e *Engine) CreateNetwork(ctx context.Context, name, driver string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("network name cannot be empty")
	}
	if driver == "" {
		driver = "bridge"
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, n := range e.networks {
		if n.Name == name {
			return "", fmt.Errorf("network %s already exists", name)
		}
	}
	newID := fmt.Sprintf("net-%012x", len(e.networks)+1)
	net := engine.Network{
		ID:         newID,
		Name:       name,
		Driver:     driver,
		Scope:      "local",
		Attachable: true,
		Subnets:    []string{"172.30.0.0/16"},
		Gateways:   []string{"172.30.0.1"},
		Created:    time.Now().UTC(),
	}
	e.networks = append(e.networks, net)
	e.broadcastLocked(engine.Event{Kind: "network", Action: "create", ID: newID, At: time.Now().UTC()})
	return newID, nil
}

// RemoveNetwork deletes an unused network.
func (e *Engine) RemoveNetwork(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	idx := slices.IndexFunc(e.networks, func(n engine.Network) bool { return n.ID == id || n.Name == id })
	if idx < 0 {
		return fmt.Errorf("network not found: %s", id)
	}
	if len(e.networks[idx].Containers) > 0 {
		return fmt.Errorf("network %s has active endpoints", e.networks[idx].Name)
	}
	removed := e.networks[idx]
	e.networks = slices.Delete(e.networks, idx, idx+1)
	e.broadcastLocked(engine.Event{Kind: "network", Action: "destroy", ID: removed.ID, At: time.Now().UTC()})
	return nil
}

// ConnectNetwork attaches a container endpoint to the target network.
func (e *Engine) ConnectNetwork(ctx context.Context, netID, containerID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	nIdx := slices.IndexFunc(e.networks, func(n engine.Network) bool { return n.ID == netID || n.Name == netID })
	if nIdx < 0 {
		return fmt.Errorf("network not found: %s", netID)
	}
	cIdx := slices.IndexFunc(e.containers, func(c engine.Container) bool { return c.ID == containerID || c.Name == containerID })
	if cIdx < 0 {
		return fmt.Errorf("container not found: %s", containerID)
	}
	targetNet := &e.networks[nIdx]
	container := e.containers[cIdx]
	for _, ep := range targetNet.Containers {
		if ep.ContainerID == container.ID {
			return nil
		}
	}
	ip := fmt.Sprintf("172.28.0.%d/16", len(targetNet.Containers)+10)
	targetNet.Containers = append(targetNet.Containers, engine.NetworkEndpoint{
		ContainerID: container.ID,
		Name:        container.Name,
		IPv4:        ip,
	})
	e.broadcastLocked(engine.Event{Kind: "network", Action: "connect", ID: targetNet.ID, At: time.Now().UTC()})
	return nil
}

// DisconnectNetwork removes a container endpoint from the target network.
func (e *Engine) DisconnectNetwork(ctx context.Context, netID, containerID string, force bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	nIdx := slices.IndexFunc(e.networks, func(n engine.Network) bool { return n.ID == netID || n.Name == netID })
	if nIdx < 0 {
		return fmt.Errorf("network not found: %s", netID)
	}
	targetNet := &e.networks[nIdx]
	epIdx := slices.IndexFunc(targetNet.Containers, func(ep engine.NetworkEndpoint) bool {
		return ep.ContainerID == containerID || ep.Name == containerID
	})
	if epIdx < 0 {
		if force {
			return nil
		}
		return fmt.Errorf("container %s is not connected to network %s", containerID, targetNet.Name)
	}
	targetNet.Containers = slices.Delete(targetNet.Containers, epIdx, epIdx+1)
	e.broadcastLocked(engine.Event{Kind: "network", Action: "disconnect", ID: targetNet.ID, At: time.Now().UTC()})
	return nil
}

func cloneNetwork(n engine.Network) engine.Network {
	out := n
	out.Subnets = slices.Clone(n.Subnets)
	out.Gateways = slices.Clone(n.Gateways)
	out.Containers = slices.Clone(n.Containers)
	out.Labels = maps.Clone(n.Labels)
	return out
}
