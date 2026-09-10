package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) beginEx() {
	m.chordG = false
	m.exTyping = true
	m.exLine = ""
	m.status = ":"
}

func (m *Model) cancelEx() {
	m.exTyping = false
	m.exLine = ""
	m.refreshStatus()
}

func (m Model) handleExKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.cancelEx()
		return m, nil
	case "enter":
		line := m.exLine
		m.exTyping = false
		m.exLine = ""
		return m, m.runExCommand(line)
	case "backspace":
		if m.exLine != "" {
			r := []rune(m.exLine)
			m.exLine = string(r[:len(r)-1])
		}
		m.status = ":" + m.exLine + "█"
		return m, nil
	case "ctrl+u":
		m.exLine = ""
		m.status = ":█"
		return m, nil
	default:
		if len(msg.Runes) > 0 && !msg.Alt && msg.Type == tea.KeyRunes {
			m.exLine += string(msg.Runes)
			m.status = ":" + m.exLine + "█"
			return m, nil
		}
		if s := msg.String(); len(s) == 1 && s[0] >= 32 {
			m.exLine += s
			m.status = ":" + m.exLine + "█"
		}
	}
	return m, nil
}

func (m *Model) runExCommand(line string) tea.Cmd {
	line = strings.TrimSpace(line)
	if line == "" {
		m.refreshStatus()
		return nil
	}
	// Allow typing with or without leading ':'.
	line = strings.TrimPrefix(line, ":")
	line = strings.TrimSpace(line)
	fields := strings.Fields(line)
	if len(fields) == 0 {
		m.refreshStatus()
		return nil
	}
	verb, args := fields[0], fields[1:]
	spec := exLookup(verb)
	if spec == nil {
		m.status = fmt.Sprintf("E492: Not an editor command: %s", verb)
		return nil
	}
	return spec.run(m, args)
}

func (m *Model) exBlame(args []string) tea.Cmd {
	if len(args) == 0 {
		m.status = ":blame needs a path — :blame <path> [rev]"
		return nil
	}
	path := args[0]
	rev := "HEAD"
	if len(args) >= 2 {
		rev = args[1]
	} else if hash := m.selectedHash(); hash != "" {
		rev = hash
	}
	return m.startBlame(path, rev, m.main)
}

func (m *Model) exHistory(args []string) tea.Cmd {
	if len(args) == 0 {
		m.status = ":history needs a path — :history <path>"
		return nil
	}
	path := args[0]
	m.filter = ""
	m.filterTyping = false
	m.historyPath = path
	m.loadingHistory = true
	m.focus = FocusMain
	m.status = fmt.Sprintf("loading history · %s", path)
	return loadHistoryCmd(m.repo, path)
}

func (m *Model) exGoto(args []string) tea.Cmd {
	if len(args) == 0 {
		m.status = ":goto needs a hash — :goto <hash>"
		return nil
	}
	prefix := strings.ToLower(args[0])
	var matches []int
	for i, c := range m.commits {
		h := strings.ToLower(c.Hash)
		s := strings.ToLower(c.ShortHash)
		if strings.HasPrefix(h, prefix) || strings.HasPrefix(s, prefix) {
			matches = append(matches, i)
		}
	}
	switch len(matches) {
	case 0:
		// Not in the loaded window — still try to show the commit.
		m.filter = ""
		m.filterTyping = false
		m.main = MainCommits
		m.focus = FocusMain
		m.detailFilterPath = ""
		m.detailExpectHash = args[0]
		m.loadingDetail = true
		m.status = fmt.Sprintf("goto %s", shortHash(prefix))
		return loadDetailCmd(m.repo, args[0], "")
	case 1:
		m.filter = ""
		m.filterTyping = false
		m.main = MainCommits
		m.focus = FocusMain
		m.cursor = matches[0]
		m.ensureCommitVisible()
		m.refreshStatus()
		return m.reloadDetail()
	default:
		m.status = fmt.Sprintf("ambiguous hash %q (%d matches)", prefix, len(matches))
		return nil
	}
}
