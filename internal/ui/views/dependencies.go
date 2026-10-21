package views

import (
	"fmt"
	"strings"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/components"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
)

// ReverseDependencies renders the "Why is this image here?" sounding (IMG-11).
func ReverseDependencies(s style.Sheet, image engine.Image, containers []engine.Container, projects []engine.ComposeProject, width int) string {
	var depContainers []engine.Container
	for _, c := range containers {
		if c.ImageID == image.ID || c.Image == image.Name() || matchesRepoTag(image.RepoTags, c.Image) {
			depContainers = append(depContainers, c)
		}
	}

	depProjects := make(map[string][]string) // project -> []services
	for _, c := range depContainers {
		if c.Project != "" {
			svc := c.Service
			if svc == "" {
				svc = c.Name
			}
			depProjects[c.Project] = append(depProjects[c.Project], svc)
		}
	}

	var lines []string

	if len(depContainers) == 0 {
		lines = append(lines, s.Muted("No active or stopped containers reference this image."))
		lines = append(lines, s.Text("✓ Safe to remove or prune.", s.Palette.Success))
		return strings.Join(lines, "\n")
	}

	lines = append(lines, fmt.Sprintf("Referenced by %d container(s) across %d Compose project(s):", len(depContainers), len(depProjects)), "")

	if len(depProjects) > 0 {
		lines = append(lines, s.Heading("COMPOSE PROJECTS"))
		for proj, svcs := range depProjects {
			line := fmt.Sprintf("• %s (%s)", proj, strings.Join(svcs, ", "))
			lines = append(lines, s.Text(line, s.Palette.Accent))
		}
		lines = append(lines, "")
	}

	lines = append(lines, s.Heading("CONTAINERS"))
	for _, c := range depContainers {
		statePill := components.Pill(s, c.State)
		name := style.Fit(c.Name, max(10, width-30), 1)
		ports := ""
		if len(c.Ports) > 0 {
			ports = "  " + strings.Join(c.Ports, ", ")
		}
		lines = append(lines, fmt.Sprintf("• %-20s %s%s", name, statePill, s.Muted(ports)))
	}

	return strings.Join(lines, "\n")
}

func matchesRepoTag(tags []string, target string) bool {
	for _, t := range tags {
		if t == target || strings.TrimPrefix(t, "docker.io/library/") == target || strings.TrimPrefix(t, "docker.io/") == target {
			return true
		}
	}
	return false
}
