package ui

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
)

// yankVisual is the selection mode inside the detail yank browser.
type yankVisual int

const (
	yankVisualNone yankVisual = iota
	yankVisualChar
	yankVisualLine
)

// yankPending mirrors creel's thin operator-pending chords (y → y/w/$).
type yankPending int

const (
	yankPendingNone yankPending = iota
	yankPendingY
)

// detailYank is a thin readonly vim-like browser for the detail pane.
// Motions, visual, and yank only — no insert, no mutating operators.
// Patterns borrowed from creel's VimBuffer ReadOnly stance / chords / esc layers;
// the textarea engine itself is not ported.
type detailYank struct {
	row, col     int
	visual       yankVisual
	anchorRow    int
	anchorCol    int
	chordG       bool
	pending      yankPending
	pendingFind  bool // f/F/t/T awaiting target
	findTill     bool
	findBack     bool
	lastFind     rune
	lastFindTill bool
	lastFindBack bool
	searchTyping bool
	search       string
	lastSearch   string
	register     string // last yanked text (creel-style internal register)

	// Copy flash (creel cell-flash shape): soft wash toggles over the yanked region.
	flashActive bool
	flashOn     bool
	flashTicks  int
	flashVisual yankVisual
	flashAR     int
	flashAC     int
	flashR      int
	flashC      int
}

// Yank region flash: off → on → off so the pulse is visible after visual clears.
const (
	yankFlashInterval  = 120 // ms per beat
	yankFlashTickCount = 2   // start off; tick1 on; tick2 clear
)

func (y *detailYank) reset() {
	reg := y.register
	*y = detailYank{register: reg}
}

func (y *detailYank) clearChord() {
	y.chordG = false
	y.pending = yankPendingNone
	y.pendingFind = false
}

func (y *detailYank) clearFlash() {
	y.flashActive = false
	y.flashOn = false
	y.flashTicks = 0
}

func (y *detailYank) startFlash(sel detailYank) {
	vis := sel.visual
	if vis == yankVisualNone {
		vis = yankVisualLine
	}
	y.flashActive = true
	y.flashOn = false // beat 1: off (gap after visual deselect)
	y.flashTicks = yankFlashTickCount
	y.flashVisual = vis
	y.flashAR, y.flashAC = sel.anchorRow, sel.anchorCol
	y.flashR, y.flashC = sel.row, sel.col
	if vis == yankVisualLine {
		// Keep the full anchor→cursor row span; only normalize cols.
		y.flashAC, y.flashC = 0, 0
	}
}

func (y *detailYank) flashSelection() detailYank {
	return detailYank{
		visual:    y.flashVisual,
		anchorRow: y.flashAR,
		anchorCol: y.flashAC,
		row:       y.flashR,
		col:       y.flashC,
	}
}

// AdvanceFlash steps off→on→off. Returns whether another tick is needed.
func (y *detailYank) AdvanceFlash() bool {
	if !y.flashActive {
		return false
	}
	y.flashTicks--
	if y.flashTicks <= 0 {
		// beat 3: off / done
		y.clearFlash()
		return false
	}
	// beat 2: on
	y.flashOn = true
	return true
}

func (y *detailYank) modeLabel() string {
	switch {
	case y.searchTyping:
		return "SEARCH"
	case y.visual == yankVisualLine:
		return "V-LINE"
	case y.visual == yankVisualChar:
		return "VISUAL"
	default:
		return "VIEW"
	}
}

func plainDetailLines(styled []string) []string {
	out := make([]string, len(styled))
	for i, s := range styled {
		out[i] = ansi.Strip(s)
	}
	return out
}

func lineRunes(s string) []rune { return []rune(s) }

func clampYankPos(lines []string, row, col int) (int, int) {
	if len(lines) == 0 {
		return 0, 0
	}
	if row < 0 {
		row = 0
	}
	if row >= len(lines) {
		row = len(lines) - 1
	}
	rs := lineRunes(lines[row])
	if col < 0 {
		col = 0
	}
	if len(rs) == 0 {
		return row, 0
	}
	if col >= len(rs) {
		col = len(rs) - 1
	}
	return row, col
}

func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func moveLeft(lines []string, row, col int) (int, int) {
	row, col = clampYankPos(lines, row, col)
	if col > 0 {
		return row, col - 1
	}
	if row > 0 {
		row--
		rs := lineRunes(lines[row])
		if len(rs) == 0 {
			return row, 0
		}
		return row, len(rs) - 1
	}
	return row, col
}

func moveRight(lines []string, row, col int) (int, int) {
	row, col = clampYankPos(lines, row, col)
	rs := lineRunes(lines[row])
	if col+1 < len(rs) {
		return row, col + 1
	}
	if row+1 < len(lines) {
		return row + 1, 0
	}
	return row, col
}

func moveUp(lines []string, row, col int) (int, int) {
	if row <= 0 {
		return clampYankPos(lines, 0, col)
	}
	return clampYankPos(lines, row-1, col)
}

func moveDown(lines []string, row, col int) (int, int) {
	if row+1 >= len(lines) {
		return clampYankPos(lines, row, col)
	}
	return clampYankPos(lines, row+1, col)
}

func moveLineStart(_ []string, row, _ int) (int, int) { return row, 0 }

func moveLineEnd(lines []string, row, _ int) (int, int) {
	row, _ = clampYankPos(lines, row, 0)
	rs := lineRunes(lines[row])
	if len(rs) == 0 {
		return row, 0
	}
	return row, len(rs) - 1
}

func moveWordForward(lines []string, row, col int) (int, int) {
	row, col = clampYankPos(lines, row, col)
	for row < len(lines) {
		rs := lineRunes(lines[row])
		if col >= len(rs) {
			row++
			col = 0
			continue
		}
		if isWordChar(rs[col]) {
			for col < len(rs) && isWordChar(rs[col]) {
				col++
			}
		} else if !unicode.IsSpace(rs[col]) {
			for col < len(rs) && !isWordChar(rs[col]) && !unicode.IsSpace(rs[col]) {
				col++
			}
		}
		for col < len(rs) && unicode.IsSpace(rs[col]) {
			col++
		}
		if col < len(rs) {
			return row, col
		}
		row++
		col = 0
		if row >= len(lines) {
			break
		}
		rs = lineRunes(lines[row])
		for col < len(rs) && unicode.IsSpace(rs[col]) {
			col++
		}
		if col < len(rs) {
			return row, col
		}
	}
	return clampYankPos(lines, len(lines)-1, 1<<30)
}

func moveWordBackward(lines []string, row, col int) (int, int) {
	row, col = clampYankPos(lines, row, col)
	for {
		rs := lineRunes(lines[row])
		if col > 0 {
			col--
		} else if row > 0 {
			row--
			rs = lineRunes(lines[row])
			if len(rs) == 0 {
				col = 0
				continue
			}
			col = len(rs) - 1
		} else {
			return 0, 0
		}
		rs = lineRunes(lines[row])
		for col > 0 && unicode.IsSpace(rs[col]) {
			col--
		}
		if len(rs) == 0 {
			continue
		}
		if isWordChar(rs[col]) {
			for col > 0 && isWordChar(rs[col-1]) {
				col--
			}
			return row, col
		}
		if !unicode.IsSpace(rs[col]) {
			for col > 0 && !isWordChar(rs[col-1]) && !unicode.IsSpace(rs[col-1]) {
				col--
			}
			return row, col
		}
	}
}

func moveWordEnd(lines []string, row, col int) (int, int) {
	row, col = clampYankPos(lines, row, col)
	rs := lineRunes(lines[row])
	if len(rs) == 0 {
		return moveWordForward(lines, row, col)
	}
	if col < len(rs)-1 {
		col++
	} else if row+1 < len(lines) {
		row++
		col = 0
	}
	for {
		rs = lineRunes(lines[row])
		for col < len(rs) && unicode.IsSpace(rs[col]) {
			col++
		}
		if col >= len(rs) {
			if row+1 >= len(lines) {
				return clampYankPos(lines, row, 1<<30)
			}
			row++
			col = 0
			continue
		}
		if isWordChar(rs[col]) {
			for col+1 < len(rs) && isWordChar(rs[col+1]) {
				col++
			}
			return row, col
		}
		for col+1 < len(rs) && !isWordChar(rs[col+1]) && !unicode.IsSpace(rs[col+1]) {
			col++
		}
		return row, col
	}
}

func findOnLine(lines []string, row, col int, target rune, till, back bool) (int, int, bool) {
	row, col = clampYankPos(lines, row, col)
	rs := lineRunes(lines[row])
	if len(rs) == 0 {
		return row, col, false
	}
	if back {
		for i := col - 1; i >= 0; i-- {
			if rs[i] == target {
				if till {
					return row, min(i+1, len(rs)-1), true
				}
				return row, i, true
			}
		}
		return row, col, false
	}
	for i := col + 1; i < len(rs); i++ {
		if rs[i] == target {
			if till {
				return row, max(0, i-1), true
			}
			return row, i, true
		}
	}
	return row, col, false
}

func searchForward(lines []string, row, col int, query string) (int, int, bool) {
	q := strings.ToLower(query)
	if q == "" || len(lines) == 0 {
		return row, col, false
	}
	startRow, startCol := row, col
	r, c := row, col+1
	for {
		if r >= len(lines) {
			r, c = 0, 0
		}
		rs := lineRunes(lines[r])
		if c > len(rs) {
			c = len(rs)
		}
		low := strings.ToLower(string(rs[c:]))
		if i := strings.Index(low, q); i >= 0 {
			return r, c + utf8.RuneCountInString(low[:i]), true
		}
		r++
		c = 0
		if r == startRow {
			// Last segment: from 0..startCol on start row.
			rs = lineRunes(lines[r])
			limit := min(startCol, len(rs))
			low = strings.ToLower(string(rs[:limit]))
			if i := strings.Index(low, q); i >= 0 {
				return r, utf8.RuneCountInString(low[:i]), true
			}
			return row, col, false
		}
		if r >= len(lines) {
			r = 0
		}
	}
}

func searchBackward(lines []string, row, col int, query string) (int, int, bool) {
	q := strings.ToLower(query)
	if q == "" || len(lines) == 0 {
		return row, col, false
	}
	startRow, startCol := row, col
	r, c := row, col
	for {
		rs := lineRunes(lines[r])
		limit := min(c, len(rs))
		low := strings.ToLower(string(rs[:limit]))
		if i := strings.LastIndex(low, q); i >= 0 {
			return r, utf8.RuneCountInString(low[:i]), true
		}
		if r == 0 {
			r = len(lines) - 1
			c = len(lineRunes(lines[r]))
		} else {
			r--
			c = len(lineRunes(lines[r]))
		}
		if r == startRow {
			rs = lineRunes(lines[r])
			from := min(startCol+1, len(rs))
			low = strings.ToLower(string(rs[from:]))
			if i := strings.LastIndex(low, q); i >= 0 {
				return r, from + utf8.RuneCountInString(low[:i]), true
			}
			return row, col, false
		}
	}
}

func wordFlashRegion(lines []string, row, col int) detailYank {
	row, col = clampYankPos(lines, row, col)
	rs := lineRunes(lines[row])
	if len(rs) == 0 {
		return detailYank{visual: yankVisualChar, row: row, col: 0, anchorRow: row, anchorCol: 0}
	}
	lo, hi := col, col
	if isWordChar(rs[col]) {
		for lo > 0 && isWordChar(rs[lo-1]) {
			lo--
		}
		for hi+1 < len(rs) && isWordChar(rs[hi+1]) {
			hi++
		}
	}
	return detailYank{visual: yankVisualChar, row: row, col: hi, anchorRow: row, anchorCol: lo}
}

func restFlashRegion(lines []string, row, col int) detailYank {
	row, col = clampYankPos(lines, row, col)
	rs := lineRunes(lines[row])
	hi := col
	if len(rs) > 0 {
		hi = len(rs) - 1
	}
	return detailYank{visual: yankVisualChar, row: row, col: hi, anchorRow: row, anchorCol: col}
}

func lineFlashRegion(row, col int) detailYank {
	return detailYank{visual: yankVisualLine, row: row, col: col, anchorRow: row, anchorCol: 0}
}

func wordAtCursor(lines []string, row, col int) string {
	row, col = clampYankPos(lines, row, col)
	rs := lineRunes(lines[row])
	if len(rs) == 0 {
		return ""
	}
	if !isWordChar(rs[col]) {
		return string(rs[col])
	}
	lo, hi := col, col
	for lo > 0 && isWordChar(rs[lo-1]) {
		lo--
	}
	for hi+1 < len(rs) && isWordChar(rs[hi+1]) {
		hi++
	}
	return string(rs[lo : hi+1])
}

func restOfLine(lines []string, row, col int) string {
	row, col = clampYankPos(lines, row, col)
	rs := lineRunes(lines[row])
	if col >= len(rs) {
		return ""
	}
	return string(rs[col:])
}

// yankSelection returns visual selection text, or the current line when not visual.
func yankSelection(lines []string, y detailYank) string {
	if len(lines) == 0 {
		return ""
	}
	row, col := clampYankPos(lines, y.row, y.col)
	switch y.visual {
	case yankVisualNone:
		return lines[row]
	case yankVisualLine:
		r0, r1 := y.anchorRow, row
		if r0 > r1 {
			r0, r1 = r1, r0
		}
		r0 = max(0, min(r0, len(lines)-1))
		r1 = max(0, min(r1, len(lines)-1))
		return strings.Join(lines[r0:r1+1], "\n")
	default:
		r0, c0 := y.anchorRow, y.anchorCol
		r1, c1 := row, col
		if r0 > r1 || (r0 == r1 && c0 > c1) {
			r0, c0, r1, c1 = r1, c1, r0, c0
		}
		r0, c0 = clampYankPos(lines, r0, c0)
		r1, c1 = clampYankPos(lines, r1, c1)
		if r0 == r1 {
			rs := lineRunes(lines[r0])
			if len(rs) == 0 {
				return ""
			}
			return string(rs[c0 : c1+1])
		}
		var b strings.Builder
		rs0 := lineRunes(lines[r0])
		if len(rs0) > 0 {
			b.WriteString(string(rs0[c0:]))
		}
		b.WriteByte('\n')
		for r := r0 + 1; r < r1; r++ {
			b.WriteString(lines[r])
			b.WriteByte('\n')
		}
		rs1 := lineRunes(lines[r1])
		if len(rs1) > 0 {
			b.WriteString(string(rs1[:c1+1]))
		}
		return b.String()
	}
}

func inYankSelection(y detailYank, row, col int) bool {
	if y.visual == yankVisualNone {
		return false
	}
	r0, c0 := y.anchorRow, y.anchorCol
	r1, c1 := y.row, y.col
	if y.visual == yankVisualLine {
		if r0 > r1 {
			r0, r1 = r1, r0
		}
		return row >= r0 && row <= r1
	}
	if r0 > r1 || (r0 == r1 && c0 > c1) {
		r0, c0, r1, c1 = r1, c1, r0, c0
	}
	if row < r0 || row > r1 {
		return false
	}
	if row == r0 && row == r1 {
		return col >= c0 && col <= c1
	}
	if row == r0 {
		return col >= c0
	}
	if row == r1 {
		return col <= c1
	}
	return true
}

func yankCursorStyle() lipgloss.Style {
	// creel normal-mode block cursor: reverse video
	return lipgloss.NewStyle().Reverse(true)
}

func yankSelStyle() lipgloss.Style {
	// Soft slate wash (same family as row focus) — keep body fg readable,
	// not inverted primary/black.
	return lipgloss.NewStyle().Foreground(colorFg).Background(colorRowFocusBg)
}

// runeDisplayCol returns the screen column of rune index i in plain text.
func runeDisplayCol(plain string, i int) int {
	rs := lineRunes(plain)
	if i <= 0 {
		return 0
	}
	if i > len(rs) {
		i = len(rs)
	}
	return runewidth.StringWidth(string(rs[:i]))
}

// paintYankOnStyled keeps the normal detail highlighting and only overlays
// visual-mode selection (or a post-yank flash wash) plus a reverse block cursor.
func paintYankOnStyled(styled, plain string, width, row int, y detailYank) string {
	line := styled
	switch {
	case y.visual != yankVisualNone && yankRowSelected(y, row):
		line = paintYankVisual(styled, plain, width, row, y)
	case y.flashActive && y.flashOn:
		f := y.flashSelection()
		if yankRowSelected(f, row) {
			line = paintYankVisual(styled, plain, width, row, f)
		}
	}
	if row == y.row {
		line = overlayYankCursor(line, plain, y.col)
	}
	return fitWidth(line, width)
}

func yankRowSelected(y detailYank, row int) bool {
	if y.visual == yankVisualNone {
		return false
	}
	if y.visual == yankVisualLine {
		return inYankSelection(y, row, 0)
	}
	// Char visual: any overlap with this row.
	r0, r1 := y.anchorRow, y.row
	if r0 > r1 {
		r0, r1 = r1, r0
	}
	return row >= r0 && row <= r1
}

func paintYankVisual(styled, plain string, width, row int, y detailYank) string {
	sel := yankSelStyle()
	if y.visual == yankVisualLine {
		text := plain
		if text == "" {
			text = " "
		}
		out := sel.Render(text)
		w := lipgloss.Width(out)
		if w < width {
			out += sel.Render(strings.Repeat(" ", width-w))
		}
		return out
	}
	// Char visual: preserve highlighting outside the selection via ansi.Cut.
	c0, c1 := yankCharRangeOnRow(y, row, plain)
	if c0 < 0 {
		return fitWidth(styled, width)
	}
	rs := lineRunes(plain)
	x0 := runeDisplayCol(plain, c0)
	x1 := runeDisplayCol(plain, c1+1)
	if c1 >= len(rs) {
		x1 = max(x0+1, runewidth.StringWidth(plain))
	}
	left := ansi.Cut(styled, 0, x0)
	midPlain := ""
	if len(rs) == 0 {
		midPlain = " "
	} else if c0 < len(rs) {
		hi := min(c1+1, len(rs))
		midPlain = string(rs[c0:hi])
	}
	if midPlain == "" {
		midPlain = " "
	}
	mid := sel.Render(midPlain)
	right := ""
	bgW := lipgloss.Width(styled)
	if x1 < bgW {
		right = ansi.Cut(styled, x1, bgW)
	}
	out := left + mid + right
	if lipgloss.Width(out) < width {
		// Trailing pad: selected only when line visual would cover it; char mode stays plain.
		out += strings.Repeat(" ", width-lipgloss.Width(out))
	}
	return out
}

// yankCharRangeOnRow returns inclusive rune indices selected on row, or -1,-1.
func yankCharRangeOnRow(y detailYank, row int, plain string) (c0, c1 int) {
	if y.visual != yankVisualChar {
		return -1, -1
	}
	r0, a0 := y.anchorRow, y.anchorCol
	r1, a1 := y.row, y.col
	if r0 > r1 || (r0 == r1 && a0 > a1) {
		r0, a0, r1, a1 = r1, a1, r0, a0
	}
	if row < r0 || row > r1 {
		return -1, -1
	}
	rs := lineRunes(plain)
	last := max(0, len(rs)-1)
	switch {
	case r0 == r1 && row == r0:
		return clamp(a0, 0, last), clamp(a1, 0, last)
	case row == r0:
		return clamp(a0, 0, last), last
	case row == r1:
		return 0, clamp(a1, 0, last)
	default:
		return 0, last
	}
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func overlayYankCursor(styled, plain string, col int) string {
	rs := lineRunes(plain)
	if col < 0 {
		col = 0
	}
	if col > len(rs) {
		col = len(rs)
	}
	x := runeDisplayCol(plain, col)
	ch := " "
	if col < len(rs) {
		ch = string(rs[col])
	}
	return overlayLine(styled, yankCursorStyle().Render(ch), x)
}
