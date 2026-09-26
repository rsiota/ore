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

func TestSnapshotSessionWindows(t *testing.T) {
	m := Model{
		main:         MainCommits,
		logLimit:     1500,
		historyLimit: 1000,
		pickLimit:    250,
		pickKind:     hitListPickaxe,
	}
	st := m.snapshotSession()
	if st.LogLimit != 1500 || st.HistoryLimit != 1000 || st.HitLimit != 250 {
		t.Fatalf("windows = %#v", st)
	}

	m.logLimit = git.DefaultLogLimit
	m.historyLimit = git.DefaultLogLimit
	m.pickLimit = git.DefaultPickaxeLimit
	st = m.snapshotSession()
	if st.LogLimit != 0 || st.HistoryLimit != 0 || st.HitLimit != 0 {
		t.Fatalf("defaults should omit windows: %#v", st)
	}
}

func TestApplySessionChromeWindows(t *testing.T) {
	m := Model{}
	m.applySessionChrome(session.State{LogLimit: 2000, HistoryLimit: 800, HitLimit: 180})
	if m.logLimit != 2000 || m.historyLimit != 800 || m.pickLimit != 180 {
		t.Fatalf("applied log=%d hist=%d hit=%d", m.logLimit, m.historyLimit, m.pickLimit)
	}
}

func TestSnapshotSessionPickaxeAndHunks(t *testing.T) {
	m := Model{
		main:      MainPickaxe,
		pickKind:  hitListPickaxe,
		pickQuery: "UniqueToken",
		pickMode:  git.PickaxeRegexp,
		pickPath:  "a.go",
		pickaxe: []git.PickaxeHit{{
			Commit: git.Commit{Hash: "cccccccccccccccccccccccccccccccccccccccc", ShortHash: "ccccccc"},
		}},
		pickCursor: 0,
		hunkMode:   true,
		viewRev:    "main",
	}
	st := m.snapshotSession()
	if st.Main != "pickaxe" || st.PickQuery != "UniqueToken" || st.PickMode != "regexp" || st.PickPath != "a.go" {
		t.Fatalf("pickaxe snapshot = %#v", st)
	}
	if st.Commit != "cccccccccccccccccccccccccccccccccccccccc" || !st.Hunks {
		t.Fatalf("commit/hunks = %#v", st)
	}

	m.pickKind = hitListCouple
	m.coupleSeeds = []string{"a.go"}
	m.couplePartner = "b.go"
	st = m.snapshotSession()
	if st.Main != "couple" || st.CoupleWith != "b.go" || len(st.CoupleSeeds) != 1 {
		t.Fatalf("couple snapshot = %#v", st)
	}

	m.pickKind = hitListAuthors
	m.authorFilter = "alice"
	m.authorPaths = []string{"a.go"}
	m.authorRev = "HEAD"
	st = m.snapshotSession()
	if st.Main != "authors" || st.Author != "alice" || st.AuthorRev != "HEAD" {
		t.Fatalf("authors snapshot = %#v", st)
	}
}

func TestSnapshotSessionEvolve(t *testing.T) {
	m := Model{
		main:          MainLineEvo,
		evoOriginPath: "code.txt",
		evoOriginRev:  "HEAD",
		blameFrom:     MainHistory,
		evoCursor:     2,
		evo: []git.LineEvolutionStep{
			{Index: 0, Line: git.BlameLine{Line: 9}},
		},
		hunkMode: true,
	}
	st := m.snapshotSession()
	if st.Main != "evolve" || st.Path != "code.txt" || st.BlameRev != "HEAD" {
		t.Fatalf("evolve snapshot = %#v", st)
	}
	if st.BlameLine != 9 || st.EvoStep != 2 || st.BlameFrom != "history" || !st.Hunks {
		t.Fatalf("evolve fields = %#v", st)
	}
}

func TestContinueSessionRestorePickaxe(t *testing.T) {
	m := Model{
		repo: &git.Repo{Path: "/tmp/demo"},
		pendingRestore: &session.State{
			Main:      "pickaxe",
			PickQuery: "token",
			PickMode:  "string",
			Commit:    "aaa",
		},
	}
	cmd := m.continueSessionRestore()
	if cmd == nil || !m.restoreAfterPick || !m.loadingPick || m.pickKind != hitListPickaxe {
		t.Fatalf("cmd=%v pick=%v loading=%v kind=%v", cmd, m.restoreAfterPick, m.loadingPick, m.pickKind)
	}
	if m.pickQuery != "token" {
		t.Fatalf("query = %q", m.pickQuery)
	}
}

func TestFinishRestorePickSelectsCommit(t *testing.T) {
	m := Model{
		main: MainPickaxe,
		pickaxe: []git.PickaxeHit{
			{Commit: git.Commit{Hash: "aaa111aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}},
			{Commit: git.Commit{Hash: "bbb222bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},
		},
		pendingRestore: &session.State{
			Main:   "pickaxe",
			Commit: "bbb222bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
		restoreAfterPick: true,
	}
	m.finishRestorePick()
	if m.pickCursor != 1 {
		t.Fatalf("cursor = %d", m.pickCursor)
	}
	if m.pendingRestore != nil || m.restoreAfterPick {
		t.Fatal("pending should clear")
	}
	if !strings.Contains(m.status, "restored") {
		t.Fatalf("status = %q", m.status)
	}
}

func TestFinishRestoreFilesEvolveKeepsPending(t *testing.T) {
	m := Model{
		repo:            &git.Repo{Path: "/tmp/demo"},
		main:            MainFiles,
		files:           []git.FileChange{{Path: "a.go"}},
		filesCommitHash: "aaa",
		pendingRestore: &session.State{
			Main:      "evolve",
			Path:      "a.go",
			BlameRev:  "aaa",
			BlameLine: 4,
		},
		restoreAfterFiles: true,
	}
	cmd := m.finishRestoreFiles()
	if cmd == nil || !m.restoreAfterEvo || m.pendingRestore == nil {
		t.Fatalf("cmd=%v evo=%v pending=%v", cmd, m.restoreAfterEvo, m.pendingRestore)
	}
	if m.blamePreferLine != 4 || m.blamePath != "a.go" {
		t.Fatalf("prefer=%d path=%q", m.blamePreferLine, m.blamePath)
	}
}

func TestBlameLoadRestoresEvolve(t *testing.T) {
	line := git.BlameLine{Line: 3, Hash: "aaa", Text: "x"}
	m := Model{
		repo:            &git.Repo{Path: "/tmp/demo"},
		blamePath:       "code.txt",
		blameRev:        "HEAD",
		restoreAfterEvo: true,
		pendingRestore: &session.State{
			Main:      "evolve",
			Path:      "code.txt",
			BlameRev:  "HEAD",
			BlameLine: 3,
			EvoStep:   1,
		},
	}
	next, cmd := m.Update(blameLoadedMsg{
		path:  "code.txt",
		rev:   "HEAD",
		lines: []git.BlameLine{line},
	})
	mm := next.(Model)
	if !mm.loadingEvo || cmd == nil || mm.evoOriginPath != "code.txt" {
		t.Fatalf("loading=%v cmd=%v path=%q", mm.loadingEvo, cmd, mm.evoOriginPath)
	}
	next, _ = mm.Update(lineEvoLoadedMsg{
		path: "code.txt",
		rev:  "HEAD",
		steps: []git.LineEvolutionStep{
			{Index: 0, Path: "code.txt", Line: line},
			{Index: 1, Path: "code.txt", Line: git.BlameLine{Line: 2}},
		},
	})
	mm = next.(Model)
	if mm.main != MainLineEvo || mm.evoCursor != 1 {
		t.Fatalf("main=%v cursor=%d", mm.main, mm.evoCursor)
	}
	if mm.pendingRestore != nil || mm.restoreAfterEvo {
		t.Fatal("pending should clear after evolve")
	}
	if !strings.Contains(mm.status, "restored") {
		t.Fatalf("status = %q", mm.status)
	}
}

func TestApplySessionChromeHunks(t *testing.T) {
	m := Model{}
	m.applySessionChrome(session.State{Main: "commits", Hunks: true, DiffMode: "unified"})
	if !m.hunkMode || m.diffMode != DiffUnified {
		t.Fatalf("hunks=%v mode=%v", m.hunkMode, m.diffMode)
	}
}
