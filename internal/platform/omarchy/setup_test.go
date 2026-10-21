package omarchy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectTerminal(t *testing.T) {
	// Should return a non-empty terminal emulator string
	term := DetectTerminal()
	if term == "" {
		t.Fatal("expected non-empty terminal emulator")
	}
}

func TestSetupOutput(t *testing.T) {
	// Non-apply run
	out := Setup(false)
	if !strings.Contains(out, "Lenger · Omarchy Desktop Keybinding Setup") {
		t.Fatalf("expected header in setup output: %s", out)
	}
	if !strings.Contains(out, "$mainMod ALT, D") {
		t.Fatalf("expected recommended keybinding $mainMod ALT, D in output: %s", out)
	}
	if !strings.Contains(out, "No files were modified") {
		t.Fatalf("expected notice that no files were modified: %s", out)
	}
}

func TestSetupApply(t *testing.T) {
	tmpDir := t.TempDir()
	hyprDir := filepath.Join(tmpDir, ".config", "hypr")
	if err := os.MkdirAll(hyprDir, 0o755); err != nil {
		t.Fatalf("failed to create test hypr dir: %v", err)
	}
	configFile := filepath.Join(hyprDir, "bindings.conf")
	initialContent := "# Existing hyprland bindings\nbind = $mainMod SHIFT, D, exec, foot -e lazydocker\n"
	if err := os.WriteFile(configFile, []byte(initialContent), 0o644); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}

	t.Setenv("HOME", tmpDir)

	// Run Setup(true)
	out := Setup(true)
	if !strings.Contains(out, "Successfully added keybinding") {
		t.Fatalf("expected success message: %s", out)
	}

	// Read back file
	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("failed to read back config: %v", err)
	}
	newContent := string(data)
	if !strings.Contains(newContent, "bind = $mainMod ALT, D, exec") {
		t.Fatalf("expected binding line in file: %s", newContent)
	}
	if !strings.Contains(newContent, "lazydocker") {
		t.Fatal("expected original content to be preserved")
	}

	// Running Setup again should detect it is already configured
	secondOut := Setup(true)
	if !strings.Contains(secondOut, "already configured") {
		t.Fatalf("expected 'already configured' notice on second run: %s", secondOut)
	}
}
