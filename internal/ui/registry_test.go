package ui

import (
	"strings"
	"testing"
)

func TestRegistryHasHelpAndFilter(t *testing.T) {
	var (
		hasHelp    bool
		hasFilter  bool
		hasQuit    bool
		hasRefresh bool
	)
	for _, sec := range registry() {
		for _, b := range sec.Items {
			for _, tok := range b.Tokens {
				switch tok {
				case "?":
					hasHelp = true
				case "/":
					hasFilter = true
				case "q":
					hasQuit = true
				case "ctrl+r":
					hasRefresh = true
				}
			}
		}
	}
	if !hasHelp || !hasFilter || !hasQuit || !hasRefresh {
		t.Fatalf("registry missing core bindings: help=%v filter=%v quit=%v refresh=%v",
			hasHelp, hasFilter, hasQuit, hasRefresh)
	}
}

func TestStatusHintsAreKeyOnly(t *testing.T) {
	got := statusHints(MainCommits, false)
	if got != "j/k/h/l/o///enter/gr/tab/?/esc/q" {
		// "/" is a hint key, so it appears as an empty group between slashes (creel-style).
		t.Fatalf("commits hints = %q", got)
	}
	got = statusHints(MainCommits, true)
	if got != "gr/j/k/enter/esc/tab" {
		t.Fatalf("explorer hints = %q", got)
	}
	for _, s := range []string{got, statusHints(MainFiles, false), statusHints(MainBlame, false)} {
		if strings.Contains(s, " ") || strings.Contains(s, "·") {
			t.Fatalf("hints should be key-only, got %q", s)
		}
	}
}

func TestFilterMatch(t *testing.T) {
	if !filterMatch("auth", "Refactor auth middleware") {
		t.Fatal("expected substring match")
	}
	if filterMatch("zzz", "hello") {
		t.Fatal("expected no match")
	}
	if !filterMatch("", "anything") {
		t.Fatal("empty query matches all")
	}
}
