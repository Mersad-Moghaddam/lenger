package fake

import (
	"context"
	"testing"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

func TestStatsAvailability(t *testing.T) {
	e := New()
	containers, err := e.ListContainers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range containers {
		s, err := e.ContainerStats(context.Background(), c.ID)
		if err != nil {
			t.Fatal(err)
		}
		if c.Running() {
			if s.State != engine.TelemetryActive || !s.CPUAvailable || !s.MemoryAvailable || !s.NetworkAvailable || !s.BlockAvailable || !s.PIDsAvailable {
				t.Fatalf("missing fixture availability: %+v", s)
			}
		} else if s.State != engine.TelemetryUnavailable || s.MemoryAvailable {
			t.Fatalf("stopped fixture has measurements: %+v", s)
		}
	}
}

func TestPreviousLogsApplyBoundaryBeforeTail(t *testing.T) {
	e := New()
	containers, _ := e.ListContainers(context.Background())
	until := time.Date(2026, 10, 21, 11, 59, 40, 0, time.UTC)
	var entries []engine.LogEntry
	err := e.Logs(context.Background(), containers[0].ID, engine.LogOptions{Until: until, Previous: true, Follow: true, Tail: 5}, func(entry engine.LogEntry) error { entries = append(entries, entry); return nil })
	if err != nil || len(entries) != 5 {
		t.Fatalf("got %d entries, err=%v", len(entries), err)
	}
	for _, entry := range entries {
		if !entry.At.Before(until) {
			t.Fatal("historical logs cross restart boundary")
		}
	}
}
