package docker

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// ListVolumes merges volume summaries with disk-usage data when available.
func (e *Engine) ListVolumes(ctx context.Context) ([]engine.Volume, error) {
	result, err := e.client.VolumeList(ctx, client.VolumeListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list volumes: %w", err)
	}
	usage := map[string]volume.UsageData{}
	df, dfErr := e.client.DiskUsage(ctx, client.DiskUsageOptions{Volumes: true, Verbose: true})
	if dfErr == nil {
		for _, v := range df.Volumes.Items {
			if v.UsageData != nil {
				usage[v.Name] = *v.UsageData
			}
		}
	}
	items := make([]engine.Volume, 0, len(result.Items))
	for _, v := range result.Items {
		item := engine.Volume{
			Name: v.Name, Driver: v.Driver, Mountpoint: v.Mountpoint, Scope: v.Scope,
			CreatedAt: parseTime(v.CreatedAt), Size: -1, RefCount: -1,
			Labels: maps.Clone(v.Labels),
		}
		if data, ok := usage[v.Name]; ok {
			item.Size = data.Size
			item.RefCount = int(data.RefCount)
		}
		items = append(items, item)
	}
	slices.SortFunc(items, func(a, b engine.Volume) int {
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

// RemoveVolume performs a non-forced deletion; the daemon rejects in-use volumes.
func (e *Engine) RemoveVolume(ctx context.Context, name string) error {
	if _, err := e.client.VolumeRemove(ctx, name, client.VolumeRemoveOptions{Force: false}); err != nil {
		return fmt.Errorf("remove volume: %w", err)
	}
	return nil
}

// SystemUsage reads the daemon's disk-usage breakdown.
func (e *Engine) SystemUsage(ctx context.Context) (engine.SystemUsage, error) {
	result, err := e.client.DiskUsage(ctx, client.DiskUsageOptions{
		Containers: true, Images: true, Volumes: true, BuildCache: true,
	})
	if err != nil {
		return engine.SystemUsage{}, fmt.Errorf("read disk usage: %w", err)
	}
	return engine.SystemUsage{
		Images: engine.UsageCategory{
			Total: int(result.Images.TotalCount), Active: int(result.Images.ActiveCount),
			Size: result.Images.TotalSize, Reclaimable: result.Images.Reclaimable,
		},
		Containers: engine.UsageCategory{
			Total: int(result.Containers.TotalCount), Active: int(result.Containers.ActiveCount),
			Size: result.Containers.TotalSize, Reclaimable: result.Containers.Reclaimable,
		},
		Volumes: engine.UsageCategory{
			Total: int(result.Volumes.TotalCount), Active: int(result.Volumes.ActiveCount),
			Size: result.Volumes.TotalSize, Reclaimable: result.Volumes.Reclaimable,
		},
		BuildCache: engine.UsageCategory{
			Total: int(result.BuildCache.TotalCount), Active: int(result.BuildCache.ActiveCount),
			Size: result.BuildCache.TotalSize, Reclaimable: result.BuildCache.Reclaimable,
		},
		Used:        result.Images.TotalSize + result.Containers.TotalSize + result.Volumes.TotalSize + result.BuildCache.TotalSize,
		Reclaimable: result.Images.Reclaimable + result.Containers.Reclaimable + result.Volumes.Reclaimable + result.BuildCache.Reclaimable,
	}, nil
}

// PruneBuildCache deletes unused build cache and reports reclaimed bytes.
func (e *Engine) PruneBuildCache(ctx context.Context) (int64, error) {
	result, err := e.client.BuildCachePrune(ctx, client.BuildCachePruneOptions{All: true})
	if err != nil {
		return 0, fmt.Errorf("prune build cache: %w", err)
	}
	return int64(result.Report.SpaceReclaimed), nil
}
