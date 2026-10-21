package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// Logs streams demultiplexed lines and closes the daemon response on return.
func (e *Engine) Logs(ctx context.Context, id string, opts engine.LogOptions, emit func(engine.LogEntry) error) error {
	if opts.Tail <= 0 {
		opts.Tail = 200
	}
	if opts.Previous {
		if opts.Until.IsZero() {
			return fmt.Errorf("previous logs require a restart boundary")
		}
		opts.Follow = false
	}
	until := ""
	if !opts.Until.IsZero() {
		until = opts.Until.Format(time.RFC3339Nano)
	}
	body, err := e.client.ContainerLogs(ctx, id, client.ContainerLogsOptions{
		ShowStdout: true, ShowStderr: true, Follow: opts.Follow,
		Timestamps: true, Tail: strconv.Itoa(opts.Tail), Until: until,
	})
	if err != nil {
		text := strings.ToLower(err.Error())
		if strings.Contains(text, "logging driver") && strings.Contains(text, "does not support reading") {
			return fmt.Errorf("read container logs: %w", engine.ErrLogsNotSupported)
		}
		return fmt.Errorf("read container logs: %w", err)
	}
	defer body.Close() //nolint:errcheck // The stream/context error is authoritative.

	inspect, err := e.client.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return fmt.Errorf("inspect container for logs: %w", err)
	}
	stdout := &logWriter{stream: "stdout", emit: emit}
	stderr := &logWriter{stream: "stderr", emit: emit}
	if inspect.Container.Config != nil && inspect.Container.Config.Tty {
		stdout.stream = "output"
		_, err = io.Copy(stdout, body)
	} else {
		_, err = stdcopy.StdCopy(stdout, stderr, body)
	}
	if flushErr := stdout.Flush(); err == nil {
		err = flushErr
	}
	if flushErr := stderr.Flush(); err == nil {
		err = flushErr
	}
	if err != nil {
		return fmt.Errorf("stream container logs: %w", err)
	}
	return nil
}

type logWriter struct {
	stream    string
	partial   []byte
	emit      func(engine.LogEntry) error
	truncated bool
}

// Retain a bounded prefix per record, discarding the rest until newline. The
// marker is inside the byte cap. This bounds both pending and emitted storage
// even when Docker writes one huge record in a single Write call. Records that
// fit retain their entire content; only oversized records reserve marker space.
// The inspector independently enforces its aggregate retained-text budget.
const maxLogRecordBytes = 64 * 1024
const logTruncated = " [truncated]"

func (w *logWriter) Write(p []byte) (int, error) {
	if w.partial == nil {
		w.partial = make([]byte, 0, maxLogRecordBytes)
	}
	for rest := p; len(rest) > 0; {
		n := bytes.IndexByte(rest, '\n')
		chunk := rest
		if n >= 0 {
			chunk = rest[:n]
		}
		available := maxLogRecordBytes - len(w.partial)
		if len(chunk) > available {
			w.truncated = true
			chunk = chunk[:available]
		}
		w.partial = append(w.partial, chunk...)
		if n < 0 {
			break
		}
		var err error
		if len(w.partial) == 0 {
			err = w.line(nil)
		} else {
			err = w.Flush()
		}
		if err != nil {
			return 0, err
		}
		rest = rest[n+1:]
	}
	return len(p), nil
}

func (w *logWriter) Flush() error {
	if len(w.partial) == 0 {
		return nil
	}
	line := append([]byte(nil), w.partial...)
	if !w.truncated && !utf8.Valid(line) {
		size := 0
		for _, r := range string(line) {
			size += utf8.RuneLen(r)
		}
		if size > maxLogRecordBytes {
			w.truncated = true
		}
	}
	limit := maxLogRecordBytes
	if w.truncated {
		limit -= len(logTruncated)
	}
	line = sanitizeLogUTF8(line, limit, w.truncated)
	if w.truncated {
		// Never leave a partial UTF-8 rune or terminal escape at the cut. Normal
		// complete ANSI sequences pass through unchanged.
		if at := bytes.LastIndexByte(line, 0x1b); at >= 0 {
			tail := line[at:]
			complete := false
			if len(tail) >= 3 && tail[1] == '[' {
				for _, b := range tail[2:] {
					if b >= 0x40 && b <= 0x7e {
						complete = true
						break
					}
				}
			}
			if !complete {
				line = line[:at]
			}
		}
		line = append(line, logTruncated...)
	}
	w.partial = w.partial[:0]
	w.truncated = false
	return w.line(line)
}

// Replace invalid interior bytes without dropping subsequent valid evidence.
// An incomplete rune is omitted only at an actual truncated boundary. Bound
// replacement expansion too (one invalid byte can become three UTF-8 bytes).
func sanitizeLogUTF8(raw []byte, limit int, cut bool) []byte {
	var out strings.Builder
	out.Grow(min(len(raw), limit))
	for len(raw) > 0 {
		if cut && !utf8.FullRune(raw) {
			break
		}
		r, n := utf8.DecodeRune(raw)
		needed := utf8.RuneLen(r)
		if out.Len()+needed > limit {
			break
		}
		out.WriteRune(r)
		raw = raw[n:]
	}
	return []byte(out.String())
}

func (w *logWriter) line(raw []byte) error {
	text := strings.TrimSuffix(string(raw), "\r")
	at, text := splitTimestamp(text)
	return w.emit(engine.LogEntry{At: at, Stream: w.stream, Text: text})
}

func splitTimestamp(line string) (time.Time, string) {
	prefix, rest, ok := strings.Cut(line, " ")
	if !ok {
		return time.Time{}, line
	}
	at, err := time.Parse(time.RFC3339Nano, prefix)
	if err != nil {
		return time.Time{}, line
	}
	return at, rest
}
