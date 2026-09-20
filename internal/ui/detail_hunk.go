package ui

import (
	"strings"
)

// isHunkHeaderPlain reports whether a stripped detail row is a diff hunk header.
// Works for zen (painted @@ text) and unified (`@@ -a,b +c,d @@`).
func isHunkHeaderPlain(plain string) bool {
	s := strings.TrimSpace(plain)
	return strings.HasPrefix(s, "@@")
}

// detailHunkRows returns visual-row indexes of hunk headers in plain detail lines.
func detailHunkRows(plain []string) []int {
	var out []int
	for i, line := range plain {
		if isHunkHeaderPlain(line) {
			out = append(out, i)
		}
	}
	return out
}

// nextHunkRow returns the next hunk at or after cur+1.
func nextHunkRow(hunks []int, cur int) (row int, ok bool) {
	for _, h := range hunks {
		if h > cur {
			return h, true
		}
	}
	return 0, false
}

// prevHunkRow returns the previous hunk before cur.
func prevHunkRow(hunks []int, cur int) (row int, ok bool) {
	for i := len(hunks) - 1; i >= 0; i-- {
		if hunks[i] < cur {
			return hunks[i], true
		}
	}
	return 0, false
}

// hunkOrdinal returns 1-based index of row in hunks, or 0 if row is not a hunk header.
func hunkOrdinal(hunks []int, row int) int {
	for i, h := range hunks {
		if h == row {
			return i + 1
		}
		if h > row {
			return 0
		}
	}
	return 0
}
