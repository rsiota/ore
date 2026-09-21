package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsiota/ore/internal/bookmarks"
	"github.com/rsiota/ore/internal/git"
	"github.com/rsiota/ore/internal/session"
)

func TestAddBookmarkAndToggle(t *testing.T) {
	dir := t.TempDir()
	store := bookmarks.NewStore(dir)
	m := Model{
		repo:          &git.Repo{Path: dir + "/repo"},
		main:          MainCommits,
		commits:       []git.Commit{{Hash: "aaa111aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}},
		cursor:        0,
		bookmarkStore: store,
		diffMode:      DiffZen,
		zenContext:    3,
	}
	m.addBookmark("home")
	entries, err := store.Get(m.repo.Path)
	if err != nil || len(entries) != 1 || entries[0].Name != "home" {
		t.Fatalf("entries=%#v err=%v", entries, err)
	}
	m.addBookmark("home")
	if !strings.Contains(m.status, "already") {
		t.Fatalf("status=%q", m.status)
	}

	m.toggleBookmarks()
	if !m.bookmarks.IsVisible() {
		t.Fatal("expected panel open")
	}
	if len(m.bookmarks.filtered) != 1 {
		t.Fatalf("filtered=%d", len(m.bookmarks.filtered))
	}
	m.toggleBookmarks()
	if m.bookmarks.IsVisible() {
		t.Fatal("expected panel closed")
	}
}

func TestGMOpensBookmarksNotSave(t *testing.T) {
	dir := t.TempDir()
	store := bookmarks.NewStore(dir)
	m := Model{
		repo:          &git.Repo{Path: dir + "/repo"},
		main:          MainCommits,
		focus:         FocusMain,
		width:         80,
		height:        24,
		bookmarkStore: store,
		diffMode:      DiffZen,
	}
	mm, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = mm.(Model)
	if !m.chordG {
		t.Fatal("expected g chord pending")
	}
	mm, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = mm.(Model)
	if !m.bookmarks.IsVisible() {
		t.Fatalf("g m should open picker; status=%q visible=%v", m.status, m.bookmarks.IsVisible())
	}
	entries, _ := store.Get(m.repo.Path)
	if len(entries) != 0 {
		t.Fatalf("g m should not save a bookmark, got %d", len(entries))
	}
}

func TestBookmarkPanelJumpAndDelete(t *testing.T) {
	dir := t.TempDir()
	store := bookmarks.NewStore(dir)
	repo := dir + "/repo"
	_ = store.Add(repo, bookmarks.Bookmark{
		Name:    "alpha",
		SavedAt: time.Now(),
		View:    session.State{Main: "commits", Commit: "aaa111aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	})
	_ = store.Add(repo, bookmarks.Bookmark{
		Name:    "beta",
		SavedAt: time.Now().Add(time.Second),
		View:    session.State{Main: "files", Path: "x.go", Commit: "bbb222bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
	})

	m := Model{
		repo: &git.Repo{Path: repo},
		commits: []git.Commit{
			{Hash: "aaa111aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			{Hash: "bbb222bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		},
		bookmarkStore: store,
		main:          MainCommits,
		diffMode:      DiffZen,
	}
	m.toggleBookmarks()
	// Newest first → beta at cursor 0.
	if b, ok := m.bookmarks.selected(); !ok || b.Name != "beta" {
		t.Fatalf("selected=%v ok=%v", b, ok)
	}

	mm, cmd := m.bookmarks.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m.bookmarks = mm
	if cmd == nil {
		t.Fatal("expected delete cmd")
	}
	msg := cmd()
	if _, ok := msg.(bookmarkDeleteReqMsg); !ok {
		t.Fatalf("msg=%T", msg)
	}
	m.deleteSelectedBookmark()
	entries, _ := store.Get(repo)
	if len(entries) != 1 || entries[0].Name != "alpha" {
		t.Fatalf("after delete %#v", entries)
	}

	m.bookmarks.Open(entries)
	mm, cmd = m.bookmarks.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.bookmarks = mm
	if cmd == nil {
		t.Fatal("expected pick cmd")
	}
	picked, ok := cmd().(bookmarkPickedMsg)
	if !ok || picked.bm.Name != "alpha" {
		t.Fatalf("picked=%#v", picked)
	}
	_ = m.jumpBookmark(picked.bm.View)
	if m.cursor != 0 {
		t.Fatalf("cursor=%d want 0 for alpha commit", m.cursor)
	}
	if m.pendingRestore != nil {
		t.Fatalf("commits jump should clear pending restore, got %#v", m.pendingRestore)
	}
}
