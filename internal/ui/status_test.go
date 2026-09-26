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

func TestStatusLeftOmitsKeybindings(t *testing.T) {
	m := Model{
		main:        MainHistory,
		historyPath: "a.go",
		history:     make([]git.PathCommit, 3),
	}
	m.refreshStatus()
	for _, banned := range []string{":more", "enter", "esc", "blame", "follow"} {
		if strings.Contains(m.status, banned) {
			t.Fatalf("history status still has %q: %q", banned, m.status)
		}
	}

	m.main = MainPickaxe
	m.pickKind = hitListPickaxe
	m.pickQuery = "token"
	m.pickaxe = make([]git.PickaxeHit, git.DefaultPickaxeLimit)
	m.pickHlOn = true
	m.refreshStatus()
	if strings.Contains(m.status, ":more") || strings.Contains(m.status, ":nohl") || strings.Contains(m.status, "esc") {
		t.Fatalf("pickaxe status still has keys: %q", m.status)
	}
	if !strings.Contains(m.status, "hl") {
		t.Fatalf("pickaxe should still report highlight state: %q", m.status)
	}
}

func TestRenderStatusKeepsRightHints(t *testing.T) {
	m := Model{
		repo:    &git.Repo{Path: "/tmp/demo"},
		width:   160,
		height:  40,
		main:    MainBlame,
		status:  "blame · a.go @ abc1234 · 12 lines",
		branch:  "main",
		head:    "abc1234",
	}
	plain := stripANSI(m.renderStatus())
	if !strings.Contains(plain, "j/") || !strings.Contains(plain, "ctrl+p") {
		t.Fatalf("right-hand hints missing: %q", plain)
	}
	left := strings.Split(plain, "j/")[0]
	for _, banned := range []string{"follow", "evolve", "esc back", ":more"} {
		if strings.Contains(left, banned) {
			t.Fatalf("left status still has %q: %q", banned, left)
		}
	}
}
