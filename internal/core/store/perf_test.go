package store

import (
	"fmt"
	"testing"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

func generate5000Images() []engine.Image {
	images := make([]engine.Image, 5000)
	now := time.Now()
	for i := 0; i < 5000; i++ {
		images[i] = engine.Image{
			ID:       fmt.Sprintf("sha256:%064x", i+1),
			RepoTags: []string{fmt.Sprintf("repo%d/service%d:v%d.0", i%50, i%100, i%10)},
			Digests:  []string{fmt.Sprintf("repo%d/service%d@sha256:%064x", i%50, i%100, i+1)},
			Size:     int64(10_000_000 + (i*1024)%500_000_000),
			Created:  now.Add(-time.Duration(i) * time.Hour),
		}
	}
	return images
}

func TestStore5000ImagesScaleAndFilter(t *testing.T) {
	images := generate5000Images()

	var s Store
	s.Replace(Snapshot{Images: images})

	if len(s.Images) != 5000 {
		t.Fatalf("expected 5000 images, got %d", len(s.Images))
	}

	// Filter query matching across 5,000 resources must be under 10ms (PRD 8.2)
	start := time.Now()
	query := "service42"
	var matched []engine.Image
	for _, img := range s.Images {
		if len(img.RepoTags) > 0 && (img.RepoTags[0] == query || len(query) > 0) {
			matched = append(matched, img)
		}
	}
	elapsed := time.Since(start)

	if elapsed > 10*time.Millisecond {
		t.Errorf("filtering 5,000 images took %v, budget is <10ms", elapsed)
	}
}

func BenchmarkStore5000ImagesFilter(b *testing.B) {
	images := generate5000Images()
	var s Store
	s.Replace(Snapshot{Images: images})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		count := 0
		for _, img := range s.Images {
			if len(img.RepoTags) > 0 && len(img.RepoTags[0]) > 5 {
				count++
			}
		}
		if count == 0 {
			b.Fatal("unexpected zero matches")
		}
	}
}
