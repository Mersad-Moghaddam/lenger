package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// ListNetworks retrieves all Docker networks under the standard operation timeout.
func (s *Service) ListNetworks(ctx context.Context) ([]engine.Network, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	items, err := s.engine.ListNetworks(ctx)
	if err != nil {
		return nil, engine.Classify(err)
	}
	return items, nil
}

// InspectNetwork retrieves full network details and connected container endpoints.
func (s *Service) InspectNetwork(ctx context.Context, id string) (engine.Network, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return engine.Network{}, fmt.Errorf("network ID or name is required")
	}
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	net, err := s.engine.InspectNetwork(ctx, id)
	if err != nil {
		return engine.Network{}, engine.Classify(err)
	}
	return net, nil
}

// CreateNetwork provisions a new Docker network with name and driver validation (NET-01).
func (s *Service) CreateNetwork(ctx context.Context, name, driver string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("network name cannot be empty")
	}
	driver = strings.TrimSpace(driver)
	if driver == "" {
		driver = "bridge"
	}
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()
	id, err := s.engine.CreateNetwork(ctx, name, driver)
	if err != nil {
		return "", engine.Classify(err)
	}
	return id, nil
}

// RemoveNetwork ensures a network has no attached endpoints before deletion.
func (s *Service) RemoveNetwork(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("network ID or name is required")
	}
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()

	// Guard: check if network has active endpoints
	net, err := s.engine.InspectNetwork(ctx, id)
	if err == nil {
		if len(net.Containers) > 0 {
			return fmt.Errorf("network %q still has %d active container(s) attached; disconnect them first", net.Name, len(net.Containers))
		}
	}

	if err := s.engine.RemoveNetwork(ctx, id); err != nil {
		return engine.Classify(err)
	}
	return nil
}

// ConnectNetwork attaches a container to a network.
func (s *Service) ConnectNetwork(ctx context.Context, netID, containerID string) error {
	netID = strings.TrimSpace(netID)
	containerID = strings.TrimSpace(containerID)
	if netID == "" || containerID == "" {
		return fmt.Errorf("network ID and container ID are both required")
	}
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()
	if err := s.engine.ConnectNetwork(ctx, netID, containerID); err != nil {
		return engine.Classify(err)
	}
	return nil
}

// DisconnectNetwork detaches a container from a network.
func (s *Service) DisconnectNetwork(ctx context.Context, netID, containerID string, force bool) error {
	netID = strings.TrimSpace(netID)
	containerID = strings.TrimSpace(containerID)
	if netID == "" || containerID == "" {
		return fmt.Errorf("network ID and container ID are both required")
	}
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()
	if err := s.engine.DisconnectNetwork(ctx, netID, containerID, force); err != nil {
		return engine.Classify(err)
	}
	return nil
}
