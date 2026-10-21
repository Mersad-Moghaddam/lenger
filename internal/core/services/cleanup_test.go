package services

import (
	"context"
	"testing"

	"github.com/Mersad-Moghaddam/lenger/internal/engine/fake"
)

func TestCleanupPreviewIsReadOnlyAndExact(t *testing.T) {
	ctx := context.Background()
	s := New(fake.New())
	before, err := s.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := s.CleanupPreview(ctx)
	if err != nil {
		t.Fatal(err)
	}
	groups := map[string]int{}
	for _, group := range plan.Groups {
		groups[string(group.Kind)] = group.Count
	}
	if groups[string(CleanupStoppedContainers)] != 1 || groups[string(CleanupUnusedVolumes)] != 1 || groups[string(CleanupBuildCache)] != 1 {
		t.Fatalf("unexpected preview groups: %#v", groups)
	}
	if plan.Total <= 0 {
		t.Fatal("preview did not estimate reclaimable space")
	}
	after, err := s.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Containers) != len(before.Containers) || len(after.Volumes) != len(before.Volumes) {
		t.Fatal("preview mutated daemon resources")
	}
}

func TestCleanupVolumesRequiresTypedWord(t *testing.T) {
	ctx := context.Background()
	s := New(fake.New())
	if _, err := s.Cleanup(ctx, []CleanupKind{CleanupUnusedVolumes}, ""); err == nil {
		t.Fatal("volume cleanup was allowed without confirmation")
	}
	volumes, err := s.ListVolumes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(volumes) != 4 {
		t.Fatal("refused cleanup mutated volumes")
	}
	result, err := s.Cleanup(ctx, []CleanupKind{CleanupUnusedVolumes}, "volumes")
	if err != nil {
		t.Fatal(err)
	}
	if result.Removed != 1 || len(result.Errors) != 0 {
		t.Fatalf("unexpected cleanup result: %#v", result)
	}
	volumes, err = s.ListVolumes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(volumes) != 3 {
		t.Fatalf("got %d volumes after cleanup", len(volumes))
	}
}

func TestCleanupStoppedContainersAndBuildCache(t *testing.T) {
	ctx := context.Background()
	s := New(fake.New())
	result, err := s.Cleanup(ctx, []CleanupKind{CleanupStoppedContainers}, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Removed != 1 || len(result.Errors) != 0 {
		t.Fatalf("unexpected container cleanup result: %#v", result)
	}
	result, err = s.Cleanup(ctx, []CleanupKind{CleanupBuildCache}, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Reclaimed != 700_000_000 || result.Removed != 1 || len(result.Errors) != 0 {
		t.Fatalf("unexpected cache cleanup result: %#v", result)
	}
	usage, err := s.SystemUsage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if usage.BuildCache.Reclaimable != 0 {
		t.Fatal("build cache was not cleared")
	}
}

func TestSystemUsageAndVolumeReferences(t *testing.T) {
	ctx := context.Background()
	s := New(fake.New())
	usage, err := s.SystemUsage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if usage.Images.Total != 6 || usage.Volumes.Total != 4 || usage.Used <= 0 {
		t.Fatalf("unexpected usage: %#v", usage)
	}
	snapshot, err := s.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, volume := range snapshot.Volumes {
		if volume.Name == "web-content" && len(volume.UsedBy) != 1 {
			t.Fatal("web volume reference was not indexed")
		}
		if volume.Name == "scratch-data" && len(volume.UsedBy) != 0 {
			t.Fatal("unused volume gained a reference")
		}
	}
}
