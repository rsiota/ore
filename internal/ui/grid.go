package ui

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// cellPad is the number of spaces left and right of cell content (creel-style).
const cellPad = 1

// SortDir is the sort direction for a grid column.
type SortDir int

const (
	SortNone SortDir = iota
	SortAsc
	SortDesc
)

func (d SortDir) Arrow() string {
	switch d {
	case SortAsc:
		return "↑"
	case SortDesc:
		return "↓"
	default:
		return ""
	}
}

// CycleSort advances none → asc → desc → none.
func CycleSort(dir SortDir) SortDir {
	switch dir {
	case SortNone:
		return SortAsc
	case SortAsc:
		return SortDesc
	default:
		return SortNone
	}
}

// Grid is a read-only tabular view with row/column cursor (creel-style).
type Grid struct {
	Columns []string
	Rows    [][]string
	Widths  []int // per-column total cell widths (content + left/right pad)

	CursorRow int
	CursorCol int
	OffsetRow int

	SortCol int
	SortDir SortDir

	Width   int
	Height  int
	Focused bool

	// NoStripe disables zebra striping (blame: one background language).
	NoStripe bool
	// SoftCursor paints the focused row with a light wash; the active cell
	// still uses the blue cursor unless listed in SkipCursorPaintCols.
	SoftCursor bool
	// SkipStripeCols never receive zebra striping when striping is enabled.
	SkipStripeCols []int
	// SkipCursorPaintCols never get the blue cursor cell (blame code column).
	SkipCursorPaintCols []int
	// RightAlignCols right-align cell content (and headers) within the column.
	RightAlignCols []int
	// MuteCols render muted when not otherwise styled (quiet blame gutter).
	MuteCols []int
	// HScrollCol / HScroll shift a column's content left for long cells (blame code).
	HScrollCol int
	HScroll    int
	// HiddenCols are omitted from layout (no width, no separator) so siblings
	// can reclaim space — used by blame gutter fold.
	HiddenCols []int
	// CellStyle styles a non-cursor cell. ok=true replaces the default plain/stripe look.
	CellStyle func(row, col int, text string) (styled string, ok bool)
}

// SetSize updates the viewport.
func (g *Grid) SetSize(width, height int) {
	g.Width = width
	g.Height = height
	g.ensureVisible()
}

// NumRows returns the number of data rows.
func (g Grid) NumRows() int { return len(g.Rows) }

// NumCols returns the number of columns.
func (g Grid) NumCols() int { return len(g.Columns) }

// ClampCursor keeps the cursor inside the grid.
func (g *Grid) ClampCursor() {
	if g.NumRows() == 0 {
		g.CursorRow = 0
		g.CursorCol = 0
		g.OffsetRow = 0
		return
	}
	g.CursorRow = min(max(0, g.CursorRow), g.NumRows()-1)
	if g.NumCols() == 0 {
		g.CursorCol = 0
	} else {
		g.CursorCol = min(max(0, g.CursorCol), g.NumCols()-1)
	}
	g.ensureVisible()
}

func (g *Grid) MoveUp() {
	if g.CursorRow > 0 {
		g.CursorRow--
		g.ensureVisible()
	}
}

func (g *Grid) MoveDown() {
	if g.CursorRow < g.NumRows()-1 {
		g.CursorRow++
		g.ensureVisible()
	}
}

func (g *Grid) MoveLeft() {
	if g.CursorCol > 0 {
		g.CursorCol--
	}
}

func (g *Grid) MoveRight() {
	if g.CursorCol < g.NumCols()-1 {
		g.CursorCol++
	}
}

func (g *Grid) FirstCol() { g.CursorCol = 0 }
func (g *Grid) LastCol() {
	if g.NumCols() > 0 {
		g.CursorCol = g.NumCols() - 1
	}
}

func (g *Grid) Top() {
	g.CursorRow = 0
	g.ensureVisible()
}

func (g *Grid) Bottom() {
	if g.NumRows() > 0 {
		g.CursorRow = g.NumRows() - 1
	}
	g.ensureVisible()
}

func (g *Grid) Page(delta int) {
	if g.NumRows() == 0 {
		return
	}
	g.CursorRow = min(g.NumRows()-1, max(0, g.CursorRow+delta))
	g.ensureVisible()
}

func (g *Grid) listHeight() int {
	return max(1, g.Height-2) // header + rule
}

func (g *Grid) ensureVisible() {
	h := g.listHeight()
	if g.CursorRow < g.OffsetRow {
		g.OffsetRow = g.CursorRow
	}
	if g.CursorRow >= g.OffsetRow+h {
		g.OffsetRow = g.CursorRow - h + 1
	}
}

// AutoWidths sizes columns from headers and visible content, fitting into width.
func (g *Grid) AutoWidths() {
	g.AutoWidthsCaps(nil)
}

// AutoWidthsCaps is AutoWidths with per-column max content widths (excluding
// cell padding) for every column except the last, which still flexes to fill.
// A max of 0 (or a nil/short slice entry) keeps the default 16-rune cap.
// HiddenCols get width 0 and are skipped in the fit math.
func (g *Grid) AutoWidthsCaps(maxContent []int) {
	n := g.NumCols()
	if n == 0 {
		g.Widths = nil
		return
	}
	pad := 2 * cellPad
	g.Widths = make([]int, n)
	for i, col := range g.Columns {
		if g.colHidden(i) {
			g.Widths[i] = 0
			continue
		}
		g.Widths[i] = runewidth.StringWidth(col) + 1 + pad // room for sort arrow + pad
	}
	for _, row := range g.Rows {
		for i := 0; i < n && i < len(row); i++ {
			if g.colHidden(i) {
				continue
			}
			w := runewidth.StringWidth(row[i]) + pad
			if w > g.Widths[i] {
				g.Widths[i] = w
			}
		}
	}
	minCell := 3 + pad
	flex := n - 1 // last column flexes (blame code / subject)
	for i := 0; i < n; i++ {
		if g.colHidden(i) || i == flex {
			continue
		}
		capContent := 16
		if i < len(maxContent) && maxContent[i] > 0 {
			capContent = maxContent[i]
		}
		if g.Widths[i] > capContent+pad {
			g.Widths[i] = capContent + pad
		}
		if g.Widths[i] < minCell {
			g.Widths[i] = minCell
		}
	}
	visible := 0
	used := 0
	for i := 0; i < n; i++ {
		if g.colHidden(i) {
			continue
		}
		visible++
		if i != flex {
			used += g.Widths[i]
		}
	}
	if visible > 1 {
		used += visible - 1 // │ between visible columns
	}
	remain := g.Width - used
	if remain < 8+pad {
		remain = 8 + pad
	}
	if !g.colHidden(flex) {
		g.Widths[flex] = remain
	}
}

func (g Grid) colHidden(col int) bool {
	for _, c := range g.HiddenCols {
		if c == col {
			return true
		}
	}
	return false
}

// joinVisibleCols joins only non-hidden column parts with sep.
func (g Grid) joinVisibleCols(parts []string, sep string) string {
	var out []string
	for i, p := range parts {
		if g.colHidden(i) {
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, sep)
}

// View renders the grid into Width x Height cells.
func (g Grid) View() string {
	if g.Width <= 0 || g.Height <= 0 {
		return ""
	}
	var lines []string
	if g.NumCols() == 0 {
		return padPane(lines, g.Width, g.Height)
	}

	lines = append(lines, g.renderHeader())
	lines = append(lines, g.renderHeaderRule())

	h := g.listHeight()
	end := min(g.NumRows(), g.OffsetRow+h)
	filled := 0
	if g.NumRows() == 0 {
		lines = append(lines, fitWidth(styleMuted.Render("(no rows)"), g.Width))
		filled = 1
	} else {
		for i := g.OffsetRow; i < end; i++ {
			lines = append(lines, g.renderRow(i))
			filled++
		}
	}
	// Extend unused viewport slots (zebra padding when striping is on).
	for slot := filled; slot < h; slot++ {
		lines = append(lines, g.renderEmptyRow(g.OffsetRow+slot))
	}
	return padPane(lines, g.Width, g.Height)
}

func (g Grid) renderHeader() string {
	parts := make([]string, g.NumCols())
	for i, name := range g.Columns {
		if g.colHidden(i) {
			continue
		}
		label := name
		if i == g.SortCol {
			if a := g.SortDir.Arrow(); a != "" {
				label = name + a
			}
		}
		parts[i] = renderHeaderCell(label, g.widthAt(i), g.Focused && i == g.CursorCol, g.rightAlign(i))
	}
	return fitWidth(g.joinVisibleCols(parts, g.colSep()), g.Width)
}

// renderHeaderCell draws a creel-style header: blue text, with underline
// scoped to the word when selected (padding stays un-underlined).
func renderHeaderCell(label string, totalWidth int, selected bool, right bool) string {
	inner := totalWidth - 2*cellPad
	if inner < 1 {
		inner = 1
	}
	var content string
	if right {
		content = padTrimRight(label, inner)
	} else {
		content = padTrim(label, inner)
	}
	pad := strings.Repeat(" ", cellPad)
	if !selected {
		return styleGridHeader.Render(pad + content + pad)
	}
	text := strings.TrimSpace(content)
	lead := 0
	trail := 0
	if right {
		lead = inner - runewidth.StringWidth(text)
		if lead < 0 {
			lead = 0
		}
	} else {
		trail = inner - runewidth.StringWidth(text)
		if trail < 0 {
			trail = 0
		}
	}
	var b strings.Builder
	b.WriteString(styleGridHeader.Render(pad))
	if lead > 0 {
		b.WriteString(styleGridHeader.Render(strings.Repeat(" ", lead)))
	}
	b.WriteString(styleGridHeader.Underline(true).Render(text))
	if trail > 0 {
		b.WriteString(styleGridHeader.Render(strings.Repeat(" ", trail)))
	}
	b.WriteString(styleGridHeader.Render(pad))
	return b.String()
}

// renderHeaderRule draws the muted ─┼─ line under column headers (creel-style).
func (g Grid) renderHeaderRule() string {
	var b strings.Builder
	sep := styleGridBorder.Render("┼")
	first := true
	for i := 0; i < g.NumCols(); i++ {
		if g.colHidden(i) {
			continue
		}
		if !first {
			b.WriteString(sep)
		}
		first = false
		b.WriteString(styleGridBorder.Render(strings.Repeat("─", g.widthAt(i))))
	}
	return fitWidth(b.String(), g.Width)
}

func (g Grid) renderRow(rowIdx int) string {
	row := g.Rows[rowIdx]
	parts := make([]string, g.NumCols())
	cursorRow := g.Focused && rowIdx == g.CursorRow
	stripe := !g.NoStripe && rowIdx%2 == 1
	for i := 0; i < g.NumCols(); i++ {
		if g.colHidden(i) {
			continue
		}
		val := ""
		if i < len(row) {
			val = row[i]
		}
		text := g.padCellAt(val, i)
		switch {
		case cursorRow && i == g.CursorCol && !g.skipCursorPaint(i):
			parts[i] = styleCursorCell.Render(text)
		default:
			if g.CellStyle != nil {
				if styled, ok := g.CellStyle(rowIdx, i, text); ok {
					// Re-fit: per-glyph ANSI styling can desync lipgloss width
					// from the column, which shifts the row and lets the
					// detail pane wash bleed into the main grid.
					parts[i] = fitWidth(styled, g.widthAt(i))
					continue
				}
			}
			switch {
			case g.SoftCursor && cursorRow:
				parts[i] = styleRowFocus.Render(text)
			case g.muteCol(i):
				parts[i] = styleMuted.Render(text)
			case stripe && !g.skipStripe(i):
				parts[i] = styleStripe.Render(text)
			default:
				// Always set theme fg — bare text inherits the terminal default,
				// which is unreadable once paintBg applies a dark theme bg.
				parts[i] = styleCell.Render(text)
			}
		}
	}
	sep := g.colSep()
	switch {
	case cursorRow && g.SoftCursor:
		sep = styleGridBorder.Background(colorRowFocusBg).Render("│")
	case stripe:
		sep = styleGridBorder.Background(colorStripe).Render("│")
	}
	return fitWidth(g.joinVisibleCols(parts, sep), g.Width)
}

// padCellAt pads a cell, applying HScroll for HScrollCol when set.
func (g Grid) padCellAt(s string, col int) string {
	right := g.rightAlign(col)
	if g.HScroll > 0 && col == g.HScrollCol {
		rest := displaySkip(s, g.HScroll)
		inner := g.widthAt(col) - 2*cellPad
		if inner < 1 {
			return padCellAlign(rest, g.widthAt(col), right)
		}
		// Leading ellipsis marks that content continues to the left.
		marker := "…"
		mw := runewidth.StringWidth(marker)
		bodyW := inner - mw
		if bodyW < 1 {
			return padCellAlign(rest, g.widthAt(col), right)
		}
		body := runewidth.Truncate(rest, bodyW, "…")
		pad := strings.Repeat(" ", cellPad)
		return pad + runewidth.FillRight(marker+body, inner) + pad
	}
	return padCellAlign(s, g.widthAt(col), right)
}

func (g Grid) rightAlign(col int) bool {
	for _, c := range g.RightAlignCols {
		if c == col {
			return true
		}
	}
	return false
}

// renderEmptyRow fills a viewport slot past the last data row. When striping
// is enabled, zebra continues to the bottom of the pane.
func (g Grid) renderEmptyRow(rowIdx int) string {
	stripe := !g.NoStripe && rowIdx%2 == 1
	parts := make([]string, g.NumCols())
	for i := 0; i < g.NumCols(); i++ {
		if g.colHidden(i) {
			continue
		}
		blank := strings.Repeat(" ", g.widthAt(i))
		if stripe && !g.skipStripe(i) {
			parts[i] = styleStripe.Render(blank)
		} else {
			parts[i] = blank
		}
	}
	sep := g.colSep()
	if stripe {
		sep = styleGridBorder.Background(colorStripe).Render("│")
	}
	return fitWidth(g.joinVisibleCols(parts, sep), g.Width)
}

func (g Grid) skipStripe(col int) bool {
	for _, c := range g.SkipStripeCols {
		if c == col {
			return true
		}
	}
	return false
}

func (g Grid) skipCursorPaint(col int) bool {
	for _, c := range g.SkipCursorPaintCols {
		if c == col {
			return true
		}
	}
	return false
}

func (g Grid) muteCol(col int) bool {
	for _, c := range g.MuteCols {
		if c == col {
			return true
		}
	}
	return false
}

func (g Grid) colSep() string {
	return styleGridBorder.Render("│")
}

func (g Grid) widthAt(i int) int {
	if i >= 0 && i < len(g.Widths) {
		return g.Widths[i]
	}
	return 8 + 2*cellPad
}

func joinCols(parts []string, sep string) string {
	return strings.Join(parts, sep)
}

func padTrim(s string, width int) string {
	if width <= 0 {
		return ""
	}
	return runewidth.FillRight(runewidth.Truncate(s, width, "…"), width)
}

func padTrimRight(s string, width int) string {
	if width <= 0 {
		return ""
	}
	return runewidth.FillLeft(runewidth.Truncate(s, width, "…"), width)
}

// padCell truncates content to fit inside totalWidth with cellPad spaces on each side.
func padCell(s string, totalWidth int) string {
	return padCellAlign(s, totalWidth, false)
}

func padCellAlign(s string, totalWidth int, right bool) string {
	inner := totalWidth - 2*cellPad
	if inner < 1 {
		inner = 1
		if totalWidth < 1 {
			return ""
		}
	}
	pad := strings.Repeat(" ", cellPad)
	if right {
		return pad + padTrimRight(s, inner) + pad
	}
	return pad + padTrim(s, inner) + pad
}
