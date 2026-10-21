package inspector

import (
	"strings"
	"sync"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

const logBufferCapacity = 10000
const logBufferByteBudget = 16 * 1024 * 1024

// LogBuffer is a zero-value-ready, concurrency-safe bounded chronological ring.
// Copies of a LogBuffer must not be made after first use.
type LogBuffer struct {
	mu          sync.RWMutex
	entries     []engine.LogEntry
	start, size int
	total       uint64
	textBytes   int
}

func (b *LogBuffer) Append(entry engine.LogEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.entries == nil {
		b.entries = make([]engine.LogEntry, logBufferCapacity)
	}
	// Engines other than Docker may provide oversized entries. Bound those too;
	// clone to avoid retaining a large backing string behind a short substring.
	if len(entry.Text) > logBufferByteBudget {
		entry.Text = strings.Clone(entry.Text[:logBufferByteBudget-len(" [truncated]")]) + " [truncated]"
	}
	for b.size > 0 && (b.size == logBufferCapacity || b.textBytes+len(entry.Text) > logBufferByteBudget) {
		b.textBytes -= len(b.entries[b.start].Text)
		b.entries[b.start] = engine.LogEntry{}
		b.start = (b.start + 1) % logBufferCapacity
		b.size--
	}
	entry.Text = strings.Clone(entry.Text)
	b.entries[(b.start+b.size)%logBufferCapacity] = entry
	b.size++
	b.textBytes += len(entry.Text)
	b.total++
}

// Lines returns an independent snapshot, oldest first. Render does not use this
// full-buffer export: it only copies the visible matching viewport.
func (b *LogBuffer) Lines() []engine.LogEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()
	lines := make([]engine.LogEntry, b.size)
	for i := range lines {
		lines[i] = b.entries[(b.start+i)%logBufferCapacity]
	}
	return lines
}

func (b *LogBuffer) Len() int { b.mu.RLock(); defer b.mu.RUnlock(); return b.size }

func (b *LogBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	clear(b.entries)
	b.start, b.size = 0, 0
	b.textBytes = 0
}

func (b *LogBuffer) sequence() uint64 { b.mu.RLock(); defer b.mu.RUnlock(); return b.total }

// viewport copies at most rows entries. Unfiltered tail work is O(rows);
// filtered views scan the bounded ring without formatting offscreen entries.
func (b *LogBuffer) clampEnd(end uint64, rows int) uint64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.clampEndLocked(end, rows)
}

func (b *LogBuffer) clampEndLocked(end uint64, rows int) uint64 {
	oldest := b.total - uint64(b.size)
	return max(min(end, b.total), min(b.total, oldest+uint64(max(0, rows))))
}

func (b *LogBuffer) viewport(end uint64, rows int, match func(engine.LogEntry) bool) ([]engine.LogEntry, uint64) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	end = b.clampEndLocked(end, rows)
	oldest := b.total - uint64(b.size)
	// A viewport cannot scroll beyond its first full page, including after the
	// anchored entries have been evicted while collection continued.
	result := make([]engine.LogEntry, 0, min(rows, b.size))
	for seq := end; seq > oldest && len(result) < rows; {
		seq--
		entry := b.entries[(b.start+int(seq-oldest))%logBufferCapacity]
		if match == nil || match(entry) {
			result = append(result, entry)
		}
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result, end
}
