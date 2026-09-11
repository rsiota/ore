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
	n := g.NumCols()
	if n == 0 {
		g.Widths = nil
		return
	}
	pad := 2 * cellPad
	g.Widths = make([]int, n)
	for i, col := range g.Columns {
		g.Widths[i] = runewidth.StringWidth(col) + 1 + pad // room for sort arrow + pad
	}
	for _, row := range g.Rows {
		for i := 0; i < n && i < len(row); i++ {
			w := runewidth.StringWidth(row[i]) + pad
			if w > g.Widths[i] {
				g.Widths[i] = w
			}
		}
	}
	// Cap subject-like trailing column flex; cap others.
	minCell := 4 + pad
	for i := 0; i < n-1; i++ {
		if g.Widths[i] > 16+pad {
			g.Widths[i] = 16 + pad
		}
		if g.Widths[i] < minCell {
			g.Widths[i] = minCell
		}
	}
	// Fit into pane: give leftover to last column. Separators cost 1 col each.
	used := n - 1 // │ between columns
	for i := 0; i < n-1; i++ {
		used += g.Widths[i]
	}
	remain := g.Width - used
	if remain < 8+pad {
		remain = 8 + pad
	}
	g.Widths[n-1] = remain
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
	// Extend zebra through unused viewport slots (creel padding rows).
	for slot := filled; slot < h; slot++ {
		lines = append(lines, g.renderEmptyRow(g.OffsetRow+slot))
	}
	return padPane(lines, g.Width, g.Height)
}

func (g Grid) renderHeader() string {
	parts := make([]string, g.NumCols())
	for i, name := range g.Columns {
		label := name
		if i == g.SortCol {
			if a := g.SortDir.Arrow(); a != "" {
				label = name + a
			}
		}
		parts[i] = renderHeaderCell(label, g.widthAt(i), g.Focused && i == g.CursorCol)
	}
	return fitWidth(joinCols(parts, g.colSep()), g.Width)
}

// renderHeaderCell draws a creel-style header: blue text, with underline
// scoped to the word when selected (padding stays un-underlined).
func renderHeaderCell(label string, totalWidth int, selected bool) string {
	inner := totalWidth - 2*cellPad
	if inner < 1 {
		inner = 1
	}
	content := padTrim(label, inner)
	pad := strings.Repeat(" ", cellPad)
	if !selected {
		return styleGridHeader.Render(pad + content + pad)
	}
	text := strings.TrimRight(content, " ")
	trail := inner - runewidth.StringWidth(text)
	var b strings.Builder
	b.WriteString(styleGridHeader.Render(pad))
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
	for i := 0; i < g.NumCols(); i++ {
		if i > 0 {
			b.WriteString(sep)
		}
		b.WriteString(styleGridBorder.Render(strings.Repeat("─", g.widthAt(i))))
	}
	return fitWidth(b.String(), g.Width)
}

func (g Grid) renderRow(rowIdx int) string {
	row := g.Rows[rowIdx]
	parts := make([]string, g.NumCols())
	cursorCell := g.Focused && rowIdx == g.CursorRow
	stripe := rowIdx%2 == 1
	for i := 0; i < g.NumCols(); i++ {
		val := ""
		if i < len(row) {
			val = row[i]
		}
		text := padCell(val, g.widthAt(i))
		switch {
		case cursorCell && i == g.CursorCol:
			parts[i] = styleCursorCell.Render(text)
		case stripe:
			parts[i] = styleStripe.Render(text)
		default:
			parts[i] = text
		}
	}
	sep := g.colSep()
	if stripe {
		sep = styleGridBorder.Background(colorStripe).Render("│")
	}
	return fitWidth(joinCols(parts, sep), g.Width)
}

// renderEmptyRow fills a viewport slot past the last data row with zebra
// chrome so the stripe continues to the bottom of the pane.
func (g Grid) renderEmptyRow(rowIdx int) string {
	stripe := rowIdx%2 == 1
	parts := make([]string, g.NumCols())
	for i := 0; i < g.NumCols(); i++ {
		blank := strings.Repeat(" ", g.widthAt(i))
		if stripe {
			parts[i] = styleStripe.Render(blank)
		} else {
			parts[i] = blank
		}
	}
	sep := g.colSep()
	if stripe {
		sep = styleGridBorder.Background(colorStripe).Render("│")
	}
	return fitWidth(joinCols(parts, sep), g.Width)
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

// padCell truncates content to fit inside totalWidth with cellPad spaces on each side.
func padCell(s string, totalWidth int) string {
	inner := totalWidth - 2*cellPad
	if inner < 1 {
		inner = 1
		if totalWidth < 1 {
			return ""
		}
	}
	pad := strings.Repeat(" ", cellPad)
	return pad + padTrim(s, inner) + pad
}
