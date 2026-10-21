package views

import (
	"fmt"
	"testing"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/theme"
)

func generate5000ImagesFixture() []engine.Image {
	images := make([]engine.Image, 5000)
	now := time.Now()
	for i := 0; i < 5000; i++ {
		images[i] = engine.Image{
			ID:       fmt.Sprintf("sha256:%064x", i+1),
			RepoTags: []string{fmt.Sprintf("service%d:v%d.0", i%100, i%10)},
			Size:     int64(15_000_000 + (i*1024)%200_000_000),
			Created:  now.Add(-time.Duration(i) * time.Hour),
		}
	}
	return images
}

func TestImageListWindowing5000Images(t *testing.T) {
	p, _ := theme.Bundled("omarchy")
	s := style.Sheet{Palette: p, Plain: true, ASCII: true}
	images := generate5000ImagesFixture()
	selected := make(map[string]bool)

	// Test render at cursor in middle of 5,000 items
	start := time.Now()
	out := ImageList(s, images, 2500, selected, "name", "", 120, 40, time.Now())
	elapsed := time.Since(start)

	if out == "" {
		t.Fatal("expected non-empty rendered view")
	}

	// 60fps budget is 16.6ms per frame (PRD 8.2)
	if elapsed > 16*time.Millisecond {
		t.Errorf("rendering 5,000 image window took %v, budget is <16ms", elapsed)
	}
}

func BenchmarkImageListWindowing5000Images(b *testing.B) {
	p, _ := theme.Bundled("omarchy")
	s := style.Sheet{Palette: p, Plain: true, ASCII: true}
	images := generate5000ImagesFixture()
	selected := make(map[string]bool)
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cursor := (i * 37) % 5000
		_ = ImageList(s, images, cursor, selected, "name", "", 120, 40, now)
	}
}
