package ui

import (
	"strings"
	"testing"

	"github.com/rsiota/ore/internal/git"
)

func TestJumpDagChildUsesLoadedLog(t *testing.T) {
	parent := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	child := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	m := Model{
		main: MainCommits,
		commits: []git.Commit{
			{Hash: child, ShortHash: "bbbbbbb", Subject: "tip", Parents: []string{parent}},
			{Hash: parent, ShortHash: "aaaaaaa", Subject: "base"},
		},
		cursor: 1,
	}
	mm, cmd := m.jumpDagChild()
	m = mm.(Model)
	if m.cursor != 0 {
		t.Fatalf("cursor=%d want 0 (child in log)", m.cursor)
	}
	if !strings.Contains(m.status, "child") {
		t.Fatalf("status=%q", m.status)
	}
	if cmd == nil {
		t.Fatal("expected reload detail")
	}
}

func TestJumpDagParent(t *testing.T) {
	parent := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	child := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	m := Model{
		main: MainCommits,
		commits: []git.Commit{
			{Hash: child, ShortHash: "bbbbbbb", Subject: "tip", Parents: []string{parent}},
			{Hash: parent, ShortHash: "aaaaaaa", Subject: "base", Parents: nil},
		},
		cursor: 0,
	}
	mm, cmd := m.jumpDagParent()
	m = mm.(Model)
	if m.cursor != 1 {
		t.Fatalf("cursor=%d want 1 (parent)", m.cursor)
	}
	if !strings.Contains(m.status, "parent") {
		t.Fatalf("status=%q", m.status)
	}
	if cmd == nil {
		t.Fatal("expected reload detail cmd")
	}
}

func TestJumpDagParentNone(t *testing.T) {
	m := Model{
		main: MainCommits,
		commits: []git.Commit{
			{Hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Parents: nil},
		},
		cursor: 0,
	}
	mm, cmd := m.jumpDagParent()
	m = mm.(Model)
	if cmd != nil {
		t.Fatal("expected no cmd")
	}
	if !strings.Contains(m.status, "no parent") {
		t.Fatalf("status=%q", m.status)
	}
}

func TestHandleDagJumpChild(t *testing.T) {
	parent := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	child := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	m := Model{
		main: MainCommits,
		commits: []git.Commit{
			{Hash: child, ShortHash: "bbbbbbb"},
			{Hash: parent, ShortHash: "aaaaaaa"},
		},
		cursor: 1,
	}
	mm, cmd := m.handleDagJumpMsg(dagJumpMsg{kind: "child", from: parent, to: child, total: 2})
	m = mm.(Model)
	if m.cursor != 0 {
		t.Fatalf("cursor=%d want 0 (child)", m.cursor)
	}
	if !strings.Contains(m.status, "child") || !strings.Contains(m.status, "1/2") {
		t.Fatalf("status=%q", m.status)
	}
	if cmd == nil {
		t.Fatal("expected reload")
	}
}

func TestHandleDagJumpMergeBaseSame(t *testing.T) {
	h := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	m := Model{main: MainCommits, commits: []git.Commit{{Hash: h}}, cursor: 0}
	mm, cmd := m.handleDagJumpMsg(dagJumpMsg{kind: "merge-base", from: h, to: h, total: 1})
	m = mm.(Model)
	if cmd != nil || !strings.Contains(m.status, "already") {
		t.Fatalf("status=%q cmd=%v", m.status, cmd)
	}
}

func TestDagJumpOutsideLogWindow(t *testing.T) {
	m := Model{
		main:     MainCommits,
		commits:  []git.Commit{{Hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}},
		cursor:   0,
		diffMode: DiffZen,
	}
	cmd := (&m).applyDagJump("parent", "cccccccccccccccccccccccccccccccccccccccc", 1, "")
	if m.detailExpectHash == "" || !strings.HasPrefix(m.detailExpectHash, "ccc") {
		t.Fatalf("expectHash=%q", m.detailExpectHash)
	}
	if !strings.Contains(m.status, "not in log") {
		t.Fatalf("status=%q", m.status)
	}
	if cmd == nil {
		t.Fatal("expected detail load")
	}
}
