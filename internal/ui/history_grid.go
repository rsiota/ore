package ui

import (
	"sort"
	"strings"

	"github.com/rsiota/ore/internal/git"
)

// History grid column indexes (commit log + path-at-rev / rename edge).
const (
	histColGraph = iota
	histColHash
	histColDate
	histColAuthor
	histColPath
	histColSubject
	histColCount
)

var histColumns = []string{"graph", "hash", "date", "author", "path", "subject"}

func histCell(pc git.PathCommit, col int) string {
	switch col {
	case histColGraph:
		return ""
	case histColHash:
		return pc.ShortHash
	case histColDate:
		if pc.Date.IsZero() {
			return ""
		}
		return pc.Date.Local().Format("2006-01-02")
	case histColAuthor:
		return pc.Author
	case histColPath:
		return pc.PathLabel()
	case histColSubject:
		return pc.Subject
	default:
		return ""
	}
}

func histRow(pc git.PathCommit, graph string) []string {
	return []string{
		graph,
		histCell(pc, histColHash),
		histCell(pc, histColDate),
		histCell(pc, histColAuthor),
		histCell(pc, histColPath),
		histCell(pc, histColSubject),
	}
}

func filterHistIndicesCol(commits []git.PathCommit, query string, col int) []int {
	q := strings.TrimSpace(query)
	if q == "" {
		return identityIndices(len(commits))
	}
	out := make([]int, 0, len(commits))
	for i, c := range commits {
		var ok bool
		if col < 0 || col >= histColCount || col == histColGraph {
			ok = filterMatch(q, c.Subject, c.Author, c.ShortHash, c.Hash, c.Email,
				histCell(c, histColDate), c.Path, c.OldPath, c.PathLabel(), c.EdgeLabel())
		} else {
			ok = filterMatch(q, histCell(c, col))
			if col == histColHash {
				ok = ok || filterMatch(q, c.Hash)
			}
			if col == histColPath {
				ok = ok || filterMatch(q, c.Path, c.OldPath, c.EdgeLabel())
			}
		}
		if ok {
			out = append(out, i)
		}
	}
	return out
}

func sortHistIndices(commits []git.PathCommit, idx []int, col int, dir SortDir) []int {
	if dir == SortNone || col < 0 || col >= histColCount || col == histColGraph || len(idx) < 2 {
		return idx
	}
	out := append([]int(nil), idx...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := commits[out[i]], commits[out[j]]
		cmp := 0
		switch col {
		case histColHash:
			cmp = strings.Compare(a.Hash, b.Hash)
		case histColDate:
			switch {
			case a.Date.Before(b.Date):
				cmp = -1
			case a.Date.After(b.Date):
				cmp = 1
			}
		case histColAuthor:
			cmp = strings.Compare(a.Author, b.Author)
		case histColPath:
			cmp = strings.Compare(a.PathLabel(), b.PathLabel())
		case histColSubject:
			cmp = strings.Compare(a.Subject, b.Subject)
		}
		if dir == SortDesc {
			cmp = -cmp
		}
		return cmp < 0
	})
	return out
}

func histGraphLines(commits []git.PathCommit, idx []int, sortDir SortDir) []string {
	return commitGraphLines(git.PathCommitsAsCommits(commits), idx, sortDir)
}
