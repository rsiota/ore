package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsiota/ore/internal/git"
)

func TestEnterBlameYankFromTab(t *testing.T) {
	m := Model{
		main:   MainBlame,
		focus:  FocusMain,
		width:  120,
		height: 40,
		blame: []git.BlameLine{
			{Line: 1, Text: "alpha beta", Hash: "aaa111aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortHash: "aaa111"},
			{Line: 2, Text: "gamma", Hash: "bbb222bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ShortHash: "bbb222"},
		},
		blameCursor: 1,
		blameCol:    blameColCode,
		diffMode:    DiffZen,
	}
	mm, _ := m.cycleFocus()
	m = mm.(Model)
	if m.focus != FocusBlameYank {
		t.Fatalf("focus=%v want FocusBlameYank", m.focus)
	}
	if m.yank.row != 1 {
		t.Fatalf("yank.row=%d want 1", m.yank.row)
	}
	lines := m.blameYankLines()
	if len(lines) != 2 || lines[0] != "alpha beta" {
		t.Fatalf("lines=%#v", lines)
	}

	mm, _ = m.handleBlameYankKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = mm.(Model)
	if m.yank.row != 0 || m.blameCursor != 0 {
		t.Fatalf("k should move yank+cursor: yank=%d cursor=%d", m.yank.row, m.blameCursor)
	}

	mm, _ = m.handleBlameYankKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	m = mm.(Model)
	if m.yank.col != 6 {
		t.Fatalf("w col=%d want 6", m.yank.col)
	}

	mm, _ = m.handleBlameYankKeys(tea.KeyMsg{Type: tea.KeyTab})
	m = mm.(Model)
	if m.focus != FocusDetail {
		t.Fatalf("tab from blame yank → detail, got %v", m.focus)
	}
}

func TestBlameYankEscLeaves(t *testing.T) {
	m := Model{
		main: MainBlame,
		blame: []git.BlameLine{
			{Line: 1, Text: "hello", Hash: "aaa111aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		},
		width:  80,
		height: 24,
	}
	m.enterBlameYank()
	mm, _ := m.handleBlameYankKeys(tea.KeyMsg{Type: tea.KeyEsc})
	m = mm.(Model)
	if m.focus != FocusMain {
		t.Fatalf("esc should leave to main, got %v", m.focus)
	}
}

func TestCycleFocusBlameThenDetailThenMain(t *testing.T) {
	m := Model{
		main: MainBlame,
		blame: []git.BlameLine{
			{Line: 1, Text: "x", Hash: "aaa111aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		},
		width:  80,
		height: 24,
		diffMode: DiffZen,
	}
	mm, _ := m.cycleFocus()
	m = mm.(Model)
	if m.focus != FocusBlameYank {
		t.Fatal("1st tab → blame yank")
	}
	mm, _ = m.cycleFocus()
	m = mm.(Model)
	if m.focus != FocusDetail {
		t.Fatal("2nd tab → detail")
	}
	mm, _ = m.cycleFocus()
	m = mm.(Model)
	if m.focus != FocusMain {
		t.Fatal("3rd tab → main")
	}
}
