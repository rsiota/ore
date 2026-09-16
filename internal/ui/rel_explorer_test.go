package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsiota/ore/internal/git"
)

func TestRelExplorerCommitSelectable(t *testing.T) {
	var e RelExplorer
	e.LoadCommit(git.CommitRelations{
		Hash:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Subject: "root change",
		Author:  "a",
		Email:   "a@b",
		Parents: []git.Commit{{Hash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ShortHash: "bbbbbbb", Subject: "parent"}},
		Files:   []git.FileChange{{Path: "x.go", Status: "M"}},
	})
	if !e.Opened() {
		t.Fatal("expected open")
	}
	row, ok := e.Selected()
	if !ok || row.kind != relCommit {
		t.Fatalf("selected = %#v ok=%v", row, ok)
	}
	if row.hash == "" {
		t.Fatal("expected commit hash on selection")
	}
	if !row.expandable {
		t.Fatal("commit rows should be expandable")
	}
}

func TestRelExplorerActivateFileRow(t *testing.T) {
	var e RelExplorer
	e.LoadCommit(git.CommitRelations{
		Hash:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Subject: "s",
		Author:  "a",
		Email:   "a@b",
		Files:   []git.FileChange{{Path: "readme.md"}},
	})
	found := false
	vis := e.visibleNodes()
	for i := range vis {
		e.cursor = i
		if row, ok := e.Selected(); ok && row.kind == relFile {
			found = true
			if row.path != "readme.md" {
				t.Fatalf("path = %q", row.path)
			}
			break
		}
	}
	if !found {
		t.Fatal("expected a selectable file row")
	}
}

func TestRelExplorerExpandShowsNested(t *testing.T) {
	var e RelExplorer
	e.LoadCommit(git.CommitRelations{
		Hash:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Subject: "root",
		Author:  "a",
		Email:   "a@b",
		Parents: []git.Commit{{Hash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ShortHash: "bbbbbbb", Subject: "parent"}},
	})
	// Select parent commit.
	vis := e.visibleNodes()
	parentIdx := -1
	for i, n := range vis {
		if n.kind == relCommit && strings.HasPrefix(n.hash, "bbbbbbb") {
			parentIdx = i
			break
		}
	}
	if parentIdx < 0 {
		t.Fatal("parent commit not found")
	}
	e.cursor = parentIdx
	before := len(e.visibleNodes())
	act, cmd := e.ExpandOrDive()
	if act || cmd == nil {
		t.Fatalf("expand should request load: act=%v cmd=%v", act, cmd)
	}
	msg := cmd().(relExpandRequestMsg)
	if msg.hash == "" || msg.nodeID == "" {
		t.Fatalf("bad request: %#v", msg)
	}
	e.ApplyExpand(msg.nodeID, git.CommitRelations{
		Hash:     "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Subject:  "parent",
		Parents:  []git.Commit{{Hash: "cccccccccccccccccccccccccccccccccccccccc", ShortHash: "ccccccc", Subject: "grand"}},
		Children: nil,
		Files:    []git.FileChange{{Path: "nested.go"}},
	}, nil)
	after := len(e.visibleNodes())
	if after <= before {
		t.Fatalf("expected nested rows: before=%d after=%d", before, after)
	}
	foundFile := false
	for _, n := range e.visibleNodes() {
		if n.kind == relFile && n.path == "nested.go" {
			foundFile = true
			if n.depth < 2 {
				t.Fatalf("nested file depth = %d", n.depth)
			}
		}
	}
	if !foundFile {
		t.Fatal("expected nested file row")
	}
}

func TestRelExplorerCollapse(t *testing.T) {
	var e RelExplorer
	e.LoadCommit(git.CommitRelations{
		Hash:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Subject: "root",
		Author:  "a",
		Email:   "a@b",
		Parents: []git.Commit{{Hash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ShortHash: "bbbbbbb", Subject: "parent"}},
	})
	for i, n := range e.visibleNodes() {
		if n.kind == relCommit {
			e.cursor = i
			break
		}
	}
	_, cmd := e.ExpandOrDive()
	req := cmd().(relExpandRequestMsg)
	e.ApplyExpand(req.nodeID, git.CommitRelations{
		Hash:  "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Files: []git.FileChange{{Path: "f.go"}},
	}, nil)
	expanded := len(e.visibleNodes())
	// Move onto a nested row then collapse.
	for i, n := range e.visibleNodes() {
		if n.kind == relFile && n.path == "f.go" {
			e.cursor = i
			break
		}
	}
	closed := e.CollapseOrClose()
	if closed {
		t.Fatal("collapse should not close explorer")
	}
	if len(e.visibleNodes()) >= expanded {
		t.Fatalf("expected fewer visible rows after collapse")
	}
}

func TestRelExplorerKeysExpandVsEnter(t *testing.T) {
	var e RelExplorer
	e.LoadCommit(git.CommitRelations{
		Hash:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Subject: "root",
		Author:  "a",
		Email:   "a@b",
		Parents: []git.Commit{{Hash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ShortHash: "bbbbbbb", Subject: "parent"}},
	})
	for i, n := range e.visibleNodes() {
		if n.kind == relCommit {
			e.cursor = i
			break
		}
	}
	consumed, activate, cmd := e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	if !consumed || activate || cmd == nil {
		t.Fatalf("l should expand: consumed=%v activate=%v cmd=%v", consumed, activate, cmd)
	}
	consumed, activate, cmd = e.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !consumed || !activate || cmd != nil {
		t.Fatalf("enter should activate: consumed=%v activate=%v cmd=%v", consumed, activate, cmd)
	}
}
