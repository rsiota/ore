// Package config holds ore settings (YAML under ~/.config/ore/).
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const appName = "ore"

// Config is the top-level application configuration.
type Config struct {
	// Theme selects the colour palette (`light` or `dark`). Empty falls back
	// to light at apply time.
	Theme string `yaml:"theme,omitempty"`
}

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

// Path returns ~/.config/ore/config.yaml, creating the directory if needed.
func Path() (string, error) {
	dir, err := EnsureDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Load reads the config file. Missing file → empty Config (not an error).
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("could not read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("could not parse config: %w", err)
	}
	return &cfg, nil
}

// Save writes the config file to disk.
func (c *Config) Save() error {
	path, err := Path()
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("could not write config: %w", err)
	}
	return nil
}
