package ui

import (
	"regexp"
	"strings"
	"sync"

	"github.com/rsiota/ore/internal/git"
)

// pickaxeSpans returns byte ranges in text that match the active pickaxe query.
// String mode mirrors git log -S (literal, case-sensitive). Regexp mode uses
// Go's RE2 engine (close to git -G for common patterns).
func pickaxeSpans(text, query string, mode git.PickaxeMode) []byteSpan {
	query = strings.TrimSpace(query)
	if query == "" || text == "" {
		return nil
	}
	switch mode {
	case git.PickaxeRegexp:
		re, err := compilePickaxeRegexp(query)
		if err != nil || re == nil {
			return nil
		}
		idxs := re.FindAllStringIndex(text, -1)
		if len(idxs) == 0 {
			return nil
		}
		out := make([]byteSpan, 0, len(idxs))
		for _, pair := range idxs {
			if pair[1] > pair[0] {
				out = append(out, byteSpan{Start: pair[0], End: pair[1]})
			}
		}
		return mergeByteSpans(out)
	default:
		var out []byteSpan
		from := 0
		for {
			i := strings.Index(text[from:], query)
			if i < 0 {
				break
			}
			start := from + i
			end := start + len(query)
			out = append(out, byteSpan{Start: start, End: end})
			from = end
			if len(query) == 0 {
				break
			}
		}
		return mergeByteSpans(out)
	}
}

var pickaxeRegexpCache sync.Map // string → *regexp.Regexp or error sentinel

type pickaxeRegexpErr struct{ error }

func compilePickaxeRegexp(query string) (*regexp.Regexp, error) {
	if v, ok := pickaxeRegexpCache.Load(query); ok {
		switch t := v.(type) {
		case *regexp.Regexp:
			return t, nil
		case pickaxeRegexpErr:
			return nil, t.error
		}
	}
	re, err := regexp.Compile(query)
	if err != nil {
		pickaxeRegexpCache.Store(query, pickaxeRegexpErr{err})
		return nil, err
	}
	pickaxeRegexpCache.Store(query, re)
	return re, nil
}

func (m Model) pickaxeHLActive() bool {
	return m.pickHlOn && strings.TrimSpace(m.pickQuery) != ""
}

func (m Model) pickaxeSearchSpans(text string) []byteSpan {
	if !m.pickaxeHLActive() {
		return nil
	}
	return pickaxeSpans(text, m.pickQuery, m.pickMode)
}

func (m *Model) clearPickaxeHighlight() {
	if !m.pickHlOn {
		m.status = "pickaxe highlight already off"
		return
	}
	m.pickHlOn = false
	m.invalidateDetailCache()
	m.status = "pickaxe highlight off · :pickaxe to search again"
}

func (m *Model) enablePickaxeHighlight() {
	m.pickHlOn = true
	m.invalidateDetailCache()
}
