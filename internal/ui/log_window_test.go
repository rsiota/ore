package ui

import (
	"strings"
	"testing"

	"github.com/rsiota/ore/internal/git"
)

func TestCommitsTitleShowsWindowWhenCapped(t *testing.T) {
	m := Model{
		commits:  make([]git.Commit, git.DefaultLogLimit),
		logLimit: git.DefaultLogLimit,
		logTotal: 4231,
	}
	if got := m.mainTitle(); got != "commits · 500/4231" {
		t.Fatalf("title = %q", got)
	}
	if !m.logCapped() {
		t.Fatal("expected capped")
	}
	if !strings.Contains(m.commitsWindowText(), "500/4231") {
		t.Fatalf("window = %q", m.commitsWindowText())
	}

	m.logTotal = git.DefaultLogLimit
	if m.logCapped() {
		t.Fatal("full window should not be capped")
	}
	if m.mainTitle() != "commits" {
		t.Fatalf("complete title = %q", m.mainTitle())
	}
}

func TestCommitsTitleUnknownTotal(t *testing.T) {
	m := Model{
		commits:  make([]git.Commit, git.DefaultLogLimit),
		logLimit: git.DefaultLogLimit,
	}
	if got := m.mainTitle(); got != "commits · 500+" {
		t.Fatalf("title = %q", got)
	}
}

func TestLoadMoreNoopsWhenComplete(t *testing.T) {
	m := Model{
		repo:     &git.Repo{Path: "/tmp/demo"},
		commits:  []git.Commit{{Hash: "aaa"}},
		logLimit: git.DefaultLogLimit,
		logTotal: 1,
	}
	if m.loadMore(0) != nil {
		t.Fatal("expected nil when showing all")
	}
	m.exMore(nil)
	if m.status != "already showing all commits" {
		t.Fatalf("status = %q", m.status)
	}
}

func TestLoadMoreExtendsLimit(t *testing.T) {
	m := Model{
		repo:     &git.Repo{Path: "/tmp/demo"},
		commits:  make([]git.Commit, git.DefaultLogLimit),
		logLimit: git.DefaultLogLimit,
		logTotal: 2000,
	}
	m.commits[0].Hash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	cmd := m.loadMore(0)
	if cmd == nil {
		t.Fatal("expected load cmd")
	}
	if m.logLimit != git.DefaultLogLimit*2 {
		t.Fatalf("limit = %d", m.logLimit)
	}
	if !m.logExtending || !m.loading {
		t.Fatalf("extending=%v loading=%v", m.logExtending, m.loading)
	}
	if m.refreshPreferHash != m.commits[0].Hash {
		t.Fatalf("prefer = %q", m.refreshPreferHash)
	}
}

func TestExMoreRejectsBadCount(t *testing.T) {
	m := Model{
		repo:     &git.Repo{Path: "/tmp/demo"},
		commits:  make([]git.Commit, git.DefaultLogLimit),
		logLimit: git.DefaultLogLimit,
		logTotal: 2000,
	}
	if cmd := m.exMore([]string{"nope"}); cmd != nil {
		t.Fatal("expected no cmd")
	}
	if !strings.HasPrefix(m.status, "E488") {
		t.Fatalf("status = %q", m.status)
	}
}

func TestCappedSuffix(t *testing.T) {
	if cappedSuffix(99, 100) != "" {
		t.Fatal("under cap")
	}
	if cappedSuffix(100, 100) != " · capped" {
		t.Fatal("at cap")
	}
}

func TestCommitsLoadedMoreKeepsView(t *testing.T) {
	hash := "bbb2222222222222222222222222222222222222"
	m := Model{
		main:              MainFiles,
		filesCommitHash:   hash,
		refreshPreferHash: hash,
		logExtending:      true,
		loading:           true,
		cursor:            0,
		commits: []git.Commit{
			{Hash: "aaa1111111111111111111111111111111111111"},
		},
	}
	next, _ := m.Update(commitsLoadedMsg{
		commits: []git.Commit{
			{Hash: "aaa1111111111111111111111111111111111111"},
			{Hash: hash},
		},
		total:  2,
		limit:  1000,
		branch: "main",
		head:   "aaa1111",
	})
	mm := next.(Model)
	if mm.main != MainFiles {
		t.Fatalf("main = %v", mm.main)
	}
	if mm.cursor != 1 {
		t.Fatalf("cursor = %d", mm.cursor)
	}
	if mm.logLimit != 1000 || mm.logTotal != 2 {
		t.Fatalf("limit=%d total=%d", mm.logLimit, mm.logTotal)
	}
	if mm.logExtending {
		t.Fatal("extending should clear")
	}
	if strings.HasPrefix(mm.status, "refreshed") {
		t.Fatalf("more should not say refreshed: %q", mm.status)
	}
}

func TestPickaxeTitleCapped(t *testing.T) {
	m := Model{main: MainPickaxe, pickKind: hitListPickaxe, pickaxe: make([]git.PickaxeHit, git.DefaultPickaxeLimit)}
	if m.mainTitle() != "pickaxe · 100+" {
		t.Fatalf("title = %q", m.mainTitle())
	}
	m.pickKind = hitListCouple
	m.pickaxe = make([]git.PickaxeHit, git.CoChangeCommitCap)
	if m.mainTitle() != "couple · 80+" {
		t.Fatalf("title = %q", m.mainTitle())
	}
	m.pickKind = hitListAuthors
	m.pickaxe = make([]git.PickaxeHit, git.OwnershipCommitCap)
	if m.mainTitle() != "authors · 100+" {
		t.Fatalf("title = %q", m.mainTitle())
	}
}
