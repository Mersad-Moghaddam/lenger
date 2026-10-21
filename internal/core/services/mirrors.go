package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/distribution/reference"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// RewriteMirrorReference adapts an image reference to use an alternative registry mirror (REG-02).
func RewriteMirrorReference(ref string, mirrorURL string) (string, error) {
	named, err := reference.ParseNormalizedNamed(ref)
	if err != nil {
		return "", fmt.Errorf("invalid reference: %w", err)
	}

	cleanMirror := strings.TrimSpace(mirrorURL)
	if u, err := url.Parse(cleanMirror); err == nil && u.Host != "" {
		cleanMirror = u.Host
	}
	cleanMirror = strings.TrimPrefix(cleanMirror, "http://")
	cleanMirror = strings.TrimPrefix(cleanMirror, "https://")
	cleanMirror = strings.TrimRight(cleanMirror, "/")

	path := reference.Path(named)
	var tagOrDigest string
	if tagged, ok := named.(reference.Tagged); ok {
		tagOrDigest = ":" + tagged.Tag()
	} else if digested, ok := named.(reference.Digested); ok {
		tagOrDigest = "@" + digested.Digest().String()
	} else {
		tagOrDigest = ":latest"
	}

	return fmt.Sprintf("%s/%s%s", cleanMirror, path, tagOrDigest), nil
}

// GenerateDaemonMirrorsDiff merges mirror URLs into /etc/docker/daemon.json and produces a unified diff (REG-03).
func GenerateDaemonMirrorsDiff(existingConfig string, mirrors []string) (string, string, error) {
	var cfg map[string]any
	trimmed := strings.TrimSpace(existingConfig)
	if trimmed == "" {
		cfg = make(map[string]any)
	} else {
		if err := json.Unmarshal([]byte(trimmed), &cfg); err != nil {
			return "", "", fmt.Errorf("parse existing daemon.json: %w", err)
		}
	}

	// Filter and clean mirror list
	cleaned := make([]string, 0, len(mirrors))
	for _, m := range mirrors {
		cm := strings.TrimSpace(m)
		if cm != "" {
			if !strings.HasPrefix(cm, "http://") && !strings.HasPrefix(cm, "https://") {
				cm = "https://" + cm
			}
			cleaned = append(cleaned, cm)
		}
	}

	cfg["registry-mirrors"] = cleaned
	newBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("format daemon.json: %w", err)
	}
	newContent := string(newBytes) + "\n"

	var sb strings.Builder
	sb.WriteString("--- /etc/docker/daemon.json (current)\n")
	sb.WriteString("+++ /etc/docker/daemon.json (proposed)\n")

	oldLines := strings.Split(trimmed, "\n")
	newLines := strings.Split(strings.TrimSpace(newContent), "\n")

	if trimmed == "" {
		for _, nl := range newLines {
			sb.WriteString("+ " + nl + "\n")
		}
	} else {
		for _, ol := range oldLines {
			sb.WriteString("- " + ol + "\n")
		}
		for _, nl := range newLines {
			sb.WriteString("+ " + nl + "\n")
		}
	}

	return sb.String(), newContent, nil
}

// MirrorPull attempts pulling ref directly; falls back to configured mirrors if blocked (REG-02).
func (s *Service) MirrorPull(ctx context.Context, ref string, mirrors []string, emit func(engine.Progress) error) error {
	pullErr := s.engine.Pull(ctx, ref, emit)
	if pullErr == nil {
		return nil
	}

	// If direct pull failed, attempt configured mirrors
	if len(mirrors) == 0 {
		return pullErr
	}

	lastErr := pullErr
	for _, mirror := range mirrors {
		rewritten, err := RewriteMirrorReference(ref, mirror)
		if err != nil {
			continue
		}
		_ = emit(engine.Progress{
			ID:     "mirror",
			Status: fmt.Sprintf("Direct pull failed (%v). Trying mirror: %s", pullErr, mirror),
		})

		if err := s.engine.Pull(ctx, rewritten, emit); err == nil {
			// Tag rewritten image back to requested canonical reference
			_ = emit(engine.Progress{
				ID:     "retag",
				Status: fmt.Sprintf("Retagging %s -> %s", rewritten, ref),
			})
			_ = s.engine.TagImage(ctx, rewritten, ref)
			return nil
		}
		lastErr = err
	}
	return fmt.Errorf("pull failed across direct registry and all %d mirrors: %w", len(mirrors), lastErr)
}
