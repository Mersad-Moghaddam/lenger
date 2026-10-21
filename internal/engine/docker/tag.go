package docker

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/moby/moby/client"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// TagImage tags a local image with a new target reference (IMG-08).
func (e *Engine) TagImage(ctx context.Context, sourceID, targetRef string) error {
	if _, err := e.client.ImageTag(ctx, client.ImageTagOptions{Source: sourceID, Target: targetRef}); err != nil {
		return fmt.Errorf("tag image: %w", err)
	}
	return nil
}

// PushImage uploads an image to a remote registry with streamed progress (IMG-08).
func (e *Engine) PushImage(ctx context.Context, ref string, emit func(engine.Progress) error) error {
	auth, err := registryAuth(ctx, ref)
	if err != nil {
		return fmt.Errorf("resolve registry credentials: %w", err)
	}
	body, err := e.client.ImagePush(ctx, ref, client.ImagePushOptions{RegistryAuth: auth})
	if err != nil {
		return fmt.Errorf("push image: %w", err)
	}
	defer body.Close() //nolint:errcheck // The decoder/context error is authoritative.
	return DecodePull(ctx, body, emit)
}

// SaveImage exports an image archive to the specified tarball filepath (IMG-08).
func (e *Engine) SaveImage(ctx context.Context, id, targetFile string) error {
	reader, err := e.client.ImageSave(ctx, []string{id})
	if err != nil {
		return fmt.Errorf("export image: %w", err)
	}
	defer reader.Close() //nolint:errcheck

	f, err := os.OpenFile(targetFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create export file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, reader); err != nil {
		return fmt.Errorf("write image tar: %w", err)
	}
	return nil
}

// LoadImage imports an image archive from a tarball filepath (IMG-08).
func (e *Engine) LoadImage(ctx context.Context, sourceFile string) error {
	f, err := os.Open(sourceFile)
	if err != nil {
		return fmt.Errorf("open import file: %w", err)
	}
	defer func() { _ = f.Close() }()

	res, err := e.client.ImageLoad(ctx, f)
	if err != nil {
		return fmt.Errorf("import image: %w", err)
	}
	defer res.Close() //nolint:errcheck
	_, _ = io.Copy(io.Discard, res)
	return nil
}
