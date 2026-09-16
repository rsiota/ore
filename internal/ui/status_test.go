package ui

import (
	"strings"
	"testing"

	"github.com/rsiota/ore/internal/git"
)

func TestStatusHintsStayDuringDetailReload(t *testing.T) {
	m := Model{
		repo:          &git.Repo{Path: "/tmp/demo"},
		width:         120,
		height:        40,
		main:          MainFiles,
		status:        "3 files in abc1234",
		loadingDetail: true, // j/k through files
		branch:        "main",
		head:          "abc1234",
	}
	plain := stripANSI(m.renderStatus())
	if !strings.Contains(plain, "j/") || !strings.Contains(plain, "k/") {
		t.Fatalf("hints missing while loadingDetail: %q", plain)
	}
	if strings.Contains(plain, "fetching…") {
		t.Fatalf("detail reload should not mark status bar busy: %q", plain)
	}
}

func TestStatusHintsStayDuringHistoryFetch(t *testing.T) {
	m := Model{
		repo:           &git.Repo{Path: "/tmp/demo"},
		width:          160,
		height:         40,
		main:           MainFiles,
		status:         "loading history · path.go",
		loadingHistory: true,
		branch:         "main",
		head:           "abc1234",
	}
	plain := stripANSI(m.renderStatus())
	// Mid grows with "fetching…"; assert a stable hint fragment rather than
	// the full joined string (narrow widths truncate the right-hand cluster).
	if !strings.Contains(plain, "j/") || !strings.Contains(plain, "ctrl+p") {
		t.Fatalf("hints missing while loadingHistory: %q", plain)
	}
	if !strings.Contains(plain, "fetching…") {
		t.Fatalf("history load should still show fetching: %q", plain)
	}
}
