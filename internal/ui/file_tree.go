package ui

import (
	"strings"

	"github.com/rsiota/ore/internal/git"
)

type treeRow struct {
	name     string
	path     string
	dir      bool
	depth    int
	expanded bool
}

// fileTree is a lazy directory browser for one revision.
type fileTree struct {
	rev          string
	kids         map[string][]git.TreeEntry // directory path → children; "" = root
	expanded     map[string]bool
	cursor       int
	offset       int
	filter       string
	filterTyping bool
	matches      []string
	filesLoaded  bool
	preferPath   string
}

func newFileTree(rev, prefer string) fileTree {
	return fileTree{
		rev:        rev,
		kids:       make(map[string][]git.TreeEntry),
		expanded:   make(map[string]bool),
		preferPath: prefer,
	}
}

func (t *fileTree) reset(rev, prefer string) {
	*t = newFileTree(rev, prefer)
}

func (t *fileTree) rows() []treeRow {
	if strings.TrimSpace(t.filter) != "" {
		return t.filterRows()
	}
	if t.kids == nil {
		return nil
	}
	var out []treeRow
	t.walk("", 0, &out)
	return out
}

func (t *fileTree) walk(prefix string, depth int, out *[]treeRow) {
	for _, e := range t.kids[prefix] {
		row := treeRow{
			name:     e.Name,
			path:     e.Path,
			dir:      e.Dir,
			depth:    depth,
			expanded: e.Dir && t.expanded[e.Path],
		}
		*out = append(*out, row)
		if row.expanded {
			t.walk(e.Path, depth+1, out)
		}
	}
}

func (t *fileTree) filterRows() []treeRow {
	out := make([]treeRow, 0, len(t.matches))
	for _, path := range t.matches {
		if !filterMatch(t.filter, path, git.TreeBase(path)) {
			continue
		}
		out = append(out, treeRow{
			name: git.TreeBase(path),
			path: path,
		})
		if len(out) >= maxTreeFilterMatches {
			break
		}
	}
	return out
}

func (t *fileTree) selected() (treeRow, bool) {
	rows := t.rows()
	if t.cursor < 0 || t.cursor >= len(rows) {
		return treeRow{}, false
	}
	return rows[t.cursor], true
}

func (t *fileTree) selectedPath() string {
	if row, ok := t.selected(); ok {
		return row.path
	}
	return t.preferPath
}

func (t *fileTree) applyKids(prefix string, kids []git.TreeEntry) {
	if t.kids == nil {
		t.kids = make(map[string][]git.TreeEntry)
	}
	t.kids[prefix] = append([]git.TreeEntry(nil), kids...)
	t.clampCursor()
}

func (t *fileTree) applyMatches(files []string) {
	t.matches = append([]string(nil), files...)
	t.filesLoaded = true
	t.clampCursor()
}

func (t *fileTree) clampCursor() {
	n := len(t.rows())
	if n == 0 {
		t.cursor = 0
		t.offset = 0
		return
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
	if t.cursor >= n {
		t.cursor = n - 1
	}
}

func (t *fileTree) move(delta int) {
	n := len(t.rows())
	if n == 0 {
		return
	}
	t.cursor = clamp(t.cursor+delta, 0, n-1)
}

func (t *fileTree) gotoTop() {
	t.cursor = 0
}

func (t *fileTree) gotoBottom() {
	n := len(t.rows())
	if n == 0 {
		return
	}
	t.cursor = n - 1
}

func (t *fileTree) ensureVisible(h int) {
	if h < 1 {
		h = 1
	}
	n := len(t.rows())
	if n == 0 {
		t.offset = 0
		return
	}
	if t.cursor < t.offset {
		t.offset = t.cursor
	}
	if t.cursor >= t.offset+h {
		t.offset = t.cursor - h + 1
	}
	if t.offset < 0 {
		t.offset = 0
	}
}

func (t *fileTree) toggleExpand() (loadPrefix string, loaded bool) {
	row, ok := t.selected()
	if !ok || !row.dir {
		return "", false
	}
	if t.expanded[row.path] {
		delete(t.expanded, row.path)
		t.clampCursor()
		return "", true
	}
	if _, have := t.kids[row.path]; have {
		t.expanded[row.path] = true
		return "", true
	}
	t.expanded[row.path] = true
	return row.path, true
}

func (t *fileTree) collapseOrParent() (moved bool) {
	row, ok := t.selected()
	if !ok {
		return false
	}
	if row.dir && t.expanded[row.path] {
		delete(t.expanded, row.path)
		return true
	}
	parent := git.TreeParent(row.path)
	if parent == "" {
		return false
	}
	return t.selectPath(parent)
}

func (t *fileTree) selectPath(path string) bool {
	if path == "" {
		return false
	}
	for _, dir := range git.TreeAncestors(path) {
		if dir == "" {
			continue
		}
		if _, have := t.kids[dir]; have {
			t.expanded[dir] = true
		}
	}
	rows := t.rows()
	for i, row := range rows {
		if row.path == path {
			t.cursor = i
			return true
		}
	}
	for p := git.TreeParent(path); p != ""; p = git.TreeParent(p) {
		for i, row := range rows {
			if row.path == p {
				t.cursor = i
				return true
			}
		}
	}
	return false
}

func (t *fileTree) missingPrefixes(path string) []string {
	var missing []string
	if _, ok := t.kids[""]; !ok {
		missing = append(missing, "")
	}
	if path == "" {
		return missing
	}
	for _, dir := range git.TreeAncestors(path) {
		if dir == "" {
			continue
		}
		if _, ok := t.kids[dir]; !ok {
			missing = append(missing, dir)
		}
	}
	return missing
}
