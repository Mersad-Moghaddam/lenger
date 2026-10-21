// Package containers provides container-list presentation primitives.
package containers

import "github.com/Mersad-Moghaddam/lenger/internal/engine"

// ComposeGroup is an initially expanded project header and its containers.
// An empty Project identifies the standalone/default group; renderers may label
// that header "Standalone" without colliding with a real project of that name.
type ComposeGroup struct {
	Project   string
	Services  []engine.Container
	Collapsed bool
}

// GroupByCompose preserves first-seen project order and service input order.
// The canonical Compose project label wins over the engine's derived Project
// field when present. Returned service slices do not alias the input slice;
// nested container metadata remains read-only, as in engine.Container itself.
func GroupByCompose(containers []engine.Container) []ComposeGroup {
	var groups []ComposeGroup
	indices := make(map[string]int)
	for _, container := range containers {
		project := container.Labels["com.docker.compose.project"]
		if project == "" {
			project = container.Project
		}
		i, ok := indices[project]
		if !ok {
			i = len(groups)
			indices[project] = i
			groups = append(groups, ComposeGroup{Project: project})
		}
		groups[i].Services = append(groups[i].Services, container)
	}
	return groups
}
