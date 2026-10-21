package store

import (
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"testing"
)

func TestReplaceBuildsUsageAndClearsOldIndex(t *testing.T) {
	var s Store
	s.Replace(Snapshot{Images: []engine.Image{{ID: "a"}}, Containers: []engine.Container{{ID: "c", ImageID: "a"}}})
	if len(s.Images[0].UsedBy) != 1 {
		t.Fatal("missing usage")
	}
	s.Replace(Snapshot{Images: []engine.Image{{ID: "a"}}})
	if len(s.Images[0].UsedBy) != 0 || len(s.ByImage) != 0 {
		t.Fatal("stale usage retained")
	}
}
