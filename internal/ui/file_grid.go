package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rsiota/ore/internal/git"
)

// File grid column indexes.
const (
	fileColStatus = iota
	fileColAdd
	fileColDel
	fileColPath
	fileColCount
)

var fileColumns = []string{"st", "+", "-", "path"}

func fileCell(f git.FileChange, col int) string {
	switch col {
	case fileColStatus:
		if f.Status == "" {
			return "M"
		}
		return f.Status
	case fileColAdd:
		return fmt.Sprintf("%d", f.Additions)
	case fileColDel:
		return fmt.Sprintf("%d", f.Deletions)
	case fileColPath:
		if f.OldPath != "" {
			return f.OldPath + " → " + f.Path
		}
		return f.Path
	default:
		return ""
	}
}

func fileRow(f git.FileChange) []string {
	return []string{
		fileCell(f, fileColStatus),
		fileCell(f, fileColAdd),
		fileCell(f, fileColDel),
		fileCell(f, fileColPath),
	}
}

func filterFileIndicesCol(files []git.FileChange, query string, col int) []int {
	q := strings.TrimSpace(query)
	if q == "" {
		return identityIndices(len(files))
	}
	out := make([]int, 0, len(files))
	for i, f := range files {
		var ok bool
		if col < 0 || col >= fileColCount {
			ok = filterMatch(q, f.Path, f.OldPath, f.Status, fileCell(f, fileColAdd), fileCell(f, fileColDel))
		} else {
			ok = filterMatch(q, fileCell(f, col))
			if col == fileColPath {
				ok = ok || filterMatch(q, f.Path, f.OldPath)
			}
		}
		if ok {
			out = append(out, i)
		}
	}
	return out
}

func sortFileIndices(files []git.FileChange, idx []int, col int, dir SortDir) []int {
	if dir == SortNone || col < 0 || col >= fileColCount || len(idx) < 2 {
		return idx
	}
	out := append([]int(nil), idx...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := files[out[i]], files[out[j]]
		cmp := 0
		switch col {
		case fileColStatus:
			cmp = strings.Compare(fileCell(a, fileColStatus), fileCell(b, fileColStatus))
		case fileColAdd:
			cmp = a.Additions - b.Additions
		case fileColDel:
			cmp = a.Deletions - b.Deletions
		case fileColPath:
			cmp = strings.Compare(a.Path, b.Path)
		}
		if dir == SortDesc {
			cmp = -cmp
		}
		return cmp < 0
	})
	return out
}
