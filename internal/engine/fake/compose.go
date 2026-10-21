package fake

import (
	"context"
	"fmt"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// Compose simulates executing docker compose actions with realistic log streams (PRJ-01..PRJ-05).
func (e *Engine) Compose(ctx context.Context, action engine.ComposeAction, opts engine.ComposeOptions, emit func(string) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	projectName := opts.ProjectName
	if projectName == "" {
		projectName = "storefront"
	}

	lines := []string{
		fmt.Sprintf("[+] Running compose %s on project %s", action, projectName),
	}

	switch action {
	case engine.ComposeUp:
		lines = append(lines,
			" Network storefront_default  Created",
			" Container storefront-db-1  Starting",
			" Container storefront-cache-1  Starting",
			" Container storefront-web-1  Starting",
			" Container storefront-db-1  Started",
			" Container storefront-cache-1  Started",
			" Container storefront-web-1  Started",
		)
	case engine.ComposeDown:
		lines = append(lines,
			" Container storefront-web-1  Stopping",
			" Container storefront-cache-1  Stopping",
			" Container storefront-db-1  Stopping",
			" Container storefront-web-1  Stopped",
			" Container storefront-cache-1  Stopped",
			" Container storefront-db-1  Stopped",
			" Container storefront-web-1  Removed",
			" Container storefront-cache-1  Removed",
			" Container storefront-db-1  Removed",
			" Network storefront_default  Removed",
		)
		if opts.RemoveVolumes {
			lines = append(lines, " Volume storefront_web-content  Removed", " Volume storefront_postgres-data  Removed")
		}
	case engine.ComposeRestart:
		lines = append(lines,
			" Container storefront-web-1  Restarting",
			" Container storefront-cache-1  Restarting",
			" Container storefront-db-1  Restarting",
			" Container storefront-web-1  Started",
			" Container storefront-cache-1  Started",
			" Container storefront-db-1  Started",
		)
	case engine.ComposePull:
		lines = append(lines,
			" db Pulling fs layer",
			" web Pulling fs layer",
			" cache Pulling fs layer",
			" db Download complete",
			" web Download complete",
			" cache Download complete",
		)
	default:
		return fmt.Errorf("unsupported compose action: %s", action)
	}

	for _, line := range lines {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if emit != nil {
			if err := emit(line); err != nil {
				return err
			}
		}
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	for i := range e.containers {
		if e.containers[i].Project == projectName {
			switch action {
			case engine.ComposeUp, engine.ComposeRestart:
				e.containers[i].State = "running"
				e.containers[i].Status = "Up less than a second"
			case engine.ComposeDown:
				e.containers[i].State = "exited"
				e.containers[i].Status = "Exited (0)"
			}
		}
	}
	e.broadcastLocked(engine.Event{Kind: "compose", Action: string(action), ID: projectName, At: time.Now().UTC()})

	return nil
}
