package inspector

import (
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"strings"
	"testing"
)

func TestResidualAggregateLogBudgetAndAnchor(t *testing.T) {
	m := LogsViewModel{Follow: true}
	record := strings.Repeat("x", 64*1024)
	for i := 0; i < 100; i++ {
		m.Append(engine.LogEntry{Text: record})
	}
	m.Render(style.Sheet{Plain: true}, 80, 8)
	m.Scroll(-5)
	anchor := m.end
	for i := 0; i < 100; i++ {
		m.Append(engine.LogEntry{Text: record})
	}
	if m.end != anchor || m.UnreadCount != 100 {
		t.Fatalf("paused anchor/unread changed: %d %d", m.end, m.UnreadCount)
	}
	for i := 0; i < 200; i++ {
		m.Append(engine.LogEntry{Text: record})
	}
	lines := m.Buffer.Lines()
	bytes := 0
	for _, line := range lines {
		bytes += len(line.Text)
	}
	if bytes > 16*1024*1024 || len(lines) != 256 {
		t.Fatalf("aggregate text %d bytes in %d records", bytes, len(lines))
	}
	m.Render(style.Sheet{Plain: true}, 80, 8)
	if m.end < m.Buffer.sequence()-uint64(len(lines)) {
		t.Fatal("evicted paused anchor not normalized")
	}
	m.SetFollow(true)
	if m.UnreadCount != 0 || m.end != m.Buffer.sequence() {
		t.Fatal("resume semantics lost")
	}
	m.Buffer.Clear()
	if m.Buffer.Len() != 0 {
		t.Fatal("clear retained entries")
	}
	if m.Buffer.textBytes != 0 {
		t.Fatal("clear retained byte accounting")
	}
	m.Buffer.Append(engine.LogEntry{Text: strings.Repeat("z", 20*1024*1024)})
	if len(m.Buffer.Lines()[0].Text) > 16*1024*1024 {
		t.Fatal("non-Docker producer escaped aggregate budget")
	}
}
