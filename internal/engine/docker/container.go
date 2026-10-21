package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// InspectContainer returns structured metadata without retaining the raw JSON.
func (e *Engine) InspectContainer(ctx context.Context, id string) (engine.ContainerInspection, error) {
	result, err := e.client.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return engine.ContainerInspection{}, fmt.Errorf("inspect container: %w", err)
	}
	c := result.Container
	out := engine.ContainerInspection{
		ID: c.ID, Name: strings.TrimPrefix(c.Name, "/"), ImageID: c.Image,
		Platform: c.Platform, Created: parseTime(c.Created),
		RestartCount: c.RestartCount, Mounts: mountList(c.Mounts),
	}
	if c.Config != nil {
		out.Image = c.Config.Image
		out.Env = slices.Clone(c.Config.Env)
		out.Entrypoint = slices.Clone(c.Config.Entrypoint)
		out.Command = slices.Clone(c.Config.Cmd)
		out.Labels = maps.Clone(c.Config.Labels)
		out.TTY = c.Config.Tty
		for port := range c.Config.ExposedPorts {
			out.Ports = append(out.Ports, port.String())
		}
	}
	if c.State != nil {
		out.State = string(c.State.Status)
		out.Status = string(c.State.Status)
		out.ExitCode = c.State.ExitCode
		out.OOMKilled = c.State.OOMKilled
		out.StartedAt = parseTime(c.State.StartedAt)
		out.FinishedAt = parseTime(c.State.FinishedAt)
		if c.State.Health != nil {
			out.Health = string(c.State.Health.Status)
			probe := &engine.HealthProbe{
				Status:        string(c.State.Health.Status),
				FailingStreak: c.State.Health.FailingStreak,
			}
			if len(c.State.Health.Log) > 0 {
				last := c.State.Health.Log[len(c.State.Health.Log)-1]
				if last != nil {
					probe.Start = last.Start
					probe.End = last.End
					probe.ExitCode = last.ExitCode
					probe.Output = last.Output
				}
			}
			out.HealthCheck = probe
		}
	}
	if c.HostConfig != nil {
		out.NetworkMode = string(c.HostConfig.NetworkMode)
		out.Restart = string(c.HostConfig.RestartPolicy.Name)
	}
	if c.NetworkSettings != nil {
		for port, bindings := range c.NetworkSettings.Ports {
			for _, binding := range bindings {
				value := formatPortBinding(port, binding)
				if !slices.Contains(out.Ports, value) {
					out.Ports = append(out.Ports, value)
				}
			}
		}
		for name := range c.NetworkSettings.Networks {
			out.Networks = append(out.Networks, name)
		}
	}
	slices.Sort(out.Ports)
	slices.Sort(out.Networks)
	return out, nil
}

// ContainerStats returns one immediate sample. The caller computes CPU percent
// from cumulative totals so the daemon never waits for a second sample.
func (e *Engine) ContainerStats(ctx context.Context, id string) (engine.ContainerStats, error) {
	result, err := e.client.ContainerStats(ctx, id, client.ContainerStatsOptions{Stream: false})
	if err != nil {
		return engine.ContainerStats{}, fmt.Errorf("read container stats: %w", err)
	}
	defer func() { _ = result.Body.Close() }()
	var response container.StatsResponse
	if err := json.NewDecoder(result.Body).Decode(&response); err != nil {
		return engine.ContainerStats{}, fmt.Errorf("decode container stats: %w", err)
	}
	out := engine.ContainerStats{
		At: response.Read, CPUTotal: response.CPUStats.CPUUsage.TotalUsage,
		SystemTotal: response.CPUStats.SystemUsage, OnlineCPUs: response.CPUStats.OnlineCPUs,
		Memory: response.MemoryStats.Usage, MemoryLimit: response.MemoryStats.Limit,
		PIDs: response.PidsStats.Current,
	}
	if cache := response.MemoryStats.Stats["inactive_file"]; cache > 0 && cache < out.Memory {
		out.Memory -= cache
	}
	for _, network := range response.Networks {
		out.NetworkRX += network.RxBytes
		out.NetworkTX += network.TxBytes
	}
	for _, entry := range response.BlkioStats.IoServiceBytesRecursive {
		switch strings.ToLower(entry.Op) {
		case "read":
			out.BlockRead += entry.Value
		case "write":
			out.BlockWrite += entry.Value
		}
	}
	return out, nil
}

// CreateContainer applies a validated run spec and returns the new container ID.
func (e *Engine) CreateContainer(ctx context.Context, spec engine.RunSpec) (string, error) {
	exposed := network.PortSet{}
	bindings := network.PortMap{}
	for _, value := range spec.Ports {
		port, binding, hasBinding, err := parsePortMapping(value)
		if err != nil {
			return "", err
		}
		if _, exists := exposed[port]; exists {
			return "", fmt.Errorf("port %s is listed more than once", port)
		}
		exposed[port] = struct{}{}
		if hasBinding {
			bindings[port] = []network.PortBinding{binding}
		}
	}
	mounts, err := parseMounts(spec.Volumes)
	if err != nil {
		return "", err
	}
	restart, err := restartPolicy(spec.Restart)
	if err != nil {
		return "", err
	}
	config := &container.Config{Image: spec.Image, Env: slices.Clone(spec.Env), ExposedPorts: exposed}
	if len(spec.Command) > 0 {
		config.Cmd = slices.Clone(spec.Command)
	}
	host := &container.HostConfig{PortBindings: bindings, Mounts: mounts, RestartPolicy: restart}
	if spec.Network != "" {
		host.NetworkMode = container.NetworkMode(spec.Network)
	}
	result, err := e.client.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config: config, HostConfig: host, Name: spec.Name,
	})
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}
	return result.ID, nil
}

// ActContainer performs one non-forced lifecycle action.
func (e *Engine) ActContainer(ctx context.Context, id string, action engine.ContainerAction) error {
	var err error
	switch action {
	case engine.ActionStart:
		_, err = e.client.ContainerStart(ctx, id, client.ContainerStartOptions{})
	case engine.ActionStop:
		_, err = e.client.ContainerStop(ctx, id, client.ContainerStopOptions{})
	case engine.ActionRestart:
		_, err = e.client.ContainerRestart(ctx, id, client.ContainerRestartOptions{})
	case engine.ActionPause:
		_, err = e.client.ContainerPause(ctx, id, client.ContainerPauseOptions{})
	case engine.ActionUnpause:
		_, err = e.client.ContainerUnpause(ctx, id, client.ContainerUnpauseOptions{})
	case engine.ActionKill:
		_, err = e.client.ContainerKill(ctx, id, client.ContainerKillOptions{Signal: "SIGKILL"})
	case engine.ActionRemove:
		_, err = e.client.ContainerRemove(ctx, id, client.ContainerRemoveOptions{})
	default:
		return fmt.Errorf("unsupported container action %q", action)
	}
	if err != nil {
		return fmt.Errorf("%s container: %w", action, err)
	}
	return nil
}

func parsePortMapping(value string) (network.Port, network.PortBinding, bool, error) {
	parts := strings.Split(value, ":")
	if len(parts) > 3 || len(parts) == 0 {
		return network.Port{}, network.PortBinding{}, false, fmt.Errorf("invalid port mapping %q; use [host:]container[/tcp|udp]", value)
	}
	containerPart := parts[len(parts)-1]
	port, err := network.ParsePort(containerPart)
	if err != nil {
		return network.Port{}, network.PortBinding{}, false, fmt.Errorf("invalid container port %q: %w", containerPart, err)
	}
	if len(parts) == 1 {
		return port, network.PortBinding{}, false, nil
	}
	hostPort := parts[len(parts)-2]
	if hostPort != "" {
		if n, err := strconv.Atoi(hostPort); err != nil || n < 0 || n > math.MaxUint16 {
			return network.Port{}, network.PortBinding{}, false, fmt.Errorf("invalid host port %q", hostPort)
		}
	}
	binding := network.PortBinding{HostPort: hostPort}
	if len(parts) == 3 && parts[0] != "" {
		ip, err := netip.ParseAddr(parts[0])
		if err != nil {
			return network.Port{}, network.PortBinding{}, false, fmt.Errorf("invalid host IP %q", parts[0])
		}
		binding.HostIP = ip
	}
	return port, binding, true, nil
}

func parseMounts(values []string) ([]mount.Mount, error) {
	result := make([]mount.Mount, 0, len(values))
	targets := map[string]bool{}
	for _, value := range values {
		parts := strings.SplitN(value, ":", 3)
		if len(parts) < 2 || strings.TrimSpace(parts[0]) == "" || !strings.HasPrefix(parts[1], "/") {
			return nil, fmt.Errorf("invalid mount %q; use source:/absolute/target[:ro]", value)
		}
		mode := ""
		if len(parts) == 3 {
			mode = parts[2]
		}
		if mode != "" && mode != "ro" && mode != "rw" {
			return nil, fmt.Errorf("invalid mount mode %q; use ro or rw", mode)
		}
		if targets[parts[1]] {
			return nil, fmt.Errorf("mount target %s is listed more than once", parts[1])
		}
		targets[parts[1]] = true
		kind := mount.TypeVolume
		if strings.HasPrefix(parts[0], "/") {
			kind = mount.TypeBind
		} else if strings.HasPrefix(parts[0], ".") {
			return nil, fmt.Errorf("use an absolute host path or a named volume in %q", value)
		}
		result = append(result, mount.Mount{Type: kind, Source: parts[0], Target: parts[1], ReadOnly: mode == "ro"})
	}
	return result, nil
}

func restartPolicy(value string) (container.RestartPolicy, error) {
	switch value {
	case "", "no":
		return container.RestartPolicy{Name: container.RestartPolicyDisabled}, nil
	case "always":
		return container.RestartPolicy{Name: container.RestartPolicyAlways}, nil
	case "unless-stopped":
		return container.RestartPolicy{Name: container.RestartPolicyUnlessStopped}, nil
	case "on-failure":
		return container.RestartPolicy{Name: container.RestartPolicyOnFailure}, nil
	default:
		return container.RestartPolicy{}, fmt.Errorf("invalid restart policy %q", value)
	}
}

func formatPortSummary(p container.PortSummary) string {
	containerPort := fmt.Sprintf("%d/%s", p.PrivatePort, p.Type)
	if p.PublicPort == 0 {
		return containerPort
	}
	host := fmt.Sprintf("%d", p.PublicPort)
	if p.IP.IsValid() && !p.IP.IsUnspecified() {
		host = p.IP.Unmap().String() + ":" + host
	}
	return host + "->" + containerPort
}

func formatPortBinding(port network.Port, binding network.PortBinding) string {
	containerPort := port.String()
	if binding.HostPort == "" {
		return containerPort
	}
	host := binding.HostPort
	if binding.HostIP.IsValid() && !binding.HostIP.IsUnspecified() {
		host = binding.HostIP.Unmap().String() + ":" + host
	}
	return host + "->" + containerPort
}

func mountList(values []container.MountPoint) []engine.Mount {
	result := make([]engine.Mount, 0, len(values))
	for _, m := range values {
		result = append(result, engine.Mount{Type: string(m.Type), Name: m.Name, Source: m.Source, Destination: m.Destination, Mode: m.Mode, RW: m.RW})
	}
	return result
}

func parseTime(value string) time.Time {
	if value == "" || strings.HasPrefix(value, "0001-") {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return t
}
