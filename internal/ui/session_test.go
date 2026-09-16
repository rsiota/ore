package ui

import (
	"strings"
	"testing"

	"github.com/rsiota/ore/internal/git"
	"github.com/rsiota/ore/internal/session"
)

func TestSnapshotSessionBlame(t *testing.T) {
	m := Model{
		main:            MainBlame,
		viewRev:         "feature",
		filesCommitHash: "aaa111",
		blamePath:       "a.go",
		blameRev:        "bbb222",
		blameFrom:       MainFiles,
		blame: []git.BlameLine{
			{Line: 10, Hash: "bbb222"},
			{Line: 11, Hash: "bbb222"},
		},
		blameCursor:     1,
		diffMode:        DiffZen,
		zenContext:      5,
		detailWrap:      true,
		blameGutterFold: 1,
	}
	st := m.snapshotSession()
	if st.Main != "blame" || st.Path != "a.go" || st.BlameLine != 11 {
		t.Fatalf("snapshot = %#v", st)
	}
	if st.ViewRev != "feature" || st.BlameFrom != "files" || st.ZenContext != 5 {
		t.Fatalf("chrome = %#v", st)
	}
	if !st.DetailWrap || st.DiffMode != "zen" {
		t.Fatalf("diff chrome = %#v", st)
	}
}

func TestContinueSessionRestoreFiles(t *testing.T) {
	m := Model{
		repo: &git.Repo{Path: "/tmp/demo"},
		commits: []git.Commit{
			{Hash: "aaa111aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortHash: "aaa111"},
			{Hash: "bbb222bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ShortHash: "bbb222"},
		},
		pendingRestore: &session.State{
			Main:   "files",
			Commit: "bbb222bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Path:   "x.go",
		},
	}
	cmd := m.continueSessionRestore()
	if cmd == nil {
		t.Fatal("expected detail reload cmd")
	}
	if m.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", m.cursor)
	}
	if !m.openFilesPending || !m.restoreAfterFiles {
		t.Fatalf("flags open=%v afterFiles=%v", m.openFilesPending, m.restoreAfterFiles)
	}
}

func TestFinishRestoreFilesSelectsPath(t *testing.T) {
	m := Model{
		main: MainFiles,
		files: []git.FileChange{
			{Path: "a.go"},
			{Path: "b.go"},
		},
		filesCommitHash: "aaa",
		pendingRestore: &session.State{
			Main:   "files",
			Path:   "b.go",
			Commit: "aaa",
		},
		restoreAfterFiles: true,
	}
	_ = m.finishRestoreFiles()
	if m.fileCursor != 1 {
		t.Fatalf("fileCursor = %d, want 1", m.fileCursor)
	}
	if m.pendingRestore != nil {
		t.Fatal("pending should clear")
	}
}

func TestExSessionClear(t *testing.T) {
	dir := t.TempDir()
	store := session.NewStore(dir)
	repo := "/tmp/ore-session-test"
	_ = store.Save(repo, session.State{Main: "commits", Commit: "abc"})

	m := Model{
		repo:           &git.Repo{Path: repo},
		sessionStore:   store,
		pendingRestore: &session.State{Main: "commits"},
	}
	_ = m.exSession([]string{"clear"})
	if m.pendingRestore != nil {
		t.Fatal("pending should clear")
	}
	st, _ := store.Load(repo)
	if st.HasContent() {
		t.Fatalf("store still has content: %#v", st)
	}
	if !strings.Contains(m.status, "cleared") {
		t.Fatalf("status = %q", m.status)
	}
}
