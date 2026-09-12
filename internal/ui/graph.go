package ui

import (
	"strings"

	"github.com/rsiota/ore/internal/git"
)

// Max lanes drawn in the soft graph column (keeps the grid usable).
const graphMaxLanes = 10

// buildSoftGraph returns one Unicode graph cell per row in idx order.
// Lanes follow parent links among commits that appear later in the same list.
// Parallel children keep separate lanes until their shared parent is drawn;
// merge/fork bridges (─ ╮ ├ …) make that topology scannable on one row.
func buildSoftGraph(commits []git.Commit, idx []int) []string {
	n := len(idx)
	out := make([]string, n)
	if n == 0 {
		return out
	}

	pos := make(map[string]int, n)
	for i, src := range idx {
		pos[commits[src].Hash] = i
	}

	// Active lanes: hash expected next in that column ("" = reusable hole).
	// Duplicate hashes are intentional — two children waiting on one parent.
	active := make([]string, 0, 8)

	for row, src := range idx {
		c := commits[src]
		lane := findLane(active, c.Hash)
		if lane < 0 {
			lane = claimLane(&active, c.Hash)
		}

		glyphs := make([]rune, max(len(active), 1))
		for i := range glyphs {
			glyphs[i] = ' '
		}
		for i, slot := range active {
			if slot == "" {
				continue
			}
			if i == lane {
				glyphs[i] = '●'
			} else {
				glyphs[i] = '│'
			}
		}

		// Other lanes also waiting for this commit merge into the node.
		for i, slot := range active {
			if i == lane || slot != c.Hash {
				continue
			}
			paintMergeLink(glyphs, i, lane)
			active[i] = ""
		}

		parents := visibleParents(c.Parents, pos, row)
		if len(parents) == 0 {
			active[lane] = ""
		} else {
			// First parent continues this lane (duplicates allowed).
			active[lane] = parents[0]
			for _, p := range parents[1:] {
				if target := findLane(active, p); target >= 0 {
					paintMergeLink(glyphs, lane, target)
					continue
				}
				target := claimLane(&active, p)
				glyphs = ensureGlyphWidth(glyphs, target+1)
				paintMergeLink(glyphs, lane, target)
			}
		}

		trimActive(&active)
		out[row] = spaceGraphGlyphs(strings.TrimRight(string(glyphs), " "))
		if out[row] == "" {
			out[row] = "●"
		}
	}
	return out
}

// spaceGraphGlyphs inserts a thin gap between lane columns for scannability.
func spaceGraphGlyphs(s string) string {
	if s == "" || len([]rune(s)) == 1 {
		return s
	}
	runes := []rune(s)
	var b strings.Builder
	b.Grow(len(runes) * 2)
	for i, r := range runes {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func ensureGlyphWidth(glyphs []rune, width int) []rune {
	if len(glyphs) >= width {
		return glyphs
	}
	extra := make([]rune, width-len(glyphs))
	for i := range extra {
		extra[i] = ' '
	}
	return append(glyphs, extra...)
}

func visibleParents(parents []string, pos map[string]int, row int) []string {
	if len(parents) == 0 {
		return nil
	}
	out := make([]string, 0, len(parents))
	seen := make(map[string]bool, len(parents))
	for _, p := range parents {
		if p == "" || seen[p] {
			continue
		}
		if at, ok := pos[p]; ok && at > row {
			out = append(out, p)
			seen[p] = true
		}
	}
	return out
}

func findLane(active []string, hash string) int {
	for i, slot := range active {
		if slot == hash {
			return i
		}
	}
	return -1
}

func claimLane(active *[]string, hash string) int {
	for i, slot := range *active {
		if slot == "" {
			(*active)[i] = hash
			return i
		}
	}
	if len(*active) >= graphMaxLanes {
		i := len(*active) - 1
		(*active)[i] = hash
		return i
	}
	*active = append(*active, hash)
	return len(*active) - 1
}

func trimActive(active *[]string) {
	for len(*active) > 0 && (*active)[len(*active)-1] == "" {
		*active = (*active)[:len(*active)-1]
	}
}

// paintMergeLink draws a horizontal bridge between two columns on the current row.
// Prefer corners (╭ ╮ ╰ ╯) over tees (├ ┤ ┼) so forks/merges read as angles.
func paintMergeLink(glyphs []rune, from, to int) {
	if from == to || from < 0 || to < 0 {
		return
	}
	lo, hi := from, to
	if lo > hi {
		lo, hi = hi, lo
	}
	if hi >= len(glyphs) {
		return
	}
	for i := lo + 1; i < hi; i++ {
		switch glyphs[i] {
		case '●':
			// keep node
		default:
			// Prefer a clean horizontal over a tee/cross through a pipe.
			glyphs[i] = '─'
		}
	}
	if glyphs[lo] != '●' {
		if from < to {
			glyphs[lo] = '╭'
		} else {
			glyphs[lo] = '╰'
		}
	}
	if glyphs[hi] != '●' {
		if from < to {
			glyphs[hi] = '╮'
		} else {
			glyphs[hi] = '╯'
		}
	}
}
