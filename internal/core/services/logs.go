package services

import (
	"context"
	"fmt"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// Logs bounds the initial tail and lets follow streams live until cancellation.
func (s *Service) Logs(ctx context.Context, id string, opts engine.LogOptions, emit func(engine.LogEntry) error) error {
	if id == "" {
		return fmt.Errorf("container ID is required")
	}
	if opts.Tail <= 0 {
		opts.Tail = 500
	}
	if opts.Tail > 50_000 {
		opts.Tail = 50_000
	}
	if !opts.Follow {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, usageTimeout)
		defer cancel()
	}
	if err := s.engine.Logs(ctx, id, opts, emit); err != nil {
		return engine.Classify(err)
	}
	return nil
}
