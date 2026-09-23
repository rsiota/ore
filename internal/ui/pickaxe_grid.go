package ui

import (
	"fmt"
	"strings"

	"github.com/rsiota/ore/internal/git"
)

// Pickaxe result grid columns.
const (
	pickColHash = iota
	pickColDate
	pickColAuthor
	pickColPath
	pickColSubject
	pickColCount
)

var pickColumns = []string{"hash", "date", "author", "path", "subject"}

var pickMetaMaxContent = []int{
	pickColHash:   7,
	pickColDate:   10,
	pickColAuthor: 12,
	pickColPath:   28,
}

func pickPathCell(h git.PickaxeHit) string {
	switch len(h.Paths) {
	case 0:
		return ""
	case 1:
		return h.Paths[0]
	default:
		return fmt.Sprintf("%s +%d", h.Paths[0], len(h.Paths)-1)
	}
}

func pickRow(h git.PickaxeHit) []string {
	c := h.Commit
	date := ""
	if !c.Date.IsZero() {
		date = c.Date.Local().Format("2006-01-02")
	}
	return []string{
		c.ShortHash,
		date,
		c.Author,
		pickPathCell(h),
		c.Subject,
	}
}

func (m Model) renderPickaxePane(width, height int) string {
	rows := make([][]string, len(m.pickaxe))
	for i, h := range m.pickaxe {
		rows[i] = pickRow(h)
	}
	g := Grid{
		Columns:        pickColumns,
		Rows:           rows,
		CursorRow:      m.pickCursor,
		CursorCol:      m.pickCol,
		OffsetRow:      m.pickOffset,
		Width:          width,
		Height:         height,
		Focused:        m.focus == FocusMain,
		SoftCursor:     true,
		MuteCols:       []int{pickColPath},
		RightAlignCols: nil,
	}
	g.AutoWidthsCaps(pickMetaMaxContent)
	g.ClampCursor()
	return g.View()
}

func (m *Model) ensurePickVisible() {
	h := max(1, m.paneContentHeight()-1)
	if m.pickCursor < m.pickOffset {
		m.pickOffset = m.pickCursor
	}
	if m.pickCursor >= m.pickOffset+h {
		m.pickOffset = m.pickCursor - h + 1
	}
	if m.pickOffset < 0 {
		m.pickOffset = 0
	}
}

func (m Model) selectedPickaxeHit() (git.PickaxeHit, bool) {
	if m.pickCursor < 0 || m.pickCursor >= len(m.pickaxe) {
		return git.PickaxeHit{}, false
	}
	return m.pickaxe[m.pickCursor], true
}

func pickaxeModeLabel(mode git.PickaxeMode) string {
	return mode.Label()
}

func truncateQuery(q string, max int) string {
	q = strings.TrimSpace(q)
	r := []rune(q)
	if len(r) <= max {
		return q
	}
	return string(r[:max-1]) + "…"
}

type hitListKind int

const (
	hitListPickaxe hitListKind = iota
	hitListCouple
	hitListAuthors
)

func formatAuthorLabel(author string, paths []string) string {
	if len(paths) == 0 {
		return author
	}
	if len(paths) == 1 {
		return author + " · " + paths[0]
	}
	return fmt.Sprintf("%s · %s +%d", author, paths[0], len(paths)-1)
}

func formatCoupleLabel(seeds []string, partner string) string {
	seed := ""
	if len(seeds) == 1 {
		seed = seeds[0]
	} else if len(seeds) > 1 {
		seed = seeds[0] + "…"
	}
	if seed == "" {
		return partner
	}
	return seed + " ∩ " + partner
}

func (m Model) hitListShortStatus() string {
	label := truncateQuery(m.pickQuery, 32)
	n := len(m.pickaxe)
	if m.pickKind == hitListCouple {
		return fmt.Sprintf("couple · %s · %d commits", label, n)
	}
	if m.pickKind == hitListAuthors {
		return fmt.Sprintf("authors · %s · %d commits", label, n)
	}
	s := fmt.Sprintf("pickaxe · %s %q · %d hits", pickaxeModeLabel(m.pickMode), label, n)
	if m.pickHlOn {
		s += " · hl · :nohl"
	}
	return s
}
