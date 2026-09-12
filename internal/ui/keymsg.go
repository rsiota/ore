package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// synthesizeKeyMsg constructs a tea.KeyMsg whose String() matches the given
// dispatch token so the palette can replay bindings through normal Update.
func synthesizeKeyMsg(token string) (tea.KeyMsg, bool) {
	switch token {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}, true
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}, true
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}, true
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}, true
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}, true
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}, true
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}, true
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}, true
	case "home":
		return tea.KeyMsg{Type: tea.KeyHome}, true
	case "end":
		return tea.KeyMsg{Type: tea.KeyEnd}, true
	}

	if strings.HasPrefix(token, "ctrl+") && len(token) == 6 {
		ch := token[5]
		if ch >= 'a' && ch <= 'z' {
			return tea.KeyMsg{Type: tea.KeyType(ch - 'a' + 1)}, true
		}
	}

	if token != "" {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(token)}, true
	}
	return tea.KeyMsg{}, false
}

// keyFilterChar reports whether msg is a printable character for filter input.
func keyFilterChar(msg tea.KeyMsg) (string, bool) {
	if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
		return msg.String(), true
	}
	return "", false
}

// replayKeySequence synthesizes keys through normal dispatch (chords via Sequence).
func replayKeySequence(seq []string) tea.Cmd {
	var cmds []tea.Cmd
	for _, tok := range seq {
		kmsg, ok := synthesizeKeyMsg(tok)
		if !ok {
			return nil
		}
		msg := kmsg
		cmds = append(cmds, func() tea.Msg { return msg })
	}
	switch len(cmds) {
	case 0:
		return nil
	case 1:
		return cmds[0]
	default:
		return tea.Sequence(cmds...)
	}
}
