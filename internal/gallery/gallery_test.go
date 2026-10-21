package gallery

import (
	"charm.land/lipgloss/v2"
	"context"
	"flag"
	"fmt"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/theme"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "explicitly update reviewed gallery goldens")

func TestDrawerFocusFrameDistinct(t *testing.T) {
	frames, err := Frames(context.Background(), "midnight", Size{120, 40})
	if err != nil {
		t.Fatal(err)
	}
	var split, focus string
	for _, f := range frames {
		if f.Name == "drawer-split" {
			split = f.Content
		}
		if f.Name == "drawer-focus" {
			focus = f.Content
		}
	}
	if split == focus {
		t.Fatal("focus frame duplicates split state")
	}
	if !strings.Contains(focus, "> 2. Restart container") {
		t.Fatal("focus frame lacks distinct Health control selection")
	}
}

func TestResourceFrames(t *testing.T) {
	frames, err := Frames(context.Background(), theme.Names()[0], Size{120, 40})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"containers": "Containers", "images": "Images", "volumes": "Volumes", "networks": "Networks"}
	for _, frame := range frames {
		if frame.Name == "dashboard" || frame.Name == "projects" || frame.Name == "system" {
			t.Errorf("removed resource view remains: %s", frame.Name)
		}
		if title, ok := want[frame.Name]; ok {
			lines := strings.Split(frame.Content, "\n")
			if !strings.Contains(strings.ToLower(strings.Join(lines[2:5], "\n")), strings.ToLower(title)) {
				t.Errorf("%s frame does not render its resource panel", frame.Name)
			}
			delete(want, frame.Name)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing resource frames: %v", want)
	}
}

func TestLayout(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	for _, name := range theme.Names() {
		for _, size := range Sizes() {
			t.Run(fmt.Sprintf("%s/%dx%d", name, size.Width, size.Height), func(t *testing.T) {
				frames, err := Frames(context.Background(), name, size)
				if err != nil {
					t.Fatal(err)
				}
				for _, frame := range frames {
					lines := strings.Split(frame.Content, "\n")
					if len(lines) != size.Height {
						t.Errorf("%s: %d lines, want %d", frame.Name, len(lines), size.Height)
					}
					for n, line := range lines {
						if lipgloss.Width(line) > size.Width {
							t.Errorf("%s line %d overflows", frame.Name, n)
						}
					}
				}
			})
		}
	}
}

func TestGolden(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	for _, name := range theme.Names() {
		for _, size := range Sizes() {
			frames, err := Frames(context.Background(), name, size)
			if err != nil {
				t.Fatal(err)
			}
			for _, frame := range frames {
				path := filepath.Join("..", "..", "testdata", "golden", fmt.Sprintf("%s-%dx%d-%s.txt", name, size.Width, size.Height, frame.Name))
				if *update {
					if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, []byte(frame.Content), 0644); err != nil {
						t.Fatal(err)
					}
				}
				want, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("golden missing: %s; render/review first, then use -update: %v", path, err)
				}
				if string(want) != frame.Content {
					t.Errorf("golden mismatch: %s", path)
				}
			}
		}
	}
}
