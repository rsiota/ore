package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
)

func (m Model) blameYankLines() []string {
	idx := m.blameIndices()
	out := make([]string, len(idx))
	for i, src := range idx {
		out[i] = blameCell(m.blame[src], blameColCode)
	}
	return out
}

func (m *Model) enterBlameYank() {
	lines := m.blameYankLines()
	m.yank.reset()
	row := m.blameCursor
	if row < 0 {
		row = 0
	}
	if len(lines) > 0 && row >= len(lines) {
		row = len(lines) - 1
	}
	m.yank.row, m.yank.col = clampYankPos(lines, row, 0)
	m.blameCol = blameColCode
	m.focus = FocusBlameYank
	m.ensureBlameYankVisible(max(1, m.mainListHeight()))
	m.status = "blame · " + m.yank.modeLabel() + " · esc leave · y yank · tab detail"
}

func (m *Model) leaveBlameYank() {
	m.yank.clearChord()
	m.yank.clearFlash()
	m.yank.visual = yankVisualNone
	m.yank.searchTyping = false
	m.yank.search = ""
	m.focus = FocusMain
	m.refreshStatus()
}

func (m *Model) ensureBlameYankVisible(viewH int) {
	m.blameCursor = m.yank.row
	ensureVisible(&m.blameOffset, m.blameCursor, viewH)
	lines := m.blameYankLines()
	if m.yank.row < 0 || m.yank.row >= len(lines) {
		return
	}
	plain := lines[m.yank.row]
	x := runeDisplayCol(plain, m.yank.col)
	codeW := m.blameCodeInnerWidth()
	if x < m.blameCodeScroll {
		m.blameCodeScroll = x
	}
	if x >= m.blameCodeScroll+codeW {
		m.blameCodeScroll = max(0, x-codeW+1)
	}
	if m.blameCodeScroll < 0 {
		m.blameCodeScroll = 0
	}
}

func (m Model) blameCodeInnerWidth() int {
	w := m.mainPaneWidth() - borderOverhead
	for c := 0; c < blameColCode; c++ {
		if blameColIsHidden(m.blameGutterFold, c) {
			continue
		}
		cap := 8
		if c < len(blameMetaMaxContent) && blameMetaMaxContent[c] > 0 {
			cap = blameMetaMaxContent[c]
		}
		w -= cap + 2*cellPad + 1 // content + pad + sep
	}
	return max(8, w-2*cellPad)
}

// runeIndexAtDisplayCol returns the rune index whose display column is >= dcol.
func runeIndexAtDisplayCol(s string, dcol int) int {
	if dcol <= 0 {
		return 0
	}
	w := 0
	rs := []rune(s)
	for i, r := range rs {
		if w >= dcol {
			return i
		}
		w += runewidth.RuneWidth(r)
	}
	return len(rs)
}

func (m Model) paintBlameYankCell(row int, raw string, width int) string {
	idx := m.blameIndices()
	plain := raw
	newest, oldest := blameAgeRange(m.blame)
	base := lipgloss.NewStyle()
	if row >= 0 && row < len(idx) {
		base = blameAgeStyle(m.blame[idx[row]].When, newest, oldest)
	}
	search := m.pickaxeSearchSpans(plain)
	var styled string
	if len(search) == 0 {
		styled = base.Render(plain)
	} else {
		w := lipgloss.Width(plain)
		if w < 1 {
			w = 1
		}
		styled = renderLayeredCell(base, base, styleSearchStrong, plain, nil, search, w)
	}

	scroll := m.blameCodeScroll
	startRune := runeIndexAtDisplayCol(plain, scroll)
	scrolledPlain := displaySkip(plain, scroll)
	scrolledStyled := ansi.Cut(styled, scroll, scroll+runewidth.StringWidth(plain)+16)

	y := m.yank
	y.col = max(0, y.col-startRune)
	y.anchorCol = max(0, y.anchorCol-startRune)
	y.flashAC = max(0, y.flashAC-startRune)
	y.flashC = max(0, y.flashC-startRune)

	inner := max(1, width-2*cellPad)
	pad := strings.Repeat(" ", cellPad)
	if scroll > 0 {
		marker := "…"
		mw := runewidth.StringWidth(marker)
		bodyW := max(1, inner-mw)
		body := paintYankOnStyled(scrolledStyled, scrolledPlain, bodyW, row, y)
		return fitWidth(pad+styleMuted.Render(marker)+body+pad, width)
	}
	body := paintYankOnStyled(scrolledStyled, scrolledPlain, inner, row, y)
	return fitWidth(pad+body+pad, width)
}

func (m Model) handleBlameYankKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	prevHash := m.selectedHash()
	mm, cmd := m.handleYankBrowserKeys(msg, yankBrowserOpts{
		prefix:    "blame",
		lines:     m.blameYankLines(),
		viewH:     max(1, m.mainListHeight()),
		allowHunk: false,
		leave:     (*Model).leaveBlameYank,
		ensure:    (*Model).ensureBlameYankVisible,
		onTab: func(m *Model) {
			m.leaveBlameYank()
			m.enterDetailYank()
		},
	})
	m2 := mm.(Model)
	if m2.focus == FocusBlameYank {
		if next := m2.blameReloadIfCommitChanged(prevHash); next != nil {
			return m2, tea.Batch(cmd, next)
		}
	}
	return m2, cmd
}
