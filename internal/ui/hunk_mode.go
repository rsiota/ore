package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
)

// detailHunk is one @@ header located in the detail visual rows.
type detailHunk struct {
	Row   int    // visual row index (detailYankLines)
	Title string // @@ header text
	File  string // nearest file path above the hunk, when known
}

const (
	hunkStripInnerRows = 5
	hunkStripChrome    = 2 // border top+bottom
)

func hunkStripOuterHeight() int {
	return hunkStripInnerRows + hunkStripChrome
}

// collectDetailHunks builds the hunk list from painted detail rows, attaching
// file paths from the logical body in hunk order.
func collectDetailHunks(visualPlain []string, logical []string) []detailHunk {
	var logFiles []string
	file := ""
	for _, line := range logical {
		raw := ansi.Strip(line)
		switch {
		case strings.HasPrefix(line, zenFilePrefix):
			file = strings.TrimPrefix(line, zenFilePrefix)
		case strings.HasPrefix(line, "diff --git "):
			file = diffGitPath(line)
		case strings.HasPrefix(line, zenHunkPrefix):
			logFiles = append(logFiles, file)
		case strings.HasPrefix(strings.TrimSpace(raw), "@@"):
			logFiles = append(logFiles, file)
		}
	}

	var out []detailHunk
	for i, line := range visualPlain {
		plain := strings.TrimSpace(line)
		if !strings.HasPrefix(plain, "@@") {
			continue
		}
		title := plain
		if runewidth.StringWidth(title) > 72 {
			title = runewidth.Truncate(title, 72, "…")
		}
		f := ""
		if len(out) < len(logFiles) {
			f = logFiles[len(out)]
		}
		out = append(out, detailHunk{Row: i, Title: title, File: f})
	}
	return out
}

func diffGitPath(line string) string {
	// diff --git a/foo b/foo
	fields := strings.Fields(line)
	if len(fields) >= 4 {
		return strings.TrimPrefix(fields[3], "b/")
	}
	if len(fields) >= 3 {
		return strings.TrimPrefix(fields[2], "a/")
	}
	return ""
}

func (m *Model) rebuildDetailHunks() {
	width := m.detailInnerWidth()
	logical := m.ensureDetailLogical(width)
	visual := m.detailYankLines()
	m.hunks = collectDetailHunks(visual, logical)
	if m.hunkCursor >= len(m.hunks) {
		m.hunkCursor = max(0, len(m.hunks)-1)
	}
	if m.hunkCursor < 0 {
		m.hunkCursor = 0
	}
	m.ensureHunkVisible()
}

func (m *Model) toggleHunkMode() {
	m.hunkMode = !m.hunkMode
	if !m.hunkMode {
		if m.focus == FocusHunks {
			m.focus = FocusMain
		}
		m.refreshStatus()
		m.status = "hunks off"
		return
	}
	m.rebuildDetailHunks()
	if len(m.hunks) == 0 {
		m.status = "hunks · none in this detail"
	} else {
		m.status = fmtHunkStatus(m.hunkCursor, len(m.hunks), m.hunks[m.hunkCursor])
	}
}

func fmtHunkStatus(i, n int, h detailHunk) string {
	label := h.Title
	if h.File != "" {
		label = h.File + " · " + h.Title
	}
	return fmt.Sprintf("hunk %d/%d · %s", i+1, n, label)
}

func (m *Model) ensureHunkVisible() {
	n := hunkStripInnerRows
	if m.hunkCursor < m.hunkOffset {
		m.hunkOffset = m.hunkCursor
	}
	if m.hunkCursor >= m.hunkOffset+n {
		m.hunkOffset = m.hunkCursor - n + 1
	}
	if m.hunkOffset < 0 {
		m.hunkOffset = 0
	}
}

// selectHunk jumps detail (and yank, when focused) to hunk i.
func (m *Model) selectHunk(i int) tea.Cmd {
	m.rebuildDetailHunks()
	if len(m.hunks) == 0 {
		m.status = "hunks · none in this detail"
		return nil
	}
	if i < 0 {
		i = 0
	}
	if i >= len(m.hunks) {
		i = len(m.hunks) - 1
	}
	m.hunkCursor = i
	m.ensureHunkVisible()
	h := m.hunks[i]
	m.detailOffset = h.Row
	if m.focus == FocusDetail {
		lines := m.detailYankLines()
		m.yank.row, m.yank.col = clampYankPos(lines, h.Row, 0)
		m.ensureYankVisible(max(1, m.detailViewHeight()))
	}
	m.status = fmtHunkStatus(i, len(m.hunks), h)
	return nil
}

// jumpHunkList moves the hunk cursor by dir (-1/+1) and selects it.
func (m *Model) jumpHunkList(dir int) tea.Cmd {
	m.rebuildDetailHunks()
	if len(m.hunks) == 0 {
		m.status = "hunks · none in this detail"
		return nil
	}
	next := m.hunkCursor + dir
	if next < 0 {
		m.status = fmtHunkStatus(0, len(m.hunks), m.hunks[0]) + " · top"
		return nil
	}
	if next >= len(m.hunks) {
		last := len(m.hunks) - 1
		m.status = fmtHunkStatus(last, len(m.hunks), m.hunks[last]) + " · bottom"
		return nil
	}
	return m.selectHunk(next)
}

// syncHunkCursorFromDetailRow sets hunkCursor from a detail visual row.
func (m *Model) syncHunkCursorFromDetailRow(row int) {
	if !m.hunkMode {
		return
	}
	if len(m.hunks) == 0 {
		m.rebuildDetailHunks()
	}
	best := -1
	for i, h := range m.hunks {
		if h.Row <= row {
			best = i
		}
		if h.Row == row {
			best = i
			break
		}
		if h.Row > row {
			break
		}
	}
	if best >= 0 {
		m.hunkCursor = best
		m.ensureHunkVisible()
	}
}

func (m Model) renderHunkStrip(width, height int) string {
	innerW := max(1, width-borderOverhead)
	innerH := max(1, height-borderOverhead)
	var lines []string
	title := styleMuted.Render(" hunks")
	if m.focus == FocusHunks {
		title = styleSelected.Render(" hunks")
	}
	count := ""
	if len(m.hunks) > 0 {
		count = styleMuted.Render(" · " + strconv.Itoa(len(m.hunks)))
	}
	lines = append(lines, fitWidth(title+count, innerW))

	if len(m.hunks) == 0 {
		lines = append(lines, styleMuted.Render("  (no hunks in this detail)"))
	} else {
		start := m.hunkOffset
		end := min(len(m.hunks), start+innerH-1)
		for i := start; i < end; i++ {
			lines = append(lines, renderHunkRow(m.hunks[i], i, i == m.hunkCursor && m.focus == FocusHunks, innerW))
		}
	}
	for len(lines) < innerH {
		lines = append(lines, "")
	}
	body := strings.Join(lines[:innerH], "\n")
	return m.framePane(body, width, height, FocusHunks)
}

func renderHunkRow(h detailHunk, idx int, selected bool, width int) string {
	num := runewidth.FillLeft(strconv.Itoa(idx+1), 3)
	label := h.Title
	if h.File != "" {
		base := h.File
		if i := strings.LastIndex(base, "/"); i >= 0 && i+1 < len(base) {
			base = base[i+1:]
		}
		label = base + "  " + h.Title
	}
	text := " " + num + " " + label
	text = runewidth.Truncate(text, max(1, width), "…")
	if selected {
		// Soft wash on the text only — avoid a full-width primary bar.
		styled := lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			Background(colorRowFocusBg).
			Render(text)
		pad := max(0, width-lipgloss.Width(styled))
		return styled + strings.Repeat(" ", pad)
	}
	line := runewidth.FillRight(text, width)
	return lipgloss.NewStyle().Foreground(colorPrimary).Render(line)
}
