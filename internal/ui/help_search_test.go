package ui

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestHelpRowSearchText(t *testing.T) {
	row := helpRow{
		{text: "  ", style: lipgloss.NewStyle()},
		{text: "x", style: lipgloss.NewStyle().Foreground(colorLabel)},
		{text: "    ", style: lipgloss.NewStyle()},
		{text: "export to CSV", style: lipgloss.NewStyle().Foreground(colorFg)},
	}
	if got := row.searchText(); got != "  x    export to CSV" {
		t.Errorf("searchText=%q", got)
	}
}

func TestHelpSearchOpensTyping(t *testing.T) {
	h := NewHelpPanel()
	h.Show()
	h.SetSize(120, 40)

	if !h.HandleKey(helpKey("/")) {
		t.Fatal("/ should be consumed")
	}
	if !h.Typing() {
		t.Fatal("expected typing mode after /")
	}
}

func TestHelpSearchTypingFindsMatches(t *testing.T) {
	h := NewHelpPanel()
	h.Show()
	h.page = helpPageKeys
	h.SetSize(120, 40)

	h.HandleKey(helpKey("/"))
	for _, r := range "blame" {
		h.HandleKey(helpKey(string(r)))
	}
	if h.query != "blame" {
		t.Fatalf("query=%q want blame", h.query)
	}
	if h.matchRe == nil {
		t.Fatal("matchRe should be compiled for a non-empty query")
	}
	if ms := h.matches(); len(ms) < 2 {
		t.Fatalf("expected >=2 'blame' matches on Keys page, got %d", len(ms))
	}
}

func TestHelpSearchAdvanceAndWrap(t *testing.T) {
	h := NewHelpPanel()
	h.Show()
	h.page = helpPageKeys
	h.SetSize(120, 40)
	h.query = "blame"
	h.rebuildMatchRe()
	ms := h.matches()
	if len(ms) < 2 {
		t.Fatalf("need >=2 matches, got %d", len(ms))
	}

	h.scrollToCurrentMatch()
	first := h.currentMatchLine()
	if first != ms[0] {
		t.Fatalf("first match line=%d want %d", first, ms[0])
	}

	for i := 0; i < len(ms); i++ {
		h.HandleKey(helpKey("n"))
	}
	if got := h.currentMatchLine(); got != first {
		t.Errorf("after wrapping n×%d, match line=%d want %d", len(ms), got, first)
	}

	h.HandleKey(helpKey("n"))
	second := h.currentMatchLine()
	if second != ms[1] {
		t.Errorf("after one n, match line=%d want %d", second, ms[1])
	}
	h.HandleKey(helpKey("N"))
	if got := h.currentMatchLine(); got != first {
		t.Errorf("after N, match line=%d want %d (back to first)", got, first)
	}

	h.HandleKey(helpKey("N"))
	if got := h.currentMatchLine(); got != ms[len(ms)-1] {
		t.Errorf("N wrap, match line=%d want %d (last)", got, ms[len(ms)-1])
	}
}

func TestHelpSearchNoOpWithoutQuery(t *testing.T) {
	h := NewHelpPanel()
	h.Show()
	h.SetSize(120, 40)
	if !h.HandleKey(helpKey("n")) {
		t.Error("n with no query should be consumed")
	}
	if !h.HandleKey(helpKey("N")) {
		t.Error("N with no query should be consumed")
	}
}

func TestHelpSearchEscClearsThenCloses(t *testing.T) {
	h := NewHelpPanel()
	h.Show()
	h.SetSize(120, 40)
	h.HandleKey(helpKey("/"))
	for _, r := range "blame" {
		h.HandleKey(helpKey(string(r)))
	}
	if !h.HandleKey(tea.KeyMsg{Type: tea.KeyEsc}) {
		t.Fatal("esc while typing should be consumed")
	}
	if h.Typing() || h.query != "" || h.matchRe != nil {
		t.Errorf("esc did not clear search: typing=%v query=%q", h.Typing(), h.query)
	}
	if h.HandleKey(tea.KeyMsg{Type: tea.KeyEsc}) {
		t.Error("esc while not typing should not be consumed (caller closes)")
	}
}

func TestHelpSearchCommittedEscClearsThenCloses(t *testing.T) {
	h := NewHelpPanel()
	h.Show()
	h.SetSize(120, 40)
	h.HandleKey(helpKey("/"))
	for _, r := range "blame" {
		h.HandleKey(helpKey(string(r)))
	}
	if !h.HandleKey(helpKey("enter")) {
		t.Fatal("enter should be consumed")
	}
	if h.Typing() || h.matchRe == nil {
		t.Fatal("enter should leave a committed (non-typing) search active")
	}

	if !h.HandleKey(tea.KeyMsg{Type: tea.KeyEsc}) {
		t.Error("esc with an active search should be consumed (panel stays open)")
	}
	if h.matchRe != nil || h.query != "" {
		t.Errorf("esc did not clear committed search: query=%q", h.query)
	}

	if h.HandleKey(tea.KeyMsg{Type: tea.KeyEsc}) {
		t.Error("esc with no search should not be consumed (caller closes)")
	}
}

func TestHelpSearchEnterKeepsQuery(t *testing.T) {
	h := NewHelpPanel()
	h.Show()
	h.SetSize(120, 40)
	h.HandleKey(helpKey("/"))
	for _, r := range "tab" {
		h.HandleKey(helpKey(string(r)))
	}
	if !h.HandleKey(helpKey("enter")) {
		t.Fatal("enter should be consumed")
	}
	if h.Typing() {
		t.Error("enter should exit typing mode")
	}
	if h.query != "tab" {
		t.Errorf("query should persist after enter, got %q", h.query)
	}
	if len(h.matches()) > 1 {
		before := h.currentMatchLine()
		h.HandleKey(helpKey("n"))
		if h.currentMatchLine() == before {
			t.Error("n should advance after enter")
		}
	}
}

func TestHelpSearchCurrentPageOnly(t *testing.T) {
	h := NewHelpPanel()
	h.Show()
	h.SetSize(140, 50)

	h.page = helpPageKeys
	h.query = ":goto"
	h.rebuildMatchRe()
	if n := len(h.matches()); n != 0 {
		t.Errorf(":goto should not match the Keys page, got %d", n)
	}

	h.page = helpPageCommands
	if n := len(h.matches()); n == 0 {
		t.Error(":goto should match the Commands page")
	}
}

func TestHelpRenderHighlightPreservesText(t *testing.T) {
	row := helpRow{{text: "export to CSV", style: lipgloss.NewStyle().Foreground(colorFg)}}
	re := regexp.MustCompile("(?i)export")

	if got := stripAnsi(renderHelpRow(row, re, false, false)); !strings.Contains(got, "export to CSV") {
		t.Errorf("highlight lost text: %q", got)
	}
	cur := renderHelpRow(row, re, true, false)
	if w := lipgloss.Width(cur); w != 13 {
		t.Errorf("current-match line width=%d, want 13 (no full-line bar)", w)
	}
	if got := stripAnsi(cur); !strings.Contains(got, "export to CSV") {
		t.Errorf("highlight lost text on current match: %q", got)
	}
}

func TestHelpStatusLineMatchCount(t *testing.T) {
	readout := regexp.MustCompile(`match \d+/\d+`)
	h := NewHelpPanel()
	h.Show()
	h.SetSize(120, 40)

	h.query = "blame"
	h.rebuildMatchRe()
	if !readout.MatchString(stripAnsi(h.View())) {
		t.Error("status line should show a 'match n/N' readout while searching")
	}

	h.clearSearch()
	if readout.MatchString(stripAnsi(h.View())) {
		t.Error("status line should not show a match readout without a search")
	}
}

func TestHelpStatusLineNoMatches(t *testing.T) {
	h := NewHelpPanel()
	h.Show()
	h.SetSize(120, 40)
	h.query = "zzqqxxnope"
	h.rebuildMatchRe()
	if !strings.Contains(stripAnsi(h.View()), "no matches") {
		t.Error("status line should report 'no matches'")
	}
}

func TestHelpSearchScrollsMatchIntoView(t *testing.T) {
	h := NewHelpPanel()
	h.Show()
	h.SetSize(120, 40)

	h.HandleKey(helpKey("/"))
	for _, r := range "blame" {
		h.HandleKey(helpKey(string(r)))
	}
	ms := h.matches()
	if len(ms) == 0 {
		t.Fatal("expected matches")
	}
	target := ms[0]
	vp := h.scrollPage()
	off := h.curOff()
	if target < off || target >= off+vp {
		t.Errorf("first match line %d not in viewport [%d,%d)", target, off, off+vp)
	}
}
