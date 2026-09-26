package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadClear(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	repo := "/tmp/demo-repo"

	st := State{
		Main:       "blame",
		Commit:     "abc123",
		Path:       "internal/ui/app.go",
		BlameRev:   "abc123",
		BlameLine:  42,
		BlameFrom:  "files",
		DiffMode:   "zen",
		ZenContext: 3,
	}
	if err := store.Save(repo, st); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "sessions")
	entries, err := os.ReadDir(path)
	if err != nil || len(entries) != 1 {
		t.Fatalf("sessions dir: %v entries=%v", err, entries)
	}

	loaded, err := store.Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.HasContent() || loaded.Main != "blame" || loaded.BlameLine != 42 {
		t.Fatalf("loaded = %#v", loaded)
	}

	// Cache hit
	again, err := store.Load(repo)
	if err != nil || again.Path != st.Path {
		t.Fatalf("cache load: %#v %v", again, err)
	}

	if err := store.Clear(repo); err != nil {
		t.Fatal(err)
	}
	empty, err := store.Load(repo)
	if err != nil || empty.HasContent() {
		t.Fatalf("after clear: %#v %v", empty, err)
	}
}

func TestSaveLoadPickaxeEvolveHunks(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	repo := "/tmp/demo-modes"
	st := State{
		Main:        "pickaxe",
		PickQuery:   "token",
		PickMode:    "regexp",
		PickPath:    "a.go",
		CoupleSeeds: []string{"a.go"},
		CoupleWith:  "b.go",
		Author:      "alice",
		Hunks:       true,
		EvoStep:     2,
	}
	if err := store.Save(repo, st); err != nil {
		t.Fatal(err)
	}
	store.cache = map[string]State{}
	loaded, err := store.Load(repo)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Main != "pickaxe" || loaded.PickQuery != "token" || loaded.PickMode != "regexp" {
		t.Fatalf("pickaxe = %#v", loaded)
	}
	if !loaded.Hunks || loaded.EvoStep != 2 || loaded.CoupleWith != "b.go" || loaded.Author != "alice" {
		t.Fatalf("extra fields = %#v", loaded)
	}
}

func TestFileKeyStable(t *testing.T) {
	a := fileKey("/Users/x/code/ore")
	b := fileKey("/Users/x/code/ore")
	if a != b || a == "" {
		t.Fatalf("unstable key %q %q", a, b)
	}
	if fileKey("/Users/x/code/other") == a {
		t.Fatal("different repos should not collide")
	}
}
