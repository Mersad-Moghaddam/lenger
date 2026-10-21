// Package gallery renders deterministic offline screens without starting a TUI.
package gallery

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"fmt"
	"github.com/Mersad-Moghaddam/lenger/internal/app"
	"github.com/Mersad-Moghaddam/lenger/internal/config"
	"github.com/Mersad-Moghaddam/lenger/internal/core/services"
	"github.com/Mersad-Moghaddam/lenger/internal/engine/fake"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/theme"
	"io"
	"strconv"
	"strings"
)

// Size records a terminal fixture size.
type Size struct{ Width, Height int }

// Sizes returns the PRD's four reference viewports.
func Sizes() []Size { return []Size{{80, 24}, {120, 40}, {160, 45}, {200, 60}} }

// ParseSize accepts WIDTHxHEIGHT and rejects nonsensical or unbounded allocation.
func ParseSize(value string) (Size, error) {
	w, h, ok := strings.Cut(value, "x")
	if !ok {
		return Size{}, fmt.Errorf("size must be WIDTHxHEIGHT")
	}
	width, e1 := strconv.Atoi(w)
	height, e2 := strconv.Atoi(h)
	if e1 != nil || e2 != nil || width < 1 || height < 1 || width > 500 || height > 200 {
		return Size{}, fmt.Errorf("size must be between 1x1 and 500x200")
	}
	return Size{width, height}, nil
}

// Frame names one independent screen snapshot.
type Frame struct{ Name, Content string }

// Frames renders all screens and overlays using the production model renderer.
func Frames(ctx context.Context, name string, size Size) ([]Frame, error) {
	p, err := theme.Bundled(name)
	if err != nil {
		return nil, err
	}
	e := fake.New()
	svc := services.New(e)
	batch, err := svc.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	inspection, err := svc.Inspect(ctx, batch.Images[0].ID)
	if err != nil {
		return nil, err
	}
	type screen struct {
		name    string
		tab     int
		overlay string
	}
	screens := []screen{
		{"dashboard", 0, ""}, {"images", 1, ""}, {"containers", 2, ""}, {"projects", 3, ""}, {"volumes", 4, ""}, {"networks", 5, ""}, {"system", 6, ""},
		{"pull", 1, "pull"}, {"palette", 1, "palette"}, {"help", 0, "help"}, {"removal", 1, "remove"}, {"jobs", 1, "jobs"},
		{"run", 1, "run"}, {"logs", 2, "logs"}, {"cleanup", 6, "cleanup"}, {"container-action", 2, "container-action"}, {"exec", 2, "exec"}, {"volume-removal", 4, "remove-volume"},
		{"compose-action", 3, "compose-action"}, {"create-network", 5, "create-network"}, {"remove-network", 5, "remove-network"},
		{"registry-search", 1, "registry-search"}, {"tag-image", 1, "tag-image"}, {"push-image", 1, "push-image"},
		{"export-image", 1, "export-image"}, {"import-image", 1, "import-image"}, {"daemon-mirrors", 1, "daemon-mirrors"},
	}
	frames := make([]Frame, 0, len(screens))
	for _, screen := range screens {
		c := config.Defaults()
		c.UI.Theme = name
		c.UI.ReduceMotion = true
		c.UI.ASCII = true
		m := app.New(ctx, svc, c, p)
		_, _ = m.Update(tea.WindowSizeMsg{Width: size.Width, Height: size.Height})
		m.Gallery(batch, screen.tab, screen.overlay, &inspection)
		frames = append(frames, Frame{screen.name, m.Render()})
	}
	return frames, nil
}

// Render emits a labeled matrix or a selected theme/size, without Docker access.
func Render(ctx context.Context, out io.Writer, name, sizeValue string) error {
	names := theme.Names()
	if name != "" {
		names = []string{name}
	}
	sizes := Sizes()
	if sizeValue != "" {
		size, err := ParseSize(sizeValue)
		if err != nil {
			return err
		}
		sizes = []Size{size}
	}
	for _, name := range names {
		for _, size := range sizes {
			frames, err := Frames(ctx, name, size)
			if err != nil {
				return err
			}
			for _, frame := range frames {
				if _, err := fmt.Fprintf(out, "\n=== %s / %dx%d / %s ===\n%s\n", name, size.Width, size.Height, frame.Name, frame.Content); err != nil {
					return fmt.Errorf("write gallery: %w", err)
				}
			}
		}
	}
	return nil
}
