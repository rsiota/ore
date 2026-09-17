package ui

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// byteSpan is a half-open UTF-8 byte range [Start, End) within a line.
type byteSpan struct {
	Start, End int
}

// annotateZenIntraLine attaches word-diff spans to 1:1 paired -/ + runs.
func annotateZenIntraLine(body []zenHunkLine) {
	for i := 0; i < len(body); {
		if body[i].kind != '-' {
			i++
			continue
		}
		j := i
		for j < len(body) && body[j].kind == '-' {
			j++
		}
		k := j
		for k < len(body) && body[k].kind == '+' {
			k++
		}
		n := min(j-i, k-j)
		for t := 0; t < n; t++ {
			delSpans, addSpans := intraLineDiff(body[i+t].text, body[j+t].text)
			body[i+t].spans = delSpans
			body[j+t].spans = addSpans
		}
		i = k
	}
}

// annotateUnifiedIntra rewrites paired -/ + patch lines with intra markers so
// renderDetailRows can apply dual washes. Unpaired lines stay as plain -/ +.
func annotateUnifiedIntra(lines []string) []string {
	out := make([]string, 0, len(lines))
	i := 0
	for i < len(lines) {
		line := strings.ReplaceAll(lines[i], "\r", "")
		if !isUnifiedChangeLine(line, '-') {
			out = append(out, line)
			i++
			continue
		}
		j := i
		for j < len(lines) && isUnifiedChangeLine(strings.ReplaceAll(lines[j], "\r", ""), '-') {
			j++
		}
		k := j
		for k < len(lines) && isUnifiedChangeLine(strings.ReplaceAll(lines[k], "\r", ""), '+') {
			k++
		}
		n := min(j-i, k-j)
		for t := 0; t < j-i; t++ {
			raw := strings.ReplaceAll(lines[i+t], "\r", "")
			text := raw[1:]
			if t < n {
				delSpans, _ := intraLineDiff(text, strings.ReplaceAll(lines[j+t], "\r", "")[1:])
				out = append(out, encodeIntraLine(false, text, delSpans))
			} else {
				out = append(out, raw)
			}
		}
		for t := 0; t < k-j; t++ {
			raw := strings.ReplaceAll(lines[j+t], "\r", "")
			text := raw[1:]
			if t < n {
				_, addSpans := intraLineDiff(strings.ReplaceAll(lines[i+t], "\r", "")[1:], text)
				out = append(out, encodeIntraLine(true, text, addSpans))
			} else {
				out = append(out, raw)
			}
		}
		i = k
	}
	return out
}

func isUnifiedChangeLine(line string, kind byte) bool {
	if line == "" || line[0] != kind {
		return false
	}
	// File headers --- / +++ are not change lines.
	if kind == '-' && strings.HasPrefix(line, "---") {
		return false
	}
	if kind == '+' && strings.HasPrefix(line, "+++") {
		return false
	}
	return true
}

const (
	intraAddPrefix = "\x1eintra+:"
	intraDelPrefix = "\x1eintra-:"
)

func encodeIntraLine(add bool, text string, spans []byteSpan) string {
	prefix := intraDelPrefix
	if add {
		prefix = intraAddPrefix
	}
	if len(spans) == 0 {
		return prefix + text
	}
	return prefix + text + zenFieldSep + formatByteSpans(spans)
}

func parseIntraPayload(payload string) (text string, spans []byteSpan) {
	text, rest, cut := strings.Cut(payload, zenFieldSep)
	if !cut {
		return payload, nil
	}
	return text, parseByteSpans(rest)
}

func formatByteSpans(spans []byteSpan) string {
	if len(spans) == 0 {
		return ""
	}
	parts := make([]string, 0, len(spans))
	for _, sp := range mergeByteSpans(spans) {
		if sp.End <= sp.Start {
			continue
		}
		parts = append(parts, strconv.Itoa(sp.Start)+"-"+strconv.Itoa(sp.End))
	}
	return strings.Join(parts, ",")
}

func parseByteSpans(s string) []byteSpan {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []byteSpan
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		a, b, ok := strings.Cut(part, "-")
		if !ok {
			continue
		}
		start, err1 := strconv.Atoi(a)
		end, err2 := strconv.Atoi(b)
		if err1 != nil || err2 != nil || end <= start || start < 0 {
			continue
		}
		out = append(out, byteSpan{Start: start, End: end})
	}
	return mergeByteSpans(out)
}

func mergeByteSpans(spans []byteSpan) []byteSpan {
	if len(spans) == 0 {
		return nil
	}
	out := append([]byteSpan(nil), spans...)
	// Insertion sort by Start — spans are few.
	for i := 1; i < len(out); i++ {
		j := i
		for j > 0 && out[j-1].Start > out[j].Start {
			out[j-1], out[j] = out[j], out[j-1]
			j--
		}
	}
	merged := []byteSpan{out[0]}
	for _, sp := range out[1:] {
		last := &merged[len(merged)-1]
		if sp.Start <= last.End {
			if sp.End > last.End {
				last.End = sp.End
			}
			continue
		}
		merged = append(merged, sp)
	}
	return merged
}

type diffToken struct {
	text  string
	start int // byte offset in original string
}

// tokenizeDiff splits into word runs and individual non-word runes so edits
// like renaming an identifier highlight the identifier, not the whole line.
func tokenizeDiff(s string) []diffToken {
	if s == "" {
		return nil
	}
	var out []diffToken
	i := 0
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if isDiffWordRune(r) {
			j := i + size
			for j < len(s) {
				r2, sz := utf8.DecodeRuneInString(s[j:])
				if !isDiffWordRune(r2) {
					break
				}
				j += sz
			}
			out = append(out, diffToken{text: s[i:j], start: i})
			i = j
			continue
		}
		out = append(out, diffToken{text: s[i : i+size], start: i})
		i += size
	}
	return out
}

func isDiffWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// intraLineDiff returns changed byte spans for a paired old/new line.
func intraLineDiff(oldText, newText string) (delSpans, addSpans []byteSpan) {
	if oldText == newText {
		return nil, nil
	}
	if oldText == "" {
		return nil, []byteSpan{{0, len(newText)}}
	}
	if newText == "" {
		return []byteSpan{{0, len(oldText)}}, nil
	}
	a := tokenizeDiff(oldText)
	b := tokenizeDiff(newText)
	aCh, bCh := tokenDiffChanged(a, b)
	return spansFromChangedTokens(a, aCh, len(oldText)), spansFromChangedTokens(b, bCh, len(newText))
}

func spansFromChangedTokens(toks []diffToken, changed []bool, strLen int) []byteSpan {
	var out []byteSpan
	for i, tok := range toks {
		if i >= len(changed) || !changed[i] {
			continue
		}
		end := tok.start + len(tok.text)
		if end > strLen {
			end = strLen
		}
		if end > tok.start {
			out = append(out, byteSpan{Start: tok.start, End: end})
		}
	}
	return mergeByteSpans(out)
}

// tokenDiffChanged marks tokens not on the LCS as changed (classic DP).
func tokenDiffChanged(a, b []diffToken) (aCh, bCh []bool) {
	n, m := len(a), len(b)
	aCh = make([]bool, n)
	bCh = make([]bool, m)
	if n == 0 && m == 0 {
		return aCh, bCh
	}
	// Cap pathological lines — fall back to whole-line highlight.
	if n > 400 || m > 400 || n*m > 80_000 {
		for i := range aCh {
			aCh[i] = true
		}
		for i := range bCh {
			bCh[i] = true
		}
		return aCh, bCh
	}
	// dp[i][j] = LCS length of a[i:] and b[j:]
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i].text == b[j].text {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	i, j := 0, 0
	for i < n && j < m {
		if a[i].text == b[j].text {
			i++
			j++
			continue
		}
		if dp[i+1][j] >= dp[i][j+1] {
			aCh[i] = true
			i++
		} else {
			bCh[j] = true
			j++
		}
	}
	for ; i < n; i++ {
		aCh[i] = true
	}
	for ; j < m; j++ {
		bCh[j] = true
	}
	return aCh, bCh
}

// shiftSpans returns spans overlapping [from,to), remapped to local offsets.
func shiftSpans(spans []byteSpan, from, to int) []byteSpan {
	if len(spans) == 0 || to <= from {
		return nil
	}
	var out []byteSpan
	for _, sp := range spans {
		start := max(sp.Start, from)
		end := min(sp.End, to)
		if end > start {
			out = append(out, byteSpan{Start: start - from, End: end - from})
		}
	}
	return out
}

// renderHighlightedCell paints soft wash on the whole cell and a stronger wash
// on changed spans. Padding uses the soft wash so the line reads as one band.
func renderHighlightedCell(base, strong lipgloss.Style, text string, spans []byteSpan, width int) string {
	return renderLayeredCell(base, strong, strong, text, spans, nil, width)
}

// renderLayeredCell paints base across the cell, intraStrong on intra spans, and
// searchStrong on search spans (search wins on overlap).
func renderLayeredCell(base, intraStrong, searchStrong lipgloss.Style, text string, intra, search []byteSpan, width int) string {
	if width <= 0 {
		return ""
	}
	intra = mergeByteSpans(intra)
	search = mergeByteSpans(search)
	if len(intra) == 0 && len(search) == 0 {
		return cell(base, text, width)
	}
	n := len(text)
	if n == 0 {
		return cell(base, text, width)
	}
	cover := make([]byte, n) // 0 base, 1 intra, 2 search
	mark := func(spans []byteSpan, layer byte) {
		for _, sp := range spans {
			start, end := sp.Start, sp.End
			if start < 0 {
				start = 0
			}
			if end > n {
				end = n
			}
			for i := start; i < end; i++ {
				if layer >= cover[i] {
					cover[i] = layer
				}
			}
		}
	}
	mark(intra, 1)
	mark(search, 2)

	var b strings.Builder
	i := 0
	for i < n {
		layer := cover[i]
		j := i + 1
		for j < n && cover[j] == layer {
			j++
		}
		chunk := text[i:j]
		switch layer {
		case 2:
			b.WriteString(searchStrong.Render(chunk))
		case 1:
			b.WriteString(intraStrong.Render(chunk))
		default:
			b.WriteString(base.Render(chunk))
		}
		i = j
	}
	styled := b.String()
	w := lipgloss.Width(styled)
	switch {
	case w < width:
		return styled + base.Render(strings.Repeat(" ", width-w))
	case w > width:
		return fitWidth(styled, width)
	default:
		return styled
	}
}

