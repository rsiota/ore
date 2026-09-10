package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestFileHistoryFollowsRename(t *testing.T) {
	dir := t.TempDir()
	run := gitTestRunner(t, dir)

	run("init", "-b", "main")
	run("config", "user.email", "ore@test")
	run("config", "user.name", "ore-test")

	oldPath := filepath.Join(dir, "old.txt")
	if err := os.WriteFile(oldPath, []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "old.txt")
	run("commit", "-m", "add old")

	run("mv", "old.txt", "new.txt")
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("v1\nv2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "new.txt")
	run("commit", "-m", "rename and edit")

	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hist, err := repo.FileHistory(ctx, "new.txt", LogOptions{MaxCount: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) < 2 {
		t.Fatalf("FileHistory len = %d, want >= 2 (follow rename); %#v", len(hist), hist)
	}
	if hist[0].Subject != "rename and edit" {
		t.Errorf("newest = %q", hist[0].Subject)
	}
	if hist[len(hist)-1].Subject != "add old" {
		t.Errorf("oldest = %q", hist[len(hist)-1].Subject)
	}

	detail, err := repo.ShowPath(ctx, hist[0].Hash, "new.txt")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Diff == "" {
		t.Fatal("expected path-scoped diff")
	}
	if strings.Contains(detail.Diff, "old.txt") && !strings.Contains(detail.Diff, "new.txt") {
		// rename patch may mention both; at least new.txt should appear
		t.Errorf("diff should mention new.txt: %s", detail.Diff)
	}
}

func gitTestRunner(t *testing.T, dir string) func(args ...string) {
	t.Helper()
	return func(args ...string) {
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
}
