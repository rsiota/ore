package ui

const (
	minMainPaneWidth  = 40
	minRightPaneWidth = 24
	paneResizeStep    = 3
)

// normalizePaneResizeKey maps resize key strings to a canonical alt+letter
// form. Bubble Tea reports ctrl+alt+letter as "alt+ctrl+letter".
func normalizePaneResizeKey(key string) (string, bool) {
	switch key {
	case "alt+h", "alt+j", "alt+k", "alt+l":
		return key, true
	case "alt+ctrl+h":
		return "alt+h", true
	case "alt+ctrl+j":
		return "alt+j", true
	case "alt+ctrl+k":
		return "alt+k", true
	case "alt+ctrl+l":
		return "alt+l", true
	default:
		return "", false
	}
}

func (m Model) clampMainPaneWidth(w int) int {
	if m.width < 80 {
		return max(20, m.width)
	}
	region := m.splitRegionWidth()
	minMain := minMainPaneWidth
	minRight := minRightPaneWidth
	if m.sidebarDocked() {
		minMain = minMainWithSidebar
		minRight = minRightWithSidebar
	}
	maxMain := region - minRight
	if w < minMain {
		w = minMain
	}
	if w > maxMain {
		w = maxMain
	}
	return w
}

// resizePane nudges the focused seam (creel alt+h/l). Sidebar focus resizes
// the tree column; otherwise the main|detail seam moves. alt+j/k are
// accepted so the keys match creel, but ore has no vertical split.
func (m Model) resizePane(direction string) Model {
	direction, ok := normalizePaneResizeKey(direction)
	if !ok || m.width < 80 {
		return m
	}
	if m.focus == FocusSidebar && m.sidebarDocked() {
		before := m.sidebarWidth()
		if m.sidebarSplitW <= 0 {
			m.sidebarSplitW = before
		}
		switch direction {
		case "alt+h":
			m.sidebarSplitW -= paneResizeStep
		case "alt+l":
			m.sidebarSplitW += paneResizeStep
		default:
			return m
		}
		m.sidebarSplitW = m.clampSidebarWidth(m.sidebarSplitW)
		if m.sidebarWidth() != before {
			m.invalidateDetailCache()
			m.layoutExplorer()
		}
		return m
	}
	before := m.mainPaneWidth()
	switch direction {
	case "alt+h":
		if m.mainPaneSplitW <= 0 {
			m.mainPaneSplitW = before
		}
		m.mainPaneSplitW -= paneResizeStep
	case "alt+l":
		if m.mainPaneSplitW <= 0 {
			m.mainPaneSplitW = before
		}
		m.mainPaneSplitW += paneResizeStep
	default:
		return m
	}
	m.mainPaneSplitW = m.clampMainPaneWidth(m.mainPaneSplitW)
	if m.mainPaneWidth() != before {
		m.invalidateDetailCache()
		m.layoutExplorer()
	}
	return m
}
