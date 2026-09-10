package ui

import (
	"strings"
	"testing"

	"github.com/rsiota/ore/internal/git"
)

func TestExLookup(t *testing.T) {
	if exLookup("blame") == nil {
		t.Fatal("expected :blame")
	}
	if exLookup("hist") == nil {
		t.Fatal("expected :hist alias")
	}
	if exLookup("go") == nil {
		t.Fatal("expected :go alias")
	}
	if exLookup("nope") != nil {
		t.Fatal("expected nil for unknown")
	}
}

func TestRunExCommandEmpty(t *testing.T) {
	m := Model{}
	if cmd := m.runExCommand("  "); cmd != nil {
		t.Fatal("empty should no-op")
	}
}

func TestRunExUnknown(t *testing.T) {
	m := Model{}
	_ = m.runExCommand("foobar")
	if !strings.HasPrefix(m.status, "E492") {
		t.Fatalf("status = %q, want E492…", m.status)
	}
}

func TestExGotoAmbiguous(t *testing.T) {
	m := Model{
		commits: []git.Commit{
			{Hash: "aaaaaaa1aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortHash: "aaaaaaa"},
			{Hash: "aaaaaaa2aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortHash: "aaaaaaa"},
		},
	}
	_ = m.exGoto([]string{"aaaaaaa"})
	if !strings.HasPrefix(m.status, "ambiguous") {
		t.Fatalf("status = %q", m.status)
	}
}

func TestExCommandsListedInHelp(t *testing.T) {
	h := HelpPanel{visible: true, width: 80, height: 40}
	rows := strings.Join(h.rows(), "\n")
	for _, want := range []string{":blame", ":history", ":goto", "Commands"} {
		if !strings.Contains(rows, want) {
			t.Fatalf("help missing %q", want)
		}
	}
}
