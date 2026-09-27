package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsiota/ore/internal/git"
)

func TestMoveFocusThreeColumns(t *testing.T) {
	m := Model{
		width:       120,
		height:      30,
		sidebarOpen: true,
		focus:       FocusSidebar,
		detailCache: &detailRenderCache{},
	}
	next, _ := m.moveFocus("ctrl+l")
	m = next.(Model)
	if m.focus != FocusMain {
		t.Fatalf("tree→l: %v, want FocusMain", m.focus)
	}
	next, _ = m.moveFocus("ctrl+l")
	m = next.(Model)
	if m.focus != FocusDetail {
		t.Fatalf("main→l: %v, want FocusDetail", m.focus)
	}
	next, _ = m.moveFocus("ctrl+l")
	m = next.(Model)
	if m.focus != FocusDetail {
		t.Fatalf("detail→l should stay: %v", m.focus)
	}
	next, _ = m.moveFocus("ctrl+h")
	m = next.(Model)
	if m.focus != FocusMain {
		t.Fatalf("detail→h: %v, want FocusMain", m.focus)
	}
	next, _ = m.moveFocus("ctrl+h")
	m = next.(Model)
	if m.focus != FocusSidebar {
		t.Fatalf("main→h: %v, want FocusSidebar", m.focus)
	}
	next, _ = m.moveFocus("ctrl+h")
	m = next.(Model)
	if m.focus != FocusSidebar {
		t.Fatalf("tree→h should stay: %v", m.focus)
	}
}

func TestMoveFocusNoSidebarStaysOnMain(t *testing.T) {
	m := Model{width: 120, height: 30, focus: FocusMain, detailCache: &detailRenderCache{}}
	next, _ := m.moveFocus("ctrl+h")
	if next.(Model).focus != FocusMain {
		t.Fatalf("main→h with no tree: %v", next.(Model).focus)
	}
	next, _ = m.moveFocus("ctrl+l")
	if next.(Model).focus != FocusDetail {
		t.Fatalf("main→l: %v, want FocusDetail", next.(Model).focus)
	}
}

func TestMoveFocusSkipsHunksAndYank(t *testing.T) {
	m := Model{
		width:       120,
		height:      30,
		sidebarOpen: true,
		hunkMode:    true,
		main:        MainBlame,
		focus:       FocusMain,
		blame: []git.BlameLine{
			{Line: 1, Text: "x", Hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		},
		detailCache: &detailRenderCache{},
	}
	next, _ := m.moveFocus("ctrl+l")
	if next.(Model).focus != FocusDetail {
		t.Fatalf("centre→l should skip hunks/yank: %v", next.(Model).focus)
	}
	next, _ = next.(Model).moveFocus("ctrl+h")
	if next.(Model).focus != FocusMain {
		t.Fatalf("detail→h should land on main, not hunks: %v", next.(Model).focus)
	}
}

func TestCycleFocusIncludesTreeAndReverses(t *testing.T) {
	m := Model{
		width:       120,
		height:      30,
		sidebarOpen: true,
		focus:       FocusSidebar,
		detailCache: &detailRenderCache{},
	}
	next, _ := m.cycleFocus()
	m = next.(Model)
	if m.focus != FocusMain {
		t.Fatalf("tab tree→main: %v", m.focus)
	}
	next, _ = m.cycleFocus()
	m = next.(Model)
	if m.focus != FocusDetail {
		t.Fatalf("tab main→detail: %v", m.focus)
	}
	next, _ = m.cycleFocus()
	m = next.(Model)
	if m.focus != FocusSidebar {
		t.Fatalf("tab detail→tree: %v", m.focus)
	}
	next, _ = m.cycleFocusBack()
	m = next.(Model)
	if m.focus != FocusDetail {
		t.Fatalf("shift-tab tree→detail: %v", m.focus)
	}
}

func TestHandleKeyCtrlHLAndShiftTab(t *testing.T) {
	m := Model{
		width:       120,
		height:      30,
		sidebarOpen: true,
		focus:       FocusMain,
		detailCache: &detailRenderCache{},
		commits: []git.Commit{
			{Hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortHash: "aaaaaaa"},
		},
	}
	out, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlL})
	mm := out.(Model)
	if mm.focus != FocusDetail {
		t.Fatalf("handleKey ctrl+l: %v", mm.focus)
	}
	out, _ = mm.handleKey(tea.KeyMsg{Type: tea.KeyCtrlH})
	mm = out.(Model)
	if mm.focus != FocusMain {
		t.Fatalf("handleKey ctrl+h: %v", mm.focus)
	}
	out, _ = mm.handleKey(tea.KeyMsg{Type: tea.KeyShiftTab})
	mm = out.(Model)
	if mm.focus != FocusSidebar {
		t.Fatalf("handleKey shift+tab: %v", mm.focus)
	}
}

func TestMoveFocusExplorerIsRightColumn(t *testing.T) {
	m := Model{
		width:       120,
		height:      30,
		sidebarOpen: true,
		focus:       FocusMain,
		detailCache: &detailRenderCache{},
	}
	m.explorer.Open()
	next, _ := m.moveFocus("ctrl+l")
	m = next.(Model)
	if m.focus != FocusExplorer {
		t.Fatalf("main→l with explorer: %v", m.focus)
	}
	next, _ = m.moveFocus("ctrl+h")
	if next.(Model).focus != FocusMain {
		t.Fatalf("explorer→h: %v", next.(Model).focus)
	}
}

func TestCycleFocusExplorerWithTree(t *testing.T) {
	m := Model{
		width:       120,
		height:      30,
		sidebarOpen: true,
		focus:       FocusMain,
		detailCache: &detailRenderCache{},
		repo:        &git.Repo{Path: "/tmp/demo"},
	}
	m.explorer.Open()
	next, _ := m.cycleFocus()
	if next.(Model).focus != FocusExplorer {
		t.Fatalf("tab main→explorer: %v", next.(Model).focus)
	}
	next, _ = next.(Model).cycleFocus()
	if next.(Model).focus != FocusSidebar {
		t.Fatalf("tab explorer→tree: %v", next.(Model).focus)
	}
}
