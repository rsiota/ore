package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/rsiota/ore/internal/git"
)

func TestBlameRowCells(t *testing.T) {
	bl := git.BlameLine{
		Line: 12, ShortHash: "abc1234", Author: "alice",
		When: time.Now().Add(-2 * time.Hour),
		Text: "\tfmt.Println()",
	}
	row := blameRow(bl)
	if row[blameColLine] != "12" || row[blameColCommit] != "abc1234" {
		t.Fatalf("row = %#v", row)
	}
	if !strings.HasPrefix(row[blameColCode], "    ") {
		t.Fatalf("tabs should expand: %q", row[blameColCode])
	}
}

func TestFilterSortBlameIndices(t *testing.T) {
	lines := []git.BlameLine{
		{Line: 1, Hash: "aaa", ShortHash: "aaa", Author: "bob", Text: "one"},
		{Line: 2, Hash: "bbb", ShortHash: "bbb", Author: "alice", Text: "two"},
		{Line: 3, Hash: "ccc", ShortHash: "ccc", Author: "alice", Text: "three"},
	}
	idx := filterBlameIndicesCol(lines, "alice", blameColAuthor)
	if len(idx) != 2 {
		t.Fatalf("author filter: %v", idx)
	}
	sorted := sortBlameIndices(lines, []int{0, 1, 2}, blameColAuthor, SortAsc)
	if sorted[0] != 1 && sorted[0] != 2 {
		t.Fatalf("author sort: %v", sorted)
	}
}

func TestBlameGridHybridChrome(t *testing.T) {
	newest := time.Now()
	oldest := newest.Add(-48 * time.Hour)
	lines := []git.BlameLine{
		{Line: 1, ShortHash: "aaa", Author: "bob", When: newest, Text: "new"},
		{Line: 2, ShortHash: "bbb", Author: "alice", When: oldest, Text: "old"},
	}
	g := Grid{
		Columns:             blameColumns,
		Rows:                [][]string{blameRow(lines[0]), blameRow(lines[1])},
		CursorRow:           0,
		CursorCol:           blameColAuthor,
		Width:               80,
		Height:              8,
		Focused:             true,
		NoStripe:            true,
		SoftCursor:          true,
		SkipCursorPaintCols: []int{blameColCode},
		MuteCols:            []int{blameColLine, blameColCommit, blameColAge, blameColAuthor},
		CellStyle: func(row, col int, text string) (string, bool) {
			if col != blameColCode {
				return "", false
			}
			return blameAgeStyle(lines[row].When, newest, oldest).Render(text), true
		},
	}
	g.AutoWidths()

	empty := g.renderEmptyRow(1)
	if strings.Contains(empty, "\x1b[48;") {
		t.Fatalf("NoStripe empty row should not paint backgrounds: %q", empty)
	}

	row0 := g.renderRow(0)
	// Active meta cell uses blue cursor; code still age-washed (no blue on code).
	if !strings.Contains(row0, "bob") {
		t.Fatalf("expected author in row: %q", row0)
	}
	g.CursorCol = blameColCode
	codeFocus := g.renderRow(0)
	if !strings.Contains(codeFocus, "new") {
		t.Fatalf("code cursor should keep code text: %q", codeFocus)
	}
	_ = g.View()
}
