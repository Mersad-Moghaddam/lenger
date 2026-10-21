package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// DecodePull parses progress and embedded errors; malformed streams never succeed.
func DecodePull(ctx context.Context, r io.Reader, emit func(engine.Progress) error) error {
	decoder := json.NewDecoder(r)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		var msg struct {
			ID          string `json:"id"`
			Status      string `json:"status"`
			Error       string `json:"error"`
			ErrorDetail struct {
				Message string `json:"message"`
			} `json:"errorDetail"`
			Progress struct{ Current, Total int64 } `json:"progressDetail"`
		}
		if err := decoder.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("decode pull progress: %w", err)
		}
		if msg.Error != "" {
			return engine.Classify(errors.New(msg.Error))
		}
		if msg.ErrorDetail.Message != "" {
			return engine.Classify(errors.New(msg.ErrorDetail.Message))
		}
		if msg.ID == "" {
			continue
		}
		if err := emit(engine.Progress{ID: msg.ID, Status: msg.Status, Current: msg.Progress.Current, Total: msg.Progress.Total}); err != nil {
			return err
		}
	}
}
