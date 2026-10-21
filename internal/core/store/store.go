// Package store normalizes snapshots and image-to-container indices.
package store

import (
	"maps"
	"slices"
	"strings"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// Snapshot is an immutable-by-convention resource batch.
type Snapshot struct {
	Images     []engine.Image
	Containers []engine.Container
	Volumes    []engine.Volume
	Networks   []engine.Network
	Usage      engine.SystemUsage
	Daemon     engine.DaemonInfo
}

// Store retains ordered resources and their lookup indices on the model thread.
type Store struct {
	Images     []engine.Image
	Containers []engine.Container
	Volumes    []engine.Volume
	Networks   []engine.Network
	Projects   []engine.ComposeProject
	Usage      engine.SystemUsage
	Daemon     engine.DaemonInfo
	ByImage    map[string][]engine.Container
	ByVolume   map[string][]engine.Container
	ByProject  map[string][]engine.Container
	ByNetwork  map[string][]engine.Container
}

// Replace rebuilds indices atomically from a complete resource batch.
func (s *Store) Replace(batch Snapshot) {
	s.Images = slices.Clone(batch.Images)
	s.Containers = slices.Clone(batch.Containers)
	s.Volumes = slices.Clone(batch.Volumes)
	s.Networks = slices.Clone(batch.Networks)
	s.Usage = batch.Usage
	s.Daemon = batch.Daemon

	s.ByImage = make(map[string][]engine.Container)
	s.ByVolume = make(map[string][]engine.Container)
	s.ByProject = make(map[string][]engine.Container)
	s.ByNetwork = make(map[string][]engine.Container)

	for n := range s.Containers {
		s.Containers[n].Labels = maps.Clone(s.Containers[n].Labels)
		s.Containers[n].Ports = slices.Clone(s.Containers[n].Ports)
		s.Containers[n].Mounts = slices.Clone(s.Containers[n].Mounts)
		s.Containers[n].ProjectConfigFiles = slices.Clone(s.Containers[n].ProjectConfigFiles)
		c := s.Containers[n]
		s.ByImage[c.ImageID] = append(s.ByImage[c.ImageID], c)
		for _, m := range c.Mounts {
			if m.Type == "volume" && m.Name != "" {
				s.ByVolume[m.Name] = append(s.ByVolume[m.Name], c)
			}
		}
		if proj := strings.TrimSpace(c.Project); proj != "" {
			s.ByProject[proj] = append(s.ByProject[proj], c)
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

	// Index containers by network endpoints
	for n := range s.Networks {
		s.Networks[n].Subnets = slices.Clone(s.Networks[n].Subnets)
		s.Networks[n].Gateways = slices.Clone(s.Networks[n].Gateways)
		s.Networks[n].Containers = slices.Clone(s.Networks[n].Containers)
		s.Networks[n].Labels = maps.Clone(s.Networks[n].Labels)

		for _, ep := range s.Networks[n].Containers {
			for _, c := range s.Containers {
				if c.ID == ep.ContainerID || c.Name == ep.Name {
					s.ByNetwork[s.Networks[n].ID] = append(s.ByNetwork[s.Networks[n].ID], c)
					s.ByNetwork[s.Networks[n].Name] = append(s.ByNetwork[s.Networks[n].Name], c)
				}
			}
		}
	}

	s.Projects = GroupProjects(s.Containers)
}

// GroupProjects aggregates containers by their Compose project label (PRJ-01).
func GroupProjects(containers []engine.Container) []engine.ComposeProject {
	groups := make(map[string][]engine.Container)
	projectOrder := make([]string, 0)
	for _, c := range containers {
		proj := strings.TrimSpace(c.Project)
		if proj == "" {
			continue
		}
		if _, exists := groups[proj]; !exists {
			projectOrder = append(projectOrder, proj)
		}
		groups[proj] = append(groups[proj], c)
	}
	slices.Sort(projectOrder)

	projects := make([]engine.ComposeProject, 0, len(projectOrder))
	for _, name := range projectOrder {
		conts := groups[name]
		var workingDir string
		var configFiles []string
		serviceMap := make(map[string]bool)
		allRunning := true
		anyRunning := false

		for _, c := range conts {
			if workingDir == "" && c.ProjectDir != "" {
				workingDir = c.ProjectDir
			}
			if len(configFiles) == 0 && len(c.ProjectConfigFiles) > 0 {
				configFiles = slices.Clone(c.ProjectConfigFiles)
			}
			if c.Service != "" {
				serviceMap[c.Service] = true
			}
			if c.Running() {
				anyRunning = true
			} else {
				allRunning = false
			}
		}

		services := make([]string, 0, len(serviceMap))
		for svc := range serviceMap {
			services = append(services, svc)
		}
		slices.Sort(services)

		status := "stopped"
		if len(conts) > 0 {
			if allRunning {
				status = "running"
			} else if anyRunning {
				status = "partial"
			}
		}

		projects = append(projects, engine.ComposeProject{
			Name:        name,
			WorkingDir:  workingDir,
			ConfigFiles: configFiles,
			Services:    services,
			Status:      status,
			Containers:  conts,
		})
	}
	return projects
}

// Invalidate identifies events which require a resource refresh.
func Invalidate(event engine.Event) bool {
	switch event.Kind {
	case "image", "container", "volume", "build-cache", "network", "compose":
		return true
	case "connection":
		return event.Action == "connecting"
	}
	return false
}
