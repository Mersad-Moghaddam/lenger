package fake

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// Logs emits a deterministic fixture stream and follows until cancellation.
func (e *Engine) Logs(ctx context.Context, id string, opts engine.LogOptions, emit func(engine.LogEntry) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	e.mu.Lock()
	n := slices.IndexFunc(e.containers, func(c engine.Container) bool { return c.ID == id || c.Name == id })
	if n < 0 {
		e.mu.Unlock()
		return fmt.Errorf("container not found")
	}
	name := e.containers[n].Name
	e.mu.Unlock()

	tail := opts.Tail
	if tail <= 0 {
		tail = 200
	}
	base := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	const seed = 40
	start := max(0, seed-tail)
	for line := start; line < seed; line++ {
		if err := emit(fixtureLogLine(name, line, base.Add(time.Duration(line-seed)*time.Second))); err != nil {
			return err
		}
	}
	if !opts.Follow {
		return nil
	}
	interval := e.Delay
	if interval <= 0 {
		interval = 150 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for line := seed; ; line++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case at := <-ticker.C:
			if err := emit(fixtureLogLine(name, line, at)); err != nil {
				return err
			}
		}
	}
}

func fixtureLogLine(name string, line int, at time.Time) engine.LogEntry {
	stream := "stdout"
	text := fmt.Sprintf("request container=%s path=/health status=200 line=%d", name, line)
	if line%11 == 7 {
		stream = "stderr"
		text = fmt.Sprintf("warn container=%s cache miss key=session-%d", name, line)
	}
	return engine.LogEntry{At: at, Stream: stream, Text: text}
}
