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
	if exLookup("refresh") == nil {
		t.Fatal("expected :refresh")
	}
	if exLookup("reload") == nil {
		t.Fatal("expected :reload alias")
	}
	if exLookup("authors") == nil {
		t.Fatal("expected :authors")
	}
	if exLookup("more") == nil {
		t.Fatal("expected :more")
	}
	if exLookup("author") == nil {
		t.Fatal("expected :author alias")
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
	h := NewHelpPanel()
	h.Show()
	h.page = helpPageCommands
	h.SetSize(80, 40)
	var rows strings.Builder
	for _, row := range renderCommandsRows(80) {
		rows.WriteString(row.searchText())
		rows.WriteByte('\n')
	}
	body := rows.String()
	for _, want := range []string{":blame", ":history", ":goto", ":authors", ":branch", ":theme", ":set", ":session", ":refresh", ":more"} {
		if !strings.Contains(body, want) {
			t.Fatalf("help missing %q", want)
		}
	}
	if !strings.Contains(stripAnsi(h.View()), "Commands") {
		t.Fatal("help missing Commands tab")
	}
}
