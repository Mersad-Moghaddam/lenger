package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// Logs streams demultiplexed lines and closes the daemon response on return.
func (e *Engine) Logs(ctx context.Context, id string, opts engine.LogOptions, emit func(engine.LogEntry) error) error {
	if opts.Tail <= 0 {
		opts.Tail = 200
	}
	body, err := e.client.ContainerLogs(ctx, id, client.ContainerLogsOptions{
		ShowStdout: true, ShowStderr: true, Follow: opts.Follow,
		Timestamps: true, Tail: strconv.Itoa(opts.Tail),
	})
	if err != nil {
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
	stream  string
	partial []byte
	emit    func(engine.LogEntry) error
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.partial = append(w.partial, p...)
	for {
		n := bytes.IndexByte(w.partial, '\n')
		if n < 0 {
			return len(p), nil
		}
		if err := w.line(w.partial[:n]); err != nil {
			return 0, err
		}
		w.partial = w.partial[n+1:]
	}
}

func (w *logWriter) Flush() error {
	if len(w.partial) == 0 {
		return nil
	}
	line := append([]byte(nil), w.partial...)
	w.partial = nil
	return w.line(line)
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
