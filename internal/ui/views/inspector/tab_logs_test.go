package inspector

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
)

func TestLogsBufferBoundedChronologicalAndClear(t *testing.T) {
	b := &LogBuffer{}
	for i := 0; i < 12000; i++ {
		b.Append(engine.LogEntry{Text: fmt.Sprint(i)})
	}
	lines := b.Lines()
	if b.Len() != 10000 || lines[0].Text != "2000" || lines[9999].Text != "11999" {
		t.Fatalf("incorrect bounded chronology: %d %v %v", b.Len(), lines[0], lines[len(lines)-1])
	}
	lines[0].Text = "mutated"
	if b.Lines()[0].Text != "2000" {
		t.Fatal("snapshot aliases storage")
	}
	b.Clear()
	b.Append(engine.LogEntry{Text: "fresh"})
	if b.Len() != 1 || b.Lines()[0].Text != "fresh" {
		t.Fatal("clear did not reset ring")
	}
}

func TestLogsBufferConcurrent(t *testing.T) {
	b := &LogBuffer{}
	var wg sync.WaitGroup
	for n := 0; n < 4; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				b.Append(engine.LogEntry{Text: "entry"})
				_ = b.Len()
				_ = b.Lines()
				if i%50 == 0 {
					b.Clear()
				}
			}
		}()
	}
	wg.Wait()
	if b.Len() > 10000 {
		t.Fatal("unbounded")
	}
}

func TestLogsPauseCollectAndResume(t *testing.T) {
	m := &LogsViewModel{Follow: true}
	for i := 0; i < 10; i++ {
		m.Append(engine.LogEntry{Text: fmt.Sprintf("line-%02d", i)})
	}
	m.Render(style.Sheet{Plain: true}, 100, 4)
	m.Scroll(-2)
	before := m.Render(style.Sheet{Plain: true}, 100, 4)
	m.Append(engine.LogEntry{Text: "later"})
	after := m.Render(style.Sheet{Plain: true}, 100, 4)
	if m.Follow || m.UnreadCount != 1 || !strings.Contains(after, "[↓ 1 new lines]") || strings.Contains(after, "later") {
		t.Fatalf("pause state: %#v %q", m, after)
	}
	if !strings.Contains(before, "line-07") || !strings.Contains(after, "line-07") {
		t.Fatalf("viewport shifted: %q -> %q", before, after)
	}
	m.SetFollow(true)
	if m.UnreadCount != 0 || !strings.Contains(m.Render(style.Sheet{Plain: true}, 100, 4), "later") {
		t.Fatal("explicit resume did not tail")
	}
	m.Follow = false
	m.Append(engine.LogEntry{Text: "newer"})
	if m.UnreadCount != 1 {
		t.Fatal("direct pause incompatible")
	}
	m.Follow = true
	m.Render(style.Sheet{Plain: true}, 100, 4)
	if m.UnreadCount != 0 {
		t.Fatal("direct resume incompatible")
	}
}

func TestLogsDirectBufferAppendAndWrapPause(t *testing.T) {
	m := &LogsViewModel{Buffer: &LogBuffer{}, Follow: true}
	for i := 0; i < 10000; i++ {
		m.Buffer.Append(engine.LogEntry{Text: fmt.Sprintf("line-%05d", i)})
	}
	m.Render(style.Sheet{Plain: true}, 100, 4)
	m.SetFollow(false)
	for i := 0; i < 20; i++ {
		m.Buffer.Append(engine.LogEntry{Text: "incoming"})
	}
	out := m.Render(style.Sheet{Plain: true}, 100, 4)
	if m.UnreadCount != 20 || !strings.Contains(out, "line-09999") || strings.Contains(out, "incoming") {
		t.Fatalf("wrap pause: %q unread=%d", out, m.UnreadCount)
	}
}

func TestLogsSearchLevelAndVirtualization(t *testing.T) {
	m := &LogsViewModel{Follow: true}
	for i := 0; i < 10000; i++ {
		m.Append(engine.LogEntry{Text: fmt.Sprintf("INFO ordinary-%05d", i)})
	}
	m.Append(engine.LogEntry{Text: "ERROR NeEdLe failed"})
	m.Append(engine.LogEntry{Text: "WARN needle caution"})
	m.SearchQuery = "NEEDLE"
	m.LevelFilter = "error"
	out := m.Render(style.Sheet{Plain: true}, 100, 3)
	if !strings.Contains(out, "NeEdLe") || strings.Contains(out, "caution") || strings.Contains(out, "ordinary") {
		t.Fatalf("filter: %q", out)
	}
	m.LevelFilter = "warn"
	if out = m.Render(style.Sheet{Plain: true}, 100, 3); !strings.Contains(out, "caution") || strings.Contains(out, "failed") {
		t.Fatalf("warn filter: %q", out)
	}
	m.SearchQuery = ""
	m.LevelFilter = "all"
	out = m.Render(style.Sheet{Plain: true}, 30, 2)
	if strings.Contains(out, "ordinary") || len(strings.Split(out, "\n")) != 2 {
		t.Fatalf("not a bounded tail: %q", out)
	}
}

func TestLogsPreviousAndDriverErrors(t *testing.T) {
	at := time.Date(2026, 10, 21, 8, 30, 0, 0, time.UTC)
	m := &LogsViewModel{IsPrevious: true, PreviousUntil: at, Follow: true}
	out := m.Render(style.Sheet{Plain: true}, 150, 3)
	if !strings.Contains(out, "Viewing logs prior to restart at 2026-10-21T08:30:00Z") || !strings.Contains(out, "Press 'l' to resume live tail") {
		t.Fatalf("banner: %q", out)
	}
	m.DriverErr = fmt.Errorf("wrapped: %w", engine.ErrLogsNotSupported)
	if out = m.Render(style.Sheet{Plain: true}, 150, 4); !strings.Contains(out, "logging driver") || !strings.Contains(out, "does not support") {
		t.Fatalf("unsupported: %q", out)
	}
	m.DriverErr = errors.New("permission denied https://user:fake-secret@daemon/private/socket")
	out = m.Render(style.Sheet{Plain: true}, 200, 4)
	if !strings.Contains(out, "Unable to read container logs") {
		t.Fatal("lost safe driver explanation")
	}
	for _, secret := range []string{"permission denied", "fake-secret", "daemon", "/private/socket"} {
		if strings.Contains(out, secret) {
			t.Fatalf("raw error leaked %q: %q", secret, out)
		}
	}
}

func TestLogsScrollRecoversAfterOverscrollAndEviction(t *testing.T) {
	for _, eviction := range []bool{false, true} {
		t.Run(fmt.Sprintf("eviction=%t", eviction), func(t *testing.T) {
			m := &LogsViewModel{Follow: true}
			for i := 0; i < 8; i++ {
				m.Append(engine.LogEntry{Text: fmt.Sprintf("row-%05d", i)})
			}
			m.Render(style.Sheet{Plain: true}, 60, 4)
			m.Scroll(-1000000)
			m.Render(style.Sheet{Plain: true}, 60, 4)
			last := 8
			if eviction {
				last = 10020
				for i := 8; i < last; i++ {
					m.Buffer.Append(engine.LogEntry{Text: fmt.Sprintf("row-%05d", i)})
				}
				// Scroll must reconcile eviction even before another Render.
			}
			m.Scroll(1)
			oldest := max(0, last-10000)
			out := m.Render(style.Sheet{Plain: true}, 60, 4)
			if strings.Contains(out, fmt.Sprintf("row-%05d", oldest)) || !strings.Contains(out, fmt.Sprintf("row-%05d", oldest+3)) {
				t.Fatalf("down step stuck after clamp: %q", out)
			}
			m.Scroll(-1)
			out = m.Render(style.Sheet{Plain: true}, 60, 4)
			if !strings.Contains(out, fmt.Sprintf("row-%05d", oldest)) {
				t.Fatalf("up step failed: %q", out)
			}
			m.Append(engine.LogEntry{Text: "incoming"})
			wantUnread := last - 8 + 1
			if m.UnreadCount != wantUnread {
				t.Fatalf("unread=%d want=%d", m.UnreadCount, wantUnread)
			}
			m.Scroll(1000000)
			m.Render(style.Sheet{Plain: true}, 60, 4)
			m.Scroll(-1)
			if out = m.Render(style.Sheet{Plain: true}, 60, 4); strings.Contains(out, "incoming") {
				t.Fatalf("up step stuck at bottom: %q", out)
			}
			if m.UnreadCount != wantUnread || m.Follow {
				t.Fatal("navigation changed unread/follow")
			}
			m.SetFollow(true)
			if out = m.Render(style.Sheet{Plain: true}, 60, 4); !strings.Contains(out, "incoming") || m.UnreadCount != 0 {
				t.Fatalf("resume failed: %q", out)
			}
		})
	}
}

func TestLogsSanitizeControlsPreserveSGR(t *testing.T) {
	m := &LogsViewModel{Follow: true}
	m.Append(engine.LogEntry{Text: "\x1b[38;2;1;2;3mcolor\x1b[0m\x1b[2J\x1b]52;c;c2VjcmV0\aOK\r\n\x1b[H\u009b2J\u2028end"})
	out := m.Render(style.Sheet{}, 200, 2)
	if !strings.Contains(out, "\x1b[38;2;1;2;3mcolor") || strings.Contains(out, "\x1b[2J") || strings.Contains(out, "52;") || strings.ContainsAny(out, "\r\a\u009b\u2028") {
		t.Fatalf("unsafe ANSI: %q", out)
	}
	if strings.Contains(m.Render(style.Sheet{Plain: true}, 200, 2), "\x1b") {
		t.Fatal("plain mode retained ANSI")
	}
	if m.Render(style.Sheet{}, 0, 2) != "" || m.Render(style.Sheet{}, 5, 0) != "" {
		t.Fatal("invalid geometry")
	}
}

func TestLogsScrollClampsAndMalformedANSI(t *testing.T) {
	m := &LogsViewModel{Follow: true}
	for i := 0; i < 8; i++ {
		m.Append(engine.LogEntry{Text: fmt.Sprintf("row-%d", i)})
	}
	m.Scroll(-1000)
	out := m.Render(style.Sheet{Plain: true}, 60, 4)
	if !strings.Contains(out, "row-0") || !strings.Contains(out, "row-2") {
		t.Fatalf("scroll past top lost oldest viewport: %q", out)
	}
	m.SetFollow(true)
	m.Append(engine.LogEntry{Text: "\xffbefore\x1b]52;c;unterminated"})
	out = m.Render(style.Sheet{}, 100, 2)
	if !strings.Contains(out, "before") || strings.Contains(out, "52;") {
		t.Fatalf("malformed input: %q", out)
	}
}
