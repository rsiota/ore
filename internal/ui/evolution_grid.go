package ui

import (
	"fmt"
	"strings"

	"github.com/rsiota/ore/internal/git"
)

// Evolution grid columns.
const (
	evoColStep = iota
	evoColCommit
	evoColAge
	evoColAuthor
	evoColLine
	evoColPath
	evoColCode
	evoColCount
)

var evoColumns = []string{"#", "commit", "age", "author", "line", "path", "code"}

var evoMetaMaxContent = []int{
	evoColStep:   2,
	evoColCommit: 7,
	evoColAge:    3,
	evoColAuthor: 10,
	evoColLine:   5,
	evoColPath:   24,
}

func evoCell(s git.LineEvolutionStep, col int) string {
	switch col {
	case evoColStep:
		return fmt.Sprintf("%d", s.Index)
	case evoColCommit:
		return s.Line.ShortHash
	case evoColAge:
		return compactAge(s.Line.When)
	case evoColAuthor:
		return s.Line.Author
	case evoColLine:
		return fmt.Sprintf("%d", s.Line.Line)
	case evoColPath:
		return s.Path
	case evoColCode:
		return strings.ReplaceAll(s.Line.Text, "\t", "    ")
	default:
		return ""
	}
}

func evoRow(s git.LineEvolutionStep) []string {
	row := make([]string, evoColCount)
	for c := 0; c < evoColCount; c++ {
		row[c] = evoCell(s, c)
	}
	return row
}

func (m Model) renderEvolutionPane(width, height int) string {
	rows := make([][]string, len(m.evo))
	for i, s := range m.evo {
		rows[i] = evoRow(s)
	}
	g := Grid{
		Columns:    evoColumns,
		Rows:       rows,
		CursorRow:  m.evoCursor,
		CursorCol:  m.evoCol,
		OffsetRow:  m.evoOffset,
		Width:      width,
		Height:     height,
		Focused:    m.focus == FocusMain,
		SoftCursor: true,
		MuteCols:   []int{evoColStep, evoColPath},
		RightAlignCols: []int{evoColStep, evoColLine},
	}
	g.AutoWidthsCaps(evoMetaMaxContent)
	g.ClampCursor()
	return g.View()
}

func (m *Model) ensureEvoVisible() {
	h := max(1, m.paneContentHeight()-1)
	if m.evoCursor < m.evoOffset {
		m.evoOffset = m.evoCursor
	}
	if m.evoCursor >= m.evoOffset+h {
		m.evoOffset = m.evoCursor - h + 1
	}
	if m.evoOffset < 0 {
		m.evoOffset = 0
	}
}
