package fake

import (
	"context"
	"errors"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"testing"
)

func TestPullCancellationDoesNotAddImage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	e := New()
	e.Delay = 0
	before, err := e.ListImages(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = e.Pull(ctx, "example:new", func(engine.Progress) error { cancel(); return ctx.Err() })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	after, err := e.ListImages(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != len(after) {
		t.Fatal("cancelled pull added image")
	}
}
func TestEventSubscriberCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := New().Events(ctx, func(engine.Event) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}
