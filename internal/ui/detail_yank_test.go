package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/rsiota/ore/internal/git"
)

func TestYankMotions(t *testing.T) {
	lines := []string{"alpha beta", "gamma", ""}
	r, c := moveWordForward(lines, 0, 0)
	if r != 0 || c != 6 {
		t.Fatalf("w from start = %d,%d want 0,6", r, c)
	}
	r, c = moveWordEnd(lines, 0, 0)
	if r != 0 || c != 4 {
		t.Fatalf("e = %d,%d want 0,4", r, c)
	}
	r, c = moveWordBackward(lines, 0, 6)
	if r != 0 || c != 0 {
		t.Fatalf("b = %d,%d want 0,0", r, c)
	}
	r, c = moveLineEnd(lines, 0, 0)
	if c != 9 {
		t.Fatalf("$ col = %d", c)
	}
	r, c, ok := findOnLine(lines, 0, 0, 'b', false, false)
	if !ok || c != 6 {
		t.Fatalf("f b = %d,%d ok=%v", r, c, ok)
	}
}

func TestYankSelectionCharAndLine(t *testing.T) {
	lines := []string{"abcdef", "ghij"}
	y := detailYank{row: 0, col: 2, visual: yankVisualChar, anchorRow: 0, anchorCol: 0}
	if got := yankSelection(lines, y); got != "abc" {
		t.Fatalf("char sel = %q", got)
	}
	y = detailYank{row: 1, col: 0, visual: yankVisualLine, anchorRow: 0, anchorCol: 0}
	if got := yankSelection(lines, y); got != "abcdef\nghij" {
		t.Fatalf("line sel = %q", got)
	}
	if got := wordAtCursor(lines, 0, 1); got != "abcdef" {
		t.Fatalf("word = %q", got)
	}
}

func TestYankSearch(t *testing.T) {
	lines := []string{"one", "two foo", "three"}
	r, c, ok := searchForward(lines, 0, 0, "foo")
	if !ok || r != 1 || c != 4 {
		t.Fatalf("search foo = %d,%d ok=%v", r, c, ok)
	}
	r, c, ok = searchBackward(lines, 2, 0, "one")
	if !ok || r != 0 || c != 0 {
		t.Fatalf("search back one = %d,%d ok=%v", r, c, ok)
	}
}

func TestEnterLeaveDetailYank(t *testing.T) {
	m := Model{
		focus:       FocusMain,
		width:       80,
		height:      24,
		detailCache: &detailRenderCache{},
		detail: &git.CommitDetail{
			Commit: git.Commit{Hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortHash: "aaaaaaa", Subject: "subj"},
			Diff:   "+hello world\n",
		},
		diffMode: DiffZen,
	}
	m.enterDetailYank()
	if m.focus != FocusDetail {
		t.Fatal("expected FocusDetail")
	}
	if m.yank.modeLabel() != "VIEW" {
		t.Fatalf("mode = %s", m.yank.modeLabel())
	}
	mm, cmd := m.handleDetailKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = mm.(Model)
	if m.yank.pending != yankPendingY {
		t.Fatal("expected y-pending")
	}
	mm, cmd = m.handleDetailKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = mm.(Model)
	if cmd == nil {
		t.Fatal("yy should return clipboard cmd")
	}
	if m.yank.register == "" {
		t.Fatal("expected register set")
	}
	// Drive the clipboard cmd (may skip write errors in headless envs).
	if msg := cmd(); msg != nil {
		if yc, ok := msg.(yankCopiedMsg); ok && yc.n == 0 {
			t.Fatal("expected non-zero yank length")
		}
	}
	mm, _ = m.handleDetailKeys(tea.KeyMsg{Type: tea.KeyEsc})
	m = mm.(Model)
	if m.focus != FocusMain {
		t.Fatalf("esc should leave yank, focus=%v", m.focus)
	}
}

func TestYankFlashToggles(t *testing.T) {
	var y detailYank
	y.startFlash(lineFlashRegion(2, 0))
	if !y.flashActive || y.flashOn {
		t.Fatalf("flash should start off: active=%v on=%v", y.flashActive, y.flashOn)
	}
	// off → on
	if !y.AdvanceFlash() || !y.flashOn || !y.flashActive {
		t.Fatal("first tick should turn flash on")
	}
	// on → off / done
	if y.AdvanceFlash() || y.flashActive {
		t.Fatal("second tick should clear the flash")
	}
}

func TestYankFlashKeepsLineRange(t *testing.T) {
	var y detailYank
	y.startFlash(detailYank{
		visual: yankVisualLine, anchorRow: 1, row: 4, col: 0, anchorCol: 0,
	})
	f := y.flashSelection()
	if f.anchorRow != 1 || f.row != 4 {
		t.Fatalf("line flash range = %d..%d, want 1..4", f.anchorRow, f.row)
	}
	if !yankRowSelected(f, 2) || !yankRowSelected(f, 4) || yankRowSelected(f, 0) {
		t.Fatalf("expected mid/end rows selected: %#v", f)
	}
}

func TestYankEscPeelsVisualFirst(t *testing.T) {
	m := Model{focus: FocusDetail, width: 80, height: 24, detailCache: &detailRenderCache{}}
	m.yank.visual = yankVisualLine
	m.yank.anchorRow = 0
	mm, _ := m.handleDetailKeys(tea.KeyMsg{Type: tea.KeyEsc})
	m = mm.(Model)
	if m.yank.visual != yankVisualNone || m.focus != FocusDetail {
		t.Fatalf("esc should clear visual first: visual=%v focus=%v", m.yank.visual, m.focus)
	}
}

func TestPaintYankKeepsHighlightOutsideVisual(t *testing.T) {
	applyTheme("light")
	styled := styleAdd.Render("hello") + styleDel.Render(" world")
	plain := ansi.Strip(styled)
	y := detailYank{row: 0, col: 1, visual: yankVisualNone}
	got := paintYankOnStyled(styled, plain, 20, 0, y)
	// Diff colours should still be present (SGR), not flattened to plain cell fg only.
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("expected ANSI highlight preserved: %q", got)
	}
	y.visual = yankVisualLine
	y.anchorRow = 0
	washed := paintYankOnStyled(styled, plain, 20, 0, y)
	if ansi.Strip(washed) == "" {
		t.Fatal("visual line should still show text")
	}
}

func TestJumpDetailHunk(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/f b/f",
		"--- a/f",
		"+++ b/f",
		"@@ -1,2 +1,2 @@ top",
		" a",
		"-b",
		"+B",
		"@@ -20,2 +20,2 @@ bottom",
		" c",
		"-d",
		"+D",
	}, "\n")
	m := Model{
		focus:      FocusDetail,
		width:      120,
		height:     40,
		diffMode:   DiffZen,
		zenContext: 3,
		detail: &git.CommitDetail{
			Commit: git.Commit{Hash: "h", Subject: "s", Files: 1},
			Diff:   diff,
		},
		detailCache: &detailRenderCache{},
	}
	m.enterDetailYank()
	mm, _ := m.handleDetailKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	m = mm.(Model)
	hunks := detailHunkRows(m.detailYankLines())
	if len(hunks) < 2 {
		t.Fatalf("expected >=2 hunks, got %v from %#v", hunks, m.detailYankLines())
	}
	if m.yank.row != hunks[0] {
		t.Fatalf("] from top should land on first hunk: row=%d want=%d", m.yank.row, hunks[0])
	}
	mm, _ = m.handleDetailKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	m = mm.(Model)
	if m.yank.row != hunks[1] {
		t.Fatalf("] should land on second hunk: row=%d want=%d", m.yank.row, hunks[1])
	}
	mm, _ = m.handleDetailKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	m = mm.(Model)
	if m.yank.row != hunks[0] {
		t.Fatalf("[ should return to first hunk: row=%d want=%d", m.yank.row, hunks[0])
	}

	m.diffMode = DiffUnified
	m.invalidateDetailCache()
	m.enterDetailYank()
	uh := detailHunkRows(m.detailYankLines())
	if len(uh) < 2 {
		t.Fatalf("unified expected >=2 hunks, got %v", uh)
	}
	mm, _ = m.handleDetailKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	m = mm.(Model)
	if m.yank.row != uh[0] {
		t.Fatalf("unified ] first hunk: row=%d want=%d", m.yank.row, uh[0])
	}
}
