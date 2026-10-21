package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/engine/fake"
)

type removalEngine struct {
	engine.Engine
	hasDeadline      bool
	removed          bool
	cancelAfterCheck context.CancelFunc
}

func (e *removalEngine) ListContainers(ctx context.Context) ([]engine.Container, error) {
	_, e.hasDeadline = ctx.Deadline()
	if e.cancelAfterCheck != nil {
		e.cancelAfterCheck()
	}
	return nil, nil
}

func TestRemovalCancellationAfterReferenceCheckPreventsDeletion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e := &removalEngine{cancelAfterCheck: cancel}
	result := New(e).Remove(ctx, []string{"image"})
	if !errors.Is(result["image"], context.Canceled) || e.removed {
		t.Fatal("removal continued after cancellation during reference checking")
	}
}

func (e *removalEngine) RemoveImage(ctx context.Context, _ string) error {
	e.removed = true
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("missing operation deadline")
	}
	return errors.New("permission denied: sensitive adapter detail")
}

func TestRemovalBoundsIOAndClassifiesAdapterErrors(t *testing.T) {
	e := &removalEngine{}
	result := New(e).Remove(context.Background(), []string{"image"})
	var classified *engine.ClassifiedError
	if !e.hasDeadline || !e.removed || !errors.As(result["image"], &classified) {
		t.Fatal("removal did not enforce its I/O and error boundary")
	}
	if strings.Contains(result["image"].Error(), "sensitive") {
		t.Fatal("raw adapter details escaped removal policy")
	}
}

func TestCancelledRemovalNeverCallsAdapter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	e := &removalEngine{}
	result := New(e).Remove(ctx, []string{"first", "second"})
	for _, id := range []string{"first", "second"} {
		if !errors.Is(result[id], context.Canceled) {
			t.Fatalf("%s lost cancellation", id)
		}
	}
	if e.hasDeadline || e.removed {
		t.Fatal("cancelled removal reached the adapter")
	}
}

func TestNormalizeReference(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		bad         bool
	}{{"redis", "redis:latest", false}, {"localhost:5000/team/app:v1", "localhost:5000/team/app:v1", false}, {"", "", true}, {"bad name", "", true}, {" redis", "", true}} {
		t.Run(tc.input, func(t *testing.T) {
			got, err := NormalizeReference(tc.input)
			if (err != nil) != tc.bad || got != tc.want {
				t.Fatalf("got %q, %v", got, err)
			}
		})
	}
}

func TestInspectMasksSecrets(t *testing.T) {
	ctx := context.Background()
	s := New(fake.New())
	batch, err := s.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	i, err := s.Inspect(ctx, batch.Images[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range i.Env {
		if strings.Contains(v, "fixture-secret") || !strings.HasSuffix(v, "=••••") {
			t.Fatalf("unmasked environment: %s", v)
		}
	}
	for _, v := range i.Labels {
		if v != "••••" {
			t.Fatal("unmasked label")
		}
	}
}

func TestRemoveInUseImageRefused(t *testing.T) {
	ctx := context.Background()
	s := New(fake.New())
	batch, err := s.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	id := batch.Images[0].ID
	if s.Remove(ctx, []string{id})[id] == nil {
		t.Fatal("in-use image was removed")
	}
	after, err := s.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Images) != len(batch.Images) {
		t.Fatal("image count changed")
	}
}
