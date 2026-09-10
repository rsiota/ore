package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenAndCommitLog(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=ore-test",
			"GIT_AUTHOR_EMAIL=ore@test",
			"GIT_COMMITTER_NAME=ore-test",
			"GIT_COMMITTER_EMAIL=ore@test",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run("init", "-b", "main")
	run("config", "user.email", "ore@test")
	run("config", "user.name", "ore-test")

	path := filepath.Join(dir, "readme.txt")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "readme.txt")
	run("commit", "-m", "initial commit")

	if err := os.WriteFile(path, []byte("hello\nworld\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "readme.txt")
	run("commit", "-m", "add world")

	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if repo.Path != dir {
		// macOS /var vs /private/var — compare via EvalSymlinks
		want, _ := filepath.EvalSymlinks(dir)
		got, _ := filepath.EvalSymlinks(repo.Path)
		if got != want {
			t.Fatalf("Path = %q, want %q", repo.Path, dir)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	commits, err := repo.CommitLog(ctx, LogOptions{MaxCount: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 2 {
		t.Fatalf("len(commits) = %d, want 2", len(commits))
	}
	if commits[0].Subject != "add world" {
		t.Errorf("newest subject = %q", commits[0].Subject)
	}
	if commits[1].Subject != "initial commit" {
		t.Errorf("oldest subject = %q", commits[1].Subject)
	}

	detail, err := repo.Show(ctx, commits[0].Hash)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Commit.Subject != "add world" {
		t.Errorf("detail subject = %q", detail.Commit.Subject)
	}
	if len(detail.Files) != 1 || detail.Files[0].Path != "readme.txt" {
		t.Errorf("files = %+v", detail.Files)
	}
	if detail.Diff == "" {
		t.Error("expected non-empty diff")
	}
	if repo.BranchName(ctx) != "main" {
		t.Errorf("branch = %q", repo.BranchName(ctx))
	}
}

func TestOpenNotRepo(t *testing.T) {
	dir := t.TempDir()
	_, err := Open(dir)
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsNotRepository(err) {
		t.Fatalf("err = %v, want ErrNotRepository", err)
	}
}
