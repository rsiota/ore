package ui

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripAnsi(s string) string { return ansiRe.ReplaceAllString(s, "") }

func helpKey(s string) tea.KeyMsg {
	switch s {
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "pgdown":
		return tea.KeyMsg{Type: tea.KeyPgDown}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
}

func TestHelpListsKeysAndCommands(t *testing.T) {
	h := NewHelpPanel()
	h.Show()
	h.SetSize(120, 40)
	out := stripAnsi(h.View())
	if !strings.Contains(out, "Keybindings") {
		t.Error("Keys page should show the Keybindings header")
	}
	if !strings.Contains(out, "Global") {
		t.Error("Keys page should list the Global section")
	}
	if h.page != helpPageKeys {
		t.Errorf("Show() page = %d, want helpPageKeys", h.page)
	}

	h.page = helpPageCommands
	var body strings.Builder
	for _, row := range renderCommandsRows(120) {
		body.WriteString(row.searchText())
		body.WriteByte('\n')
	}
	for _, want := range []string{":blame", ":goto", ":theme", ":help"} {
		if !strings.Contains(body.String(), want) {
			t.Errorf("Commands rows missing %q", want)
		}
	}
}

func TestHelpTabSwitchAndScroll(t *testing.T) {
	h := NewHelpPanel()
	h.Show()
	h.SetSize(120, 40)

	keys := stripAnsi(h.View())
	if !strings.Contains(keys, "Keybindings") {
		t.Error("Keys page should show the Keybindings header")
	}
	if strings.Contains(keys, ":goto <hash>") {
		t.Error("command usages should not appear on the Keys page")
	}

	if !h.HandleKey(helpKey("tab")) {
		t.Fatal("tab should be consumed to switch help pages")
	}
	if !strings.Contains(stripAnsi(h.View()), ":goto <hash>") {
		t.Error("Commands page should list command usages")
	}

	h.page = helpPageKeys
	h.keysOff = 0
	h.keysCur = 0
	if !strings.Contains(stripAnsi(h.View()), "Global") {
		t.Error("Global should be visible at the top of the Keys page")
	}
	for i := 0; i < 3; i++ {
		h.HandleKey(helpKey("pgdown"))
	}
	if strings.Contains(stripAnsi(h.View()), "Global") {
		t.Error("Global should have scrolled off the top after paging down")
	}
}

func TestHelpCursorAndStatusLine(t *testing.T) {
	h := NewHelpPanel()
	h.Show()
	h.SetSize(120, 40)
	out := stripAnsi(h.View())
	readout := regexp.MustCompile(`scroll \d+–\d+/\d+`)
	if !readout.MatchString(out) {
		t.Fatalf("status line should show scroll a–b/total, got %q", out[max(0, len(out)-80):])
	}

	if h.curCursor() != 0 {
		t.Fatalf("cursor = %d, want 0", h.curCursor())
	}
	h.HandleKey(helpKey("j"))
	if h.curCursor() != 1 {
		t.Fatalf("after j, cursor = %d, want 1", h.curCursor())
	}
	h.HandleKey(helpKey("G"))
	last := h.pageLineCount() - 1
	if h.curCursor() != last {
		t.Fatalf("after G, cursor = %d, want %d", h.curCursor(), last)
	}
	if h.curOff() != h.maxOff() {
		t.Fatalf("after G, offset = %d, want maxOff %d", h.curOff(), h.maxOff())
	}
	bottom := stripAnsi(h.View())
	if !strings.Contains(bottom, "100%") {
		t.Error("status line should report 100% at the bottom")
	}
}

func TestHelpCloseKeysDismisses(t *testing.T) {
	h := NewHelpPanel()
	h.Show()
	for _, k := range []string{"esc", "?", "q"} {
		if h.HandleKey(helpKey(k)) {
			t.Errorf("%s should not be consumed (caller closes)", k)
		}
	}
	if !h.HandleKey(helpKey("j")) {
		t.Error("j should be consumed")
	}
}

func TestHelpMouseScroll(t *testing.T) {
	h := NewHelpPanel()
	h.Show()
	h.SetSize(120, 40)
	if !strings.Contains(stripAnsi(h.View()), "Global") {
		t.Fatal("Global should be visible at the top initially")
	}
	h.ScrollBy(1000)
	if strings.Contains(stripAnsi(h.View()), "Global") {
		t.Error("Global should scroll off after scrolling down")
	}
	h.ScrollBy(-1000)
	if !strings.Contains(stripAnsi(h.View()), "Global") {
		t.Error("Global should reappear after scrolling back to the top")
	}
}
