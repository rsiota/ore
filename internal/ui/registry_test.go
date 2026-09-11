package ui

import "testing"

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
