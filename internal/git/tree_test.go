package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListTreeAndFiles(t *testing.T) {
	dir := t.TempDir()
	run := gitTestRunner(t, dir)
	run("init", "-b", "main")
	run("config", "user.email", "ore@test")
	run("config", "user.name", "ore-test")

	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("readme.txt", "hi\n")
	write("src/a.go", "package src\n")
	write("src/nested/b.go", "package nested\n")
	write("docs/spaced name.md", "note\n")
	run("add", ".")
	run("commit", "-m", "tree")

	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	root, err := repo.ListTree(ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !treeHas(root, "docs", true) || !treeHas(root, "src", true) || !treeHas(root, "readme.txt", false) {
		t.Fatalf("root = %#v", root)
	}
	if i, j := treeIndex(root, "docs"), treeIndex(root, "readme.txt"); i < 0 || j < 0 || i > j {
		t.Fatalf("dirs should sort before files: %#v", root)
	}

	src, err := repo.ListTree(ctx, "HEAD", "src")
	if err != nil {
		t.Fatal(err)
	}
	if !treeHas(src, "nested", true) || !treeHas(src, "a.go", false) {
		t.Fatalf("src = %#v", src)
	}
	for _, e := range src {
		if e.Name == "a.go" && e.Path != "src/a.go" {
			t.Fatalf("a.go path = %q", e.Path)
		}
	}

	nested, err := repo.ListTree(ctx, "HEAD", "src/nested")
	if err != nil {
		t.Fatal(err)
	}
	if len(nested) != 1 || nested[0].Path != "src/nested/b.go" || nested[0].Dir {
		t.Fatalf("nested = %#v", nested)
	}

	files, err := repo.ListFiles(ctx, "HEAD", "")
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(files, "readme.txt") || !containsStr(files, "src/a.go") ||
		!containsStr(files, "src/nested/b.go") || !containsStr(files, "docs/spaced name.md") {
		t.Fatalf("files = %#v", files)
	}

	srcFiles, err := repo.ListFiles(ctx, "", "src")
	if err != nil {
		t.Fatal(err)
	}
	if containsStr(srcFiles, "readme.txt") || !containsStr(srcFiles, "src/a.go") {
		t.Fatalf("src files = %#v", srcFiles)
	}
}

func TestTreePathHelpers(t *testing.T) {
	if TreeParent("src/ui/app.go") != "src/ui" {
		t.Fatalf("parent = %q", TreeParent("src/ui/app.go"))
	}
	if TreeParent("readme.txt") != "" {
		t.Fatalf("root parent = %q", TreeParent("readme.txt"))
	}
	if TreeBase("src/ui/app.go") != "app.go" {
		t.Fatalf("base = %q", TreeBase("src/ui/app.go"))
	}
	got := TreeAncestors("src/ui/app.go")
	if len(got) != 3 || got[0] != "" || got[1] != "src" || got[2] != "src/ui" {
		t.Fatalf("ancestors = %#v", got)
	}
}

func treeHas(entries []TreeEntry, name string, dir bool) bool {
	for _, e := range entries {
		if e.Name == name && e.Dir == dir {
			return true
		}
	}
	return false
}

func treeIndex(entries []TreeEntry, name string) int {
	for i, e := range entries {
		if e.Name == name {
			return i
		}
	}
	return -1
}

func containsStr(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
