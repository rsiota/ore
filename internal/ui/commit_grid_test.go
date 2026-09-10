package ui

import (
	"testing"
	"time"

	"github.com/rsiota/ore/internal/git"
)

func sampleCommits() []git.Commit {
	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	return []git.Commit{
		{Hash: "aaa111", ShortHash: "aaa111", Author: "alice", Subject: "Add auth", Date: t2},
		{Hash: "bbb222", ShortHash: "bbb222", Author: "bob", Subject: "Fix typo", Date: t1},
		{Hash: "ccc333", ShortHash: "ccc333", Author: "alice", Subject: "Refactor db", Date: t3},
	}
}

func TestFilterCommitIndicesCol(t *testing.T) {
	cs := sampleCommits()
	idx := filterCommitIndicesCol(cs, "alice", commitColAuthor)
	if len(idx) != 2 || idx[0] != 0 || idx[1] != 2 {
		t.Fatalf("author filter: got %v", idx)
	}
	idx = filterCommitIndicesCol(cs, "auth", commitColSubject)
	if len(idx) != 1 || idx[0] != 0 {
		t.Fatalf("subject filter: got %v", idx)
	}
	idx = filterCommitIndicesCol(cs, "bbb", commitColHash)
	if len(idx) != 1 || idx[0] != 1 {
		t.Fatalf("hash filter: got %v", idx)
	}
	idx = filterCommitIndicesCol(cs, "", commitColAuthor)
	if len(idx) != 3 {
		t.Fatalf("empty query: got %v", idx)
	}
}

func TestSortCommitIndices(t *testing.T) {
	cs := sampleCommits()
	base := []int{0, 1, 2}

	asc := sortCommitIndices(cs, base, commitColDate, SortAsc)
	if asc[0] != 1 || asc[1] != 0 || asc[2] != 2 {
		t.Fatalf("date asc: got %v", asc)
	}
	desc := sortCommitIndices(cs, base, commitColDate, SortDesc)
	if desc[0] != 2 || desc[1] != 0 || desc[2] != 1 {
		t.Fatalf("date desc: got %v", desc)
	}
	byAuthor := sortCommitIndices(cs, base, commitColAuthor, SortAsc)
	if byAuthor[0] != 0 || byAuthor[1] != 2 || byAuthor[2] != 1 {
		t.Fatalf("author asc: got %v", byAuthor)
	}
	none := sortCommitIndices(cs, base, commitColDate, SortNone)
	if none[0] != 0 || none[1] != 1 || none[2] != 2 {
		t.Fatalf("sort none should preserve order: got %v", none)
	}
}

func TestCycleSort(t *testing.T) {
	if got := CycleSort(SortNone); got != SortAsc {
		t.Fatalf("none→asc: %v", got)
	}
	if got := CycleSort(SortAsc); got != SortDesc {
		t.Fatalf("asc→desc: %v", got)
	}
	if got := CycleSort(SortDesc); got != SortNone {
		t.Fatalf("desc→none: %v", got)
	}
}

func TestGridClampAndMove(t *testing.T) {
	g := Grid{
		Columns: []string{"A", "B"},
		Rows:    [][]string{{"1", "2"}, {"3", "4"}, {"5", "6"}},
		Width:   40,
		Height:  10,
	}
	g.CursorRow = 99
	g.CursorCol = 99
	g.ClampCursor()
	if g.CursorRow != 2 || g.CursorCol != 1 {
		t.Fatalf("clamp: row=%d col=%d", g.CursorRow, g.CursorCol)
	}
	g.MoveLeft()
	if g.CursorCol != 0 {
		t.Fatalf("move left: col=%d", g.CursorCol)
	}
	g.Top()
	if g.CursorRow != 0 {
		t.Fatalf("top: row=%d", g.CursorRow)
	}
}
