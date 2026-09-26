package ui

import tea "github.com/charmbracelet/bubbletea"

// handleMouse routes mouse events. Overlays own the screen; otherwise a
// left-click in the main grid selects that cell (creel results-panel style).
func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.help.Visible() || m.palette.IsVisible() || m.branches.IsVisible() || m.bookmarks.IsVisible() {
		return m, nil
	}
	if m.exTyping || m.filterTyping {
		return m, nil
	}
	if !isMouseLeftClick(msg) {
		return m, nil
	}
	return m.handleMainGridClick(msg.X, msg.Y)
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

func (m Model) leftPaneGeom() (x, y, w, h int, ok bool) {
	if m.width < 1 || m.height < 2 {
		return 0, 0, 0, 0, false
	}
	if m.width < 80 && m.explorer.Opened() {
		return 0, 0, 0, 0, false
	}
	w = m.mainPaneWidth()
	if m.width < 80 {
		w = m.width
	}
	return 0, 0, w, m.panesHeight(), true
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
