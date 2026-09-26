package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsiota/ore/internal/git"
)

func clickAt(x, y int) tea.MouseMsg {
	return tea.MouseMsg{Type: tea.MouseLeft, Action: tea.MouseActionPress, X: x, Y: y}
}

func TestMainGridClickSelectsCell(t *testing.T) {
	m := Model{
		width:  120,
		height: 30,
		main:   MainCommits,
		focus:  FocusDetail,
		commits: []git.Commit{
			{Hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortHash: "aaaaaaa", Subject: "one"},
			{Hash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ShortHash: "bbbbbbb", Subject: "two"},
			{Hash: "cccccccccccccccccccccccccccccccccccccccc", ShortHash: "ccccccc", Subject: "three"},
		},
		cursor:    0,
		commitCol: 0,
	}
	innerW := max(1, m.mainPaneWidth()-borderOverhead)
	innerH := m.paneContentHeight()
	g := m.commitGrid(innerW, innerH)
	col := commitColAuthor
	x := 1 // skip left border
	for i := 0; i < col; i++ {
		x += g.widthAt(i) + 1
	}
	y := 1 + 2 + 1 // top border + header + rule + second data row

	next, _ := m.handleMouse(clickAt(x, y))
	mm := next.(Model)
	if mm.focus != FocusMain {
		t.Fatalf("focus = %v, want FocusMain", mm.focus)
	}
	if mm.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", mm.cursor)
	}
	if mm.commitCol != col {
		t.Fatalf("col = %d, want %d", mm.commitCol, col)
	}
}

func TestMainGridClickHeaderSelectsColumn(t *testing.T) {
	m := Model{
		width:     120,
		height:    30,
		main:      MainCommits,
		commits:   []git.Commit{{Hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortHash: "aaaaaaa"}},
		cursor:    0,
		commitCol: commitColHash,
	}
	innerW := max(1, m.mainPaneWidth()-borderOverhead)
	g := m.commitGrid(innerW, m.paneContentHeight())
	x := 1 + g.widthAt(0) + 1 // into hash/date… second visible col
	y := 1                    // header row inside border
	next, _ := m.handleMouse(clickAt(x, y))
	mm := next.(Model)
	if mm.cursor != 0 {
		t.Fatalf("header click should keep row, cursor=%d", mm.cursor)
	}
	if mm.commitCol == commitColGraph {
		t.Fatal("header click should move off the first column")
	}
}

func testDetailModel() Model {
	return Model{
		width:       120,
		height:      30,
		main:        MainCommits,
		focus:       FocusMain,
		detailCache: &detailRenderCache{},
		detail: &git.CommitDetail{
			Commit: git.Commit{Hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortHash: "aaaaaaa", Subject: "hello world"},
			Diff: strings.Join([]string{
				"diff --git a/foo.go b/foo.go",
				"--- a/foo.go",
				"+++ b/foo.go",
				"@@ -1,1 +1,2 @@",
				" hello",
				"+world",
			}, "\n"),
		},
		diffMode:   DiffZen,
		zenContext: 3,
		commits:    []git.Commit{{Hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortHash: "aaaaaaa", Subject: "hello world"}},
		cursor:     0,
		commitCol:  0,
	}
}

func TestDetailPaneClickSelectsCell(t *testing.T) {
	m := testDetailModel()
	lines := m.detailYankLines()
	if len(lines) < 2 {
		t.Fatalf("need at least 2 detail lines, got %d", len(lines))
	}
	row := 1
	innerX := 3
	wantCol := runeIndexAtDisplayCol(lines[row], innerX)
	x := m.mainPaneWidth() + 1 + innerX
	y := 1 + row - m.detailOffset

	next, _ := m.handleMouse(clickAt(x, y))
	mm := next.(Model)
	if mm.focus != FocusDetail {
		t.Fatalf("focus = %v, want FocusDetail", mm.focus)
	}
	if mm.yank.row != row {
		t.Fatalf("yank.row = %d, want %d", mm.yank.row, row)
	}
	if mm.yank.col != wantCol {
		t.Fatalf("yank.col = %d, want %d (line %q innerX=%d)", mm.yank.col, wantCol, lines[row], innerX)
	}
	if mm.cursor != 0 || mm.commitCol != 0 {
		t.Fatalf("detail click moved the grid: cursor=%d col=%d", mm.cursor, mm.commitCol)
	}
}

func TestDetailPaneClickMovesExistingYank(t *testing.T) {
	m := testDetailModel()
	m.enterDetailYank()
	if m.yank.row != 0 {
		t.Fatalf("setup: yank.row = %d, want 0", m.yank.row)
	}
	row := 1
	innerX := 2
	x := m.mainPaneWidth() + 1 + innerX
	y := 1 + row
	next, _ := m.handleMouse(clickAt(x, y))
	mm := next.(Model)
	if mm.focus != FocusDetail {
		t.Fatalf("focus = %v, want FocusDetail", mm.focus)
	}
	if mm.yank.row != row {
		t.Fatalf("yank.row = %d, want %d", mm.yank.row, row)
	}
}

func TestDetailPaneClickIgnoredWhenNarrow(t *testing.T) {
	m := testDetailModel()
	m.width = 79
	x := m.mainPaneWidth() + 2
	next, _ := m.handleMouse(clickAt(x, 4))
	mm := next.(Model)
	if mm.focus == FocusDetail {
		t.Fatal("narrow layout has no right pane; click should not enter detail yank")
	}
}

func TestExplorerPaneClickSelectsRow(t *testing.T) {
	m := testDetailModel()
	m.explorer.LoadCommit(git.CommitRelations{
		Hash:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Subject: "root",
		Author:  "a",
		Email:   "a@b",
		Files:   []git.FileChange{{Path: "a.go"}, {Path: "b.go"}},
	})
	m.layoutExplorer()
	vis := m.explorer.visibleNodes()
	fileIdx := -1
	for i, n := range vis {
		if n.kind == relFile && n.path == "b.go" {
			fileIdx = i
			break
		}
	}
	if fileIdx < 0 {
		t.Fatal("expected b.go row")
	}
	if m.explorer.cursor == fileIdx {
		t.Fatal("setup: cursor already on b.go")
	}
	x := m.mainPaneWidth() + 2
	y := 1 + fileIdx - m.explorer.offset
	next, _ := m.handleMouse(clickAt(x, y))
	mm := next.(Model)
	if mm.focus != FocusExplorer {
		t.Fatalf("focus = %v, want FocusExplorer", mm.focus)
	}
	if mm.explorer.cursor != fileIdx {
		t.Fatalf("explorer.cursor = %d, want %d", mm.explorer.cursor, fileIdx)
	}
	row, ok := mm.explorer.Selected()
	if !ok || row.path != "b.go" {
		t.Fatalf("selected = %#v ok=%v, want b.go", row, ok)
	}
}

func TestMainGridClickIgnoresMotion(t *testing.T) {
	m := Model{
		width:     120,
		height:    30,
		commits:   []git.Commit{{Hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortHash: "aaaaaaa"}},
		cursor:    0,
		commitCol: 0,
	}
	msg := tea.MouseMsg{Type: tea.MouseLeft, Action: tea.MouseActionMotion, X: 4, Y: 4}
	next, _ := m.handleMouse(msg)
	mm := next.(Model)
	if mm.cursor != 0 || mm.commitCol != 0 {
		t.Fatal("drag motion should not select a cell")
	}
}

func TestMainGridClickFilesRow(t *testing.T) {
	m := Model{
		width:  120,
		height: 30,
		main:   MainFiles,
		files: []git.FileChange{
			{Path: "a.go"},
			{Path: "b.go"},
		},
		fileCursor: 0,
		fileCol:    0,
	}
	y := 1 + 2 + 1
	next, _ := m.handleMouse(clickAt(2, y))
	mm := next.(Model)
	if mm.fileCursor != 1 {
		t.Fatalf("fileCursor = %d, want 1", mm.fileCursor)
	}
}
