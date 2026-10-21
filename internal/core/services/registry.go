package services

import (
	"context"
	"fmt"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// SearchRegistry queries remote container registries (IMG-06).
func (s *Service) SearchRegistry(ctx context.Context, query string) ([]engine.RegistrySearchResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	results, err := s.engine.SearchRegistry(ctx, query)
	if err != nil {
		return nil, engine.Classify(err)
	}
	return results, nil
}

// ListRemoteTags retrieves remote repository tags (IMG-06).
func (s *Service) ListRemoteTags(ctx context.Context, repo string) ([]engine.RemoteTag, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	tags, err := s.engine.ListRemoteTags(ctx, repo)
	if err != nil {
		return nil, engine.Classify(err)
	}
	return tags, nil
}

// CheckUpdate queries remote manifest to see if a newer version exists (IMG-09).
func (s *Service) CheckUpdate(ctx context.Context, ref, localDigest string) (engine.ImageUpdateStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	status, err := s.engine.CheckImageUpdate(ctx, ref, localDigest)
	if err != nil {
		return status, engine.Classify(err)
	}
	return status, nil
}

// TagImage creates an alias tag for an image (IMG-08).
func (s *Service) TagImage(ctx context.Context, sourceID, targetRef string) error {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	if err := s.engine.TagImage(ctx, sourceID, targetRef); err != nil {
		return engine.Classify(fmt.Errorf("tag image: %w", err))
	}
	return nil
}

// PushImage pushes an image to a remote registry with progress (IMG-08).
func (s *Service) PushImage(ctx context.Context, ref string, emit func(engine.Progress) error) error {
	ctx, cancel := context.WithTimeout(ctx, runTimeout*2)
	defer cancel()
	if err := s.engine.PushImage(ctx, ref, emit); err != nil {
		return engine.Classify(fmt.Errorf("push image: %w", err))
	}
	return nil
}

// SaveImage exports an image tarball to the destination path (IMG-08).
func (s *Service) SaveImage(ctx context.Context, id, targetFile string) error {
	ctx, cancel := context.WithTimeout(ctx, runTimeout*2)
	defer cancel()
	if err := s.engine.SaveImage(ctx, id, targetFile); err != nil {
		return engine.Classify(fmt.Errorf("export image: %w", err))
	}
	return nil
}

// LoadImage imports an image archive from a tarball file (IMG-08).
func (s *Service) LoadImage(ctx context.Context, sourceFile string) error {
	ctx, cancel := context.WithTimeout(ctx, runTimeout*2)
	defer cancel()
	if err := s.engine.LoadImage(ctx, sourceFile); err != nil {
		return engine.Classify(fmt.Errorf("import image: %w", err))
	}
	return nil
}
