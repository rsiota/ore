package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
	"github.com/rsiota/ore/internal/git"
)

func TestBranchPickerFilter(t *testing.T) {
	var p branchPicker
	p.Open([]git.Ref{
		{Name: "main", Hash: "aaa", Subject: "on main", Current: true},
		{Name: "feature/branch", Hash: "bbb", Subject: "wip"},
		{Name: "origin/main", Hash: "ccc", Subject: "remote", Remote: true},
	}, "")
	if !p.IsVisible() {
		t.Fatal("expected visible")
	}
	if len(p.filtered) != 3 {
		t.Fatalf("filtered = %d", len(p.filtered))
	}
	if got := p.filtered[p.cursor].Name; got != "main" {
		t.Fatalf("cursor on %q, want main", got)
	}

	p.input = "feat"
	p.refilter()
	if len(p.filtered) != 1 || p.filtered[0].Name != "feature/branch" {
		t.Fatalf("filter feat → %+v", p.filtered)
	}

	p.input = "zzz"
	p.refilter()
	if len(p.filtered) != 0 {
		t.Fatalf("expected no matches, got %+v", p.filtered)
	}
}

func TestBranchPickerActiveCursor(t *testing.T) {
	var p branchPicker
	p.Open([]git.Ref{
		{Name: "main", Current: true},
		{Name: "dev"},
		{Name: "origin/dev", Remote: true},
	}, "dev")
	if p.filtered[p.cursor].Name != "dev" {
		t.Fatalf("cursor on %q, want dev", p.filtered[p.cursor].Name)
	}
}

func TestApplyViewRev(t *testing.T) {
	m := Model{branch: "main", viewRev: ""}
	cmd := m.applyViewRev(git.Ref{Name: "dev"})
	if m.viewRev != "dev" {
		t.Fatalf("viewRev = %q", m.viewRev)
	}
	if !m.loading {
		t.Fatal("expected loading")
	}
	if cmd == nil {
		t.Fatal("expected load cmd")
	}

	// Selecting worktree current clears override.
	m.loading = false
	_ = m.applyViewRev(git.Ref{Name: "main", Current: true})
	if m.viewRev != "" {
		t.Fatalf("viewRev = %q, want empty", m.viewRev)
	}
}

func TestSwitchViewRevHEAD(t *testing.T) {
	m := Model{branch: "main", viewRev: "dev"}
	_ = m.switchViewRev("HEAD")
	if m.viewRev != "" {
		t.Fatalf("viewRev = %q after HEAD", m.viewRev)
	}
}

func TestExBranchLookup(t *testing.T) {
	if exLookup("branch") == nil {
		t.Fatal("expected :branch")
	}
	if exLookup("br") == nil {
		t.Fatal("expected :br alias")
	}
}

func TestBranchPickerTypesJK(t *testing.T) {
	var p branchPicker
	p.Open([]git.Ref{{Name: "main"}, {Name: "jack"}}, "")
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if p.input != "j" {
		t.Fatalf("input = %q, want j", p.input)
	}
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if p.input != "jk" {
		t.Fatalf("input = %q, want jk", p.input)
	}
}

func TestRenderBranchItemRightAlignsKind(t *testing.T) {
	const width = 40
	line := renderBranchItem(git.Ref{Name: "main"}, width, false, "")
	plain := stripStyle(line)
	if runewidth.StringWidth(plain) != width {
		t.Fatalf("width = %d, want %d (%q)", runewidth.StringWidth(plain), width, plain)
	}
	if !strings.HasSuffix(plain, "local") {
		t.Fatalf("expected …local, got %q", plain)
	}
	remote := stripStyle(renderBranchItem(git.Ref{Name: "origin/x", Remote: true}, width, false, ""))
	if !strings.HasSuffix(remote, "remote") {
		t.Fatalf("expected …remote, got %q", remote)
	}
	if runewidth.StringWidth(remote) != width {
		t.Fatalf("remote width = %d, want %d", runewidth.StringWidth(remote), width)
	}
}

// stripStyle removes lipgloss/ANSI SGR sequences for assertions.
func stripStyle(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			inEsc = true
		case inEsc:
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
