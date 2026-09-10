package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

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
	Title   string
	Columns []string
	Rows    [][]string
	Widths  []int // per-column content widths (without padding)

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
	return max(1, g.Height-2) // title + header
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
	g.Widths = make([]int, n)
	for i, col := range g.Columns {
		g.Widths[i] = runewidth.StringWidth(col) + 1 // room for sort arrow
	}
	for _, row := range g.Rows {
		for i := 0; i < n && i < len(row); i++ {
			w := runewidth.StringWidth(row[i])
			if w > g.Widths[i] {
				g.Widths[i] = w
			}
		}
	}
	// Cap subject-like trailing column flex; cap others.
	for i := 0; i < n-1; i++ {
		if g.Widths[i] > 16 {
			g.Widths[i] = 16
		}
		if g.Widths[i] < 4 {
			g.Widths[i] = 4
		}
	}
	// Fit into pane: give leftover to last column.
	used := 0
	for i := 0; i < n-1; i++ {
		used += g.Widths[i] + 1 // gap
	}
	remain := g.Width - used
	if remain < 8 {
		remain = 8
	}
	g.Widths[n-1] = remain
}

// View renders the grid into Width x Height cells.
func (g Grid) View() string {
	if g.Width <= 0 || g.Height <= 0 {
		return ""
	}
	var lines []string
	title := g.Title
	if title == "" {
		title = " grid"
	}
	if g.Focused {
		lines = append(lines, cell(styleFocus, title, g.Width))
	} else {
		lines = append(lines, cell(styleMuted, title, g.Width))
	}

	if g.NumCols() == 0 {
		return padPane(lines, g.Width, g.Height)
	}

	lines = append(lines, g.renderHeader())

	h := g.listHeight()
	end := min(g.NumRows(), g.OffsetRow+h)
	for i := g.OffsetRow; i < end; i++ {
		lines = append(lines, g.renderRow(i))
	}
	if g.NumRows() == 0 {
		lines = append(lines, fitWidth(styleMuted.Render("(no rows)"), g.Width))
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
		style := styleHeader
		if g.Focused && i == g.CursorCol {
			style = styleHeader.Underline(true).Foreground(lipgloss.Color("#0969da"))
		}
		parts[i] = style.Render(padTrim(label, g.widthAt(i)))
	}
	return fitWidth(joinCols(parts, g.Widths), g.Width)
}

func (g Grid) renderRow(rowIdx int) string {
	row := g.Rows[rowIdx]
	parts := make([]string, g.NumCols())
	for i := 0; i < g.NumCols(); i++ {
		val := ""
		if i < len(row) {
			val = row[i]
		}
		w := g.widthAt(i)
		text := padTrim(val, w)
		switch {
		case g.Focused && rowIdx == g.CursorRow && i == g.CursorCol:
			parts[i] = styleFocus.Render(text)
		case g.Focused && rowIdx == g.CursorRow:
			parts[i] = styleRowFocus.Render(text)
		default:
			parts[i] = text
		}
	}
	return fitWidth(joinCols(parts, g.Widths), g.Width)
}

func (g Grid) widthAt(i int) int {
	if i >= 0 && i < len(g.Widths) {
		return g.Widths[i]
	}
	return 8
}

func joinCols(parts []string, widths []int) string {
	var b strings.Builder
	for i, p := range parts {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(p)
		_ = widths
	}
	return b.String()
}

func padTrim(s string, width int) string {
	if width <= 0 {
		return ""
	}
	return runewidth.FillRight(runewidth.Truncate(s, width, "…"), width)
}
