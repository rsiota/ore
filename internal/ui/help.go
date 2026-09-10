package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpPanel is a simple scrollable keybinding overlay driven by registry().
type HelpPanel struct {
	visible bool
	offset  int
	width   int
	height  int
}

func (h *HelpPanel) Toggle() {
	if h.visible {
		h.Hide()
		return
	}
	h.Show()
}

func (h *HelpPanel) Show() {
	h.visible = true
	h.offset = 0
}

func (h *HelpPanel) Hide() { h.visible = false }

func (h HelpPanel) Visible() bool { return h.visible }

func (h *HelpPanel) SetSize(width, height int) {
	h.width = width
	h.height = height
}

func (h *HelpPanel) Update(msg tea.KeyMsg) bool {
	if !h.visible {
		return false
	}
	switch msg.String() {
	case "?", "q", "esc":
		h.Hide()
	case "j", "down":
		h.offset++
	case "k", "up":
		if h.offset > 0 {
			h.offset--
		}
	case "g", "home":
		h.offset = 0
	case "G", "end":
		h.offset = 1 << 20 // clamped in View
	case "ctrl+d":
		h.offset += max(1, h.bodyHeight()/2)
	case "ctrl+u":
		h.offset = max(0, h.offset-max(1, h.bodyHeight()/2))
	}
	return true
}

func (h HelpPanel) bodyHeight() int {
	return max(1, h.height-4)
}

func (h *HelpPanel) View() string {
	if !h.visible || h.width == 0 || h.height == 0 {
		return ""
	}
	rows := h.rows()
	bodyH := h.bodyHeight()
	maxOff := max(0, len(rows)-bodyH)
	if h.offset > maxOff {
		h.offset = maxOff
	}
	end := min(len(rows), h.offset+bodyH)
	visible := rows[h.offset:end]

	var b strings.Builder
	title := styleHelpTitle.Render(" ore help ")
	hint := styleMuted.Render("j/k scroll · ?/esc close")
	pad := h.width - lipgloss.Width(title) - lipgloss.Width(hint)
	if pad < 1 {
		b.WriteString(fitWidth(title, h.width))
	} else {
		b.WriteString(title + strings.Repeat(" ", pad) + hint)
	}
	b.WriteByte('\n')
	b.WriteString(styleMuted.Render(strings.Repeat("─", max(0, h.width))))
	b.WriteByte('\n')
	for _, row := range visible {
		b.WriteString(fitWidth(row, h.width))
		b.WriteByte('\n')
	}
	for len(visible) < bodyH {
		b.WriteString(strings.Repeat(" ", h.width))
		b.WriteByte('\n')
		visible = append(visible, "")
	}
	b.WriteString(fitWidth(styleMuted.Render("registry-driven · sibling spirit to creel"), h.width))
	return clampFrame(b.String(), h.height, h.width)
}

func (h HelpPanel) rows() []string {
	var rows []string
	rows = append(rows, styleMuted.Render("Archaeology-only Git TUI. Read-only by design."))
	rows = append(rows, "")
	for _, sec := range registry() {
		rows = append(rows, styleHelpSection.Render(sec.Title))
		for _, item := range sec.Items {
			key := item.Display
			if key == "" {
				key = "—"
			}
			rows = append(rows, fmt.Sprintf("  %-18s  %s", key, item.Desc))
		}
		rows = append(rows, "")
	}
	rows = append(rows, styleHelpSection.Render("Commands"))
	for _, cmd := range exCommands() {
		rows = append(rows, fmt.Sprintf("  %-18s  %s", cmd.usage, cmd.desc))
	}
	rows = append(rows, "")
	return rows
}
