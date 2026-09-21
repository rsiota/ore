package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/rsiota/ore/internal/bookmarks"
)

// bookmarkPanel is a fuzzy overlay for saved named views (creel-style).
type bookmarkPanel struct {
	visible  bool
	input    string
	cursor   int
	items    []bookmarks.Bookmark // newest first for display
	filtered []bookmarks.Bookmark
	origIdx  []int // filtered[i] → index in items
}

const maxBookmarkPanelItems = 12

func (p *bookmarkPanel) Open(entries []bookmarks.Bookmark) {
	p.visible = true
	p.input = ""
	p.cursor = 0
	// Newest first (creel).
	p.items = make([]bookmarks.Bookmark, len(entries))
	for i, e := range entries {
		p.items[len(entries)-1-i] = e
	}
	p.refilter()
}

func (p *bookmarkPanel) Hide() { p.visible = false }

func (p bookmarkPanel) IsVisible() bool { return p.visible }

func (p *bookmarkPanel) SetEntries(entries []bookmarks.Bookmark) {
	p.items = make([]bookmarks.Bookmark, len(entries))
	for i, e := range entries {
		p.items[len(entries)-1-i] = e
	}
	p.refilter()
	if p.cursor >= len(p.filtered) {
		p.cursor = max(0, len(p.filtered)-1)
	}
}

func (p *bookmarkPanel) refilter() {
	if p.input == "" {
		p.filtered = append([]bookmarks.Bookmark(nil), p.items...)
		p.origIdx = make([]int, len(p.items))
		for i := range p.items {
			p.origIdx[i] = i
		}
		if p.cursor >= len(p.filtered) {
			p.cursor = max(0, len(p.filtered)-1)
		}
		return
	}
	ranked := fuzzyRank(p.input, p.items,
		func(b bookmarks.Bookmark) string {
			return b.Label() + " " + bookmarks.ViewSummary(b.View)
		},
		func(a, b fuzzyResult[bookmarks.Bookmark]) bool {
			return a.Item.SavedAt.After(b.Item.SavedAt)
		})
	p.filtered = make([]bookmarks.Bookmark, len(ranked))
	p.origIdx = make([]int, len(ranked))
	for i, r := range ranked {
		p.filtered[i] = r.Item
		p.origIdx[i] = r.Index
	}
	if p.cursor >= len(p.filtered) {
		p.cursor = max(0, len(p.filtered)-1)
	}
}

func (p *bookmarkPanel) moveCursor(delta int) {
	n := len(p.filtered)
	if n == 0 {
		return
	}
	p.cursor = (p.cursor + delta + n) % n
}

func (p bookmarkPanel) selected() (bookmarks.Bookmark, bool) {
	if p.cursor < 0 || p.cursor >= len(p.filtered) {
		return bookmarks.Bookmark{}, false
	}
	return p.filtered[p.cursor], true
}

// selectedStoreIndex returns the oldest-first index in the on-disk list, or -1.
func (p bookmarkPanel) selectedStoreIndex(total int) int {
	if p.cursor < 0 || p.cursor >= len(p.origIdx) || total <= 0 {
		return -1
	}
	disp := p.origIdx[p.cursor]
	return total - 1 - disp
}

type bookmarkPickedMsg struct {
	bm bookmarks.Bookmark
}

func (p bookmarkPanel) Update(msg tea.KeyMsg) (bookmarkPanel, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		p.visible = false
		return p, nil
	case "enter":
		b, ok := p.selected()
		p.visible = false
		if !ok {
			return p, nil
		}
		return p, func() tea.Msg { return bookmarkPickedMsg{bm: b} }
	case "up", "k", "ctrl+p":
		p.moveCursor(-1)
		return p, nil
	case "down", "j", "ctrl+n":
		p.moveCursor(1)
		return p, nil
	case "d":
		// Deletion is handled by the model (needs store + reload).
		return p, func() tea.Msg { return bookmarkDeleteReqMsg{} }
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

type bookmarkDeleteReqMsg struct{}

func bookmarkPopupDim() (w, h int) {
	return 72, maxBookmarkPanelItems + 4
}

// View renders the bookmark panel — caller overlays it centered.
func (p bookmarkPanel) View(width, height int) string {
	if !p.visible {
		return ""
	}
	innerW := width - 4
	if innerW < 24 {
		innerW = 24
	}

	start := 0
	if p.cursor >= maxBookmarkPanelItems {
		start = p.cursor - maxBookmarkPanelItems + 1
	}
	end := start + maxBookmarkPanelItems
	if end > len(p.filtered) {
		end = len(p.filtered)
	}

	var lines []string
	for i := start; i < end; i++ {
		lines = append(lines, renderBookmarkItem(p.filtered[i], innerW, i == p.cursor))
	}
	if len(lines) == 0 {
		lines = append(lines, styleMuted.Render("  no bookmarks — m to save"))
	}
	for len(lines) < maxBookmarkPanelItems {
		lines = append(lines, "")
	}

	preview := "  " + styleMuted.Render("enter jump · d delete · esc close")
	if b, ok := p.selected(); ok {
		preview = "  " + styleMuted.Render(runewidth.Truncate(
			bookmarks.FormatTime(b.SavedAt)+" · "+bookmarks.ViewSummary(b.View),
			innerW-2, "…"))
	}
	lines = append(lines, preview)

	prompt := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render("❯ ") + lipgloss.NewStyle().Foreground(colorPrimary).Render(p.input)
	body := prompt + "\n" + strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Width(width - 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(0, 1).
		Render(body)
}

func renderBookmarkItem(b bookmarks.Bookmark, width int, selected bool) string {
	label := runewidth.Truncate(b.Label(), max(1, width-2), "…")
	if selected {
		return lipgloss.NewStyle().
			Foreground(colorBg).
			Background(colorPrimary).
			Render(runewidth.FillRight("❯ "+label, width))
	}
	return lipgloss.NewStyle().Foreground(colorPrimary).Render(runewidth.FillRight("  "+label, width))
}
