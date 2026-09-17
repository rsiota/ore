package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rsiota/ore/internal/config"
	"github.com/rsiota/ore/internal/git"
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

func (m *Model) exEvolve() tea.Cmd {
	if m.main != MainBlame {
		m.status = ":evolve needs the blame grid — open blame first"
		return nil
	}
	nm, cmd := m.openLineEvolution()
	*m = nm.(Model)
	return cmd
}

func (m *Model) exPickaxe(args []string, mode git.PickaxeMode) tea.Cmd {
	if len(args) == 0 {
		if mode == git.PickaxeRegexp {
			m.status = ":G needs a regexp — :G <pattern>"
		} else {
			m.status = ":pickaxe needs a string — :pickaxe <text> or :S <text>"
		}
		return nil
	}
	query := strings.Join(args, " ")
	path := ""
	// Optional trailing -- path (vim-ish): :pickaxe foo -- path/to/file
	if i := strings.Index(query, " -- "); i >= 0 {
		path = strings.TrimSpace(query[i+4:])
		query = strings.TrimSpace(query[:i])
	}
	nm, cmd := m.startPickaxe(query, mode, path)
	*m = nm.(Model)
	return cmd
}

func (m *Model) exTheme(args []string) tea.Cmd {
	if len(args) == 0 {
		m.status = fmt.Sprintf("theme %s — :theme light|dark", m.theme)
		return nil
	}
	resolved, ok := resolveThemeName(args[0])
	if !ok {
		m.status = fmt.Sprintf("unknown theme %q (light, dark)", args[0])
		return nil
	}
	applyTheme(resolved)
	m.theme = resolved
	m.invalidateDetailCache()
	if m.config == nil {
		m.config = &config.Config{}
	}
	m.config.Theme = resolved
	if err := m.config.Save(); err != nil {
		m.status = fmt.Sprintf("theme %s (save failed: %v)", resolved, err)
		return nil
	}
	m.status = "theme " + resolved
	return nil
}

func (m *Model) exSet(args []string) tea.Cmd {
	if len(args) == 0 {
		m.status = fmt.Sprintf("transparent_background=%s — :set transparent_background on|off", boolOnOff(m.transparentBg))
		return nil
	}
	key := strings.ToLower(args[0])
	switch key {
	case "transparent_background", "transparent", "transparency":
		if len(args) == 1 {
			m.status = fmt.Sprintf("transparent_background=%s", boolOnOff(m.transparentBg))
			return nil
		}
		on, ok := parseBoolSetting(args[1])
		if !ok {
			m.status = ":set transparent_background needs on or off"
			return nil
		}
		m.transparentBg = on
		if m.config == nil {
			m.config = &config.Config{}
		}
		m.config.TransparentBackground = on
		if err := m.config.Save(); err != nil {
			m.status = fmt.Sprintf("transparent_background=%s (save failed: %v)", boolOnOff(on), err)
			return nil
		}
		m.status = fmt.Sprintf("transparent_background=%s", boolOnOff(on))
		return nil
	default:
		m.status = fmt.Sprintf("unknown setting %q (transparent_background)", args[0])
		return nil
	}
}

func boolOnOff(v bool) string {
	if v {
		return "on"
	}
	return "off"
}

func parseBoolSetting(s string) (val bool, ok bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "on", "true", "1", "yes":
		return true, true
	case "off", "false", "0", "no":
		return false, true
	default:
		return false, false
	}
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
		return m.reloadDetailNow()
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
