package ui

import (
	"testing"

	"github.com/rsiota/ore/internal/git"
)

func TestStartPickaxeStagesSearch(t *testing.T) {
	m := Model{
		repo: &git.Repo{Path: "/tmp/demo"},
		main: MainCommits,
	}
	nm, cmd := m.startPickaxe("UniqueToken", git.PickaxeString, "")
	m = nm.(Model)
	if !m.loadingPick || m.pickQuery != "UniqueToken" || cmd == nil {
		t.Fatalf("start: loading=%v query=%q cmd=%v", m.loadingPick, m.pickQuery, cmd)
	}
	if m.pickaxeFrom != MainCommits {
		t.Fatalf("from = %v", m.pickaxeFrom)
	}

	nm2, _ := m.Update(pickaxeLoadedMsg{
		query: "UniqueToken",
		mode:  git.PickaxeString,
		hits: []git.PickaxeHit{{
			Commit: git.Commit{Hash: "aaa", ShortHash: "aaa", Subject: "add UniqueToken"},
			Paths:  []string{"app.go"},
		}},
	})
	m = nm2.(Model)
	if m.main != MainPickaxe || len(m.pickaxe) != 1 {
		t.Fatalf("after load main=%v len=%d", m.main, len(m.pickaxe))
	}
	view := m.renderPickaxePane(100, 20)
	if view == "" {
		t.Fatal("expected pickaxe pane")
	}
}

func TestActivatePickaxeHitOpensHistory(t *testing.T) {
	m := Model{
		repo:      &git.Repo{Path: "/tmp/demo"},
		main:      MainPickaxe,
		pickQuery: "tok",
		pickaxe: []git.PickaxeHit{{
			Commit: git.Commit{Hash: "aaa", ShortHash: "aaa", Subject: "hit"},
			Paths:  []string{"only.go"},
		}},
		pickCursor: 0,
	}
	nm, cmd := m.activatePickaxeHit()
	m = nm.(Model)
	if !m.pickaxeDive || m.historyPath != "only.go" || !m.loadingHistory || cmd == nil {
		t.Fatalf("dive=%v path=%q loading=%v cmd=%v", m.pickaxeDive, m.historyPath, m.loadingHistory, cmd)
	}
}

func TestPickaxeEscClearsDive(t *testing.T) {
	m := Model{
		main:         MainPickaxe,
		pickaxeFrom:  MainCommits,
		pickQuery:    "x",
		pickaxe:      []git.PickaxeHit{{Commit: git.Commit{Hash: "a"}}},
		pickaxeDive:  true,
	}
	nm, _ := m.goBack()
	m = nm.(Model)
	if m.main != MainCommits || m.pickaxeDive || m.pickQuery != "" {
		t.Fatalf("main=%v dive=%v query=%q", m.main, m.pickaxeDive, m.pickQuery)
	}
}
