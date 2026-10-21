package app

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	groups "github.com/Mersad-Moghaddam/lenger/internal/ui/views/containers"
	"strings"
)

type groupedRow struct {
	project string
	id      string
}

func (m *Model) groupedRows() []groupedRow {
	var rows []groupedRow
	for _, g := range groups.GroupByCompose(m.containers()) {
		rows = append(rows, groupedRow{project: g.Project})
		if !m.groupCollapsed[g.Project] {
			for _, c := range g.Services {
				rows = append(rows, groupedRow{g.Project, c.ID})
			}
		}
	}
	return rows
}
func (m *Model) groupKey(k string) (bool, tea.Cmd) {
	if m.tab != 0 {
		return false, nil
	}
	if k == "g" {
		m.closeDrawer()
		m.grouped = !m.grouped
		m.groupCursor = 0
		return true, nil
	}
	if !m.grouped || m.focus.Area != FocusList {
		return false, nil
	}
	rows := m.groupedRows()
	if len(rows) == 0 {
		return false, nil
	}
	m.groupCursor = max(0, min(m.groupCursor, len(rows)-1))
	if k == "down" || k == "j" || k == "up" || k == "k" {
		d := 1
		if k == "up" || k == "k" {
			d = -1
		}
		m.groupCursor = max(0, min(m.groupCursor+d, len(rows)-1))
		row := rows[m.groupCursor]
		if row.id != "" {
			for i, c := range m.containers() {
				if c.ID == row.id {
					m.containerCursor = i
				}
			}
			if m.focus.DrawerOpen {
				return true, m.openDrawer()
			}
		} else if m.focus.DrawerOpen {
			m.closeDrawer()
		}
		return true, nil
	}
	row := rows[m.groupCursor]
	if row.id != "" {
		for i, c := range m.containers() {
			if c.ID == row.id {
				m.containerCursor = i
			}
		}
		return false, nil
	}
	switch k {
	case "enter", "space", " ":
		m.closeDrawer()
		m.groupCollapsed[row.project] = !m.groupCollapsed[row.project]
		return true, nil
	case "s", "r":
		if row.project == "" {
			return true, m.notify("Select a Compose project to act on a stack.")
		}
		m.stackProject = row.project
		m.stackIDs = nil
		for _, g := range groups.GroupByCompose(m.store.Containers) {
			if g.Project == row.project {
				for _, c := range g.Services {
					m.stackIDs = append(m.stackIDs, c.ID)
				}
			}
		}
		m.actionName = engine.ActionStop
		if k == "r" {
			m.actionName = engine.ActionRestart
		}
		m.overlay = "stack-stop"
		return true, nil
	}
	return false, nil
}

func (m *Model) currentGroupedRow() groupedRow {
	rows := m.groupedRows()
	if !m.grouped || len(rows) == 0 {
		return groupedRow{}
	}
	return rows[max(0, min(m.groupCursor, len(rows)-1))]
}

// Keep headers distinct from service IDs. A vanished service falls back to its
// project header, then the nearest surviving row. Rebind the flat action cursor
// before drawer reconciliation so highlight and action identity cannot diverge.
func (m *Model) restoreGroupedRow(want groupedRow) {
	if !m.grouped {
		return
	}
	rows := m.groupedRows()
	if len(rows) == 0 {
		m.groupCursor = 0
		return
	}
	index := -1
	for i, row := range rows {
		if row == want || (want.id != "" && row.id == want.id) {
			index = i
			break
		}
	}
	if index < 0 {
		for i, row := range rows {
			if row.project == want.project && row.id == "" {
				index = i
				break
			}
		}
	}
	if index < 0 {
		index = max(0, min(m.groupCursor, len(rows)-1))
		for index > 0 && rows[index].id != "" {
			index--
		}
	}
	m.groupCursor = index
	row := rows[index]
	if row.id == "" {
		if m.focus.DrawerOpen {
			m.closeDrawer()
		}
		return
	}
	for i, c := range m.containers() {
		if c.ID == row.id {
			m.containerCursor = i
			break
		}
	}
}
func (m *Model) renderGroupedList(w, h int) string {
	rows := m.groupedRows()
	var lines []string
	lines = append(lines, "COMPOSE GROUPS · g ungroup · Enter collapse · s stop · r restart")
	start := m.groupWindowStart(h)
	for i := start; i < len(rows) && len(lines) < h; i++ {
		r := rows[i]
		prefix := "  "
		if i == m.groupCursor {
			prefix = "> "
		}
		if r.id == "" {
			name := r.project
			if name == "" {
				name = "Standalone"
			}
			mark := "▾"
			if m.groupCollapsed[r.project] {
				mark = "▸"
			}
			lines = append(lines, prefix+mark+" "+style.Clean(name))
		} else {
			c, _ := m.containerByID(r.id)
			lines = append(lines, prefix+"  "+style.Clean(c.Name)+" · "+style.Clean(c.State))
		}
	}
	return style.Fit(strings.Join(lines, "\n"), w, h)
}

func (m *Model) groupWindowStart(h int) int { return max(0, m.groupCursor-max(0, h-3)) }
func (m *Model) stackConfirmation() string {
	lines := []string{fmt.Sprintf("%s stack %s (%d containers)?", m.actionName, style.Clean(m.stackProject), len(m.stackIDs)), ""}
	for _, id := range m.stackIDs {
		c, _ := m.containerByID(id)
		lines = append(lines, style.Clean(c.Name))
	}
	return strings.Join(lines, "\n") + "\n\nStops containers only; never compose down or volume deletion.\nEnter confirm · Esc cancel"
}
