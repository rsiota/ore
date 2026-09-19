package ui

import (
	"testing"

	"github.com/rsiota/ore/internal/git"
)

func TestHistRowRenameEdge(t *testing.T) {
	pc := git.PathCommit{
		Commit:  git.Commit{ShortHash: "abc1234", Author: "a", Subject: "move"},
		Path:    "new.go",
		OldPath: "old.go",
		Status:  "R100",
	}
	row := histRow(pc, "")
	if row[histColPath] != "new.go (moved from old.go)" {
		t.Fatalf("path = %q", row[histColPath])
	}
	if row[histColSubject] != "move" {
		t.Fatalf("subject = %q", row[histColSubject])
	}
}

func TestFilterHistPathIncludesOld(t *testing.T) {
	commits := []git.PathCommit{
		{Commit: git.Commit{Subject: "a"}, Path: "new.go", OldPath: "legacy.go", Status: "R050"},
		{Commit: git.Commit{Subject: "b"}, Path: "other.go"},
	}
	idx := filterHistIndicesCol(commits, "legacy", histColPath)
	if len(idx) != 1 || idx[0] != 0 {
		t.Fatalf("idx = %#v", idx)
	}
}
