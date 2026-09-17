package ui

import (
	"strings"
	"testing"

	"github.com/rsiota/ore/internal/git"
)

func TestPickaxeSpansString(t *testing.T) {
	spans := pickaxeSpans("foo UniqueToken bar UniqueToken", "UniqueToken", git.PickaxeString)
	if len(spans) != 2 {
		t.Fatalf("spans = %#v", spans)
	}
	if spans[0].Start != 4 || spans[0].End != 15 {
		t.Fatalf("first = %#v", spans[0])
	}
}

func TestPickaxeSpansRegexp(t *testing.T) {
	spans := pickaxeSpans("alpha UniqueToken beta", "UniqueTo[a-z]+", git.PickaxeRegexp)
	if len(spans) != 1 || spans[0].Start != 6 {
		t.Fatalf("spans = %#v", spans)
	}
}

func TestPickaxeHighlightClears(t *testing.T) {
	m := Model{
		pickQuery: "UniqueToken",
		pickMode:  git.PickaxeString,
		pickHlOn:  true,
		detailCache: &detailRenderCache{},
	}
	m.clearPickaxeHighlight()
	if m.pickHlOn {
		t.Fatal("expected highlight off")
	}
	if m.pickaxeHLActive() {
		t.Fatal("HL should be inactive")
	}
	if m.pickQuery != "UniqueToken" {
		t.Fatal("query should remain for results")
	}
}

func TestPickaxeHighlightInDetailRender(t *testing.T) {
	m := Model{
		pickQuery: "Token",
		pickMode:  git.PickaxeString,
		pickHlOn:  true,
	}
	line := strings.Repeat(" ", cellPad) + "+func Token() {}"
	rows := m.renderDetailRows(line, 40, false)
	if len(rows) == 0 {
		t.Fatal("expected rows")
	}
	flash := sgrPrefix(styleSearchStrong)
	if flash == "" || !strings.Contains(rows[0], flash) {
		t.Fatalf("expected search strong wash in %q", rows[0])
	}
	m.pickHlOn = false
	rows2 := m.renderDetailRows(line, 40, false)
	if strings.Contains(rows2[0], flash) {
		t.Fatalf("highlight should be off: %q", rows2[0])
	}
}
