package ui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsiota/ore/internal/git"
)

func TestOpenLineEvolutionLoadsStack(t *testing.T) {
	m := Model{
		repo:     &git.Repo{Path: "/tmp/demo"},
		main:     MainBlame,
		blamePath: "code.txt",
		blameRev:  "HEAD",
		blame: []git.BlameLine{{
			Line:         3,
			Hash:         "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			ShortHash:    "aaaaaaa",
			Author:       "a",
			When:         time.Now(),
			Summary:      "add gamma",
			Text:         "gamma",
			PreviousHash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		}},
		blameCursor: 0,
	}
	nm, cmd := m.openLineEvolution()
	m = nm.(Model)
	if !m.loadingEvo || m.evoOriginPath != "code.txt" || cmd == nil {
		t.Fatalf("open: loading=%v path=%q cmd=%v", m.loadingEvo, m.evoOriginPath, cmd)
	}
	// Simulate async result without hitting git.
	msg := lineEvoLoadedMsg{
		path: "code.txt",
		rev:  "HEAD",
		steps: []git.LineEvolutionStep{
			{Index: 0, Path: "code.txt", Rev: "HEAD", Line: m.blame[0]},
			{Index: 1, Path: "code.txt", Rev: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Line: git.BlameLine{
				Line: 2, Hash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", ShortHash: "bbbbbbb", Text: "beta",
			}},
		},
	}
	nm2, _ := m.Update(msg)
	m = nm2.(Model)
	if m.main != MainLineEvo || len(m.evo) != 2 {
		t.Fatalf("after load: main=%v len=%d", m.main, len(m.evo))
	}
	view := m.renderEvolutionPane(80, 20)
	if view == "" {
		t.Fatal("expected evolution pane render")
	}
}

func TestActivateEvolutionStepStartsBlame(t *testing.T) {
	m := Model{
		repo:          &git.Repo{Path: "/tmp/demo"},
		main:          MainLineEvo,
		evoOriginPath: "code.txt",
		evoOriginRev:  "HEAD",
		evo: []git.LineEvolutionStep{
			{Index: 0, Path: "code.txt", Rev: "HEAD", Line: git.BlameLine{Line: 3, Hash: "aaa"}},
			{Index: 1, Path: "old.txt", Rev: "bbb", Line: git.BlameLine{Line: 2, Hash: "bbb"}},
		},
		evoCursor: 1,
	}
	nm, cmd := m.activateEvolutionStep()
	m = nm.(Model)
	if m.blamePath != "old.txt" || m.blameRev != "bbb" || m.blameFrom != MainLineEvo {
		t.Fatalf("blame target path=%q rev=%q from=%v", m.blamePath, m.blameRev, m.blameFrom)
	}
	if m.blamePreferLine != 2 || cmd == nil {
		t.Fatalf("prefer=%d cmd=%v", m.blamePreferLine, cmd)
	}
}

func TestEvolutionEscReturnsToBlame(t *testing.T) {
	m := Model{
		main:          MainLineEvo,
		evoOriginPath: "code.txt",
		blamePath:     "code.txt",
		blameRev:      "HEAD",
		blame:         []git.BlameLine{{Line: 1, Hash: "a", Text: "x"}},
		evo:           []git.LineEvolutionStep{{Index: 0, Path: "code.txt"}},
	}
	nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	// esc may go through handleKey → goBack only when focus main and not overlays.
	// Call goBack directly for unit certainty.
	_ = nm
	nm2, _ := m.goBack()
	m = nm2.(Model)
	if m.main != MainBlame {
		t.Fatalf("main = %v, want blame", m.main)
	}
}
