package ui

import (
	"testing"

	"github.com/rsiota/ore/internal/git"
)

func TestStartAuthorsStagesSearch(t *testing.T) {
	m := Model{
		repo: &git.Repo{Path: "/tmp/demo"},
		main: MainCommits,
	}
	nm, cmd := m.startAuthors("alice", []string{"a.go"}, "HEAD")
	m = nm.(Model)
	if !m.loadingPick || m.pickKind != hitListAuthors || cmd == nil {
		t.Fatalf("loading=%v kind=%v cmd=%v", m.loadingPick, m.pickKind, cmd)
	}
	if m.pickQuery != "alice · a.go" {
		t.Fatalf("label = %q", m.pickQuery)
	}
	nm2, _ := m.Update(authorLoadedMsg{
		author: "alice",
		paths:  []string{"a.go"},
		rev:    "HEAD",
		label:  "alice · a.go",
		hits: []git.PickaxeHit{{
			Commit: git.Commit{Hash: "aaa", ShortHash: "aaa", Author: "alice", Subject: "a1"},
			Paths:  []string{"a.go"},
		}},
	})
	m = nm2.(Model)
	if m.main != MainPickaxe || len(m.pickaxe) != 1 {
		t.Fatalf("main=%v len=%d", m.main, len(m.pickaxe))
	}
	if m.mainTitle() != "authors" {
		t.Fatalf("title = %q", m.mainTitle())
	}
}

func TestExAuthorsNeedsName(t *testing.T) {
	m := Model{main: MainCommits}
	if cmd := m.exAuthors(nil); cmd != nil {
		t.Fatal("expected no cmd")
	}
	if m.status == "" {
		t.Fatal("expected usage status")
	}
}

func TestActivateOwnershipRow(t *testing.T) {
	var e RelExplorer
	e.LoadCommit(git.CommitRelations{
		Hash:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Subject: "root",
		Author:  "a",
		Email:   "a@b",
		Files:   []git.FileChange{{Path: "a.go"}},
		Ownership: []git.AuthorShare{
			{Name: "alice", Count: 2, Pct: 67},
		},
	})
	var own *relNode
	for _, n := range e.visibleNodes() {
		if n.kind == relOwn {
			own = n
			break
		}
	}
	if own == nil {
		t.Fatal("expected ownership row")
	}
	m := Model{
		repo:     &git.Repo{Path: "/tmp/demo"},
		main:     MainCommits,
		explorer: e,
		focus:    FocusExplorer,
	}
	// Land cursor on the ownership row.
	for i, n := range m.explorer.visibleNodes() {
		if n.kind == relOwn {
			m.explorer.cursor = i
			break
		}
	}
	nm, cmd := m.activateRelation()
	m = nm.(Model)
	if cmd == nil || m.pickKind != hitListAuthors || m.authorFilter != "alice" {
		t.Fatalf("kind=%v author=%q cmd=%v", m.pickKind, m.authorFilter, cmd)
	}
	if !m.loadingPick || m.explorer.Opened() {
		t.Fatalf("loading=%v explorer=%v", m.loadingPick, m.explorer.Opened())
	}
}
