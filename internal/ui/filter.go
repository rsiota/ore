package ui

import (
	"strings"

	"github.com/rsiota/ore/internal/git"
)

// filterMatch reports whether query matches any of the haystacks (case-insensitive substring).
func filterMatch(query string, haystacks ...string) bool {
	q := strings.TrimSpace(strings.ToLower(query))
	if q == "" {
		return true
	}
	for _, h := range haystacks {
		if strings.Contains(strings.ToLower(h), q) {
			return true
		}
	}
	return false
}

func filterCommitIndices(commits []git.Commit, query string) []int {
	out := make([]int, 0, len(commits))
	for i, c := range commits {
		if filterMatch(query, c.Subject, c.Author, c.ShortHash, c.Hash, c.Email) {
			out = append(out, i)
		}
	}
	return out
}

func filterFileIndices(files []git.FileChange, query string) []int {
	out := make([]int, 0, len(files))
	for i, f := range files {
		if filterMatch(query, f.Path, f.OldPath, f.Status) {
			out = append(out, i)
		}
	}
	return out
}

func filterBlameIndices(lines []git.BlameLine, query string) []int {
	out := make([]int, 0, len(lines))
	for i, l := range lines {
		if filterMatch(query, l.Text, l.Author, l.Summary, l.ShortHash, l.Hash) {
			out = append(out, i)
		}
	}
	return out
}
