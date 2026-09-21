package ui

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

type yankCopiedMsg struct {
	n   int
	err error
}

func (m Model) detailYankLines() []string {
	return plainDetailLines(m.detailVisualLines())
}

func (m *Model) enterDetailYank() {
	lines := m.detailYankLines()
	m.yank.reset()
	row := m.detailOffset
	if row < 0 {
		row = 0
	}
	if len(lines) > 0 && row >= len(lines) {
		row = len(lines) - 1
	}
	m.yank.row, m.yank.col = clampYankPos(lines, row, 0)
	m.focus = FocusDetail
	m.status = "detail · " + m.yank.modeLabel() + " · esc leave · y yank"
}

func (m *Model) leaveDetailYank() {
	m.yank.clearChord()
	m.yank.clearFlash()
	m.yank.visual = yankVisualNone
	m.yank.searchTyping = false
	m.yank.search = ""
	m.focus = FocusMain
	m.refreshStatus()
}

func (m *Model) ensureYankVisible(viewH int) {
	if viewH < 1 {
		viewH = 1
	}
	if m.yank.row < m.detailOffset {
		m.detailOffset = m.yank.row
	}
	if m.yank.row >= m.detailOffset+viewH {
		m.detailOffset = m.yank.row - viewH + 1
	}
	if m.detailOffset < 0 {
		m.detailOffset = 0
	}
}

type yankFlashTickMsg struct{}

func yankFlashTickCmd() tea.Cmd {
	return tea.Tick(time.Duration(yankFlashInterval)*time.Millisecond, func(time.Time) tea.Msg {
		return yankFlashTickMsg{}
	})
}

func (m *Model) commitYank(text string, flash detailYank) tea.Cmd {
	if text == "" {
		m.status = "yank · empty"
		return nil
	}
	m.yank.register = text
	m.yank.startFlash(flash)
	n := utf8.RuneCountInString(text)
	m.yank.visual = yankVisualNone
	m.yank.clearChord()
	return tea.Batch(
		func() tea.Msg {
			err := clipboard.WriteAll(text)
			return yankCopiedMsg{n: n, err: err}
		},
		yankFlashTickCmd(),
	)
}

// jumpDetailHunk moves to the previous (dir<0) or next (dir>0) diff hunk header
// in the detail pane. Bound to [ / ] (and { / }) while FocusDetail.
func (m *Model) jumpDetailHunk(dir int) tea.Cmd {
	lines := m.detailYankLines()
	hunks := detailHunkRows(lines)
	if len(hunks) == 0 {
		m.status = "detail · no hunks"
		return nil
	}
	viewH := max(1, m.detailViewHeight())
	cur := m.yank.row
	var (
		row int
		ok  bool
	)
	if dir < 0 {
		row, ok = prevHunkRow(hunks, cur)
		if !ok {
			m.status = fmt.Sprintf("detail · hunk 1/%d · top", len(hunks))
			return nil
		}
	} else {
		row, ok = nextHunkRow(hunks, cur)
		if !ok {
			m.status = fmt.Sprintf("detail · hunk %d/%d · bottom", len(hunks), len(hunks))
			return nil
		}
	}
	m.yank.row, m.yank.col = clampYankPos(lines, row, 0)
	m.ensureYankVisible(viewH)
	ord := hunkOrdinal(hunks, m.yank.row)
	m.status = fmt.Sprintf("detail · hunk %d/%d", ord, len(hunks))
	if m.hunkMode {
		m.syncHunkCursorFromDetailRow(m.yank.row)
	}
	return nil
}

// yankBrowserOpts configures the shared readonly yank key handler for detail
// or blame-code surfaces.
type yankBrowserOpts struct {
	prefix    string
	lines     []string
	viewH     int
	allowHunk bool
	leave     func(*Model)
	ensure    func(*Model, int)
	onTab     func(*Model)
}

func (m Model) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m.handleYankBrowserKeys(msg, yankBrowserOpts{
		prefix:    "detail",
		lines:     m.detailYankLines(),
		viewH:     max(1, m.detailViewHeight()),
		allowHunk: true,
		leave:     (*Model).leaveDetailYank,
		ensure:    (*Model).ensureYankVisible,
	})
}

func (m Model) handleYankBrowserKeys(msg tea.KeyMsg, opts yankBrowserOpts) (tea.Model, tea.Cmd) {
	lines := opts.lines
	viewH := opts.viewH
	if viewH < 1 {
		viewH = 1
	}
	key := msg.String()
	ensure := func() {
		if opts.ensure != nil {
			opts.ensure(&m, viewH)
		}
	}
	statusMode := func() {
		m.status = opts.prefix + " · " + m.yank.modeLabel()
	}

	// Pane-local search prompt (creel / layer).
	if m.yank.searchTyping {
		switch key {
		case "esc", "ctrl+c":
			m.yank.searchTyping = false
			m.yank.search = ""
			statusMode()
			return m, nil
		case "enter":
			m.yank.lastSearch = m.yank.search
			m.yank.searchTyping = false
			m.yank.search = ""
			if m.yank.lastSearch != "" {
				if r, c, ok := searchForward(lines, m.yank.row, m.yank.col, m.yank.lastSearch); ok {
					m.yank.row, m.yank.col = r, c
					ensure()
				}
			}
			statusMode()
			return m, nil
		case "backspace":
			rs := []rune(m.yank.search)
			if len(rs) > 0 {
				m.yank.search = string(rs[:len(rs)-1])
			}
			m.status = opts.prefix + " /" + m.yank.search + "█"
			return m, nil
		}
		if len(msg.Runes) == 1 && !msg.Alt && msg.Type == tea.KeyRunes {
			m.yank.search += string(msg.Runes)
			m.status = opts.prefix + " /" + m.yank.search + "█"
			return m, nil
		}
		return m, nil
	}

	// f/F/t/T target char.
	if m.yank.pendingFind {
		m.yank.pendingFind = false
		if len(msg.Runes) == 1 {
			ch := msg.Runes[0]
			m.yank.lastFind = ch
			m.yank.lastFindTill = m.yank.findTill
			m.yank.lastFindBack = m.yank.findBack
			if r, c, ok := findOnLine(lines, m.yank.row, m.yank.col, ch, m.yank.findTill, m.yank.findBack); ok {
				m.yank.row, m.yank.col = r, c
				ensure()
			}
		}
		return m, nil
	}

	// y-pending: yy / yw / y$ (creel chord).
	if m.yank.pending == yankPendingY {
		m.yank.pending = yankPendingNone
		switch key {
		case "y":
			return m, m.commitYank(
				yankSelection(lines, detailYank{row: m.yank.row, col: m.yank.col, visual: yankVisualNone}),
				lineFlashRegion(m.yank.row, m.yank.col),
			)
		case "w":
			return m, m.commitYank(
				wordAtCursor(lines, m.yank.row, m.yank.col),
				wordFlashRegion(lines, m.yank.row, m.yank.col),
			)
		case "$":
			return m, m.commitYank(
				restOfLine(lines, m.yank.row, m.yank.col),
				restFlashRegion(lines, m.yank.row, m.yank.col),
			)
		}
		// Unknown motion after y — treat as line yank (creel fallthrough).
		return m, m.commitYank(
			yankSelection(lines, detailYank{row: m.yank.row, col: m.yank.col, visual: yankVisualNone}),
			lineFlashRegion(m.yank.row, m.yank.col),
		)
	}

	// g-pending → gg
	if m.yank.chordG {
		m.yank.chordG = false
		if key == "g" {
			m.yank.row, m.yank.col = clampYankPos(lines, 0, m.yank.col)
			ensure()
		}
		return m, nil
	}

	switch key {
	case "esc":
		if m.yank.visual != yankVisualNone {
			m.yank.visual = yankVisualNone
			statusMode()
			return m, nil
		}
		if opts.leave != nil {
			opts.leave(&m)
		}
		return m, nil
	case "tab":
		if opts.onTab != nil {
			opts.onTab(&m)
			return m, nil
		}
		if opts.leave != nil {
			opts.leave(&m)
		}
		return m, nil

	case "v":
		if m.yank.visual == yankVisualChar {
			m.yank.visual = yankVisualNone
		} else {
			m.yank.visual = yankVisualChar
			m.yank.anchorRow, m.yank.anchorCol = m.yank.row, m.yank.col
		}
		statusMode()
		return m, nil
	case "V":
		if m.yank.visual == yankVisualLine {
			m.yank.visual = yankVisualNone
		} else {
			m.yank.visual = yankVisualLine
			m.yank.anchorRow, m.yank.anchorCol = m.yank.row, m.yank.col
		}
		statusMode()
		return m, nil

	case "y":
		if m.yank.visual != yankVisualNone {
			flash := m.yank // snapshot selection before commitYank clears visual
			return m, m.commitYank(yankSelection(lines, m.yank), flash)
		}
		m.yank.pending = yankPendingY
		m.status = opts.prefix + " · y…"
		return m, nil
	case "Y":
		return m, m.commitYank(
			yankSelection(lines, detailYank{row: m.yank.row, col: m.yank.col, visual: yankVisualNone}),
			lineFlashRegion(m.yank.row, m.yank.col),
		)

	case "/":
		m.yank.searchTyping = true
		m.yank.search = ""
		m.status = opts.prefix + " /█"
		return m, nil
	case "n":
		if m.yank.lastSearch != "" {
			if r, c, ok := searchForward(lines, m.yank.row, m.yank.col, m.yank.lastSearch); ok {
				m.yank.row, m.yank.col = r, c
				ensure()
			}
		}
		return m, nil
	case "N":
		if m.yank.lastSearch != "" {
			if r, c, ok := searchBackward(lines, m.yank.row, m.yank.col, m.yank.lastSearch); ok {
				m.yank.row, m.yank.col = r, c
				ensure()
			}
		}
		return m, nil

	case "f":
		m.yank.pendingFind = true
		m.yank.findTill, m.yank.findBack = false, false
		return m, nil
	case "F":
		m.yank.pendingFind = true
		m.yank.findTill, m.yank.findBack = false, true
		return m, nil
	case "t":
		m.yank.pendingFind = true
		m.yank.findTill, m.yank.findBack = true, false
		return m, nil
	case "T":
		m.yank.pendingFind = true
		m.yank.findTill, m.yank.findBack = true, true
		return m, nil
	case ";":
		if m.yank.lastFind != 0 {
			if r, c, ok := findOnLine(lines, m.yank.row, m.yank.col, m.yank.lastFind, m.yank.lastFindTill, m.yank.lastFindBack); ok {
				m.yank.row, m.yank.col = r, c
				ensure()
			}
		}
		return m, nil
	case ",":
		if m.yank.lastFind != 0 {
			if r, c, ok := findOnLine(lines, m.yank.row, m.yank.col, m.yank.lastFind, m.yank.lastFindTill, !m.yank.lastFindBack); ok {
				m.yank.row, m.yank.col = r, c
				ensure()
			}
		}
		return m, nil

	case "[", "{":
		if opts.allowHunk {
			return m, m.jumpDetailHunk(-1)
		}
		return m, nil
	case "]", "}":
		if opts.allowHunk {
			return m, m.jumpDetailHunk(1)
		}
		return m, nil

	case "h", "left":
		m.yank.row, m.yank.col = moveLeft(lines, m.yank.row, m.yank.col)
	case "l", "right":
		m.yank.row, m.yank.col = moveRight(lines, m.yank.row, m.yank.col)
	case "j", "down":
		m.yank.row, m.yank.col = moveDown(lines, m.yank.row, m.yank.col)
	case "k", "up":
		m.yank.row, m.yank.col = moveUp(lines, m.yank.row, m.yank.col)
	case "w":
		m.yank.row, m.yank.col = moveWordForward(lines, m.yank.row, m.yank.col)
	case "b":
		m.yank.row, m.yank.col = moveWordBackward(lines, m.yank.row, m.yank.col)
	case "e":
		m.yank.row, m.yank.col = moveWordEnd(lines, m.yank.row, m.yank.col)
	case "0", "home":
		m.yank.row, m.yank.col = moveLineStart(lines, m.yank.row, m.yank.col)
	case "$", "end":
		m.yank.row, m.yank.col = moveLineEnd(lines, m.yank.row, m.yank.col)
	case "g":
		m.yank.chordG = true
		return m, nil
	case "G":
		m.yank.row, m.yank.col = clampYankPos(lines, len(lines)-1, m.yank.col)
	case "ctrl+d":
		for i := 0; i < viewH; i++ {
			m.yank.row, m.yank.col = moveDown(lines, m.yank.row, m.yank.col)
		}
	case "ctrl+u":
		for i := 0; i < viewH; i++ {
			m.yank.row, m.yank.col = moveUp(lines, m.yank.row, m.yank.col)
		}
	default:
		return m, nil
	}
	ensure()
	return m, nil
}
