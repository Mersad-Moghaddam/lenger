package app

import "strings"

func (m *Model) renderResourceDetail(rows []string, w, h int) string {
	capacity := max(1, h-5)
	start := min(m.resourceScroll, max(0, len(rows)-capacity))
	end := min(len(rows), start+capacity)
	return m.sheet.Panel("DETAIL · Esc Back · ↑/↓ scroll", strings.Join(rows[start:end], "\n"), w, h, true)
}
