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
