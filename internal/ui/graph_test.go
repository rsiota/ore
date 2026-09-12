package ui

import (
	"strings"
	"testing"

	"github.com/rsiota/ore/internal/git"
)

func TestBuildSoftGraphLinear(t *testing.T) {
	commits := []git.Commit{
		{Hash: "c", Parents: []string{"b"}},
		{Hash: "b", Parents: []string{"a"}},
		{Hash: "a", Parents: nil},
	}
	got := buildSoftGraph(commits, []int{0, 1, 2})
	for i, g := range got {
		if !strings.Contains(g, "●") {
			t.Fatalf("row %d missing node: %q (full %#v)", i, g, got)
		}
		if strings.ContainsAny(g, "│╮╰├┤─┼") {
			t.Fatalf("linear row %d should be a lone node, got %q", i, g)
		}
	}
}

func TestBuildSoftGraphBranch(t *testing.T) {
	commits := []git.Commit{
		{Hash: "m2", Parents: []string{"m1"}},
		{Hash: "s1", Parents: []string{"m1"}},
		{Hash: "m1", Parents: []string{"m0"}},
		{Hash: "m0", Parents: nil},
	}
	got := buildSoftGraph(commits, []int{0, 1, 2, 3})
	t.Logf("branch graph: %#v", got)
	if got[0] != "●" {
		t.Fatalf("m2: %q", got[0])
	}
	if !strings.Contains(got[1], "●") || !strings.Contains(got[1], "│") {
		t.Fatalf("s1 should show parallel lanes: %q", got[1])
	}
	if got[1] != "│ ●" && got[1] != "● │" {
		// Prefer spaced two-lane form.
		if !strings.Contains(got[1], " ") {
			t.Fatalf("s1 should space lanes: %q", got[1])
		}
	}
	if !strings.Contains(got[2], "●") {
		t.Fatalf("m1: %q", got[2])
	}
	if !strings.ContainsAny(got[2], "╯╰╮╭─") {
		t.Fatalf("m1 should close the side lane with a corner: %q", got[2])
	}
}

func TestBuildSoftGraphMerge(t *testing.T) {
	commits := []git.Commit{
		{Hash: "merge", Parents: []string{"main", "side"}},
		{Hash: "side", Parents: []string{"base"}},
		{Hash: "main", Parents: []string{"base"}},
		{Hash: "base", Parents: nil},
	}
	got := buildSoftGraph(commits, []int{0, 1, 2, 3})
	t.Logf("merge graph: %#v", got)
	if !strings.Contains(got[0], "●") {
		t.Fatalf("merge: %q", got[0])
	}
	if !strings.ContainsAny(got[0], "╮╭─") {
		t.Fatalf("merge should show fork corner: %q", got[0])
	}
	if !strings.Contains(got[1], "●") {
		t.Fatalf("side: %q", got[1])
	}
}

func TestBuildSoftGraphSkipsParentsOutsideView(t *testing.T) {
	commits := []git.Commit{
		{Hash: "b", Parents: []string{"missing"}},
		{Hash: "a", Parents: []string{"missing"}},
	}
	got := buildSoftGraph(commits, []int{0, 1})
	if got[0] != "●" || got[1] != "●" {
		t.Fatalf("got %#v", got)
	}
}
