package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleHunkKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		m.focus = FocusMain
		m.refreshStatus()
		return m, nil
	case "tab":
		m.focus = FocusMain
		if m.main == MainBlame {
			m.enterBlameYank()
		} else {
			m.enterDetailYank()
		}
		return m, nil
	case "H":
		m.toggleHunkMode()
		return m, nil
	case "j", "down", "]", "}":
		return m, m.jumpHunkList(1)
	case "k", "up", "[", "{":
		return m, m.jumpHunkList(-1)
	case "g":
		return m, m.selectHunk(0)
	case "G":
		m.rebuildDetailHunks()
		if len(m.hunks) == 0 {
			m.status = "hunks · none in this detail"
			return m, nil
		}
		return m, m.selectHunk(len(m.hunks) - 1)
	case "enter", "l":
		m.rebuildDetailHunks()
		if len(m.hunks) == 0 {
			m.status = "hunks · none in this detail"
			return m, nil
		}
		return m, m.selectHunk(m.hunkCursor)
	case "ctrl+d":
		return m, m.jumpHunkList(hunkStripInnerRows)
	case "ctrl+u":
		return m, m.jumpHunkList(-hunkStripInnerRows)
	}
	return m, nil
}
