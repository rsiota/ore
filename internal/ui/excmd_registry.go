package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// exCmdSpec is one ":" command. exCommands() is the source of truth for
// dispatch and the help Commands section.
type exCmdSpec struct {
	verbs []string // canonical first, then aliases
	desc  string
	usage string
	run   func(m *Model, args []string) tea.Cmd
}

func exCommands() []exCmdSpec {
	return []exCmdSpec{
		{
			verbs: []string{"blame"},
			desc:  "open blame grid for a path",
			usage: ":blame <path> [rev]",
			run: func(m *Model, args []string) tea.Cmd {
				return m.exBlame(args)
			},
		},
		{
			verbs: []string{"history", "hist"},
			desc:  "open path history (--follow)",
			usage: ":history <path>",
			run: func(m *Model, args []string) tea.Cmd {
				return m.exHistory(args)
			},
		},
		{
			verbs: []string{"goto", "go"},
			desc:  "jump to a commit by hash prefix",
			usage: ":goto <hash>",
			run: func(m *Model, args []string) tea.Cmd {
				return m.exGoto(args)
			},
		},
		{
			verbs: []string{"branch", "br"},
			desc:  "view another branch tip (read-only; no checkout)",
			usage: ":branch [name]",
			run: func(m *Model, args []string) tea.Cmd {
				if len(args) == 0 {
					return m.openBranchPicker()
				}
				return m.switchViewRev(args[0])
			},
		},
		{
			verbs: []string{"theme"},
			desc:  "switch colour theme (light or dark)",
			usage: ":theme [light|dark]",
			run: func(m *Model, args []string) tea.Cmd {
				return m.exTheme(args)
			},
		},
		{
			verbs: []string{"set"},
			desc:  "change a setting (transparent_background)",
			usage: ":set transparent_background [on|off]",
			run: func(m *Model, args []string) tea.Cmd {
				return m.exSet(args)
			},
		},
		{
			verbs: []string{"refresh", "reload"},
			desc:  "reload the commit log and current view",
			usage: ":refresh",
			run: func(m *Model, _ []string) tea.Cmd {
				return m.refresh()
			},
		},
		{
			verbs: []string{"help", "h"},
			desc:  "open the help overlay",
			usage: ":help",
			run: func(m *Model, _ []string) tea.Cmd {
				m.help.Show()
				m.status = "help"
				return nil
			},
		},
		{
			verbs: []string{"quit", "q"},
			desc:  "quit ore",
			usage: ":q",
			run: func(_ *Model, _ []string) tea.Cmd {
				return tea.Quit
			},
		},
	}
}

func exLookup(verb string) *exCmdSpec {
	v := strings.ToLower(verb)
	cmds := exCommands()
	for i := range cmds {
		for _, alias := range cmds[i].verbs {
			if strings.ToLower(alias) == v {
				return &cmds[i]
			}
		}
	}
	return nil
}
