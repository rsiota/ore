package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/rsiota/ore/internal/git"
)

func TestTruncateDiff(t *testing.T) {
	small := strings.Repeat("a\n", 10)
	got, trunc := truncateDiff(small)
	if trunc || got != small {
		t.Fatalf("small diff truncated unexpectedly: trunc=%v len=%d", trunc, len(got))
	}
	huge := strings.Repeat("line\n", maxDiffBytes)
	got, trunc = truncateDiff(huge)
	if !trunc {
		t.Fatal("expected truncation")
	}
	if len(got) > maxDiffBytes {
		t.Fatalf("truncated len %d > max %d", len(got), maxDiffBytes)
	}
	if !strings.HasSuffix(got, "\n") && strings.Contains(got, "\n") {
		// prefer cutting on a newline when possible
		t.Fatalf("expected newline-aligned cut: %q", got[len(got)-20:])
	}
}

func TestDetailDebounceCancelsPriorTick(t *testing.T) {
	m := Model{
		repo:        &git.Repo{Path: t.TempDir()},
		commits:     []git.Commit{{Hash: "aaa", ShortHash: "aaa"}},
		cursor:      0,
		detailCache: &detailRenderCache{},
	}
	cmd1 := m.reloadDetail()
	if cmd1 == nil {
		t.Fatal("expected debounce tick")
	}
	seq1 := m.detailDebounceSeq
	cmd2 := m.reloadDetail()
	if cmd2 == nil {
		t.Fatal("expected second debounce tick")
	}
	if m.detailDebounceSeq <= seq1 {
		t.Fatal("debounce seq should advance")
	}
	// Stale tick must be ignored.
	next, cmd := m.Update(detailDebounceMsg{seq: seq1})
	mm := next.(Model)
	if cmd != nil {
		t.Fatal("stale debounce should not start a load")
	}
	if mm.detailLoadSeq != 0 {
		t.Fatalf("load seq = %d, want 0", mm.detailLoadSeq)
	}
}

func TestDetailViewportSkipsFullExpandWhenUnwrapped(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 500; i++ {
		b.WriteString("+line\n")
	}
	m := Model{
		diffMode:    DiffUnified,
		detailWrap:  false,
		detailCache: &detailRenderCache{},
		detail: &git.CommitDetail{
			Commit: git.Commit{Hash: "abc", Subject: "s", Author: "a", Email: "e", Date: time.Now()},
			Diff:   b.String(),
		},
		width:  120,
		height: 40,
	}
	rows, total := m.detailVisualWindow(40, 10, 0)
	if total < 500 {
		t.Fatalf("total = %d, want >= 500 logical rows", total)
	}
	if len(rows) > 12 {
		t.Fatalf("viewport painted %d rows, want ~10", len(rows))
	}
	// Cache should hold logical only (no full visual expand).
	if m.detailCache.visual != nil {
		t.Fatal("wrap-off should not build full visual cache")
	}
}

func TestClampDetailBody(t *testing.T) {
	body := make([]string, maxDetailBodyLines+50)
	for i := range body {
		body[i] = "x"
	}
	got := clampDetailBody(body)
	if len(got) != maxDetailBodyLines+1 {
		t.Fatalf("len = %d", len(got))
	}
	if !strings.Contains(got[len(got)-1], "truncated") {
		t.Fatalf("missing trunc notice: %q", got[len(got)-1])
	}
}

func TestDetailPatchAppliesAfterHeader(t *testing.T) {
	m := Model{
		commits:          []git.Commit{{Hash: "abcabcabc", ShortHash: "abcabca"}},
		cursor:           0,
		detailFilterPath: "",
		detailLoadSeq:    3,
		detailCache:      &detailRenderCache{},
	}
	next, cmd := m.Update(detailHeaderMsg{
		seq:  3,
		hash: "abcabcabc",
		detail: git.CommitDetail{
			Commit: git.Commit{Hash: "abcabcabc", ShortHash: "abcabca", Subject: "hi"},
		},
	})
	mm := next.(Model)
	if mm.detail == nil || mm.detail.Commit.Subject != "hi" {
		t.Fatal("header not applied")
	}
	if !mm.detailPatchPending || cmd == nil {
		t.Fatal("expected patch pending + follow-up cmd")
	}
	next, _ = mm.Update(detailPatchMsg{
		seq:  3,
		hash: "abcabcabc",
		diff: "diff --git a/x b/x\n+hi\n",
	})
	mm = next.(Model)
	if mm.detailPatchPending || mm.loadingDetail {
		t.Fatal("patch should finish loading")
	}
	if mm.detail.Diff == "" {
		t.Fatal("diff not applied")
	}
}