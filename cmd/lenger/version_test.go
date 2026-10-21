package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Exercise the built CLI: version must not initialize config, Docker, or a TUI.
func TestVersionOffline(t *testing.T) {
	for _, tc := range []struct {
		name, ldflags, want string
	}{
		{"development", "", "lenger dev (commit unknown, built unknown)\n"},
		{"release", "-X main.version=0.2.0 -X main.commit=abc123 -X main.date=2026-10-21T00:00:00Z", "lenger 0.2.0 (commit abc123, built 2026-10-21T00:00:00Z)\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			binary := filepath.Join(dir, "lenger")
			build := exec.Command("go", "build", "-ldflags="+tc.ldflags, "-o", binary, ".")
			if output, err := build.CombinedOutput(); err != nil {
				t.Fatalf("build: %v\n%s", err, output)
			}
			configDir := filepath.Join(dir, "config")
			if err := os.MkdirAll(filepath.Join(configDir, "lenger"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(configDir, "lenger", "config.toml"), []byte("invalid = ["), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(binary, "--version")
			// Replace rather than duplicate inherited variables.
			for _, entry := range os.Environ() {
				if !strings.HasPrefix(entry, "XDG_CONFIG_HOME=") && !strings.HasPrefix(entry, "DOCKER_HOST=") && !strings.HasPrefix(entry, "DOCKER_CONTEXT=") && !strings.HasPrefix(entry, "LENGER_ENGINE=") {
					cmd.Env = append(cmd.Env, entry)
				}
			}
			cmd.Env = append(cmd.Env, "XDG_CONFIG_HOME="+configDir, "DOCKER_HOST=invalid://unreachable", "DOCKER_CONTEXT=unavailable", "LENGER_ENGINE=invalid")
			output, err := cmd.CombinedOutput()
			if err != nil || string(output) != tc.want {
				t.Fatalf("--version: err=%v output=%q; want %q", err, output, tc.want)
			}
		})
	}
}
