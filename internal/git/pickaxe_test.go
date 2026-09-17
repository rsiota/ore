package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPickaxeStringFindsIntroducingCommit(t *testing.T) {
	dir := t.TempDir()
	run := gitTestRunner(t, dir)

	run("init", "-b", "main")
	run("config", "user.email", "ore@test")
	run("config", "user.name", "ore-test")

	p := filepath.Join(dir, "app.go")
	if err := os.WriteFile(p, []byte("package app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "app.go")
	run("commit", "-m", "scaffold")

	if err := os.WriteFile(p, []byte("package app\nfunc UniqueToken() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "app.go")
	run("commit", "-m", "add UniqueToken")

	if err := os.WriteFile(p, []byte("package app\nfunc UniqueToken() {}\nfunc other() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "app.go")
	run("commit", "-m", "add other")

	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hits, err := repo.Pickaxe(ctx, PickaxeOptions{Query: "UniqueToken", Mode: PickaxeString})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected pickaxe hits")
	}
	if hits[0].Commit.Subject != "add UniqueToken" {
		t.Fatalf("top hit = %#v, want add UniqueToken", hits[0])
	}
	if len(hits[0].Paths) != 1 || hits[0].Paths[0] != "app.go" {
		t.Fatalf("paths = %#v", hits[0].Paths)
	}

	reHits, err := repo.Pickaxe(ctx, PickaxeOptions{Query: "UniqueTo[a-z]+", Mode: PickaxeRegexp})
	if err != nil {
		t.Fatal(err)
	}
	if len(reHits) == 0 {
		t.Fatal("expected regexp pickaxe hits")
	}
}
