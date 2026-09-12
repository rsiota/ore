package ui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rsiota/ore/internal/git"
)

// Commit grid column indexes.
const (
	commitColGraph = iota
	commitColHash
	commitColDate
	commitColAuthor
	commitColSubject
	commitColCount
)

var commitColumns = []string{"graph", "hash", "date", "author", "subject"}

func commitCell(c git.Commit, col int) string {
	switch col {
	case commitColGraph:
		return ""
	case commitColHash:
		return c.ShortHash
	case commitColDate:
		if c.Date.IsZero() {
			return ""
		}
		return c.Date.Local().Format("2006-01-02")
	case commitColAuthor:
		return c.Author
	case commitColSubject:
		return c.Subject
	default:
		return ""
	}
}

func commitRow(c git.Commit, graph string) []string {
	return []string{
		graph,
		commitCell(c, commitColHash),
		commitCell(c, commitColDate),
		commitCell(c, commitColAuthor),
		commitCell(c, commitColSubject),
	}
}

// commitGraphLines builds soft-graph glyphs for idx. Blank when sorted so
// lanes are not drawn against a non-history order.
func commitGraphLines(commits []git.Commit, idx []int, sortDir SortDir) []string {
	if sortDir != SortNone {
		out := make([]string, len(idx))
		return out
	}
	return buildSoftGraph(commits, idx)
}

func styleCommitGraphCell(row, col int, text string) (string, bool) {
	if col != commitColGraph {
		return "", false
	}
	// Match zebra cell backgrounds (forced white was showing as a mismatch).
	stripe := row%2 == 1
	if strings.TrimSpace(text) == "" {
		if stripe {
			return styleStripe.Render(text), true
		}
		return text, true
	}
	runes := []rune(text)
	var b strings.Builder
	b.Grow(len(runes) * 12)
	for _, r := range runes {
		st := lipgloss.NewStyle()
		if stripe {
			st = st.Background(colorStripe)
		}
		if r == ' ' {
			if stripe {
				b.WriteString(st.Render(" "))
			} else {
				b.WriteByte(' ')
			}
			continue
		}
		if r == '●' {
			st = st.Foreground(graphNodeColor).Bold(true)
		} else {
			st = st.Foreground(graphLineColor)
		}
		b.WriteString(st.Render(string(r)))
	}
	return b.String(), true
}

func filterCommitIndicesCol(commits []git.Commit, query string, col int) []int {
	q := strings.TrimSpace(query)
	if q == "" {
		return identityIndices(len(commits))
	}
	out := make([]int, 0, len(commits))
	for i, c := range commits {
		var ok bool
		if col < 0 || col >= commitColCount || col == commitColGraph {
			ok = filterMatch(q, c.Subject, c.Author, c.ShortHash, c.Hash, c.Email, commitCell(c, commitColDate))
		} else {
			ok = filterMatch(q, commitCell(c, col))
			if col == commitColHash {
				ok = ok || filterMatch(q, c.Hash)
			}
		}
		if ok {
			out = append(out, i)
		}
	}
	return out
}

func sortCommitIndices(commits []git.Commit, idx []int, col int, dir SortDir) []int {
	if dir == SortNone || col < 0 || col >= commitColCount || col == commitColGraph || len(idx) < 2 {
		return idx
	}
	out := append([]int(nil), idx...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := commits[out[i]], commits[out[j]]
		cmp := 0
		switch col {
		case commitColHash:
			cmp = strings.Compare(a.Hash, b.Hash)
		case commitColDate:
			switch {
			case a.Date.Before(b.Date):
				cmp = -1
			case a.Date.After(b.Date):
				cmp = 1
			}
		case commitColAuthor:
			cmp = strings.Compare(a.Author, b.Author)
		case commitColSubject:
			cmp = strings.Compare(a.Subject, b.Subject)
		}
		if dir == SortDesc {
			cmp = -cmp
		}
		return cmp < 0
	})
	return out
}
