package docker

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// ListNetworks retrieves all Docker networks including bridge, host, and user-defined overlays.
func (e *Engine) ListNetworks(ctx context.Context) ([]engine.Network, error) {
	result, err := e.client.NetworkList(ctx, client.NetworkListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list networks: %w", err)
	}
	items := make([]engine.Network, 0, len(result.Items))
	for _, n := range result.Items {
		items = append(items, convertNetworkSummary(n))
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

// InspectNetwork retrieves detailed network configuration, IPAM pools, and connected containers.
func (e *Engine) InspectNetwork(ctx context.Context, id string) (engine.Network, error) {
	result, err := e.client.NetworkInspect(ctx, id, client.NetworkInspectOptions{Verbose: true})
	if err != nil {
		return engine.Network{}, fmt.Errorf("inspect network: %w", err)
	}
	return convertNetworkInspect(result.Network), nil
}

// CreateNetwork provisions a new bridge or custom driver network.
func (e *Engine) CreateNetwork(ctx context.Context, name, driver string) (string, error) {
	if driver == "" {
		driver = "bridge"
	}
	result, err := e.client.NetworkCreate(ctx, name, client.NetworkCreateOptions{
		Driver: driver,
	})
	if err != nil {
		return "", fmt.Errorf("create network: %w", err)
	}
	return result.ID, nil
}

// RemoveNetwork removes an unused network.
func (e *Engine) RemoveNetwork(ctx context.Context, id string) error {
	if _, err := e.client.NetworkRemove(ctx, id, client.NetworkRemoveOptions{}); err != nil {
		return fmt.Errorf("remove network: %w", err)
	}
	return nil
}

// ConnectNetwork attaches a container to a network.
func (e *Engine) ConnectNetwork(ctx context.Context, netID, containerID string) error {
	if _, err := e.client.NetworkConnect(ctx, netID, client.NetworkConnectOptions{
		Container: containerID,
	}); err != nil {
		return fmt.Errorf("connect network: %w", err)
	}
	return nil
}

// DisconnectNetwork detaches a container from a network.
func (e *Engine) DisconnectNetwork(ctx context.Context, netID, containerID string, force bool) error {
	if _, err := e.client.NetworkDisconnect(ctx, netID, client.NetworkDisconnectOptions{
		Container: containerID,
		Force:     force,
	}); err != nil {
		return fmt.Errorf("disconnect network: %w", err)
	}
	return nil
}

func convertNetworkSummary(n network.Summary) engine.Network {
	subnets := make([]string, 0, len(n.IPAM.Config))
	gateways := make([]string, 0, len(n.IPAM.Config))
	for _, cfg := range n.IPAM.Config {
		if cfg.Subnet.IsValid() {
			subnets = append(subnets, cfg.Subnet.String())
		}
		if cfg.Gateway.IsValid() {
			gateways = append(gateways, cfg.Gateway.String())
		}
	}
	return engine.Network{
		ID:         n.ID,
		Name:       n.Name,
		Driver:     n.Driver,
		Scope:      n.Scope,
		Internal:   n.Internal,
		Attachable: n.Attachable,
		Ingress:    n.Ingress,
		IPv6:       n.EnableIPv6,
		Subnets:    subnets,
		Gateways:   gateways,
		Labels:     maps.Clone(n.Labels),
		Created:    n.Created,
	}
}

func convertNetworkInspect(n network.Inspect) engine.Network {
	subnets := make([]string, 0, len(n.IPAM.Config))
	gateways := make([]string, 0, len(n.IPAM.Config))
	for _, cfg := range n.IPAM.Config {
		if cfg.Subnet.IsValid() {
			subnets = append(subnets, cfg.Subnet.String())
		}
		if cfg.Gateway.IsValid() {
			gateways = append(gateways, cfg.Gateway.String())
		}
	}
	endpoints := make([]engine.NetworkEndpoint, 0, len(n.Containers))
	for cid, ep := range n.Containers {
		var ipv4, ipv6 string
		if ep.IPv4Address.IsValid() {
			ipv4 = ep.IPv4Address.String()
		}
		if ep.IPv6Address.IsValid() {
			ipv6 = ep.IPv6Address.String()
		}
		endpoints = append(endpoints, engine.NetworkEndpoint{
			ContainerID: cid,
			Name:        strings.TrimPrefix(ep.Name, "/"),
			IPv4:        ipv4,
			IPv6:        ipv6,
			MacAddress:  ep.MacAddress.String(),
		})
	}
	slices.SortFunc(endpoints, func(a, b engine.NetworkEndpoint) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return engine.Network{
		ID:         n.ID,
		Name:       n.Name,
		Driver:     n.Driver,
		Scope:      n.Scope,
		Internal:   n.Internal,
		Attachable: n.Attachable,
		Ingress:    n.Ingress,
		IPv6:       n.EnableIPv6,
		Subnets:    subnets,
		Gateways:   gateways,
		Containers: endpoints,
		Labels:     maps.Clone(n.Labels),
		Created:    n.Created,
	}
}
