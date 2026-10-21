package docker

import (
	"context"
	"errors"
	"fmt"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/views/inspector"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"net/http"
	"net/netip"
	"strings"
	"testing"
)

func TestWholeAdapterPortPreview(t *testing.T) {
	for _, host := range []string{"0.0.0.0", "127.0.0.1", "192.0.2.1", "::", "::1", "2001:db8::1"} {
		t.Run(host, func(t *testing.T) {
			ip := netip.MustParseAddr(host)
			port, err := network.ParsePort("80/tcp")
			if err != nil {
				t.Fatal(err)
			}
			// Real adapter formatters, not hand-authored drawer fixtures.
			for _, mapping := range []string{formatPortBinding(port, network.PortBinding{HostIP: ip, HostPort: "8080"}), formatPortSummary(container.PortSummary{IP: ip, PrivatePort: 80, PublicPort: 8080, Type: "tcp"})} {
				p, err := inspector.PreviewPort(mapping)
				if err != nil {
					t.Fatalf("%s: %v", mapping, err)
				}
				if (p.Warning != "") == ip.IsLoopback() {
					t.Fatalf("warning for %s: %q", mapping, p.Warning)
				}
			}
		})
	}
}

func TestWholeHugeTerminatedLogAndCancel(t *testing.T) {
	for _, tty := range []bool{true, false} {
		t.Run(fmt.Sprint(tty), func(t *testing.T) {
			var seen engine.LogEntry
			e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/json") {
					_, _ = fmt.Fprintf(w, `{"Config":{"Tty":%t}}`, tty)
					return
				}
				record := "2026-10-21T12:00:00.123456789Z " + strings.Repeat("x", 1024*1024) + "\n"
				if !tty {
					header := []byte{1, 0, 0, 0, 0, 0, 0, 0}
					n := len(record)
					header[4] = byte(n >> 24)
					header[5] = byte(n >> 16)
					header[6] = byte(n >> 8)
					header[7] = byte(n)
					_, _ = w.Write(header)
				}
				_, _ = fmt.Fprint(w, record)
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			err := e.Logs(ctx, "c", engine.LogOptions{Follow: true}, func(entry engine.LogEntry) error { seen = entry; cancel(); return ctx.Err() })
			if !errors.Is(err, context.Canceled) || len(seen.Text) > maxLogRecordBytes || !strings.Contains(seen.Text, "truncated") || seen.At.Nanosecond() != 123456789 {
				t.Fatalf("entry bytes %d timestamp %v error %v", len(seen.Text), seen.At, err)
			}
		})
	}
}

func TestWholeLogRecordByteBound(t *testing.T) {
	var entries []engine.LogEntry
	w := logWriter{emit: func(e engine.LogEntry) error { entries = append(entries, e); return nil }}
	for range 100 {
		if _, err := w.Write([]byte(strings.Repeat("x", 8192))); err != nil {
			t.Fatal(err)
		}
		if len(w.partial) > maxLogRecordBytes || cap(w.partial) > maxLogRecordBytes {
			t.Fatalf("pending bytes %d", len(w.partial))
		}
	}
	if _, err := w.Write([]byte("\n2026-10-21T12:00:00.123456789Z \x1b[31mred\x1b[0m\n")); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || len(entries[0].Text) > maxLogRecordBytes || !strings.Contains(entries[0].Text, "truncated") || entries[1].At.Nanosecond() != 123456789 || entries[1].Text != "\x1b[31mred\x1b[0m" {
		t.Fatalf("bounded entries: %#v", entries)
	}
}

func TestWholeLogCutPreservesCompleteANSIAndUTF8(t *testing.T) {
	var entries []engine.LogEntry
	w := logWriter{emit: func(e engine.LogEntry) error { entries = append(entries, e); return nil }}
	prefix := strings.Repeat("x", maxLogRecordBytes-len(logTruncated)-2)
	if _, err := w.Write([]byte(prefix + "\x1b[31mrest" + strings.Repeat("z", 100) + "\n\n")); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || strings.Contains(entries[0].Text, "\x1b") || entries[1].Text != "" {
		t.Fatalf("partial ANSI or blank line lost: %#v", entries)
	}
	entries = nil
	if _, err := w.Write([]byte(prefix + "界more" + strings.Repeat("z", 100) + "\n")); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || strings.Contains(entries[0].Text, "�") || len(entries[0].Text) > maxLogRecordBytes {
		t.Fatalf("UTF8 cut: %#v", entries)
	}
}
