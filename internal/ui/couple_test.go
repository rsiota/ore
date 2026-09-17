package ui

import (
	"testing"

	"github.com/rsiota/ore/internal/git"
)

func TestStartCoupleStagesSearch(t *testing.T) {
	m := Model{
		repo: &git.Repo{Path: "/tmp/demo"},
		main: MainCommits,
	}
	nm, cmd := m.startCouple([]string{"a.go"}, "b.go")
	m = nm.(Model)
	if !m.loadingPick || m.pickKind != hitListCouple || cmd == nil {
		t.Fatalf("loading=%v kind=%v cmd=%v", m.loadingPick, m.pickKind, cmd)
	}
	if m.pickQuery != "a.go ∩ b.go" {
		t.Fatalf("label = %q", m.pickQuery)
	}
	nm2, _ := m.Update(coupleLoadedMsg{
		seeds:   []string{"a.go"},
		partner: "b.go",
		label:   "a.go ∩ b.go",
		hits: []git.PickaxeHit{{
			Commit: git.Commit{Hash: "aaa", ShortHash: "aaa", Subject: "pair"},
			Paths:  []string{"a.go", "b.go"},
		}},
	})
	m = nm2.(Model)
	if m.main != MainPickaxe || len(m.pickaxe) != 1 {
		t.Fatalf("main=%v len=%d", m.main, len(m.pickaxe))
	}
	if m.mainTitle() != "couple" {
		t.Fatalf("title = %q", m.mainTitle())
	}
}
