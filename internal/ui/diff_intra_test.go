package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestIntraLineDiffHighlightsWord(t *testing.T) {
	delSpans, addSpans := intraLineDiff("return foo(x)", "return bar(x)")
	if len(delSpans) != 1 || len(addSpans) != 1 {
		t.Fatalf("del=%v add=%v", delSpans, addSpans)
	}
	old := "return foo(x)"
	new := "return bar(x)"
	if old[delSpans[0].Start:delSpans[0].End] != "foo" {
		t.Fatalf("del span = %q", old[delSpans[0].Start:delSpans[0].End])
	}
	if new[addSpans[0].Start:addSpans[0].End] != "bar" {
		t.Fatalf("add span = %q", new[addSpans[0].Start:addSpans[0].End])
	}
}

func TestIntraLineDiffIdentical(t *testing.T) {
	d, a := intraLineDiff("same", "same")
	if d != nil || a != nil {
		t.Fatalf("expected nil spans, got %v %v", d, a)
	}
}

func TestAnnotateZenIntraLinePairs(t *testing.T) {
	body := []zenHunkLine{
		{kind: ' ', text: "keep"},
		{kind: '-', text: "alpha beta"},
		{kind: '+', text: "alpha gamma"},
		{kind: ' ', text: "tail"},
	}
	annotateZenIntraLine(body)
	if len(body[1].spans) == 0 || len(body[2].spans) == 0 {
		t.Fatalf("expected spans on paired lines: %#v %#v", body[1].spans, body[2].spans)
	}
	if body[1].text[body[1].spans[0].Start:body[1].spans[0].End] != "beta" {
		t.Fatalf("del highlight = %q", body[1].text[body[1].spans[0].Start:body[1].spans[0].End])
	}
	if body[2].text[body[2].spans[0].Start:body[2].spans[0].End] != "gamma" {
		t.Fatalf("add highlight = %q", body[2].text[body[2].spans[0].Start:body[2].spans[0].End])
	}
}

func TestZenDiffLinesEncodeIntraSpans(t *testing.T) {
	diff := "diff --git a/f b/f\n@@ -1,1 +1,1 @@\n-hello world\n+hello there\n"
	lines := zenDiffLines(diff, 3)
	var del, add string
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, zenDelPrefix):
			del = line
		case strings.HasPrefix(line, zenAddPrefix):
			add = line
		}
	}
	if del == "" || add == "" {
		t.Fatalf("missing change lines: %v", lines)
	}
	_, delText, delSpans, ok := parseZenChangePayload(strings.TrimPrefix(del, zenDelPrefix))
	if !ok || len(delSpans) == 0 {
		t.Fatalf("del payload spans missing: %q", del)
	}
	_, addText, addSpans, ok := parseZenChangePayload(strings.TrimPrefix(add, zenAddPrefix))
	if !ok || len(addSpans) == 0 {
		t.Fatalf("add payload spans missing: %q", add)
	}
	if delText[delSpans[0].Start:delSpans[0].End] != "world" {
		t.Fatalf("del = %q spans %v", delText, delSpans)
	}
	if addText[addSpans[0].Start:addSpans[0].End] != "there" {
		t.Fatalf("add = %q spans %v", addText, addSpans)
	}
}

func TestRenderHighlightedCellUsesStrongWash(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer applyTheme(defaultThemeName)
	applyTheme("light")

	got := renderHighlightedCell(styleAddWash, styleAddStrong, "hello world", []byteSpan{{6, 11}}, 20)
	if !strings.Contains(got, "48;2;") {
		t.Fatalf("expected background washes: %q", got)
	}
	// Strong wash #aceebb = 172,238,187
	if !strings.Contains(got, "48;2;172;238;187") && !strings.Contains(got, "172;238;187") {
		t.Fatalf("expected strong add wash in %q", got)
	}
	if lipgloss.Width(got) != 20 {
		t.Fatalf("width = %d, want 20", lipgloss.Width(got))
	}
}

func TestAnnotateUnifiedIntra(t *testing.T) {
	in := []string{
		"@@ -1,1 +1,1 @@",
		"-foo bar",
		"+foo baz",
		" context",
	}
	out := annotateUnifiedIntra(in)
	if !strings.HasPrefix(out[1], intraDelPrefix) || !strings.HasPrefix(out[2], intraAddPrefix) {
		t.Fatalf("expected intra markers: %#v", out)
	}
	_, spans := parseIntraPayload(strings.TrimPrefix(out[1], intraDelPrefix))
	if len(spans) == 0 {
		t.Fatalf("expected del spans in %q", out[1])
	}
}
