package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// Compose executes docker compose CLI commands with streamed output (PRJ-01..PRJ-05).
func (e *Engine) Compose(ctx context.Context, action engine.ComposeAction, opts engine.ComposeOptions, emit func(string) error) error {
	args := []string{"compose"}
	if opts.ProjectDir != "" {
		args = append(args, "--project-directory", opts.ProjectDir)
	}
	for _, cfg := range opts.ConfigFiles {
		if strings.TrimSpace(cfg) != "" {
			args = append(args, "-f", strings.TrimSpace(cfg))
		}
	}
	if opts.ProjectName != "" {
		args = append(args, "-p", opts.ProjectName)
	}

	switch action {
	case engine.ComposeUp:
		args = append(args, "up", "-d")
		args = append(args, opts.Services...)
	case engine.ComposeDown:
		args = append(args, "down")
		if opts.RemoveVolumes {
			args = append(args, "-v")
		}
	case engine.ComposeRestart:
		args = append(args, "restart")
		args = append(args, opts.Services...)
	case engine.ComposePull:
		args = append(args, "pull")
		args = append(args, opts.Services...)
	default:
		return fmt.Errorf("unsupported compose action: %s", action)
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	if opts.ProjectDir != "" {
		cmd.Dir = opts.ProjectDir
	}

	reader, writer := io.Pipe()
	cmd.Stdout = writer
	cmd.Stderr = writer

	scanDone := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			if emit != nil {
				if err := emit(line); err != nil {
					_ = reader.CloseWithError(err)
					scanDone <- err
					return
				}
			}
		}
		scanDone <- scanner.Err()
	}()

	startErr := cmd.Start()
	if startErr != nil {
		_ = writer.Close()
		return fmt.Errorf("execute docker compose: %w", startErr)
	}

	cmdErr := cmd.Wait()
	_ = writer.Close()
	<-scanDone

	if cmdErr != nil {
		return fmt.Errorf("compose %s failed: %w", action, cmdErr)
	}
	return nil
}
