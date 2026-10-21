package inspector

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/charmbracelet/x/ansi"
)

// LogsViewModel is owned by the UI loop. Buffer may also be appended to from
// reader goroutines; all other fields and methods belong to the UI loop only.
type LogsViewModel struct {
	Buffer        *LogBuffer
	Follow        bool
	UnreadCount   int
	SearchQuery   string
	LevelFilter   string
	IsPrevious    bool
	DriverErr     error
	PreviousUntil time.Time
	observed      uint64
	end           uint64
	wasFollow     bool
	initialized   bool
	viewportRows  int
}

// Append collects regardless of follow state and immediately updates unread.
func (m *LogsViewModel) Append(entry engine.LogEntry) {
	m.logsSync()
	m.Buffer.Append(entry)
	m.logsSync()
}

// SetFollow(true) implements f/G/End: clear unread and jump to the live bottom.
// Direct Follow assignment is reconciled by the next Append, Scroll or Render.
func (m *LogsViewModel) SetFollow(follow bool) {
	m.logsSync()
	if m.Follow != follow {
		if follow {
			m.viewportRows++
		} else {
			m.viewportRows = max(0, m.viewportRows-1)
		}
	}
	m.Follow = follow
	if follow {
		m.UnreadCount = 0
		m.end = m.Buffer.sequence()
	}
	m.wasFollow = follow
}

// Scroll moves the viewport by raw log entries (negative is upward), pausing
// follow. The sequence anchor survives ring wrap until its entries are evicted.
func (m *LogsViewModel) Scroll(delta int) {
	m.logsSync()
	m.SetFollow(false)
	// Normalize against current retention before applying a delta: collection
	// may have evicted the paused anchor since the last rendered snapshot.
	m.end = m.Buffer.clampEnd(m.end, m.viewportRows)
	if delta < 0 {
		distance := uint64(-(delta + 1)) + 1
		if distance > m.end {
			m.end = 0
		} else {
			m.end -= distance
		}
	} else {
		m.end = min(m.end+uint64(delta), m.Buffer.sequence())
	}
	m.end = m.Buffer.clampEnd(m.end, m.viewportRows)
}

func (m *LogsViewModel) logsSync() {
	if m.Buffer == nil {
		m.Buffer = &LogBuffer{}
	}
	total := m.Buffer.sequence()
	if !m.initialized {
		m.end = total
		m.initialized = true
	}
	if m.Follow {
		m.UnreadCount = 0
		m.end = total
	} else {
		if total >= m.observed {
			m.UnreadCount += int(total - m.observed)
		}
		if m.wasFollow {
			m.end = m.observed
		}
	}
	m.observed = total
	m.wasFollow = m.Follow
}

var logsErrorLevel = regexp.MustCompile(`(?i)\b(error|err|fatal|panic)\b`)
var logsWarnLevel = regexp.MustCompile(`(?i)\b(warn|warning)\b`)

func (m *LogsViewModel) Render(s style.Sheet, w, h int) string {
	m.logsSync()
	if w <= 0 || h <= 0 {
		return ""
	}
	var rows []string
	if m.IsPrevious {
		at := "unknown time"
		if !m.PreviousUntil.IsZero() {
			at = m.PreviousUntil.Format(time.RFC3339)
		}
		rows = append(rows, s.Text("[Viewing logs prior to restart at "+at+" · Press 'l' to resume live tail]", s.Palette.Warning))
	}
	if m.DriverErr != nil {
		message := "Unable to read container logs. Check container access and logging configuration, then retry."
		if errors.Is(m.DriverErr, engine.ErrLogsNotSupported) {
			message = "This container's logging driver does not support reading logs. Configure a readable driver (such as json-file or local) to view logs here."
		}
		rows = append(rows, s.Text(message, s.Palette.Warning))
	}
	if !m.Follow {
		rows = append(rows, s.Muted(fmt.Sprintf("[↓ %d new lines]", m.UnreadCount)))
	}
	count := max(0, h-len(rows))
	m.viewportRows = count
	query := strings.ToLower(m.SearchQuery)
	level := strings.ToLower(m.LevelFilter)
	var match func(engine.LogEntry) bool
	if query != "" || (level != "" && level != "all") {
		match = func(entry engine.LogEntry) bool {
			text := logsSanitize(entry.Text, false)
			if !strings.Contains(strings.ToLower(text), query) {
				return false
			}
			switch level {
			case "error":
				return logsErrorLevel.MatchString(text)
			case "warn":
				return logsWarnLevel.MatchString(text)
			}
			return true
		}
	}
	entries, effectiveEnd := m.Buffer.viewport(m.end, count, match)
	m.end = effectiveEnd
	for _, entry := range entries {
		text := logsSanitize(entry.Text, !s.Plain)
		if !s.Plain && strings.Contains(text, "\x1b[") {
			text += "\x1b[0m"
		}
		rows = append(rows, ansi.Truncate(text, w, ""))
	}
	return style.Fit(strings.Join(rows, "\n"), w, h)
}

// logsSanitize allowlists SGR only, including 256-color and TrueColor. OSC
// (clipboard/hyperlinks), DCS and cursor/window controls are never forwarded.
// Each entry occupies one viewport row, even if the driver includes newlines.
func logsSanitize(text string, color bool) string {
	var out strings.Builder
	for i := 0; i < len(text); {
		if text[i] == 0x1b {
			start := i
			i++
			if i >= len(text) {
				break
			}
			switch text[i] {
			case '[':
				i++
				params := i
				for i < len(text) && text[i] >= 0x20 && text[i] <= 0x3f {
					i++
				}
				if i < len(text) && text[i] >= 0x40 && text[i] <= 0x7e {
					valid := text[i] == 'm'
					for j := params; j < i; j++ {
						if (text[j] < '0' || text[j] > '9') && text[j] != ';' && text[j] != ':' {
							valid = false
						}
					}
					i++
					if valid && color {
						out.WriteString(text[start:i])
					}
				}
			case ']', 'P', '^', '_', 'X':
				i++
				for i < len(text) {
					if text[i] == 7 {
						i++
						break
					}
					if text[i] == 0x1b && i+1 < len(text) && text[i+1] == '\\' {
						i += 2
						break
					}
					i++
				}
			default:
				for i < len(text) && text[i] >= 0x20 && text[i] <= 0x2f {
					i++
				}
				if i < len(text) {
					i++
				}
			}
			continue
		}
		// Decode C1 Unicode controls as well as ASCII control bytes.
		r, size := utf8.DecodeRuneInString(text[i:])
		i += size
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			out.WriteByte(' ')
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}
