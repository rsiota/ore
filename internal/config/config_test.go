package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSaveTheme(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "" {
		t.Fatalf("empty default theme = %q", cfg.Theme)
	}

	cfg.Theme = "dark"
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "ore", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "theme: dark") {
		t.Fatalf("yaml = %q", data)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Theme != "dark" {
		t.Fatalf("loaded theme = %q", loaded.Theme)
	}
}

func TestLoadSaveTransparentBackground(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg := &Config{TransparentBackground: true}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "ore", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "transparent_background: true") {
		t.Fatalf("yaml = %q", data)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.TransparentBackground {
		t.Fatal("expected transparent_background true")
	}
}
