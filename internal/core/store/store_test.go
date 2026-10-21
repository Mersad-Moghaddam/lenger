package store

import (
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"testing"
)

func TestReplaceBuildsVolumeUsageAndClearsOldIndex(t *testing.T) {
	var s Store
	s.Replace(Snapshot{
		Volumes: []engine.Volume{{Name: "data"}},
		Containers: []engine.Container{{
			ID: "c", ImageID: "a", Mounts: []engine.Mount{{Type: "volume", Name: "data", Destination: "/data", RW: true}},
		}},
	})
	if len(s.Volumes[0].UsedBy) != 1 || len(s.ByVolume["data"]) != 1 {
		t.Fatal("missing volume usage")
	}
	s.Replace(Snapshot{Volumes: []engine.Volume{{Name: "data"}}})
	if len(s.Volumes[0].UsedBy) != 0 || len(s.ByVolume) != 0 {
		t.Fatal("stale volume usage retained")
	}
}

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
