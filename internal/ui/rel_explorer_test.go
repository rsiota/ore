package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsiota/ore/internal/git"
)

func TestRelExplorerCommitRenameWasLabel(t *testing.T) {
	var e RelExplorer
	e.LoadCommit(git.CommitRelations{
		Hash:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Subject: "move",
		Author:  "a",
		Email:   "a@b",
		Files:   []git.FileChange{{Path: "new.go", OldPath: "old.go", Status: "R"}},
	})
	found := false
	for _, n := range e.visibleNodes() {
		if n.kind == relFile && n.path == "new.go" {
			found = true
			if !strings.Contains(n.label, "was old.go") {
				t.Fatalf("label = %q, want was old.go", n.label)
			}
		}
	}
	if !found {
		t.Fatal("expected rename file row")
	}
}

func TestRelExplorerLineMovedFrom(t *testing.T) {
	var e RelExplorer
	e.LoadLine(git.LineRelations{
		Line: git.BlameLine{
			Line: 1, Hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			ShortHash: "aaaaaaa", Summary: "here", Text: "x",
			PreviousHash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			PreviousPath: "old.go",
		},
		Path:         "new.go",
		Rev:          "HEAD",
		PreviousPath: "old.go",
		Previous: &git.Commit{
			Hash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			ShortHash: "bbbbbbb", Subject: "before",
		},
		History: []git.PathCommit{
			{
				Commit:  git.Commit{Hash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ShortHash: "aaaaaaa", Subject: "rename"},
				Path:    "new.go",
				OldPath: "old.go",
				Status:  "R100",
			},
			{
				Commit: git.Commit{Hash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ShortHash: "bbbbbbb", Subject: "add"},
				Path:   "old.go",
				Status: "A",
			},
		},
	})
	prevOK, histOK, fileOK := false, false, false
	for _, n := range e.visibleNodes() {
		if n.kind == relCommit && strings.Contains(n.label, "moved from old.go") {
			prevOK = true
		}
		if n.kind == relCommit && strings.Contains(n.label, "moved from old.go") && strings.Contains(n.label, "rename") {
			histOK = true
		}
		if n.kind == relFile && strings.Contains(n.label, "was old.go") {
			fileOK = true
		}
	}
	if !prevOK {
		t.Fatal("expected Previous row with moved from")
	}
	if !histOK {
		t.Fatal("expected history rename hop label")
	}
	if !fileOK {
		t.Fatal("expected File row with was")
	}
}

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

func TestRelExplorerOwnership(t *testing.T) {
	var e RelExplorer
	e.SetSize(48, 20)
	e.LoadCommit(git.CommitRelations{
		Hash:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Subject: "root",
		Author:  "a",
		Email:   "a@b",
		Files:   []git.FileChange{{Path: "a.go"}},
		Ownership: []git.AuthorShare{
			{Name: "alice", Count: 12, Pct: 60},
			{Name: "bob", Count: 8, Pct: 40},
		},
	})
	foundSec, foundOwn := false, false
	for _, n := range e.visibleNodes() {
		if n.kind == relSection && n.label == "Ownership" {
			foundSec = true
		}
		if n.kind == relOwn && n.author == "alice" {
			foundOwn = true
			if n.selectable {
				t.Fatal("ownership rows should not be selectable")
			}
			if !strings.Contains(n.label, "60%") {
				t.Fatalf("label = %q", n.label)
			}
		}
	}
	if !foundSec || !foundOwn {
		t.Fatalf("sec=%v own=%v", foundSec, foundOwn)
	}
	view := e.View(true)
	if !strings.Contains(view, "alice") || !strings.Contains(view, "60%") {
		t.Fatalf("view missing ownership:\n%s", view)
	}
}

func TestRelExplorerHotSpots(t *testing.T) {
	var e RelExplorer
	e.LoadCommit(git.CommitRelations{
		Hash:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Subject: "root",
		Author:  "a",
		Email:   "a@b",
		Files:   []git.FileChange{{Path: "a.go"}},
		HotSpots: []git.CoChange{
			{Path: "b.go", Count: 5, Total: 8},
			{Path: "c.go", Count: 3, Total: 8},
		},
	})
	found := false
	var hot *relNode
	for _, n := range e.visibleNodes() {
		if n.kind == relSection && strings.Contains(n.label, "Often with") {
			found = true
		}
		if n.kind == relHotSpot && n.path == "b.go" {
			hot = n
		}
	}
	if !found {
		t.Fatal("expected Often with section")
	}
	if hot == nil {
		t.Fatal("expected hot spot row")
	}
	if !hot.selectable || hot.count != 5 {
		t.Fatalf("hot spot = %#v", hot)
	}
	if len(hot.coupleWith) != 1 || hot.coupleWith[0] != "a.go" {
		t.Fatalf("coupleWith = %#v, want [a.go]", hot.coupleWith)
	}
	got := renderRelNode(hot)
	if !strings.Contains(got, "  5  b.go") {
		t.Fatalf("leading count render = %q", got)
	}
}

func TestRelExplorerPathRowsMutedVsCommit(t *testing.T) {
	var e RelExplorer
	e.SetSize(40, 20)
	e.LoadCommit(git.CommitRelations{
		Hash:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Subject:  "root",
		Author:   "a",
		Email:    "a@b",
		Parents:  []git.Commit{{Hash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ShortHash: "bbbbbbb", Subject: "parent"}},
		Children: []git.Commit{{Hash: "cccccccccccccccccccccccccccccccccccccccc", ShortHash: "ccccccc", Subject: "child"}},
		Files:    []git.FileChange{{Path: "a.go"}},
		HotSpots: []git.CoChange{{Path: "b.go", Count: 4, Total: 6}},
	})
	// Leave cursor on the first commit so the second commit paints with cell fg.
	view := e.View(true)
	mutedPrefix := sgrPrefix(styleMuted)
	cellPrefix := sgrPrefix(styleCell)
	if mutedPrefix == "" || cellPrefix == "" {
		t.Fatal("expected styled prefixes")
	}
	if mutedPrefix == cellPrefix {
		t.Fatal("muted and cell styles collapsed; hierarchy invisible")
	}
	if !strings.Contains(view, mutedPrefix) {
		t.Fatalf("expected muted path styling in view")
	}
	if !strings.Contains(view, cellPrefix) {
		t.Fatalf("expected full-fg commit styling in view:\n%s", view)
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
