package ui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// DiffMode selects how the detail pane renders the patch body.
type DiffMode int

const (
	DiffZen DiffMode = iota // context + numbered changes, light hunk headers
	DiffUnified             // classic git show patch
)

// Default / bounds for zen context lines (git -U and in-hunk trim).
const (
	defaultZenContext = 3
	minZenContext     = 0
	maxZenContext     = 15
)

func (m DiffMode) Label() string {
	switch m {
	case DiffUnified:
		return "unified"
	default:
		return "zen"
	}
}

func (m DiffMode) Next() DiffMode {
	if m == DiffZen {
		return DiffUnified
	}
	return DiffZen
}

func clampZenContext(n int) int {
	if n < minZenContext {
		return minZenContext
	}
	if n > maxZenContext {
		return maxZenContext
	}
	return n
}

// Zen row markers — expanded in renderDetailLine with pane width available.
const (
	zenFilePrefix     = "\x1ezenf:"  // path
	zenFileRuleMarker = "\x1ezenfr"  // full-width rule above/below file path
	zenHunkPrefix     = "\x1ezenh:"  // full @@ header text
	zenCtxPrefix      = "\x1ezenc:"  // num\x1ftext (unchanged context)
	zenAddPrefix      = "\x1ezena:"  // num\x1ftext
	zenDelPrefix      = "\x1ezend:"  // num\x1ftext
	zenFieldSep       = "\x1f"
)

var hunkHeaderRe = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

type zenHunkLine struct {
	kind  byte // ' ', '+', '-'
	num   int  // display line: old for del/ctx preference, new for add
	text  string
	spans []byteSpan // intra-line highlights for paired -/ + edits
}

// zenDiffLines turns a unified diff into Fork-like zen rows: file banners,
// superlight @@ headers between separated hunks, numbered context around
// changes, and washed add/delete lines (no +/− prefix).
func zenDiffLines(diff string, context int) []string {
	context = clampZenContext(context)
	diff = strings.TrimRight(diff, "\n")
	if diff == "" {
		return nil
	}
	var out []string
	var (
		oldLine, newLine int
		inHunk           bool
		currentFile      string
		hunkHeader       string
		hunkBody         []zenHunkLine
	)
	flushHunk := func() {
		if !inHunk && hunkHeader == "" && len(hunkBody) == 0 {
			return
		}
		if hunkHeader != "" {
			out = append(out, zenHunkPrefix+hunkHeader)
		}
		annotateZenIntraLine(hunkBody)
		for _, row := range trimZenHunk(hunkBody, context) {
			switch row.kind {
			case '+':
				out = append(out, encodeZenChange(true, row.num, row.text, row.spans))
			case '-':
				out = append(out, encodeZenChange(false, row.num, row.text, row.spans))
			default:
				out = append(out, zenCtxPrefix+strconv.Itoa(row.num)+zenFieldSep+row.text)
			}
		}
		hunkHeader = ""
		hunkBody = nil
		inHunk = false
	}
	for _, raw := range strings.Split(diff, "\n") {
		line := strings.ReplaceAll(raw, "\r", "")
		switch {
		case strings.HasPrefix(line, "diff --git "):
			flushHunk()
			path := pathFromDiffGit(line)
			if path != "" && path != currentFile {
				currentFile = path
				out = appendZenFile(out, path)
			}
		case strings.HasPrefix(line, "+++ "):
			path := strings.TrimPrefix(line, "+++ ")
			path = strings.TrimPrefix(path, "b/")
			if path == "/dev/null" {
				break
			}
			if path != "" && path != currentFile {
				currentFile = path
				out = appendZenFile(out, path)
			}
		case strings.HasPrefix(line, "@@"):
			flushHunk()
			hunkHeader = line
			if m := hunkHeaderRe.FindStringSubmatch(line); len(m) == 3 {
				oldLine, _ = strconv.Atoi(m[1])
				newLine, _ = strconv.Atoi(m[2])
				inHunk = true
			}
		case !inHunk:
			continue
		case strings.HasPrefix(line, "+"):
			hunkBody = append(hunkBody, zenHunkLine{kind: '+', num: newLine, text: line[1:]})
			newLine++
		case strings.HasPrefix(line, "-"):
			hunkBody = append(hunkBody, zenHunkLine{kind: '-', num: oldLine, text: line[1:]})
			oldLine++
		case strings.HasPrefix(line, "\\"):
			continue
		default:
			// Context: show new-side line number (Fork-like single gutter).
			text := line
			if strings.HasPrefix(text, " ") {
				text = text[1:]
			}
			hunkBody = append(hunkBody, zenHunkLine{kind: ' ', num: newLine, text: text})
			oldLine++
			newLine++
		}
	}
	flushHunk()
	return out
}

// appendZenFile adds a left-aligned file banner with grid-coloured rules
// above and below. If the trailing block is already a file banner, only the
// path is updated (diff --git then +++ b/path).
func appendZenFile(out []string, path string) []string {
	n := len(out)
	if n >= 3 &&
		out[n-1] == zenFileRuleMarker &&
		strings.HasPrefix(out[n-2], zenFilePrefix) &&
		out[n-3] == zenFileRuleMarker {
		out[n-2] = zenFilePrefix + path
		return out
	}
	if n >= 1 && strings.HasPrefix(out[n-1], zenFilePrefix) {
		out[n-1] = zenFilePrefix + path
		return out
	}
	return append(out, zenFileRuleMarker, zenFilePrefix+path, zenFileRuleMarker)
}

// trimZenHunk keeps change lines plus up to context lines of unchanged
// neighbourhood on each side (Fork-style). With context matching git -U,
// this usually keeps the whole hunk.
func trimZenHunk(body []zenHunkLine, context int) []zenHunkLine {
	if len(body) == 0 || context < 0 {
		return body
	}
	first, last := -1, -1
	for i, row := range body {
		if row.kind == '+' || row.kind == '-' {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		return nil
	}
	start := first - context
	if start < 0 {
		start = 0
	}
	end := last + context + 1
	if end > len(body) {
		end = len(body)
	}
	return body[start:end]
}

func pathFromDiffGit(line string) string {
	rest := strings.TrimPrefix(line, "diff --git ")
	parts := strings.Fields(rest)
	if len(parts) < 2 {
		return ""
	}
	b := parts[len(parts)-1]
	return strings.TrimPrefix(b, "b/")
}

func parseZenNumText(payload string) (num int, text string, ok bool) {
	num, text, _, ok = parseZenChangePayload(payload)
	return num, text, ok
}

func parseZenChangePayload(payload string) (num int, text string, spans []byteSpan, ok bool) {
	parts := strings.SplitN(payload, zenFieldSep, 3)
	if len(parts) < 2 {
		return 0, "", nil, false
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", nil, false
	}
	text = parts[1]
	if len(parts) == 3 {
		spans = parseByteSpans(parts[2])
	}
	return n, text, spans, true
}

func encodeZenChange(add bool, num int, text string, spans []byteSpan) string {
	prefix := zenDelPrefix
	if add {
		prefix = zenAddPrefix
	}
	s := prefix + strconv.Itoa(num) + zenFieldSep + text
	if len(spans) > 0 {
		s += zenFieldSep + formatByteSpans(spans)
	}
	return s
}

func renderZenFile(path string, width int, wrap bool) []string {
	pad := strings.Repeat(" ", cellPad)
	bodyW := width - cellPad - cellPad
	if bodyW < 1 {
		bodyW = max(1, width-cellPad)
	}
	if !wrap {
		return []string{fitWidth(pad+styleZenFile.Render(runewidth.Truncate(path, bodyW, "…")), width)}
	}
	chunks := wrapDisplay(path, bodyW)
	rows := make([]string, 0, len(chunks))
	for _, c := range chunks {
		rows = append(rows, fitWidth(pad+styleZenFile.Render(c), width))
	}
	return rows
}

func renderZenHunk(header string, width int, wrap bool) []string {
	pad := strings.Repeat(" ", zenGutterCols())
	bodyW := width - zenGutterCols() - cellPad
	if bodyW < 1 {
		bodyW = max(1, width-zenGutterCols())
	}
	if !wrap {
		return []string{fitWidth(pad+styleZenHunk.Render(runewidth.Truncate(header, bodyW, "…")), width)}
	}
	chunks := wrapDisplay(header, bodyW)
	rows := make([]string, 0, len(chunks))
	for _, c := range chunks {
		rows = append(rows, fitWidth(pad+styleZenHunk.Render(c), width))
	}
	return rows
}

// zenGutterNums is the digit width. The full gutter is:
// cellPad + digits + one space + │  (matches filepath left pad).
const zenGutterNums = 5

func zenGutterCols() int {
	return cellPad + zenGutterNums + 1 + 1 // pad + digits + gap + rule
}

func zenGutter(num int) string {
	left := strings.Repeat(" ", cellPad)
	nums := styleZenHunk.Render(fmt.Sprintf("%*d", zenGutterNums, num))
	gap := styleZenHunk.Render(" ")
	return left + nums + gap + styleGridBorder.Render("│")
}

func zenGutterCont() string {
	return strings.Repeat(" ", cellPad+zenGutterNums+1) + styleGridBorder.Render("│")
}

func renderZenContext(num int, text string, width int, wrap bool) []string {
	return renderZenCode(styleMuted, num, text, width, wrap)
}

func renderZenChange(add bool, num int, text string, spans []byteSpan, width int, wrap bool) []string {
	base, strong := styleDelWash, styleDelStrong
	if add {
		base, strong = styleAddWash, styleAddStrong
	}
	return renderZenCodeSegments(base, strong, num, text, spans, width, wrap)
}

func renderZenCode(st lipgloss.Style, num int, text string, width int, wrap bool) []string {
	return renderZenCodeSegments(st, st, num, text, nil, width, wrap)
}

func renderZenCodeSegments(base, strong lipgloss.Style, num int, text string, spans []byteSpan, width int, wrap bool) []string {
	gutter := zenGutter(num)
	bodyW := width - lipgloss.Width(gutter) - cellPad
	if bodyW < 1 {
		bodyW = max(1, width-lipgloss.Width(gutter))
	}
	if !wrap {
		return []string{fitWidth(gutter+renderHighlightedCell(base, strong, text, spans, bodyW), width)}
	}
	chunks := wrapDisplay(text, bodyW)
	cont := zenGutterCont()
	rows := make([]string, 0, len(chunks))
	offset := 0
	for i, c := range chunks {
		g := gutter
		if i > 0 {
			g = cont
		}
		chunkSpans := shiftSpans(spans, offset, offset+len(c))
		rows = append(rows, fitWidth(g+renderHighlightedCell(base, strong, c, chunkSpans, bodyW), width))
		offset += len(c)
	}
	return rows
}
