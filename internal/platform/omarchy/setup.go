package omarchy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DetectTerminal returns the preferred Omarchy terminal emulator found in PATH.
func DetectTerminal() string {
	if term := os.Getenv("TERMINAL"); term != "" {
		if _, err := exec.LookPath(term); err == nil {
			return term
		}
	}
	candidates := []string{"ghostty", "foot", "kitty", "alacritty"}
	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			return c
		}
	}
	return "xterm"
}

// FindBindingsConfig locates the active Hyprland bindings configuration file.
func FindBindingsConfig() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	candidates := []string{
		filepath.Join(home, ".config", "hypr", "bindings.lua"),
		filepath.Join(home, ".config", "hypr", "bindings.conf"),
		filepath.Join(home, ".config", "hypr", "hyprland.lua"),
		filepath.Join(home, ".config", "hypr", "hyprland.conf"),
		filepath.Join(home, ".config", "omarchy", "hypr", "bindings.conf"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join(home, ".config", "hypr", "bindings.conf")
}

// Setup generates or applies Hyprland desktop integration for Lenger (PRD 6.6, 7.3).
func Setup(apply bool) string {
	terminal := DetectTerminal()
	configFile := FindBindingsConfig()

	isLua := strings.HasSuffix(configFile, ".lua")
	bindingLine := fmt.Sprintf("bind = $mainMod ALT, D, exec, %s -e lenger", terminal)
	if isLua {
		bindingLine = fmt.Sprintf("o.bind(\"SUPER + ALT + D\", \"Lenger\", \"%s -e lenger\")", terminal)
	}

	var sb strings.Builder
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("  Lenger · Omarchy Desktop Keybinding Setup\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// Check if already bound
	if configFile != "" {
		if data, err := os.ReadFile(configFile); err == nil {
			content := string(data)
			if strings.Contains(content, "lenger") {
				fmt.Fprintf(&sb, "✓ Lenger is already configured in %s\n", configFile)
				return sb.String()
			}
			if strings.Contains(content, "$mainMod SHIFT, D") || strings.Contains(content, "SUPER + SHIFT + D") {
				sb.WriteString("ℹ lazydocker is bound to Super+Shift+D\n")
				sb.WriteString("  Using non-conflicting binding: Super+Alt+D\n\n")
			}
		}
	}

	if apply && configFile != "" {
		f, err := os.OpenFile(configFile, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			fmt.Fprintf(&sb, "✗ Failed to open %s: %v\n", configFile, err)
			sb.WriteString("  Add the following line manually:\n\n")
			sb.WriteString("  " + bindingLine + "\n")
			return sb.String()
		}
		defer func() { _ = f.Close() }()

		comment := "#"
		if isLua {
			comment = "--"
		}
		addition := fmt.Sprintf("\n%s Lenger Docker TUI\n%s\n", comment, bindingLine)
		if _, err := f.WriteString(addition); err != nil {
			fmt.Fprintf(&sb, "✗ Failed to write to %s: %v\n", configFile, err)
			return sb.String()
		}

		fmt.Fprintf(&sb, "✓ Successfully added keybinding to %s:\n\n", configFile)
		sb.WriteString("  " + bindingLine + "\n\n")
		sb.WriteString("Reload Hyprland config (Super+Shift+R or 'hyprctl reload') to activate.\n")
		return sb.String()
	}

	sb.WriteString("Add this line to your Hyprland bindings configuration:\n")
	if configFile != "" {
		fmt.Fprintf(&sb, "File: %s\n\n", configFile)
	} else {
		sb.WriteString("File: ~/.config/hypr/bindings.conf\n\n")
	}
	sb.WriteString("  " + bindingLine + "\n\n")
	sb.WriteString("To append automatically, re-run with:\n")
	sb.WriteString("  lenger setup-omarchy --apply\n\n")
	sb.WriteString("No files were modified.\n")
	return sb.String()
}
