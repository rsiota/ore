package ui

import "testing"

func TestDetailHunkRows(t *testing.T) {
	plain := []string{
		"subject",
		"@@ -1,2 +1,3 @@",
		" context",
		"+add",
		"@@ -10 +10 @@ fn",
		"-del",
	}
	hunks := detailHunkRows(plain)
	if len(hunks) != 2 || hunks[0] != 1 || hunks[1] != 4 {
		t.Fatalf("hunks = %#v", hunks)
	}
	if !isHunkHeaderPlain("  @@ -1 +1 @@") {
		t.Fatal("expected trimmed @@ header")
	}
	if isHunkHeaderPlain("+++ b/file") || isHunkHeaderPlain("+@@ not a header") {
		t.Fatal("false positive hunk header")
	}
}

func TestNextPrevHunkRow(t *testing.T) {
	hunks := []int{2, 8, 15}
	if r, ok := nextHunkRow(hunks, 2); !ok || r != 8 {
		t.Fatalf("next from 2 = %d %v", r, ok)
	}
	if _, ok := nextHunkRow(hunks, 15); ok {
		t.Fatal("next past last should fail")
	}
	if r, ok := prevHunkRow(hunks, 8); !ok || r != 2 {
		t.Fatalf("prev from 8 = %d %v", r, ok)
	}
	if _, ok := prevHunkRow(hunks, 2); ok {
		t.Fatal("prev before first should fail")
	}
	if r, ok := nextHunkRow(hunks, 0); !ok || r != 2 {
		t.Fatalf("next from before first = %d %v", r, ok)
	}
}

func TestHunkOrdinal(t *testing.T) {
	hunks := []int{2, 8, 15}
	if hunkOrdinal(hunks, 8) != 2 {
		t.Fatal("expected ordinal 2")
	}
	if hunkOrdinal(hunks, 9) != 0 {
		t.Fatal("non-header should be 0")
	}
}
