// Package config holds ore settings (YAML under ~/.config/ore/).
// Wave 0 keeps this minimal; expand as themes, keybindings, and session
// restore land.
package config

import (
	"os"
	"path/filepath"
)

const appName = "ore"

// Dir returns ~/.config/ore (or $XDG_CONFIG_HOME/ore).
func Dir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, appName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", appName), nil
}

// EnsureDir creates the config directory if missing.
func EnsureDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
