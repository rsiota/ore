package ui

import (
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

func TestMainGridClickIgnoresDetailPane(t *testing.T) {
	m := Model{
		width:     120,
		height:    30,
		main:      MainCommits,
		focus:     FocusMain,
		commits:   []git.Commit{{Hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortHash: "aaaaaaa"}},
		cursor:    0,
		commitCol: 0,
	}
	x := m.mainPaneWidth() + 2
	next, _ := m.handleMouse(clickAt(x, 5))
	mm := next.(Model)
	if mm.cursor != 0 || mm.commitCol != 0 || mm.focus != FocusMain {
		t.Fatalf("detail click moved the grid: cursor=%d col=%d focus=%v", mm.cursor, mm.commitCol, mm.focus)
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
