package ui

import (
	"testing"

	"github.com/rsiota/ore/internal/git"
)

func sampleTree() fileTree {
	t := newFileTree("HEAD", "")
	t.applyKids("", []git.TreeEntry{
		{Name: "src", Path: "src", Dir: true},
		{Name: "readme.txt", Path: "readme.txt"},
	})
	t.applyKids("src", []git.TreeEntry{
		{Name: "nested", Path: "src/nested", Dir: true},
		{Name: "a.go", Path: "src/a.go"},
	})
	t.applyKids("src/nested", []git.TreeEntry{
		{Name: "b.go", Path: "src/nested/b.go"},
	})
	return t
}

func TestFileTreeWalkExpand(t *testing.T) {
	tr := sampleTree()
	rows := tr.rows()
	if len(rows) != 2 || rows[0].name != "src" || rows[1].name != "readme.txt" {
		t.Fatalf("collapsed rows = %#v", rows)
	}
	tr.cursor = 0
	if prefix, ok := tr.toggleExpand(); !ok || prefix != "" {
		t.Fatalf("expand src: prefix=%q ok=%v", prefix, ok)
	}
	rows = tr.rows()
	if len(rows) != 4 || rows[1].path != "src/nested" || rows[2].path != "src/a.go" {
		t.Fatalf("expanded src rows = %#v", rows)
	}
}

func TestFileTreeSelectPathExpandsAncestors(t *testing.T) {
	tr := sampleTree()
	if !tr.selectPath("src/nested/b.go") {
		t.Fatal("selectPath failed")
	}
	row, ok := tr.selected()
	if !ok || row.path != "src/nested/b.go" {
		t.Fatalf("selected = %#v ok=%v", row, ok)
	}
	if !tr.expanded["src"] || !tr.expanded["src/nested"] {
		t.Fatalf("ancestors not expanded: %#v", tr.expanded)
	}
}

func TestFileTreeCollapseOrParent(t *testing.T) {
	tr := sampleTree()
	tr.selectPath("src/a.go")
	if !tr.collapseOrParent() {
		t.Fatal("expected move to parent")
	}
	row, _ := tr.selected()
	if row.path != "src" {
		t.Fatalf("parent = %q", row.path)
	}
	if !tr.collapseOrParent() {
		t.Fatal("expected collapse src")
	}
	if tr.expanded["src"] {
		t.Fatal("src still expanded")
	}
	if tr.collapseOrParent() {
		t.Fatal("root should not walk further")
	}
}

func TestFileTreeFilterRows(t *testing.T) {
	tr := sampleTree()
	tr.applyMatches([]string{"readme.txt", "src/a.go", "src/nested/b.go"})
	tr.filter = "a.go"
	rows := tr.rows()
	if len(rows) != 1 || rows[0].path != "src/a.go" {
		t.Fatalf("filter rows = %#v", rows)
	}
}

func TestFileTreeMissingPrefixes(t *testing.T) {
	tr := newFileTree("HEAD", "src/nested/b.go")
	got := tr.missingPrefixes("src/nested/b.go")
	if len(got) != 3 || got[0] != "" || got[1] != "src" || got[2] != "src/nested" {
		t.Fatalf("missing = %#v", got)
	}
	tr.applyKids("", []git.TreeEntry{{Name: "src", Path: "src", Dir: true}})
	got = tr.missingPrefixes("src/nested/b.go")
	if len(got) != 2 || got[0] != "src" || got[1] != "src/nested" {
		t.Fatalf("after root missing = %#v", got)
	}
}
