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
}

// Store retains ordered resources and their lookup indices on the model thread.
type Store struct {
	Images     []engine.Image
	Containers []engine.Container
	ByImage    map[string][]engine.Container
}

// Replace rebuilds indices atomically from a complete resource batch.
func (s *Store) Replace(batch Snapshot) {
	s.Images = slices.Clone(batch.Images)
	s.Containers = slices.Clone(batch.Containers)
	s.ByImage = make(map[string][]engine.Container)
	for n := range s.Containers {
		s.Containers[n].Labels = maps.Clone(s.Containers[n].Labels)
		c := s.Containers[n]
		s.ByImage[c.ImageID] = append(s.ByImage[c.ImageID], c)
	}
	for n := range s.Images {
		s.Images[n].RepoTags = slices.Clone(s.Images[n].RepoTags)
		s.Images[n].Digests = slices.Clone(s.Images[n].Digests)
		s.Images[n].UsedBy = slices.Clone(s.ByImage[s.Images[n].ID])
	}
}

// Invalidate identifies events which require a resource refresh.
func Invalidate(event engine.Event) bool {
	return event.Kind == "image" || event.Kind == "container" || event.Kind == "connection" && event.Action == "connecting"
}
