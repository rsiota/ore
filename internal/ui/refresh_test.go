package ui

import (
	"strings"
	"testing"

	"github.com/rsiota/ore/internal/git"
)

func TestRestoreCommitCursor(t *testing.T) {
	m := Model{
		commits: []git.Commit{
			{Hash: "aaa1111111111111111111111111111111111111", ShortHash: "aaa1111"},
			{Hash: "bbb2222222222222222222222222222222222222", ShortHash: "bbb2222"},
			{Hash: "ccc3333333333333333333333333333333333333", ShortHash: "ccc3333"},
		},
		cursor: 0,
	}
	m.restoreCommitCursor("bbb2222222222222222222222222222222222222")
	if m.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", m.cursor)
	}
	m.restoreCommitCursor("deadbeef")
	if m.cursor != 0 {
		t.Fatalf("missing hash should reset cursor, got %d", m.cursor)
	}
}

func TestRefreshSetsPreferHashAndLoading(t *testing.T) {
	m := Model{
		commits: []git.Commit{
			{Hash: "aaa1111111111111111111111111111111111111", ShortHash: "aaa1111"},
			{Hash: "bbb2222222222222222222222222222222222222", ShortHash: "bbb2222"},
		},
		cursor: 1,
	}
	cmd := m.refresh()
	if cmd == nil {
		t.Fatal("expected load cmd")
	}
	if !m.loading {
		t.Fatal("expected loading")
	}
	if m.refreshPreferHash != m.commits[1].Hash {
		t.Fatalf("prefer = %q", m.refreshPreferHash)
	}
	if m.status != "refreshing…" {
		t.Fatalf("status = %q", m.status)
	}
	// Second refresh while loading is a no-op.
	if m.refresh() != nil {
		t.Fatal("expected nil while already loading")
	}
}

func TestCommitsLoadedRefreshPreservesView(t *testing.T) {
	m := Model{
		main:              MainFiles,
		filesCommitHash:   "bbb2222222222222222222222222222222222222",
		refreshPreferHash: "bbb2222222222222222222222222222222222222",
		cursor:            0,
		loading:           true,
		commits: []git.Commit{
			{Hash: "aaa1111111111111111111111111111111111111", ShortHash: "aaa1111"},
		},
	}
	next, _ := m.Update(commitsLoadedMsg{
		commits: []git.Commit{
			{Hash: "aaa1111111111111111111111111111111111111", ShortHash: "aaa1111"},
			{Hash: "bbb2222222222222222222222222222222222222", ShortHash: "bbb2222"},
			{Hash: "ccc3333333333333333333333333333333333333", ShortHash: "ccc3333"},
		},
		branch: "main",
		head:   "ccc3333",
	})
	mm := next.(Model)
	if mm.main != MainFiles {
		t.Fatalf("main = %v, want MainFiles", mm.main)
	}
	if mm.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", mm.cursor)
	}
	if mm.refreshPreferHash != "" {
		t.Fatal("prefer hash should clear")
	}
	if !strings.HasPrefix(mm.status, "refreshed") {
		t.Fatalf("status = %q", mm.status)
	}
	if mm.branch != "main" || mm.head != "ccc3333" {
		t.Fatalf("branch/head not updated: %q @ %q", mm.branch, mm.head)
	}
}

func TestSyncFilesFromDetailPreservesPath(t *testing.T) {
	m := Model{
		main: MainFiles,
		files: []git.FileChange{
			{Path: "a.go"},
			{Path: "b.go"},
			{Path: "c.go"},
		},
		fileCursor: 1,
		detail: &git.CommitDetail{
			Commit: git.Commit{Hash: "abc", ShortHash: "abc"},
			Files: []git.FileChange{
				{Path: "a.go"},
				{Path: "b.go"},
				{Path: "d.go"},
			},
		},
	}
	m.syncFilesFromDetail()
	if m.fileCursor != 1 {
		t.Fatalf("fileCursor = %d, want 1 (b.go)", m.fileCursor)
	}
	if len(m.files) != 3 || m.files[2].Path != "d.go" {
		t.Fatalf("files not synced: %+v", m.files)
	}
}
