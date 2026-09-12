package ui

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// paletteItem is one searchable row in the Ctrl+P command palette.
type paletteItem struct {
	display string
	desc    string
	section string
	replay  []string // key sequence; nil unless a binding
	exLine  string   // ":" command verb (no leading colon); empty unless Commands
	exArgs  bool     // true → open ex line prefilled for args
}

// paletteExMsg runs or prefills an ex command chosen from the palette.
type paletteExMsg struct {
	line   string
	typing bool // true → open : prompt with line as prefix
}

// palette is the fuzzy-searchable command overlay (Ctrl+P).
type palette struct {
	visible  bool
	input    string
	cursor   int
	items    []paletteItem
	filtered []paletteItem
}

const maxPaletteItems = 16

func (p *palette) Open() {
	p.visible = true
	p.input = ""
	p.cursor = 0
	p.items = buildPaletteItems()
	sortPaletteItems(p.items)
	p.filtered = append([]paletteItem(nil), p.items...)
}

func (p *palette) Hide() { p.visible = false }

func (p palette) IsVisible() bool { return p.visible }

func buildPaletteItems() []paletteItem {
	var items []paletteItem
	for _, cmd := range exCommands() {
		verb := cmd.verbs[0]
		items = append(items, paletteItem{
			display: cmd.usage,
			desc:    cmd.desc,
			section: "Commands",
			exLine:  verb,
			exArgs:  strings.Contains(cmd.usage, "<"),
		})
	}
	for _, sec := range registry() {
		for _, b := range sec.Items {
			replay := b.replayTokens()
			if replay == nil && b.Tokens == nil {
				// View descriptions — searchable but not executable.
				items = append(items, paletteItem{
					display: b.Display,
					desc:    b.Desc,
					section: sec.Title,
				})
				continue
			}
			if replay == nil {
				continue
			}
			items = append(items, paletteItem{
				display: b.Display,
				desc:    b.Desc,
				section: sec.Title,
				replay:  replay,
			})
		}
	}
	return items
}

func sortPaletteItems(items []paletteItem) {
	rank := func(section string) int {
		if section == "Commands" {
			return 0
		}
		for i, sec := range registry() {
			if sec.Title == section {
				return i + 1
			}
		}
		return len(registry()) + 1
	}
	sort.SliceStable(items, func(i, j int) bool {
		ri, rj := rank(items[i].section), rank(items[j].section)
		if ri != rj {
			return ri < rj
		}
		return items[i].desc < items[j].desc
	})
}

// chordReplays maps binding Display → key sequence for g-chords / doubles.
var chordReplays = map[string][]string{
	"g r":     {"g", "r"},
	"g g / G": {"g", "g"},
	"f / g f": {"f"},
}

// replayTokens returns the key sequence the palette should replay, or nil.
func (b Binding) replayTokens() []string {
	if seq, ok := chordReplays[b.Display]; ok {
		return seq
	}
	if len(b.Tokens) == 0 {
		return nil
	}
	if len(b.Tokens) == 1 {
		return []string{b.Tokens[0]}
	}
	// Alternatives documented as "a / b" — pick the first token.
	if strings.Contains(b.Display, "/") {
		return []string{b.Tokens[0]}
	}
	return nil
}

func (p *palette) refilter() {
	if p.input == "" {
		p.filtered = append([]paletteItem(nil), p.items...)
		sortPaletteItems(p.filtered)
		p.cursor = 0
		return
	}
	ranked := fuzzyRank(p.input, p.items,
		func(it paletteItem) string {
			return it.desc + " " + it.display + " " + it.section + " " + it.exLine
		},
		nil)
	p.filtered = make([]paletteItem, len(ranked))
	for i, r := range ranked {
		p.filtered[i] = r.Item
	}
	if p.cursor >= len(p.filtered) {
		p.cursor = max(0, len(p.filtered)-1)
	}
}

func (p *palette) moveCursor(delta int) {
	n := len(p.filtered)
	if n == 0 {
		return
	}
	p.cursor = (p.cursor + delta + n) % n
}

func (p palette) selectedItem() paletteItem {
	if p.cursor < 0 || p.cursor >= len(p.filtered) {
		return paletteItem{}
	}
	return p.filtered[p.cursor]
}

func (p palette) Update(msg tea.KeyMsg) (palette, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		p.visible = false
		return p, nil
	case "enter":
		it := p.selectedItem()
		p.visible = false
		if it.exLine != "" {
			line := it.exLine
			if it.exArgs {
				line += " "
				return p, func() tea.Msg { return paletteExMsg{line: line, typing: true} }
			}
			return p, func() tea.Msg { return paletteExMsg{line: line} }
		}
		if len(it.replay) == 0 {
			return p, nil
		}
		return p, replayKeySequence(it.replay)
	case "up", "ctrl+p", "k":
		p.moveCursor(-1)
		return p, nil
	case "down", "ctrl+n", "j":
		p.moveCursor(1)
		return p, nil
	case "backspace":
		if p.input != "" {
			r := []rune(p.input)
			p.input = string(r[:len(r)-1])
			p.refilter()
		}
		return p, nil
	case "ctrl+u":
		p.input = ""
		p.refilter()
		return p, nil
	}
	if ch, ok := keyFilterChar(msg); ok {
		p.input += ch
		p.refilter()
	}
	return p, nil
}

// View renders the palette panel only — the caller overlays it centered.
func (p palette) View(width, height int) string {
	if !p.visible {
		return ""
	}

	// Inner content width: panel Width(width-2) with Padding(0, 1).
	innerW := width - 4
	if innerW < 24 {
		innerW = 24
	}

	keyW, descW, secW := paletteColumnWidths(innerW)

	start := 0
	if p.cursor >= maxPaletteItems {
		start = p.cursor - maxPaletteItems + 1
	}
	end := start + maxPaletteItems
	if end > len(p.filtered) {
		end = len(p.filtered)
	}

	var lines []string
	for i := start; i < end; i++ {
		lines = append(lines, renderPaletteItemLine(p.filtered[i], keyW, descW, secW, i == p.cursor))
	}
	if len(lines) == 0 {
		lines = append(lines, styleMuted.Render("  no matches"))
	}
	// Pad to a fixed row count so the panel height never changes.
	for len(lines) < maxPaletteItems {
		lines = append(lines, "")
	}

	prompt := renderPalettePrompt(p.input)
	body := prompt + "\n" + strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Width(width-2).
		Height(height-2).
		Border(panelBorder()).
		BorderForeground(colorPrimary).
		Padding(0, 1).
		Render(body)
}

const (
	paletteKeyColW = 12
	paletteSecColW = 16
)

func paletteColumnWidths(innerW int) (keyW, descW, secW int) {
	const (
		prefix   = 2
		colGap   = 2
		minDescW = 8
	)
	keyW = paletteKeyColW
	secW = paletteSecColW
	descW = innerW - (prefix + keyW + colGap + secW + colGap)
	if descW < minDescW {
		descW = minDescW
	}
	return keyW, descW, secW
}

func renderPaletteItemLine(it paletteItem, keyW, descW, secW int, selected bool) string {
	const colGap = 2
	prefix := "  "
	if selected {
		prefix = "❯ "
	}
	gap := strings.Repeat(" ", colGap)
	key := padPalette(clampPaletteText(it.display, keyW), keyW)
	desc := padPalette(clampPaletteText(it.desc, descW), descW)
	sec := clampPaletteText(it.section, secW)
	secPad := secW - runewidth.StringWidth(sec)
	if secPad < 0 {
		secPad = 0
	}
	sec = strings.Repeat(" ", secPad) + sec
	full := prefix + key + gap + desc + gap + sec
	if selected {
		return lipgloss.NewStyle().
			Background(colorPrimary).
			Foreground(colorBg).
			Render(full)
	}
	keyStr := lipgloss.NewStyle().Foreground(colorPrimary).Render(key)
	descStr := lipgloss.NewStyle().Foreground(lipgloss.Color("#1f2328")).Render(desc)
	secStr := styleMuted.Render(sec)
	return prefix + keyStr + gap + descStr + gap + secStr
}

func renderPalettePrompt(input string) string {
	chevron := lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render("❯ ")
	text := lipgloss.NewStyle().Foreground(colorPrimary).Render(input)
	cursor := lipgloss.NewStyle().Underline(true).Render(" ")
	return chevron + text + cursor
}

func clampPaletteText(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= width {
		return s
	}
	return runewidth.Truncate(s, width, "…")
}

func padPalette(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
