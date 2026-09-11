package ui

import (
	"testing"

	"github.com/rsiota/ore/internal/git"
)

func TestFilterFileIndicesCol(t *testing.T) {
	files := []git.FileChange{
		{Path: "a.go", Status: "M", Additions: 1, Deletions: 0},
		{Path: "b.txt", Status: "A", Additions: 10, Deletions: 2},
		{Path: "old.go", OldPath: "prev.go", Status: "R", Additions: 0, Deletions: 0},
	}
	idx := filterFileIndicesCol(files, "b.txt", fileColPath)
	if len(idx) != 1 || idx[0] != 1 {
		t.Fatalf("path filter: got %v", idx)
	}
	idx = filterFileIndicesCol(files, "prev", fileColPath)
	if len(idx) != 1 || idx[0] != 2 {
		t.Fatalf("old path filter: got %v", idx)
	}
	idx = filterFileIndicesCol(files, "A", fileColStatus)
	if len(idx) != 1 || idx[0] != 1 {
		t.Fatalf("status filter: got %v", idx)
	}
}

func TestSortFileIndices(t *testing.T) {
	files := []git.FileChange{
		{Path: "a", Additions: 5, Deletions: 1},
		{Path: "b", Additions: 1, Deletions: 9},
		{Path: "c", Additions: 3, Deletions: 3},
	}
	base := []int{0, 1, 2}
	asc := sortFileIndices(files, base, fileColAdd, SortAsc)
	if asc[0] != 1 || asc[1] != 2 || asc[2] != 0 {
		t.Fatalf("add asc: got %v", asc)
	}
	desc := sortFileIndices(files, base, fileColDel, SortDesc)
	if desc[0] != 1 || desc[1] != 2 || desc[2] != 0 {
		t.Fatalf("del desc: got %v", desc)
	}
}

func TestFileRowRename(t *testing.T) {
	row := fileRow(git.FileChange{Path: "new.go", OldPath: "old.go", Status: "R"})
	if row[fileColPath] != "old.go → new.go" {
		t.Fatalf("path cell = %q", row[fileColPath])
	}
}
