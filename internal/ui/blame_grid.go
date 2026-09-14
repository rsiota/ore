package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rsiota/ore/internal/git"
)

// Blame grid column indexes.
const (
	blameColLine = iota
	blameColCommit
	blameColAge
	blameColAuthor
	blameColCode
	blameColCount
)

var blameColumns = []string{"line", "commit", "age", "author", "code"}

func blameCell(b git.BlameLine, col int) string {
	switch col {
	case blameColLine:
		return fmt.Sprintf("%d", b.Line)
	case blameColCommit:
		return b.ShortHash
	case blameColAge:
		return compactAge(b.When)
	case blameColAuthor:
		return b.Author
	case blameColCode:
		return strings.ReplaceAll(b.Text, "\t", "    ")
	default:
		return ""
	}
}

// blameMetaMaxContent is the max inner width (no cell pad) for each meta
// column. Code stays flexible so the file dominates the pane.
var blameMetaMaxContent = []int{
	blameColLine:   4,  // up to 9999
	blameColCommit: 7,  // short hash
	blameColAge:    3,  // now / 2h / 3d / 2mo / 1y
	blameColAuthor: 10, // truncate long names
}

// compactAge is a short blame-gutter age (keeps meta columns narrow).
func compactAge(t time.Time) string {
	if t.IsZero() {
		return "?"
	}
	d := time.Since(t)
	switch {
	case d < time.Hour:
		return "now"
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo", int(d.Hours()/24/30))
	default:
		return fmt.Sprintf("%dy", int(d.Hours()/24/365))
	}
}

func blameRow(b git.BlameLine) []string {
	return []string{
		blameCell(b, blameColLine),
		blameCell(b, blameColCommit),
		blameCell(b, blameColAge),
		blameCell(b, blameColAuthor),
		blameCell(b, blameColCode),
	}
}

// blameRowDisplay builds a row, blanking commit/age/author when the previous
// visible line belongs to the same commit (quieter gutter, still sortable).
func blameRowDisplay(lines []git.BlameLine, idx []int, row int) []string {
	if row < 0 || row >= len(idx) {
		return nil
	}
	out := blameRow(lines[idx[row]])
	if row == 0 {
		return out
	}
	prev := lines[idx[row-1]]
	cur := lines[idx[row]]
	if prev.Hash != "" && prev.Hash == cur.Hash {
		out[blameColCommit] = ""
		out[blameColAge] = ""
		out[blameColAuthor] = ""
	}
	return out
}

func filterBlameIndicesCol(lines []git.BlameLine, query string, col int) []int {
	q := strings.TrimSpace(query)
	if q == "" {
		return identityIndices(len(lines))
	}
	out := make([]int, 0, len(lines))
	for i, l := range lines {
		var ok bool
		if col < 0 || col >= blameColCount {
			ok = filterMatch(q, l.Text, l.Author, l.Summary, l.ShortHash, l.Hash, blameCell(l, blameColAge), blameCell(l, blameColLine))
		} else {
			ok = filterMatch(q, blameCell(l, col))
			if col == blameColCommit {
				ok = ok || filterMatch(q, l.Hash)
			}
			if col == blameColCode {
				ok = ok || filterMatch(q, l.Summary)
			}
		}
		if ok {
			out = append(out, i)
		}
	}
	return out
}

func sortBlameIndices(lines []git.BlameLine, idx []int, col int, dir SortDir) []int {
	if dir == SortNone || col < 0 || col >= blameColCount || len(idx) < 2 {
		return idx
	}
	out := append([]int(nil), idx...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := lines[out[i]], lines[out[j]]
		cmp := 0
		switch col {
		case blameColLine:
			cmp = a.Line - b.Line
		case blameColCommit:
			cmp = strings.Compare(a.Hash, b.Hash)
		case blameColAge:
			switch {
			case a.When.Before(b.When):
				cmp = -1
			case a.When.After(b.When):
				cmp = 1
			}
		case blameColAuthor:
			cmp = strings.Compare(a.Author, b.Author)
		case blameColCode:
			cmp = strings.Compare(a.Text, b.Text)
		}
		if dir == SortDesc {
			cmp = -cmp
		}
		return cmp < 0
	})
	return out
}
