package ui

import (
	"fmt"
	"sort"
	"strings"

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
		return relativeAge(b.When)
	case blameColAuthor:
		return b.Author
	case blameColCode:
		return strings.ReplaceAll(b.Text, "\t", "    ")
	default:
		return ""
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
