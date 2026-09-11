package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsiota/ore/internal/git"
)

type relRowKind int

const (
	relSection relRowKind = iota
	relCommit
	relFile
	relAuthor
	relMeta
)

type relRow struct {
	kind       relRowKind
	depth      int
	label      string
	hash       string
	path       string
	selectable bool
}

// RelExplorer is a docked relationship browser (creel-style g r).
type RelExplorer struct {
	open   bool
	title  string
	rows   []relRow
	cursor int
	offset int
	width  int
	height int
}

func (e *RelExplorer) Open()  { e.open = true }
func (e *RelExplorer) Close() { e.open = false; e.rows = nil; e.cursor = 0; e.offset = 0 }
func (e RelExplorer) Opened() bool { return e.open }

func (e *RelExplorer) SetSize(w, h int) {
	e.width = w
	e.height = h
}

func (e *RelExplorer) LoadCommit(rel git.CommitRelations) {
	var rows []relRow
	short := rel.Hash
	if len(short) > 7 {
		short = short[:7]
	}
	rows = append(rows, relRow{
		kind: relMeta, label: fmt.Sprintf("%s  %s", short, rel.Subject), selectable: false,
	})
	rows = append(rows, relRow{kind: relAuthor, label: fmt.Sprintf("%s <%s>", rel.Author, rel.Email), depth: 0, selectable: false})
	rows = append(rows, relRow{})

	rows = append(rows, relRow{kind: relSection, label: fmt.Sprintf("Parents (%d)", len(rel.Parents))})
	if len(rel.Parents) == 0 {
		rows = append(rows, relRow{kind: relMeta, depth: 1, label: "(root)"})
	}
	for _, p := range rel.Parents {
		rows = append(rows, relRow{
			kind: relCommit, depth: 1, selectable: true,
			hash: p.Hash, label: formatRelCommit(p),
		})
	}

	rows = append(rows, relRow{kind: relSection, label: fmt.Sprintf("Children (%d)", len(rel.Children))})
	if len(rel.Children) == 0 {
		rows = append(rows, relRow{kind: relMeta, depth: 1, label: "(none)"})
	}
	for _, c := range rel.Children {
		rows = append(rows, relRow{
			kind: relCommit, depth: 1, selectable: true,
			hash: c.Hash, label: formatRelCommit(c),
		})
	}

	rows = append(rows, relRow{kind: relSection, label: fmt.Sprintf("Files (%d)", len(rel.Files))})
	if len(rel.Files) == 0 {
		rows = append(rows, relRow{kind: relMeta, depth: 1, label: "(none)"})
	}
	for _, f := range rel.Files {
		path := f.Path
		if f.OldPath != "" {
			path = f.OldPath + " → " + f.Path
		}
		rows = append(rows, relRow{
			kind: relFile, depth: 1, selectable: true,
			path: f.Path, hash: rel.Hash, label: path,
		})
	}

	e.title = " relationships"
	e.rows = rows
	e.cursor = firstSelectable(rows)
	e.offset = 0
	e.open = true
}

func (e *RelExplorer) LoadLine(rel git.LineRelations) {
	var rows []relRow
	bl := rel.Line
	rows = append(rows, relRow{
		kind: relMeta,
		label: fmt.Sprintf("line %d · %s  %s", bl.Line, bl.ShortHash, bl.Summary),
	})
	rows = append(rows, relRow{kind: relMeta, depth: 1, label: truncateRunes(bl.Text, 60)})
	rows = append(rows, relRow{})

	rows = append(rows, relRow{kind: relSection, label: "This commit"})
	rows = append(rows, relRow{
		kind: relCommit, depth: 1, selectable: true,
		hash: bl.Hash, label: fmt.Sprintf("%s  %s", bl.ShortHash, bl.Summary),
	})

	rows = append(rows, relRow{kind: relSection, label: "Previous"})
	if rel.Previous != nil {
		rows = append(rows, relRow{
			kind: relCommit, depth: 1, selectable: true,
			hash: rel.Previous.Hash, label: formatRelCommit(*rel.Previous),
		})
	} else {
		rows = append(rows, relRow{kind: relMeta, depth: 1, label: "(none)"})
	}

	rows = append(rows, relRow{kind: relSection, label: fmt.Sprintf("File history (%d)", len(rel.History))})
	for _, c := range rel.History {
		rows = append(rows, relRow{
			kind: relCommit, depth: 1, selectable: true,
			hash: c.Hash, path: rel.Path, label: formatRelCommit(c),
		})
	}

	rows = append(rows, relRow{kind: relSection, label: "File"})
	rows = append(rows, relRow{
		kind: relFile, depth: 1, selectable: true,
		path: rel.Path, hash: rel.Rev, label: rel.Path,
	})

	e.title = " line relationships"
	e.rows = rows
	e.cursor = firstSelectable(rows)
	e.offset = 0
	e.open = true
}

func formatRelCommit(c git.Commit) string {
	return fmt.Sprintf("%s  %s", c.ShortHash, c.Subject)
}

func firstSelectable(rows []relRow) int {
	for i, r := range rows {
		if r.selectable {
			return i
		}
	}
	return 0
}

func (e *RelExplorer) Selected() (relRow, bool) {
	if e.cursor < 0 || e.cursor >= len(e.rows) {
		return relRow{}, false
	}
	r := e.rows[e.cursor]
	if !r.selectable {
		return relRow{}, false
	}
	return r, true
}

func (e *RelExplorer) move(delta int) {
	if len(e.rows) == 0 {
		return
	}
	for i := 0; i < len(e.rows); i++ {
		e.cursor += delta
		if e.cursor < 0 {
			e.cursor = len(e.rows) - 1
		}
		if e.cursor >= len(e.rows) {
			e.cursor = 0
		}
		if e.rows[e.cursor].selectable || !hasSelectable(e.rows) {
			break
		}
	}
	e.ensureVisible()
}

func hasSelectable(rows []relRow) bool {
	for _, r := range rows {
		if r.selectable {
			return true
		}
	}
	return false
}

func (e *RelExplorer) ensureVisible() {
	h := max(1, e.height)
	if e.cursor < e.offset {
		e.offset = e.cursor
	}
	if e.cursor >= e.offset+h {
		e.offset = e.cursor - h + 1
	}
}

// Update handles keys while the explorer is focused. Returns true if consumed.
func (e *RelExplorer) Update(msg tea.KeyMsg) (consumed bool, activate bool) {
	if !e.open {
		return false, false
	}
	switch msg.String() {
	case "j", "down":
		e.move(1)
		return true, false
	case "k", "up":
		e.move(-1)
		return true, false
	case "g", "home":
		e.cursor = firstSelectable(e.rows)
		e.ensureVisible()
		return true, false
	case "G", "end":
		for i := len(e.rows) - 1; i >= 0; i-- {
			if e.rows[i].selectable {
				e.cursor = i
				break
			}
		}
		e.ensureVisible()
		return true, false
	case "ctrl+d":
		e.move(max(1, e.height/2))
		return true, false
	case "ctrl+u":
		e.move(-max(1, e.height/2))
		return true, false
	case "enter", "l":
		return true, true
	case "esc", "h", "q":
		e.Close()
		return true, false
	}
	return false, false
}

func (e RelExplorer) View(focused bool) string {
	if !e.open || e.width <= 0 || e.height <= 0 {
		return ""
	}
	var lines []string
	h := e.height
	if h < 1 {
		h = 1
	}
	end := min(len(e.rows), e.offset+h)
	for i := e.offset; i < end; i++ {
		r := e.rows[i]
		prefix := strings.Repeat("  ", r.depth)
		text := prefix + r.label
		switch {
		case i == e.cursor && r.selectable:
			lines = append(lines, cell(styleFocus, text, e.width))
		case r.kind == relSection:
			lines = append(lines, cell(styleHeader, text, e.width))
		case !r.selectable:
			lines = append(lines, fitWidth(styleMuted.Render(text), e.width))
		default:
			lines = append(lines, fitWidth(text, e.width))
		}
	}
	return padPane(lines, e.width, e.height)
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
