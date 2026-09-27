package ui

import tea "github.com/charmbracelet/bubbletea"

// handleMouse routes mouse events. Overlays own the screen; otherwise a
// left-click selects a cell in the main grid or a line in the right pane.
// A press on the main|detail seam starts a creel-style resize drag.
func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.splitDragging {
		return m.handleSplitDrag(msg)
	}
	if m.help.Visible() || m.palette.IsVisible() || m.branches.IsVisible() || m.bookmarks.IsVisible() {
		return m, nil
	}
	if m.exTyping || m.filterTyping || m.tree.filterTyping {
		return m, nil
	}
	if isMouseLeftClick(msg) && m.onSidebarSplit(msg.X, msg.Y) {
		return m.beginSidebarSplitDrag(msg.X)
	}
	if isMouseLeftClick(msg) && m.onMainSplit(msg.X, msg.Y) {
		return m.beginSplitDrag(msg.X)
	}
	if !isMouseLeftClick(msg) {
		return m, nil
	}
	if next, ok := m.handleSidebarClick(msg.X, msg.Y); ok {
		return next, nil
	}
	if next, cmd, ok := m.handleRightPaneClick(msg.X, msg.Y); ok {
		return next, cmd
	}
	return m.handleMainGridClick(msg.X, msg.Y)
}

func (m Model) onSidebarSplit(x, y int) bool {
	if !m.sidebarDocked() || m.height < 2 {
		return false
	}
	if y < 0 || y >= m.panesHeight() {
		return false
	}
	seam := m.sidebarWidth()
	return x == seam-1 || x == seam
}

func (m Model) onMainSplit(x, y int) bool {
	if m.width < 80 || m.height < 2 {
		return false
	}
	if y < 0 || y >= m.panesHeight() {
		return false
	}
	seam := m.sidebarWidth() + m.mainPaneWidth()
	return x == seam-1 || x == seam
}

func (m Model) beginSidebarSplitDrag(x int) (tea.Model, tea.Cmd) {
	cur := m.sidebarWidth()
	m.splitDragging = true
	m.splitDragSidebar = true
	m.splitDragOff = x - cur
	return m.applySplitDragX(x)
}

func (m Model) beginSplitDrag(x int) (tea.Model, tea.Cmd) {
	cur := m.mainPaneWidth()
	m.splitDragging = true
	m.splitDragSidebar = false
	m.splitDragOff = x - cur
	return m.applySplitDragX(x)
}

func (m Model) handleSplitDrag(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Action == tea.MouseActionMotion ||
		(msg.Type == tea.MouseLeft && msg.Action != tea.MouseActionRelease) {
		return m.applySplitDragX(msg.X)
	}
	if msg.Type == tea.MouseRelease || msg.Action == tea.MouseActionRelease {
		m.splitDragging = false
		m.splitDragSidebar = false
		return m, nil
	}
	return m, nil
}

func (m Model) applySplitDragX(x int) (tea.Model, tea.Cmd) {
	if m.splitDragSidebar {
		before := m.sidebarWidth()
		m.sidebarSplitW = m.clampSidebarWidth(x - m.splitDragOff)
		if m.sidebarWidth() != before {
			m.invalidateDetailCache()
			m.layoutExplorer()
		}
		return m, nil
	}
	before := m.mainPaneWidth()
	m.mainPaneSplitW = m.clampMainPaneWidth(x - m.splitDragOff)
	if m.mainPaneWidth() != before {
		m.invalidateDetailCache()
		m.layoutExplorer()
	}
	return m, nil
}

func isMouseLeftClick(msg tea.MouseMsg) bool {
	if msg.Type != tea.MouseLeft {
		return false
	}
	switch msg.Action {
	case tea.MouseActionMotion, tea.MouseActionRelease:
		return false
	default:
		return true
	}
}

func (m Model) handleMainGridClick(x, y int) (tea.Model, tea.Cmd) {
	px, py, pw, ph, ok := m.leftPaneGeom()
	if !ok {
		return m, nil
	}
	if x < px || x >= px+pw || y < py || y >= py+ph {
		return m, nil
	}

	if m.focus == FocusDetail {
		m.leaveDetailYank()
	}
	if m.focus == FocusBlameYank {
		m.leaveBlameYank()
	}
	m.focus = FocusMain

	innerX := x - px - 1
	innerY := y - py - 1
	innerW := max(1, pw-borderOverhead)
	innerH := max(1, ph-borderOverhead)
	if innerX < 0 || innerY < 0 || innerX >= innerW || innerY >= innerH {
		return m, nil
	}

	g := m.mainGrid(innerW, innerH)
	col := g.ColumnAtX(innerX)
	if col < 0 {
		return m, nil
	}
	if innerY == 0 {
		return m, m.applyMainCol(col)
	}
	if innerY == 1 {
		return m, nil
	}
	row := g.OffsetRow + innerY - 2
	if row < 0 || row >= g.NumRows() {
		return m, nil
	}
	return m, m.applyMainCell(row, col)
}

func (m Model) handleRightPaneClick(x, y int) (tea.Model, tea.Cmd, bool) {
	px, py, pw, ph, ok := m.rightPaneGeom()
	if !ok {
		return m, nil, false
	}
	if x < px || x >= px+pw || y < py || y >= py+ph {
		return m, nil, false
	}

	innerX := x - px - 1
	innerY := y - py - 1
	innerW := max(1, pw-borderOverhead)
	innerH := max(1, ph-borderOverhead)
	if innerX < 0 || innerY < 0 || innerX >= innerW || innerY >= innerH {
		return m.focusRightPane(), nil, true
	}
	if m.explorer.Opened() {
		return m.applyExplorerClick(innerY), nil, true
	}
	return m.applyDetailClick(innerX, innerY), nil, true
}

func (m Model) rightPaneGeom() (x, y, w, h int, ok bool) {
	if m.width < 80 || m.height < 2 {
		return 0, 0, 0, 0, false
	}
	x = m.sidebarWidth() + m.mainPaneWidth()
	return x, 0, m.width - x, m.panesHeight(), true
}

func (m Model) focusRightPane() Model {
	if m.explorer.Opened() {
		if m.focus == FocusDetail {
			m.leaveDetailYank()
		}
		if m.focus == FocusBlameYank {
			m.leaveBlameYank()
		}
		m.focus = FocusExplorer
		m.status = "relationships"
		return m
	}
	if m.focus != FocusDetail {
		m.enterDetailYank()
	}
	return m
}

func (m Model) applyExplorerClick(innerY int) Model {
	if m.focus == FocusDetail {
		m.leaveDetailYank()
	}
	if m.focus == FocusBlameYank {
		m.leaveBlameYank()
	}
	m.layoutExplorer()
	m.focus = FocusExplorer
	m.explorer.clickRow(innerY)
	m.status = "relationships"
	return m
}

func (m Model) applyDetailClick(innerX, innerY int) Model {
	if m.focus == FocusBlameYank {
		m.leaveBlameYank()
	}
	n := m.detailVisualCount()
	row := m.detailOffset + innerY
	if n == 0 || row < 0 || row >= n {
		if m.focus != FocusDetail {
			m.enterDetailYank()
		}
		return m
	}
	plain := m.plainDetailAt(row)
	col := runeIndexAtDisplayCol(plain, innerX)
	if m.focus != FocusDetail {
		m.enterDetailYankAt(row, col)
		return m
	}
	m.yank.searchTyping = false
	m.yank.row, m.yank.col = clampYankPos([]string{plain}, 0, col)
	m.yank.row = row
	m.ensureYankVisible(m.detailViewHeight())
	m.status = "detail · " + m.yank.modeLabel()
	return m
}

func (m Model) leftPaneGeom() (x, y, w, h int, ok bool) {
	if m.width < 1 || m.height < 2 {
		return 0, 0, 0, 0, false
	}
	if m.width < 80 && (m.explorer.Opened() || m.sidebarNarrow()) {
		return 0, 0, 0, 0, false
	}
	w = m.mainPaneWidth()
	x = m.sidebarWidth()
	if m.width < 80 {
		x = 0
		w = m.width
	}
	return x, 0, w, m.panesHeight(), true
}

func (m Model) sidebarPaneGeom() (x, y, w, h int, ok bool) {
	if !m.sidebarShowing() || m.height < 2 {
		return 0, 0, 0, 0, false
	}
	if m.sidebarNarrow() {
		return 0, 0, m.width, m.panesHeight(), true
	}
	return 0, 0, m.sidebarWidth(), m.panesHeight(), true
}

func (m Model) handleSidebarClick(x, y int) (Model, bool) {
	px, py, pw, ph, ok := m.sidebarPaneGeom()
	if !ok {
		return m, false
	}
	if x < px || x >= px+pw || y < py || y >= py+ph {
		return m, false
	}
	innerY := y - py - 1
	return m.applySidebarClick(innerY), true
}

func (m Model) mainGrid(width, height int) Grid {
	switch m.main {
	case MainFiles:
		return m.filesGrid(width, height)
	case MainHistory:
		return m.historyGrid(width, height)
	case MainBlame:
		return m.blameGrid(width, height)
	case MainLineEvo:
		return m.evoGrid(width, height)
	case MainPickaxe:
		return m.pickaxeGrid(width, height)
	default:
		return m.commitGrid(width, height)
	}
}

func (m *Model) applyMainCol(col int) tea.Cmd {
	switch m.main {
	case MainFiles:
		m.fileCol = col
	case MainHistory:
		m.historyCol = col
	case MainBlame:
		m.blameCol = col
	case MainLineEvo:
		m.evoCol = col
	case MainPickaxe:
		m.pickCol = col
	default:
		m.commitCol = col
	}
	m.refreshStatus()
	return nil
}

func (m *Model) applyMainCell(row, col int) tea.Cmd {
	rowChanged := false
	switch m.main {
	case MainFiles:
		rowChanged = row != m.fileCursor
		m.fileCursor = row
		m.fileCol = col
		m.ensureFileVisible()
	case MainHistory:
		rowChanged = row != m.historyCursor
		m.historyCursor = row
		m.historyCol = col
		m.ensureHistoryVisible()
	case MainBlame:
		rowChanged = row != m.blameCursor
		m.blameCursor = row
		m.blameCol = col
		m.ensureBlameVisible()
	case MainLineEvo:
		rowChanged = row != m.evoCursor
		m.evoCursor = row
		m.evoCol = col
		m.ensureEvoVisible()
	case MainPickaxe:
		rowChanged = row != m.pickCursor
		m.pickCursor = row
		m.pickCol = col
		m.ensurePickVisible()
	default:
		rowChanged = row != m.cursor
		m.cursor = row
		m.commitCol = col
		m.ensureCommitVisible()
	}
	m.refreshStatus()
	if rowChanged {
		return m.reloadDetail()
	}
	return nil
}
