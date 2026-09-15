package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestApplyThemeSwitchesPalette(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer applyTheme(defaultThemeName)

	applyTheme("light")
	if activeThemeName != "light" {
		t.Fatalf("activeThemeName = %q", activeThemeName)
	}
	if string(colorBg) != "#ffffff" {
		t.Fatalf("light bg = %q", colorBg)
	}
	if string(colorAgeNewBg) != "#eef6f0" {
		t.Fatalf("light ageNew = %q", colorAgeNewBg)
	}

	applyTheme("dark")
	if activeThemeName != "dark" {
		t.Fatalf("activeThemeName = %q", activeThemeName)
	}
	if string(colorBg) != "#0d1117" {
		t.Fatalf("dark bg = %q", colorBg)
	}
	if string(colorAgeNewBg) != "#12261e" {
		t.Fatalf("dark ageNew = %q", colorAgeNewBg)
	}
	if string(colorPrimary) != "#c9d1d9" {
		t.Fatalf("dark primary = %q", colorPrimary)
	}

	applyTheme("nope")
	if activeThemeName != "light" {
		t.Fatalf("unknown should fall back to light, got %q", activeThemeName)
	}
}

func TestBlameAgeStyleUsesThemeWashes(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer applyTheme(defaultThemeName)

	newest := time.Now()
	oldest := newest.Add(-48 * time.Hour)

	applyTheme("dark")
	got := blameAgeStyle(newest, newest, oldest).Render("code")
	if !strings.Contains(got, "48;2;") {
		t.Fatalf("expected dark age wash bg in %q", got)
	}
	if string(colorAgeNewBg) != "#12261e" {
		t.Fatalf("dark ageNew slot = %q", colorAgeNewBg)
	}

	applyTheme("light")
	got = blameAgeStyle(oldest, newest, oldest).Render("code")
	if !strings.Contains(got, "48;2;") {
		t.Fatalf("expected light age wash bg in %q", got)
	}
	if string(colorAgeOldBg) != "#f0e6e4" {
		t.Fatalf("light ageOld slot = %q", colorAgeOldBg)
	}
	// Dark and light washes must differ.
	applyTheme("dark")
	darkWash := string(colorAgeNewBg)
	applyTheme("light")
	if string(colorAgeNewBg) == darkWash {
		t.Fatal("light and dark age washes should differ")
	}
}

func TestExTheme(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer applyTheme(defaultThemeName)

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	m := Model{config: nil, detailCache: &detailRenderCache{}}
	_ = m.exTheme(nil)
	if !strings.Contains(m.status, "theme") {
		t.Fatalf("no-arg status = %q", m.status)
	}

	_ = m.exTheme([]string{"nope"})
	if !strings.HasPrefix(m.status, "unknown theme") {
		t.Fatalf("bad theme status = %q", m.status)
	}

	_ = m.exTheme([]string{"Dark"})
	if m.theme != "dark" || activeThemeName != "dark" {
		t.Fatalf("theme = %q active = %q", m.theme, activeThemeName)
	}
	if m.config == nil || m.config.Theme != "dark" {
		t.Fatalf("config not updated: %#v", m.config)
	}
	if !strings.Contains(m.status, "theme dark") {
		t.Fatalf("status = %q", m.status)
	}
}

func TestThemeListedInHelp(t *testing.T) {
	h := HelpPanel{visible: true, width: 80, height: 40}
	rows := strings.Join(h.rows(), "\n")
	if !strings.Contains(rows, ":theme") {
		t.Fatal("help missing :theme")
	}
}
