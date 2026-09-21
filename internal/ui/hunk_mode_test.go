package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsiota/ore/internal/git"
)

func TestCollectDetailHunks(t *testing.T) {
	logical := []string{
		zenFilePrefix + "pkg/a.go",
		zenHunkPrefix + "@@ -1,2 +1,3 @@ top",
		zenAddPrefix + "1\x1fhi",
		zenHunkPrefix + "@@ -10 +10 @@ bottom",
	}
	visual := []string{
		"pkg/a.go",
		"@@ -1,2 +1,3 @@ top",
		"hi",
		"@@ -10 +10 @@ bottom",
	}
	hunks := collectDetailHunks(visual, logical)
	if len(hunks) != 2 {
		t.Fatalf("hunks=%#v", hunks)
	}
	if hunks[0].Row != 1 || hunks[1].Row != 3 {
		t.Fatalf("rows=%d,%d", hunks[0].Row, hunks[1].Row)
	}
	if hunks[0].File != "pkg/a.go" || hunks[1].File != "pkg/a.go" {
		t.Fatalf("files=%q,%q", hunks[0].File, hunks[1].File)
	}
}

func TestToggleHunkModeAndJump(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/f b/f",
		"--- a/f",
		"+++ b/f",
		"@@ -1,2 +1,2 @@ top",
		" a",
		"-b",
		"+B",
		"@@ -20,2 +20,2 @@ bottom",
		" c",
		"-d",
		"+D",
	}, "\n")
	m := Model{
		width:      120,
		height:     40,
		diffMode:   DiffZen,
		zenContext: 3,
		detail: &git.CommitDetail{
			Commit: git.Commit{Hash: "h", Subject: "s", Files: 1},
			Diff:   diff,
		},
		detailCache: &detailRenderCache{},
		focus:       FocusMain,
	}
	m.toggleHunkMode()
	if !m.hunkMode || len(m.hunks) < 2 {
		t.Fatalf("mode=%v hunks=%d", m.hunkMode, len(m.hunks))
	}
	mm, _ := m.cycleFocus()
	m = mm.(Model)
	if m.focus != FocusHunks {
		t.Fatalf("tab → hunks, got %v", m.focus)
	}
	_ = m.selectHunk(1)
	if m.hunkCursor != 1 {
		t.Fatalf("cursor=%d", m.hunkCursor)
	}
	if m.detailOffset != m.hunks[1].Row {
		t.Fatalf("detailOffset=%d want %d", m.detailOffset, m.hunks[1].Row)
	}

	mm, _ = m.handleHunkKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = mm.(Model)
	if m.hunkCursor != 0 {
		t.Fatalf("k → hunk 0, got %d", m.hunkCursor)
	}

	m.toggleHunkMode()
	if m.hunkMode || m.focus == FocusHunks {
		t.Fatal("H off should clear mode and leave hunks focus")
	}
}

func TestHunkModeBracketsFromMain(t *testing.T) {
	diff := "diff --git a/f b/f\n--- a/f\n+++ b/f\n@@ -1 +1 @@\n-a\n+b\n@@ -2 +2 @@\n-c\n+d\n"
	m := Model{
		width: 120, height: 40, diffMode: DiffZen, zenContext: 3,
		detail:      &git.CommitDetail{Commit: git.Commit{Hash: "h", Subject: "s"}, Diff: diff},
		detailCache: &detailRenderCache{},
		hunkMode:    true,
		focus:       FocusMain,
	}
	m.rebuildDetailHunks()
	if len(m.hunks) < 2 {
		t.Fatalf("hunks=%d", len(m.hunks))
	}
	_ = m.jumpHunkList(1)
	if m.hunkCursor != 1 {
		t.Fatalf("] → %d", m.hunkCursor)
	}
}
