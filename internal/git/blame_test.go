package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBlameAndFollowPrevious(t *testing.T) {
	dir := t.TempDir()
	run := gitTestRunner(t, dir)

	run("init", "-b", "main")
	run("config", "user.email", "ore@test")
	run("config", "user.name", "ore-test")

	path := filepath.Join(dir, "code.txt")
	if err := os.WriteFile(path, []byte("alpha\nbeta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "code.txt")
	run("commit", "-m", "add alpha beta")

	if err := os.WriteFile(path, []byte("alpha\nbeta\ngamma\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "code.txt")
	run("commit", "-m", "add gamma")

	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	lines, err := repo.Blame(ctx, "code.txt", BlameOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 {
		t.Fatalf("len = %d, want 3", len(lines))
	}
	if lines[0].Text != "alpha" || lines[2].Text != "gamma" {
		t.Fatalf("texts = %#v", []string{lines[0].Text, lines[1].Text, lines[2].Text})
	}
	if lines[2].Summary != "add gamma" {
		t.Errorf("line3 summary = %q", lines[2].Summary)
	}
	if lines[0].Hash == lines[2].Hash {
		t.Fatal("expected different commits for alpha vs gamma")
	}
	if lines[2].PreviousHash == "" {
		t.Fatal("expected previous hash on gamma line")
	}

	// Follow: blame at previous revision should still see alpha/beta.
	prev, err := repo.Blame(ctx, "code.txt", BlameOptions{Rev: lines[2].PreviousHash})
	if err != nil {
		t.Fatal(err)
	}
	if len(prev) != 2 {
		t.Fatalf("prev blame len = %d, want 2", len(prev))
	}
}
