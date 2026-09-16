package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCoChangedFilesRanksPartners(t *testing.T) {
	dir := t.TempDir()
	run := gitTestRunner(t, dir)

	run("init", "-b", "main")
	run("config", "user.email", "ore@test")
	run("config", "user.name", "ore-test")

	write := func(name, body string) {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Seed pair: a.go + b.go change together often; c.go is a loner.
	write("a.go", "a1\n")
	write("b.go", "b1\n")
	run("add", "a.go", "b.go")
	run("commit", "-m", "pair 1")

	write("a.go", "a2\n")
	write("b.go", "b2\n")
	run("add", "a.go", "b.go")
	run("commit", "-m", "pair 2")

	write("a.go", "a3\n")
	write("b.go", "b3\n")
	run("add", "a.go", "b.go")
	run("commit", "-m", "pair 3")

	write("a.go", "a4\n")
	write("c.go", "c1\n")
	run("add", "a.go", "c.go")
	run("commit", "-m", "a with c once")

	write("c.go", "c2\n")
	run("add", "c.go")
	run("commit", "-m", "c alone")

	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log, err := repo.CommitLog(ctx, LogOptions{MaxCount: 10})
	if err != nil || len(log) == 0 {
		t.Fatalf("log: %v", err)
	}
	// Newest commit is c alone — use an earlier commit that touches a.go.
	var seedHash string
	for _, c := range log {
		if c.Subject == "a with c once" {
			seedHash = c.Hash
			break
		}
	}
	if seedHash == "" {
		t.Fatal("seed commit not found")
	}

	hot, err := repo.CoChangedFiles(ctx, []string{"a.go"}, seedHash)
	if err != nil {
		t.Fatal(err)
	}
	if len(hot) == 0 {
		t.Fatal("expected co-changed paths")
	}
	if hot[0].Path != "b.go" {
		t.Fatalf("top hot spot = %#v, want b.go first", hot)
	}
	if hot[0].Count < 2 {
		t.Fatalf("b.go count = %d, want >= 2", hot[0].Count)
	}
	for _, h := range hot {
		if h.Path == "a.go" {
			t.Fatal("seed path should not appear in hot spots")
		}
	}

	rel, err := repo.Relations(ctx, seedHash)
	if err != nil {
		t.Fatal(err)
	}
	if len(rel.HotSpots) == 0 {
		t.Fatal("Relations should populate HotSpots")
	}
	if rel.HotSpots[0].Path != "b.go" {
		t.Fatalf("Relations HotSpots = %#v", rel.HotSpots)
	}
}
