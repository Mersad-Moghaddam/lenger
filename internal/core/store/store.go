// Package store normalizes snapshots and image-to-container indices.
package store

import (
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"maps"
	"slices"
)

// Snapshot is an immutable-by-convention resource batch.
type Snapshot struct {
	Images     []engine.Image
	Containers []engine.Container
	Volumes    []engine.Volume
	Usage      engine.SystemUsage
}

// Store retains ordered resources and their lookup indices on the model thread.
type Store struct {
	Images     []engine.Image
	Containers []engine.Container
	Volumes    []engine.Volume
	Usage      engine.SystemUsage
	ByImage    map[string][]engine.Container
	ByVolume   map[string][]engine.Container
}

// Replace rebuilds indices atomically from a complete resource batch.
func (s *Store) Replace(batch Snapshot) {
	s.Images = slices.Clone(batch.Images)
	s.Containers = slices.Clone(batch.Containers)
	s.Volumes = slices.Clone(batch.Volumes)
	s.Usage = batch.Usage
	s.ByImage = make(map[string][]engine.Container)
	s.ByVolume = make(map[string][]engine.Container)
	for n := range s.Containers {
		s.Containers[n].Labels = maps.Clone(s.Containers[n].Labels)
		s.Containers[n].Ports = slices.Clone(s.Containers[n].Ports)
		s.Containers[n].Mounts = slices.Clone(s.Containers[n].Mounts)
		c := s.Containers[n]
		s.ByImage[c.ImageID] = append(s.ByImage[c.ImageID], c)
		for _, m := range c.Mounts {
			if m.Type == "volume" && m.Name != "" {
				s.ByVolume[m.Name] = append(s.ByVolume[m.Name], c)
			}
		}
	}
	for n := range s.Images {
		s.Images[n].RepoTags = slices.Clone(s.Images[n].RepoTags)
		s.Images[n].Digests = slices.Clone(s.Images[n].Digests)
		s.Images[n].UsedBy = slices.Clone(s.ByImage[s.Images[n].ID])
	}
	for n := range s.Volumes {
		s.Volumes[n].Labels = maps.Clone(s.Volumes[n].Labels)
		s.Volumes[n].UsedBy = slices.Clone(s.ByVolume[s.Volumes[n].Name])
	}
}

// Invalidate identifies events which require a resource refresh.
func Invalidate(event engine.Event) bool {
	switch event.Kind {
	case "image", "container", "volume", "build-cache":
		return true
	case "connection":
		return event.Action == "connecting"
	}
	return false
}
