package ui

import (
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// wrapDisplay splits s into display-width chunks of at most width cells.
// Empty input yields a single empty string so callers always get a row.
func wrapDisplay(s string, width int) []string {
	if width < 1 {
		width = 1
	}
	if s == "" {
		return []string{""}
	}
	var out []string
	for s != "" {
		if runewidth.StringWidth(s) <= width {
			out = append(out, s)
			break
		}
		cut := 0
		w := 0
		for i, r := range s {
			rw := runewidth.RuneWidth(r)
			if w+rw > width {
				break
			}
			w += rw
			cut = i + utf8.RuneLen(r)
		}
		if cut == 0 {
			_, size := utf8.DecodeRuneInString(s)
			cut = size
		}
		out = append(out, s[:cut])
		s = s[cut:]
	}
	return out
}
