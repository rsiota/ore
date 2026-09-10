package ui

import "testing"

func TestRegistryHasHelpAndFilter(t *testing.T) {
	var (
		hasHelp   bool
		hasFilter bool
		hasQuit   bool
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
				}
			}
		}
	}
	if !hasHelp || !hasFilter || !hasQuit {
		t.Fatalf("registry missing core bindings: help=%v filter=%v quit=%v", hasHelp, hasFilter, hasQuit)
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
