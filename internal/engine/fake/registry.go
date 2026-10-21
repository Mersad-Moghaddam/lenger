package fake

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// SearchRegistry simulates remote registry searches with deterministic fixtures (IMG-06).
func (e *Engine) SearchRegistry(ctx context.Context, query string) ([]engine.RegistrySearchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	all := []engine.RegistrySearchResult{
		{Name: "nginx", Description: "Official build of Nginx.", Stars: 20100, Official: true, Automated: false},
		{Name: "postgres", Description: "The PostgreSQL object-relational database system.", Stars: 14800, Official: true, Automated: false},
		{Name: "redis", Description: "Redis is an open source key-value store that functions as a data structure server.", Stars: 12500, Official: true, Automated: false},
		{Name: "alpine", Description: "A minimal Docker image based on Alpine Linux with a complete package index.", Stars: 10400, Official: true, Automated: false},
		{Name: "node", Description: "Node.js JavaScript runtime environment.", Stars: 13900, Official: true, Automated: false},
		{Name: "golang", Description: "Go (golang) is a general purpose, higher-level, imperative programming language.", Stars: 12100, Official: true, Automated: false},
		{Name: "bitnami/nginx", Description: "Bitnami container image for NGINX.", Stars: 430, Official: false, Automated: true},
		{Name: "bitnami/postgresql", Description: "Bitnami container image for PostgreSQL.", Stars: 380, Official: false, Automated: true},
		{Name: "bitnami/redis", Description: "Bitnami container image for Redis.", Stars: 320, Official: false, Automated: true},
	}
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return all, nil
	}
	var matched []engine.RegistrySearchResult
	for _, res := range all {
		if strings.Contains(strings.ToLower(res.Name), q) || strings.Contains(strings.ToLower(res.Description), q) {
			matched = append(matched, res)
		}
	}
	return matched, nil
}

// ListRemoteTags returns published tags for a repository fixture (IMG-06).
func (e *Engine) ListRemoteTags(ctx context.Context, repo string) ([]engine.RemoteTag, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	cleanRepo := strings.TrimPrefix(repo, "library/")
	switch cleanRepo {
	case "nginx":
		return []engine.RemoteTag{
			{Name: "latest", Digest: "sha256:e0b68630386ff018593c72b22105c317b6a4a6e382b6c7a7bbad373d3289069d", Size: 48 * 1000 * 1000, UpdatedAt: now.Add(-2 * time.Hour)},
			{Name: "1.27-alpine", Digest: "sha256:7bc59b9a67a030f73bdfd80766be1b4e3e37130a08e9a26be72df292f397f259", Size: 44 * 1000 * 1000, UpdatedAt: now.Add(-24 * time.Hour)},
			{Name: "1.27", Digest: "sha256:a649361eb2828b6d801eeec5a6064f7756fcfab02ecf2cbbd49b251219ee94e6", Size: 142 * 1000 * 1000, UpdatedAt: now.Add(-48 * time.Hour)},
			{Name: "mainline-alpine", Digest: "sha256:c221a97561f71f65f02c676d3d9d300eb231bbd9c7e0fce59560f89895240292", Size: 45 * 1000 * 1000, UpdatedAt: now.Add(-12 * time.Hour)},
			{Name: "alpine-slim", Digest: "sha256:f01a084ef19a16f1a8e100e4e67f08cfb5c00e12d1b54f9a35e26715fbc74d82", Size: 18 * 1000 * 1000, UpdatedAt: now.Add(-96 * time.Hour)},
		}, nil
	case "postgres":
		return []engine.RemoteTag{
			{Name: "latest", Digest: "sha256:91bc8601c7d23a1f4b00a0680cb70e1762c4a961f7db12ee1824987ba8d2345e", Size: 432 * 1000 * 1000, UpdatedAt: now.Add(-10 * time.Hour)},
			{Name: "16-alpine", Digest: "sha256:1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b", Size: 120 * 1000 * 1000, UpdatedAt: now.Add(-30 * time.Hour)},
			{Name: "16", Digest: "sha256:8b9a0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b", Size: 430 * 1000 * 1000, UpdatedAt: now.Add(-72 * time.Hour)},
		}, nil
	default:
		return []engine.RemoteTag{
			{Name: "latest", Digest: "sha256:d3b07384d113edec49eaa6238ad5ff00", Size: 100 * 1000 * 1000, UpdatedAt: now.Add(-1 * time.Hour)},
			{Name: "stable", Digest: "sha256:a1b2c3d4e5f60718293a4b5c6d7e8f90", Size: 95 * 1000 * 1000, UpdatedAt: now.Add(-24 * time.Hour)},
		}, nil
	}
}

// CheckImageUpdate checks if an image's remote tag digest has diverged from local (IMG-09).
func (e *Engine) CheckImageUpdate(ctx context.Context, ref, localDigest string) (engine.ImageUpdateStatus, error) {
	if err := ctx.Err(); err != nil {
		return engine.ImageUpdateStatus{}, err
	}
	now := time.Now()
	// Deterministic mock remote digest
	remoteDigest := "sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"
	hasUpdate := false
	if strings.Contains(ref, "nginx") || strings.Contains(ref, "latest") {
		hasUpdate = true
	} else if localDigest != "" && !strings.Contains(localDigest, remoteDigest) {
		hasUpdate = true
	}
	return engine.ImageUpdateStatus{
		Ref:          ref,
		LocalDigest:  localDigest,
		RemoteDigest: remoteDigest,
		HasUpdate:    hasUpdate,
		CheckedAt:    now,
	}, nil
}

// TagImage tags an existing image in the fixture store (IMG-08).
func (e *Engine) TagImage(ctx context.Context, sourceID, targetRef string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	idx := slices.IndexFunc(e.images, func(img engine.Image) bool {
		return img.ID == sourceID || slices.Contains(img.RepoTags, sourceID)
	})
	if idx < 0 {
		return fmt.Errorf("source image not found: %s", sourceID)
	}
	if !slices.Contains(e.images[idx].RepoTags, targetRef) {
		e.images[idx].RepoTags = append(e.images[idx].RepoTags, targetRef)
	}
	return nil
}

// PushImage simulates remote registry push with streamed progress (IMG-08).
func (e *Engine) PushImage(ctx context.Context, _ string, emit func(engine.Progress) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	layers := []string{"layer-1", "layer-2", "layer-3"}
	total := int64(10 * 1024 * 1024)
	for _, l := range layers {
		if err := emit(engine.Progress{ID: l, Status: "Preparing", Current: 0, Total: total}); err != nil {
			return err
		}
		if e.Delay > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(e.Delay / 2):
			}
		}
		if err := emit(engine.Progress{ID: l, Status: "Pushing", Current: total / 2, Total: total}); err != nil {
			return err
		}
		if err := emit(engine.Progress{ID: l, Status: "Pushed", Current: total, Total: total}); err != nil {
			return err
		}
	}
	return emit(engine.Progress{ID: "", Status: "latest: digest: sha256:pushdigest size: 10485760", Current: 0, Total: 0})
}

// SaveImage exports a dummy tar archive to targetFile (IMG-08).
func (e *Engine) SaveImage(ctx context.Context, id, targetFile string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Create or overwrite file with dummy tarball content
	data := []byte("LENGER_FIXTURE_IMAGE_TAR:" + id + "\n")
	if err := os.WriteFile(targetFile, data, 0o644); err != nil {
		return fmt.Errorf("write export file: %w", err)
	}
	return nil
}

// LoadImage imports an image from sourceFile into fixture store (IMG-08).
func (e *Engine) LoadImage(ctx context.Context, sourceFile string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := os.ReadFile(sourceFile)
	if err != nil {
		return fmt.Errorf("read import file: %w", err)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	newID := fmt.Sprintf("sha256:imported%x", len(data))
	e.images = append(e.images, engine.Image{
		ID:       newID,
		RepoTags: []string{"imported/image:latest"},
		Size:     int64(len(data)) * 1024,
		Created:  time.Now(),
	})
	return nil
}
