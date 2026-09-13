package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/rsiota/ore/internal/git"
)

func TestZenDiffLinesWithContext(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/foo.go b/foo.go",
		"index 111..222 100644",
		"--- a/foo.go",
		"+++ b/foo.go",
		"@@ -10,4 +10,5 @@ func main() {",
		" context",
		"-old",
		"+new",
		"+extra",
		" more",
	}, "\n")
	got := zenDiffLines(diff, 3)
	want := []string{
		zenFilePrefix + "foo.go",
		zenHunkPrefix + "@@ -10,4 +10,5 @@ func main() {",
		zenCtxPrefix + "10" + zenFieldSep + "context",
		zenDelPrefix + "11" + zenFieldSep + "old",
		zenAddPrefix + "11" + zenFieldSep + "new",
		zenAddPrefix + "12" + zenFieldSep + "extra",
		zenCtxPrefix + "13" + zenFieldSep + "more",
	}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d\n%#v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestZenDiffTrimsContext(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/f b/f",
		"+++ b/f",
		"@@ -1,7 +1,7 @@",
		" a",
		" b",
		" c",
		"-x",
		"+y",
		" d",
		" e",
		" f",
	}, "\n")
	got := zenDiffLines(diff, 1)
	// Keep 1 context before/after the change; drop a,b and e,f.
	var kinds []string
	for _, line := range got {
		switch {
		case strings.HasPrefix(line, zenHunkPrefix):
			kinds = append(kinds, "hunk")
		case strings.HasPrefix(line, zenCtxPrefix):
			_, text, _ := parseZenNumText(strings.TrimPrefix(line, zenCtxPrefix))
			kinds = append(kinds, "ctx:"+text)
		case strings.HasPrefix(line, zenDelPrefix):
			kinds = append(kinds, "del")
		case strings.HasPrefix(line, zenAddPrefix):
			kinds = append(kinds, "add")
		case strings.HasPrefix(line, zenFilePrefix):
			kinds = append(kinds, "file")
		}
	}
	want := []string{"file", "hunk", "ctx:c", "del", "add", "ctx:d"}
	if strings.Join(kinds, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v want %v\n%#v", kinds, want, got)
	}
}

func TestZenDiffSeparateHunkHeaders(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/f b/f",
		"+++ b/f",
		"@@ -1,3 +1,3 @@ top",
		" a",
		"-b",
		"+B",
		" c",
		"@@ -20,3 +20,3 @@ bottom",
		" x",
		"-y",
		"+Y",
		" z",
	}, "\n")
	got := zenDiffLines(diff, 3)
	headers := 0
	for _, line := range got {
		if strings.HasPrefix(line, zenHunkPrefix) {
			headers++
		}
	}
	if headers != 2 {
		t.Fatalf("want 2 hunk headers, got %d in %#v", headers, got)
	}
}

func TestRenderZenChangeWidth(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	got := renderZenChange(true, 42, "hello world", 24)
	if w := lipgloss.Width(got); w != 24 {
		t.Fatalf("width=%d want 24 (%q)", w, got)
	}
	if !strings.Contains(got, "│") {
		t.Fatalf("expected gutter rule: %q", got)
	}
	hunk := renderZenHunk("@@ -1 +1 @@", 40)
	if w := lipgloss.Width(hunk); w != 40 {
		t.Fatalf("hunk width=%d", w)
	}
}

func TestZenGutterAlignsHunkWithCode(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	if w := lipgloss.Width(zenGutter(12)); w != zenGutterCols() {
		t.Fatalf("gutter width %d, want %d", w, zenGutterCols())
	}
	code := renderZenContext(12, "hello", 40)
	hunk := renderZenHunk("@@ -1 +1 @@ note", 40)
	if lipgloss.Width(code) != 40 || lipgloss.Width(hunk) != 40 {
		t.Fatal("row widths")
	}
}

func TestDiffModeCycle(t *testing.T) {
	if DiffZen.Next() != DiffUnified || DiffUnified.Next() != DiffZen {
		t.Fatal("cycle")
	}
}

func TestDetailLinesUsesZenByDefault(t *testing.T) {
	m := Model{
		diffMode:   DiffZen,
		zenContext: 3,
		detail: &git.CommitDetail{
			Commit: git.Commit{Hash: "h", Subject: "s", Files: 1},
			Diff:   "diff --git a/f b/f\n--- a/f\n+++ b/f\n@@ -1 +1 @@\n-a\n+b\n",
		},
	}
	lines := m.detailLines()
	var sawZen, sawUnified bool
	for _, line := range lines {
		if strings.HasPrefix(line, zenAddPrefix) || strings.HasPrefix(line, zenDelPrefix) || strings.HasPrefix(line, zenHunkPrefix) {
			sawZen = true
		}
		if strings.HasPrefix(line, "@@") {
			sawUnified = true
		}
	}
	if !sawZen || sawUnified {
		t.Fatalf("zen=%v unified=%v lines=%#v", sawZen, sawUnified, lines)
	}
}
