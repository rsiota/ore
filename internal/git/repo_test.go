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

	total, err := repo.RevCount(ctx, "", "")
	if err != nil || total != 2 {
		t.Fatalf("RevCount = %d, %v, want 2", total, err)
	}

	detail, err := repo.Show(ctx, commits[0].Hash, 3)
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

func TestShowHeaderMultilineBody(t *testing.T) {
	dir := t.TempDir()
	run := gitTestRunner(t, dir)
	run("init", "-b", "main")
	run("config", "user.email", "ore@test")
	run("config", "user.name", "ore-test")
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "f.txt")
	run("commit", "-m", "subject line", "-m", "body line one\nbody line two")

	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	commits, err := repo.CommitLog(ctx, LogOptions{MaxCount: 1})
	if err != nil || len(commits) != 1 {
		t.Fatalf("log: %v %#v", err, commits)
	}
	header, err := repo.ShowHeader(ctx, commits[0].Hash, "")
	if err != nil {
		t.Fatal(err)
	}
	if header.Commit.Subject != "subject line" {
		t.Fatalf("subject = %q", header.Commit.Subject)
	}
	if !strings.Contains(header.Body, "body line one") || !strings.Contains(header.Body, "body line two") {
		t.Fatalf("body = %q", header.Body)
	}
	if len(header.Files) != 1 || header.Stat == "" {
		t.Fatalf("files/stat: %+v %q", header.Files, header.Stat)
	}
	if header.Diff != "" {
		t.Fatal("header must not include patch")
	}
	patch, err := repo.ShowPatch(ctx, commits[0].Hash, "", 3)
	if err != nil || patch == "" {
		t.Fatalf("patch: %v %q", err, patch)
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
	if hist[0].Path != "new.txt" || hist[0].OldPath != "old.txt" || !strings.HasPrefix(hist[0].Status, "R") {
		t.Fatalf("rename hop = path=%q old=%q status=%q", hist[0].Path, hist[0].OldPath, hist[0].Status)
	}
	if hist[0].EdgeLabel() != "moved from old.txt" {
		t.Errorf("edge = %q", hist[0].EdgeLabel())
	}
	oldest := hist[len(hist)-1]
	if oldest.Path != "old.txt" {
		t.Fatalf("oldest path = %q, want old.txt", oldest.Path)
	}

	detail, err := repo.ShowPath(ctx, hist[0].Hash, "new.txt", 3)
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

	// Pre-rename commit must be shown with the path at that revision.
	oldDetail, err := repo.ShowPath(ctx, oldest.Hash, oldest.Path, 3)
	if err != nil {
		t.Fatal(err)
	}
	if oldDetail.Stat == "" && len(oldDetail.Files) == 0 {
		t.Fatal("expected path-scoped stat for old.txt at introducing commit")
	}
	tipOnOld, err := repo.ShowPath(ctx, oldest.Hash, "new.txt", 3)
	if err != nil {
		t.Fatal(err)
	}
	if tipOnOld.Stat != "" || len(tipOnOld.Files) > 0 {
		t.Fatalf("tip path on pre-rename commit should be empty; got stat=%q files=%#v", tipOnOld.Stat, tipOnOld.Files)
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

func TestListBranches(t *testing.T) {
	dir := t.TempDir()
	run := gitTestRunner(t, dir)
	run("init", "-b", "main")
	run("config", "user.email", "ore@test")
	run("config", "user.name", "ore-test")
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "a.txt")
	run("commit", "-m", "on main")
	run("branch", "feature")
	run("checkout", "-b", "other")
	if err := os.WriteFile(path, []byte("b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "a.txt")
	run("commit", "-m", "on other")
	run("checkout", "main")

	// Fake a remote-tracking ref without a network remote.
	run("update-ref", "refs/remotes/origin/main", "HEAD")
	run("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")

	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	refs, err := repo.ListBranches(ctx)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Ref{}
	for _, r := range refs {
		byName[r.Name] = r
	}
	for _, name := range []string{"main", "feature", "other", "origin/main"} {
		if _, ok := byName[name]; !ok {
			t.Fatalf("missing %q in %+v", name, refs)
		}
	}
	if _, ok := byName["origin/HEAD"]; ok {
		t.Fatal("origin/HEAD should be skipped")
	}
	if !byName["main"].Current {
		t.Fatal("main should be current")
	}
	if !byName["origin/main"].Remote {
		t.Fatal("origin/main should be remote")
	}
	if byName["feature"].Remote {
		t.Fatal("feature should be local")
	}

	// Locals before remotes.
	sawRemote := false
	for _, r := range refs {
		if r.Remote {
			sawRemote = true
		} else if sawRemote {
			t.Fatalf("local %q after remote", r.Name)
		}
	}

	log, err := repo.CommitLog(ctx, LogOptions{Rev: "other", MaxCount: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(log) == 0 || log[0].Subject != "on other" {
		t.Fatalf("log other = %+v", log)
	}
	if got := repo.RevShort(ctx, "other"); got == "" {
		t.Fatal("RevShort other empty")
	}
}
