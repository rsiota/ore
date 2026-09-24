package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRelationsParentsChildrenFiles(t *testing.T) {
	dir := t.TempDir()
	run := gitTestRunner(t, dir)

	run("init", "-b", "main")
	run("config", "user.email", "ore@test")
	run("config", "user.name", "ore-test")

	p := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(p, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "a.txt")
	run("commit", "-m", "first")

	if err := os.WriteFile(p, []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "a.txt")
	run("commit", "-m", "second")

	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log, err := repo.CommitLog(ctx, LogOptions{MaxCount: 5})
	if err != nil || len(log) < 2 {
		t.Fatalf("log: %v %#v", err, log)
	}
	newest, oldest := log[0], log[1]

	rel, err := repo.Relations(ctx, newest.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if len(rel.Parents) != 1 || rel.Parents[0].Hash != oldest.Hash {
		t.Fatalf("parents = %#v, want %s", rel.Parents, oldest.Hash)
	}
	if len(rel.Files) != 1 || rel.Files[0].Path != "a.txt" {
		t.Fatalf("files = %#v", rel.Files)
	}

	relOld, err := repo.Relations(ctx, oldest.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if len(relOld.Children) != 1 {
		t.Fatalf("children of oldest = %#v, want 1", relOld.Children)
	}
	if relOld.Children[0].Hash != newest.Hash {
		t.Fatalf("child hash = %s, want %s", relOld.Children[0].Hash, newest.Hash)
	}
}

func TestMergeBaseAndChildrenOf(t *testing.T) {
	dir := t.TempDir()
	run := gitTestRunner(t, dir)

	run("init", "-b", "main")
	run("config", "user.email", "ore@test")
	run("config", "user.name", "ore-test")

	p := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(p, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "a.txt")
	run("commit", "-m", "root")

	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log, err := repo.CommitLog(ctx, LogOptions{MaxCount: 5})
	if err != nil || len(log) < 1 {
		t.Fatalf("log=%v err=%v", log, err)
	}
	root := log[len(log)-1].Hash

	if err := os.WriteFile(p, []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "a.txt")
	run("commit", "-m", "tip")

	log, err = repo.CommitLog(ctx, LogOptions{MaxCount: 5})
	if err != nil || len(log) < 2 {
		t.Fatalf("log2=%v err=%v", log, err)
	}
	tip := log[0].Hash

	base, err := repo.MergeBase(ctx, tip, tip)
	if err != nil || base != tip {
		t.Fatalf("merge-base(tip,tip)=%s err=%v", base, err)
	}
	base, err = repo.MergeBase(ctx, tip, root)
	if err != nil || base != root {
		t.Fatalf("merge-base(tip,root)=%s want %s err=%v", base, root, err)
	}

	kids, err := repo.ChildrenOf(ctx, root)
	if err != nil || len(kids) != 1 || kids[0] != tip {
		t.Fatalf("children=%v err=%v want [%s]", kids, err, tip)
	}

	toward, err := repo.ChildrenToward(ctx, root, tip)
	if err != nil || len(toward) != 1 || toward[0] != tip {
		t.Fatalf("toward=%v err=%v want [%s]", toward, err, tip)
	}

	// Cache must not hide a new child until ResetGraph.
	if err := os.WriteFile(p, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "a.txt")
	run("commit", "-m", "newer")
	log, err = repo.CommitLog(ctx, LogOptions{MaxCount: 5})
	if err != nil || len(log) < 3 {
		t.Fatalf("log3=%v err=%v", log, err)
	}
	newer := log[0].Hash
	stale, err := repo.ChildrenOf(ctx, tip)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Fatalf("cached children of tip = %v, want empty before reset", stale)
	}
	repo.ResetGraph()
	fresh, err := repo.ChildrenOf(ctx, tip)
	if err != nil || len(fresh) != 1 || fresh[0] != newer {
		t.Fatalf("after reset children=%v err=%v want [%s]", fresh, err, newer)
	}
}

func TestChildrenAmong(t *testing.T) {
	parent := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	child := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	other := "cccccccccccccccccccccccccccccccccccccccc"
	commits := []Commit{
		{Hash: child, Parents: []string{parent}},
		{Hash: other, Parents: []string{"dddddddddddddddddddddddddddddddddddddddd"}},
		{Hash: parent},
	}
	got := ChildrenAmong(commits, parent)
	if len(got) != 1 || got[0] != child {
		t.Fatalf("got %v, want [%s]", got, child)
	}
	if ChildrenAmong(commits, "") != nil {
		t.Fatal("empty hash should yield nil")
	}
}
