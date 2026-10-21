package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/containerd/errdefs"
	"github.com/moby/moby/client"
)

func TestInspectHistoryHasNoAggregateHealth(t *testing.T) {
	e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"State":{"Health":{"Status":"unhealthy","FailingStreak":7,"Log":[{"Start":"2026-10-21T01:00:00Z","ExitCode":1},{"Start":"2026-10-21T01:01:00Z","ExitCode":0}]}}}`)
	})
	i, err := e.InspectContainer(context.Background(), "fixture")
	if err != nil {
		t.Fatal(err)
	}
	d := i.DiagnosticEvidence()
	if len(d.HealthHistory) != 2 {
		t.Fatal("missing executions")
	}
	for n, p := range d.HealthHistory {
		if p.Status != "" || p.FailingStreak != 0 {
			t.Errorf("history %d invents aggregate health: status=%q streak=%d", n, p.Status, p.FailingStreak)
		}
		if p.ExitCode != 1-n || !p.ExecutionObserved {
			t.Error("lost execution facts")
		}
	}
	if i.HealthCheck.Status != "unhealthy" || i.HealthCheck.FailingStreak != 7 || i.HealthCheck.ExitCode != 0 {
		t.Error("lost compatibility aggregate/latest execution")
	}
}

func TestInspectHistoryCapSkipsNilSlots(t *testing.T) {
	for _, count := range []int{1, engine.MaxHealthHistory, engine.MaxHealthHistory + 3} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			var slots []string
			for n := 0; n < count; n++ {
				slots = append(slots, "null", fmt.Sprintf(`{"Start":"2026-10-21T01:00:00Z","ExitCode":%d}`, n))
			}
			for n := 0; n < engine.MaxHealthHistory+1; n++ {
				slots = append(slots, "null")
			}
			e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"State":{"Health":{"Log":[%s]}}}`, strings.Join(slots, ","))
			})
			i, err := e.InspectContainer(context.Background(), "fixture")
			if err != nil {
				t.Fatal(err)
			}
			h := i.DiagnosticEvidence().HealthHistory
			want := min(count, engine.MaxHealthHistory)
			if len(h) != want {
				t.Fatalf("retained %d available probes, want %d", len(h), want)
			}
			for n, p := range h {
				if p.ExitCode != count-want+n {
					t.Error("changed daemon order/boundary")
				}
			}
			if i.HealthCheck.ExitCode != count-1 || !i.HealthCheck.ExecutionObserved {
				t.Error("lost latest nonnil compatibility probe")
			}
		})
	}
}

func TestInspectSafeClassification(t *testing.T) {
	t.Run("connection refused", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		address := s.URL
		s.Close()
		c, err := client.New(client.WithHost(address), client.WithAPIVersion("1.52"))
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		e := &Engine{client: c}
		_, err = e.InspectContainer(context.Background(), "fixture")
		if err == nil || engine.Classify(err).Category != "daemon" {
			t.Fatalf("connection refusal not classified as daemon: category=%s cause=%v", engine.Classify(err).Category, errors.Unwrap(err))
		}
		if strings.Contains(err.Error(), address) {
			t.Error("public error exposes connection endpoint")
		}
	})
	for _, tc := range []struct {
		name, message, category string
		status                  int
	}{
		{"missing", "No such container: raw-secret", "not-found", 404},
		{"permission", "permission denied: raw-secret", "permission", 403},
		{"connectivity", "Cannot connect to the Docker daemon: raw-secret", "daemon", 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": tc.message})
			})
			_, err := e.InspectContainer(context.Background(), "fixture")
			if err == nil {
				t.Fatal("missing failure")
			}
			classified := engine.Classify(err)
			if classified.Category != tc.category {
				t.Errorf("category=%s want=%s", classified.Category, tc.category)
			}
			if tc.status == 404 && !errdefs.IsNotFound(err) {
				t.Error("lost SDK not-found unwrap")
			}
			for _, value := range []any{err, classified} {
				for _, format := range []string{"%v", "%+v", "%#v"} {
					if strings.Contains(fmt.Sprintf(format, value), "raw-secret") {
						t.Errorf("%s leaks raw response", format)
					}
				}
				b, _ := json.Marshal(value)
				if strings.Contains(string(b), "raw-secret") {
					t.Error("JSON leaks raw response")
				}
			}
		})
	}
	t.Run("cancellation", func(t *testing.T) {
		e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) { t.Error("cancelled request reached HTTP server") })
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := e.InspectContainer(ctx, "fixture")
		if !errors.Is(err, context.Canceled) || engine.Classify(err).Category != "cancelled" {
			t.Error("lost cancellation classification/unwrap")
		}
	})
}
