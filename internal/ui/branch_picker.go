package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/rsiota/ore/internal/git"
)

// branchPicker is a fuzzy popup for choosing a read-only view revision.
type branchPicker struct {
	visible  bool
	input    string
	cursor   int
	items    []git.Ref
	filtered []git.Ref
	active   string // currently viewed rev (empty = worktree HEAD)
}

const maxBranchPickerItems = 16

func (p *branchPicker) Open(refs []git.Ref, activeRev string) {
	p.visible = true
	p.input = ""
	p.cursor = 0
	p.items = append([]git.Ref(nil), refs...)
	p.active = activeRev
	p.refilter()
	// Prefer highlighting the active view target.
	if activeRev != "" {
		for i, r := range p.filtered {
			if r.Name == activeRev {
				p.cursor = i
				break
			}
		}
	} else {
		for i, r := range p.filtered {
			if r.Current {
				p.cursor = i
				break
			}
		}
	}
}

func (p *branchPicker) Hide() { p.visible = false }

func (p branchPicker) IsVisible() bool { return p.visible }

func (p *branchPicker) refilter() {
	if p.input == "" {
		p.filtered = append([]git.Ref(nil), p.items...)
		p.cursor = 0
		return
	}
	ranked := fuzzyRank(p.input, p.items,
		func(r git.Ref) string {
			return r.Name + " " + r.Hash + " " + r.Subject
		},
		nil)
	p.filtered = make([]git.Ref, len(ranked))
	for i, r := range ranked {
		p.filtered[i] = r.Item
	}
	if p.cursor >= len(p.filtered) {
		p.cursor = max(0, len(p.filtered)-1)
	}
}

func (p *branchPicker) moveCursor(delta int) {
	n := len(p.filtered)
	if n == 0 {
		return
	}
	p.cursor = (p.cursor + delta + n) % n
}

func (p branchPicker) selected() (git.Ref, bool) {
	if p.cursor < 0 || p.cursor >= len(p.filtered) {
		return git.Ref{}, false
	}
	return p.filtered[p.cursor], true
}

type branchPickedMsg struct {
	ref git.Ref
}

func (p branchPicker) Update(msg tea.KeyMsg) (branchPicker, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		p.visible = false
		return p, nil
	case "enter":
		ref, ok := p.selected()
		p.visible = false
		if !ok {
			return p, nil
		}
		return p, func() tea.Msg { return branchPickedMsg{ref: ref} }
	case "up", "ctrl+p":
		// j/k type into the filter (branch names); arrows / ctrl+n/p move.
		p.moveCursor(-1)
		return p, nil
	case "down", "ctrl+n":
		p.moveCursor(1)
		return p, nil
	case "backspace":
		if p.input != "" {
			r := []rune(p.input)
			p.input = string(r[:len(r)-1])
			p.refilter()
		}
		return p, nil
	case "ctrl+u":
		p.input = ""
		p.refilter()
		return p, nil
	}
	if ch, ok := keyFilterChar(msg); ok {
		p.input += ch
		p.refilter()
	}
	return p, nil
}

func branchPopupDim() (w, h int) {
	return 71, maxBranchPickerItems + 4 // items + prompt + preview + border
}

// View renders the branch picker panel only — caller overlays it centered.
func (p branchPicker) View(width, height int) string {
	if !p.visible {
		return ""
	}
	innerW := width - 4
	if innerW < 24 {
		innerW = 24
	}

	start := 0
	if p.cursor >= maxBranchPickerItems {
		start = p.cursor - maxBranchPickerItems + 1
	}
	end := start + maxBranchPickerItems
	if end > len(p.filtered) {
		end = len(p.filtered)
	}

	var lines []string
	for i := start; i < end; i++ {
		lines = append(lines, renderBranchItem(p.filtered[i], innerW, i == p.cursor, p.active))
	}
	if len(lines) == 0 {
		lines = append(lines, styleMuted.Render("  no matches"))
	}
	for len(lines) < maxBranchPickerItems {
		lines = append(lines, "")
	}

	preview := "  "
	if ref, ok := p.selected(); ok {
		preview = "  " + styleMuted.Render(runewidth.Truncate(ref.Hash+" · "+ref.Subject, innerW-2, "…"))
	}
	lines = append(lines, preview)

	prompt := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render("❯ ") + lipgloss.NewStyle().Foreground(colorPrimary).Render(p.input)
	body := prompt + "\n" + strings.Join(lines, "\n")

	return lipgloss.NewStyle().
		Width(width - 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(0, 1).
		Render(body)
}

func renderBranchItem(ref git.Ref, width int, selected bool, activeRev string) string {
	mark := " "
	if ref.Current {
		mark = "*"
	} else if activeRev != "" && ref.Name == activeRev {
		mark = "·"
	}
	kind := "local"
	if ref.Remote {
		kind = "remote"
	}
	kindW := runewidth.StringWidth(kind)
	leftW := width - kindW
	if leftW < 4 {
		leftW = 4
		width = leftW + kindW
	}
	name := runewidth.Truncate(ref.Name, max(1, leftW-2), "…")
	left := runewidth.FillRight(fmt.Sprintf("%s %s", mark, name), leftW)
	line := left + kind

	if selected {
		return lipgloss.NewStyle().
			Foreground(colorBg).
			Background(colorPrimary).
			Render(line)
	}
	if ref.Remote {
		return styleMuted.Render(line)
	}
	return lipgloss.NewStyle().Foreground(colorPrimary).Render(line)
}
