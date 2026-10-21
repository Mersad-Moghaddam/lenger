package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/core/store"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

const composeTimeout = 180 * time.Second

// ListProjects queries containers and auto-discovers compose projects (PRJ-01).
func (s *Service) ListProjects(ctx context.Context) ([]engine.ComposeProject, error) {
	ctx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	containers, err := s.engine.ListContainers(ctx)
	if err != nil {
		return nil, engine.Classify(err)
	}
	return store.GroupProjects(containers), nil
}

// Compose executes a lifecycle action on a Compose project with volume protection (PRJ-02..PRJ-05).
func (s *Service) Compose(ctx context.Context, action engine.ComposeAction, opts engine.ComposeOptions, emit func(string) error) error {
	if strings.TrimSpace(opts.ProjectName) == "" && strings.TrimSpace(opts.ProjectDir) == "" && len(opts.ConfigFiles) == 0 {
		return fmt.Errorf("compose requires a project name, directory, or compose file")
	}
	if opts.RemoveVolumes && action != engine.ComposeDown {
		return fmt.Errorf("volume removal is only permitted when running compose down")
	}

	ctx, cancel := context.WithTimeout(ctx, composeTimeout)
	defer cancel()

	if err := s.engine.Compose(ctx, action, opts, emit); err != nil {
		return engine.Classify(err)
	}
	return nil
}
