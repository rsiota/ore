package bookmarks

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rsiota/ore/internal/session"
)

func TestAddGetDuplicate(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	repo := filepath.Join(dir, "repo")

	b := Bookmark{
		Name: "spot",
		View: session.State{Main: "blame", Path: "a.go", BlameLine: 10, Commit: "abc"},
	}
	if err := s.Add(repo, b); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(repo, b); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
	entries, err := s.Get(repo)
	if err != nil || len(entries) != 1 {
		t.Fatalf("entries=%v err=%v", entries, err)
	}
	if entries[0].Label() != "spot" {
		t.Fatalf("label=%q", entries[0].Label())
	}
}

func TestPersistenceAndRemove(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	repo := "/tmp/ore-demo-repo"

	_ = s.Add(repo, Bookmark{View: session.State{Main: "commits", Commit: "aaa"}, SavedAt: time.Now()})
	_ = s.Add(repo, Bookmark{View: session.State{Main: "files", Path: "x.go", Commit: "bbb"}})

	s2 := NewStore(dir)
	entries, err := s2.Get(repo)
	if err != nil || len(entries) != 2 {
		t.Fatalf("reload len=%d err=%v", len(entries), err)
	}
	if err := s2.RemoveAt(repo, 0); err != nil {
		t.Fatal(err)
	}
	entries, _ = s2.Get(repo)
	if len(entries) != 1 || entries[0].View.Path != "x.go" {
		t.Fatalf("after remove: %#v", entries)
	}
}

func TestClear(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	repo := "r"
	_ = s.Add(repo, Bookmark{View: session.State{Main: "commits"}})
	if err := s.Clear(repo); err != nil {
		t.Fatal(err)
	}
	entries, _ := s.Get(repo)
	if len(entries) != 0 {
		t.Fatalf("expected empty, got %d", len(entries))
	}
	if _, err := os.Stat(s.pathFor(repo)); !os.IsNotExist(err) {
		t.Fatalf("file should be gone: %v", err)
	}
}

func TestViewSummary(t *testing.T) {
	got := ViewSummary(session.State{Main: "blame", Path: "pkg/a.go", BlameLine: 42, Commit: "abcdefg123"})
	if got != "blame · abcdefg · pkg/a.go · L42" {
		t.Fatalf("summary=%q", got)
	}
}

func TestSameViewIgnoresChrome(t *testing.T) {
	a := session.State{Main: "files", Path: "a.go", Commit: "c", ZenContext: 3}
	b := session.State{Main: "files", Path: "a.go", Commit: "c", ZenContext: 9, DetailWrap: true}
	if !SameView(a, b) {
		t.Fatal("expected same location")
	}
}
