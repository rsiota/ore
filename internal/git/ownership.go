package git

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// AuthorShare is one author in a quiet ownership rollup.
type AuthorShare struct {
	Name  string
	Count int
	Pct   int // 0–100 of the sample
}

const (
	OwnershipSample    = 200 // commits examined for a PathOwnership rollup
	OwnershipTop       = 5   // authors returned
	OwnershipCommitCap = 100 // PathAuthorCommits default window
)

// SummarizeAuthors counts names and returns the top shares (quiet rollup).
func SummarizeAuthors(names []string, top int) []AuthorShare {
	if top <= 0 {
		top = OwnershipTop
	}
	counts := map[string]int{}
	order := make([]string, 0)
	total := 0
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, ok := counts[n]; !ok {
			order = append(order, n)
		}
		counts[n]++
		total++
	}
	if total == 0 {
		return nil
	}
	type pair struct {
		name  string
		count int
	}
	pairs := make([]pair, 0, len(counts))
	for _, name := range order {
		pairs = append(pairs, pair{name: name, count: counts[name]})
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		if pairs[i].count != pairs[j].count {
			return pairs[i].count > pairs[j].count
		}
		return strings.ToLower(pairs[i].name) < strings.ToLower(pairs[j].name)
	})
	if len(pairs) > top {
		pairs = pairs[:top]
	}
	out := make([]AuthorShare, len(pairs))
	for i, p := range pairs {
		pct := (p.count * 100) / total
		if pct == 0 && p.count > 0 {
			pct = 1
		}
		out[i] = AuthorShare{Name: p.name, Count: p.count, Pct: pct}
	}
	return out
}

// PathOwnership samples recent authors of paths at rev (newest-first log).
// The int is how many commits were examined (the sample, not the top-N count).
func (r *Repo) PathOwnership(ctx context.Context, rev string, paths []string, sample, top int) ([]AuthorShare, int, error) {
	paths = uniquePaths(paths)
	if len(paths) == 0 {
		return nil, 0, nil
	}
	if rev == "" {
		rev = "HEAD"
	}
	if sample <= 0 {
		sample = OwnershipSample
	}
	if top <= 0 {
		top = OwnershipTop
	}
	if len(paths) > coChangeMaxSeeds {
		paths = paths[:coChangeMaxSeeds]
	}

	args := []string{
		"log", rev,
		"--format=%aN",
		"-n", fmt.Sprintf("%d", sample),
		"--",
	}
	args = append(args, paths...)
	out, err := r.run(ctx, args...)
	if err != nil {
		return nil, 0, err
	}
	raw := strings.Split(strings.TrimSpace(string(out)), "\n")
	names := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line != "" {
			names = append(names, line)
		}
	}
	return SummarizeAuthors(names, top), len(names), nil
}

// PathAuthorCommits lists newest-first commits by author that touched paths
// (or the whole tree when paths is empty). Used to drill from an Ownership row.
func (r *Repo) PathAuthorCommits(ctx context.Context, author, rev string, paths []string, maxCount int) ([]PickaxeHit, error) {
	author = strings.TrimSpace(author)
	if author == "" {
		return nil, fmt.Errorf("author required")
	}
	paths = uniquePaths(paths)
	if len(paths) > coChangeMaxSeeds {
		paths = paths[:coChangeMaxSeeds]
	}
	if rev == "" {
		rev = "HEAD"
	}
	if maxCount <= 0 {
		maxCount = OwnershipCommitCap
	}

	format := recSep + strings.Join([]string{
		"%H", "%h", "%an", "%ae", "%aI", "%s", "%P",
	}, fieldSep)

	args := []string{
		"log", rev,
		"--author=" + regexp.QuoteMeta(author),
		"--name-only",
		"--date=iso-strict",
		"--pretty=format:" + format,
		"-n", strconv.Itoa(maxCount),
	}
	if len(paths) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
	}

	out, err := r.run(ctx, args...)
	if err != nil {
		return nil, err
	}
	hits, err := parsePickaxeLog(out)
	if err != nil {
		return nil, err
	}
	// --author is a regex against the ident; keep %aN matches from the rollup.
	exact := hits[:0]
	for _, h := range hits {
		if strings.EqualFold(h.Commit.Author, author) {
			exact = append(exact, h)
		}
	}
	return exact, nil
}
