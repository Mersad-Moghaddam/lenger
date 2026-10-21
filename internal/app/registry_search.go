package app

import (
	"fmt"
	"strings"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/components"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
)

func (m *Model) registrySearchOverlay() string {
	s := m.sheet
	lines := []string{s.Heading("REGISTRY SEARCH  [Docker Hub / GHCR / Quay]"), ""}

	// Search input prompt
	searchPrompt := "Search: " + m.input.View()
	lines = append(lines, searchPrompt, "")

	if m.registrySearching {
		lines = append(lines, s.Muted("Searching registries…"), "")
	}

	if m.registryErr != "" {
		lines = append(lines, s.Text("Search error: "+style.Clean(m.registryErr), s.Palette.Danger), "")
	}

	if len(m.registryResults) == 0 && !m.registrySearching && m.registryErr == "" {
		if strings.TrimSpace(m.input.Value()) != "" {
			lines = append(lines, s.Muted("No registry repositories found matching query."))
		} else {
			lines = append(lines, s.Muted("Type an image name (e.g. 'nginx', 'postgres', 'redis') and press Enter to search."))
		}
		lines = append(lines, "", s.Muted("enter search   esc close"))
		return strings.Join(lines, "\n")
	}

	// Two-pane or stacked view: Search Results & Remote Tag Browser
	if m.registryBrowsingTags {
		selectedRepo := ""
		if m.registryCursor >= 0 && m.registryCursor < len(m.registryResults) {
			selectedRepo = m.registryResults[m.registryCursor].Name
		}
		lines = append(lines, s.Heading("TAGS FOR "+selectedRepo), "")

		if len(m.registryTags) == 0 {
			lines = append(lines, s.Muted("Loading tags or no tags available…"))
		} else {
			capacity := 10
			start := max(0, m.registryTagCursor-capacity+1)
			end := min(len(m.registryTags), start+capacity)
			g := tokens.Icons(s.ASCII)
			for i := start; i < end; i++ {
				tag := m.registryTags[i]
				marker := " "
				if i == m.registryTagCursor {
					marker = g.Cursor
				}
				sizeStr := components.Bytes(tag.Size)
				digestStr := tag.Digest
				if len(digestStr) > 19 {
					digestStr = digestStr[:19] + "…"
				}
				tagLine := fmt.Sprintf("%s %-20s %8s  %s", marker, style.Fit(tag.Name, 20, 1), sizeStr, s.Muted(digestStr))
				if i == m.registryTagCursor {
					tagLine = s.Text(tagLine, s.Palette.Accent)
				}
				lines = append(lines, tagLine)
			}
		}
		lines = append(lines, "", "↑↓ select tag   enter pull selected   esc back to results")
		return strings.Join(lines, "\n")
	}

	// Results list
	capacity := 8
	start := max(0, m.registryCursor-capacity+1)
	end := min(len(m.registryResults), start+capacity)
	g := tokens.Icons(s.ASCII)

	for i := start; i < end; i++ {
		res := m.registryResults[i]
		marker := " "
		if i == m.registryCursor {
			marker = g.Cursor
		}

		starStr := fmt.Sprintf("★ %d", res.Stars)
		if res.Stars >= 1000 {
			starStr = fmt.Sprintf("★ %.1fk", float64(res.Stars)/1000.0)
		}

		officialBadge := ""
		if res.Official {
			officialBadge = " " + s.Text("[OFFICIAL]", s.Palette.Success)
		} else if res.Automated {
			officialBadge = " " + s.Text("[AUTO]", s.Palette.Accent)
		}

		desc := style.Fit(style.Clean(res.Description), max(10, m.width-45), 1)
		row := fmt.Sprintf("%s %-24s %7s%s", marker, style.Fit(res.Name, 24, 1), starStr, officialBadge)
		if i == m.registryCursor {
			row = s.Text(row, s.Palette.Accent)
		}
		lines = append(lines, row)
		if desc != "" {
			lines = append(lines, "    "+s.Muted(desc))
		}
	}

	lines = append(lines, "", "↑↓ select   t view tags   enter pull latest   / search   esc close")
	return strings.Join(lines, "\n")
}
