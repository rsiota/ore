package ui

import tea "github.com/charmbracelet/bubbletea"

func isPaneNavKey(key string) bool {
	switch key {
	case "tab", "shift+tab", "ctrl+h", "ctrl+l":
		return true
	default:
		return false
	}
}

func (m Model) handlePaneNav(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "tab":
		return m.cycleFocus()
	case "shift+tab":
		return m.cycleFocusBack()
	case "ctrl+h", "ctrl+l":
		return m.moveFocus(key)
	default:
		return m, nil
	}
}

// focusRing is the Tab order: tree (when open) → main → hunks → blame yank →
// detail / explorer. Matches creel's cycle, with ore's extra surfaces inserted
// in reading order.
func (m Model) focusRing() []Focus {
	var ring []Focus
	if m.sidebarOpen {
		ring = append(ring, FocusSidebar)
	}
	ring = append(ring, FocusMain)
	if m.hunkMode {
		ring = append(ring, FocusHunks)
	}
	if m.main == MainBlame && !m.explorer.Opened() {
		ring = append(ring, FocusBlameYank)
	}
	if m.explorer.Opened() {
		ring = append(ring, FocusExplorer)
	} else {
		ring = append(ring, FocusDetail)
	}
	return ring
}

func (m Model) cycleFocus() (tea.Model, tea.Cmd) {
	return m.applyPaneFocus(m.ringNeighbour(1))
}

func (m Model) cycleFocusBack() (tea.Model, tea.Cmd) {
	return m.applyPaneFocus(m.ringNeighbour(-1))
}

func (m Model) ringNeighbour(delta int) Focus {
	ring := m.focusRing()
	if len(ring) == 0 {
		return FocusMain
	}
	idx := 0
	for i, f := range ring {
		if f == m.focus {
			idx = i
			break
		}
	}
	n := len(ring)
	return ring[(idx+delta%n+n)%n]
}

// moveFocus hops columns the way creel does: ctrl+l right, ctrl+h left.
// Columns are tree | main | detail (or explorer). Hunks and blame yank stay
// in the centre column, so a left/right hop never lands on them.
func (m Model) moveFocus(direction string) (tea.Model, tea.Cmd) {
	col := m.focusColumn()
	switch direction {
	case "ctrl+l":
		if dest, ok := m.columnHub(col + 1); ok {
			return m.applyPaneFocus(dest)
		}
	case "ctrl+h":
		if dest, ok := m.columnHub(col - 1); ok {
			return m.applyPaneFocus(dest)
		}
	}
	return m, nil
}

func (m Model) focusColumn() int {
	switch m.focus {
	case FocusSidebar:
		return 0
	case FocusDetail, FocusExplorer:
		return 2
	default:
		return 1
	}
}

func (m Model) columnHub(col int) (Focus, bool) {
	switch col {
	case 0:
		if m.sidebarOpen {
			return FocusSidebar, true
		}
	case 1:
		return FocusMain, true
	case 2:
		if m.explorer.Opened() {
			return FocusExplorer, true
		}
		if m.width >= 80 {
			return FocusDetail, true
		}
	}
	return 0, false
}

func (m Model) applyPaneFocus(next Focus) (tea.Model, tea.Cmd) {
	if next == m.focus {
		return m, nil
	}
	switch m.focus {
	case FocusDetail:
		m.leaveDetailYank()
	case FocusBlameYank:
		m.leaveBlameYank()
	}
	switch next {
	case FocusSidebar:
		m.focus = FocusSidebar
		m.status = m.treeStatus()
		return m, m.ensureTreeLoaded()
	case FocusHunks:
		m.rebuildDetailHunks()
		m.focus = FocusHunks
		if len(m.hunks) == 0 {
			m.status = "hunks · none in this detail"
		} else {
			m.status = fmtHunkStatus(m.hunkCursor, len(m.hunks), m.hunks[m.hunkCursor])
		}
		return m, nil
	case FocusBlameYank:
		m.enterBlameYank()
		return m, nil
	case FocusDetail:
		m.enterDetailYank()
		return m, nil
	case FocusExplorer:
		m.focus = FocusExplorer
		m.status = "relationships"
		return m, nil
	default:
		m.focus = FocusMain
		m.refreshStatus()
		return m, nil
	}
}
