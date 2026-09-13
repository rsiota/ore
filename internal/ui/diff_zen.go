package ui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
	kind byte // ' ', '+', '-'
	num  int  // display line: old for del/ctx preference, new for add
	text string
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
		for _, row := range trimZenHunk(hunkBody, context) {
			switch row.kind {
			case '+':
				out = append(out, zenAddPrefix+strconv.Itoa(row.num)+zenFieldSep+row.text)
			case '-':
				out = append(out, zenDelPrefix+strconv.Itoa(row.num)+zenFieldSep+row.text)
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
	numStr, text, cut := strings.Cut(payload, zenFieldSep)
	if !cut {
		return 0, "", false
	}
	n, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, "", false
	}
	return n, text, true
}

func renderZenFile(path string, width int) string {
	pad := strings.Repeat(" ", cellPad)
	return fitWidth(pad+styleZenFile.Render(path), width)
}

func renderZenHunk(header string, width int) string {
	// Align @@ with code, not with the line-number column.
	pad := strings.Repeat(" ", zenGutterCols())
	return fitWidth(pad+styleZenHunk.Render(header), width)
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

func renderZenContext(num int, text string, width int) string {
	gutter := zenGutter(num)
	bodyW := width - lipgloss.Width(gutter)
	if bodyW < 1 {
		return fitWidth(styleMuted.Render(text), width)
	}
	return fitWidth(gutter+cell(styleMuted, text, bodyW), width)
}

func renderZenChange(add bool, num int, text string, width int) string {
	gutter := zenGutter(num)
	bodyW := width - lipgloss.Width(gutter)
	st := styleDelWash
	if add {
		st = styleAddWash
	}
	if bodyW < 1 {
		return cell(st, text, width)
	}
	return fitWidth(gutter+cell(st, text, bodyW), width)
}
