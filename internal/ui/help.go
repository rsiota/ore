package ui

import (
	"regexp"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpPanel renders a full-screen, scrollable, tabbed help overlay:
//
//   - Keys     — every keybinding (registry()), one row per binding.
//   - Commands — every ":" command (exCommands()), descriptions wrapped.
//
// Both pages scroll vertically (↑/↓ or j/k, PgUp/PgDn, g/G). j/k move a line
// cursor within the viewport; the content only scrolls when the cursor hits
// the top or bottom edge. The cursor is drawn on the first character of the
// line (not a full-row highlight). Tab / shift+tab switch pages; ? / q / esc
// close it (unmapped keys leave it open so mouse overscroll noise cannot
// dismiss it).
//
// Search (/): typing `/query` live-highlights matches on the current page and
// scrolls the first match into view; n / N cycle matches; esc clears. Offsets
// are clamped at render time so the panel never overflows its borders.
type HelpPanel struct {
	visible bool
	page    int // helpPageKeys | helpPageCommands
	keysOff int
	cmdsOff int
	keysCur int
	cmdsCur int
	width   int
	height  int

	query    string
	typing   bool
	matchRe  *regexp.Regexp
	matchIdx int
}

const (
	helpPageKeys = iota
	helpPageCommands
	helpPageCount
)

// NewHelpPanel creates a hidden help panel.
func NewHelpPanel() HelpPanel {
	return HelpPanel{}
}

// Toggle shows or hides the help panel. Opening resets to Keys at the top.
func (h *HelpPanel) Toggle() {
	if h.visible {
		h.visible = false
		return
	}
	h.Show()
}

// Show forces the panel visible, on the Keys page, scrolled to the top.
func (h *HelpPanel) Show() {
	h.visible = true
	h.page = helpPageKeys
	h.keysOff = 0
	h.cmdsOff = 0
	h.keysCur = 0
	h.cmdsCur = 0
	h.clearSearch()
}

// Hide forces the panel hidden.
func (h *HelpPanel) Hide() { h.visible = false }

// Visible reports whether the help panel is shown.
func (h HelpPanel) Visible() bool { return h.visible }

// SetSize stores the terminal dimensions and clamps the current page.
func (h *HelpPanel) SetSize(width, height int) {
	h.width = width
	h.height = height
	h.clampCursor()
	h.adjustScrollToCursor()
}

// Typing reports whether the / search prompt has focus.
func (h HelpPanel) Typing() bool { return h.typing }

// HandleKey routes a keypress to the open help overlay. It returns true when
// the key is consumed and the overlay should stay open. It returns false only
// for explicit close keys (esc with no search, ?, q, ctrl+c).
//
// Unmapped keys are consumed rather than dismissed: stray CSI / overscroll
// events must not close the panel.
func (h *HelpPanel) HandleKey(msg tea.KeyMsg) bool {
	if h.typing {
		switch msg.String() {
		case "esc", "ctrl+c":
			h.clearSearch()
			return true
		case "enter":
			h.typing = false
			h.scrollToCurrentMatch()
			return true
		case "backspace":
			if len(h.query) > 0 {
				h.query = h.query[:len(h.query)-1]
				h.afterQueryChange()
			}
			return true
		case "up", "down", "left", "right", "pgup", "pgdown", "home", "end":
			return true
		}
		if ch, ok := keyFilterChar(msg); ok {
			h.query += ch
			h.afterQueryChange()
			return true
		}
		return true
	}

	switch msg.String() {
	case "esc":
		if h.matchRe != nil {
			h.clearSearch()
			return true
		}
		return false
	case "?", "q", "ctrl+c":
		return false
	case "/":
		h.typing = true
		h.query = ""
		h.matchRe = nil
		h.matchIdx = 0
		return true
	case "tab":
		h.page = (h.page + 1) % helpPageCount
		h.afterPageChange()
		return true
	case "shift+tab":
		h.page = (h.page - 1 + helpPageCount) % helpPageCount
		h.afterPageChange()
		return true
	case "n":
		if h.matchRe != nil {
			h.advanceMatch(1)
		}
		return true
	case "N":
		if h.matchRe != nil {
			h.advanceMatch(-1)
		}
		return true
	case "j", "down":
		h.moveCursor(1)
		return true
	case "k", "up":
		h.moveCursor(-1)
		return true
	case "pgdown", "ctrl+d", " ", "f":
		h.moveCursor(h.scrollPage())
		return true
	case "pgup", "ctrl+u", "b":
		h.moveCursor(-h.scrollPage())
		return true
	case "g":
		h.setCurCursor(0)
		h.setCurOff(0)
		return true
	case "G":
		n := h.pageLineCount()
		if n > 0 {
			h.setCurCursor(n - 1)
		} else {
			h.setCurCursor(0)
		}
		h.adjustScrollToCursor()
		return true
	}
	return true
}

func (h HelpPanel) curOff() int {
	if h.page == helpPageCommands {
		return h.cmdsOff
	}
	return h.keysOff
}

func (h *HelpPanel) setCurOff(v int) {
	if v < 0 {
		v = 0
	}
	if max := h.maxOff(); v > max {
		v = max
	}
	if h.page == helpPageCommands {
		h.cmdsOff = v
		return
	}
	h.keysOff = v
}

func (h HelpPanel) curCursor() int {
	if h.page == helpPageCommands {
		return h.cmdsCur
	}
	return h.keysCur
}

func (h *HelpPanel) setCurCursor(v int) {
	if v < 0 {
		v = 0
	}
	if n := h.pageLineCount(); n > 0 && v >= n {
		v = n - 1
	} else if n == 0 {
		v = 0
	}
	if h.page == helpPageCommands {
		h.cmdsCur = v
		return
	}
	h.keysCur = v
}

func (h HelpPanel) pageLineCount() int {
	return len(h.pageRows(helpContentWidth(h.width)))
}

func (h *HelpPanel) clampCursor() {
	h.setCurCursor(h.curCursor())
}

func (h *HelpPanel) moveCursor(delta int) {
	h.setCurCursor(h.curCursor() + delta)
	h.adjustScrollToCursor()
}

func (h *HelpPanel) adjustScrollToCursor() {
	vp := h.scrollPage()
	cur := h.curCursor()
	off := h.curOff()
	if cur < off {
		h.setCurOff(cur)
		return
	}
	if cur >= off+vp {
		h.setCurOff(cur - vp + 1)
	}
}

func (h HelpPanel) maxOff() int {
	off := h.pageLineCount() - h.scrollPage()
	if off < 0 {
		return 0
	}
	return off
}

func (h HelpPanel) scrollPage() int {
	v := h.height - 2 - 2*helpPadY - 6
	if v < 4 {
		return 4
	}
	return v
}

// ScrollBy moves the active page's viewport by delta lines and keeps the
// line cursor inside the new viewport.
func (h *HelpPanel) ScrollBy(delta int) {
	h.setCurOff(h.curOff() + delta)
	off := h.curOff()
	vp := h.scrollPage()
	cur := h.curCursor()
	if cur < off {
		h.setCurCursor(off)
	} else if last := off + vp - 1; cur > last {
		h.setCurCursor(last)
	}
}

func (h *HelpPanel) rebuildMatchRe() {
	q := strings.TrimSpace(h.query)
	if q == "" {
		h.matchRe = nil
		return
	}
	h.matchRe = regexp.MustCompile("(?i)" + regexp.QuoteMeta(q))
}

func (h *HelpPanel) afterQueryChange() {
	h.rebuildMatchRe()
	h.matchIdx = 0
	h.scrollToCurrentMatch()
}

func (h *HelpPanel) afterPageChange() {
	h.matchIdx = 0
	h.scrollToCurrentMatch()
}

func (h *HelpPanel) clearSearch() {
	h.query = ""
	h.typing = false
	h.matchRe = nil
	h.matchIdx = 0
}

func (h HelpPanel) currentRows() []helpRow {
	return h.pageRows(helpContentWidth(h.width))
}

func (h HelpPanel) pageRows(contentW int) []helpRow {
	if h.page == helpPageCommands {
		return renderCommandsRows(contentW)
	}
	return renderKeysRows(contentW)
}

func (h HelpPanel) matches() []int {
	if h.matchRe == nil {
		return nil
	}
	rows := h.currentRows()
	var out []int
	for i, r := range rows {
		if h.matchRe.MatchString(r.searchText()) {
			out = append(out, i)
		}
	}
	return out
}

func (h HelpPanel) currentMatchLine() int {
	ms := h.matches()
	if len(ms) == 0 {
		return -1
	}
	idx := h.matchIdx
	if idx < 0 || idx >= len(ms) {
		idx = 0
	}
	return ms[idx]
}

func (h *HelpPanel) advanceMatch(dir int) {
	ms := h.matches()
	if len(ms) == 0 {
		return
	}
	if h.matchIdx < 0 || h.matchIdx >= len(ms) {
		h.matchIdx = 0
	}
	n := len(ms)
	h.matchIdx = ((h.matchIdx+dir)%n + n) % n
	h.scrollToCurrentMatch()
}

func (h *HelpPanel) scrollToCurrentMatch() {
	target := h.currentMatchLine()
	if target < 0 {
		return
	}
	h.setCurCursor(target)
	vp := h.scrollPage()
	off := target - vp/3
	if off < 0 {
		off = 0
	}
	h.setCurOff(off)
}

// View renders the help overlay, sized to fit the terminal.
func (h HelpPanel) View() string {
	if !h.visible || h.width == 0 || h.height == 0 {
		return ""
	}

	contentW := helpContentWidth(h.width)
	viewportH := h.scrollPage()

	rows := h.pageRows(contentW)
	curMatch := h.currentMatchLine()

	maxOff := len(rows) - viewportH
	if maxOff < 0 {
		maxOff = 0
	}
	off := h.curOff()
	if off > maxOff {
		off = maxOff
	}
	end := off + viewportH
	if end > len(rows) {
		end = len(rows)
	}

	var bodyVisible []string
	cursor := h.curCursor()
	for i := off; i < end; i++ {
		bodyVisible = append(bodyVisible, renderHelpRow(rows[i], h.matchRe, i == curMatch, i == cursor))
	}
	for len(bodyVisible) < viewportH {
		bodyVisible = append(bodyVisible, "")
	}
	body := strings.Join(bodyVisible, "\n")

	title := "Keybindings"
	if h.page == helpPageCommands {
		title = "Commands"
	}
	header := lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render(title)
	tabbar := h.renderTabBar()
	pos := h.renderStatusLine(off, end, len(rows), maxOff)

	layers := []string{header, "", tabbar, "", body, "", pos}
	content := lipgloss.JoinVertical(lipgloss.Left, layers...)

	return lipgloss.NewStyle().
		Width(h.width-2).
		Height(h.height-2).
		Border(panelBorder()).
		BorderForeground(colorPrimary).
		Padding(helpPadY, helpPadX).
		Render(content)
}

func (h HelpPanel) renderStatusLine(off, end, total, maxOff int) string {
	switch {
	case h.typing:
		return lipgloss.NewStyle().Foreground(colorPrimary).Render("/"+h.query) +
			lipgloss.NewStyle().Foreground(colorAccent).Underline(true).Render(" ")
	case h.matchRe != nil:
		ms := h.matches()
		if len(ms) == 0 {
			return styleMuted.Render("/" + h.query + "  no matches")
		}
		idx := h.matchIdx + 1
		if idx < 1 || idx > len(ms) {
			idx = 1
		}
		return styleMuted.Render("/" + h.query + "  match " + itoa(idx) + "/" + itoa(len(ms)) + "  (n/N)")
	default:
		if maxOff > 0 {
			pct := off * 100 / maxOff
			return styleMuted.Render("scroll " + itoa(off+1) + "–" + itoa(end) + "/" + itoa(total) + "  " + itoa(pct) + "%")
		}
		return ""
	}
}

const (
	helpPadX = 2
	helpPadY = 1
)

func helpTabLabel(i int) string {
	if i == helpPageCommands {
		return "Commands"
	}
	return "Keys"
}

func (h HelpPanel) renderTabBar() string {
	var parts []string
	for i := 0; i < helpPageCount; i++ {
		l := helpTabLabel(i)
		var s string
		if i == h.page {
			s = lipgloss.NewStyle().
				Bold(true).
				Background(colorPrimary).
				Foreground(colorBg).
				Render(" " + l + " ")
		} else {
			s = lipgloss.NewStyle().Foreground(colorMuted).Render(" " + l + " ")
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " ")
}

func helpContentWidth(termW int) int {
	w := termW - 2 - 2*helpPadX
	if w < 40 {
		w = 40
	}
	return w
}

// ── Row model ──────────────────────────────────────────────────────────────

type helpSegment struct {
	text  string
	style lipgloss.Style
}

type helpRow []helpSegment

func (r helpRow) searchText() string {
	var b strings.Builder
	for _, s := range r {
		b.WriteString(s.text)
	}
	return b.String()
}

func helpSearchStyles() (match, curMatch lipgloss.Style) {
	return lipgloss.NewStyle().Background(colorSearchMatch).Foreground(colorFg),
		lipgloss.NewStyle().Background(colorPrimary).Foreground(colorBg).Bold(true)
}

func renderHelpRow(row helpRow, re *regexp.Regexp, isCurrent, isCursor bool) string {
	match, curMatch := helpSearchStyles()
	cursorStyle := lipgloss.NewStyle().Background(colorPrimary).Foreground(colorBg).Inline(true)
	var b strings.Builder
	appliedCursor := !isCursor
	for _, seg := range row {
		if !appliedCursor && seg.text != "" {
			r, size := utf8.DecodeRuneInString(seg.text)
			if r == utf8.RuneError && size == 1 {
				size = 1
			}
			b.WriteString(cursorStyle.Render(string(r)))
			rest := seg.text[size:]
			if rest != "" {
				b.WriteString(renderHelpSegment(helpSegment{text: rest, style: seg.style}, re, isCurrent, match, curMatch))
			}
			appliedCursor = true
			continue
		}
		b.WriteString(renderHelpSegment(seg, re, isCurrent, match, curMatch))
	}
	if !appliedCursor {
		b.WriteString(cursorStyle.Render(" "))
	}
	return b.String()
}

func renderHelpSegment(seg helpSegment, re *regexp.Regexp, isCurrent bool, match, curMatch lipgloss.Style) string {
	base := seg.style
	m := match
	if isCurrent {
		m = curMatch
	}
	if re == nil {
		return base.Render(seg.text)
	}
	locs := re.FindAllStringIndex(seg.text, -1)
	if len(locs) == 0 {
		return base.Render(seg.text)
	}
	var b strings.Builder
	prev := 0
	for _, loc := range locs {
		if loc[0] > prev {
			b.WriteString(base.Render(seg.text[prev:loc[0]]))
		}
		b.WriteString(m.Render(seg.text[loc[0]:loc[1]]))
		prev = loc[1]
	}
	if prev < len(seg.text) {
		b.WriteString(base.Render(seg.text[prev:]))
	}
	return b.String()
}

func renderKeysRows(contentW int) []helpRow {
	sections := registry()
	keyW := 0
	for _, s := range sections {
		for _, b := range s.Items {
			if w := runeLen(b.Display); w > keyW {
				keyW = w
			}
		}
	}
	const gap = 4
	descW := contentW - keyW - gap - 2
	if descW < 8 {
		descW = 8
	}
	labelStyle := lipgloss.NewStyle().Foreground(colorLabel)
	fgStyle := lipgloss.NewStyle().Foreground(colorFg)
	secTitleStyle := lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	muted := lipgloss.NewStyle().Foreground(colorMuted)
	plain := lipgloss.NewStyle()
	var out []helpRow
	for _, s := range sections {
		out = append(out, helpRow{{text: s.Title, style: secTitleStyle}})
		for _, bd := range s.Items {
			key := bd.Display + strings.Repeat(" ", max(0, keyW-runeLen(bd.Display)))
			desc := truncateRunes(bd.Desc, descW)
			out = append(out, helpRow{
				{text: "  ", style: plain},
				{text: key, style: labelStyle},
				{text: strings.Repeat(" ", gap), style: plain},
				{text: desc, style: fgStyle},
			})
		}
		out = append(out, helpRow{})
	}
	out = append(out, helpRow{{text: "  tab — switch tabs · / — search this page", style: muted}})
	return out
}

func renderCommandsRows(contentW int) []helpRow {
	cmds := exCommands()
	usageW := 0
	for _, c := range cmds {
		if w := runeLen(c.usage); w > usageW {
			usageW = w
		}
	}
	const gap = 4
	descW := contentW - usageW - gap - 2
	if descW < 8 {
		descW = 8
	}
	indent := strings.Repeat(" ", 2+usageW+gap)
	labelStyle := lipgloss.NewStyle().Foreground(colorLabel)
	fgStyle := lipgloss.NewStyle().Foreground(colorFg)
	plain := lipgloss.NewStyle()

	var out []helpRow
	for _, c := range cmds {
		usage := c.usage + strings.Repeat(" ", max(0, usageW-runeLen(c.usage)))
		wrapped := wrapRunes(c.desc, descW)
		for i, w := range wrapped {
			if i == 0 {
				out = append(out, helpRow{
					{text: "  ", style: plain},
					{text: usage, style: labelStyle},
					{text: strings.Repeat(" ", gap), style: plain},
					{text: w, style: fgStyle},
				})
			} else {
				out = append(out, helpRow{
					{text: indent, style: plain},
					{text: w, style: fgStyle},
				})
			}
		}
	}
	return out
}

func wrapRunes(s string, width int) []string {
	if width <= 1 {
		return []string{s}
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		if runeLen(cur)+1+runeLen(w) <= width {
			cur += " " + w
		} else {
			lines = append(lines, cur)
			cur = w
		}
	}
	lines = append(lines, cur)
	return lines
}

func runeLen(s string) int { return utf8.RuneCountInString(s) }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
