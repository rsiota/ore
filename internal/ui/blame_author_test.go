package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestAuthorColorStableAndThemeAware(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer applyTheme(defaultThemeName)

	applyTheme("light")
	a := string(authorColor("alice"))
	b := string(authorColor("alice"))
	if a != b {
		t.Fatalf("unstable: %q vs %q", a, b)
	}
	if authorHueIndex("alice") == authorHueIndex("zzzz-unlikely-same") &&
		string(authorColor("alice")) == string(authorColor("zzzz-unlikely-same")) {
		// Extremely unlikely with distinct names; keep as soft signal only.
		t.Log("hue collision for distinct names (acceptable)")
	}

	applyTheme("dark")
	dark := string(authorColor("alice"))
	if dark == a {
		t.Fatalf("dark hue should differ from light for same name: %q", dark)
	}
}

func TestAuthorNameStyleEmptyIsMuted(t *testing.T) {
	st := authorNameStyle("  ")
	got := st.Render("x")
	muted := styleMuted.Render("x")
	if got != muted {
		t.Fatalf("empty author should use muted, got %q want %q", got, muted)
	}
}

func TestBlameAuthorColumnUsesAuthorHue(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer applyTheme(defaultThemeName)
	applyTheme("light")

	text := " alice     "
	st := authorNameStyle("alice")
	got := st.Render(text)
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("expected coloured author: %q", got)
	}
	// Must not paint a background — code/age washes stay the only fills.
	if strings.Contains(got, "48;") {
		t.Fatalf("author style should be fg-only: %q", got)
	}
}
