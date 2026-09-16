package git

import (
	"context"
	"sort"
	"strconv"
	"strings"
)

// CoChange is a path that often changes in the same commits as a set of seed
// paths — a hot spot for archaeology (“what usually moved with this?”).
type CoChange struct {
	Path  string
	Count int // commits in the sample that also touched Path
	Total int // sample size (commits examined that touched the seeds)
}

const (
	coChangeSample   = 80 // max commits to sample
	coChangeTop      = 12 // max hot spots returned
	coChangeMaxSeeds = 20 // cap seeds when a commit touches many files
)

// CoChangedFiles ranks paths that co-occur with seeds across recent history.
// excludeHash, when set, skips that commit (typically the seed commit itself so
// its own files are not counted as hot spots). Returns nil when there are no
// seeds or no co-occurrences.
func (r *Repo) CoChangedFiles(ctx context.Context, seeds []string, excludeHash string) ([]CoChange, error) {
	seeds = uniquePaths(seeds)
	if len(seeds) == 0 {
		return nil, nil
	}
	if len(seeds) > coChangeMaxSeeds {
		seeds = seeds[:coChangeMaxSeeds]
	}
	seedSet := make(map[string]struct{}, len(seeds))
	for _, s := range seeds {
		seedSet[s] = struct{}{}
	}

	// Path-limited log only lists the matching paths; gather commit hashes
	// first, then re-show each commit's full name-only file list.
	listArgs := []string{"rev-list", "--all", "-n", strconv.Itoa(coChangeSample), "--"}
	listArgs = append(listArgs, seeds...)
	listOut, err := r.run(ctx, listArgs...)
	if err != nil {
		return nil, err
	}
	var hashes []string
	for _, line := range strings.Split(strings.TrimSpace(string(listOut)), "\n") {
		h := strings.TrimSpace(line)
		if h == "" {
			continue
		}
		if excludeHash != "" && hashMatch(h, excludeHash) {
			continue
		}
		hashes = append(hashes, h)
	}
	if len(hashes) == 0 {
		return nil, nil
	}

	showArgs := append([]string{
		"show", "--name-only", "--pretty=format:" + recSep + "%H",
	}, hashes...)
	out, err := r.run(ctx, showArgs...)
	if err != nil {
		return nil, err
	}

	counts := map[string]int{}
	total := 0
	for _, block := range strings.Split(string(out), recSep) {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		lines := strings.Split(block, "\n")
		hash := strings.TrimSpace(lines[0])
		if hash == "" {
			continue
		}
		total++
		seen := map[string]struct{}{}
		for _, line := range lines[1:] {
			path := strings.TrimSpace(line)
			if path == "" {
				continue
			}
			if _, skip := seedSet[path]; skip {
				continue
			}
			if _, dup := seen[path]; dup {
				continue
			}
			seen[path] = struct{}{}
			counts[path]++
		}
	}
	if total == 0 || len(counts) == 0 {
		return nil, nil
	}

	minCount := 1
	if total >= 5 {
		minCount = 2 // need a repeated signal once the sample is large enough
	}

	outList := make([]CoChange, 0, len(counts))
	for path, n := range counts {
		if n < minCount {
			continue
		}
		outList = append(outList, CoChange{Path: path, Count: n, Total: total})
	}
	sort.Slice(outList, func(i, j int) bool {
		if outList[i].Count != outList[j].Count {
			return outList[i].Count > outList[j].Count
		}
		return outList[i].Path < outList[j].Path
	})
	if len(outList) > coChangeTop {
		outList = outList[:coChangeTop]
	}
	return outList, nil
}

func uniquePaths(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	var out []string
	for _, p := range in {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func hashMatch(a, b string) bool {
	if a == b {
		return true
	}
	return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
}
