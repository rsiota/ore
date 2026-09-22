package git

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// AuthorShare is one author in a quiet ownership rollup.
type AuthorShare struct {
	Name  string
	Count int
	Pct   int // 0–100 of the sample
}

const (
	ownershipSample = 200
	ownershipTop    = 5
)

// SummarizeAuthors counts names and returns the top shares (quiet rollup).
func SummarizeAuthors(names []string, top int) []AuthorShare {
	if top <= 0 {
		top = ownershipTop
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
func (r *Repo) PathOwnership(ctx context.Context, rev string, paths []string, sample, top int) ([]AuthorShare, error) {
	paths = uniquePaths(paths)
	if len(paths) == 0 {
		return nil, nil
	}
	if rev == "" {
		rev = "HEAD"
	}
	if sample <= 0 {
		sample = ownershipSample
	}
	if top <= 0 {
		top = ownershipTop
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
		return nil, err
	}
	raw := strings.Split(strings.TrimSpace(string(out)), "\n")
	names := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line != "" {
			names = append(names, line)
		}
	}
	return SummarizeAuthors(names, top), nil
}
