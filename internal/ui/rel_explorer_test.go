package ui

import (
	"testing"

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
	// Move to the file row (skip parent section empties / commit rows).
	found := false
	for i := 0; i < len(e.rows); i++ {
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
