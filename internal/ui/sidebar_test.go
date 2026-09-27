package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsiota/ore/internal/git"
	"github.com/rsiota/ore/internal/session"
)

func TestToggleSidebarOpensAndCloses(t *testing.T) {
	m := Model{width: 120, height: 30, focus: FocusMain}
	if cmd := m.toggleSidebar(); cmd != nil {
		t.Fatal("no repo: ensureTreeLoaded should be nil")
	}
	if !m.sidebarOpen || m.focus != FocusSidebar {
		t.Fatalf("open: sidebar=%v focus=%v", m.sidebarOpen, m.focus)
	}
	if !m.sidebarDocked() || m.sidebarWidth() != defaultSidebarWidth {
		t.Fatalf("docked width = %d", m.sidebarWidth())
	}
	_ = m.toggleSidebar()
	if m.sidebarOpen || m.focus != FocusMain {
		t.Fatalf("close: sidebar=%v focus=%v", m.sidebarOpen, m.focus)
	}
}

func TestSidebarLayoutLeavesRoomForMainAndDetail(t *testing.T) {
	m := Model{width: 120, height: 30, sidebarOpen: true, focus: FocusSidebar}
	sw := m.sidebarWidth()
	mw := m.mainPaneWidth()
	dw := m.width - sw - mw
	if sw != defaultSidebarWidth {
		t.Fatalf("sidebar = %d", sw)
	}
	if mw < minMainWithSidebar || dw < minRightWithSidebar {
		t.Fatalf("main=%d detail=%d", mw, dw)
	}
	if sw+mw+dw != 120 {
		t.Fatalf("sum = %d", sw+mw+dw)
	}

	closed := Model{width: 120, height: 30}
	if closed.mainPaneWidth() != 60 {
		t.Fatalf("closed default half = %d", closed.mainPaneWidth())
	}
}

func TestDetailInnerWidthAccountsForSidebar(t *testing.T) {
	m := Model{width: 120, height: 30, sidebarOpen: true}
	dw := m.width - m.sidebarWidth() - m.mainPaneWidth()
	want := max(1, dw-borderOverhead)
	if got := m.detailInnerWidth(); got != want {
		t.Fatalf("detailInnerWidth=%d want %d", got, want)
	}
	closed := Model{width: 120, height: 30}
	wantClosed := max(1, closed.width-closed.mainPaneWidth()-borderOverhead)
	if got := closed.detailInnerWidth(); got != wantClosed {
		t.Fatalf("closed detailInnerWidth=%d want %d", got, wantClosed)
	}
}

func TestResizeSidebarWhenFocused(t *testing.T) {
	m := Model{width: 120, height: 30, sidebarOpen: true, focus: FocusSidebar}
	before := m.sidebarWidth()
	m = m.resizePane("alt+l")
	if m.sidebarWidth() != before+paneResizeStep {
		t.Fatalf("alt+l sidebar %d → %d", before, m.sidebarWidth())
	}
	mainBefore := m.mainPaneWidth()
	m.focus = FocusMain
	m = m.resizePane("alt+l")
	if m.sidebarWidth() != before+paneResizeStep {
		t.Fatal("main resize should not change sidebar")
	}
	if m.mainPaneWidth() != mainBefore+paneResizeStep {
		t.Fatalf("main seam %d → %d", mainBefore, m.mainPaneWidth())
	}
}

func TestAltBTogglesFromHandleKey(t *testing.T) {
	m := Model{width: 120, height: 30, focus: FocusMain}
	out, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}, Alt: true})
	mm := out.(Model)
	if !mm.sidebarOpen || mm.focus != FocusSidebar {
		t.Fatalf("alt+b open: sidebar=%v focus=%v", mm.sidebarOpen, mm.focus)
	}
}

func TestSidebarEnterOpensHistory(t *testing.T) {
	m := Model{
		width:  120,
		height: 30,
		focus:  FocusSidebar,
		tree:   sampleTree(),
	}
	m.tree.selectPath("src/a.go")
	next, cmd := m.sidebarActivate(false)
	mm := next.(Model)
	if mm.focus != FocusMain {
		t.Fatalf("focus = %v", mm.focus)
	}
	if mm.historyPath != "src/a.go" {
		t.Fatalf("historyPath = %q", mm.historyPath)
	}
	if cmd == nil {
		t.Fatal("expected history load cmd")
	}
}

func TestSidebarBlameUsesViewRev(t *testing.T) {
	m := Model{
		width:   120,
		height:  30,
		focus:   FocusSidebar,
		viewRev: "feature",
		tree:    sampleTree(),
	}
	m.tree.selectPath("readme.txt")
	next, cmd := m.sidebarActivate(true)
	mm := next.(Model)
	if mm.blamePath != "readme.txt" || mm.blameRev != "feature" {
		t.Fatalf("blame %s @ %s", mm.blamePath, mm.blameRev)
	}
	if mm.blameFrom != MainFiles {
		t.Fatalf("blameFrom = %v", mm.blameFrom)
	}
	if cmd == nil {
		t.Fatal("expected blame load cmd")
	}
}

func TestCycleFocusIncludesSidebar(t *testing.T) {
	m := Model{width: 120, height: 30, sidebarOpen: true, focus: FocusSidebar, detailCache: &detailRenderCache{}}
	next, _ := m.cycleFocus()
	mm := next.(Model)
	if mm.focus != FocusMain {
		t.Fatalf("sidebar → main: %v", mm.focus)
	}
	mm.focus = FocusDetail
	next, _ = mm.cycleFocus()
	mm = next.(Model)
	if mm.focus != FocusSidebar {
		t.Fatalf("detail → sidebar: %v", mm.focus)
	}
}

func TestSnapshotSessionSidebar(t *testing.T) {
	m := Model{
		main:          MainCommits,
		sidebarOpen:   true,
		sidebarSplitW: 28,
		tree:          sampleTree(),
	}
	m.tree.selectPath("src/a.go")
	st := m.snapshotSession()
	if !st.Sidebar || st.SidebarWidth != 28 || st.SidebarPath != "src/a.go" {
		t.Fatalf("snapshot = %#v", st)
	}

	var applied Model
	applied.applySessionChrome(session.State{
		Sidebar:      true,
		SidebarWidth: 30,
		SidebarPath:  "readme.txt",
	})
	if !applied.sidebarOpen || applied.sidebarSplitW != 30 || applied.tree.preferPath != "readme.txt" {
		t.Fatalf("applied sidebar=%v w=%d prefer=%q", applied.sidebarOpen, applied.sidebarSplitW, applied.tree.preferPath)
	}
}

func TestSidebarClickSelectsRow(t *testing.T) {
	m := Model{
		width:       120,
		height:      30,
		sidebarOpen: true,
		focus:       FocusMain,
		tree:        sampleTree(),
		detailCache: &detailRenderCache{},
	}
	// First data row is below border + title + rule.
	y := 1 + 2
	next, _ := m.handleMouse(clickAt(2, y))
	mm := next.(Model)
	if mm.focus != FocusSidebar {
		t.Fatalf("focus = %v, want FocusSidebar", mm.focus)
	}
	row, ok := mm.tree.selected()
	if !ok || row.path != "src" {
		t.Fatalf("selected = %#v ok=%v", row, ok)
	}
}

func TestMainGridClickAccountsForSidebar(t *testing.T) {
	m := Model{
		width:       120,
		height:      30,
		sidebarOpen: true,
		focus:       FocusSidebar,
		main:        MainCommits,
		commits: []git.Commit{
			{Hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortHash: "aaaaaaa", Subject: "one"},
			{Hash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ShortHash: "bbbbbbb", Subject: "two"},
		},
		detailCache: &detailRenderCache{},
	}
	innerW := max(1, m.mainPaneWidth()-borderOverhead)
	g := m.commitGrid(innerW, m.paneContentHeight())
	x := m.sidebarWidth() + 1
	for i := 0; i < commitColAuthor; i++ {
		x += g.widthAt(i) + 1
	}
	y := 1 + 2 + 1
	next, _ := m.handleMouse(clickAt(x, y))
	mm := next.(Model)
	if mm.focus != FocusMain {
		t.Fatalf("focus = %v, want FocusMain", mm.focus)
	}
	if mm.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", mm.cursor)
	}
}

func TestRenderBodyIncludesSidebar(t *testing.T) {
	m := Model{
		repo:        &git.Repo{Path: "/tmp/demo"},
		width:       120,
		height:      30,
		sidebarOpen: true,
		focus:       FocusSidebar,
		tree:        sampleTree(),
		detailCache: &detailRenderCache{},
	}
	body := m.renderBody()
	plain := stripAnsi(body)
	if !strings.Contains(plain, "tree") || !strings.Contains(plain, "src") {
		t.Fatalf("body missing tree: %q", plain)
	}
}
