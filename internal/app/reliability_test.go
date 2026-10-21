package app

import (
	"testing"

	"github.com/Mersad-Moghaddam/lenger/internal/core/jobs"
	"github.com/Mersad-Moghaddam/lenger/internal/core/store"
)

// Regression: overlapping invalidations must trigger one follow-up fetch instead
// of discarding an event or launching competing snapshots.
func TestRefreshCoalescesInvalidations(t *testing.T) {
	m := fixtureModel(t)
	if m.refresh() == nil {
		t.Fatal("initial refresh did not produce a command")
	}
	if m.refresh() != nil {
		t.Fatal("overlapping refresh produced a competing command")
	}
	if !m.refreshPending {
		t.Fatal("invalidation was lost")
	}
	_, command := m.Update(snapshotMsg{data: store.Snapshot{}})
	if command == nil || m.refreshPending || !m.refreshing {
		t.Fatal("pending refresh was not consumed")
	}
}

// Regression: coalesced notifications can first expose a completed job, with no
// prior running snapshot. Completion must still invalidate image resources.
func TestFastPullCompletionTriggersRefresh(t *testing.T) {
	m := fixtureModel(t)
	m.Update(jobsMsg{{ID: 1, State: "succeeded"}})
	if !m.refreshing || !m.completedJobs[1] {
		t.Fatal("fast completion did not refresh resources")
	}
	m.refreshing = false
	m.Update(jobsMsg([]jobs.Job{{ID: 1, State: "succeeded"}}))
	if m.refreshing {
		t.Fatal("same completion refreshed resources repeatedly")
	}
}
