package services

import (
	"context"
	"fmt"
	"maps"
	"net/netip"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

var (
	containerName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)
	envName       = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	resourceName  = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)
)

// InspectContainer returns a detached, conservatively masked structured view.
func (s *Service) InspectContainer(ctx context.Context, id string) (engine.ContainerInspection, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	i, err := s.engine.InspectContainer(ctx, id)
	if err != nil {
		return i, engine.Classify(err)
	}
	i.Env = slices.Clone(i.Env)
	i.Labels = maps.Clone(i.Labels)
	i.Entrypoint = slices.Clone(i.Entrypoint)
	i.Command = slices.Clone(i.Command)
	i.Ports = slices.Clone(i.Ports)
	i.Networks = slices.Clone(i.Networks)
	i.Mounts = slices.Clone(i.Mounts)
	for n, v := range i.Env {
		key, _, _ := strings.Cut(v, "=")
		i.Env[n] = key + "=••••"
	}
	for key := range i.Labels {
		i.Labels[key] = "••••"
	}
	i.Entrypoint = maskedArguments(i.Entrypoint)
	i.Command = maskedArguments(i.Command)
	slices.Sort(i.Ports)
	slices.Sort(i.Networks)
	return i, nil
}

// ContainerStats returns one bounded, measured resource sample.
func (s *Service) ContainerStats(ctx context.Context, id string) (engine.ContainerStats, error) {
	if id == "" {
		return engine.ContainerStats{}, fmt.Errorf("container ID is required")
	}
	ctx, cancel := context.WithTimeout(ctx, statsTimeout)
	defer cancel()
	stats, err := s.engine.ContainerStats(ctx, id)
	if err != nil {
		return engine.ContainerStats{}, engine.Classify(err)
	}
	return stats, nil
}

// ActContainers performs safe lifecycle actions with a deadline per resource.
// Removal never forces a running container or deletes its volumes.
func (s *Service) ActContainers(ctx context.Context, ids []string, action engine.ContainerAction) map[string]error {
	result := make(map[string]error, len(ids))
	for _, id := range ids {
		result[id] = s.actContainer(ctx, id, action)
	}
	return result
}

func (s *Service) actContainer(ctx context.Context, id string, action engine.ContainerAction) error {
	if id == "" {
		return fmt.Errorf("container ID is required")
	}
	switch action {
	case engine.ActionStart, engine.ActionStop, engine.ActionRestart,
		engine.ActionPause, engine.ActionUnpause, engine.ActionKill, engine.ActionRemove:
	default:
		return fmt.Errorf("unsupported container action %q", action)
	}
	if err := ctx.Err(); err != nil {
		return engine.Classify(err)
	}
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()
	if action == engine.ActionRemove {
		containers, err := s.engine.ListContainers(ctx)
		if err != nil {
			return engine.Classify(err)
		}
		n := slices.IndexFunc(containers, func(c engine.Container) bool { return c.ID == id })
		if n >= 0 && containers[n].Running() {
			return fmt.Errorf("container is running; stop it before removing")
		}
	}
	if err := s.engine.ActContainer(ctx, id, action); err != nil {
		return engine.Classify(err)
	}
	return nil
}

// Run creates and starts one container from a validated image and spec.
func (s *Service) Run(ctx context.Context, spec engine.RunSpec) (string, error) {
	spec, err := NormalizeRunSpec(spec)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()
	id, err := s.engine.CreateContainer(ctx, spec)
	if err != nil {
		return "", engine.Classify(err)
	}
	if err := s.engine.ActContainer(ctx, id, engine.ActionStart); err != nil {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		removeErr := s.engine.ActContainer(cleanup, id, engine.ActionRemove)
		stop()
		if removeErr == nil {
			return "", fmt.Errorf("start new container: %w; the unstarted container was removed", engine.Classify(err))
		}
		return id, fmt.Errorf("start new container: %w", engine.Classify(err))
	}
	return id, nil
}

// NormalizeRunSpec validates the wizard form and normalizes the image reference.
func NormalizeRunSpec(spec engine.RunSpec) (engine.RunSpec, error) {
	spec.Name = strings.TrimSpace(spec.Name)
	spec.Image = strings.TrimSpace(spec.Image)
	spec.Network = strings.TrimSpace(spec.Network)
	spec.Restart = strings.TrimSpace(spec.Restart)
	if spec.Name != "" && !containerName.MatchString(spec.Name) {
		return spec, fmt.Errorf("name must start with a letter or digit and contain only letters, digits, _ . or -")
	}
	image, err := NormalizeReference(spec.Image)
	if err != nil {
		return spec, err
	}
	spec.Image = image
	if spec.Network != "" && !resourceName.MatchString(spec.Network) {
		return spec, fmt.Errorf("network must be a name containing only letters, digits, _ . or -")
	}
	switch spec.Restart {
	case "", "no", "always", "unless-stopped", "on-failure":
	default:
		return spec, fmt.Errorf("restart policy must be no, always, unless-stopped or on-failure")
	}
	spec.Command = slices.Clone(spec.Command)
	spec.Ports = slices.Clone(spec.Ports)
	seenPort := map[string]bool{}
	for n, value := range spec.Ports {
		value = strings.TrimSpace(value)
		if err := validatePort(value); err != nil {
			return spec, err
		}
		containerPort := strings.ToLower(value[strings.LastIndex(value, ":")+1:])
		if seenPort[containerPort] {
			return spec, fmt.Errorf("container port %s is listed more than once", containerPort)
		}
		seenPort[containerPort] = true
		spec.Ports[n] = value
	}
	spec.Env = slices.Clone(spec.Env)
	for _, value := range spec.Env {
		if strings.ContainsAny(value, "\x00\n\r") {
			return spec, fmt.Errorf("environment values cannot contain control characters")
		}
		key, _, ok := strings.Cut(value, "=")
		if !ok || !envName.MatchString(key) {
			return spec, fmt.Errorf("environment must use KEY=value with an uppercase, lowercase or underscore key")
		}
	}
	spec.Volumes = slices.Clone(spec.Volumes)
	seenTarget := map[string]bool{}
	for n, value := range spec.Volumes {
		value = strings.TrimSpace(value)
		target, err := validateMount(value, seenTarget)
		if err != nil {
			return spec, err
		}
		seenTarget[target] = true
		spec.Volumes[n] = value
	}
	return spec, nil
}

func validatePort(value string) error {
	parts := strings.Split(value, ":")
	if len(parts) < 1 || len(parts) > 3 {
		return fmt.Errorf("invalid port %q; use [host:]container[/tcp|udp]", value)
	}
	containerPort := parts[len(parts)-1]
	number, proto, ok := strings.Cut(containerPort, "/")
	if !ok {
		proto = "tcp"
	}
	if proto != "tcp" && proto != "udp" && proto != "sctp" {
		return fmt.Errorf("invalid protocol in port %q", value)
	}
	if n, err := strconv.Atoi(number); err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("invalid container port in %q", value)
	}
	if len(parts) >= 2 && parts[len(parts)-2] != "" {
		if n, err := strconv.Atoi(parts[len(parts)-2]); err != nil || n < 0 || n > 65535 {
			return fmt.Errorf("invalid host port in %q", value)
		}
	}
	if len(parts) == 3 && parts[0] != "" {
		if _, err := netip.ParseAddr(parts[0]); err != nil {
			return fmt.Errorf("invalid host IP in %q", value)
		}
	}
	return nil
}

func validateMount(value string, seen map[string]bool) (string, error) {
	parts := strings.SplitN(value, ":", 3)
	if len(parts) < 2 || parts[0] == "" || !strings.HasPrefix(parts[1], "/") {
		return "", fmt.Errorf("invalid mount %q; use source:/absolute/target[:ro]", value)
	}
	if seen[parts[1]] {
		return "", fmt.Errorf("mount target %s is listed more than once", parts[1])
	}
	if len(parts) == 3 && parts[2] != "" && parts[2] != "ro" && parts[2] != "rw" {
		return "", fmt.Errorf("invalid mount mode %q", parts[2])
	}
	if strings.HasPrefix(parts[0], "/") {
		return parts[1], nil
	}
	if strings.HasPrefix(parts[0], ".") || !resourceName.MatchString(parts[0]) {
		return "", fmt.Errorf("use an absolute host path or a named volume in %q", value)
	}
	return parts[1], nil
}

// RunCommand renders the equivalent detached docker run command.
func RunCommand(spec engine.RunSpec) string {
	parts := []string{"docker", "run", "-d"}
	if spec.Name != "" {
		parts = append(parts, "--name", shellQuote(spec.Name))
	}
	for _, value := range spec.Ports {
		parts = append(parts, "-p", shellQuote(value))
	}
	for _, value := range spec.Env {
		parts = append(parts, "-e", shellQuote(value))
	}
	for _, value := range spec.Volumes {
		parts = append(parts, "-v", shellQuote(value))
	}
	if spec.Restart != "" && spec.Restart != "no" {
		parts = append(parts, "--restart", shellQuote(spec.Restart))
	}
	if spec.Network != "" {
		parts = append(parts, "--network", shellQuote(spec.Network))
	}
	parts = append(parts, shellQuote(spec.Image))
	for _, arg := range spec.Command {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	if value != "" && strings.IndexFunc(value, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("_./:@%+=,-", r)
	}) < 0 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

// ParseCommandLine splits a command into arguments with basic quote handling.
func ParseCommandLine(value string) ([]string, error) {
	var args []string
	var current strings.Builder
	var single, double, escaped, started bool
	for _, r := range value {
		if escaped {
			current.WriteRune(r)
			escaped, started = false, true
			continue
		}
		if r == '\\' && !single {
			escaped, started = true, true
			continue
		}
		if single {
			if r == '\'' {
				single = false
			} else {
				current.WriteRune(r)
			}
			continue
		}
		if double {
			if r == '"' {
				double = false
			} else {
				current.WriteRune(r)
			}
			continue
		}
		switch {
		case r == '\'':
			single, started = true, true
		case r == '"':
			double, started = true, true
		case unicode.IsSpace(r):
			if started {
				args = append(args, current.String())
				current.Reset()
				started = false
			}
		default:
			current.WriteRune(r)
			started = true
		}
	}
	if escaped {
		return nil, fmt.Errorf("command ends with an unfinished escape")
	}
	if single || double {
		return nil, fmt.Errorf("command has an unfinished quote")
	}
	if started {
		args = append(args, current.String())
	}
	return args, nil
}
