package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/rsiota/ore/internal/git"
)

func TestFitWidthPreservesANSIWidth(t *testing.T) {
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Render("+added line that is quite long and should truncate cleanly")
	got := fitWidth(styled, 20)
	if w := lipgloss.Width(got); w != 20 {
		t.Fatalf("lipgloss.Width = %d, want 20 (got %q)", w, got)
	}
	// Truncation must not leave an open SGR that would bleed into a following cell.
	plain := fitWidth("hello", 20)
	joined := got + plain
	if lipgloss.Width(joined) != 40 {
		t.Fatalf("joined width = %d, want 40", lipgloss.Width(joined))
	}
}

func TestCellDoesNotWrapLongRows(t *testing.T) {
	long := strings.Repeat("x", 200)
	got := cell(styleFocus, long, 40)
	if strings.Contains(got, "\n") {
		t.Fatalf("cell wrapped into multiple lines:\n%q", got)
	}
	if w := lipgloss.Width(got); w != 40 {
		t.Fatalf("width = %d, want 40", w)
	}
}

func TestClampFrameExactHeight(t *testing.T) {
	in := "a\nb\nc"
	got := clampFrame(in, 5, 10)
	lines := strings.Split(got, "\n")
	if len(lines) != 5 {
		t.Fatalf("len = %d, want 5", len(lines))
	}
	got = clampFrame(strings.Repeat("row\n", 20), 3, 10)
	if n := strings.Count(got, "\n") + 1; n != 3 {
		t.Fatalf("lines = %d, want 3", n)
	}
}

func TestDetailLinesSplitsStat(t *testing.T) {
	m := Model{
		detail: &git.CommitDetail{
			Commit: git.Commit{
				Hash:      "abc",
				ShortHash: "abc",
				Author:    "a",
				Email:     "a@b",
				Subject:   "s",
				Files:     2,
			},
			Stat: " a.txt | 1 +\n b.txt | 2 ++\n 2 files changed\n",
			Diff: "+one\n-two\n",
		},
	}
	lines := m.detailLines()
	joined := strings.Join(lines, "\n")
	if strings.Count(joined, "\n") != len(lines)-1 {
		t.Fatal("detailLines embedded a raw newline inside a row")
	}
	var statRows int
	for _, line := range lines {
		if strings.Contains(line, "a.txt") || strings.Contains(line, "b.txt") || strings.Contains(line, "files changed") {
			statRows++
		}
	}
	if statRows < 3 {
		t.Fatalf("expected split stat rows, got %d in %#v", statRows, lines)
	}
}
