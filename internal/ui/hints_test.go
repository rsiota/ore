package ui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsiota/ore/internal/git"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

// sgrPrefix returns the leading SGR sequence of a rendered style sample.
func sgrPrefix(sty interface{ Render(...string) string }) string {
	sample := sty.Render("x")
	if i := strings.Index(sample, "x"); i > 0 {
		return sample[:i]
	}
	return ""
}

func TestMatchHint(t *testing.T) {
	hints := []string{"j/k", "h/l", "o", "/", "enter", "ctrl+p", "gr"}
	cases := map[string]string{
		"j":      "j",
		"k":      "k",
		"/":      "/",
		"enter":  "enter",
		"ctrl+p": "ctrl+p",
		"gr":     "gr",
		"z":      "",
		"g":      "", // chord groups only match whole token
	}
	for key, want := range cases {
		if got := matchHint(hints, key); got != want {
			t.Errorf("matchHint(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestHintDescriptionLookup(t *testing.T) {
	m := Model{main: MainCommits}
	cases := map[string]string{
		"j":     "move row",
		"o":     "cycle sort on current column (asc→desc→off)",
		"enter": "open files for commit",
		"q":     "quit",
	}
	for key, want := range cases {
		if got := m.hintDescription(key); got != want {
			t.Errorf("hintDescription(%q) = %q, want %q", key, got, want)
		}
	}
	m.explorer.open = true
	if got := m.hintDescription("l"); got != "expand commit / dive into nested" {
		t.Errorf("explorer hintDescription(l) = %q", got)
	}
}

func TestHintFlashStagesOnKey(t *testing.T) {
	m := Model{
		repo:   &git.Repo{Path: "/tmp/demo"},
		width:  140,
		height: 40,
		main:   MainCommits,
		branch: "main",
		head:   "abc1234",
	}
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = mm.(Model)
	if m.hintFlash != "j" {
		t.Fatalf("hintFlash = %q, want j", m.hintFlash)
	}
	if m.hintDesc != "move row" {
		t.Fatalf("hintDesc = %q, want move row", m.hintDesc)
	}
	bar := stripANSI(m.renderStatus())
	if !strings.Contains(bar, "move row") {
		t.Fatalf("status missing description: %q", bar)
	}
}

func TestHintDescriptionExpires(t *testing.T) {
	m := Model{
		repo:      &git.Repo{Path: "/tmp/demo"},
		width:     140,
		height:    40,
		main:      MainCommits,
		hintDesc:  "move row",
		hintDescAt: time.Now().Add(-(hintDescDuration + time.Millisecond)),
		branch:    "main",
		head:      "abc",
	}
	bar := stripANSI(m.renderStatus())
	if strings.Contains(bar, "move row") {
		t.Fatalf("expired description still shown: %q", bar)
	}
}

func TestHintFlashClearedOnUnrelatedKey(t *testing.T) {
	m := Model{main: MainCommits, hintFlash: "j", hintDesc: "move row"}
	m.stageHintFlash("z")
	if m.hintFlash != "" || m.hintDesc != "" {
		t.Fatalf("expected clear, got flash=%q desc=%q", m.hintFlash, m.hintDesc)
	}
}

func TestHintFlashFgBold(t *testing.T) {
	m := Model{
		repo:        &git.Repo{Path: "/tmp/demo"},
		width:       140,
		height:      40,
		main:        MainCommits,
		hintFlash:   "j",
		hintFlashAt: time.Now(),
		branch:      "main",
		head:        "abc",
	}
	bar := m.renderStatus()
	flashPrefix := sgrPrefix(styleHintFlash)
	if flashPrefix == "" || !strings.Contains(bar, flashPrefix) {
		t.Fatalf("status missing hint flash SGR %q:\n%q", flashPrefix, bar)
	}
	idlePrefix := sgrPrefix(styleMuted)
	if flashPrefix == idlePrefix {
		t.Fatal("styleHintFlash collapsed to styleMuted; flash would be invisible")
	}
}
