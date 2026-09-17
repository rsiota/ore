package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLineEvolutionWalksPrevious(t *testing.T) {
	dir := t.TempDir()
	run := gitTestRunner(t, dir)

	run("init", "-b", "main")
	run("config", "user.email", "ore@test")
	run("config", "user.name", "ore-test")

	path := filepath.Join(dir, "code.txt")
	if err := os.WriteFile(path, []byte("alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "code.txt")
	run("commit", "-m", "add alpha")

	if err := os.WriteFile(path, []byte("alpha\nbeta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "code.txt")
	run("commit", "-m", "add beta")

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
	if len(lines) < 3 {
		t.Fatalf("blame len = %d", len(lines))
	}
	gamma := lines[2]
	if gamma.PreviousHash == "" {
		t.Fatal("gamma should have previous")
	}

	stack, err := repo.LineEvolution(ctx, "code.txt", "HEAD", gamma, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(stack) < 2 {
		t.Fatalf("evolution len = %d, want >= 2: %#v", len(stack), stack)
	}
	if stack[0].Line.Text != "gamma" || stack[0].Index != 0 {
		t.Fatalf("step0 = %#v", stack[0])
	}
	// Walking previous should leave the introducing commit of gamma.
	if stack[1].Rev != gamma.PreviousHash && !hashMatch(stack[1].Rev, gamma.PreviousHash) {
		t.Fatalf("step1 rev = %s, want previous %s", stack[1].Rev, gamma.PreviousHash)
	}
}
