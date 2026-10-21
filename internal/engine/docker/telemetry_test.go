package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/moby/moby/client"
)

func testAdapter(t *testing.T, handler http.HandlerFunc) *Engine {
	t.Helper()
	s := httptest.NewServer(handler)
	t.Cleanup(s.Close)
	c, err := client.New(client.WithHost(s.URL), client.WithAPIVersion("1.52"), client.WithHTTPClient(&http.Client{Transport: &failedResponseTransport{base: http.DefaultTransport}}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return &Engine{client: c}
}

type trackingTransport struct{ closed *atomic.Int32 }

func (tr trackingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := http.DefaultTransport.RoundTrip(r)
	if err == nil {
		response.Body = &trackingBody{ReadCloser: response.Body, closed: tr.closed}
	}
	return response, err
}

type trackingBody struct {
	io.ReadCloser
	closed *atomic.Int32
}

func (b *trackingBody) Close() error { b.closed.Add(1); return b.ReadCloser.Close() }

func TestStatsStopBetweenRequests(t *testing.T) {
	for _, tc := range []struct {
		name, payload      string
		status             int
		stopped, wantError bool
		inspections        int
	}{
		{"state error stopped", `{"message":"container is not running"}`, 409, true, false, 2},
		{"empty stopped", "", 200, true, false, 2},
		{"object stopped", `{}`, 200, true, false, 2},
		{"null stopped", `null`, 200, true, false, 2},
		{"empty still running", "", 200, false, true, 2},
		{"object still running", `{}`, 200, false, true, 2},
		{"state error still running", `{"message":"container is not running"}`, 409, false, true, 2},
		{"permission", `{"message":"permission denied"}`, 403, true, true, 1},
		{"malformed", `{broken`, 200, true, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var inspections atomic.Int32
			var closed atomic.Int32
			e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/json") {
					n := inspections.Add(1)
					fmt.Fprintf(w, `{"State":{"Running":%t,"Status":%q}}`, n == 1 || !tc.stopped, map[bool]string{true: "exited", false: "running"}[n > 1 && tc.stopped])
					return
				}
				w.WriteHeader(tc.status)
				fmt.Fprint(w, tc.payload)
			})
			// Track response closure while preserving real HTTP and SDK behavior.
			c, err := client.New(client.WithHost(e.client.DaemonHost()), client.WithAPIVersion("1.52"), client.WithHTTPClient(&http.Client{Transport: &failedResponseTransport{base: trackingTransport{closed: &closed}}}))
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			e.client = c
			s, err := e.ContainerStats(context.Background(), "c")
			if (err != nil) != tc.wantError {
				t.Fatalf("stats=%+v err=%v", s, err)
			}
			if !tc.wantError && s.State != engine.TelemetryUnavailable {
				t.Fatalf("stats=%+v", s)
			}
			if got := int(inspections.Load()); got != tc.inspections {
				t.Errorf("inspections=%d want=%d", got, tc.inspections)
			}
			if got := int(closed.Load()); got != tc.inspections+1 {
				t.Errorf("closed bodies=%d want=%d", got, tc.inspections+1)
			}
		})
	}
}

func TestStatsInspectErrorDoesNotExposeCredentials(t *testing.T) {
	for _, second := range []bool{false, true} {
		var inspections int
		e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if strings.HasSuffix(r.URL.Path, "/json") {
				inspections++
				if second && inspections == 1 {
					fmt.Fprint(w, `{"State":{"Running":true}}`)
					return
				}
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprint(w, `{"message":"proxy https://user:credential-secret@example.test token=credential-secret"}`)
			}
		})
		_, err := e.ContainerStats(context.Background(), "c")
		if err == nil || strings.Contains(err.Error(), "credential-secret") {
			t.Fatalf("unsafe visible error: %v", err)
		}
		if errors.Unwrap(err) == nil {
			t.Fatal("inspection cause lost")
		}
	}
}

func TestStatsCPUCompatibilityEdges(t *testing.T) {
	for _, tc := range []struct {
		name, cpu string
		count     uint32
		available bool
	}{
		{"per cpu fallback", `{"cpu_usage":{"total_usage":0,"percpu_usage":[0,0]},"system_cpu_usage":100}`, 2, true},
		{"unknown count", `{"cpu_usage":{"total_usage":0},"system_cpu_usage":100}`, 0, true},
		{"null total", `{"cpu_usage":{"total_usage":null},"system_cpu_usage":100}`, 0, false},
		{"null cpu", `null`, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/json") {
					fmt.Fprint(w, `{"State":{"Running":true}}`)
					return
				}
				fmt.Fprintf(w, `{"read":"2026-10-21T10:00:00Z","cpu_stats":%s,"memory_stats":{"usage":null,"limit":100},"networks":null,"blkio_stats":null,"pids_stats":{"current":null}}`, tc.cpu)
			})
			s, err := e.ContainerStats(context.Background(), "c")
			if err != nil {
				t.Fatal(err)
			}
			if s.OnlineCPUs != tc.count || s.CPUAvailable != tc.available || s.MemoryAvailable || s.NetworkAvailable || s.BlockAvailable || s.PIDsAvailable {
				t.Fatalf("stats=%+v", s)
			}
			previous := s
			previous.At = s.At.Add(-time.Second)
			if _, ok := s.CPUPercent(previous); ok {
				t.Fatal("equal system counter available")
			}
			previous.SystemTotal = 50
			previous.State = engine.TelemetryUnavailable
			if _, ok := s.CPUPercent(previous); ok {
				t.Fatal("unavailable prior state accepted")
			}
			previous.State = engine.TelemetryActive
			s.State = engine.TelemetryUnavailable
			if _, ok := s.CPUPercent(previous); ok {
				t.Fatal("unavailable current state accepted")
			}
			s.State = engine.TelemetryActive
			if _, ok := s.CPUPercent(previous); ok != (tc.available && tc.count > 0) {
				t.Fatal("CPU count fallback calculation unavailable")
			}
		})
	}
}

func TestStatsTransportErrorIsNotUnavailable(t *testing.T) {
	var inspections atomic.Int32
	e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/json") {
			inspections.Add(1)
			fmt.Fprint(w, `{"State":{"Running":true}}`)
			return
		}
		connection, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		_ = connection.Close()
	})
	if _, err := e.ContainerStats(context.Background(), "c"); err == nil {
		t.Fatal("transport error downgraded")
	}
	if inspections.Load() != 1 {
		t.Fatal("re-inspected on transport failure")
	}
}

func TestNewEngineStatsHTTP(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/_ping") {
			w.Header().Set("API-Version", "1.52")
			return
		}
		fmt.Fprint(w, `{"State":{"Running":false}}`)
	}))
	defer s.Close()
	t.Setenv("DOCKER_HOST", s.URL)
	t.Setenv("DOCKER_CERT_PATH", "")
	t.Setenv("DOCKER_API_VERSION", "1.52")
	e, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	stats, err := e.ContainerStats(context.Background(), "c")
	if err != nil || stats.State != engine.TelemetryUnavailable {
		t.Fatalf("stats=%+v err=%v", stats, err)
	}
}

func TestLogsCloseBodiesOnExit(t *testing.T) {
	for _, mode := range []string{"callback", "cancel", "inspect failure"} {
		t.Run(mode, func(t *testing.T) {
			var closed atomic.Int32
			e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/json") {
					if mode == "inspect failure" {
						w.WriteHeader(http.StatusForbidden)
						fmt.Fprint(w, `{"message":"denied"}`)
						return
					}
					fmt.Fprint(w, `{"Config":{"Tty":true}}`)
					return
				}
				fmt.Fprint(w, "line\n")
			})
			c, err := client.New(client.WithHost(e.client.DaemonHost()), client.WithAPIVersion("1.52"), client.WithHTTPClient(&http.Client{Transport: &failedResponseTransport{base: trackingTransport{closed: &closed}}}))
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			e.client = c
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			callbackError := errors.New("callback stopped")
			err = e.Logs(ctx, "c", engine.LogOptions{}, func(engine.LogEntry) error {
				if mode == "cancel" {
					cancel()
					return ctx.Err()
				}
				return callbackError
			})
			if err == nil {
				t.Fatal("expected stream error")
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation lost")
			}
			if mode == "callback" && !errors.Is(err, callbackError) {
				t.Fatal("callback error lost")
			}
			if closed.Load() != 2 {
				t.Fatalf("closed=%d want=2", closed.Load())
			}
		})
	}
}

func TestLogsUntilAndPrevious(t *testing.T) {
	until := time.Date(2026, 10, 21, 10, 0, 0, 0, time.FixedZone("offset", 3600))
	e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/json") {
			fmt.Fprint(w, `{"Config":{"Tty":true}}`)
			return
		}
		// The adapter supplies RFC3339; Moby normalizes that to Unix time on the wire.
		if got := r.URL.Query().Get("until"); got != strconv.FormatInt(until.Unix(), 10)+".000000000" {
			t.Errorf("until=%q", got)
		}
		if r.URL.Query().Get("follow") == "1" || r.URL.Query().Get("follow") == "true" {
			t.Error("historical logs must not follow")
		}
		fmt.Fprint(w, "2026-10-21T08:59:59Z previous run\n")
	})
	var entries []engine.LogEntry
	err := e.Logs(context.Background(), "c", engine.LogOptions{Until: until, Previous: true, Follow: true}, func(entry engine.LogEntry) error { entries = append(entries, entry); return nil })
	if err != nil || len(entries) != 1 || entries[0].Text != "previous run" {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
}

func TestLogsUnsupportedDriver(t *testing.T) {
	e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		fmt.Fprint(w, `{"message":"configured logging driver does not support reading"}`)
	})
	err := e.Logs(context.Background(), "c", engine.LogOptions{}, func(engine.LogEntry) error { return nil })
	if !errors.Is(err, engine.ErrLogsNotSupported) {
		t.Fatalf("missing unsupported-driver sentinel: %v", err)
	}
}

func TestStatsAvailabilityAndSafeCPU(t *testing.T) {
	for _, tc := range []struct {
		name, payload string
		available     bool
	}{
		{"omitted", `{"read":"2026-10-21T10:00:00Z"}`, false},
		{"empty groups", `{"read":"2026-10-21T10:00:00Z","cpu_stats":{"cpu_usage":{},"system_cpu_usage":100},"memory_stats":{},"pids_stats":{},"blkio_stats":{}}`, false},
		{"zero", `{"read":"2026-10-21T10:00:00Z","cpu_stats":{"cpu_usage":{"total_usage":0},"system_cpu_usage":100,"online_cpus":2},"memory_stats":{"usage":0,"limit":100},"networks":{},"blkio_stats":{"io_service_bytes_recursive":[]},"pids_stats":{"current":0}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/json") {
					fmt.Fprint(w, `{"State":{"Status":"running","Running":true}}`)
					return
				}
				fmt.Fprint(w, tc.payload)
			})
			s, err := e.ContainerStats(context.Background(), "c")
			if err != nil {
				t.Fatal(err)
			}
			if s.CPUAvailable != tc.available || s.MemoryAvailable != tc.available || s.NetworkAvailable != tc.available || s.BlockAvailable != tc.available || s.PIDsAvailable != tc.available {
				t.Fatalf("incorrect availability: %+v", s)
			}
			if tc.available {
				previous := s
				previous.At = s.At.Add(-time.Second)
				previous.SystemTotal = 50
				if percent, ok := s.CPUPercent(previous); !ok || percent != 0 {
					t.Fatalf("measured zero CPU=%v, available=%v", percent, ok)
				}
				current := s
				current.CPUTotal = 5
				if percent, ok := current.CPUPercent(previous); !ok || percent != 20 {
					t.Fatalf("CPU=%v, available=%v", percent, ok)
				}
				for _, previous := range []engine.ContainerStats{s, {At: s.At.Add(-time.Second), CPUAvailable: true, CPUTotal: 1, SystemTotal: 50}, {At: s.At.Add(-time.Second), CPUAvailable: true, SystemTotal: 200}} {
					if percent, ok := s.CPUPercent(previous); ok || percent != 0 {
						t.Fatalf("invalid delta CPU=%v available=%v", percent, ok)
					}
				}
			}
		})
	}
}

func TestStatsStoppedUnavailable(t *testing.T) {
	for _, state := range []string{"exited", "dead", "created"} {
		t.Run(state, func(t *testing.T) {
			e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if !strings.HasSuffix(r.URL.Path, "/json") {
					t.Error("stats requested for inactive container")
				}
				fmt.Fprintf(w, `{"State":{"Status":%q,"Running":false}}`, state)
			})
			s, err := e.ContainerStats(context.Background(), "c")
			if err != nil || s.State != engine.TelemetryUnavailable || s.MemoryAvailable {
				t.Fatalf("stats=%+v err=%v", s, err)
			}
		})
	}
}

func TestLogsPreviousRequiresBoundary(t *testing.T) {
	e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) { t.Error("unbounded previous logs requested") })
	if err := e.Logs(context.Background(), "c", engine.LogOptions{Previous: true}, func(engine.LogEntry) error { return nil }); err == nil {
		t.Fatal("missing boundary accepted")
	}
}

func TestStatsErrorsRemainErrors(t *testing.T) {
	for _, payload := range []string{`broken`, `{"message":"permission denied"}`} {
		e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if strings.HasSuffix(r.URL.Path, "/json") {
				fmt.Fprint(w, `{"State":{"Running":true}}`)
				return
			}
			if strings.Contains(payload, "permission") {
				w.WriteHeader(http.StatusForbidden)
			}
			fmt.Fprint(w, payload)
		})
		if _, err := e.ContainerStats(context.Background(), "c"); err == nil {
			t.Fatal("stats failure silently treated as missing measurement")
		}
	}
}
