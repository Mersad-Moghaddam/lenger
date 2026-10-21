package fake

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// ListVolumes returns detached fixture volume summaries.
func (e *Engine) ListVolumes(ctx context.Context) ([]engine.Volume, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	result := make([]engine.Volume, len(e.volumes))
	copy(result, e.volumes)
	for n := range result {
		result[n].Labels = maps.Clone(result[n].Labels)
		result[n].UsedBy = nil
	}
	return result, nil
}

// RemoveVolume refuses a volume that a container still mounts.
func (e *Engine) RemoveVolume(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.refreshVolumeRefsLocked()
	n := slices.IndexFunc(e.volumes, func(v engine.Volume) bool { return v.Name == name })
	if n < 0 {
		return fmt.Errorf("volume not found")
	}
	if e.volumes[n].RefCount > 0 {
		return fmt.Errorf("volume is in use by a container")
	}
	e.volumes = slices.Delete(e.volumes, n, n+1)
	e.broadcastLocked(engine.Event{ID: name, Kind: "volume", Action: "destroy", At: time.Now()})
	return nil
}

// SystemUsage derives deterministic usage totals from fixture resources.
func (e *Engine) SystemUsage(ctx context.Context) (engine.SystemUsage, error) {
	if err := ctx.Err(); err != nil {
		return engine.SystemUsage{}, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.refreshVolumeRefsLocked()
	usedImages := map[string]bool{}
	activeContainers := 0
	for _, c := range e.containers {
		usedImages[c.ImageID] = true
		if c.Running() {
			activeContainers++
		}
	}
	var out engine.SystemUsage
	out.Images.Total = len(e.images)
	for _, i := range e.images {
		out.Images.Size += i.Size
		if usedImages[i.ID] {
			out.Images.Active++
		} else {
			out.Images.Reclaimable += i.Size
		}
	}
	out.Containers.Total = len(e.containers)
	out.Containers.Active = activeContainers
	out.Volumes.Total = len(e.volumes)
	for _, v := range e.volumes {
		out.Volumes.Size += max(0, v.Size)
		if v.RefCount > 0 {
			out.Volumes.Active++
		} else {
			out.Volumes.Reclaimable += max(0, v.Size)
		}
	}
	if e.buildCache > 0 {
		out.BuildCache.Total = 1
		out.BuildCache.Size = e.buildCache
		out.BuildCache.Reclaimable = e.buildCache
	}
	out.Used = out.Images.Size + out.Containers.Size + out.Volumes.Size + out.BuildCache.Size
	out.Reclaimable = out.Images.Reclaimable + out.Containers.Reclaimable + out.Volumes.Reclaimable + out.BuildCache.Reclaimable
	return out, nil
}

// PruneBuildCache clears the fixture cache and reports reclaimed bytes.
func (e *Engine) PruneBuildCache(ctx context.Context) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	reclaimed := e.buildCache
	e.buildCache = 0
	e.broadcastLocked(engine.Event{Kind: "build-cache", Action: "prune", At: time.Now()})
	return reclaimed, nil
}
