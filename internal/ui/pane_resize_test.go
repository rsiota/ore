package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestResizePaneMovesSeamBothWays(t *testing.T) {
	m := Model{width: 120, height: 30}
	before := m.mainPaneWidth()
	if before != 60 {
		t.Fatalf("default half = %d, want 60", before)
	}

	m = m.resizePane("alt+h")
	if m.mainPaneWidth() != before-paneResizeStep {
		t.Fatalf("alt+h: width %d → %d, want %d", before, m.mainPaneWidth(), before-paneResizeStep)
	}

	m = m.resizePane("alt+l")
	if m.mainPaneWidth() != before {
		t.Fatalf("alt+l should restore half, got %d", m.mainPaneWidth())
	}
}

func TestResizePaneClamps(t *testing.T) {
	m := Model{width: 120, height: 30, mainPaneSplitW: minMainPaneWidth}
	m = m.resizePane("alt+h")
	if m.mainPaneWidth() != minMainPaneWidth {
		t.Fatalf("min clamp = %d, want %d", m.mainPaneWidth(), minMainPaneWidth)
	}

	m.mainPaneSplitW = 120 - minRightPaneWidth
	m = m.resizePane("alt+l")
	want := 120 - minRightPaneWidth
	if m.mainPaneWidth() != want {
		t.Fatalf("max clamp = %d, want %d", m.mainPaneWidth(), want)
	}
}

func TestResizePaneIgnoredWhenNarrow(t *testing.T) {
	m := Model{width: 79, height: 30}
	m = m.resizePane("alt+l")
	if m.mainPaneSplitW != 0 {
		t.Fatalf("narrow layout should ignore resize, split=%d", m.mainPaneSplitW)
	}
}

func TestResizePaneJKNoOp(t *testing.T) {
	m := Model{width: 120, height: 30}
	before := m.mainPaneWidth()
	m = m.resizePane("alt+j")
	m = m.resizePane("alt+k")
	if m.mainPaneWidth() != before || m.mainPaneSplitW != 0 {
		t.Fatal("alt+j/k should not move the horizontal seam")
	}
}

func TestResizePaneKeyDispatch(t *testing.T) {
	m := Model{width: 120, height: 30, focus: FocusMain}
	before := m.mainPaneWidth()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}, Alt: true}
	if msg.String() != "alt+l" {
		t.Fatalf("precondition: KeyMsg String=%q, want alt+l", msg.String())
	}
	out, _ := m.handleKey(msg)
	mm := out.(Model)
	if mm.mainPaneWidth() != before+paneResizeStep {
		t.Fatalf("handleKey(alt+l): %d → %d, want %d", before, mm.mainPaneWidth(), before+paneResizeStep)
	}
}

func TestResizePaneCtrlAltAlias(t *testing.T) {
	m := Model{width: 120, height: 30, focus: FocusDetail}
	before := m.mainPaneWidth()
	msg := tea.KeyMsg{Type: tea.KeyCtrlL, Alt: true}
	if msg.String() != "alt+ctrl+l" {
		t.Fatalf("precondition: KeyMsg String=%q, want alt+ctrl+l", msg.String())
	}
	out, _ := m.handleKey(msg)
	mm := out.(Model)
	if mm.mainPaneWidth() != before+paneResizeStep {
		t.Fatalf("handleKey(alt+ctrl+l): %d → %d, want %d", before, mm.mainPaneWidth(), before+paneResizeStep)
	}
}

func TestResizePaneFromExplorerFocus(t *testing.T) {
	m := Model{width: 120, height: 30, focus: FocusExplorer}
	m.explorer.Open()
	before := m.mainPaneWidth()
	out, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}, Alt: true})
	mm := out.(Model)
	if mm.mainPaneWidth() != before-paneResizeStep {
		t.Fatalf("explorer alt+h: %d → %d, want %d", before, mm.mainPaneWidth(), before-paneResizeStep)
	}
}
