package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestWrapDisplay(t *testing.T) {
	got := wrapDisplay("abcdefghij", 4)
	want := []string{"abcd", "efgh", "ij"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %v want %v", got, want)
	}
	if one := wrapDisplay("hi", 10); len(one) != 1 || one[0] != "hi" {
		t.Fatalf("short = %v", one)
	}
}

func TestZenWrapContinuesGutter(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	rows := renderZenChange(true, 9, strings.Repeat("x", 40), 20, true)
	if len(rows) < 2 {
		t.Fatalf("expected wrapped rows, got %d %#v", len(rows), rows)
	}
	for _, row := range rows {
		if w := lipgloss.Width(row); w != 20 {
			t.Fatalf("width=%d row=%q", w, row)
		}
		if !strings.Contains(row, "│") {
			t.Fatalf("missing gutter rule: %q", row)
		}
	}
	if lipgloss.Width(zenGutterCont()) != lipgloss.Width(zenGutter(9)) {
		t.Fatalf("cont gutter %d != numbered %d", lipgloss.Width(zenGutterCont()), lipgloss.Width(zenGutter(9)))
	}
}

func TestExpandDetailRowsWrapToggle(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	line := zenAddPrefix + "1" + zenFieldSep + strings.Repeat("a", 50)
	trunc := expandDetailRows([]string{line}, 24, false)
	wrapped := expandDetailRows([]string{line}, 24, true)
	if len(trunc) != 1 {
		t.Fatalf("truncate rows=%d", len(trunc))
	}
	if len(wrapped) <= 1 {
		t.Fatalf("wrap should expand, got %d", len(wrapped))
	}
}
