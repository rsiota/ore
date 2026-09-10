package ui

import (
	"sort"
	"strings"

	"github.com/rsiota/ore/internal/git"
)

// Commit grid column indexes.
const (
	commitColHash = iota
	commitColDate
	commitColAuthor
	commitColSubject
	commitColCount
)

var commitColumns = []string{"HASH", "DATE", "AUTHOR", "SUBJECT"}

func commitCell(c git.Commit, col int) string {
	switch col {
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

func commitRow(c git.Commit) []string {
	return []string{
		commitCell(c, commitColHash),
		commitCell(c, commitColDate),
		commitCell(c, commitColAuthor),
		commitCell(c, commitColSubject),
	}
}

func filterCommitIndicesCol(commits []git.Commit, query string, col int) []int {
	q := strings.TrimSpace(query)
	if q == "" {
		return identityIndices(len(commits))
	}
	out := make([]int, 0, len(commits))
	for i, c := range commits {
		var ok bool
		if col < 0 || col >= commitColCount {
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
	if dir == SortNone || col < 0 || col >= commitColCount || len(idx) < 2 {
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
