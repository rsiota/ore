package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

func TestPathScopedDetailDoesNotWipeFilesGrid(t *testing.T) {
	files := []git.FileChange{
		{Path: "a.go"},
		{Path: "b.go"},
		{Path: "c.go"},
	}
	m := Model{
		main:             MainFiles,
		files:            append([]git.FileChange(nil), files...),
		filesCommitHash:  "abcabcabc",
		fileCursor:       0,
		detailFilterPath: "a.go",
		detailPath:       "a.go",
		loadingDetail:    true,
	}
	next, _ := m.Update(detailLoadedMsg{
		hash: "abcabcabc",
		path: "a.go",
		detail: git.CommitDetail{
			Commit: git.Commit{Hash: "abcabcabc", ShortHash: "abcabca"},
			Files:  []git.FileChange{{Path: "a.go"}}, // ShowPath payload
		},
	})
	mm := next.(Model)
	if len(mm.files) != 3 {
		t.Fatalf("files wiped to %d entries: %+v", len(mm.files), mm.files)
	}
	if mm.detail == nil || len(mm.detail.Files) != 1 {
		t.Fatal("detail should still be path-scoped")
	}
	if mm.detailPath != "a.go" {
		t.Fatalf("detailPath = %q", mm.detailPath)
	}
}

func TestEnterFilesIgnoresPathScopedDetail(t *testing.T) {
	m := Model{
		main: MainCommits,
		commits: []git.Commit{
			{Hash: "abcabcabc", ShortHash: "abcabca"},
		},
		cursor:     0,
		detailPath: "a.go",
		detail: &git.CommitDetail{
			Commit: git.Commit{Hash: "abcabcabc", ShortHash: "abcabca"},
			Files:  []git.FileChange{{Path: "a.go"}},
		},
	}
	next, cmd := m.handleCommitKeys(tea.KeyMsg{Type: tea.KeyEnter})
	mm := next.(Model)
	if !mm.openFilesPending {
		t.Fatal("expected openFilesPending when detail is path-scoped")
	}
	if mm.main != MainCommits {
		t.Fatalf("should stay on commits until whole Show arrives, got %v", mm.main)
	}
	if cmd == nil {
		t.Fatal("expected whole-commit load cmd")
	}
	// Simulate whole-commit Show arriving.
	mm.loadingDetail = true
	mm.detailFilterPath = ""
	next, cmd = mm.Update(detailLoadedMsg{
		hash: "abcabcabc",
		path: "",
		detail: git.CommitDetail{
			Commit: git.Commit{Hash: "abcabcabc", ShortHash: "abcabca"},
			Files: []git.FileChange{
				{Path: "a.go"},
				{Path: "b.go"},
				{Path: "c.go"},
			},
		},
	})
	mm = next.(Model)
	if mm.main != MainFiles {
		t.Fatalf("main = %v, want MainFiles", mm.main)
	}
	if len(mm.files) != 3 {
		t.Fatalf("files = %d, want 3", len(mm.files))
	}
	if mm.openFilesPending {
		t.Fatal("openFilesPending should clear")
	}
}
