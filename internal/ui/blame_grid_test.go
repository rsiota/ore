package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsiota/ore/internal/git"
)

func TestBlameRowCells(t *testing.T) {
	bl := git.BlameLine{
		Line: 12, ShortHash: "abc1234", Author: "alice",
		When: time.Now().Add(-2 * time.Hour),
		Text: "\tfmt.Println()",
	}
	row := blameRow(bl)
	if row[blameColLine] != " 12" || row[blameColCommit] != "abc1234" {
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
		RightAlignCols:      []int{blameColLine},
		MuteCols:            []int{blameColCommit, blameColAge},
		CellStyle: func(row, col int, text string) (string, bool) {
			switch col {
			case blameColLine:
				return styleZenHunk.Render(text), true
			case blameColAuthor:
				if strings.TrimSpace(text) == "" {
					return "", false
				}
				return authorNameStyle(lines[row].Author).Render(text), true
			case blameColCode:
				return blameAgeStyle(lines[row].When, newest, oldest).Render(text), true
			default:
				return "", false
			}
		},
	}
	g.AutoWidthsCaps(blameMetaMaxContent)

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

func TestBlameRowSuppressesRepeatedMeta(t *testing.T) {
	lines := []git.BlameLine{
		{Line: 1, Hash: "aaa", ShortHash: "aaa", Author: "alice", When: time.Now(), Text: "one"},
		{Line: 2, Hash: "aaa", ShortHash: "aaa", Author: "alice", When: time.Now(), Text: "two"},
		{Line: 3, Hash: "bbb", ShortHash: "bbb", Author: "bob", When: time.Now(), Text: "three"},
	}
	idx := []int{0, 1, 2}
	r0 := blameRowDisplay(lines, idx, 0)
	r1 := blameRowDisplay(lines, idx, 1)
	r2 := blameRowDisplay(lines, idx, 2)
	if r0[blameColCommit] == "" || r0[blameColAuthor] == "" {
		t.Fatalf("first row should show meta: %#v", r0)
	}
	if r1[blameColCommit] != "" || r1[blameColAge] != "" || r1[blameColAuthor] != "" {
		t.Fatalf("same-commit row should blank meta: %#v", r1)
	}
	if r1[blameColLine] != " 2" || r1[blameColCode] != "two" {
		t.Fatalf("line/code must remain: %#v", r1)
	}
	if r2[blameColCommit] != "bbb" || r2[blameColAuthor] != "bob" {
		t.Fatalf("new commit should show meta: %#v", r2)
	}
}

func TestBlameReloadSkipsSameCommit(t *testing.T) {
	m := Model{
		main: MainBlame,
		blame: []git.BlameLine{
			{Line: 1, Hash: "aaaaaaaa", ShortHash: "aaaaaaa", Text: "a"},
			{Line: 2, Hash: "aaaaaaaa", ShortHash: "aaaaaaa", Text: "b"},
			{Line: 3, Hash: "bbbbbbbb", ShortHash: "bbbbbbb", Text: "c"},
		},
		blameCursor: 0,
		detail: &git.CommitDetail{
			Commit: git.Commit{Hash: "aaaaaaaa"},
		},
		detailCache: &detailRenderCache{},
	}
	prev := m.selectedHash()
	m.blameCursor = 1
	if cmd := m.blameReloadIfCommitChanged(prev); cmd != nil {
		t.Fatal("same commit should not reload detail")
	}
	prev = m.selectedHash()
	m.blameCursor = 2
	if cmd := m.blameReloadIfCommitChanged(prev); cmd == nil {
		t.Fatal("new commit should reload detail")
	}
}

func TestDisplaySkipAndCodeScroll(t *testing.T) {
	if got := displaySkip("abcdef", 2); got != "cdef" {
		t.Fatalf("displaySkip = %q", got)
	}
	g := Grid{
		Columns:    []string{"code"},
		Rows:       [][]string{{"abcdefghijklmnopqrstuvwxyz"}},
		Widths:     []int{12},
		Width:      12,
		Height:     3,
		HScrollCol: 0,
		HScroll:    4,
	}
	row := g.renderRow(0)
	if !strings.Contains(row, "…") {
		t.Fatalf("scrolled code should show ellipsis: %q", row)
	}
}

func TestBlameDefaultFoldShowsLineCommitCode(t *testing.T) {
	hidden := blameHiddenCols(blameGutterFoldDefault)
	if len(hidden) != 2 || hidden[0] != blameColAuthor || hidden[1] != blameColAge {
		t.Fatalf("default fold should hide author+age, got %v", hidden)
	}
	if blameColIsHidden(blameGutterFoldDefault, blameColCommit) ||
		blameColIsHidden(blameGutterFoldDefault, blameColLine) ||
		blameColIsHidden(blameGutterFoldDefault, blameColCode) {
		t.Fatal("default fold must keep line, commit, and code")
	}
}

func TestBlameGutterFoldHidesMeta(t *testing.T) {
	if got := blameHiddenCols(0); got != nil {
		t.Fatalf("fold 0: %v", got)
	}
	if got := blameHiddenCols(1); len(got) != 1 || got[0] != blameColAuthor {
		t.Fatalf("fold 1: %v", got)
	}
	if got := blameHiddenCols(3); len(got) != 3 {
		t.Fatalf("fold 3: %v", got)
	}
	lines := []git.BlameLine{
		{Line: 1, ShortHash: "abcdef1", Author: "alice", Text: "code here"},
	}
	g := Grid{
		Columns:    blameColumns,
		Rows:       [][]string{blameRow(lines[0])},
		Width:      80,
		Height:     5,
		HiddenCols: blameHiddenCols(blameGutterFoldMax),
	}
	g.AutoWidthsCaps(blameMetaMaxContent)
	if g.Widths[blameColAuthor] != 0 || g.Widths[blameColAge] != 0 || g.Widths[blameColCommit] != 0 {
		t.Fatalf("hidden meta should be width 0: %v", g.Widths)
	}
	if g.Widths[blameColLine] == 0 || g.Widths[blameColCode] < 40 {
		t.Fatalf("line+code should remain wide: %v", g.Widths)
	}
	view := g.View()
	if strings.Contains(view, "author") || strings.Contains(view, "commit") {
		t.Fatalf("folded view should omit meta headers: %q", view)
	}
	if !strings.Contains(view, "line") || !strings.Contains(view, "code") {
		t.Fatalf("line/code headers should remain: %q", view)
	}
}

func TestBlameFoldKeysOnCode(t *testing.T) {
	m := Model{
		main:     MainBlame,
		blameCol: blameColCode,
		blame:    []git.BlameLine{{Line: 1, Hash: "aaaaaaaa", Text: "x"}},
	}
	next, _ := m.handleBlameKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	mm := next.(Model)
	if mm.blameGutterFold != 1 || mm.blameCol != blameColCode {
		t.Fatalf("l on code should fold: fold=%d col=%d", mm.blameGutterFold, mm.blameCol)
	}
	next, _ = mm.handleBlameKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	mm = next.(Model)
	if mm.blameGutterFold != 0 {
		t.Fatalf("h on code should unfold first: fold=%d", mm.blameGutterFold)
	}
}

func TestBlameLineCursorMark(t *testing.T) {
	if got := blameLineCell(12, false); got != " 12" {
		t.Fatalf("inactive = %q", got)
	}
	if got := blameLineCell(12, true); got != blameCursorMark+"12" {
		t.Fatalf("active = %q", got)
	}
}

func TestPadCellRightAlign(t *testing.T) {
	got := padCellAlign("12", 6, true)
	if got != "   12 " {
		t.Fatalf("right padCell = %q", got)
	}
	if got := padTrimRight("7", 4); got != "   7" {
		t.Fatalf("padTrimRight = %q", got)
	}
}

func TestBlameMetaWidthsPreferCode(t *testing.T) {
	longAuthor := "Very Long Author Name That Would Dominate"
	lines := []git.BlameLine{
		{Line: 1234, ShortHash: "abcdef1", Author: longAuthor, When: time.Now(), Text: "fmt.Println(x)"},
	}
	g := Grid{
		Columns: blameColumns,
		Rows:    [][]string{blameRow(lines[0])},
		Width:   80,
		Height:  6,
	}
	g.AutoWidthsCaps(blameMetaMaxContent)
	if len(g.Widths) != blameColCount {
		t.Fatalf("widths = %v", g.Widths)
	}
	meta := 0
	for i := 0; i < blameColCode; i++ {
		meta += g.Widths[i]
	}
	meta += blameColCode // separators before code
	code := g.Widths[blameColCode]
	if code < meta {
		t.Fatalf("code width %d should dominate meta+seps %d; widths=%v", code, meta, g.Widths)
	}
	pad := 2 * cellPad
	if g.Widths[blameColAuthor] > blameMetaMaxContent[blameColAuthor]+pad {
		t.Fatalf("author not capped: %d", g.Widths[blameColAuthor])
	}
	age := blameCell(lines[0], blameColAge)
	if age != "now" && !strings.HasSuffix(age, "h") {
		t.Fatalf("compact age = %q", age)
	}
}
