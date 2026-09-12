package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/rsiota/ore/internal/git"
)

func TestMain(m *testing.M) {
	// Tests run without a TTY; force TrueColor so wash/ANSI assertions are meaningful.
	lipgloss.SetColorProfile(termenv.TrueColor)
	m.Run()
}

func TestFitWidthPreservesANSIWidth(t *testing.T) {
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("#1a7f37")).Render("+added line that is quite long and should truncate cleanly")
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

func TestDiffWashSpansFullWidth(t *testing.T) {
	got := renderDetailLine("+added something", 24)
	if w := lipgloss.Width(got); w != 24 {
		t.Fatalf("width = %d, want 24", w)
	}
	if strings.Contains(got, "\n") {
		t.Fatalf("wash line wrapped: %q", got)
	}
	// Background should be present in the ANSI output for a true wash.
	if !strings.Contains(got, "\x1b[") {
		t.Fatal("expected ANSI styling on add wash line")
	}
}

func TestDetailLineStripsCarriageReturn(t *testing.T) {
	// CRLF diffs leave \r after splitting on \n; that must not survive into
	// the rendered cell or wash padding paints from column 0 over the left pane.
	got := renderDetailLine("+added from windows\r", 24)
	if strings.Contains(got, "\r") {
		t.Fatalf("carriage return survived render: %q", got)
	}
	if w := lipgloss.Width(got); w != 24 {
		t.Fatalf("width = %d, want 24", w)
	}
	plain := renderDetailLine("subject with\rCR", 20)
	if strings.Contains(plain, "\r") {
		t.Fatalf("carriage return survived plain line: %q", plain)
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

func TestDetailLinesHierarchy(t *testing.T) {
	m := Model{
		detail: &git.CommitDetail{
			Commit: git.Commit{
				Hash:      "abc123",
				ShortHash: "abc123",
				Author:    "Ada",
				Email:     "ada@ex",
				Subject:   "Refactor auth",
				Files:     1,
				Additions: 3,
				Deletions: 1,
			},
			Body: "More detail here.",
			Diff: "diff --git a/x b/x\n@@ -1 +1 @@\n+hi\n",
		},
	}
	lines := m.detailLines()
	subjectAt, sepAt, diffAt := -1, -1, -1
	for i, line := range lines {
		plain := line
		if strings.Contains(plain, "Refactor auth") && !strings.Contains(plain, "More") {
			subjectAt = i
		}
		if plain == detailSepMarker {
			sepAt = i
		}
		if strings.HasPrefix(plain, "diff --git") {
			diffAt = i
		}
	}
	if subjectAt < 0 || sepAt < 0 || diffAt < 0 {
		t.Fatalf("missing hierarchy markers: subject=%d sep=%d diff=%d in %#v", subjectAt, sepAt, diffAt, lines)
	}
	if !(subjectAt < sepAt && sepAt < diffAt) {
		t.Fatalf("want subject < rule < diff, got %d < %d < %d", subjectAt, sepAt, diffAt)
	}
}

func TestRenderDetailLineHunkAndSep(t *testing.T) {
	sep := renderDetailLine(detailSepMarker, 20)
	if w := lipgloss.Width(sep); w != 20 {
		t.Fatalf("sep width = %d", w)
	}
	if !strings.Contains(sep, "─") {
		t.Fatalf("expected rule glyphs: %q", sep)
	}
	hunk := renderDetailLine("@@ -1,2 +1,3 @@", 24)
	if w := lipgloss.Width(hunk); w != 24 {
		t.Fatalf("hunk width = %d", w)
	}
	meta := renderDetailLine("diff --git a/f b/f", 24)
	if w := lipgloss.Width(meta); w != 24 {
		t.Fatalf("meta width = %d", w)
	}
	// +++ / --- file headers must not get add/del wash.
	plusPlus := renderDetailLine("+++ b/f", 16)
	if strings.Contains(plusPlus, "48;2;218;251;225") { // add wash bg
		t.Fatalf("+++ header got add wash: %q", plusPlus)
	}
}
