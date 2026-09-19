package git

import (
	"strings"
	"testing"
)

func TestParseNameStatusLine(t *testing.T) {
	st, path, old, ok := parseNameStatusLine("M\ta.go")
	if !ok || st != "M" || path != "a.go" || old != "" {
		t.Fatalf("M: %q %q %q %v", st, path, old, ok)
	}
	st, path, old, ok = parseNameStatusLine("R050\told.txt\tnew.txt")
	if !ok || st != "R050" || path != "new.txt" || old != "old.txt" {
		t.Fatalf("R: %q %q %q %v", st, path, old, ok)
	}
	st, path, old, ok = parseNameStatusLine("C100\tsrc.go\tdst.go")
	if !ok || !strings.HasPrefix(st, "C") || path != "dst.go" || old != "src.go" {
		t.Fatalf("C: %q %q %q %v", st, path, old, ok)
	}
}

func TestParsePathCommitLogRename(t *testing.T) {
	// Mirrors real git log --name-status --format=…%x1e output shape:
	// meta\x1e\n\nname-status\nmeta\x1e\n\nname-status\n
	raw := "" +
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\x1faaaaaaa\x1falice\x1fa@b\x1f2024-01-02T03:04:05Z\x1frename and edit\x1fbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\x1e\n" +
		"\n" +
		"R050\told.txt\tnew.txt\n" +
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\x1fbbbbbbb\x1fbob\x1fb@b\x1f2024-01-01T03:04:05Z\x1fadd old\x1f\x1e\n" +
		"\n" +
		"A\told.txt\n"
	pcs, err := parsePathCommitLog([]byte(raw), "new.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(pcs) != 2 {
		t.Fatalf("len = %d, want 2: %#v", len(pcs), pcs)
	}
	if pcs[0].Path != "new.txt" || pcs[0].OldPath != "old.txt" || !strings.HasPrefix(pcs[0].Status, "R") {
		t.Fatalf("rename hop = %#v", pcs[0])
	}
	if pcs[0].EdgeLabel() != "moved from old.txt" {
		t.Fatalf("edge = %q", pcs[0].EdgeLabel())
	}
	if pcs[0].PathLabel() != "new.txt (moved from old.txt)" {
		t.Fatalf("path label = %q", pcs[0].PathLabel())
	}
	if pcs[1].Path != "old.txt" || pcs[1].OldPath != "" || pcs[1].Status != "A" {
		t.Fatalf("add = %#v", pcs[1])
	}
	if pcs[1].PathLabel() != "old.txt" {
		t.Fatalf("add path label = %q", pcs[1].PathLabel())
	}
}
