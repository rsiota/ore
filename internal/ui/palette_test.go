package ui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestPaletteOpenClose(t *testing.T) {
	var p palette
	if p.IsVisible() {
		t.Fatal("palette should start hidden")
	}
	p.Open()
	if !p.IsVisible() {
		t.Fatal("palette should be visible after Open")
	}
	if len(p.items) == 0 {
		t.Fatal("expected items from registry + commands")
	}
	if len(p.filtered) != len(p.items) {
		t.Fatalf("filtered=%d items=%d", len(p.filtered), len(p.items))
	}
	p.Hide()
	if p.IsVisible() {
		t.Fatal("palette should hide")
	}
}

func TestPaletteFuzzyFilter(t *testing.T) {
	var p palette
	p.Open()
	simulatePaletteTyping(&p, "blame")
	if len(p.filtered) == 0 {
		t.Fatal("expected matches for blame")
	}
	found := false
	for _, it := range p.filtered {
		if strings.Contains(strings.ToLower(it.desc+it.display+it.exLine), "blame") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no filtered item mentions blame")
	}
	p.input = ""
	p.refilter()
	simulatePaletteTyping(&p, "zzzzz")
	if len(p.filtered) != 0 {
		t.Fatalf("expected no matches, got %d", len(p.filtered))
	}
}

func TestPaletteNavigationWraps(t *testing.T) {
	var p palette
	p.Open()
	p.cursor = 0
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.cursor != len(p.filtered)-1 {
		t.Fatalf("cursor = %d, want %d", p.cursor, len(p.filtered)-1)
	}
}

func TestPaletteEscape(t *testing.T) {
	var p palette
	p.Open()
	simulatePaletteTyping(&p, "hi")
	p, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if p.IsVisible() {
		t.Fatal("should hide on esc")
	}
	if cmd != nil {
		t.Fatal("esc should not emit a cmd")
	}
}

func TestPaletteEnterExNoArgs(t *testing.T) {
	var p palette
	p.Open()
	if !paletteSetCursor(&p, func(it paletteItem) bool {
		return it.exLine == "refresh" && !it.exArgs
	}) {
		t.Fatal("refresh command missing")
	}
	p, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if p.IsVisible() {
		t.Fatal("should hide on enter")
	}
	if cmd == nil {
		t.Fatal("expected paletteExMsg cmd")
	}
	msg := cmd()
	ex, ok := msg.(paletteExMsg)
	if !ok || ex.line != "refresh" || ex.typing {
		t.Fatalf("got %#v", msg)
	}
}

func TestPaletteEnterExNeedsArgs(t *testing.T) {
	var p palette
	p.Open()
	if !paletteSetCursor(&p, func(it paletteItem) bool {
		return it.exLine == "blame" && it.exArgs
	}) {
		t.Fatal("blame command missing")
	}
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	ex, ok := msg.(paletteExMsg)
	if !ok || ex.line != "blame " || !ex.typing {
		t.Fatalf("got %#v, want typing blame ", msg)
	}
}

func TestPaletteEnterReplaysBinding(t *testing.T) {
	var p palette
	p.Open()
	if !paletteSetCursor(&p, func(it paletteItem) bool {
		return len(it.replay) == 1 && it.replay[0] == "ctrl+r"
	}) {
		t.Fatal("ctrl+r binding missing")
	}
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected replay cmd")
	}
}

func TestPaletteChordsExecutable(t *testing.T) {
	var p palette
	p.Open()
	want := map[string][]string{
		"g r":     {"g", "r"},
		"g g / G": {"g", "g"},
		"f / g f": {"f"},
	}
	seen := map[string]bool{}
	for _, it := range p.items {
		if exp, ok := want[it.display]; ok {
			seen[it.display] = true
			if !reflect.DeepEqual(it.replay, exp) {
				t.Errorf("%q replay = %v, want %v", it.display, it.replay, exp)
			}
		}
	}
	for k := range want {
		if !seen[k] {
			t.Errorf("missing palette item %q", k)
		}
	}
}

func TestChordReplaysAreRealBindings(t *testing.T) {
	displays := map[string]bool{}
	for _, sec := range registry() {
		for _, b := range sec.Items {
			displays[b.Display] = true
		}
	}
	for k := range chordReplays {
		if !displays[k] {
			t.Errorf("chordReplays key %q is not a binding Display", k)
		}
	}
}

func TestPaletteView(t *testing.T) {
	var p palette
	p.Open()
	pw, ph := palettePopupDim()
	out := p.View(pw, ph)
	if out == "" || !strings.Contains(out, "❯") {
		t.Fatalf("unexpected view: %q", out)
	}
	if w := lipgloss.Width(out); w != pw {
		t.Fatalf("palette width = %d, want %d", w, pw)
	}
	if h := lipgloss.Height(out); h != ph {
		t.Fatalf("palette height = %d, want %d", h, ph)
	}
	p.Hide()
	if p.View(pw, ph) != "" {
		t.Fatal("hidden view should be empty")
	}
}

func TestPalettePopupDimMatchesCreel(t *testing.T) {
	w, h := palettePopupDim()
	if w != 71 || h != maxPaletteItems+3 {
		t.Fatalf("palettePopupDim = %dx%d, want 71x%d", w, h, maxPaletteItems+3)
	}
}

func TestPlaceOverlayCenters(t *testing.T) {
	bg := strings.Repeat(strings.Repeat(".", 20)+"\n", 10)
	bg = strings.TrimSuffix(bg, "\n")
	fg := "HELLO"
	got := placeOverlay(bg, fg, 5, 2)
	lines := strings.Split(got, "\n")
	if !strings.Contains(lines[2], "HELLO") {
		t.Fatalf("overlay missing on row 2: %q", lines[2])
	}
}

func TestSynthesizeAndReplay(t *testing.T) {
	if _, ok := synthesizeKeyMsg(""); ok {
		t.Fatal("empty token should fail")
	}
	k, ok := synthesizeKeyMsg("ctrl+r")
	if !ok || k.String() != "ctrl+r" {
		t.Fatalf("ctrl+r = %q ok=%v", k.String(), ok)
	}
	if replayKeySequence(nil) != nil {
		t.Fatal("empty seq → nil")
	}
	if replayKeySequence([]string{"g", "r"}) == nil {
		t.Fatal("chord should yield cmd")
	}
}

func TestFuzzyMatch(t *testing.T) {
	idx, _ := fuzzyMatch("blm", "open blame grid")
	if idx == nil {
		t.Fatal("expected fuzzy match")
	}
	if idx, _ := fuzzyMatch("zzz", "blame"); idx != nil {
		t.Fatal("expected no match")
	}
}

func paletteSetCursor(p *palette, pred func(paletteItem) bool) bool {
	for i, it := range p.filtered {
		if pred(it) {
			p.cursor = i
			return true
		}
	}
	return false
}

func simulatePaletteTyping(p *palette, s string) {
	for _, ch := range s {
		next, _ := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		*p = next
	}
}
