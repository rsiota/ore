package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestSummarizeAuthors(t *testing.T) {
	got := SummarizeAuthors([]string{"alice", "bob", "alice", "carol", "alice", "bob", ""}, 3)
	if len(got) != 3 {
		t.Fatalf("got %#v", got)
	}
	if got[0].Name != "alice" || got[0].Count != 3 || got[0].Pct != 50 {
		t.Fatalf("alice = %#v", got[0])
	}
	if got[1].Name != "bob" || got[1].Count != 2 {
		t.Fatalf("bob = %#v", got[1])
	}
}

func TestPathOwnership(t *testing.T) {
	dir := t.TempDir()
	commitAs := func(name, email string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME="+name,
			"GIT_AUTHOR_EMAIL="+email,
			"GIT_COMMITTER_NAME="+name,
			"GIT_COMMITTER_EMAIL="+email,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	init := gitTestRunner(t, dir)
	init("init", "-b", "main")
	init("config", "user.email", "ore@test")
	init("config", "user.name", "ore-test")

	p := filepath.Join(dir, "a.go")
	if err := os.WriteFile(p, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	commitAs("alice", "alice@test", "add", "a.go")
	commitAs("alice", "alice@test", "commit", "-m", "a1")

	if err := os.WriteFile(p, []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	commitAs("bob", "bob@test", "add", "a.go")
	commitAs("bob", "bob@test", "commit", "-m", "b1")

	if err := os.WriteFile(p, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	commitAs("alice", "alice@test", "add", "a.go")
	commitAs("alice", "alice@test", "commit", "-m", "a2")

	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	own, err := repo.PathOwnership(ctx, "HEAD", []string{"a.go"}, 50, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(own) < 2 || own[0].Name != "alice" || own[0].Count != 2 {
		t.Fatalf("ownership=%#v", own)
	}
	if own[1].Name != "bob" || own[1].Count != 1 {
		t.Fatalf("ownership=%#v", own)
	}

	alice, err := repo.PathAuthorCommits(ctx, "alice", "HEAD", []string{"a.go"}, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(alice) != 2 {
		t.Fatalf("alice hits = %#v", alice)
	}
	for _, h := range alice {
		if h.Commit.Author != "alice" {
			t.Fatalf("want alice, got %#v", h)
		}
	}

	bob, err := repo.PathAuthorCommits(ctx, "bob", "HEAD", []string{"a.go"}, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(bob) != 1 || bob[0].Commit.Author != "bob" {
		t.Fatalf("bob hits = %#v", bob)
	}
}
