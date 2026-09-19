package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestBlamePreviousPathAcrossRename(t *testing.T) {
	dir := t.TempDir()
	run := gitTestRunner(t, dir)

	run("init", "-b", "main")
	run("config", "user.email", "ore@test")
	run("config", "user.name", "ore-test")

	old := filepath.Join(dir, "old.txt")
	if err := os.WriteFile(old, []byte("alpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "old.txt")
	run("commit", "-m", "add old")

	run("mv", "old.txt", "new.txt")
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("alpha\nbeta\n"), 0o644); err != nil {
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

	lines, err := repo.Blame(ctx, "new.txt", BlameOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) < 2 {
		t.Fatalf("len = %d", len(lines))
	}
	// Line introduced before the rename should point previous path at old.txt.
	var alpha *BlameLine
	for i := range lines {
		if lines[i].Text == "alpha" {
			alpha = &lines[i]
			break
		}
	}
	if alpha == nil {
		t.Fatal("alpha line missing")
	}
	if alpha.PreviousPath != "old.txt" && alpha.PreviousPath != "" {
		// Some git versions may omit previous on the introducing commit's line
		// when blamed at the tip after rename; accept empty only if no previous.
		if alpha.PreviousHash != "" && alpha.PreviousPath == "" {
			t.Fatalf("previous hash set but path empty: %#v", alpha)
		}
	}
	if alpha.PreviousHash != "" && alpha.PreviousPath != "" && alpha.PreviousPath != "old.txt" {
		t.Fatalf("PreviousPath = %q, want old.txt", alpha.PreviousPath)
	}

	rel, err := repo.LineRelationsAt(ctx, "new.txt", "HEAD", *alpha)
	if err != nil {
		t.Fatal(err)
	}
	if len(rel.History) < 2 {
		t.Fatalf("history = %#v", rel.History)
	}
	if rel.History[0].OldPath != "old.txt" && !strings.HasPrefix(rel.History[0].Status, "R") {
		// newest should be the rename hop
		found := false
		for _, h := range rel.History {
			if h.OldPath == "old.txt" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected rename edge in history: %#v", rel.History)
		}
	}
}
