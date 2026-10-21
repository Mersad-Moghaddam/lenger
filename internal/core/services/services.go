// Package services is the application-facing facade for engine operations.
package services

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/distribution/reference"

	"github.com/Mersad-Moghaddam/lenger/internal/core/store"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// Service owns application policy, leaving SDK types inside adapters.
type Service struct{ engine engine.Engine }

const (
	operationTimeout = 10 * time.Second
	actionTimeout    = 30 * time.Second
	runTimeout       = 60 * time.Second
	usageTimeout     = 30 * time.Second
	statsTimeout     = 5 * time.Second
)

// New wraps a real or fake engine with the same policy.
func New(e engine.Engine) *Service { return &Service{engine: e} }

// Snapshot fetches independent resource lists concurrently under a deadline.
func (s *Service) Snapshot(ctx context.Context) (store.Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	var result store.Snapshot
	var imageErr, containerErr, volumeErr, networkErr error
	var wg sync.WaitGroup
	wg.Add(4)
	go func() { defer wg.Done(); result.Images, imageErr = s.engine.ListImages(ctx) }()
	go func() { defer wg.Done(); result.Containers, containerErr = s.engine.ListContainers(ctx) }()
	go func() { defer wg.Done(); result.Volumes, volumeErr = s.engine.ListVolumes(ctx) }()
	go func() { defer wg.Done(); result.Networks, networkErr = s.engine.ListNetworks(ctx) }()
	wg.Wait()
	if imageErr != nil {
		return result, engine.Classify(imageErr)
	}
	if containerErr != nil {
		return result, engine.Classify(containerErr)
	}
	if volumeErr != nil {
		return result, engine.Classify(volumeErr)
	}
	if networkErr != nil {
		return result, engine.Classify(networkErr)
	}
	fillVolumeUsers(result.Volumes, result.Containers)
	return result, nil
}

// DaemonInfo fetches host and engine telemetry under a deadline (SYS-03).
func (s *Service) DaemonInfo(ctx context.Context) (engine.DaemonInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	info, err := s.engine.DaemonInfo(ctx)
	if err != nil {
		return engine.DaemonInfo{}, engine.Classify(err)
	}
	return info, nil
}

// Inspect always masks environment values, label values, commands and history.
// Image history and entrypoints may contain build-time secrets as well as env.
func (s *Service) Inspect(ctx context.Context, id string) (engine.Inspection, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	i, err := s.engine.InspectImage(ctx, id)
	if err != nil {
		return i, engine.Classify(err)
	}
	// Mask detached copies: an adapter may retain the original metadata.
	i.Env = slices.Clone(i.Env)
	i.Labels = maps.Clone(i.Labels)
	i.Layers = slices.Clone(i.Layers)
	i.Ports = slices.Clone(i.Ports)
	i.Digests = slices.Clone(i.Digests)
	for n, v := range i.Env {
		key, _, _ := strings.Cut(v, "=")
		i.Env[n] = key + "=••••"
	}
	for key := range i.Labels {
		i.Labels[key] = "••••"
	}
	for n := range i.Layers {
		i.Layers[n].Command = "[build command hidden; may contain secrets]"
	}
	i.Entrypoint = maskedArguments(i.Entrypoint)
	i.Command = maskedArguments(i.Command)
	sort.Strings(i.Ports)
	return i, nil
}

func maskedArguments(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	return []string{"[configured; arguments hidden]"}
}

// Remove rechecks references immediately before each non-forced deletion.
func (s *Service) Remove(ctx context.Context, ids []string) map[string]error {
	result := make(map[string]error, len(ids))
	for _, id := range ids {
		result[id] = s.removeImage(ctx, id)
	}
	return result
}

func (s *Service) removeImage(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return engine.Classify(err)
	}
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	containers, err := s.engine.ListContainers(ctx)
	if err != nil {
		return engine.Classify(err)
	}
	for _, c := range containers {
		if c.ImageID == id {
			return fmt.Errorf("image is still used by a container; remove that container first")
		}
	}
	if err := ctx.Err(); err != nil {
		return engine.Classify(err)
	}
	// The daemon must still enforce references: another client can create a
	// container between the recheck and this non-forced deletion.
	if err := s.engine.RemoveImage(ctx, id); err != nil {
		return engine.Classify(err)
	}
	return nil
}

// NormalizeReference validates and adds the conventional latest tag.
func NormalizeReference(input string) (string, error) {
	if strings.TrimSpace(input) != input || input == "" {
		return "", fmt.Errorf("enter a valid image reference without surrounding whitespace")
	}
	named, err := reference.ParseNormalizedNamed(input)
	if err != nil {
		return "", fmt.Errorf("invalid image reference: %w", err)
	}
	return reference.FamiliarString(reference.TagNameOnly(named)), nil
}

// Pull delegates a validated reference to the engine.
func (s *Service) Pull(ctx context.Context, ref string, emit func(engine.Progress) error) error {
	ref, err := NormalizeReference(ref)
	if err != nil {
		return err
	}
	return s.engine.Pull(ctx, ref, emit)
}

// Events supplies daemon invalidations without exposing the SDK.
func (s *Service) Events(ctx context.Context, emit func(engine.Event) error) error {
	return s.engine.Events(ctx, emit)
}

// Deferred describes a capability outside the initial vertical slice.
func Deferred(requirement string) error { return engine.ErrNotImplemented{Requirement: requirement} } // TODO(M1): Add container, project, registry and cleanup services.
