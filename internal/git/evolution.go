package git

import (
	"context"
	"fmt"
)

// LineEvolutionStep is one snapshot in a blame line's provenance chain.
type LineEvolutionStep struct {
	Index int    // 0 = starting view (newest in the walk)
	Path  string // path at this revision (rename-aware)
	Rev   string // revision at which this line was blamed
	Line  BlameLine
}

// DefaultEvolutionLimit is the walk cap when maxSteps is 0.
const DefaultEvolutionLimit = 40

// LineEvolution walks porcelain `previous` from start, building a navigable
// provenance stack for one logical line. start is the line as currently shown
// at path@rev (typically the selected blame row). Renames use PreviousPath when
// present. Stops at origin, cycles, or maxSteps (0 → default cap).
func (r *Repo) LineEvolution(ctx context.Context, path, rev string, start BlameLine, maxSteps int) ([]LineEvolutionStep, error) {
	if path == "" {
		return nil, fmt.Errorf("path required")
	}
	if rev == "" {
		rev = "HEAD"
	}
	if maxSteps <= 0 {
		maxSteps = DefaultEvolutionLimit
	}
	if start.Line < 1 {
		return nil, fmt.Errorf("line must be >= 1")
	}

	out := make([]LineEvolutionStep, 0, maxSteps)
	seen := map[string]struct{}{}

	curPath, curRev := path, rev
	cur := start
	for i := 0; i < maxSteps; i++ {
		key := curRev + "\x00" + curPath + "\x00" + fmt.Sprintf("%d\x00%s", cur.Line, cur.Hash)
		if _, ok := seen[key]; ok {
			break
		}
		seen[key] = struct{}{}

		out = append(out, LineEvolutionStep{
			Index: i,
			Path:  curPath,
			Rev:   curRev,
			Line:  cur,
		})

		if cur.PreviousHash == "" {
			break
		}
		nextPath := curPath
		if cur.PreviousPath != "" {
			nextPath = cur.PreviousPath
		}
		nextRev := cur.PreviousHash
		nextLine := cur.Line // same heuristic as one-hop followPreferLine

		bl, err := r.BlameAtLine(ctx, nextPath, nextRev, nextLine)
		if err != nil {
			// Line numbers shift; try a full blame and land on prefer line.
			all, berr := r.Blame(ctx, nextPath, BlameOptions{Rev: nextRev})
			if berr != nil || len(all) == 0 {
				break
			}
			bl = all[0]
			for _, row := range all {
				bl = row
				if row.Line >= nextLine {
					break
				}
			}
		}
		curPath, curRev, cur = nextPath, nextRev, bl
	}
	return out, nil
}
