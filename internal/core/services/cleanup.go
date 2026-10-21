package services

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// CleanupKind is one category in the smart-cleanup checklist.
type CleanupKind string

const (
	CleanupDanglingImages    CleanupKind = "dangling-images"
	CleanupUnusedImages      CleanupKind = "unused-images"
	CleanupStoppedContainers CleanupKind = "stopped-containers"
	CleanupUnusedVolumes     CleanupKind = "unused-volumes"
	CleanupBuildCache        CleanupKind = "build-cache"
)

// CleanupKinds returns the canonical checklist order.
func CleanupKinds() []CleanupKind {
	return []CleanupKind{CleanupDanglingImages, CleanupUnusedImages, CleanupStoppedContainers, CleanupUnusedVolumes, CleanupBuildCache}
}

// CleanupItem is one exact resource or aggregate category shown in the preview.
type CleanupItem struct {
	Kind     CleanupKind
	ID, Name string
	Detail   string
	Size     int64
}

// CleanupGroup summarizes one checklist category.
type CleanupGroup struct {
	Kind  CleanupKind
	Count int
	Size  int64
}

// CleanupPlan is the preview shown before any mutation is allowed.
type CleanupPlan struct {
	Items  []CleanupItem
	Groups []CleanupGroup
	Total  int64
}

// CleanupResult reports successful removals and per-resource failures.
type CleanupResult struct {
	Removed   int
	Reclaimed int64
	Estimated bool
	Errors    map[string]error
}

// CleanupPreview lists exact candidates without mutating daemon state.
func (s *Service) CleanupPreview(ctx context.Context) (CleanupPlan, error) {
	ctx, cancel := context.WithTimeout(ctx, usageTimeout)
	defer cancel()
	var images []engine.Image
	var containers []engine.Container
	var volumes []engine.Volume
	var usage engine.SystemUsage
	var imageErr, containerErr, volumeErr, usageErr error
	var wg sync.WaitGroup
	wg.Add(4)
	go func() { defer wg.Done(); images, imageErr = s.engine.ListImages(ctx) }()
	go func() { defer wg.Done(); containers, containerErr = s.engine.ListContainers(ctx) }()
	go func() { defer wg.Done(); volumes, volumeErr = s.engine.ListVolumes(ctx) }()
	go func() { defer wg.Done(); usage, usageErr = s.engine.SystemUsage(ctx) }()
	wg.Wait()
	for _, err := range []error{imageErr, containerErr, volumeErr, usageErr} {
		if err != nil {
			return CleanupPlan{}, engine.Classify(err)
		}
	}
	used := map[string]bool{}
	for _, c := range containers {
		used[c.ImageID] = true
	}
	plan := CleanupPlan{}
	for _, kind := range CleanupKinds() {
		plan.Groups = append(plan.Groups, CleanupGroup{Kind: kind})
	}
	for _, image := range images {
		if used[image.ID] {
			continue
		}
		kind := CleanupUnusedImages
		if image.Dangling {
			kind = CleanupDanglingImages
		}
		plan.Items = append(plan.Items, CleanupItem{Kind: kind, ID: image.ID, Name: image.Name(), Detail: "no container references it", Size: image.Size})
	}
	for _, container := range containers {
		if container.State != "created" && container.State != "exited" {
			continue
		}
		plan.Items = append(plan.Items, CleanupItem{Kind: CleanupStoppedContainers, ID: container.ID, Name: container.Name, Detail: container.Status})
	}
	for _, volume := range volumes {
		if volume.RefCount != 0 {
			continue
		}
		detail := "no container mounts it"
		if volume.Size < 0 {
			detail += "; size unavailable"
		}
		plan.Items = append(plan.Items, CleanupItem{Kind: CleanupUnusedVolumes, ID: volume.Name, Name: volume.Name, Detail: detail, Size: max(0, volume.Size)})
	}
	if usage.BuildCache.Reclaimable > 0 {
		plan.Items = append(plan.Items, CleanupItem{Kind: CleanupBuildCache, ID: "build-cache", Name: "Build cache", Detail: "all unused build cache", Size: usage.BuildCache.Reclaimable})
	}
	slices.SortFunc(plan.Items, func(a, b CleanupItem) int {
		if a.Kind != b.Kind {
			return slices.Index(CleanupKinds(), a.Kind) - slices.Index(CleanupKinds(), b.Kind)
		}
		return strings.Compare(a.Name, b.Name)
	})
	for _, item := range plan.Items {
		n := slices.IndexFunc(plan.Groups, func(g CleanupGroup) bool { return g.Kind == item.Kind })
		if n >= 0 {
			plan.Groups[n].Count++
			plan.Groups[n].Size += item.Size
			plan.Total += item.Size
		}
	}
	return plan, nil
}

// Cleanup revalidates every candidate before its non-forced deletion. Volumes
// require the explicit typed word because their contents cannot be recovered.
func (s *Service) Cleanup(ctx context.Context, kinds []CleanupKind, confirmation string) (CleanupResult, error) {
	selected := map[CleanupKind]bool{}
	for _, kind := range kinds {
		if !slices.Contains(CleanupKinds(), kind) {
			return CleanupResult{}, fmt.Errorf("unknown cleanup category %q", kind)
		}
		selected[kind] = true
	}
	if len(selected) == 0 {
		return CleanupResult{}, fmt.Errorf("select at least one cleanup category")
	}
	if selected[CleanupUnusedVolumes] && strings.TrimSpace(confirmation) != "volumes" {
		return CleanupResult{}, fmt.Errorf(`type "volumes" to confirm deleting unused volumes`)
	}
	plan, err := s.CleanupPreview(ctx)
	if err != nil {
		return CleanupResult{}, err
	}
	result := CleanupResult{Errors: map[string]error{}}
	for _, kind := range CleanupKinds() {
		if !selected[kind] {
			continue
		}
		items := selectedItems(plan.Items, kind)
		switch kind {
		case CleanupDanglingImages, CleanupUnusedImages:
			for _, item := range items {
				if err := s.removeImage(ctx, item.ID); err != nil {
					result.Errors[item.ID] = err
					continue
				}
				result.Removed++
				result.Reclaimed += item.Size
				result.Estimated = true
			}
		case CleanupStoppedContainers:
			s.cleanupContainers(ctx, items, &result)
		case CleanupUnusedVolumes:
			s.cleanupVolumes(ctx, items, &result)
		case CleanupBuildCache:
			cacheCtx, stop := context.WithTimeout(ctx, actionTimeout)
			reclaimed, err := s.engine.PruneBuildCache(cacheCtx)
			stop()
			if err != nil {
				result.Errors["build-cache"] = engine.Classify(err)
				continue
			}
			result.Removed += len(items)
			result.Reclaimed += reclaimed
		}
	}
	if len(result.Errors) == 0 {
		result.Errors = nil
	}
	return result, nil
}

func (s *Service) cleanupContainers(ctx context.Context, items []CleanupItem, result *CleanupResult) {
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()
	containers, err := s.engine.ListContainers(ctx)
	if err != nil {
		result.Errors["containers"] = engine.Classify(err)
		return
	}
	for _, item := range items {
		n := slices.IndexFunc(containers, func(c engine.Container) bool { return c.ID == item.ID })
		if n < 0 || (containers[n].State != "created" && containers[n].State != "exited") {
			result.Errors[item.ID] = fmt.Errorf("container is no longer stopped; skipped")
			continue
		}
		itemCtx, stop := context.WithTimeout(ctx, actionTimeout)
		err := s.engine.ActContainer(itemCtx, item.ID, engine.ActionRemove)
		stop()
		if err != nil {
			result.Errors[item.ID] = engine.Classify(err)
			continue
		}
		result.Removed++
	}
}

func (s *Service) cleanupVolumes(ctx context.Context, items []CleanupItem, result *CleanupResult) {
	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()
	volumes, err := s.engine.ListVolumes(ctx)
	if err != nil {
		result.Errors["volumes"] = engine.Classify(err)
		return
	}
	for _, item := range items {
		n := slices.IndexFunc(volumes, func(v engine.Volume) bool { return v.Name == item.ID })
		if n < 0 || volumes[n].RefCount != 0 {
			result.Errors[item.ID] = fmt.Errorf("volume is no longer unused; skipped")
			continue
		}
		itemCtx, stop := context.WithTimeout(ctx, actionTimeout)
		err := s.engine.RemoveVolume(itemCtx, item.ID)
		stop()
		if err != nil {
			result.Errors[item.ID] = engine.Classify(err)
			continue
		}
		result.Removed++
		result.Reclaimed += item.Size
	}
}

func selectedItems(items []CleanupItem, kind CleanupKind) []CleanupItem {
	var out []CleanupItem
	for _, item := range items {
		if item.Kind == kind {
			out = append(out, item)
		}
	}
	return out
}
