package services

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// ListVolumes returns volumes with their container references resolved.
func (s *Service) ListVolumes(ctx context.Context) ([]engine.Volume, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	var volumes []engine.Volume
	var containers []engine.Container
	var volumeErr, containerErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); volumes, volumeErr = s.engine.ListVolumes(ctx) }()
	go func() { defer wg.Done(); containers, containerErr = s.engine.ListContainers(ctx) }()
	wg.Wait()
	if volumeErr != nil {
		return nil, engine.Classify(volumeErr)
	}
	if containerErr != nil {
		return nil, engine.Classify(containerErr)
	}
	fillVolumeUsers(volumes, containers)
	return volumes, nil
}

// SystemUsage returns the daemon disk-usage breakdown under a deadline.
func (s *Service) SystemUsage(ctx context.Context) (engine.SystemUsage, error) {
	ctx, cancel := context.WithTimeout(ctx, usageTimeout)
	defer cancel()
	usage, err := s.engine.SystemUsage(ctx)
	if err != nil {
		return usage, engine.Classify(err)
	}
	return usage, nil
}

// RemoveVolume deletes one volume only after confirming it has no container
// references. The daemon's non-forced deletion remains the final guard.
func (s *Service) RemoveVolume(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("volume name is required")
	}
	volumes, err := s.ListVolumes(ctx)
	if err != nil {
		return err
	}
	n := slices.IndexFunc(volumes, func(v engine.Volume) bool { return v.Name == name })
	if n < 0 {
		return fmt.Errorf("volume not found")
	}
	if volumes[n].RefCount > 0 || len(volumes[n].UsedBy) > 0 {
		return fmt.Errorf("volume is in use by a container; remove that container first")
	}
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()
	if err := s.engine.RemoveVolume(ctx, name); err != nil {
		return engine.Classify(err)
	}
	return nil
}

func fillVolumeUsers(volumes []engine.Volume, containers []engine.Container) {
	byVolume := make(map[string][]engine.Container)
	for _, c := range containers {
		for _, m := range c.Mounts {
			if m.Type == "volume" && m.Name != "" {
				byVolume[m.Name] = append(byVolume[m.Name], c)
			}
		}
	}
	for n := range volumes {
		volumes[n].Labels = maps.Clone(volumes[n].Labels)
		volumes[n].UsedBy = slices.Clone(byVolume[volumes[n].Name])
	}
}
